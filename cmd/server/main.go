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
	"syscall"
	"time"

	"connectrpc.com/connect"
	"connectrpc.com/otelconnect"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/cors"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/sarathsp06/sparrow/internal/auth"
	connectserver "github.com/sarathsp06/sparrow/internal/connect"
	grpcserver "github.com/sarathsp06/sparrow/internal/grpc"
	"github.com/sarathsp06/sparrow/internal/health"
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

	// Create database connection pool
	dbPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Failed to create database pool: %v", err)
	}

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		log.Fatalf("Failed to connect to database: %v", err)
	}

	sqlxDB, err := postgres.Open(databaseURL, 3)
	if err != nil {
		log.Fatalf("Failed to open sqlx database: %v", err)
	}
	defer sqlxDB.Close() //nolint:errcheck

	// ---- Auth configuration ----
	authEnabled := os.Getenv("SPARROW_AUTH_ENABLED") == "true" || os.Getenv("SPARROW_AUTH_ENABLED") == "1"
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

	// Build auth interceptor config
	authInterceptorCfg := auth.AuthInterceptorConfig{
		Enabled: authEnabled,
		SkipProcedures: map[string]bool{
			"/sparrow.HealthService/Check":       true,
			"/sparrow.HealthService/CheckHealth": true,
		},
		Logger: logger,
	}

	if authEnabled {
		apiKeyAuthenticator := auth.NewAPIKeyAuthenticator(tenantRepo)
		authenticators := []auth.Authenticator{apiKeyAuthenticator}

		// If a JWKS URL is configured, add JWT authentication alongside API keys.
		// This enables OIDC-based auth (Clerk, Auth0, Keycloak, etc.) for the
		// web UI while keeping API keys for programmatic/M2M access.
		if jwksURL := os.Getenv("SPARROW_JWKS_URL"); jwksURL != "" {
			jwksProvider := auth.NewJWKSProvider(jwksURL)

			// Build claims config from env vars with Clerk-compatible defaults
			claimsCfg := auth.DefaultJWTClaimsConfig()
			if v := os.Getenv("SPARROW_JWT_TENANT_CLAIM"); v != "" {
				claimsCfg.TenantClaim = v
			}
			if v := os.Getenv("SPARROW_JWT_ROLE_CLAIM"); v != "" {
				claimsCfg.RoleClaim = v
			}
			if v := os.Getenv("SPARROW_JWT_ISSUER"); v != "" {
				claimsCfg.Issuer = v
			}

			// Create tenant resolver with caching
			tenantResolver := auth.NewCachingTenantResolver(tenantRepo)

			jwtAuthenticator := auth.NewJWTAuthenticator(
				jwksProvider,
				auth.WithClaimsConfig(claimsCfg),
				auth.WithTenantResolver(tenantResolver),
			)

			// JWT authenticator goes first — API keys are tried if JWT fails
			authenticators = append([]auth.Authenticator{jwtAuthenticator}, authenticators...)
			fmt.Printf("🔑 JWT authentication enabled (JWKS: %s)\n", jwksURL)
		}

		authInterceptorCfg.Authenticators = authenticators
		fmt.Println("🔒 Authentication enabled")
	} else {
		fmt.Println("🔓 Authentication disabled (all requests use default tenant)")
	}

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

	// Initialize gRPC server with OpenTelemetry instrumentation and auth
	grpcServer := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(auth.NewGRPCUnaryInterceptor(authInterceptorCfg)),
	)

	webhookService := webhooks.NewWebhookService(queueManager.GetJobInserter(), webhookRepo)

	webhookGRPCServer := grpcserver.NewWebhookServer(webhooks.NewWebhookServiceInterfaceWithTracing(webhookService, ""))
	pb.RegisterWebhookServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterEventServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterSubscriptionServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterDeliveryServiceServer(grpcServer, webhookGRPCServer)
	pb.RegisterHealthServiceServer(grpcServer, webhookGRPCServer)

	// Register Tenant and API Key gRPC services
	tenantGRPCServer := grpcserver.NewTenantServer(tenantSvc)
	pb.RegisterTenantServiceServer(grpcServer, tenantGRPCServer)
	pb.RegisterAPIKeyServiceServer(grpcServer, tenantGRPCServer)

	// Initialize Connect-RPC server
	webhookConnectServer := connectserver.NewWebhookConnectServer(webhookGRPCServer, webhookGRPCServer, webhookGRPCServer, webhookGRPCServer, webhookGRPCServer)

	// Create Connect-RPC adapter for tenant/API key services
	tenantConnectServer := connectserver.NewTenantConnectServer(tenantGRPCServer, tenantGRPCServer)

	// Create HTTP mux for Connect-RPC
	mux := http.NewServeMux()
	otelInterceptor, err := otelconnect.NewInterceptor()
	if err != nil {
		log.Fatal(err)
	}

	authConnectInterceptor := auth.NewConnectInterceptor(authInterceptorCfg)
	options := connect.WithInterceptors(otelInterceptor, authConnectInterceptor)
	mux.Handle(pbconnect.NewWebhookServiceHandler(webhookConnectServer, options))
	mux.Handle(pbconnect.NewEventServiceHandler(webhookConnectServer, options))
	mux.Handle(pbconnect.NewSubscriptionServiceHandler(webhookConnectServer, options))
	mux.Handle(pbconnect.NewDeliveryServiceHandler(webhookConnectServer, options))
	mux.Handle(pbconnect.NewHealthServiceHandler(webhookConnectServer, options))
	mux.Handle(pbconnect.NewTenantServiceHandler(tenantConnectServer, options))
	mux.Handle(pbconnect.NewAPIKeyServiceHandler(tenantConnectServer, options))

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
	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      cors.AllowAll().Handler(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
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
