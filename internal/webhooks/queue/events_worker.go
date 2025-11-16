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
	w.logger.Info("Processing event", "event_id", args.EventID, "namespace", args.Namespace, "event", args.Event)

	// get trace id and set that as metadata
	carrier := make(propagation.MapCarrier)
	err := json.Unmarshal(job.Metadata, &carrier)
	if err != nil {
		w.logger.Error("Failed to unmarshal job metadata", "error", err, "event_id", args.EventID)
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	// Store the event record - this should have already been stored by the service layer
	// but let's verify and update if needed
	existingEvent, err := w.webhookRepo.GetEventByID(ctx, args.EventID)
	if err != nil {
		w.logger.Error("Event record not found in database", "error", err, "event_id", args.EventID)
		return fmt.Errorf("event record not found: %w", err)
	}

	// Update metadata if provided in the job args (for consistency)
	if len(args.Metadata) > 0 {
		existingEvent.Metadata = args.Metadata
	}

	// Find all registered webhooks for this namespace/event
	registeredWebhooks, err := w.webhookRepo.GetWebhooksByEvent(ctx, args.Namespace, args.Event)
	if err != nil {
		w.logger.Error("Failed to get registered webhooks", "error", err)
		return err
	}

	if len(registeredWebhooks) == 0 {
		w.logger.Info("No webhooks registered for event",
			"namespace", args.Namespace,
			"event", args.Event,
		)
		return nil
	}

	w.logger.Info("Found registered webhooks",
		"count", len(registeredWebhooks),
		"namespace", args.Namespace,
		"event", args.Event,
	)

	// Create webhook delivery jobs for each registered webhook
	expiresAt := time.Now().Add(time.Duration(args.TTLSeconds) * time.Second)

	for _, webhook := range registeredWebhooks {
		deliveryID := uuid.New().String()

		// Calculate max attempts from webhook configuration (default 3)
		maxAttempts := 3
		if webhook.MaxRetries > 0 {
			maxAttempts = webhook.MaxRetries + 1 // MaxRetries is retry count, so add 1 for initial attempt
		}

		// Create webhook delivery record
		delivery := &store.WebhookDelivery{
			ID:          deliveryID,
			WebhookID:   webhook.ID,
			EventID:     args.EventID,
			Status:      store.StatusPending,
			MaxAttempts: maxAttempts,
			ExpiresAt:   expiresAt,
		}

		if err := w.webhookRepo.CreateDelivery(ctx, delivery); err != nil {
			w.logger.Error("Failed to create delivery record", "error", err, "webhook_id", webhook.ID)
			continue
		}

		// Create webhook delivery job with minimal data
		webhookArgs := WebhookArgs{
			DeliveryID:  deliveryID,
			WebhookID:   webhook.ID,
			EventID:     args.EventID,
			ExpiresAt:   expiresAt,
			Namespace:   args.Namespace,
			MaxAttempts: delivery.MaxAttempts,
		}

		_, err := w.jobInserter.Insert(ctx, &webhookArgs)
		if err != nil {
			w.logger.Error("Failed to schedule webhook delivery job",
				"error", err,
				"webhook_id", webhook.ID,
				"delivery_id", deliveryID,
			)
			continue
		}

		w.logger.Info("Scheduled webhook delivery",
			"webhook_id", webhook.ID,
			"delivery_id", deliveryID,
			"webhook_url", webhook.URL,
		)
	}

	w.logger.Info("Event processing completed",
		"event_id", args.EventID,
		"webhooks_scheduled", len(registeredWebhooks),
	)

	return nil
}
