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

	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// EventProcessingWorker processes events and triggers webhook deliveries
type EventProcessingWorker struct {
	river.WorkerDefaults[EventArgs]
	logger      *slog.Logger
	webhookRepo store.RepositoryInterface
	jobInserter JobInserter
}

// NewEventProcessingWorker creates a new event processing worker with a river client
func NewEventProcessingWorker(webhookRepo store.RepositoryInterface, jobInserter JobInserter) *EventProcessingWorker {
	return &EventProcessingWorker{
		webhookRepo: webhookRepo,
		logger:      logger.NewLogger("event-processing-worker"),
		jobInserter: jobInserter,
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

	// Parse tenant ID from job args
	tenantID := uuid.MustParse(args.TenantID)

	// Store the event record - this should have already been stored by the service layer
	// but let's verify and update if needed
	existingEvent, err := w.webhookRepo.GetEventByID(ctx, tenantID, uuid.MustParse(args.EventID))
	if err != nil {
		w.logger.ErrorContext(ctx, "Event record not found in database", "error", err, "event_id", args.EventID)
		return fmt.Errorf("event record not found: %w", err)
	}

	// Update metadata if provided in the job args (for consistency)
	if len(args.Metadata) > 0 {
		existingEvent.Metadata = args.Metadata
	}

	// Find all subscriptions for this namespace/event with webhook details (including label matching)
	subscriptions, err := w.webhookRepo.GetSubscriptionsWithWebhooksByEvent(ctx, tenantID, args.Namespace, args.Event, args.Labels)
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
			EventID:        uuid.MustParse(args.EventID),
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
	if err := w.webhookRepo.BatchCreateDeliveries(ctx, tenantID, deliveries); err != nil {
		w.logger.ErrorContext(ctx, "Failed to batch-create delivery records", "error", err, "count", len(deliveries))
		return fmt.Errorf("batch create deliveries: %w", err)
	}

	// Batch-insert all River jobs (single InsertMany call).
	if _, err := w.jobInserter.BatchInsert(ctx, jobArgs); err != nil {
		w.logger.ErrorContext(ctx, "Failed to batch-insert webhook delivery jobs",
			"error", err,
			"count", len(jobArgs),
		)
		// Compensation: remove orphaned delivery records since the jobs
		// that would process them could not be created.
		for _, d := range deliveries {
			if delErr := w.webhookRepo.DeleteDeliveryByID(ctx, d.ID); delErr != nil {
				w.logger.ErrorContext(ctx, "Failed to delete orphaned delivery record",
					"error", delErr,
					"delivery_id", d.ID,
				)
			}
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
