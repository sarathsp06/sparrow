//go:build integration

package integration

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// setDeliveryStatus moves the fixture's delivery to status.
func (f *storeScopeFixture) setDeliveryStatus(t *testing.T, status store.WebhookDeliveryStatus) {
	t.Helper()
	_, err := f.db.Exec(`UPDATE webhook_deliveries SET status = $2 WHERE id = $1`, f.delivery.ID, string(status))
	require.NoError(t, err)
}

// TestResetDeliveriesForRetry_Claim pins which deliveries a retry resets for
// each status, and that a retrying one is reset only when its waiting job was
// claimed.
func TestResetDeliveriesForRetry_Claim(t *testing.T) {
	f := newStoreScopeFixture(t)
	noEnqueue := func(context.Context, *sql.Tx, []uuid.UUID) error { return nil }

	cases := []struct {
		status    store.WebhookDeliveryStatus
		claims    bool // the claimer cancels a waiting job for it
		wantClaim bool // the claimer is asked about it
		wantReset bool
	}{
		{store.StatusRetrying, true, true, true},
		{store.StatusRetrying, false, true, false}, // its job was already picked up
		{store.StatusPending, true, false, false},  // never offered to the claimer
		{store.StatusSending, true, false, false},
		{store.StatusFailed, false, false, true},
		{store.StatusPaused, false, false, true},
		{store.StatusExpired, false, false, true},
		{store.StatusSuccess, false, false, true}, // force is the service's check, not the store's
	}
	for _, tc := range cases {
		name := string(tc.status)
		if tc.claims {
			name += "/claimed"
		}
		t.Run(name, func(t *testing.T) {
			f.setDeliveryStatus(t, tc.status)
			var asked []uuid.UUID
			claim := func(_ context.Context, _ *sql.Tx, retrying []uuid.UUID) ([]uuid.UUID, error) {
				asked = append(asked, retrying...)
				if tc.claims {
					return retrying, nil
				}
				return nil, nil
			}
			reset, err := f.repo.ResetDeliveriesForRetry(f.ctx, f.owner, []uuid.UUID{f.delivery.ID}, claim, noEnqueue)
			require.NoError(t, err)

			assert.Equal(t, tc.wantClaim, len(asked) == 1, "claimer asked about the delivery")
			status, _, _ := f.deliveryRow(t)
			if tc.wantReset {
				assert.Equal(t, []uuid.UUID{f.delivery.ID}, reset)
				assert.Equal(t, string(store.StatusPending), status)
			} else {
				assert.Empty(t, reset)
				assert.Equal(t, string(tc.status), status, "a delivery not reset keeps its status")
			}
		})
	}

	t.Run("nil claim skips retrying deliveries", func(t *testing.T) {
		f.setDeliveryStatus(t, store.StatusRetrying)
		reset, err := f.repo.ResetDeliveriesForRetry(f.ctx, f.owner, []uuid.UUID{f.delivery.ID}, nil, noEnqueue)
		require.NoError(t, err)
		assert.Empty(t, reset)
	})

	t.Run("another tenant's retrying delivery is not offered to the claimer", func(t *testing.T) {
		f.setDeliveryStatus(t, store.StatusRetrying)
		claim := func(context.Context, *sql.Tx, []uuid.UUID) ([]uuid.UUID, error) {
			t.Fatal("claimer called for another tenant's delivery")
			return nil, nil
		}
		reset, err := f.repo.ResetDeliveriesForRetry(f.ctx, f.other, []uuid.UUID{f.delivery.ID}, claim, noEnqueue)
		require.NoError(t, err)
		assert.Empty(t, reset)
	})

	t.Run("a failing claim or enqueue rolls everything back", func(t *testing.T) {
		boom := errors.New("boom")
		f.setDeliveryStatus(t, store.StatusRetrying)
		_, err := f.repo.ResetDeliveriesForRetry(f.ctx, f.owner, []uuid.UUID{f.delivery.ID},
			func(context.Context, *sql.Tx, []uuid.UUID) ([]uuid.UUID, error) { return nil, boom }, noEnqueue)
		require.ErrorIs(t, err, boom)
		status, _, _ := f.deliveryRow(t)
		assert.Equal(t, string(store.StatusRetrying), status)

		claimAll := func(_ context.Context, _ *sql.Tx, ids []uuid.UUID) ([]uuid.UUID, error) { return ids, nil }
		_, err = f.repo.ResetDeliveriesForRetry(f.ctx, f.owner, []uuid.UUID{f.delivery.ID}, claimAll,
			func(context.Context, *sql.Tx, []uuid.UUID) error { return boom })
		require.ErrorIs(t, err, boom)
		status, _, _ = f.deliveryRow(t)
		assert.Equal(t, string(store.StatusRetrying), status, "the reset rolled back with the failed enqueue")
	})
}

