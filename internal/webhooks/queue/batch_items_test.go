package queue

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

var errFake = errors.New("fake failure")

// recordingInserter records inserted jobs and fails when told to.
type recordingInserter struct {
	fail bool
	jobs []river.JobArgs
}

func (r *recordingInserter) Insert(_ context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	if r.fail {
		return nil, errFake
	}
	r.jobs = append(r.jobs, args)
	return &rivertype.JobInsertResult{}, nil
}

func (r *recordingInserter) BatchInsert(context.Context, []river.JobArgs) ([]*rivertype.JobInsertResult, error) {
	return nil, nil
}

func (r *recordingInserter) BatchInsertTx(_ context.Context, _ *sql.Tx, args []river.JobArgs) error {
	if r.fail {
		return errFake
	}
	r.jobs = append(r.jobs, args...)
	return nil
}

// scriptedEventRepo serves the events it holds and fails on request.
type scriptedEventRepo struct {
	store.EventRepository
	events     map[uuid.UUID]*store.EventRecord
	storeErr   error
	deleteErr  error
	stored     []*store.EventRecord
	deleted    []uuid.UUID
	purgeCount int64
	purgeErr   error
	cutoff     time.Time
}

func (r *scriptedEventRepo) GetEventByID(_ context.Context, _, id uuid.UUID) (*store.EventRecord, error) {
	if e, ok := r.events[id]; ok {
		return e, nil
	}
	return nil, errFake
}

// StoreEventAndEnqueue keeps the event only when enqueue succeeds (the
// transaction).
func (r *scriptedEventRepo) StoreEventAndEnqueue(ctx context.Context, _ uuid.UUID, e *store.EventRecord, enqueue store.EventEnqueuer) error {
	if r.storeErr != nil {
		return r.storeErr
	}
	if err := enqueue(ctx, nil); err != nil {
		return err
	}
	r.stored = append(r.stored, e)
	return nil
}

func (r *scriptedEventRepo) DeleteEventByID(_ context.Context, _, id uuid.UUID) error {
	r.deleted = append(r.deleted, id)
	return r.deleteErr
}

func (r *scriptedEventRepo) DeleteEventsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	r.cutoff = cutoff
	return r.purgeCount, r.purgeErr
}

func TestProcessEventRepushCountsEachOutcome(t *testing.T) {
	good, system := uuid.New(), uuid.New()
	events := &scriptedEventRepo{events: map[uuid.UUID]*store.EventRecord{
		good:   {ID: good, Consumer: "acme", Event: "order.created", TTL: 60, Labels: map[string]string{"region": "eu"}},
		system: {ID: system, Consumer: "_sparrow", Event: "sparrow.webhook.disabled"},
	}}
	ins := &recordingInserter{}
	w := &BatchJobWorker{batchRepo: &fakeBatchRepo{batch: &store.BatchJob{}}, eventRepo: events, jobInserter: ins, logger: slog.Default()}

	processed, failed := w.processEventRepush(context.Background(), uuid.New(), uuid.New(),
		[]string{good.String(), "not-a-uuid", uuid.NewString(), system.String()})

	if processed != 1 || failed != 3 {
		t.Fatalf("processed=%d failed=%d, want 1 re-pushed and 3 failed (bad id, missing, system event)", processed, failed)
	}
	if len(events.stored) != 1 || events.stored[0].ID == good || events.stored[0].Event != "order.created" {
		t.Fatalf("stored %+v, want one copy of the good event under a fresh id", events.stored)
	}
	args, ok := ins.jobs[0].(EventArgs)
	if !ok || args.EventID != events.stored[0].ID.String() || args.TTLSeconds != 60 || args.Labels["region"] != "eu" {
		t.Fatalf("queued %#v, want an EventArgs for the new event carrying TTL and labels", ins.jobs[0])
	}
}

// The copy and its job commit together: a failed enqueue stores nothing.
func TestProcessEventRepushFailedEnqueueStoresNothing(t *testing.T) {
	id := uuid.New()
	events := &scriptedEventRepo{
		events: map[uuid.UUID]*store.EventRecord{id: {ID: id, Consumer: "acme", Event: "order.created"}},
	}
	w := &BatchJobWorker{eventRepo: events, jobInserter: &recordingInserter{fail: true}, logger: slog.Default()}

	processed, failed := w.processEventRepush(context.Background(), uuid.New(), uuid.New(), []string{id.String()})

	if processed != 0 || failed != 1 {
		t.Fatalf("processed=%d failed=%d, want the item failed", processed, failed)
	}
	if len(events.stored) != 0 {
		t.Fatalf("stored %v without a queued job", events.stored)
	}
}

