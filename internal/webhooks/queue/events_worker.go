package queue

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// EventProcessingWorker processes events and triggers webhook deliveries
type EventProcessingWorker struct {
	river.WorkerDefaults[EventArgs]
	logger      *slog.Logger
	webhookRepo store.RepositoryInterface
	jobInserter JobInserter
}

// NewEventProcessingWorker creates a new event processing worker with a river client
func NewEventProcessingWorker(webhookRepo store.RepositoryInterface, riverClient *river.Client[pgx.Tx]) *EventProcessingWorker {
	return &EventProcessingWorker{
		webhookRepo: webhookRepo,
		logger:      logger.NewLogger("event-processing-worker"),
		jobInserter: NewJobInserter(riverClient),
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

	// Store the event record
	eventRecord := &store.EventRecord{
		ID:        args.EventID,
		Namespace: args.Namespace,
		Event:     args.Event,
		Payload:   args.Payload,
		TTL:       args.TTLSeconds,
		Metadata:  args.Metadata,
		CreatedAt: args.CreatedAt,
	}

	if err := w.webhookRepo.StoreEvent(ctx, eventRecord); err != nil {
		w.logger.Error("Failed to store event record", "error", err, "event_id", args.EventID)
		return err
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

		// Create webhook delivery record
		delivery := &store.WebhookDelivery{
			ID:          deliveryID,
			WebhookID:   webhook.ID,
			EventID:     args.EventID,
			Status:      store.StatusPending,
			MaxAttempts: 3, // Default max attempts
			ExpiresAt:   expiresAt,
		}

		if err := w.webhookRepo.CreateDelivery(ctx, delivery); err != nil {
			w.logger.Error("Failed to create delivery record", "error", err, "webhook_id", webhook.ID)
			continue
		}

		// Create webhook delivery job
		webhookArgs := WebhookArgs{
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
			"url", webhook.URL,
		)
	}

	w.logger.Info("Event processing completed",
		"event_id", args.EventID,
		"webhooks_scheduled", len(registeredWebhooks),
	)

	return nil
}
