package workers

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/sarathsp06/httpqueue/internal/jobs"
	"github.com/sarathsp06/httpqueue/internal/logger"
	"github.com/sarathsp06/httpqueue/internal/webhooks"
)

// EventProcessingWorker processes events and triggers webhook deliveries
type EventProcessingWorker struct {
	river.WorkerDefaults[jobs.EventArgs]
	webhookRepo *webhooks.Repository
	riverClient *river.Client[pgx.Tx]
}

// NewEventProcessingWorker creates a new event processing worker with a river client
func NewEventProcessingWorker(webhookRepo *webhooks.Repository, riverClient *river.Client[pgx.Tx]) *EventProcessingWorker {
	return &EventProcessingWorker{
		webhookRepo: webhookRepo,
		riverClient: riverClient,
	}
}

// Work processes an event and creates webhook delivery jobs
func (w *EventProcessingWorker) Work(ctx context.Context, job *river.Job[jobs.EventArgs]) error {
	log := logger.NewLogger("event-worker")
	args := job.Args

	log.Info("Processing event",
		"event_id", args.EventID,
		"namespace", args.Namespace,
		"event", args.Event,
	)

	// Store the event record
	eventRecord := &webhooks.EventRecord{
		ID:        args.EventID,
		Namespace: args.Namespace,
		Event:     args.Event,
		Payload:   args.Payload,
		TTL:       args.TTLSeconds,
		Metadata:  args.Metadata,
		CreatedAt: args.CreatedAt,
	}

	if err := w.webhookRepo.StoreEvent(ctx, eventRecord); err != nil {
		log.Error("Failed to store event record", "error", err, "event_id", args.EventID)
		return err
	}

	// Find all registered webhooks for this namespace/event
	registeredWebhooks, err := w.webhookRepo.GetWebhooksByEvent(ctx, args.Namespace, args.Event)
	if err != nil {
		log.Error("Failed to get registered webhooks", "error", err)
		return err
	}

	if len(registeredWebhooks) == 0 {
		log.Info("No webhooks registered for event",
			"namespace", args.Namespace,
			"event", args.Event,
		)
		return nil
	}

	log.Info("Found registered webhooks",
		"count", len(registeredWebhooks),
		"namespace", args.Namespace,
		"event", args.Event,
	)

	// Create webhook delivery jobs for each registered webhook
	expiresAt := time.Now().Add(time.Duration(args.TTLSeconds) * time.Second)

	for _, webhook := range registeredWebhooks {
		deliveryID := uuid.New().String()

		// Create webhook delivery record
		delivery := &webhooks.WebhookDelivery{
			ID:          deliveryID,
			WebhookID:   webhook.ID,
			EventID:     args.EventID,
			Status:      webhooks.StatusPending,
			MaxAttempts: 3, // Default max attempts
			ExpiresAt:   expiresAt,
		}

		if err := w.webhookRepo.CreateDelivery(ctx, delivery); err != nil {
			log.Error("Failed to create delivery record", "error", err, "webhook_id", webhook.ID)
			continue
		}

		// Create webhook delivery job
		webhookArgs := jobs.WebhookArgs{
			DeliveryID: deliveryID,
			WebhookID:  webhook.ID,
			EventID:    args.EventID,
			URL:        webhook.URL,
			Headers:    webhook.Headers,
			Payload:    args.Payload,
			Timeout:    webhook.Timeout,
			ExpiresAt:  expiresAt,
			Namespace:  args.Namespace,
			Event:      args.Event,
		}

		_, err := w.riverClient.Insert(ctx, webhookArgs, &river.InsertOpts{
			Queue: "webhooks",
		})
		if err != nil {
			log.Error("Failed to schedule webhook delivery job",
				"error", err,
				"webhook_id", webhook.ID,
				"delivery_id", deliveryID,
			)
			continue
		}

		log.Info("Scheduled webhook delivery",
			"webhook_id", webhook.ID,
			"delivery_id", deliveryID,
			"url", webhook.URL,
		)
	}

	log.Info("Event processing completed",
		"event_id", args.EventID,
		"webhooks_scheduled", len(registeredWebhooks),
	)

	return nil
}
