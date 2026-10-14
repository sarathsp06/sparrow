package queue

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

type statusWebhookRepo struct {
	store.WebhookRepository
	webhook *store.WebhookRegistration
	err     error
}

func (f statusWebhookRepo) GetWebhookByID(context.Context, uuid.UUID, uuid.UUID, string) (*store.WebhookRegistration, error) {
	return f.webhook, f.err
}

type statusEventRepo struct{ systemEventRepo }

func (statusEventRepo) GetEventByID(_ context.Context, _, id uuid.UUID) (*store.EventRecord, error) {
	return &store.EventRecord{ID: id, Event: "order.created", Payload: map[string]any{}}, nil
}

// statusDeliveryRepo records the status writes and fails RecordDeliveryAttempt
// when recordErr is set.
type statusDeliveryRepo struct {
	store.DeliveryRepository
	statuses  []store.WebhookDeliveryStatus
	recordErr error
	recorded  []store.DeliveryAttempt
}

func (f *statusDeliveryRepo) UpdateDeliveryStatus(_ context.Context, _ uuid.UUID, status store.WebhookDeliveryStatus, _ int, _, _, _ string) error {
	f.statuses = append(f.statuses, status)
	return nil
}

func (f *statusDeliveryRepo) RecordDeliveryAttempt(_ context.Context, a store.DeliveryAttempt) error {
	if f.recordErr != nil {
		return f.recordErr
	}
	f.recorded = append(f.recorded, a)
	return nil
}

func newStatusWorker(t *testing.T, webhooks statusWebhookRepo, deliveries *statusDeliveryRepo) *WebhookWorker {
	t.Helper()
	cfg := client.DefaultConfig()
	cfg.AllowPrivateNetworks = true
	return NewWebhookWorker(webhooks, statusEventRepo{}, deliveries, nil, nil, nil, noopJobInserter{}, nil, cfg)
}

func deliveryJob(attempt, maxAttempts int) *river.Job[WebhookArgs] {
	return &river.Job[WebhookArgs]{
		JobRow: &rivertype.JobRow{Attempt: attempt, Metadata: []byte(`{}`)},
		Args: WebhookArgs{
			TenantID: uuid.NewString(), DeliveryID: uuid.NewString(), WebhookID: uuid.NewString(), EventID: uuid.NewString(),
			Consumer: "acme", MaxAttempts: maxAttempts, ExpiresAt: time.Now().Add(time.Hour),
		},
	}
}

// A transient lookup error is retried by River, so the delivery is
// "retrying" until the last attempt, never "failed" while it will run again.
func TestWebhookWorkerTransientErrorStatus(t *testing.T) {
	for _, tc := range []struct {
		attempt int
		want    store.WebhookDeliveryStatus
	}{
		{1, store.StatusRetrying},
		{3, store.StatusFailed},
	} {
		deliveries := &statusDeliveryRepo{}
		w := newStatusWorker(t, statusWebhookRepo{err: errors.New("connection reset")}, deliveries)
		if err := w.Work(context.Background(), deliveryJob(tc.attempt, 3)); err == nil {
			t.Fatal("a transient error must fail the job so River retries it")
		}
		if len(deliveries.statuses) != 1 || deliveries.statuses[0] != tc.want {
			t.Fatalf("attempt %d of 3: statuses %v, want [%s]", tc.attempt, deliveries.statuses, tc.want)
		}
	}
}

// A 2xx whose attempt cannot be recorded must fail the job: returning nil
// left the delivery pending forever although River considered it done.
func TestWebhookWorkerUnrecordedSuccessRetries(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer srv.Close()
	webhooks := statusWebhookRepo{webhook: &store.WebhookRegistration{
		ID: uuid.New(), URL: srv.URL, Active: true, RequestTimeoutSeconds: 5, ExpectedStatusCodes: []int64{200},
	}}

	deliveries := &statusDeliveryRepo{recordErr: errors.New("db down")}
	err := newStatusWorker(t, webhooks, deliveries).Work(context.Background(), deliveryJob(1, 3))
	if err == nil {
		t.Fatal("Work returned nil although the successful attempt was not recorded")
	}

	deliveries = &statusDeliveryRepo{}
	if err := newStatusWorker(t, webhooks, deliveries).Work(context.Background(), deliveryJob(1, 3)); err != nil {
		t.Fatalf("Work: %v", err)
	}
	if len(deliveries.recorded) != 1 || deliveries.recorded[0].Status != store.StatusSuccess {
		t.Fatalf("recorded %+v, want one success", deliveries.recorded)
	}
}