func TestProcessEventRepushStoreFailure(t *testing.T) {
	id := uuid.New()
	events := &scriptedEventRepo{
		events:   map[uuid.UUID]*store.EventRecord{id: {ID: id, Consumer: "acme", Event: "order.created"}},
		storeErr: errFake,
	}
	ins := &recordingInserter{}
	w := &BatchJobWorker{eventRepo: events, jobInserter: ins, logger: slog.Default()}

	if processed, failed := w.processEventRepush(context.Background(), uuid.New(), uuid.New(), []string{id.String()}); processed != 0 || failed != 1 {
		t.Fatalf("processed=%d failed=%d, want the item failed", processed, failed)
	}
	if len(ins.jobs) != 0 {
		t.Fatalf("queued %d jobs for an event that was not stored", len(ins.jobs))
	}
}

// A batch cancelled while running stops at the next progress check.
func TestProcessEventRepushStopsWhenCancelled(t *testing.T) {
	events := &scriptedEventRepo{events: map[uuid.UUID]*store.EventRecord{}}
	ids := make([]string, progressUpdateInterval+5)
	for i := range ids {
		id := uuid.New()
		ids[i] = id.String()
		events.events[id] = &store.EventRecord{ID: id, Consumer: "acme", Event: "order.created"}
	}
	batches := &fakeBatchRepo{batch: &store.BatchJob{Status: store.BatchStatusCancelled}}
	w := &BatchJobWorker{batchRepo: batches, eventRepo: events, jobInserter: &recordingInserter{}, logger: slog.Default()}

	processed, _ := w.processEventRepush(context.Background(), uuid.New(), uuid.New(), ids)

	if processed != progressUpdateInterval {
		t.Fatalf("processed %d, want to stop after %d at the first cancellation check", processed, progressUpdateInterval)
	}
}

// scriptedDeliveryRepo serves deliveries and records resets.
type scriptedDeliveryRepo struct {
	store.DeliveryRepository
	deliveries map[uuid.UUID]*store.WebhookDelivery
	resetErr   error
	reset      []uuid.UUID
}

func (r *scriptedDeliveryRepo) GetDeliveryByID(_ context.Context, _, id uuid.UUID, _ string) (*store.WebhookDelivery, error) {
	if d, ok := r.deliveries[id]; ok {
		return d, nil
	}
	return nil, errFake
}

// ResetDeliveriesForRetry resets the deliveries not in flight, as the store
// does, and keeps the reset only when enqueue succeeds (the transaction).
func (r *scriptedDeliveryRepo) ResetDeliveriesForRetry(ctx context.Context, _ uuid.UUID, ids []uuid.UUID, enqueue store.RetryEnqueuer) ([]uuid.UUID, error) {
	if r.resetErr != nil {
		return nil, r.resetErr
	}
	var reset []uuid.UUID
	for _, id := range ids {
		switch r.deliveries[id].Status {
		case store.StatusPending, store.StatusSending, store.StatusRetrying:
		default:
			reset = append(reset, id)
		}
	}
	if len(reset) == 0 {
		return nil, nil
	}
	if err := enqueue(ctx, nil, reset); err != nil {
		return nil, err
	}
	r.reset = append(r.reset, reset...)
	return reset, nil
}

// scriptedWebhookRepo serves the webhooks it holds.
type scriptedWebhookRepo struct {
	store.WebhookRepository
	webhooks map[uuid.UUID]*store.WebhookRegistration
}

func (r *scriptedWebhookRepo) GetWebhookByID(_ context.Context, _, id uuid.UUID, _ string) (*store.WebhookRegistration, error) {
	if w, ok := r.webhooks[id]; ok {
		return w, nil
	}
	return nil, errFake
}

func TestProcessDeliveryRetry(t *testing.T) {
	webhookID, subID := uuid.New(), uuid.New()
	ok, orphan, inFlight := uuid.New(), uuid.New(), uuid.New()
	deliveries := &scriptedDeliveryRepo{deliveries: map[uuid.UUID]*store.WebhookDelivery{
		ok:       {ID: ok, WebhookID: webhookID, SubscriptionID: &subID, EventID: uuid.New(), Status: store.StatusFailed},
		orphan:   {ID: orphan, WebhookID: uuid.New(), EventID: uuid.New()},
		inFlight: {ID: inFlight, WebhookID: webhookID, EventID: uuid.New(), Status: store.StatusRetrying},
	}}
	webhooks := &scriptedWebhookRepo{webhooks: map[uuid.UUID]*store.WebhookRegistration{
		webhookID: {ID: webhookID, Consumer: "acme", MaxRetries: 2, RetryBackoffSeconds: 5},
	}}
	ins := &recordingInserter{}
	w := &BatchJobWorker{deliveryRepo: deliveries, webhookRepo: webhooks, jobInserter: ins, logger: slog.Default()}

	processed, failed := w.processDeliveryRetry(context.Background(), uuid.New(), uuid.New(), "acme",
		[]string{ok.String(), "bad", uuid.NewString(), orphan.String(), inFlight.String()})

	if processed != 1 || failed != 4 {
		t.Fatalf("processed=%d failed=%d, want 1 retried and 4 failed (bad id, missing delivery, missing webhook, in flight)", processed, failed)
	}
	if len(ins.jobs) != 1 {
		t.Fatalf("queued %d jobs, want 1 (none for the delivery in flight)", len(ins.jobs))
	}
	if len(deliveries.reset) != 1 || deliveries.reset[0] != ok {
		t.Fatalf("reset %v, want only %v (never reset before the webhook is known)", deliveries.reset, ok)
	}
	args, isWebhook := ins.jobs[0].(*WebhookArgs)
	if !isWebhook || args.DeliveryID != ok.String() || args.SubscriptionID != subID.String() ||
		args.MaxAttempts != 3 || args.RetryBackoffSeconds != 5 || !args.ExpiresAt.Equal(store.NoExpiryTime) {
		t.Fatalf("queued %#v, want a never-expiring retry with the webhook's attempts and backoff", ins.jobs[0])
	}
}

