//go:build integration

package integration

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// TestE2E_EventDeliveryStats_NoDeliveries is a regression test for the NULL
// aggregate scan bug in GetEventDeliveryStats.
//
// When an event is pushed but produces zero webhook deliveries (no matching
// subscription), the stats query aggregates over zero rows. Bare SUM(...) over
// an empty set returns SQL NULL, which fails to scan into a non-nullable int32
// with:
//
//	sql: Scan error on column index 1, name "successful_deliveries":
//	converting NULL to int32 is unsupported
//
// The columns must COALESCE to 0 so an event with no deliveries reports zeros
// rather than erroring.
func TestE2E_EventDeliveryStats_NoDeliveries(t *testing.T) {
	env := setupEnv(t)
	ctx := context.Background()

	// Push an event into a namespace that has no webhook subscribed to it.
	// PushEvent auto-registers the event type and stores the event record; with
	// no matching subscription the event processing worker creates zero
	// deliveries — the exact state that triggered the reported scan error.
	eventID, isDuplicate, _, _, err := env.webhookSvc.PushEvent(
		ctx, "orphan-ns", "orphan.event",
		map[string]any{"hello": "world"},
		0, nil, nil, nil,
	)
	require.NoError(t, err, "push event")
	require.False(t, isDuplicate)
	require.NotEmpty(t, eventID)

	eventUUID, err := uuid.Parse(eventID)
	require.NoError(t, err)

	// Direct store assertion: this is the call that previously failed with the
	// NULL->int32 scan error for an event with no deliveries.
	repo := store.NewRepository(env.sqlxDB)
	webhookCount, successful, failed, pending, err := repo.GetEventDeliveryStats(ctx, tenant.DefaultTenantID, eventUUID)
	require.NoError(t, err, "GetEventDeliveryStats must not error for an event with zero deliveries")
	assert.Equal(t, int32(0), webhookCount, "webhook_count")
	assert.Equal(t, int32(0), successful, "successful_deliveries")
	assert.Equal(t, int32(0), failed, "failed_deliveries")
	assert.Equal(t, int32(0), pending, "pending_deliveries")

	// Service-level path: GetEventRecord returns the record with zeroed stats
	// and no error (it degrades gracefully, but with the fix the underlying
	// stats call no longer logs an error).
	record, wc, sc, fc, pc, err := env.webhookSvc.GetEventRecord(ctx, eventID)
	require.NoError(t, err, "GetEventRecord")
	require.NotNil(t, record, "event record should exist")
	assert.Equal(t, eventID, record.ID.String())
	assert.Equal(t, int32(0), wc)
	assert.Equal(t, int32(0), sc)
	assert.Equal(t, int32(0), fc)
	assert.Equal(t, int32(0), pc)
}