// TestCancelWaitingDeliveryJobsTx checks which River jobs a retry cancels:
// the delivery's waiting ones, never a running one, nor another delivery's,
// and the cancel commits or rolls back with the transaction.
func TestCancelWaitingDeliveryJobsTx(t *testing.T) {
	f := newStoreScopeFixture(t)
	txClient, err := queue.NewTxInsertClient()
	require.NoError(t, err)
	inserter := queue.NewJobInserter(nil, txClient)

	// addJob inserts a delivery job in state, scheduled an hour out so no
	// worker touches it, and returns its id.
	addJob := func(t *testing.T, deliveryID uuid.UUID, queueName, state string) int64 {
		t.Helper()
		var id int64
		require.NoError(t, f.db.QueryRowx(`
			INSERT INTO river_job (args, kind, max_attempts, queue, state, scheduled_at, attempt)
			VALUES (jsonb_build_object('delivery_id', $1::text), 'webhook_delivery', 4, $2, $3::river_job_state, now() + interval '1 hour', 1)
			RETURNING id`, deliveryID.String(), queueName, state).Scan(&id))
		return id
	}
	jobState := func(t *testing.T, id int64) string {
		t.Helper()
		var s string
		require.NoError(t, f.db.QueryRowx(`SELECT state FROM river_job WHERE id = $1`, id).Scan(&s))
		return s
	}
	cancel := func(t *testing.T, commit bool, ids ...uuid.UUID) []uuid.UUID {
		t.Helper()
		tx, err := f.db.BeginTx(f.ctx, nil)
		require.NoError(t, err)
		claimed, err := inserter.CancelWaitingDeliveryJobsTx(f.ctx, tx, ids)
		require.NoError(t, err)
		if commit {
			require.NoError(t, tx.Commit())
		} else {
			require.NoError(t, tx.Rollback())
		}
		return claimed
	}

	t.Run("waiting jobs are cancelled", func(t *testing.T) {
		for _, state := range []string{"retryable", "scheduled", "available"} {
			d := uuid.New()
			id := addJob(t, d, queue.QueueWebhookDelivery, state)
			assert.Equal(t, []uuid.UUID{d}, cancel(t, true, d), state)
			assert.Equal(t, "cancelled", jobState(t, id), state)
		}
	})

	t.Run("a running job is left to its worker", func(t *testing.T) {
		d := uuid.New()
		id := addJob(t, d, queue.QueueWebhookDelivery, "running")
		assert.Empty(t, cancel(t, true, d))
		assert.Equal(t, "running", jobState(t, id))
	})

	t.Run("only the named deliveries' webhook jobs are touched", func(t *testing.T) {
		mine, other := uuid.New(), uuid.New()
		mineID := addJob(t, mine, queue.QueueWebhookDelivery, "retryable")
		otherID := addJob(t, other, queue.QueueWebhookDelivery, "retryable")
		otherQueueID := addJob(t, mine, queue.QueueEventProcessing, "retryable")
		assert.Equal(t, []uuid.UUID{mine}, cancel(t, true, mine))
		assert.Equal(t, "cancelled", jobState(t, mineID))
		assert.Equal(t, "retryable", jobState(t, otherID))
		assert.Equal(t, "retryable", jobState(t, otherQueueID))
	})

	t.Run("two waiting jobs for one delivery are both cancelled, the delivery claimed once", func(t *testing.T) {
		d := uuid.New()
		a := addJob(t, d, queue.QueueWebhookDelivery, "retryable")
		b := addJob(t, d, queue.QueueWebhookDelivery, "scheduled")
		assert.Equal(t, []uuid.UUID{d}, cancel(t, true, d))
		assert.Equal(t, "cancelled", jobState(t, a))
		assert.Equal(t, "cancelled", jobState(t, b))
	})

	t.Run("a rolled-back transaction keeps the job", func(t *testing.T) {
		d := uuid.New()
		id := addJob(t, d, queue.QueueWebhookDelivery, "retryable")
		assert.Equal(t, []uuid.UUID{d}, cancel(t, false, d))
		assert.Equal(t, "retryable", jobState(t, id))
	})

	t.Run("no deliveries is a no-op", func(t *testing.T) {
		assert.Empty(t, cancel(t, true))
	})
}
