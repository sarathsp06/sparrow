package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/sarathsp06/sparrow/internal/config"
	connectserver "github.com/sarathsp06/sparrow/internal/connect"
	grpcserver "github.com/sarathsp06/sparrow/internal/grpc"
	"github.com/sarathsp06/sparrow/internal/health"
	"github.com/sarathsp06/sparrow/internal/middleware"
	"github.com/sarathsp06/sparrow/internal/migration"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/ui"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/crypto"
	"github.com/sarathsp06/sparrow/pkg/storage/postgres"
	pb "github.com/sarathsp06/sparrow/proto"
	pbconnect "github.com/sarathsp06/sparrow/proto/protoconnect"
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

	logger := slog.Default()

	// Create tenant repository and service
	tenantRepo := tenant.NewRepository(sqlxDB)
	tenantSvc := tenant.NewService(tenantRepo)

	// Bootstrap default tenant and root API key
	bootstrapCfg := tenant.DefaultBootstrapConfig()
	bootstrapCfg.Logger = logger
	if err := tenant.Bootstrap(ctx, tenantSvc, bootstrapCfg); err != nil {
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

	// Re-encrypt any plaintext webhook_secret values that were converted from TEXT to BYTEA
	// by migration 000015. These are raw UTF-8 bytes, not envelope-encrypted.
	if err := migrateWebhookSecrets(ctx, sqlxDB, cryptoSvc); err != nil {
		log.Printf("⚠️  Failed to migrate webhook secrets: %v (non-fatal, will retry on next restart)", err)
	}

	// Configure optional API key authentication.
	// When SPARROW_API_KEY is set, all API requests (HTTP + gRPC) must include
	// the key via X-API-Key header. Health/ready endpoints and static UI assets
	// are excluded. When unset, all requests are allowed (open access).
	apiKeyAuth := &middleware.APIKeyAuth{
		APIKey: cfg.APIKey,
		ExcludedPathPrefixes: []string{
			"/health",
			"/ready",
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

	// Create webhook repository
	webhookRepo := store.NewRepositoryInterfaceWithTracing(store.NewRepository(sqlxDB), "")

	// Initialize webhook HTTP client config
	clientConfig := client.DefaultConfig()
	clientConfig.AllowPrivateNetworks = cfg.AllowPrivateNetworks

	// Initialize queue manager
	queueManager, err := queue.NewManager(ctx, webhookRepo, cryptoSvc, dbPool, clientConfig)
	if err != nil {
		log.Fatalf("Failed to create queue manager: %v", err)
	}
	defer func() { _ = queueManager.Stop(ctx) }()

	// Start the queue processing
	if err := queueManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue manager: %v", err)
	}

	fmt.Println("🚀 River queue started successfully")

	// Initialize gRPC server with OpenTelemetry instrumentation and optional API key auth
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(apiKeyAuth.UnaryServerInterceptor()),
		grpc.ChainStreamInterceptor(apiKeyAuth.StreamServerInterceptor()),
		grpc.MaxRecvMsgSize(4<<20), // 4 MB max inbound message
		grpc.MaxSendMsgSize(4<<20), // 4 MB max outbound message
	)

	webhookService := webhooks.NewWebhookService(queueManager.GetJobInserter(), webhookRepo, cryptoSvc, webhooks.WithAllowPrivateNetworks(cfg.AllowPrivateNetworks))

	webhookGRPCServer := grpcserver.NewWebhookServer(webhooks.NewWebhookServiceInterfaceWithTracing(webhookService, ""))
	pb.RegisterWebhookServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterEventServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterSubscriptionServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterDeliveryServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterHealthServiceServer(grpcServer, webhookGRPCServer)

	// Initialize Connect-RPC server
	webhookConnectServer := connectserver.NewWebhookConnectServer(webhookGRPCServer, webhookGRPCServer, webhookGRPCServer, webhookGRPCServer, webhookGRPCServer)

	// Create API mux for Connect-RPC and health endpoints using chi router.
	// Chi provides clean route grouping: API routes get auth middleware,
	// health endpoints and UI are open.
	r := chi.NewRouter()

	// Global middleware: security headers, then CORS
	r.Use(middleware.SecurityHeaders)
	corsHandler := buildCORSHandler(cfg)
	r.Use(corsHandler.Handler)

	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		log.Fatal(err)
	}

	connectOpts := []connect.HandlerOption{
		connect.WithInterceptors(otelInterceptor),
		connect.WithReadMaxBytes(4 << 20), // 4 MB
		connect.WithSendMaxBytes(4 << 20), // 4 MB
	}

	// API routes — protected by API key auth.
	// Each Connect-RPC handler returns (path, handler) where path is a
	// prefix like "/webhook.WebhookService/". Chi's Handle with a
	// wildcard pattern routes all RPCs under that prefix to the handler.
	r.Group(func(r chi.Router) {
		r.Use(apiKeyAuth.HTTPMiddleware)

		for _, makeHandler := range []func() (string, http.Handler){
			func() (string, http.Handler) {
				return pbconnect.NewWebhookServiceHandler(webhookConnectServer, connectOpts...)
			},
			func() (string, http.Handler) {
				return pbconnect.NewEventServiceHandler(webhookConnectServer, connectOpts...)
			},
			func() (string, http.Handler) {
				return pbconnect.NewSubscriptionServiceHandler(webhookConnectServer, connectOpts...)
			},
			func() (string, http.Handler) {
				return pbconnect.NewDeliveryServiceHandler(webhookConnectServer, connectOpts...)
			},
			func() (string, http.Handler) {
				return pbconnect.NewHealthServiceHandler(webhookConnectServer, connectOpts...)
			},
		} {
			path, handler := makeHandler()
			r.Handle(path+"*", handler)
		}
	})

	// Initialize health checker
	healthChecker := health.NewChecker(dbPool, queueManager, startTime)

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
			uiHandler := ui.Handler(logger, uiConfig)
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

	// Start gRPC server
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", cfg.GRPCPort, err)
	}

	fmt.Println("🌐 Starting servers...")
	fmt.Printf("   gRPC server: localhost:%s\n", cfg.GRPCPort)
	fmt.Printf("   Connect-RPC (HTTP): localhost:%s\n", cfg.HTTPPort)

	// Register reflection service on gRPC server.
	reflection.Register(grpcServer)

	// Start gRPC server in a goroutine
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC: %v", err)
		}
	}()

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
	fmt.Printf("   gRPC server: localhost:%s\n", cfg.GRPCPort)
	fmt.Printf("   Connect-RPC (HTTP): localhost:%s\n", cfg.HTTPPort)
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

	// Shutdown gRPC server
	grpcServer.GracefulStop()
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

