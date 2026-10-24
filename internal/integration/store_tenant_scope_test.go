//go:build integration

package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/sarathsp06/sparrow/internal/storage"
	storePg "github.com/sarathsp06/sparrow/internal/storage/postgres"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// storeScopeFixture is one webhook with an event, a failed delivery and a
// batch job, all owned by the default tenant, plus a second tenant that owns
// nothing.
type storeScopeFixture struct {
	ctx      context.Context
	db       *sqlx.DB
	repo     *store.Repository
	owner    uuid.UUID
	other    uuid.UUID
	webhook  *store.WebhookRegistration
	event    *store.EventRecord
	delivery *store.WebhookDelivery
	batch    *store.BatchJob
}

func newStoreScopeFixture(t *testing.T) *storeScopeFixture {
	t.Helper()
	ctx := context.Background()
	databaseURL, _ := setupTestDB(t, ctx)
	runMigrations(t, ctx, databaseURL)
	db, err := storePg.Open(databaseURL, 3)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, tenant.Bootstrap(ctx, db))

	f := &storeScopeFixture{ctx: ctx, db: db, repo: store.NewRepository(db), owner: tenant.DefaultTenantID, other: uuid.New()}
	require.NoError(t, f.repo.CreateTenant(ctx, f.other, "other"))

	rps := 1000.0
	f.webhook = &store.WebhookRegistration{
		Consumer: "acme", URL: "https://example.com/hook", Active: true, RateLimitRPS: &rps,
		SignatureType: store.SignatureTypeHMAC, RetryBackoffSeconds: 60, RequestTimeoutSeconds: 30,
	}
	require.NoError(t, f.repo.RegisterWebhook(ctx, f.owner, f.webhook))
	require.NoError(t, f.repo.RegisterEvent(ctx, f.owner, &store.EventRegistration{Name: "order.created", Active: true}))
	f.event = &store.EventRecord{
		ID: uuid.New(), Consumer: "acme", Event: "order.created", Payload: store.JSONMap{"n": 1},
		TTL: 3600, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, f.repo.StoreEvent(ctx, f.owner, f.event))
	f.delivery = &store.WebhookDelivery{
		ID: uuid.New(), WebhookID: f.webhook.ID, EventID: f.event.ID,
		Status: store.StatusFailed, ExpiresAt: time.Now().Add(time.Hour),
	}
	require.NoError(t, f.repo.CreateDelivery(ctx, f.owner, f.delivery))
	f.batch, err = f.repo.CreateBatchJob(ctx, f.owner, "acme", store.BatchTypeDeliveryRetry, &store.BatchJobData{ItemIDs: []string{f.delivery.ID.String()}})
	require.NoError(t, err)
	return f
}

func (f *storeScopeFixture) deliveryRow(t *testing.T) (status string, attempts int, templateErr sql.NullString) {
	t.Helper()
	require.NoError(t, f.db.QueryRowx(`SELECT status, attempt_count, template_error FROM webhook_deliveries WHERE id = $1`, f.delivery.ID).
		Scan(&status, &attempts, &templateErr))
	return status, attempts, templateErr
}

func (f *storeScopeFixture) count(t *testing.T, query string, args ...any) int {
	t.Helper()
	var n int
	require.NoError(t, f.db.Get(&n, query, args...))
	return n
}

