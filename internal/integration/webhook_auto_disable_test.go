//go:build integration

package integration

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// TestWebhookAutoDisable exercises the auto-disable SQL against Postgres: the
// failure run a failing receiver accumulates, the threshold check that pauses
// the webhook exactly once, the fields the REST API shows, and the fresh
// window a resume gives.
func TestWebhookAutoDisable(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()
	repo := store.NewRepository(env.sqlxDB)

	const (
		consumer  = "auto-disable-test"
		eventName = "auto.disable"
	)

	targetSrv, _ := startAlwaysFailTarget(t, http.StatusInternalServerError)
	registerEventType(t, c, ctx, eventName)
	webhookID := registerWebhookPipeline(t, c, ctx, consumer, eventName, targetSrv.URL, 2)
	id := uuid.MustParse(webhookID)

	eventID := pushTestEvent(t, c, ctx, consumer, eventName)
	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	pollDeliveryStatus(t, c, pollCtx, consumer, eventID, func(d deliveryItem) bool { return d.Status == "failed" })

	// Three failed attempts started a failure run.
	var state struct {
		ConsecutiveFailures int        `db:"consecutive_failures"`
		FailingSince        *time.Time `db:"failing_since"`
	}
	readState := func() {
		t.Helper()
		require.NoError(t, env.sqlxDB.GetContext(ctx, &state,
			`SELECT consecutive_failures, failing_since FROM webhook_health_state WHERE webhook_id = $1`, id))
	}
	// Health state is folded in by the periodic evaluator (2s in tests).
	require.Eventually(t, func() bool {
		var n int
		if err := env.sqlxDB.GetContext(ctx, &n, `SELECT consecutive_failures FROM webhook_health_state WHERE webhook_id = $1`, id); err != nil {
			return false
		}
		return n == 3
	}, 30*time.Second, 200*time.Millisecond, "evaluator should fold the three failed attempts into the failure run")
	readState()
	require.Equal(t, 3, state.ConsecutiveFailures)
	require.NotNil(t, state.FailingSince, "a failure run must record when it started")

	// Not enough failures, or not failing for long enough: stays active.
	got, err := repo.AutoDisableWebhook(ctx, id, 100, time.Minute)
	require.NoError(t, err)
	assert.Nil(t, got, "below the failure minimum")
	got, err = repo.AutoDisableWebhook(ctx, id, 3, time.Hour)
	require.NoError(t, err)
	assert.Nil(t, got, "failing for less than the window")

	// Failing for longer than the window: paused once, with a reason.
	_, err = env.sqlxDB.ExecContext(ctx,
		`UPDATE webhook_health_state SET failing_since = NOW() - INTERVAL '2 hours' WHERE webhook_id = $1`, id)
	require.NoError(t, err)
	got, err = repo.AutoDisableWebhook(ctx, id, 3, time.Hour)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, 3, got.ConsecutiveFailures)
	assert.Contains(t, got.Reason, "3 failed attempts in a row")

	again, err := repo.AutoDisableWebhook(ctx, id, 3, time.Hour)
	require.NoError(t, err)
	assert.Nil(t, again, "an already disabled webhook is not disabled twice")

	type webhookOut struct {
		Active             bool   `json:"active"`
		AutoDisabledAt     string `json:"auto_disabled_at"`
		AutoDisabledReason string `json:"auto_disabled_reason"`
	}
	var out webhookOut
	resp, err := c.get(ctx, "/v1/consumers/"+consumer+"/webhooks/"+webhookID, &out)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.False(t, out.Active)
	assert.NotEmpty(t, out.AutoDisabledAt)
	assert.Equal(t, got.Reason, out.AutoDisabledReason)

	// An event for the disabled webhook is held, not dropped.
	heldEventID := pushTestEvent(t, c, ctx, consumer, eventName)
	pollDeliveryStatus(t, c, pollCtx, consumer, heldEventID, func(d deliveryItem) bool { return d.Status == "paused" })

	counts, err := repo.CountWebhooksByState(ctx, tenant.DefaultTenantID)
	require.NoError(t, err)
	assert.Contains(t, counts, store.WebhookStateCount{Health: "unknown", Status: "auto_disabled", Count: 1})

	// Resuming clears the marker and restarts the failure run, so the next
	// failure cannot disable it again straight away.
	var resumed struct {
		PausedDeliveries int `json:"paused_deliveries"`
	}
	resp, err = c.post(ctx, "/v1/consumers/"+consumer+"/webhooks/"+webhookID+":resume", map[string]any{}, &resumed)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, 1, resumed.PausedDeliveries)

	out = webhookOut{}
	_, err = c.get(ctx, "/v1/consumers/"+consumer+"/webhooks/"+webhookID, &out)
	require.NoError(t, err)
	assert.True(t, out.Active)
	assert.Empty(t, out.AutoDisabledAt)
	assert.Empty(t, out.AutoDisabledReason)

	readState()
	assert.Nil(t, state.FailingSince, "resume must restart the failure window")
	got, err = repo.AutoDisableWebhook(ctx, id, 3, time.Hour)
	require.NoError(t, err)
	assert.Nil(t, got, "a resumed webhook gets a full window again")
}
