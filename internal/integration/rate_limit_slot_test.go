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
// semantics: a busy bucket consumes nothing and keeps returning the same tail
// until the slot time passes; only a free bucket advances.
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
		Namespace:             "rate-limit-slot-test",
		URL:                   "https://example.com/hook",
		Active:                true,
		RateLimitRPS:          &rps,
		SignatureType:         store.SignatureTypeHMAC,
		RetryBackoffSeconds:   60,
		RequestTimeoutSeconds: 30,
	}
	require.NoError(t, repo.RegisterWebhook(ctx, tenant.DefaultTenantID, reg))

	// No state row yet: no rate limit configured.
	next, gotRPS, err := repo.AcquireDeliverySlot(ctx, reg.ID)
	require.NoError(t, err)
	require.True(t, next.IsZero())
	require.Zero(t, gotRPS)

	require.NoError(t, repo.UpsertRateLimitState(ctx, reg.ID))

	// Free bucket: granted. Returned tail minus one interval is our slot,
	// which must not be in the future.
	interval := time.Duration(float64(time.Second) / rps)
	next, gotRPS, err = repo.AcquireDeliverySlot(ctx, reg.ID)
	require.NoError(t, err)
	require.Equal(t, rps, gotRPS)
	require.False(t, next.Add(-interval).After(time.Now()), "first acquire must grant immediately")

	// Busy bucket: repeated calls must NOT advance the tail (no slot burn).
	busy1, _, err := repo.AcquireDeliverySlot(ctx, reg.ID)
	require.NoError(t, err)
	busy2, _, err := repo.AcquireDeliverySlot(ctx, reg.ID)
	require.NoError(t, err)
	require.True(t, busy1.Equal(busy2), "busy acquires must return the same tail, got %v then %v", busy1, busy2)
	require.True(t, busy1.After(next.Add(-time.Millisecond)), "busy tail must not rewind")

	// Once the slot time passes, the bucket grants again.
	time.Sleep(time.Until(busy1.Add(-interval)) + 50*time.Millisecond)
	granted, _, err := repo.AcquireDeliverySlot(ctx, reg.ID)
	require.NoError(t, err)
	require.False(t, granted.Add(-interval).After(time.Now()), "acquire after waiting out the tail must grant")
}
