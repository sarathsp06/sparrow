package webhooks

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/crypto"
	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/signature"
)

// readOnlyRepo answers the three reads a test send needs. Every other
// method is the nil embedded interface and panics, so a test send that
// wrote a delivery or touched health would fail the test.
type readOnlyRepo struct {
	store.RepositoryInterface
	webhook *store.WebhookRegistration
	sub     *store.EventSubscription
	event   *store.EventRegistration
}

func (r *readOnlyRepo) GetWebhookByID(_ context.Context, _ uuid.UUID, id uuid.UUID, consumer string) (*store.WebhookRegistration, error) {
	if id != r.webhook.ID || consumer != r.webhook.Consumer {
		return nil, svcerrors.Error(svcerrors.NotFound, "webhook not found")
	}
	return r.webhook, nil
}

func (r *readOnlyRepo) GetSubscription(_ context.Context, _ uuid.UUID, id uuid.UUID) (*store.EventSubscription, error) {
	if id != r.sub.ID {
		return nil, svcerrors.Error(svcerrors.NotFound, "subscription not found")
	}
	return r.sub, nil
}

func (r *readOnlyRepo) GetEventByName(context.Context, uuid.UUID, string) (*store.EventRegistration, error) {
	return r.event, nil
}

type received struct {
	header http.Header
	body   []byte
}

func testEventSetup(t *testing.T, status int) (*WebhookService, *readOnlyRepo, chan received) {
	t.Helper()
	got := make(chan received, 4)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got <- received{header: r.Header.Clone(), body: body}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(strings.Repeat("x", 2000)))
	}))
	t.Cleanup(srv.Close)

	cryptoSvc, err := crypto.NewService(make([]byte, 32))
	if err != nil {
		t.Fatal(err)
	}
	secret, err := cryptoSvc.EncryptString("whsec_dGVzdC1zZWNyZXQtZm9yLXRlc3QtZXZlbnRz")
	if err != nil {
		t.Fatal(err)
	}
	webhook := &store.WebhookRegistration{ID: uuid.New(), Consumer: "acme", URL: srv.URL + "/hook?token=x",
		WebhookSecret: secret, VerifySSL: true, RequestTimeoutSeconds: 5,
		Headers: map[string]string{"X-Sparrow-Test": "spoofed"}}
	repo := &readOnlyRepo{
		webhook: webhook,
		sub:     &store.EventSubscription{ID: uuid.New(), WebhookID: webhook.ID, Consumer: "acme", EventName: "order.paid"},
		event:   &store.EventRegistration{Name: "order.paid", Version: 3, SamplePayload: store.JSONMap{"id": "ord_1"}},
	}
	clientCfg := client.DefaultConfig()
	clientCfg.AllowPrivateNetworks = true
	svc := NewWebhookService(new(mockJobInserter), repo, cryptoSvc, WithDeliveryClient(client.NewWebhookClient(clientCfg)))
	return svc, repo, got
}

