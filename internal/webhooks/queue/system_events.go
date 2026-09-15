package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// systemEventConsumer is the internal consumer Sparrow's own self-generated
// events are scoped to. Tenants never receive these events directly; they
// opt an email address into them via webhook_alert_configs, and an operator
// wires an actual delivery channel (e.g. the sendgrid recipe) as a normal
// webhook + subscription under this consumer.
const systemEventConsumer = "_sparrow"

const (
	// systemEventHealthChanged fires on every webhook health transition
	// except unknown -> healthy (a webhook's very first delivery outcome).
	systemEventHealthChanged = "sparrow.webhook.health_changed"
	// systemEventDeliveryFailed fires once a delivery has exhausted every
	// retry and is permanently failed.
	systemEventDeliveryFailed = "sparrow.webhook.delivery_failed"
)

// systemEventRepo is the narrow event-repository surface WebhookWorker needs:
// reading event records for ordinary delivery, plus registering/storing its
// own system events. The concrete value passed in is always the full
// store.RepositoryInterface, which satisfies both.
type systemEventRepo interface {
	store.EventRepository
	store.EventTypeRepository
}

// toAlertRecipients converts resolved recipient emails into the
// payload.alert_recipients shape templates (e.g. satellites/recipes/sendgrid.yaml)
// range over to build one SendGrid personalization per recipient.
func toAlertRecipients(emails []string) []map[string]string {
	recipients := make([]map[string]string, len(emails))
	for i, email := range emails {
		recipients[i] = map[string]string{"email": email}
	}
	return recipients
}

// pushSystemEvent stores and enqueues one of Sparrow's self-generated system
// events. It mirrors WebhookService.PushEvent's auto-register + store +
// enqueue flow, but lives here (not as a shared helper) because the queue
// package cannot import webhooks (webhooks already imports queue for
// EventArgs), so WebhookWorker cannot call WebhookService.PushEvent directly.
// Failures are logged and swallowed: a missed self-generated alert must never
// fail or retry the webhook delivery job that triggered it.
func pushSystemEvent(ctx context.Context, log *slog.Logger, eventRepo systemEventRepo, jobInserter JobInserter, tenantID uuid.UUID, event string, payload map[string]any) {
	eventReg, err := eventRepo.GetEventByName(ctx, tenantID, event)
	if err != nil {
		log.ErrorContext(ctx, "Failed to lookup system event registration", "event", event, "error", err)
		return
	}
	if eventReg == nil {
		eventReg = &store.EventRegistration{
			Name:        event,
			Description: "Sparrow-generated system event.",
			Active:      true,
		}
		if err := eventRepo.RegisterEvent(ctx, tenantID, eventReg); err != nil {
			log.ErrorContext(ctx, "Failed to auto-register system event", "event", event, "error", err)
			return
		}
	}

	eventID := uuid.New()
	createdAt := time.Now()
	record := &store.EventRecord{
		ID:          eventID,
		Consumer:    systemEventConsumer,
		Event:       event,
		Payload:     payload,
		SchemaValid: true,
		CreatedAt:   createdAt,
	}
	if err := eventRepo.StoreEvent(ctx, tenantID, record); err != nil {
		log.ErrorContext(ctx, "Failed to store system event", "event", event, "error", err)
		return
	}
	if _, err := jobInserter.Insert(ctx, EventArgs{
		EventID:   eventID.String(),
		Consumer:  systemEventConsumer,
		Event:     event,
		CreatedAt: createdAt,
		TenantID:  tenantID.String(),
	}); err != nil {
		log.ErrorContext(ctx, "Failed to enqueue system event processing job", "event", event, "error", err)
	}
}

// emitHealthChangedEvent pushes systemEventHealthChanged when a delivery
// outcome moved webhookID's health from oldHealth to newHealth, skipping:
// no-op transitions, a webhook's first-ever outcome (unknown -> healthy),
// self-events for _sparrow's own webhooks (feedback-loop guard), and
// transitions nobody has opted an alert into.
func (w *WebhookWorker) emitHealthChangedEvent(ctx context.Context, log *slog.Logger, tenantID uuid.UUID, consumer string, webhookID uuid.UUID, url, oldHealth, newHealth string) {
	if consumer == systemEventConsumer || oldHealth == newHealth {
		return
	}
	if oldHealth == string(store.HealthUnknown) && newHealth == string(store.HealthHealthy) {
		return
	}
	recipients, err := w.alertConfigRepo.ResolveAlertRecipients(ctx, tenantID, webhookID, consumer, systemEventHealthChanged)
	if err != nil {
		log.ErrorContext(ctx, "Failed to resolve health_changed alert recipients", "error", err)
		return
	}
	if len(recipients) == 0 {
		return
	}
	pushSystemEvent(ctx, log, w.eventRepo, w.jobInserter, tenantID, systemEventHealthChanged, map[string]any{
		"webhook_id":       webhookID.String(),
		"consumer":         consumer,
		"url":              url,
		"old_health":       oldHealth,
		"new_health":       newHealth,
		"alert_recipients": toAlertRecipients(recipients),
	})
}

// emitDeliveryFailedEvent pushes systemEventDeliveryFailed once deliveryID has
// exhausted every retry, skipping self-events for _sparrow's own webhooks and
// deliveries nobody has opted an alert into.
func (w *WebhookWorker) emitDeliveryFailedEvent(ctx context.Context, log *slog.Logger, tenantID uuid.UUID, consumer string, webhookID, deliveryID, eventID uuid.UUID, url string, attempt int, errorCategory, errorMessage string) {
	if consumer == systemEventConsumer {
		return
	}
	recipients, err := w.alertConfigRepo.ResolveAlertRecipients(ctx, tenantID, webhookID, consumer, systemEventDeliveryFailed)
	if err != nil {
		log.ErrorContext(ctx, "Failed to resolve delivery_failed alert recipients", "error", err)
		return
	}
	if len(recipients) == 0 {
		return
	}
	pushSystemEvent(ctx, log, w.eventRepo, w.jobInserter, tenantID, systemEventDeliveryFailed, map[string]any{
		"webhook_id":       webhookID.String(),
		"consumer":         consumer,
		"url":              url,
		"delivery_id":      deliveryID.String(),
		"event_id":         eventID.String(),
		"attempt":          attempt,
		"error_category":   errorCategory,
		"error_message":    errorMessage,
		"alert_recipients": toAlertRecipients(recipients),
	})
}
