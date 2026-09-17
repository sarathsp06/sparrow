package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	slogotel "github.com/remychantenay/slog-otel"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/sarathsp06/sparrow/internal/config"
	"github.com/sarathsp06/sparrow/internal/health"
	"github.com/sarathsp06/sparrow/internal/middleware"
	"github.com/sarathsp06/sparrow/internal/migration"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/ui"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/crypto"
	"github.com/sarathsp06/sparrow/pkg/storage/postgres"
)

func main() {
	// Load .env file if present, but only outside production.
	// In production containers a .env file should not exist, but if one
	// is accidentally present it could silently override critical env vars
	// (DATABASE_URL, SPARROW_API_KEY, SPARROW_ENCRYPTION_KEY).
	// We check ENVIRONMENT from the real OS env first (before godotenv)
	// to avoid the chicken-and-egg problem.
	if os.Getenv("ENVIRONMENT") != "production" {
		_ = godotenv.Load()
	}

	// Load structured configuration from environment variables.
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	for _, warn := range cfg.Warnings() {
		log.Printf("⚠️  %s", warn)
	}

	ctx := context.Background()
	startTime := time.Now() // Track service start time for uptime calculation

	// Configure OpenTelemetry
	otelConfig := observability.DefaultConfig()

	if cfg.Environment != "" {
		otelConfig.Environment = cfg.Environment
	}

	if cfg.OTLPEndpoint != "" {
		otelConfig.OTLPEndpoint = cfg.OTLPEndpoint
	}

	// Initialize OpenTelemetry (no-op when OTEL_EXPORTER_OTLP_ENDPOINT is unset)
	otelShutdown, err := observability.Setup(ctx, otelConfig)
	if err != nil {
		log.Printf("⚠️  Failed to setup OpenTelemetry: %v", err)
		fmt.Println("🚀 Continuing without OpenTelemetry...")
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := otelShutdown(shutdownCtx); err != nil {
				log.Printf("Failed to shutdown OpenTelemetry: %v", err)
			}
		}()
		if otelConfig.OTLPEndpoint != "" {
			fmt.Printf("🔭 OpenTelemetry enabled (endpoint: %s, env: %s)\n", otelConfig.OTLPEndpoint, otelConfig.Environment)
		} else {
			fmt.Println("🔭 OpenTelemetry disabled (set OTEL_EXPORTER_OTLP_ENDPOINT to enable)")
		}
	}

	// Initialize structured logging with OTel bridge.
	slog.SetDefault(slog.New(slogotel.OtelHandler{
		Next: slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo, AddSource: true}),
	}))

	// Run database migrations before anything else touches the DB.
	// This covers both River queue schema and application schema migrations.
	// golang-migrate uses PostgreSQL advisory locks, so concurrent server
	// instances won't conflict.
	migrationLogger := slog.Default()
	fmt.Println("📦 Running database migrations...")
	if err := migration.RunAllMigrations(ctx, cfg.DatabaseURL, "up", 0, 0, migrationLogger); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	fmt.Println("✅ Database migrations completed")

	// Create database connection pool for River queue workers.
	// River runs 45 concurrent workers (20 events + 20 webhooks + 5 default)
	// so we need enough connections to avoid starvation. The default pgxpool
	// MaxConns is max(4, NumCPU) which is far too low.
	pgxConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to parse database URL for pgxpool: %v", err)
	}
	pgxConfig.MaxConns = 50                        // 45 workers + headroom
	pgxConfig.MinConns = 10                        // keep warm connections ready
	pgxConfig.MaxConnLifetime = 30 * time.Minute   // recycle connections periodically
	pgxConfig.MaxConnIdleTime = 5 * time.Minute    // release idle connections
	pgxConfig.HealthCheckPeriod = 30 * time.Second // detect stale connections

	dbPool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	if err != nil {
		log.Fatalf("Failed to create database pool: %v", err)
	}

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlxDB, err := postgres.Open(cfg.DatabaseURL, 3,
		postgres.WithMaxOpenConnections(25),
		postgres.WithMaxIdleConnections(25),
		postgres.WithConnectionMaxLifeTime(5*time.Minute),
		postgres.WithSetConnMaxIdleTime(5*time.Minute),
	)
	if err != nil {
		log.Fatalf("Failed to open sqlx database: %v", err)
	}
	defer sqlxDB.Close() //nolint:errcheck

	// Bootstrap default tenant (must exist from migrations)
	if err := tenant.Bootstrap(ctx, sqlxDB); err != nil {
		log.Fatalf("Failed to bootstrap: %v", err)
	}

	// Initialize encryption service.
	// SPARROW_ENCRYPTION_KEY is required. The server will not start without it.
	encKey, err := resolveEncryptionKey(cfg)
	if err != nil {
		log.Fatalf("Failed to resolve encryption key: %v", err)
	}
	cryptoSvc, err := crypto.NewService(encKey)
	if err != nil {
		log.Fatalf("Failed to create crypto service: %v", err)
	}
	fmt.Println("🔐 Encryption enabled (envelope encryption with per-record DEK)")

	// Consumer portal tokens: signed with the encryption key, they grant an
	// end consumer scoped access to their consumer's routes via the portal
	// gateway at /portal/api and the embedded portal UI at /portal. Minted
	// via POST .../portal-token (admin).
	portalTokens := middleware.NewPortalTokens(encKey)

	// Configure optional API key authentication.
	// When SPARROW_API_KEY is set, all /v1 requests must include the key via
	// the X-API-Key header. Health/ready, the OpenAPI docs/spec, and static UI
	// assets are excluded. Portal traffic arrives pre-authorized through the
	// /portal/api gateway (see below). When unset, all requests are open.
	apiKeyAuth := &middleware.APIKeyAuth{
		APIKey: cfg.APIKey,
		ExcludedPathPrefixes: []string{
			"/health",
			"/ready",
			"/docs",
			"/openapi",
		},
	}
	if apiKeyAuth.Enabled() {
		fmt.Println("🔑 API key authentication enabled (SPARROW_API_KEY is set)")
	} else {
		fmt.Println("⚠️  SPARROW_API_KEY not set — all endpoints are open (no authentication)")
	}

	// Private network access for webhook URLs
	// When true, localhost/private IPs are allowed as webhook targets.
	// Useful for local dev or self-hosted deployments where targets are on the same network.
	if cfg.AllowPrivateNetworks {
		fmt.Println("⚠️  SPARROW_ALLOW_PRIVATE_NETWORKS=true — SSRF protection relaxed (loopback/private IPs allowed)")
	}

	// Event retention: purge events (and their deliveries) older than N days.
	if cfg.EventRetentionDays > 0 {
		fmt.Printf("🧹 Event retention enabled — purging data older than %d days (SPARROW_EVENT_RETENTION_DAYS)\n", cfg.EventRetentionDays)
	}

	// Create webhook repository
	webhookRepo := store.NewRepositoryInterfaceWithTracing(store.NewRepository(sqlxDB), "")

	registerSystemEventTypes(ctx, webhookRepo)

	// Initialize webhook HTTP client config
	clientConfig := client.DefaultConfig()
	clientConfig.AllowPrivateNetworks = cfg.AllowPrivateNetworks

	// Initialize queue manager
	queueManager, err := queue.NewManager(ctx, webhookRepo, cryptoSvc, dbPool, clientConfig, cfg.EventRetentionDays)
	if err != nil {
		log.Fatalf("Failed to create queue manager: %v", err)
	}
	defer func() { _ = queueManager.Stop(ctx) }()

	// Start the queue processing
	if err := queueManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue manager: %v", err)
	}

	fmt.Println("🚀 River queue started successfully")

	webhookService := webhooks.NewWebhookService(queueManager.GetJobInserter(), webhookRepo, cryptoSvc, webhooks.WithAllowPrivateNetworks(cfg.AllowPrivateNetworks))
	tracedWebhookService := webhooks.NewWebhookServiceInterfaceWithTracing(webhookService, "")

	bootstrapAlertChannel(ctx, cfg, webhookService)

	// Create chi router for the REST API, health endpoints, and embedded UI.
	// Chi provides clean route grouping: API routes get auth middleware,
	// health endpoints and UI are open.
	r := chi.NewRouter()

	// Global middleware: body size cap, security headers, then CORS
	r.Use(middleware.MaxBodyBytes(cfg.MaxBodyBytes))
	r.Use(middleware.SecurityHeaders)
	corsHandler := buildCORSHandler(cfg)
	r.Use(corsHandler.Handler)
	r.Use(otelhttp.NewMiddleware("sparrow"))

	// REST API — protected by API key auth. Huma registers every /v1
	// operation plus /openapi.{json,yaml} and the Scalar reference at /docs.
	r.Group(func(r chi.Router) {
		r.Use(apiKeyAuth.HTTPMiddleware)
		rest.Mount(r, tracedWebhookService, portalTokens)
	})

	// Consumer portal API — one static public prefix. The gateway verifies the
	// consumer-scoped bearer token, maps /portal/api/<rest> to its real /v1
	// path, and re-dispatches into the router so the same handlers run. The
	// consumer is carried by the token, never the URL, so an operator exposing
	// the portal allowlists just /portal, /_app, and /portal/api.
	r.Handle("/portal/api/*", middleware.PortalGateway(portalTokens, r))

	// Initialize health checker
	healthChecker := health.NewChecker(dbPool, startTime)

	// Health and readiness endpoints bypass API key auth.
	r.Get("/health", healthChecker.HealthHandler())
	r.Get("/ready", healthChecker.ReadyHandler())

	// Serve embedded web UI if enabled.
	// The UI handler is registered as the NotFound handler so it acts as
	// a catch-all for paths that don't match any API or health route.
	// Chi's explicit routes always take precedence — API requests can
	// never accidentally be served HTML by the SPA.
	if cfg.ServeUI {
		if ui.Available() {
			uiConfig := &ui.Config{APIKey: apiKeyAuth.APIKey}
			uiHandler := ui.Handler(slog.Default(), uiConfig)
			r.NotFound(func(w http.ResponseWriter, r *http.Request) {
				// Serve SPA only for GET/HEAD requests. Non-GET to unknown
				// paths returns a JSON 404 so API clients never get HTML.
				if r.Method != http.MethodGet && r.Method != http.MethodHead {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusNotFound)
					_, _ = w.Write([]byte(`{"error":"not found"}`))
					return
				}
				uiHandler.ServeHTTP(w, r)
			})
			fmt.Println("🖥️  Embedded web UI enabled at http://localhost:" + cfg.HTTPPort + "/")
		} else {
			fmt.Println("⚠️  SPARROW_SERVE_UI=true but no frontend build found. Build with: cd web && npm run build:static")
		}
	}

	httpServer := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second, // Mitigate slowloris attacks
		MaxHeaderBytes:    1 << 20,          // 1 MB max header size
	}

	fmt.Println("🌐 Starting server...")
	fmt.Printf("   REST API: localhost:%s\n", cfg.HTTPPort)

	// Start HTTP server in a goroutine
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve HTTP: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("🎯 HTTP Queue Server is running...")
	fmt.Printf("   REST API: localhost:%s\n", cfg.HTTPPort)
	fmt.Printf("   API docs: http://localhost:%s/docs\n", cfg.HTTPPort)
	fmt.Printf("   Health check: http://localhost:%s/health\n", cfg.HTTPPort)
	fmt.Printf("   Readiness check: http://localhost:%s/ready\n", cfg.HTTPPort)
	if cfg.ServeUI && ui.Available() {
		fmt.Printf("   Web UI: http://localhost:%s/\n", cfg.HTTPPort)
	}
	if apiKeyAuth.Enabled() {
		fmt.Println("   Auth: API key required (X-API-Key header)")
	} else {
		fmt.Println("   Auth: disabled (set SPARROW_API_KEY to enable)")
	}
	if otelShutdown != nil {
		fmt.Printf("   OTLP endpoint: %s\n", otelConfig.OTLPEndpoint)
	}
	fmt.Println("   Press Ctrl+C to stop...")
	<-sigChan

	fmt.Println("\n🛑 Shutting down...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	_ = queueManager.Stop(shutdownCtx)
	fmt.Println("👋 Shutdown complete")
}

