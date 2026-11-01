//go:build integration

package integration

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/riverqueue/river"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// TestFanOutIdempotentAndAtomic covers the fan-out's two guarantees: running
// it again for an event (River retrying it after a crash or a lost
// completion) creates no second delivery and sends nothing twice, and the
// delivery rows and their River jobs commit or roll back together.
func TestFanOutIdempotentAndAtomic(t *testing.T) {
	env := setupEnv(t)
	c := newRESTClient(t, env)
	ctx := context.Background()

	const (
		consumer  = "fanout-idem"
		eventName = "fanout.idem"
	)
	target, hits := startCountingTarget(t)
	registerEventType(t, c, ctx, eventName)
	webhookID := registerWebhookPipeline(t, c, ctx, consumer, eventName, target.URL, 0)

	eventID := pushTestEvent(t, c, ctx, consumer, eventName)
	pollCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	pollDeliveryStatus(t, c, pollCtx, consumer, eventID, func(d deliveryItem) bool { return d.Status == "success" })
	require.EqualValues(t, 1, hits.Load())

	countRows := func(query string, args ...any) int {
		var n int
		require.NoError(t, env.sqlxDB.GetContext(ctx, &n, query, args...))
		return n
	}

	t.Run("re-running the fan-out delivers nothing twice", func(t *testing.T) {
		_, err := env.queueMgr.GetJobInserter().Insert(ctx, queue.EventArgs{
			TenantID: tenant.DefaultTenantID.String(), EventID: eventID, Consumer: consumer, Event: eventName, CreatedAt: time.Now(),
		})
		require.NoError(t, err)

		require.Eventually(t, func() bool {
			return countRows(`SELECT count(*) FROM river_job WHERE kind = 'event_processing' AND args->>'event_id' = $1 AND state = 'completed'`, eventID) == 2
		}, 30*time.Second, 200*time.Millisecond, "second fan-out never completed")

		require.Equal(t, 1, countRows(`SELECT count(*) FROM webhook_deliveries WHERE event_id = $1`, eventID))
		require.Equal(t, 1, countRows(`SELECT count(*) FROM river_job WHERE kind = 'webhook_delivery' AND args->>'event_id' = $1`, eventID))
		time.Sleep(time.Second)
		require.EqualValues(t, 1, hits.Load(), "the receiver got the event twice")
	})

	t.Run("rows and jobs commit together", func(t *testing.T) {
		repo := store.NewRepository(env.sqlxDB)
		inserter := env.queueMgr.GetJobInserter()
		delivery := func() *store.WebhookDelivery {
			return &store.WebhookDelivery{
				ID: uuid.New(), WebhookID: uuid.MustParse(webhookID), EventID: uuid.MustParse(eventID),
				Status: store.StatusPaused, MaxAttempts: 1, ExpiresAt: store.NoExpiryTime,
			}
		}
		// The jobs are already expired, so a committed one is never sent:
		// the test is about what exists, not what is delivered.
		enqueue := func(fail error) store.FanOutEnqueuer {
			return func(ctx context.Context, tx *sql.Tx, inserted []*store.WebhookDelivery) error {
				jobs := make([]river.JobArgs, 0, len(inserted))
				for _, d := range inserted {
					jobs = append(jobs, &queue.WebhookArgs{
						TenantID: tenant.DefaultTenantID.String(), DeliveryID: d.ID.String(), WebhookID: webhookID,
						EventID: eventID, Consumer: "fanout-idem-none", MaxAttempts: 1, ExpiresAt: time.Now().Add(-time.Hour),
					})
				}
				if err := inserter.BatchInsertTx(ctx, tx, jobs); err != nil {
					return err
				}
				return fail
			}
		}
		exists := func(id uuid.UUID) (rows, jobs int) {
			return countRows(`SELECT count(*) FROM webhook_deliveries WHERE id = $1`, id),
				countRows(`SELECT count(*) FROM river_job WHERE kind = 'webhook_delivery' AND args->>'delivery_id' = $1`, id.String())
		}

		failed := delivery()
		_, err := repo.CreateFanOutDeliveries(ctx, tenant.DefaultTenantID, []*store.WebhookDelivery{failed}, enqueue(errors.New("boom")))
		require.Error(t, err)
		rows, jobs := exists(failed.ID)
		require.Zero(t, rows, "delivery row survived a failed enqueue")
		require.Zero(t, jobs, "job survived a failed fan-out transaction")

		ok := delivery()
		inserted, err := repo.CreateFanOutDeliveries(ctx, tenant.DefaultTenantID, []*store.WebhookDelivery{ok}, enqueue(nil))
		require.NoError(t, err)
		require.Len(t, inserted, 1)
		rows, jobs = exists(ok.ID)
		require.Equal(t, 1, rows)
		require.Equal(t, 1, jobs)

		// More deliveries than one INSERT can bind (65535 parameters at
		// 12 per row) go in chunks, all in the fan-out's one transaction.
		many := make([]*store.WebhookDelivery, 6000)
		for i := range many {
			many[i] = delivery()
		}
		inserted, err = repo.CreateFanOutDeliveries(ctx, tenant.DefaultTenantID, many, func(context.Context, *sql.Tx, []*store.WebhookDelivery) error { return nil })
		require.NoError(t, err)
		require.Len(t, inserted, len(many))
		require.Equal(t, len(many), countRows(`SELECT count(*) FROM webhook_deliveries WHERE id = ANY($1)`, pq.Array(idsOf(many))))

		// Same ID again: nothing new, nothing enqueued.
		again, err := repo.CreateFanOutDeliveries(ctx, tenant.DefaultTenantID, []*store.WebhookDelivery{ok}, enqueue(nil))
		require.NoError(t, err)
		require.Empty(t, again)
		_, jobs = exists(ok.ID)
		require.Equal(t, 1, jobs)
	})
}

func idsOf(ds []*store.WebhookDelivery) []string {
	ids := make([]string, len(ds))
	for i, d := range ds {
		ids[i] = d.ID.String()
	}
	return ids
}
