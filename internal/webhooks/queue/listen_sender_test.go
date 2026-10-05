package queue

import (
	"context"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	sparrowerrors "github.com/sarathsp06/sparrow/pkg/errors"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// fakeListenRepo keeps one session and its attempts in memory.
type fakeListenRepo struct {
	store.ListenRepository
	mu         sync.Mutex
	session    *store.ListenSession
	deliveries map[uuid.UUID]*store.ListenDelivery
	// onQueue runs after an attempt is queued, standing in for the CLI.
	onQueue func(*fakeListenRepo, *store.ListenDelivery)
}

func (f *fakeListenRepo) GetListenSession(_ context.Context, _, id uuid.UUID) (*store.ListenSession, error) {
	if f.session == nil || f.session.WebhookID != id {
		return nil, storage.ErrNotFound
	}
	return f.session, nil
}

func (f *fakeListenRepo) QueueListenDelivery(_ context.Context, d *store.ListenDelivery) error {
	f.mu.Lock()
	f.deliveries[d.DeliveryID] = d
	f.mu.Unlock()
	if f.onQueue != nil {
		go f.onQueue(f, d)
	}
	return nil
}

func (f *fakeListenRepo) respond(id uuid.UUID, status int, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now()
	d := f.deliveries[id]
	d.ResponseStatus, d.ResponseBody, d.RespondedAt = &status, []byte(body), &now
}

func (f *fakeListenRepo) GetListenDelivery(_ context.Context, id uuid.UUID) (*store.ListenDelivery, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	d, ok := f.deliveries[id]
	if !ok {
		return nil, storage.ErrNotFound
	}
	cp := *d
	return &cp, nil
}

func (f *fakeListenRepo) DeleteListenDelivery(_ context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.deliveries, id)
	return nil
}

func newListenFixture(lastSeen time.Time) (*fakeListenRepo, *client.DeliveryRequest) {
	webhookID := uuid.New()
	repo := &fakeListenRepo{
		session:    &store.ListenSession{WebhookID: webhookID, LastSeenAt: lastSeen, ExpiresAt: time.Now().Add(time.Hour)},
		deliveries: map[uuid.UUID]*store.ListenDelivery{},
	}
	req := &client.DeliveryRequest{
		WebhookID:  webhookID,
		DeliveryID: uuid.NewString(),
		URL:        client.ListenURL(webhookID),
		Method:     "POST",
		Payload:    []byte(`{"n":1}`),
		Secret:     "whsec_MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3",
		Timeout:    2 * time.Second,
	}
	return repo, req
}

func TestListenSenderReturnsTheCLIResponse(t *testing.T) {
	repo, req := newListenFixture(time.Now())
	var queued *store.ListenDelivery
	repo.onQueue = func(f *fakeListenRepo, d *store.ListenDelivery) {
		queued = d
		f.respond(d.DeliveryID, 503, "local app down")
	}
	s := newListenSender(repo)
	s.poll = 5 * time.Millisecond

	resp, _, err := s.Send(context.Background(), uuid.Nil, req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 503 || string(body) != "local app down" || resp.Status != "503 Service Unavailable" {
		t.Fatalf("got %d %q %q; want the CLI's 503 response", resp.StatusCode, resp.Status, body)
	}
	if queued.Headers["Webhook-Signature"] == "" || string(queued.Body) != `{"n":1}` {
		t.Fatalf("queued request must be signed and carry the payload: %+v", queued)
	}
	if len(repo.deliveries) != 0 {
		t.Fatal("the attempt must be removed once answered")
	}
}

func TestListenSenderOfflineFailsFastAsConnectionRefused(t *testing.T) {
	repo, req := newListenFixture(time.Now().Add(-2 * ListenOfflineAfter))
	_, _, err := newListenSender(repo).Send(context.Background(), uuid.Nil, req)
	if err == nil {
		t.Fatal("want an error for a session the CLI stopped polling")
	}
	if cat := sparrowerrors.ClassifyError(err); cat != sparrowerrors.CategoryConnectionRefused {
		t.Fatalf("category %q, want connection_refused (retryable)", cat)
	}
	if len(repo.deliveries) != 0 {
		t.Fatal("nothing must be queued for an offline session")
	}

	repo.session = nil
	if _, _, err := newListenSender(repo).Send(context.Background(), uuid.Nil, req); err != errListenOffline {
		t.Fatalf("deleted session: err %v, want errListenOffline", err)
	}
}

func TestListenSenderTimesOutWhenTheCLINeverAnswers(t *testing.T) {
	repo, req := newListenFixture(time.Now())
	req.Timeout = 50 * time.Millisecond
	s := newListenSender(repo)
	s.poll = 5 * time.Millisecond

	_, _, err := s.Send(context.Background(), uuid.Nil, req)
	if err == nil {
		t.Fatal("want a timeout")
	}
	if cat := sparrowerrors.ClassifyError(err); cat != sparrowerrors.CategoryTimeout {
		t.Fatalf("category %q, want timeout (retryable)", cat)
	}
	if len(repo.deliveries) != 0 {
		t.Fatal("a timed-out attempt must be removed so a late answer finds nothing")
	}
}
