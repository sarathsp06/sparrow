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
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/sarathsp06/sparrow/internal/accessauth"
	"github.com/sarathsp06/sparrow/internal/ai"
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
	"github.com/sarathsp06/sparrow/pkg/access/httpauth"
	"github.com/sarathsp06/sparrow/pkg/crypto"
	"github.com/sarathsp06/sparrow/pkg/storage/postgres"
)

func main() {
	// Load .env file if present, but only outside production.
	// In production containers a .env file should not exist, but if one
	// is accidentally present it could silently override critical env vars
	// (DATABASE_URL, SPARROW_API_KEY, SPARROW_ENCRYPTION_KEYS).
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
	// SPARROW_ENCRYPTION_KEYS and SPARROW_ENCRYPTION_PRIMARY_KEY_ID must be
	// configured. The server will not start without a valid keyring.
	encKeyring, err := cfg.EncryptionKeyring()
	if err != nil {
		log.Fatalf("Failed to resolve encryption keyring: %v", err)
	}
	cryptoSvc, err := crypto.NewServiceFromKeyring(encKeyring)
	if err != nil {
		log.Fatalf("Failed to create crypto service: %v", err)
	}
	fmt.Println("🔐 Encryption enabled (envelope encryption with per-record DEK)")

	// Access tokens and invites (pkg/access, adapted in internal/accessauth).
	// SPARROW_API_KEY is the root key; named tokens created from it can each
	// be revoked. Consumer-scoped tokens only work through the portal gateway.
	accessSvc, err := accessauth.New(sqlxDB.DB, cfg.APIKey, accessauth.Sealer(cryptoSvc))
	if err != nil {
		log.Fatalf("Failed to create access service: %v", err)
	}

	// Authentication for /v1. When SPARROW_API_KEY is set, every request needs
	// the master key or a tenant-wide access token (X-API-Key header or
	// Authorization: Bearer). Health/ready and the OpenAPI docs/spec are
	// excluded. Portal traffic arrives pre-authorized through the /portal/api
	// gateway (see below). When unset, all requests are open.
	auth := middleware.NewAuth(cfg.APIKey, accessSvc, accessauth.Realm(), "/health", "/ready", "/docs", "/openapi")
	if auth.Enabled {
		fmt.Println("🔑 Authentication enabled (SPARROW_API_KEY or an access token required)")
	} else {
		fmt.Println("⚠️  SPARROW_API_KEY not set — all endpoints are open (no authentication)")
	}

	// Private network access for webhook URLs
	// When true, localhost/private IPs are allowed as webhook targets.
	// Useful for local dev or self-hosted deployments where targets are on the same network.
	if cfg.AllowPrivateNetworks {
		fmt.Println("⚠️  SPARROW_ALLOW_PRIVATE_NETWORKS=true — SSRF protection relaxed (loopback/private IPs allowed; cloud metadata still blocked). Prefer SPARROW_ALLOWED_NETWORKS in production")
	}
	if len(cfg.AllowedNetworks) > 0 {
		fmt.Printf("🌐 Webhook deliveries may also reach SPARROW_ALLOWED_NETWORKS: %s\n", strings.Join(cfg.AllowedNetworks, ", "))
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
	clientConfig.AllowedNetworks = cfg.AllowedNetworkList()
	clientConfig.MaxCapturedResponseBytes = cfg.MaxCapturedResponseBytes

	// Initialize queue manager
	queueManager, err := queue.NewManager(ctx, webhookRepo, cryptoSvc, dbPool, clientConfig, cfg.EventRetentionDays)
	if err != nil {
		log.Fatalf("Failed to create queue manager: %v", err)
	}
	defer func() { _ = queueManager.Stop(ctx) }()

	// Portal links are access tokens minted per visit; purge dead ones daily.
	queueManager.EnableTokenPurge(accessSvc, accessauth.TokenRetention)

	// Start the queue processing
	if err := queueManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue manager: %v", err)
	}

	fmt.Println("🚀 River queue started successfully")

	webhookService := webhooks.NewWebhookService(queueManager.GetJobInserter(), webhookRepo, cryptoSvc, webhooks.WithAllowPrivateNetworks(cfg.AllowPrivateNetworks), webhooks.WithAllowedNetworks(cfg.AllowedNetworkList()), webhooks.WithAutoRegisterEvents(cfg.AutoRegisterEvents))
	tracedWebhookService := webhooks.NewWebhookServiceInterfaceWithTracing(webhookService, "")

	// AI-assisted template drafting is opt-in (SPARROW_AI_*). Drafts are
	// verified through the same dry-run the editor's preview uses. The
	// prompt endpoint and its docs_url fetch work without a provider, so an
	// install with no AI still gets a copy-and-paste prompt.
	// docs_url fetches obey the delivery network policy (SSRF guard).
	fetch := ai.NewDocFetcher(client.NetworkPolicy{AllowPrivate: cfg.AllowPrivateNetworks, AllowedNetworks: cfg.AllowedNetworkList()})
	aiDeps := rest.AIDeps{Fetch: fetch}
	if cfg.AIEnabled() {
		helpers := make([]ai.HelperFunc, 0)
		for _, f := range tracedWebhookService.GetTemplateFunctions() {
			helpers = append(helpers, ai.HelperFunc{Name: f.Name, Description: f.Description})
		}
		// Drafts are verified strictly: a field the sample lacks is an error
		// naming the key, which the repair loop feeds back to the model.
		render := func(_ context.Context, eventName, tmpl string, payload map[string]any) (string, error) {
			return webhooks.RenderTemplatePreview(eventName, tmpl, payload, true)
		}
		drafter, err := ai.New(ai.Config{Provider: ai.Provider(cfg.AIProviderName()), APIKey: cfg.AIAPIKey, Model: cfg.AIModelOrDefault(), BaseURL: cfg.AIBaseURL}, render, helpers, fetch)
		if err != nil {
			log.Fatalf("Failed to configure AI drafting: %v", err)
		}
		aiDeps.Drafter = drafter
	}

	// Create chi router for the REST API, health endpoints, and embedded UI.
	// Chi provides clean route grouping: API routes get auth middleware,
	// health endpoints and UI are open.
	r := chi.NewRouter()

	// Global middleware: body size cap, security headers, then CORS
	r.Use(middleware.MaxBodyBytes(cfg.MaxBodyBytes))
	r.Use(middleware.SecurityHeaders(ui.InlineScriptHashes()))
	corsMiddleware, corsMode := middleware.CORS(cfg.CORSAllowedOrigins, cfg.IsProduction())
	logCORSMode(corsMode, cfg.CORSAllowedOrigins)
	r.Use(corsMiddleware)
	r.Use(otelhttp.NewMiddleware("sparrow"))

	// REST API — protected by API key auth. Huma registers every /v1
	// operation plus /openapi.{json,yaml} and the Scalar reference at /docs.
	r.Group(func(r chi.Router) {
		r.Use(auth.HTTPMiddleware)
		rest.Mount(r, tracedWebhookService, rest.AccessDeps{Service: accessSvc, AuthEnabled: auth.Enabled, TokenDefaultTTL: cfg.TokenDefaultTTL}, aiDeps)
	})

	// Invite redemption: the invite in the request is the credential, so it
	// sits outside the authenticated /v1 group.
	r.Post("/invite/redeem", httpauth.RedeemHandler(accessSvc).ServeHTTP)

	// Consumer portal API — one static public prefix. The gateway verifies the
	// consumer-scoped bearer token, maps /portal/api/<rest> to its real /v1
	// path, and re-dispatches into the router so the same handlers run. The
	// consumer is carried by the token, never the URL, so an operator exposing
	// the portal allowlists just /portal, /_app, and /portal/api.
	r.Handle("/portal/api/*", middleware.PortalGateway(middleware.NewPortalVerifier(accessSvc, accessauth.Realm()), r))

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
			// The UI is served as built — the API key is never written into
			// its pages. Operators sign in with the key (swapped for a
			// browser token), an access token, or an invite link.
			uiHandler := ui.Handler(slog.Default())
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
			fmt.Println("⚠️  SPARROW_SERVE_UI=true but no frontend build found. Build with: cd web && npm run build")
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
	if auth.Enabled {
		fmt.Println("   Auth: API key or access token required (X-API-Key or Authorization: Bearer)")
	} else {
		fmt.Println("   Auth: disabled (set SPARROW_API_KEY to enable)")
	}
	if cfg.AIEnabled() {
		fmt.Printf("   AI drafting: enabled (%s via %s)\n", cfg.AIModelOrDefault(), cfg.AIProviderName())
	} else {
		fmt.Println("   AI drafting: prompt-only (the editor offers a copy-and-paste prompt; set SPARROW_AI_API_KEY, or SPARROW_AI_PROVIDER=openai with SPARROW_AI_BASE_URL, to draft in place)")
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

// logCORSMode prints the cross-origin policy chosen by middleware.CORS.
func logCORSMode(mode middleware.CORSMode, origins []string) {
	switch mode {
	case middleware.CORSAllowList:
		fmt.Printf("🔒 CORS allowed origins: %v\n", middleware.NormalizeOrigins(origins))
	case middleware.CORSBlockAll:
		fmt.Println("🔒 CORS: production mode — cross-origin requests blocked")
		fmt.Println("   If the UI is hosted separately, set CORS_ALLOWED_ORIGINS to the UI origin")
	case middleware.CORSAllowAll:
		fmt.Println("⚠️  CORS_ALLOWED_ORIGINS not set — allowing all origins (development mode, not for production)")
	}
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
