package webhooks

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/sarathsp06/sparrow/db"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	storePostgres "github.com/sarathsp06/sparrow/pkg/storage/postgres"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func TestPushEvent_Integration(t *testing.T) {
	// DSN for the test database
	dbName := "riverqueue"
	dbUser := "riveruser"
	dbPassword := "riverpass"

	postgresContainer, err := postgres.Run(t.Context(),
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		postgres.BasicWaitStrategies(),
	)
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()
	if err != nil {
		log.Printf("failed to start container: %s", err)
		return
	}

	ctx := t.Context()
	dsn, err := postgresContainer.ConnectionString(t.Context(), "sslmode=disable")
	require.NoError(t, err)

	driver, err := iofs.New(db.GetMigrationsFS(), "migrations")
	require.NoError(t, err)

	// Create migrate instance
	m, err := migrate.NewWithSourceInstance("iofs", driver, dsn)
	require.NoError(t, err)
	err = m.Up()
	require.NoError(t, err)

	// Run river migrations
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close() //nolint:errcheck

	// Create database connection pool
	dbPool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Failed to create database pool: %v", err)
	}

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		log.Fatalf("Failed to connect to database: %v", err)
	}
	rm, err := rivermigrate.New(riverdatabasesql.New(db), nil)
	require.NoError(t, err)
	_, err = rm.Migrate(context.Background(), rivermigrate.DirectionUp, nil)
	require.NoError(t, err)

	sqlxD, err := storePostgres.Open(dsn, 3)
	require.NoError(t, err)
	webhookRepo := store.NewRepository(sqlxD)

	// Create a new queue manager
	qm, err := queue.NewManager(context.Background(), webhookRepo, dbPool)
	require.NoError(t, err)

	// Start the queue manager
	go func() {
		// This will fail because the river tables don't exist yet, but that's ok
		_ = qm.Start(context.Background())
	}()
	defer func() { _ = qm.Stop(context.Background()) }()

	// Create a new webhook service
	service := NewWebhookService(qm.GetJobInserter(), webhookRepo)

	// Create a channel to receive the webhook payload
	payloadChan := make(chan []byte, 1)

	// Create a test HTTP server to act as the webhook endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			return
		}
		payloadChan <- body
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	namespace := "test-namespace"
	eventName := "test-event"
	eventPayload := map[string]any{"key": "value"}

	// Register an event
	_, _, err = service.RegisterEvent(ctx, eventName, "description", map[string]any{}, nil, true)
	require.NoError(t, err)

	// Register a webhook
	_, _, err = service.RegisterWebhook(ctx, namespace, []string{eventName}, server.URL, nil, 30, true, "description")
	require.NoError(t, err)

	// Push an event
	_, err = service.PushEvent(ctx, namespace, eventName, eventPayload, 3600, nil)
	require.NoError(t, err)

	// Wait for the webhook to be called and assert the payload
	select {
	case payload := <-payloadChan:
		var data struct {
			Payload map[string]any `json:"payload"`
		}
		err := json.Unmarshal(payload, &data)
		require.NoError(t, err)
		assert.Equal(t, eventPayload, data.Payload)
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for webhook to be called")
	}
}
