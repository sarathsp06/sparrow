package queue

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// EventProcessingWorker processes events and triggers webhook deliveries
type EventProcessingWorker struct {
	river.WorkerDefaults[EventArgs]
	gate             *tenantGate // per-tenant cap; nil when self-hosted
	logger           *slog.Logger
	subscriptionRepo store.SubscriptionRepository
	eventRepo        store.EventRepository
	deliveryRepo     store.DeliveryRepository
	jobInserter      JobInserter
}

// NewEventProcessingWorker creates a new event processing worker with a river client
func NewEventProcessingWorker(subscriptionRepo store.SubscriptionRepository, eventRepo store.EventRepository, deliveryRepo store.DeliveryRepository, jobInserter JobInserter) *EventProcessingWorker {
	return &EventProcessingWorker{
		subscriptionRepo: subscriptionRepo,
		eventRepo:        eventRepo,
		deliveryRepo:     deliveryRepo,
		logger:           slog.Default().With("component", "event-processing-worker"),
		jobInserter:      jobInserter,
	}
}

// fanOutNamespace scopes the delivery IDs fan-out derives.
var fanOutNamespace = uuid.MustParse("6f1d4c52-8b0e-4f3a-9c41-5a2e7d9b0c13")

// FanOutDeliveryID is the ID of the delivery fan-out creates for an event and
// a subscription: the same pair always yields the same ID, which makes the
// insert idempotent.
func FanOutDeliveryID(eventID, subscriptionID uuid.UUID) uuid.UUID {
	return uuid.NewSHA1(fanOutNamespace, append(eventID[:], subscriptionID[:]...))
}

// Work processes an event and creates webhook delivery jobs
func (w *EventProcessingWorker) Work(ctx context.Context, job *river.Job[EventArgs]) error {
	args := job.Args
	w.logger.InfoContext(ctx, "Processing event", "event_id", args.EventID, "consumer", args.Consumer, "event", args.Event)

	// Restore trace context from job metadata.
	ctx = otel.GetTextMapPropagator().Extract(ctx, traceCarrier(job.Metadata))

	// Parse job-arg IDs up front. Malformed args are permanent — cancel
	// instead of burning retries.
	tenantID, err := uuid.Parse(args.TenantID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("invalid tenant ID %q in job args: %w", args.TenantID, err))
	}
	release, admitted := w.gate.acquire(ctx, tenantID)
	if !admitted {
		return fairnessSnooze()
	}
	defer release()
	eventID, err := uuid.Parse(args.EventID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("invalid event ID %q in job args: %w", args.EventID, err))
	}

	// Verify the event record exists — it should have been stored by the
	// service layer. The repo returns (nil, nil) for missing rows.
	existingEvent, err := w.eventRepo.GetEventByID(ctx, tenantID, eventID)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to load event record", "error", err, "event_id", args.EventID)
		return fmt.Errorf("failed to load event record: %w", err)
	}
	if existingEvent == nil {
		// Missing row is a permanent condition — retrying won't create it.
		w.logger.ErrorContext(ctx, "Event record not found in database", "event_id", args.EventID)
		return river.JobCancel(fmt.Errorf("event record %s not found", args.EventID))
	}

	// Find all subscriptions for this consumer/event with webhook details (including label matching)
	subscriptions, err := w.subscriptionRepo.GetSubscriptionsWithWebhooksByEvent(ctx, tenantID, args.Consumer, args.Event, args.Labels)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to get event subscriptions", "error", err)
		return err
	}

	if len(subscriptions) == 0 {
		w.logger.InfoContext(ctx, "No subscriptions found for event",
			"consumer", args.Consumer,
			"event", args.Event,
		)
		return nil
	}

	w.logger.InfoContext(ctx, "Found subscriptions",
		"count", len(subscriptions),
		"consumer", args.Consumer,
		"event", args.Event,
	)

	// Calculate expiry: TTL=0 means no expiry (far-future sentinel).
	var expiresAt time.Time
	if args.TTLSeconds <= 0 {
		expiresAt = store.NoExpiryTime
	} else {
		expiresAt = time.Now().Add(time.Duration(args.TTLSeconds) * time.Second)
	}

	// Build all deliveries and job args in memory first, then insert both in
	// one transaction.
	deliveries := make([]*store.WebhookDelivery, 0, len(subscriptions))
	jobArgs := make(map[uuid.UUID]river.JobArgs, len(subscriptions))
	paused := 0

	for _, result := range subscriptions {
		sub := result.Subscription
		webhook := result.Webhook

		deliveryID := FanOutDeliveryID(eventID, sub.ID)

		maxAttempts := webhook.MaxDeliveryAttempts()

		delivery := &store.WebhookDelivery{
			ID:             deliveryID,
			WebhookID:      webhook.ID,
			EventID:        eventID,
			SubscriptionID: &sub.ID,
			Status:         store.StatusPending,
			MaxAttempts:    maxAttempts,
			ExpiresAt:      expiresAt,
		}
		deliveries = append(deliveries, delivery)

		// A paused subscription or webhook (manual or auto-disabled) still
		// gets its delivery row, so what was not delivered stays visible and
		// retryable, but nothing is queued.
		if reason := holdReason(webhook, sub); reason != "" {
			delivery.Status = store.StatusPaused
			delivery.ErrorMessage = reason
			paused++
			continue
		}

		jobArgs[deliveryID] = &WebhookArgs{
			TenantID:            args.TenantID,
			DeliveryID:          deliveryID.String(),
			WebhookID:           webhook.ID.String(),
			SubscriptionID:      sub.ID.String(),
			EventID:             args.EventID,
			ExpiresAt:           expiresAt,
			Consumer:            args.Consumer,
			MaxAttempts:         maxAttempts,
			RetryBackoffSeconds: webhook.RetryBackoffSeconds,
		}
	}

	// Delivery rows and their River jobs commit together: a crash or error
	// in between leaves neither, and River retries the fan-out. Rows that a
	// previous run of this job already committed are skipped with their
	// jobs, so a retried fan-out never delivers twice.
	queued := 0
	inserted, err := w.deliveryRepo.CreateFanOutDeliveries(ctx, tenantID, deliveries,
		func(ctx context.Context, tx *sql.Tx, inserted []*store.WebhookDelivery) error {
			jobs := make([]river.JobArgs, 0, len(inserted))
			for _, d := range inserted {
				if args, ok := jobArgs[d.ID]; ok {
					jobs = append(jobs, args)
				}
			}
			queued = len(jobs)
			if len(jobs) == 0 {
				return nil
			}
			return w.jobInserter.BatchInsertTx(ctx, tx, jobs)
		})
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to create deliveries and their jobs", "error", err, "count", len(deliveries))
		return fmt.Errorf("create fan-out deliveries: %w", err)
	}
	if skipped := len(deliveries) - len(inserted); skipped > 0 {
		w.logger.WarnContext(ctx, "Skipped deliveries an earlier run of this fan-out already created",
			"event_id", args.EventID, "skipped", skipped)
	}

	w.logger.InfoContext(ctx, "Scheduled webhook deliveries",
		"count", queued,
		"paused", paused,
		"consumer", args.Consumer,
		"event", args.Event,
	)

	w.logger.InfoContext(ctx, "Event processing completed",
		"event_id", args.EventID,
		"webhooks_scheduled", len(subscriptions),
	)

	return nil
}
