package queue

import (
	"context"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// fakeAlertConfigRepo records ResolveAlertRecipients calls and returns a
// fixed recipient list; a test fails it if called when a guard should have
// short-circuited first.
type fakeAlertConfigRepo struct {
	store.AlertConfigRepository
	recipients []string
	calls      int
}

func (f *fakeAlertConfigRepo) ResolveAlertRecipients(ctx context.Context, tenantID, webhookID uuid.UUID, consumer, eventType string) ([]string, error) {
	f.calls++
	return f.recipients, nil
}

// fakeSystemEventRepo fails the test if a system event is pushed when a
// guard should have skipped emission (e.g. zero recipients).
type fakeSystemEventRepo struct {
	systemEventRepo
	stored []string // event names pushed via StoreEvent
}

func (f *fakeSystemEventRepo) GetEventByName(ctx context.Context, tenantID uuid.UUID, name string) (*store.EventRegistration, error) {
	return &store.EventRegistration{Name: name, Active: true}, nil
}

func (f *fakeSystemEventRepo) StoreEvent(ctx context.Context, tenantID uuid.UUID, event *store.EventRecord) error {
	f.stored = append(f.stored, event.Event)
	return nil
}

func newTestWorker(alertRepo *fakeAlertConfigRepo, eventRepo *fakeSystemEventRepo) *WebhookWorker {
	return &WebhookWorker{
		alertConfigRepo: alertRepo,
		eventRepo:       eventRepo,
		jobInserter:     noopJobInserter{},
	}
}

// noopJobInserter satisfies JobInserter without asserting on the job — these
// tests only care whether StoreEvent (and thus emission) happened.
type noopJobInserter struct{}

func (noopJobInserter) Insert(ctx context.Context, args river.JobArgs) (*rivertype.JobInsertResult, error) {
	return &rivertype.JobInsertResult{}, nil
}

func (noopJobInserter) BatchInsert(ctx context.Context, args []river.JobArgs) ([]*rivertype.JobInsertResult, error) {
	return nil, nil
}

func TestEmitHealthChangedEvent_SkipsNoOpTransition(t *testing.T) {
	alertRepo := &fakeAlertConfigRepo{recipients: []string{"a@example.com"}}
	eventRepo := &fakeSystemEventRepo{}
	w := newTestWorker(alertRepo, eventRepo)

	w.emitHealthChangedEvent(context.Background(), slog.Default(), uuid.New(), "acme", uuid.New(), "https://x", "healthy", "healthy")

	if alertRepo.calls != 0 {
		t.Errorf("expected no recipient lookup for a no-op transition, got %d calls", alertRepo.calls)
	}
	if len(eventRepo.stored) != 0 {
		t.Errorf("expected no event stored for a no-op transition, got %v", eventRepo.stored)
	}
}

func TestEmitHealthChangedEvent_SkipsFirstEverOutcome(t *testing.T) {
	alertRepo := &fakeAlertConfigRepo{recipients: []string{"a@example.com"}}
	eventRepo := &fakeSystemEventRepo{}
	w := newTestWorker(alertRepo, eventRepo)

	w.emitHealthChangedEvent(context.Background(), slog.Default(), uuid.New(), "acme", uuid.New(), "https://x", string(store.HealthUnknown), string(store.HealthHealthy))

	if alertRepo.calls != 0 {
		t.Errorf("expected no recipient lookup for unknown->healthy, got %d calls", alertRepo.calls)
	}
	if len(eventRepo.stored) != 0 {
		t.Errorf("expected no event stored for unknown->healthy, got %v", eventRepo.stored)
	}
}

func TestEmitHealthChangedEvent_SkipsSparrowConsumer(t *testing.T) {
	alertRepo := &fakeAlertConfigRepo{recipients: []string{"a@example.com"}}
	eventRepo := &fakeSystemEventRepo{}
	w := newTestWorker(alertRepo, eventRepo)

	w.emitHealthChangedEvent(context.Background(), slog.Default(), uuid.New(), systemEventConsumer, uuid.New(), "https://x", string(store.HealthHealthy), string(store.HealthDegraded))

	if alertRepo.calls != 0 {
		t.Errorf("expected no recipient lookup for _sparrow's own webhooks (feedback-loop guard), got %d calls", alertRepo.calls)
	}
}

func TestEmitHealthChangedEvent_SkipsZeroRecipients(t *testing.T) {
	alertRepo := &fakeAlertConfigRepo{recipients: nil}
	eventRepo := &fakeSystemEventRepo{}
	w := newTestWorker(alertRepo, eventRepo)

	w.emitHealthChangedEvent(context.Background(), slog.Default(), uuid.New(), "acme", uuid.New(), "https://x", string(store.HealthHealthy), string(store.HealthDegraded))

	if alertRepo.calls != 1 {
		t.Errorf("expected recipient lookup to run once, got %d calls", alertRepo.calls)
	}
	if len(eventRepo.stored) != 0 {
		t.Errorf("expected no event stored when nobody opted in, got %v", eventRepo.stored)
	}
}

func TestEmitHealthChangedEvent_EmitsOnRealTransition(t *testing.T) {
	alertRepo := &fakeAlertConfigRepo{recipients: []string{"a@example.com"}}
	eventRepo := &fakeSystemEventRepo{}
	w := newTestWorker(alertRepo, eventRepo)

	w.emitHealthChangedEvent(context.Background(), slog.Default(), uuid.New(), "acme", uuid.New(), "https://x", string(store.HealthHealthy), string(store.HealthDegraded))

	if len(eventRepo.stored) != 1 || eventRepo.stored[0] != systemEventHealthChanged {
		t.Errorf("expected one %s event stored, got %v", systemEventHealthChanged, eventRepo.stored)
	}
}

func TestEmitDeliveryFailedEvent_SkipsSparrowConsumer(t *testing.T) {
	alertRepo := &fakeAlertConfigRepo{recipients: []string{"a@example.com"}}
	eventRepo := &fakeSystemEventRepo{}
	w := newTestWorker(alertRepo, eventRepo)

	w.emitDeliveryFailedEvent(context.Background(), slog.Default(), uuid.New(), systemEventConsumer, uuid.New(), uuid.New(), uuid.New(), "https://x", 3, "server_error", "boom")

	if alertRepo.calls != 0 {
		t.Errorf("expected no recipient lookup for _sparrow's own webhooks (feedback-loop guard), got %d calls", alertRepo.calls)
	}
}

func TestEmitDeliveryFailedEvent_SkipsZeroRecipients(t *testing.T) {
	alertRepo := &fakeAlertConfigRepo{recipients: nil}
	eventRepo := &fakeSystemEventRepo{}
	w := newTestWorker(alertRepo, eventRepo)

	w.emitDeliveryFailedEvent(context.Background(), slog.Default(), uuid.New(), "acme", uuid.New(), uuid.New(), uuid.New(), "https://x", 3, "server_error", "boom")

	if len(eventRepo.stored) != 0 {
		t.Errorf("expected no event stored when nobody opted in, got %v", eventRepo.stored)
	}
}

func TestEmitDeliveryFailedEvent_EmitsWhenRecipientsOptedIn(t *testing.T) {
	alertRepo := &fakeAlertConfigRepo{recipients: []string{"a@example.com"}}
	eventRepo := &fakeSystemEventRepo{}
	w := newTestWorker(alertRepo, eventRepo)

	w.emitDeliveryFailedEvent(context.Background(), slog.Default(), uuid.New(), "acme", uuid.New(), uuid.New(), uuid.New(), "https://x", 3, "server_error", "boom")

	if len(eventRepo.stored) != 1 || eventRepo.stored[0] != systemEventDeliveryFailed {
		t.Errorf("expected one %s event stored, got %v", systemEventDeliveryFailed, eventRepo.stored)
	}
}

func TestToAlertRecipients(t *testing.T) {
	got := toAlertRecipients([]string{"a@example.com", "b@example.com"})
	if len(got) != 2 || got[0]["email"] != "a@example.com" || got[1]["email"] != "b@example.com" {
		t.Errorf("unexpected recipients shape: %v", got)
	}
}
