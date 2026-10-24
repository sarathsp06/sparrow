//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/tenant"
)

// TestSystemEventsStayInTheirTenant covers the one pass that runs across
// every tenant: health evaluation and auto-disable. A failing webhook in
// tenant B is auto-disabled; its sparrow.webhook.* events must be recorded
// in tenant B, and the default tenant, whose webhook is healthy, must get
// none and keep its webhook active.
func TestSystemEventsStayInTheirTenant(t *testing.T) {
	t.Setenv("SPARROW_AUTO_DISABLE_AFTER", "1s")
	t.Setenv("SPARROW_AUTO_DISABLE_MIN_FAILURES", "1")
	ctx := context.Background()
	sp := startEmbeddedSparrow(t)
	tenantB := sp.addTenant(t, "tenant-b")

	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(failing.Close)
	healthy := newCountingReceiver(t)

	setup := func(tenantID uuid.UUID, url string) string {
		c := sp.client(t, tenantID)
		resp, err := c.post(ctx, "/v1/event-types", map[string]any{"name": "order.created", "active": true}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		var wh struct {
			WebhookID string `json:"webhook_id"`
		}
		resp, err = c.post(ctx, "/v1/consumers/acme/webhooks", map[string]any{
			"events": []string{"order.created"}, "url": url + "/hook", "active": true,
			"http_config": map[string]any{"max_retries": 10, "retry_backoff_seconds": 1},
		}, &wh)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		resp, err = c.post(ctx, "/v1/consumers/acme/events?event=order.created", map[string]any{"payload": map[string]any{"n": 1}}, nil)
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		return wh.WebhookID
	}
	failingWebhook := setup(tenantB, failing.URL)
	healthyWebhook := setup(uuid.Nil, healthy.srv.URL)
	healthy.waitFor(t, 1)

	// B's failing webhook is auto-disabled, and the event about it is
	// recorded in B.
	require.Eventually(t, func() bool {
		var n int
		require.NoError(t, sp.db.Get(&n, `
			SELECT count(*) FROM event_records
			WHERE tenant_id = $1 AND event = 'sparrow.webhook.disabled' AND consumer = $2`, tenantB, tenant.SystemConsumer))
		return n == 1
	}, 45*time.Second, 250*time.Millisecond, "tenant B never got its sparrow.webhook.disabled event")

	var active bool
	require.NoError(t, sp.db.Get(&active, `SELECT active FROM webhook_registrations WHERE id = $1`, failingWebhook))
	require.False(t, active, "B's failing webhook should be auto-disabled")
	require.NoError(t, sp.db.Get(&active, `SELECT active FROM webhook_registrations WHERE id = $1`, healthyWebhook))
	require.True(t, active, "the default tenant's healthy webhook was disabled")

	var payloadTenantEvents int
	require.NoError(t, sp.db.Get(&payloadTenantEvents, `
		SELECT count(*) FROM event_records
		WHERE event = 'sparrow.webhook.disabled' AND payload::jsonb->>'webhook_id' = $1 AND tenant_id <> $2`, failingWebhook, tenantB))
	require.Zero(t, payloadTenantEvents, "B's webhook event was recorded in another tenant")

	var defaultSystemEvents int
	require.NoError(t, sp.db.Get(&defaultSystemEvents, `
		SELECT count(*) FROM event_records WHERE tenant_id = $1 AND event LIKE 'sparrow.%'`, tenant.DefaultTenantID))
	require.Zero(t, defaultSystemEvents, "the healthy default tenant received system events")
}
