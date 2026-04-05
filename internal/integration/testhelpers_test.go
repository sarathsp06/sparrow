//go:build integration

package integration

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	connectserver "github.com/sarathsp06/sparrow/internal/connect"
	grpcserver "github.com/sarathsp06/sparrow/internal/grpc"
	"github.com/sarathsp06/sparrow/internal/migration"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	webhookclient "github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/crypto"
	storePg "github.com/sarathsp06/sparrow/pkg/storage/postgres"
	pbconnect "github.com/sarathsp06/sparrow/proto/protoconnect"
)

const (
	testDBName     = "sparrow_integration"
	testDBUser     = "sparrow"
	testDBPassword = "sparrow"
)

// testEnv holds all resources needed for an integration test.
type testEnv struct {
	// Database
	pgContainer testcontainers.Container
	databaseURL string

	// Pools
	pgxPool *pgxpool.Pool
	sqlxDB  *storePg.SQLXDB

	// Services
	webhookSvc webhooks.WebhookServiceInterface
	tenantSvc  *tenant.Service
	queueMgr   *queue.Manager

	// HTTP server
	httpServer *http.Server
	baseURL    string
}

// setupTestDB starts a Postgres container via testcontainers and returns the
// connection string. The container is automatically cleaned up when t finishes.
func setupTestDB(t *testing.T, ctx context.Context) (string, testcontainers.Container) {
	t.Helper()

	ctr, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase(testDBName),
		postgres.WithUsername(testDBUser),
		postgres.WithPassword(testDBPassword),
		postgres.BasicWaitStrategies(),
		postgres.WithSQLDriver("pgx"),
	)
	require.NoError(t, err, "failed to start postgres container")
	testcontainers.CleanupContainer(t, ctr)

	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err, "failed to get connection string")

	return connStr, ctr
}

// runMigrations applies all database migrations (River + app).
func runMigrations(t *testing.T, ctx context.Context, databaseURL string) {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	err := migration.RunAllMigrations(ctx, databaseURL, "up", 0, 0, logger)
	require.NoError(t, err, "failed to run migrations")
}

// setupEnv creates the full server stack (DB, repos, services, queue, HTTP)
// wired together for integration testing.
func setupEnv(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()

	// 1. Start Postgres via testcontainers
	databaseURL, pgCtr := setupTestDB(t, ctx)

	// 2. Run migrations
	runMigrations(t, ctx, databaseURL)

	// 3. Create pgxpool for River queue
	pgxConfig, err := pgxpool.ParseConfig(databaseURL)
	require.NoError(t, err)
	pgxConfig.MaxConns = 20
	pgxConfig.MinConns = 2

	pgxPool, err := pgxpool.NewWithConfig(ctx, pgxConfig)
	require.NoError(t, err)

	// 4. Create sqlx pool for application queries
	sqlxDB, err := storePg.Open(databaseURL, 3,
		storePg.WithMaxOpenConnections(10),
		storePg.WithMaxIdleConnections(10),
		storePg.WithConnectionMaxLifeTime(5*time.Minute),
	)
	require.NoError(t, err)

	// 5. Create repositories
	webhookRepo := store.NewRepository(sqlxDB)
	tenantRepo := tenant.NewRepository(sqlxDB)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// 6. Create services
	tenantSvc := tenant.NewService(tenantRepo)

	// 7. Bootstrap default tenant
	bootstrapCfg := tenant.DefaultBootstrapConfig()
	bootstrapCfg.Logger = logger
	err = tenant.Bootstrap(ctx, tenantSvc, bootstrapCfg)
	require.NoError(t, err, "failed to bootstrap default tenant")

	// 8. Create crypto service with a test encryption key
	_, testKey, err := crypto.GenerateKey()
	require.NoError(t, err, "failed to generate test encryption key")
	cryptoSvc, err := crypto.NewService(testKey)
	require.NoError(t, err, "failed to create crypto service")

	// 9. Initialize queue manager (River)
	// Allow private networks so httptest.NewServer (127.0.0.1) targets work.
	clientCfg := webhookclient.DefaultConfig()
	clientCfg.AllowPrivateNetworks = true

	queueMgr, err := queue.NewManager(ctx, webhookRepo, cryptoSvc, pgxPool, clientCfg)
	require.NoError(t, err, "failed to create queue manager")

	err = queueMgr.Start(ctx)
	require.NoError(t, err, "failed to start queue manager")

	// 10. Create webhook service
	webhookSvc := webhooks.NewWebhookService(queueMgr.GetJobInserter(), webhookRepo, cryptoSvc, webhooks.WithAllowPrivateNetworks(true))

	// 11. Create gRPC servers (needed for Connect-RPC adapters)
	webhookGRPCServer := grpcserver.NewWebhookServer(webhooks.NewWebhookServiceInterfaceWithTracing(webhookSvc, ""))

	// 12. Create Connect-RPC adapters
	webhookConnectServer := connectserver.NewWebhookConnectServer(
		webhookGRPCServer, webhookGRPCServer, webhookGRPCServer,
		webhookGRPCServer, webhookGRPCServer,
	)

	// 13. Create HTTP mux and register all Connect-RPC handlers
	mux := http.NewServeMux()

	mux.Handle(pbconnect.NewWebhookServiceHandler(webhookConnectServer))
	mux.Handle(pbconnect.NewEventServiceHandler(webhookConnectServer))
	mux.Handle(pbconnect.NewSubscriptionServiceHandler(webhookConnectServer))
	mux.Handle(pbconnect.NewDeliveryServiceHandler(webhookConnectServer))
	mux.Handle(pbconnect.NewHealthServiceHandler(webhookConnectServer))

	// 14. Start HTTP server on a random free port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	httpServer := &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		if serveErr := httpServer.Serve(listener); serveErr != nil && serveErr != http.ErrServerClosed {
			t.Logf("HTTP server error: %v", serveErr)
		}
	}()

	baseURL := fmt.Sprintf("http://%s", listener.Addr().String())

	env := &testEnv{
		pgContainer: pgCtr,
		databaseURL: databaseURL,
		pgxPool:     pgxPool,
		sqlxDB:      sqlxDB,
		webhookSvc:  webhookSvc,
		tenantSvc:   tenantSvc,
		queueMgr:    queueMgr,
		httpServer:  httpServer,
		baseURL:     baseURL,
	}

	t.Cleanup(func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		_ = queueMgr.Stop(shutdownCtx)
		sqlxDB.Close()
	})

	return env
}