func TestSendTestEventSendsASignedTestDelivery(t *testing.T) {
	svc, repo, got := testEventSetup(t, http.StatusAccepted)

	res, err := svc.SendTestEvent(testContext(), "acme", repo.webhook.ID.String(), repo.sub.ID.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusAccepted || res.ErrorClass != "" {
		t.Errorf("result = %+v", res)
	}
	r := <-got
	for name, want := range map[string]string{
		"X-Sparrow-Test":            "true",
		"X-Sparrow-Event-Type":      "order.paid",
		"X-Sparrow-Event-Version":   "3",
		"X-Sparrow-Subscription-ID": repo.sub.ID.String(),
	} {
		if got := r.header.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if err := signature.VerifyHMAC(r.body, r.header, "whsec_dGVzdC1zZWNyZXQtZm9yLXRlc3QtZXZlbnRz"); err != nil {
		t.Errorf("test delivery is not signed like a real one: %v", err)
	}
	if !strings.Contains(string(r.body), `"payload":{"id":"ord_1"}`) {
		t.Errorf("body = %s, want the envelope of the sample", r.body)
	}

	// A second send within the interval is refused without sending.
	_, err = svc.SendTestEvent(testContext(), "acme", repo.webhook.ID.String(), repo.sub.ID.String(), nil)
	var svcErr *svcerrors.ServiceError
	if !errors.As(err, &svcErr) || svcErr.Status != svcerrors.ResourceExhausted {
		t.Fatalf("second send err = %v, want ResourceExhausted", err)
	}
	if len(got) != 0 {
		t.Error("rate-limited send reached the receiver")
	}
}

func TestSendTestEventSendsAGivenPayload(t *testing.T) {
	svc, repo, got := testEventSetup(t, http.StatusOK)
	repo.sub.TransformEnabled = true
	repo.sub.TransformTemplate = `order {{.payload.id}}`
	if _, err := svc.SendTestEvent(testContext(), "acme", repo.webhook.ID.String(), repo.sub.ID.String(), map[string]any{"id": "ord_edited"}); err != nil {
		t.Fatal(err)
	}
	if r := <-got; string(r.body) != "order ord_edited" {
		t.Errorf("body = %q, want the given payload through the transform", r.body)
	}
}

func TestSendTestEventReportsReceiverErrors(t *testing.T) {
	svc, repo, _ := testEventSetup(t, http.StatusBadRequest)
	res, err := svc.SendTestEvent(testContext(), "acme", repo.webhook.ID.String(), repo.sub.ID.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusBadRequest || res.ErrorClass != string(svcerrors.CategoryClientError) {
		t.Errorf("result = %+v", res)
	}
}

func TestSendTestEventRendersTheTransform(t *testing.T) {
	svc, repo, got := testEventSetup(t, http.StatusOK)
	repo.sub.TransformEnabled = true
	repo.sub.TransformTemplate = `order {{.payload.id}}`
	if _, err := svc.SendTestEvent(testContext(), "acme", repo.webhook.ID.String(), repo.sub.ID.String(), nil); err != nil {
		t.Fatal(err)
	}
	if r := <-got; string(r.body) != "order ord_1" {
		t.Errorf("body = %q", r.body)
	}

	// A template that fails to render is reported, and nothing is sent.
	svc.testEvents = testEventLimiter{}
	repo.sub.TransformTemplate = `{{.payload.missing.deeper}}`
	repo.sub.TemplateMissingKey = store.TemplateMissingKeyError
	res, err := svc.SendTestEvent(testContext(), "acme", repo.webhook.ID.String(), repo.sub.ID.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.ErrorClass != string(svcerrors.CategoryTemplateError) || len(got) != 0 {
		t.Errorf("result = %+v, sent %d", res, len(got))
	}

	// With on_transform_error=fallback it sends the envelope instead.
	svc.testEvents = testEventLimiter{}
	repo.sub.OnTransformError = store.OnTransformErrorFallback
	if res, err := svc.SendTestEvent(testContext(), "acme", repo.webhook.ID.String(), repo.sub.ID.String(), nil); err != nil || res.StatusCode != http.StatusOK {
		t.Fatalf("fallback: %+v %v", res, err)
	}
	if r := <-got; !strings.HasPrefix(string(r.body), `{"version":"1"`) {
		t.Errorf("fallback body = %s", r.body)
	}
}

func TestSendTestEventRefusals(t *testing.T) {
	svc, repo, got := testEventSetup(t, http.StatusOK)
	other := uuid.NewString()
	cases := map[string]struct {
		consumer, webhook, sub string
	}{
		"other consumer":       {"globex", repo.webhook.ID.String(), repo.sub.ID.String()},
		"unknown webhook":      {"acme", other, repo.sub.ID.String()},
		"invalid webhook id":   {"acme", "nope", repo.sub.ID.String()},
		"unknown subscription": {"acme", repo.webhook.ID.String(), other},
	}
	for name, c := range cases {
		if _, err := svc.SendTestEvent(testContext(), c.consumer, c.webhook, c.sub, nil); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
	// A subscription of another webhook in the same consumer.
	repo.sub.WebhookID = uuid.New()
	if _, err := svc.SendTestEvent(testContext(), "acme", repo.webhook.ID.String(), repo.sub.ID.String(), nil); err == nil {
		t.Error("subscription of another webhook: no error")
	}
	repo.sub.WebhookID = repo.webhook.ID
	if len(got) != 0 {
		t.Error("a refused test event reached the receiver")
	}

	// Without a delivery client the feature is off.
	off := NewWebhookService(new(mockJobInserter), repo, nil)
	if _, err := off.SendTestEvent(testContext(), "acme", repo.webhook.ID.String(), repo.sub.ID.String(), nil); err == nil {
		t.Error("no client: no error")
	}
}

func TestSendTestEventReportsTransportErrors(t *testing.T) {
	svc, repo, _ := testEventSetup(t, http.StatusOK)
	repo.webhook.URL = "http://127.0.0.1:1/unreachable"
	res, err := svc.SendTestEvent(testContext(), "acme", repo.webhook.ID.String(), repo.sub.ID.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 0 || res.ErrorClass == "" {
		t.Errorf("result = %+v", res)
	}
}

func TestTestEventLimiter(t *testing.T) {
	var l testEventLimiter
	a, b := uuid.New(), uuid.New()
	now := time.Now()
	if !l.allow(a, now) || l.allow(a, now.Add(time.Second)) || !l.allow(b, now) {
		t.Fatal("limiter does not limit per webhook")
	}
	if !l.allow(a, now.Add(TestEventInterval)) {
		t.Fatal("limiter does not allow after the interval")
	}
	if len(l.last) != 1 {
		t.Errorf("stale entries kept: %d", len(l.last))
	}
}
