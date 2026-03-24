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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	connectserver "github.com/sarathsp06/sparrow/internal/connect"
	grpcserver "github.com/sarathsp06/sparrow/internal/grpc"
	"github.com/sarathsp06/sparrow/internal/health"
	"github.com/sarathsp06/sparrow/internal/migration"
	"github.com/sarathsp06/sparrow/internal/namespace"
	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/ui"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/storage/postgres"
	pb "github.com/sarathsp06/sparrow/proto"
	pbconnect "github.com/sarathsp06/sparrow/proto/protoconnect"
)

func main() {
	// Load .env file if present (ignored if missing)
	_ = godotenv.Load()

	ctx := context.Background()
	startTime := time.Now() // Track service start time for uptime calculation

	// Configure OpenTelemetry
	otelConfig := observability.DefaultConfig()

	if env := os.Getenv("ENVIRONMENT"); env != "" {
		otelConfig.Environment = env
	}

	if otlpEndpoint := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); otlpEndpoint != "" {
		otelConfig.OTLPEndpoint = otlpEndpoint
	}

	// Initialize OpenTelemetry
	fmt.Println("🔭 Initializing OpenTelemetry...")
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
		fmt.Printf("✅ OpenTelemetry initialized (endpoint: %s, env: %s)\n", otelConfig.OTLPEndpoint, otelConfig.Environment)
	}

	// Database connection URL
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://localhost/riverqueue?sslmode=disable"
		fmt.Println("🔧 Using default database URL. Set DATABASE_URL environment variable for custom connection.")
	}

	// Run database migrations before anything else touches the DB.
	// This covers both River queue schema and application schema migrations.
	// golang-migrate uses PostgreSQL advisory locks, so concurrent server
	// instances won't conflict.
	migrationLogger := slog.Default()
	fmt.Println("📦 Running database migrations...")
	if err := migration.RunAllMigrations(ctx, databaseURL, "up", 0, 0, migrationLogger); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}
	fmt.Println("✅ Database migrations completed")

	// Create database connection pool for River queue workers.
	// River runs 45 concurrent workers (20 events + 20 webhooks + 5 default)
	// so we need enough connections to avoid starvation. The default pgxpool
	// MaxConns is max(4, NumCPU) which is far too low.
	pgxConfig, err := pgxpool.ParseConfig(databaseURL)
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

	sqlxDB, err := postgres.Open(databaseURL, 3,
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

	// Create namespace repository and service
	namespaceRepo := namespace.NewRepository(sqlxDB)

	// Create namespace service
	namespaceSvc := namespace.NewService(namespaceRepo, namespace.WithServiceLogger(logger))

	// Create webhook repository
	webhookRepo := store.NewRepositoryInterfaceWithTracing(store.NewRepository(sqlxDB), "")

	// Initialize queue manager
	queueManager, err := queue.NewManager(ctx, webhookRepo, dbPool)
	if err != nil {
		log.Fatalf("Failed to create queue manager: %v", err)
	}
	defer func() { _ = queueManager.Stop(ctx) }()

	// Start the queue processing
	if err := queueManager.Start(ctx); err != nil {
		log.Fatalf("Failed to start queue manager: %v", err)
	}

	fmt.Println("🚀 River queue started successfully")

	// Initialize gRPC server with OpenTelemetry instrumentation
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.MaxRecvMsgSize(4<<20), // 4 MB max inbound message
		grpc.MaxSendMsgSize(4<<20), // 4 MB max outbound message
	)

	webhookService := webhooks.NewWebhookService(queueManager.GetJobInserter(), webhookRepo)

	webhookGRPCServer := grpcserver.NewWebhookServer(webhooks.NewWebhookServiceInterfaceWithTracing(webhookService, ""))
	pb.RegisterWebhookServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterEventServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterSubscriptionServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterDeliveryServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterHealthServiceServer(grpcServer, webhookGRPCServer)

	// Register Namespace gRPC service
	namespaceGRPCServer := grpcserver.NewNamespaceServer(namespaceSvc)
	pb.RegisterNamespaceServiceServer(grpcServer, namespaceGRPCServer)

	// Initialize Connect-RPC server
	webhookConnectServer := connectserver.NewWebhookConnectServer(webhookGRPCServer, webhookGRPCServer, webhookGRPCServer, webhookGRPCServer, webhookGRPCServer)

	// Create Connect-RPC adapter for namespace services
	namespaceConnectServer := connectserver.NewNamespaceConnectServer(namespaceGRPCServer)

	// Create HTTP mux for Connect-RPC
	mux := http.NewServeMux()
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		log.Fatal(err)
	}

	options := connect.WithInterceptors(otelInterceptor)
	// Enforce request/response body size limits on Connect-RPC handlers
	readLimit := connect.WithReadMaxBytes(4 << 20) // 4 MB
	sendLimit := connect.WithSendMaxBytes(4 << 20) // 4 MB
	mux.Handle(pbconnect.NewWebhookServiceHandler(webhookConnectServer, options, readLimit, sendLimit))
	mux.Handle(pbconnect.NewEventServiceHandler(webhookConnectServer, options, readLimit, sendLimit))
	mux.Handle(pbconnect.NewSubscriptionServiceHandler(webhookConnectServer, options, readLimit, sendLimit))
	mux.Handle(pbconnect.NewDeliveryServiceHandler(webhookConnectServer, options, readLimit, sendLimit))
	mux.Handle(pbconnect.NewHealthServiceHandler(webhookConnectServer, options, readLimit, sendLimit))
	mux.Handle(pbconnect.NewNamespaceServiceHandler(namespaceConnectServer, options, readLimit, sendLimit))

	// Initialize health checker
	healthChecker := health.NewChecker(dbPool, queueManager, startTime)

	// Add health and readiness endpoints
	mux.HandleFunc("/health", healthChecker.HealthHandler())
	mux.HandleFunc("/ready", healthChecker.ReadyHandler())

	// Serve embedded web UI if enabled
	serveUI := os.Getenv("SPARROW_SERVE_UI") == "true" || os.Getenv("SPARROW_SERVE_UI") == "1"
	if serveUI {
		if ui.Available() {
			mux.Handle("/", ui.Handler(logger, nil))
			fmt.Println("🖥️  Embedded web UI enabled at http://localhost:8080/")
		} else {
			fmt.Println("⚠️  SPARROW_SERVE_UI=true but no frontend build found. Build with: cd web && npm run build:static")
		}
	}

	// Create HTTP server with OpenTelemetry instrumentation
	corsHandler := buildCORSHandler()

	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           corsHandler.Handler(mux),
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
		ReadHeaderTimeout: 10 * time.Second, // Mitigate slowloris attacks
		MaxHeaderBytes:    1 << 20,          // 1 MB max header size
	}

	// Start gRPC server
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen on port 50051: %v", err)
	}

	fmt.Println("🌐 Starting servers...")
	fmt.Println("   gRPC server: localhost:50051")
	fmt.Println("   Connect-RPC (HTTP): localhost:8080")

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
	fmt.Println("   gRPC server: localhost:50051")
	fmt.Println("   Connect-RPC (HTTP): localhost:8080")
	fmt.Println("   Health check: http://localhost:8080/health")
	fmt.Println("   Readiness check: http://localhost:8080/ready")
	if serveUI && ui.Available() {
		fmt.Println("   Web UI: http://localhost:8080/")
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

// buildCORSHandler creates a CORS handler configured via the CORS_ALLOWED_ORIGINS
// environment variable. When set, only the listed origins (comma-separated) are
// allowed. When unset or empty, all origins are allowed (development mode).
func buildCORSHandler() *cors.Cors {
	originsEnv := os.Getenv("CORS_ALLOWED_ORIGINS")
	if originsEnv == "" {
		fmt.Println("⚠️  CORS_ALLOWED_ORIGINS not set — allowing all origins (not recommended for production)")
		return cors.AllowAll()
	}

	var origins []string
	for _, o := range strings.Split(originsEnv, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			origins = append(origins, o)
		}
	}

	if len(origins) == 0 {
		fmt.Println("⚠️  CORS_ALLOWED_ORIGINS is empty — allowing all origins (not recommended for production)")
		return cors.AllowAll()
	}

	fmt.Printf("🔒 CORS allowed origins: %v\n", origins)
	return cors.New(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "Connect-Protocol-Version", "Connect-Timeout-Ms", "Grpc-Timeout", "X-Grpc-Web", "X-User-Agent"},
		AllowCredentials: true,
		MaxAge:           300,
	})
}
