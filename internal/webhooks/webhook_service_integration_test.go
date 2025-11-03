package webhooks

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"database/sql"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/riverqueue/river/riverdriver/riverdatabasesql"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPushEvent_Integration(t *testing.T) {
	// DSN for the test database
	dsn := "postgres://riveruser:riverpass@localhost:5432/riverqueue?sslmode=disable"

	// Run sparrow migrations
	m, err := migrate.New("file://../../db/migrations", dsn)
	require.NoError(t, err)
	_ = m.Down()
	err = m.Up()
	require.NoError(t, err)

	// Run river migrations
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	rm, err := rivermigrate.New(riverdatabasesql.New(db), nil)
	require.NoError(t, err)
	_, err = rm.Migrate(context.Background(), rivermigrate.DirectionUp, nil)
	require.NoError(t, err)

	// Create a new queue manager
	qm, err := queue.NewManager(context.Background(), dsn)
	require.NoError(t, err)

	// Start the queue manager
	go func() {
		// This will fail because the river tables don't exist yet, but that's ok
		_ = qm.Start(context.Background())
	}()
	defer qm.Stop(context.Background())

	// Create a new webhook service
	service := NewWebhookService(qm, qm.GetWebhookRepo())

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

	ctx := context.Background()
	namespace := "test-namespace"
	eventName := "test-event"
	eventPayload := `{"key":"value"}`

	// Register an event
	_, _, err = service.RegisterEvent(ctx, eventName, "description", "", nil, true)
	require.NoError(t, err)

	// Register a webhook
	_, _, err = service.RegisterWebhook(ctx, namespace, []string{eventName}, server.URL, nil, 30, true, "description")
	require.NoError(t, err)

	// Push an event
	_, _, _, err = service.PushEvent(ctx, namespace, eventName, eventPayload, 3600, nil)
	require.NoError(t, err)

	// Wait for the webhook to be called and assert the payload
	select {
	case payload := <-payloadChan:
		var data struct {
			Payload json.RawMessage `json:"payload"`
		}
		err := json.Unmarshal(payload, &data)
		require.NoError(t, err)
		assert.JSONEq(t, eventPayload, string(data.Payload))
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for webhook to be called")
	}
}