func TestProcessDeliveryRetryWriteFailures(t *testing.T) {
	webhookID := uuid.New()
	id := uuid.New()
	newWorker := func(resetErr error, failInsert bool) *BatchJobWorker {
		return &BatchJobWorker{
			deliveryRepo: &scriptedDeliveryRepo{
				deliveries: map[uuid.UUID]*store.WebhookDelivery{id: {ID: id, WebhookID: webhookID, EventID: uuid.New()}},
				resetErr:   resetErr,
			},
			webhookRepo: &scriptedWebhookRepo{webhooks: map[uuid.UUID]*store.WebhookRegistration{webhookID: {ID: webhookID}}},
			jobInserter: &recordingInserter{fail: failInsert},
			logger:      slog.Default(),
		}
	}
	for name, w := range map[string]*BatchJobWorker{
		"reset fails":   newWorker(errFake, false),
		"enqueue fails": newWorker(nil, true),
	} {
		if processed, failed := w.processDeliveryRetry(context.Background(), uuid.New(), uuid.New(), "", []string{id.String()}); processed != 0 || failed != 1 {
			t.Errorf("%s: processed=%d failed=%d, want the item failed", name, processed, failed)
		}
		// The reset and the job commit together: a failed enqueue leaves
		// the delivery as it was.
		if reset := w.deliveryRepo.(*scriptedDeliveryRepo).reset; len(reset) != 0 {
			t.Errorf("%s: reset %v without a queued job", name, reset)
		}
	}
}

func TestRetentionWorker(t *testing.T) {
	repo := &scriptedEventRepo{purgeCount: 3}
	if err := NewRetentionWorker(repo, 0).Work(context.Background(), nil); err != nil || !repo.cutoff.IsZero() {
		t.Fatalf("retention 0: err=%v cutoff=%v, want no purge", err, repo.cutoff)
	}

	if err := NewRetentionWorker(repo, 7).Work(context.Background(), nil); err != nil {
		t.Fatalf("Work: %v", err)
	}
	if d := time.Since(repo.cutoff); d < 7*24*time.Hour-time.Minute || d > 7*24*time.Hour+time.Minute {
		t.Fatalf("cutoff %v ago, want 7 days", d)
	}

	repo.purgeErr = errFake
	if err := NewRetentionWorker(repo, 7).Work(context.Background(), nil); !errors.Is(err, errFake) {
		t.Fatalf("Work error = %v, want the purge error so River retries", err)
	}

	opts := RetentionArgs{}.InsertOpts()
	if (RetentionArgs{}).Kind() != "retention_cleanup" || opts.UniqueOpts.ByPeriod != retentionInterval {
		t.Fatalf("retention job kind/opts = %q/%+v, want deduped per interval", RetentionArgs{}.Kind(), opts.UniqueOpts)
	}
}

type fakePurger struct {
	n         int64
	err       error
	retention time.Duration
}

func (f *fakePurger) PurgeTokens(_ context.Context, retention time.Duration) (int64, error) {
	f.retention = retention
	return f.n, f.err
}

func TestTokenPurgeWorker(t *testing.T) {
	p := &fakePurger{n: 2}
	if err := NewTokenPurgeWorker(p, 30*24*time.Hour).Work(context.Background(), nil); err != nil || p.retention != 30*24*time.Hour {
		t.Fatalf("Work: err=%v retention=%v", err, p.retention)
	}
	p.err = errFake
	if err := NewTokenPurgeWorker(p, time.Hour).Work(context.Background(), nil); !errors.Is(err, errFake) {
		t.Fatalf("Work error = %v, want the purge error", err)
	}
	opts := TokenPurgeArgs{}.InsertOpts()
	if (TokenPurgeArgs{}).Kind() != "access_token_purge" || opts.UniqueOpts.ByPeriod != tokenPurgeInterval {
		t.Fatalf("purge job kind/opts = %q/%+v, want deduped per interval", TokenPurgeArgs{}.Kind(), opts.UniqueOpts)
	}
}