// TestStoreMethodsScopeByTenant calls every storage method that tenant-scopes
// by its second argument (the ones commit "tenant is the second argument"
// added the parameter to) as a tenant that does not own the row, and checks
// that nothing changes and nothing is returned; then calls it as the owner
// and checks that it works, so no case passes vacuously. The signature test
// only proves the parameter exists; this proves the SQL uses it.
func TestStoreMethodsScopeByTenant(t *testing.T) {
	f := newStoreScopeFixture(t)
	ctx, repo, owner, other := f.ctx, f.repo, f.owner, f.other

	t.Run("UpdateDeliveryStatus", func(t *testing.T) {
		require.NoError(t, repo.UpdateDeliveryStatus(ctx, other, f.delivery.ID, store.StatusSuccess, 200, "", "", ""))
		status, attempts, _ := f.deliveryRow(t)
		require.Equal(t, string(store.StatusFailed), status, "another tenant changed the delivery status")
		require.Zero(t, attempts)

		require.NoError(t, repo.UpdateDeliveryStatus(ctx, owner, f.delivery.ID, store.StatusFailed, 500, "", "boom", "server_error"))
		_, attempts, _ = f.deliveryRow(t)
		require.Equal(t, 1, attempts, "the owner's update did not apply")
	})

	t.Run("UpdateDeliveryTemplateError", func(t *testing.T) {
		require.NoError(t, repo.UpdateDeliveryTemplateError(ctx, other, f.delivery.ID, "bad template"))
		_, _, te := f.deliveryRow(t)
		require.False(t, te.Valid, "another tenant set the template error")

		require.NoError(t, repo.UpdateDeliveryTemplateError(ctx, owner, f.delivery.ID, "bad template"))
		_, _, te = f.deliveryRow(t)
		require.Equal(t, "bad template", te.String)
		require.NoError(t, repo.UpdateDeliveryTemplateError(ctx, owner, f.delivery.ID, ""))
	})

	t.Run("RecordDeliveryAttempt and RecordWebhookHealthEvent", func(t *testing.T) {
		healthEvents := func() int {
			return f.count(t, `SELECT count(*) FROM webhook_health_events WHERE webhook_id = $1`, f.webhook.ID)
		}
		before := healthEvents()
		_, attemptsBefore, _ := f.deliveryRow(t)
		attempt := store.DeliveryAttempt{
			DeliveryID: f.delivery.ID, WebhookID: f.webhook.ID, UpdateStatus: true,
			Status: store.StatusFailed, ResponseCode: 500, ErrorCategory: "server_error", ResponseTimeMs: 10,
		}
		require.NoError(t, repo.RecordDeliveryAttempt(ctx, other, attempt))
		require.NoError(t, repo.RecordWebhookHealthEvent(ctx, other, f.webhook.ID, f.delivery.ID, false, 10, 500, "", "server_error"))
		_, attempts, _ := f.deliveryRow(t)
		require.Equal(t, attemptsBefore, attempts, "another tenant recorded an attempt on the delivery")
		require.Equal(t, before, healthEvents(), "another tenant recorded a health event on the webhook")

		require.NoError(t, repo.RecordDeliveryAttempt(ctx, owner, attempt))
		_, attempts, _ = f.deliveryRow(t)
		require.Equal(t, attemptsBefore+1, attempts)
		require.Equal(t, before+1, healthEvents())
	})

	t.Run("HoldDelivery", func(t *testing.T) {
		require.NoError(t, repo.HoldDelivery(ctx, other, f.delivery.ID, "paused"))
		status, _, _ := f.deliveryRow(t)
		require.NotEqual(t, string(store.StatusPaused), status, "another tenant held the delivery")

		require.NoError(t, repo.HoldDelivery(ctx, owner, f.delivery.ID, "paused"))
		status, _, _ = f.deliveryRow(t)
		require.Equal(t, string(store.StatusPaused), status)
	})

	t.Run("ResetDeliveriesForRetry", func(t *testing.T) {
		enqueued := 0
		enqueue := func(context.Context, *sql.Tx, []uuid.UUID) error { enqueued++; return nil }
		reset, err := repo.ResetDeliveriesForRetry(ctx, other, []uuid.UUID{f.delivery.ID}, enqueue)
		require.NoError(t, err)
		require.Empty(t, reset, "another tenant reset the delivery")
		require.Zero(t, enqueued, "another tenant queued a retry")

		reset, err = repo.ResetDeliveriesForRetry(ctx, owner, []uuid.UUID{f.delivery.ID}, enqueue)
		require.NoError(t, err)
		require.Equal(t, []uuid.UUID{f.delivery.ID}, reset)
		require.Equal(t, 1, enqueued)
	})

	t.Run("health state, summary and auto-disable", func(t *testing.T) {
		// Fold the recorded failures into health state. Events carry the
		// database clock, which can run ahead of the host's, so take upTo
		// from the database.
		var upTo time.Time
		require.NoError(t, f.db.Get(&upTo, `SELECT now() + interval '2 seconds'`))
		_, err := repo.EvaluateHealth(ctx, upTo, 0)
		require.NoError(t, err)

		_, err = repo.GetWebhookHealthState(ctx, other, f.webhook.ID)
		require.True(t, storage.IsNotFound(err), "another tenant read the health state: err=%v", err)
		state, err := repo.GetWebhookHealthState(ctx, owner, f.webhook.ID)
		require.NoError(t, err)
		require.Positive(t, state.ConsecutiveFailures)

		summary, err := repo.GetWebhookHealthSummary(ctx, other, f.webhook.ID, 24)
		require.NoError(t, err)
		require.Zero(t, summary.TotalDeliveries, "another tenant saw the health summary")
		summary, err = repo.GetWebhookHealthSummary(ctx, owner, f.webhook.ID, 24)
		require.NoError(t, err)
		require.Positive(t, summary.TotalDeliveries)

		disabled, err := repo.AutoDisableWebhook(ctx, other, f.webhook.ID, 1, 0)
		require.NoError(t, err)
		require.Nil(t, disabled, "another tenant auto-disabled the webhook")
		require.Equal(t, 1, f.count(t, `SELECT count(*) FROM webhook_registrations WHERE id = $1 AND active`, f.webhook.ID))

		disabled, err = repo.AutoDisableWebhook(ctx, owner, f.webhook.ID, 1, 0)
		require.NoError(t, err)
		require.NotNil(t, disabled)
		require.Zero(t, f.count(t, `SELECT count(*) FROM webhook_registrations WHERE id = $1 AND active`, f.webhook.ID))
	})

	// Regression: with no attempts in the window, the error-category SUMs
	// were NULL and the summary failed to scan (the API dropped it and
	// logged an error on every health request).
	t.Run("health summary of a webhook quiet for the whole window", func(t *testing.T) {
		_, err := f.db.Exec(`UPDATE webhook_health_events SET timestamp = timestamp - interval '25 hours' WHERE webhook_id = $1`, f.webhook.ID)
		require.NoError(t, err)
		summary, err := repo.GetWebhookHealthSummary(ctx, owner, f.webhook.ID, 24)
		require.NoError(t, err)
		require.Zero(t, summary.TotalDeliveries)
		require.Zero(t, summary.ServerErrors)
	})

	t.Run("rate limit state", func(t *testing.T) {
		rows := func() int {
			return f.count(t, `SELECT count(*) FROM webhook_rate_limit_state WHERE webhook_id = $1`, f.webhook.ID)
		}
		require.NoError(t, repo.DeleteRateLimitState(ctx, owner, f.webhook.ID))
		require.Zero(t, rows())

		require.NoError(t, repo.UpsertRateLimitState(ctx, other, f.webhook.ID))
		require.Zero(t, rows(), "another tenant created rate limit state")
		wait, rps, err := repo.AcquireDeliverySlot(ctx, other, f.webhook.ID)
		require.NoError(t, err)
		require.Zero(t, wait)
		require.Zero(t, rps, "another tenant saw the webhook's rate limit")
		require.Zero(t, rows(), "another tenant's slot request created state")

		require.NoError(t, repo.UpsertRateLimitState(ctx, owner, f.webhook.ID))
		require.Equal(t, 1, rows())
		_, rps, err = repo.AcquireDeliverySlot(ctx, owner, f.webhook.ID)
		require.NoError(t, err)
		require.InDelta(t, 1000.0, rps, 0.001)

		require.NoError(t, repo.DeleteRateLimitState(ctx, other, f.webhook.ID))
		require.Equal(t, 1, rows(), "another tenant deleted rate limit state")
		require.NoError(t, repo.DeleteRateLimitState(ctx, owner, f.webhook.ID))
		require.Zero(t, rows())
	})

	t.Run("batch status and progress", func(t *testing.T) {
		err := repo.UpdateBatchJobStatus(ctx, other, f.batch.ID, store.BatchStatusPending, store.BatchStatusProcessing)
		require.True(t, storage.IsNotFound(err), "another tenant moved the batch: err=%v", err)
		require.NoError(t, repo.UpdateBatchJobProgress(ctx, other, f.batch.ID, 5, 5))
		got, err := repo.GetBatchJob(ctx, owner, f.batch.ID)
		require.NoError(t, err)
		require.Equal(t, store.BatchStatusPending, got.Status)
		require.Zero(t, got.Processed, "another tenant advanced the batch")

		require.NoError(t, repo.UpdateBatchJobStatus(ctx, owner, f.batch.ID, store.BatchStatusPending, store.BatchStatusProcessing))
		require.NoError(t, repo.UpdateBatchJobProgress(ctx, owner, f.batch.ID, 1, 0))
		got, err = repo.GetBatchJob(ctx, owner, f.batch.ID)
		require.NoError(t, err)
		require.Equal(t, store.BatchStatusProcessing, got.Status)
		require.Equal(t, 1, got.Processed)

		foreign, err := repo.GetBatchJob(ctx, other, f.batch.ID)
		require.NoError(t, err)
		require.Nil(t, foreign, "another tenant read the batch job")
	})

	t.Run("CreateAlertConfig writes under the argument tenant", func(t *testing.T) {
		// cfg claims the owner, but the call is made for the other tenant:
		// the argument wins.
		cfg := &store.AlertConfig{TenantID: owner, Consumer: "acme", Email: "ops@example.com", EventTypes: []string{"sparrow.webhook.disabled"}}
		require.NoError(t, repo.CreateAlertConfig(ctx, other, cfg))
		require.Equal(t, other, cfg.TenantID)

		ownerConfigs, err := repo.ListAlertConfigs(ctx, owner, "acme", nil)
		require.NoError(t, err)
		require.Empty(t, ownerConfigs, "the alert config leaked into the owner's tenant")
		otherConfigs, err := repo.ListAlertConfigs(ctx, other, "acme", nil)
		require.NoError(t, err)
		require.Len(t, otherConfigs, 1)
	})
}
