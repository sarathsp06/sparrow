//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/tenant"
)

// pollSystemEvent polls event_records directly (no REST surface exposes
// _sparrow's own events to tenants) until one matching eventName appears for
// consumer, or ctx expires. Returns its decoded JSON payload.
func pollSystemEvent(t *testing.T, ctx context.Context, env *testEnv, consumer, eventName string) map[string]any {
	t.Helper()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for system event %s/%s", consumer, eventName)
		case <-ticker.C:
			var payloadJSON string
			err := env.sqlxDB.GetContext(ctx, &payloadJSON,
				`SELECT payload FROM event_records
				 WHERE tenant_id = $1 AND consumer = $2 AND event = $3
				 ORDER BY created_at DESC LIMIT 1`,
				tenant.DefaultTenantID, consumer, eventName)
			if err != nil {
				continue
			}
			var payload map[string]any
			require.NoError(t, json.Unmarshal([]byte(payloadJSON), &payload))
			return payload
		}
	}
}

// TestE2E_WebhookHealthAlerts is the end-to-end smoke test for Sparrow's
// self-generated webhook health alerts: an opted-in email recipient must
// receive both a health_changed event (on the healthy->unhealthy transition)
// and a delivery_failed event (once retries are exhausted), each carrying
// the recipient in payload.alert_recipients — Sparrow's own event pipeline,
// scoped to the internal "_sparrow" consumer.
func TestE2E_WebhookHealthAlerts(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		consumer  = "alerts-smoke-test"
		eventName = "alerts.smoke"
		email     = "ops@example.com"
	)

	// Opt in consumer-wide, for both system event types.
	var alertOut struct {
		ID string `json:"id"`
	}
	resp, err := c.post(ctx, "/v1/consumers/"+consumer+"/alert-configs", map[string]any{
		"email":       email,
		"event_types": []string{"sparrow.webhook.health_changed", "sparrow.webhook.delivery_failed"},
	}, &alertOut)
	require.NoError(t, err, "CreateAlertConfig failed")
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	require.NotEmpty(t, alertOut.ID)

	// A webhook that always fails. consecutive_failures increments on every
	// attempt (not just the terminal one), so 6 max attempts guarantees it
	// crosses the >=5 threshold that flips health unknown -> unhealthy,
	// exercising the health_changed emission before the delivery finally
	// exhausts its retries and delivery_failed fires.
	targetSrv, _ := startAlwaysFailTarget(t, http.StatusInternalServerError)
	registerEventType(t, c, ctx, eventName)
	registerWebhookPipeline(t, c, ctx, consumer, eventName, targetSrv.URL, 6)

	eventID := pushTestEvent(t, c, ctx, consumer, eventName)

	pollCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	delivery := pollDeliveryStatus(t, c, pollCtx, consumer, eventID, func(d deliveryItem) bool {
		return d.Status == "failed"
	})
	assert.Equal(t, "server_error", delivery.ErrorCategory)

	// health_changed: unknown -> unhealthy is skipped only for unknown ->
	// healthy; a brand-new webhook's first failure still transitions it away
	// from "unknown", so exactly this transition must be reported.
	healthPayload := pollSystemEvent(t, pollCtx, env, "_sparrow", "sparrow.webhook.health_changed")
	assertHasRecipient(t, healthPayload, email)
	assert.Equal(t, consumer, healthPayload["consumer"])

	failedPayload := pollSystemEvent(t, pollCtx, env, "_sparrow", "sparrow.webhook.delivery_failed")
	assertHasRecipient(t, failedPayload, email)
	assert.Equal(t, consumer, failedPayload["consumer"])
	assert.Equal(t, "server_error", failedPayload["error_category"])
}

func assertHasRecipient(t *testing.T, payload map[string]any, email string) {
	t.Helper()
	recipients, ok := payload["alert_recipients"].([]any)
	require.True(t, ok, "alert_recipients missing or wrong shape: %v", payload["alert_recipients"])
	for _, r := range recipients {
		if m, ok := r.(map[string]any); ok && m["email"] == email {
			return
		}
	}
	t.Fatalf("expected %s among alert_recipients, got %v", email, recipients)
}
