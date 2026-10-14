package queue

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

func TestFanOutDeliveryIDIsDeterministic(t *testing.T) {
	event, sub := uuid.New(), uuid.New()
	if a, b := FanOutDeliveryID(event, sub), FanOutDeliveryID(event, sub); a != b {
		t.Fatal("same event and subscription gave different delivery IDs")
	}
	if FanOutDeliveryID(event, sub) == FanOutDeliveryID(event, uuid.New()) ||
		FanOutDeliveryID(event, sub) == FanOutDeliveryID(uuid.New(), sub) {
		t.Fatal("different pairs gave the same delivery ID")
	}
}

type fanOutEventRepo struct{ store.EventRepository }

func (fanOutEventRepo) GetEventByID(_ context.Context, _, id uuid.UUID) (*store.EventRecord, error) {
	return &store.EventRecord{ID: id}, nil
}

type fanOutSubRepo struct {
	store.SubscriptionRepository
	subs []*store.SubscriptionWithWebhook
}

func (f fanOutSubRepo) GetSubscriptionsWithWebhooksByEvent(context.Context, uuid.UUID, string, string, map[string]string) ([]*store.SubscriptionWithWebhook, error) {
	return f.subs, nil
}

// fanOutDeliveryRepo keeps the delivery IDs that "committed"; enqueue errors
// roll the whole call back, like the transaction does.
type fanOutDeliveryRepo struct {
	store.DeliveryRepository
	committed map[uuid.UUID]bool
}

func (f *fanOutDeliveryRepo) CreateFanOutDeliveries(ctx context.Context, _ uuid.UUID, deliveries []*store.WebhookDelivery, enqueue store.FanOutEnqueuer) ([]*store.WebhookDelivery, error) {
	var inserted []*store.WebhookDelivery
	for _, d := range deliveries {
		if !f.committed[d.ID] {
			inserted = append(inserted, d)
		}
	}
	if len(inserted) > 0 {
		if err := enqueue(ctx, nil, inserted); err != nil {
			return nil, err
		}
	}
	for _, d := range inserted {
		f.committed[d.ID] = true
	}
	return inserted, nil
}

type recordingTxInserter struct {
	noopJobInserter
	jobs []river.JobArgs
	err  error
}

func (r *recordingTxInserter) BatchInsertTx(_ context.Context, _ *sql.Tx, args []river.JobArgs) error {
	if r.err != nil {
		return r.err
	}
	r.jobs = append(r.jobs, args...)
	return nil
}

func (r *recordingTxInserter) BatchInsert(context.Context, []river.JobArgs) ([]*rivertype.JobInsertResult, error) {
	return nil, errors.New("fan-out must insert jobs in the delivery transaction")
}

// A fan-out River runs again creates no second delivery and queues no second
// job, and a failed job insert leaves nothing behind for the retry to skip.
func TestEventWorkerFanOutIsIdempotent(t *testing.T) {
	active := &store.WebhookRegistration{ID: uuid.New(), Active: true}
	paused := &store.WebhookRegistration{ID: uuid.New(), Active: false}
	subs := []*store.SubscriptionWithWebhook{
		{Subscription: &store.EventSubscription{ID: uuid.New()}, Webhook: active},
		{Subscription: &store.EventSubscription{ID: uuid.New()}, Webhook: paused},
	}
	deliveries := &fanOutDeliveryRepo{committed: map[uuid.UUID]bool{}}
	jobs := &recordingTxInserter{err: errors.New("insert failed")}
	w := &EventProcessingWorker{
		logger: slog.Default(), eventRepo: fanOutEventRepo{}, subscriptionRepo: fanOutSubRepo{subs: subs},
		deliveryRepo: deliveries, jobInserter: jobs,
	}
	eventID := uuid.New()
	job := &river.Job[EventArgs]{
		JobRow: &rivertype.JobRow{Metadata: []byte(`{}`)},
		Args:   EventArgs{TenantID: uuid.NewString(), EventID: eventID.String(), Consumer: "acme", Event: "order.created"},
	}

	if err := w.Work(context.Background(), job); err == nil {
		t.Fatal("a failed job insert must fail the fan-out so River retries it")
	}
	if len(deliveries.committed) != 0 {
		t.Fatalf("%d deliveries committed although their jobs were not", len(deliveries.committed))
	}

	jobs.err = nil
	for range 2 {
		if err := w.Work(context.Background(), job); err != nil {
			t.Fatalf("Work: %v", err)
		}
	}
	if len(deliveries.committed) != 2 {
		t.Fatalf("committed %d deliveries, want 2 (one held for the paused webhook)", len(deliveries.committed))
	}
	if len(jobs.jobs) != 1 {
		t.Fatalf("queued %d jobs over two runs, want 1", len(jobs.jobs))
	}
	want := FanOutDeliveryID(eventID, subs[0].Subscription.ID).String()
	if got := jobs.jobs[0].(*WebhookArgs).DeliveryID; got != want {
		t.Fatalf("job delivery ID = %s, want %s", got, want)
	}
}
