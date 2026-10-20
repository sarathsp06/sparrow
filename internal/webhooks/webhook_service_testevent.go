package webhooks

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/template"
)

// TestEventInterval is how often one webhook can be sent a test event. The
// limit is per server process.
const TestEventInterval = 2 * time.Second

// TestEventSender sends one-off test deliveries.
type TestEventSender interface {
	SendTestEvent(ctx context.Context, consumer, webhookID, subscriptionID, eventName string, payload map[string]any) (*TestEventResult, error)
}

// TestEventResult is the outcome of a test delivery.
type TestEventResult struct {
	// StatusCode is the receiver's HTTP status, 0 when nothing was received.
	StatusCode int
	Latency    time.Duration
	// ErrorClass is the delivery error category (e.g. client_error,
	// timeout, template_error), empty on a 2xx.
	ErrorClass string
}

// WithDeliveryClient sets the client test events are sent with: the same
// one deliveries use, so they share its SSRF policy and timeouts.
func WithDeliveryClient(c *client.WebhookClient) WebhookServiceOption {
	return func(s *WebhookService) {
		s.deliveryClient = c
	}
}

// testEventLimiter allows one test event per webhook per interval.
type testEventLimiter struct {
	mu   sync.Mutex
	last map[uuid.UUID]time.Time
}

func (l *testEventLimiter) allow(id uuid.UUID, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.last == nil {
		l.last = map[uuid.UUID]time.Time{}
	}
	if prev, ok := l.last[id]; ok && now.Sub(prev) < TestEventInterval {
		return false
	}
	l.last[id] = now
	// Drop entries that can no longer limit anything, so the map stays
	// as small as the set of recently tested webhooks.
	for k, t := range l.last {
		if now.Sub(t) >= TestEventInterval {
			delete(l.last, k)
		}
	}
	return true
}

// SendTestEvent sends one signed delivery of payload, or of the event type's
// sample when payload is nil, to the webhook, synchronously, and reports what the receiver
// answered. It goes through the same request builder, signing, SSRF policy
// and timeout as a real delivery and carries X-Sparrow-Test: true. It records
// no delivery and never touches the webhook's health or auto-disable state:
// a test send is the sender's action, not the receiver's track record.
//
// The event is the subscription's event type; a catch-all ("*")
// subscription needs eventName. The event type need not be registered when
// payload is given.
func (s *WebhookService) SendTestEvent(ctx context.Context, consumer, webhookID, subscriptionID, eventName string, payload map[string]any) (*TestEventResult, error) {
	if s.deliveryClient == nil {
		return nil, svcerrors.Error(svcerrors.Unavailable, "test events are not available on this server")
	}
	id, err := parseUUID(webhookID, "webhook ID")
	if err != nil {
		return nil, err
	}
	webhook, err := s.webhookRepo.GetWebhookByID(ctx, tenant.DefaultTenantID, id, consumer)
	if err != nil {
		return nil, err
	}
	sub, err := s.getSubscriptionInConsumer(ctx, subscriptionID, consumer)
	if err != nil {
		return nil, err
	}
	if sub.WebhookID != webhook.ID {
		return nil, svcerrors.Error(svcerrors.NotFound, "subscription not found on this webhook")
	}
	name, err := testEventName(sub.EventName, eventName)
	if err != nil {
		return nil, err
	}
	event, err := s.webhookRepo.GetEventByName(ctx, tenant.DefaultTenantID, name)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve event type: %w", err)
	}
	if event == nil && payload == nil {
		return nil, svcerrors.Errorf(svcerrors.InvalidArgument, "event type %q is not registered, so it has no sample: send a payload", name)
	}
	if event == nil {
		event = &store.EventRegistration{Name: name}
	}
	if !s.testEvents.allow(webhook.ID, time.Now()) {
		return nil, svcerrors.Errorf(svcerrors.ResourceExhausted, "one test event per webhook every %s; try again shortly", TestEventInterval)
	}

	record := &store.EventRecord{
		ID:           uuid.New(),
		Consumer:     consumer,
		Event:        event.Name,
		Payload:      event.SamplePayload,
		EventVersion: event.Version,
	}
	if payload != nil {
		record.Payload = payload
	}
	if record.Payload == nil {
		record.Payload = store.JSONMap{}
	}
	body, renderErr := testEventBody(webhook, sub, record)
	if renderErr != nil {
		return &TestEventResult{ErrorClass: string(svcerrors.CategoryTemplateError)}, nil
	}

	dr, err := client.PrepareDeliveryRequest(webhook, sub, record, "test_"+record.ID.String(), body, s.crypto)
	if err != nil {
		return nil, fmt.Errorf("prepare test delivery: %w", err)
	}
	dr.Test = true
	resp, latency, err := s.deliveryClient.Send(ctx, dr)
	result := &TestEventResult{Latency: latency}
	if err != nil {
		result.ErrorClass = string(svcerrors.ClassifyError(err))
		return result, nil
	}
	_ = resp.Body.Close()
	result.StatusCode = resp.StatusCode
	if category := svcerrors.ClassifyHTTPStatus(resp.StatusCode); category != svcerrors.CategorySuccess {
		result.ErrorClass = string(category)
	}
	return result, nil
}

// testEventName is the event a test send is for: the subscription's, or
// for a catch-all subscription the requested one.
func testEventName(subscribed, requested string) (string, error) {
	switch {
	case subscribed != "*" && requested != "" && requested != subscribed:
		return "", svcerrors.Errorf(svcerrors.InvalidArgument, "the subscription is for %q, not %q", subscribed, requested)
	case subscribed != "*":
		return subscribed, nil
	case requested == "" || requested == "*":
		return "", svcerrors.Error(svcerrors.InvalidArgument, "event_name is required for a catch-all subscription")
	}
	return requested, nil
}

// testEventBody renders the body the subscription would send for record, as
// the delivery worker does: the transform, or the envelope without one or
// as the configured fallback. (A webhook that requires a transform cannot
// have a subscription without one: saving it is refused.)
func testEventBody(webhook *store.WebhookRegistration, sub *store.EventSubscription, record *store.EventRecord) ([]byte, error) {
	envelope := func() ([]byte, error) {
		return client.BuildEnvelopePayload(record.ID.String(), record.Event, 1, record.Payload)
	}
	if !sub.HasTransform() {
		return envelope()
	}
	ctx := template.NewWebhookContext(record.ID.String(), record.Event, time.Now().UTC().Format(time.RFC3339), 1, record.Payload)
	body, err := template.New().ExecuteWith(sub.TransformTemplate, ctx, template.ExecOptions{StrictMissingKeys: sub.StrictTemplate()})
	if err == nil {
		return body, nil
	}
	if sub.FallbackOnTransformError() && !webhook.RequiresTransform {
		return envelope()
	}
	return nil, fmt.Errorf("template transformation failed: %w", err)
}
