//go:build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	storePg "github.com/sarathsp06/sparrow/pkg/storage/postgres"
)

// Regression test for the rate-limit livelock: AcquireDeliverySlot used to
// advance next_delivery_at unconditionally, so a snoozed delivery burned a new
// slot on every wake and, for RPS < ~0.5, could re-snooze forever. The fixed
// semantics: a busy bucket consumes nothing and returns the wait until the
// tail; only a free bucket advances. The wait is computed on the database
// clock: comparing a DB timestamp against the host clock made a granted slot
// look busy whenever Postgres ran ahead of the worker, burning it.
func TestAcquireDeliverySlot_BusyBucketDoesNotBurnSlots(t *testing.T) {
	ctx := context.Background()
	databaseURL, _ := setupTestDB(t, ctx)
	runMigrations(t, ctx, databaseURL)

	sqlxDB, err := storePg.Open(databaseURL, 3)
	require.NoError(t, err)
	t.Cleanup(func() { _ = sqlxDB.Close() })

	require.NoError(t, tenant.Bootstrap(ctx, sqlxDB))
	repo := store.NewRepository(sqlxDB)

	rps := 2.0 // interval = 500ms
	reg := &store.WebhookRegistration{
		Consumer:              "rate-limit-slot-test",
		URL:                   "https://example.com/hook",
		Active:                true,
		RateLimitRPS:          &rps,
		SignatureType:         store.SignatureTypeHMAC,
		RetryBackoffSeconds:   60,
		RequestTimeoutSeconds: 30,
	}
	require.NoError(t, repo.RegisterWebhook(ctx, tenant.DefaultTenantID, reg))

	tail := func() time.Time {
		var at time.Time
		require.NoError(t, sqlxDB.GetContext(ctx, &at,
			`SELECT next_delivery_at FROM webhook_rate_limit_state WHERE webhook_id = $1`, reg.ID))
		return at
	}

	// No state row yet: no rate limit configured.
	wait, gotRPS, err := repo.AcquireDeliverySlot(ctx, reg.ID)
	require.NoError(t, err)
	require.Zero(t, wait)
	require.Zero(t, gotRPS)

	require.NoError(t, repo.UpsertRateLimitState(ctx, reg.ID))

	// Free bucket: granted immediately and the tail advances.
	interval := time.Duration(float64(time.Second) / rps)
	wait, gotRPS, err = repo.AcquireDeliverySlot(ctx, reg.ID)
	require.NoError(t, err)
	require.Equal(t, rps, gotRPS)
	require.Zero(t, wait, "first acquire must grant immediately")
	grantedTail := tail()

	// Busy bucket: repeated calls must NOT advance the tail (no slot burn)
	// and must ask the caller to wait at most one interval.
	busy1, _, err := repo.AcquireDeliverySlot(ctx, reg.ID)
	require.NoError(t, err)
	busy2, _, err := repo.AcquireDeliverySlot(ctx, reg.ID)
	require.NoError(t, err)
	require.True(t, tail().Equal(grantedTail), "busy acquires must not advance the tail")
	for _, w := range []time.Duration{busy1, busy2} {
		require.Greater(t, w, time.Duration(0), "busy bucket must ask the caller to wait")
		require.LessOrEqual(t, w, interval, "busy wait must not exceed one interval")
	}
	require.LessOrEqual(t, busy2, busy1, "busy wait must count down, not grow")

	// Once the wait passes, the bucket grants again.
	time.Sleep(busy2 + 50*time.Millisecond)
	wait, _, err = repo.AcquireDeliverySlot(ctx, reg.ID)
	require.NoError(t, err)
	require.Zero(t, wait, "acquire after waiting out the tail must grant")
	require.True(t, tail().After(grantedTail), "a grant must advance the tail")
}
