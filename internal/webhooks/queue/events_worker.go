package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// EventProcessingWorker processes events and triggers webhook deliveries
type EventProcessingWorker struct {
	river.WorkerDefaults[EventArgs]
	logger           *slog.Logger
	subscriptionRepo store.SubscriptionRepository
	eventRepo        store.EventRepository
	jobInserter      JobInserter
}

// NewEventProcessingWorker creates a new event processing worker with a river client
func NewEventProcessingWorker(subscriptionRepo store.SubscriptionRepository, eventRepo store.EventRepository, jobInserter JobInserter) *EventProcessingWorker {
	return &EventProcessingWorker{
		subscriptionRepo: subscriptionRepo,
		eventRepo:        eventRepo,
		logger:           slog.Default().With("component", "event-processing-worker"),
		jobInserter:      jobInserter,
	}
}

// Work processes an event and creates webhook delivery jobs
func (w *EventProcessingWorker) Work(ctx context.Context, job *river.Job[EventArgs]) error {
	args := job.Args
	w.logger.InfoContext(ctx, "Processing event", "event_id", args.EventID, "namespace", args.Namespace, "event", args.Event)

	// get trace id and set that as metadata
	carrier := make(propagation.MapCarrier)
	err := json.Unmarshal(job.Metadata, &carrier)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to unmarshal job metadata", "error", err, "event_id", args.EventID)
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	// Parse job-arg IDs up front. Malformed args are permanent — cancel
	// instead of burning retries.
	tenantID, err := uuid.Parse(args.TenantID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("invalid tenant ID %q in job args: %w", args.TenantID, err))
	}
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

	// Find all subscriptions for this namespace/event with webhook details (including label matching)
	subscriptions, err := w.subscriptionRepo.GetSubscriptionsWithWebhooksByEvent(ctx, tenantID, args.Namespace, args.Event, args.Labels)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to get event subscriptions", "error", err)
		return err
	}

	if len(subscriptions) == 0 {
		w.logger.InfoContext(ctx, "No subscriptions found for event",
			"namespace", args.Namespace,
			"event", args.Event,
		)
		return nil
	}

	w.logger.InfoContext(ctx, "Found subscriptions",
		"count", len(subscriptions),
		"namespace", args.Namespace,
		"event", args.Event,
	)

	// Calculate expiry: TTL=0 means no expiry (far-future sentinel).
	var expiresAt time.Time
	if args.TTLSeconds <= 0 {
		expiresAt = store.NoExpiryTime
	} else {
		expiresAt = time.Now().Add(time.Duration(args.TTLSeconds) * time.Second)
	}

	// Build all deliveries and job args in memory first, then batch-insert.
	deliveries := make([]*store.WebhookDelivery, 0, len(subscriptions))
	jobArgs := make([]river.JobArgs, 0, len(subscriptions))

	for _, result := range subscriptions {
		sub := result.Subscription
		webhook := result.Webhook

		deliveryID := uuid.New()

		// Calculate max attempts from webhook configuration (default 3)
		maxAttempts := 3
		if webhook.MaxRetries > 0 {
			maxAttempts = webhook.MaxRetries + 1 // MaxRetries is retry count, so add 1 for initial attempt
		}

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

		jobArgs = append(jobArgs, &WebhookArgs{
			TenantID:       args.TenantID,
			DeliveryID:     deliveryID.String(),
			WebhookID:      webhook.ID.String(),
			SubscriptionID: sub.ID.String(),
			EventID:        args.EventID,
			ExpiresAt:      expiresAt,
			Namespace:      args.Namespace,
			MaxAttempts:    maxAttempts,
		})
	}

	// Batch-insert all delivery records (single multi-row INSERT).
	if err := w.eventRepo.BatchCreateDeliveries(ctx, tenantID, deliveries); err != nil {
		w.logger.ErrorContext(ctx, "Failed to batch-create delivery records", "error", err, "count", len(deliveries))
		return fmt.Errorf("batch create deliveries: %w", err)
	}

	// Batch-insert all River jobs (single InsertMany call).
	// ponytail: delivery rows (sqlx) and River jobs (pgx) are inserted in
	// separate transactions with delete-based compensation. Upgrade path:
	// run both on a single pgx connection and use river.InsertManyTx.
	if _, err := w.jobInserter.BatchInsert(ctx, jobArgs); err != nil {
		w.logger.ErrorContext(ctx, "Failed to batch-insert webhook delivery jobs",
			"error", err,
			"count", len(jobArgs),
		)
		// Compensation: remove orphaned delivery records since the jobs
		// that would process them could not be created.
		var undeleted []uuid.UUID
		for _, d := range deliveries {
			if delErr := w.eventRepo.DeleteDeliveryByID(ctx, d.ID); delErr != nil {
				w.logger.ErrorContext(ctx, "Failed to delete orphaned delivery record",
					"error", delErr,
					"delivery_id", d.ID,
				)
				undeleted = append(undeleted, d.ID)
			}
		}
		if len(undeleted) > 0 {
			// Compensation itself failed. Return the original error so River
			// retries the whole job — duplicate deliveries are preferable to
			// silently-stuck pending rows.
			w.logger.ErrorContext(ctx, "COMPENSATION FAILED: orphaned delivery records remain after job-insert failure",
				"event_id", args.EventID,
				"delivery_ids", undeleted,
			)
		}
		return fmt.Errorf("batch insert jobs: %w", err)
	}

	w.logger.InfoContext(ctx, "Scheduled webhook deliveries",
		"count", len(deliveries),
		"namespace", args.Namespace,
		"event", args.Event,
	)

	w.logger.InfoContext(ctx, "Event processing completed",
		"event_id", args.EventID,
		"webhooks_scheduled", len(subscriptions),
	)

	return nil
}