// migrateWebhookSecrets re-encrypts any webhook_secret values that were
// converted from TEXT to BYTEA by migration 000015 but are not yet
// envelope-encrypted. These are raw UTF-8 bytes of the original plaintext.
func migrateWebhookSecrets(ctx context.Context, db *postgres.SQLXDB, cryptoSvc *crypto.Service) error {
	type row struct {
		ID            string `db:"id"`
		WebhookSecret []byte `db:"webhook_secret"`
	}

	var rows []row
	err := db.SelectContext(ctx, &rows,
		`SELECT id, webhook_secret FROM webhook_registrations WHERE webhook_secret IS NOT NULL`)
	if err != nil {
		return fmt.Errorf("query webhook secrets: %w", err)
	}

	var migrated int
	for _, r := range rows {
		if len(r.WebhookSecret) == 0 {
			continue
		}
		// Skip already envelope-encrypted values
		if crypto.IsEnvelopeEncrypted(r.WebhookSecret) {
			continue
		}

		// The value is raw plaintext bytes (from TEXT->BYTEA cast).
		// Encrypt it with envelope encryption.
		plaintext := string(r.WebhookSecret)
		encrypted, err := cryptoSvc.EncryptString(plaintext)
		if err != nil {
			return fmt.Errorf("encrypt webhook secret for %s: %w", r.ID, err)
		}

		_, err = db.ExecContext(ctx,
			`UPDATE webhook_registrations SET webhook_secret = $1 WHERE id = $2`,
			encrypted, r.ID)
		if err != nil {
			return fmt.Errorf("update webhook secret for %s: %w", r.ID, err)
		}
		migrated++
	}

	if migrated > 0 {
		fmt.Printf("🔐 Migrated %d webhook secret(s) to envelope encryption\n", migrated)
	}
	return nil
}