// buildCORSHandler creates a CORS handler configured via cfg.CORSAllowedOrigins.
// When set, only the listed origins are allowed. When unset: production defaults
// to no cross-origin access, development defaults to allow-all for convenience.
//
// If the UI is served separately (not embedded via SPARROW_SERVE_UI), the
// operator must set CORS_ALLOWED_ORIGINS to the UI's origin, e.g.:
//
//	CORS_ALLOWED_ORIGINS=https://sparrow-ui.internal.example.com
func buildCORSHandler(cfg *config.Config) *cors.Cors {
	origins := cfg.CORSAllowedOrigins

	if len(origins) == 0 || (len(origins) == 1 && origins[0] == "") {
		// SEC: Defaulting to allow-all lets any website make API calls on
		// behalf of a user who has network access. In production, restrict
		// by default — the embedded UI is same-origin and doesn't need CORS.
		// If the UI is hosted separately, the operator must set
		// CORS_ALLOWED_ORIGINS explicitly.
		if cfg.IsProduction() {
			fmt.Println("🔒 CORS: production mode — cross-origin requests blocked")
			fmt.Println("   If the UI is hosted separately, set CORS_ALLOWED_ORIGINS to the UI origin")
			return cors.New(cors.Options{
				AllowedOrigins: []string{}, // no origins allowed
			})
		}
		fmt.Println("⚠️  CORS_ALLOWED_ORIGINS not set — allowing all origins (development mode, not for production)")
		return cors.AllowAll()
	}

	// Filter out any empty strings from the slice.
	var filtered []string
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			filtered = append(filtered, o)
		}
	}

	if len(filtered) == 0 {
		if cfg.IsProduction() {
			fmt.Println("🔒 CORS: production mode — cross-origin requests blocked (CORS_ALLOWED_ORIGINS is empty)")
			fmt.Println("   If the UI is hosted separately, set CORS_ALLOWED_ORIGINS to the UI origin")
			return cors.New(cors.Options{
				AllowedOrigins: []string{},
			})
		}
		fmt.Println("⚠️  CORS_ALLOWED_ORIGINS is empty — allowing all origins (development mode, not for production)")
		return cors.AllowAll()
	}

	fmt.Printf("🔒 CORS allowed origins: %v\n", filtered)
	return cors.New(cors.Options{
		AllowedOrigins:   filtered,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Connect-Protocol-Version", "Connect-Timeout-Ms", "Grpc-Timeout", "X-Grpc-Web", "X-User-Agent", "X-API-Key"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}

// resolveEncryptionKey determines the 32-byte KEK.
//
// cfg.EncryptionKey must be set to a 64-character hex string (32 bytes).
// The server will not start without it. The key is NOT stored in the
// database — storing the encryption key next to the data it protects
// defeats the purpose of encryption at rest.
//
// Generate a key with: openssl rand -hex 32
func resolveEncryptionKey(cfg *config.Config) ([]byte, error) {
	if cfg.EncryptionKey == "" {
		return nil, fmt.Errorf("SPARROW_ENCRYPTION_KEY is required. Generate one with: openssl rand -hex 32")
	}
	key, err := crypto.ParseKey(cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("invalid SPARROW_ENCRYPTION_KEY: %w", err)
	}
	return key, nil
}

// registerSystemEventTypes idempotently registers the event types Sparrow
// generates about its own webhooks, with their JSON schemas and sample
// payloads (see queue.SystemEventRegistrations). Tenants opt an email address
// into them via the webhook_alert_configs API; they are never delivered
// directly, only fanned out through the normal event/subscription/delivery
// pipeline under the "_sparrow" consumer.
func registerSystemEventTypes(ctx context.Context, repo store.EventTypeRepository) {
	for _, t := range queue.SystemEventRegistrations() {
		existing, err := repo.GetEventByName(ctx, tenant.DefaultTenantID, t.Name)
		if err != nil {
			log.Printf("⚠️  Failed to look up system event type %s: %v", t.Name, err)
			continue
		}
		if existing != nil {
			continue
		}
		reg := t
		if err := repo.RegisterEvent(ctx, tenant.DefaultTenantID, &reg); err != nil {
			log.Printf("⚠️  Failed to register system event type %s: %v", t.Name, err)
		}
	}
}

// sendGridMailSendURL is the SendGrid v3 Mail Send endpoint the bootstrapped
// alert webhook posts to.
const sendGridMailSendURL = "https://api.sendgrid.com/v3/mail/send"

// sendGridTransformTemplate mirrors satellites/recipes/sendgrid.yaml (kept in
// sync by hand — the recipes package is a separate module). {{param "..."}}
// placeholders are substituted with config values at bootstrap time; the rest
// is rendered server-side per delivery.
const sendGridTransformTemplate = `{
  "personalizations": [{{range $i, $r := .payload.alert_recipients}}{{if $i}},{{end}}{"to": [{"email": {{$r.email | json}}}]}{{end}}],
  "from": {"email": "{{param "from_email"}}", "name": "{{param "from_name"}}"},
  "subject": {{if eq .event_name "sparrow.webhook.health_changed"}}{{if eq .payload.new_health "healthy"}}{{printf "Sparrow: webhook for %v recovered" .payload.consumer | json}}{{else}}{{printf "Sparrow: webhook for %v is now %v" .payload.consumer .payload.new_health | json}}{{end}}{{else}}{{printf "Sparrow: delivery to %v failed permanently" .payload.consumer | json}}{{end}},
  "content": [{"type": "text/plain", "value": {{if eq .event_name "sparrow.webhook.health_changed"}}{{printf "Webhook %v (%v) health changed: %v -> %v" .payload.webhook_id .payload.url .payload.old_health .payload.new_health | json}}{{else}}{{printf "Webhook %v (%v) delivery %v failed permanently after %v attempt(s): %v (%v)" .payload.webhook_id .payload.url .payload.delivery_id .payload.attempt .payload.error_message .payload.error_category | json}}{{end}}}]
}`

// bootstrapAlertChannel idempotently wires the default delivery channel for
// Sparrow's system events: a SendGrid webhook + one transformed subscription
// per system event under the "_sparrow" consumer. When SPARROW_SENDGRID_API_KEY
// is unset the webhook is created with a mock key and left inactive; setting
// the key on a later restart re-activates it with the real key. Any existing
// "_sparrow" webhook (operator-wired or previously bootstrapped) is otherwise
// left untouched.
func bootstrapAlertChannel(ctx context.Context, cfg *config.Config, svc *webhooks.WebhookService) {
	existing, _, err := svc.ListWebhooks(ctx, queue.SystemEventConsumer, "", "", false, "", 10, 0)
	if err != nil {
		log.Printf("⚠️  Failed to list %s webhooks for alert bootstrap: %v", queue.SystemEventConsumer, err)
		return
	}

	if len(existing) > 0 {
		if cfg.SendGridAPIKey == "" {
			return
		}
		// Key newly provided: re-activate the bootstrapped webhook with it.
		for _, w := range existing {
			if w.URL != sendGridMailSendURL || w.Active {
				continue
			}
			err := svc.UpdateWebhookConfig(ctx, w.ID.String(), queue.SystemEventConsumer, nil, "", nil, true, "", nil,
				map[string]string{"Authorization": "Bearer " + cfg.SendGridAPIKey}, "", []string{"active", "secret_headers"})
			if err != nil {
				log.Printf("⚠️  Failed to activate SendGrid alert webhook %s: %v", w.ID, err)
				continue
			}
			fmt.Println("📧 SendGrid alert webhook activated (SPARROW_SENDGRID_API_KEY is set)")
		}
		return
	}

	apiKey := cfg.SendGridAPIKey
	active := apiKey != ""
	if apiKey == "" {
		apiKey = "SG.mock-set-SPARROW_SENDGRID_API_KEY-to-activate"
	}
	webhookID, _, err := svc.RegisterWebhook(ctx, queue.SystemEventConsumer, nil, sendGridMailSendURL,
		map[string]string{"Content-Type": "application/json"}, 30, active,
		"SendGrid email alerts for Sparrow system events (auto-created; set SPARROW_SENDGRID_API_KEY to activate)",
		map[string]string{"Authorization": "Bearer " + apiKey})
	if err != nil {
		log.Printf("⚠️  Failed to create SendGrid alert webhook: %v", err)
		return
	}

	tmpl := strings.NewReplacer(
		`{{param "from_email"}}`, cfg.AlertFromEmail,
		`{{param "from_name"}}`, cfg.AlertFromName,
	).Replace(sendGridTransformTemplate)
	for _, reg := range queue.SystemEventRegistrations() {
		if _, _, err := svc.CreateSubscription(ctx, webhookID, reg.Name, queue.SystemEventConsumer, nil, "", 30, true, tmpl, nil); err != nil {
			log.Printf("⚠️  Failed to subscribe SendGrid alert webhook to %s: %v", reg.Name, err)
		}
	}
	if active {
		fmt.Println("📧 SendGrid alert webhook created and active")
	} else {
		fmt.Println("📧 SendGrid alert webhook created inactive (set SPARROW_SENDGRID_API_KEY to activate)")
	}
}
