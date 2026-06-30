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

// progressUpdateInterval controls how often the worker flushes progress counters to the DB.
const progressUpdateInterval = 25

// BatchJobWorker processes batch jobs (event re-push and delivery retry).
// It reads item IDs from the batch_jobs row and dispatches each one.
type BatchJobWorker struct {
	river.WorkerDefaults[BatchJobArgs]
	logger      *slog.Logger
	batchRepo   store.BatchRepository
	eventRepo   store.EventRepository
	webhookRepo store.WebhookRepository
	jobInserter JobInserter
}

// NewBatchJobWorker creates a new batch job worker.
func NewBatchJobWorker(batchRepo store.BatchRepository, eventRepo store.EventRepository, webhookRepo store.WebhookRepository, jobInserter JobInserter) *BatchJobWorker {
	return &BatchJobWorker{
		batchRepo:   batchRepo,
		eventRepo:   eventRepo,
		webhookRepo: webhookRepo,
		logger:      slog.Default().With("component", "batch-job-worker"),
		jobInserter: jobInserter,
	}
}

// Work processes a batch job by dispatching each item based on job_type.
func (w *BatchJobWorker) Work(ctx context.Context, job *river.Job[BatchJobArgs]) error {
	args := job.Args
	w.logger.InfoContext(ctx, "Processing batch job", "batch_id", args.BatchID, "tenant_id", args.TenantID)

	// Restore trace context from job metadata
	carrier := make(propagation.MapCarrier)
	if err := json.Unmarshal(job.Metadata, &carrier); err != nil {
		w.logger.ErrorContext(ctx, "Failed to unmarshal job metadata", "error", err)
	}
	ctx = otel.GetTextMapPropagator().Extract(ctx, carrier)

	tenantID := uuid.MustParse(args.TenantID)
	batchID := uuid.MustParse(args.BatchID)

	batch, err := w.batchRepo.GetBatchJob(ctx, tenantID, batchID)
	if err != nil {
		return fmt.Errorf("failed to get batch job: %w", err)
	}
	if batch == nil {
		return fmt.Errorf("batch job %s not found", args.BatchID)
	}

	// Check if already cancelled before starting
	if batch.Status == store.BatchStatusCancelled {
		w.logger.InfoContext(ctx, "Batch job was cancelled, skipping", "batch_id", args.BatchID)
		return nil
	}

	data, err := batch.GetData()
	if err != nil {
		_ = w.batchRepo.UpdateBatchJobStatus(ctx, batchID, store.BatchStatusFailed)
		return fmt.Errorf("failed to parse batch data: %w", err)
	}

	w.logger.InfoContext(ctx, "Batch job loaded",
		"batch_id", args.BatchID,
		"job_type", batch.JobType,
		"total_items", len(data.ItemIDs))

	var processed, failed int

	switch batch.JobType {
	case store.BatchTypeEventRepush:
		processed, failed = w.processEventRepush(ctx, tenantID, batchID, data.ItemIDs)
	case store.BatchTypeDeliveryRetry:
		processed, failed = w.processDeliveryRetry(ctx, tenantID, batchID, batch.Namespace, data.ItemIDs)
	default:
		_ = w.batchRepo.UpdateBatchJobStatus(ctx, batchID, store.BatchStatusFailed)
		return fmt.Errorf("unknown batch job type: %s", batch.JobType)
	}

	// Final progress flush
	if err := w.batchRepo.UpdateBatchJobProgress(ctx, batchID, processed, failed); err != nil {
		w.logger.ErrorContext(ctx, "Failed to update final batch progress", "error", err)
	}

	// Re-read the batch from DB to get cumulative totals for the terminal
	// status decision. The local processed/failed counters only reflect
	// the last chunk (they're reset after each periodic flush).
	batch, err = w.batchRepo.GetBatchJob(ctx, tenantID, batchID)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to re-read batch for terminal status", "error", err)
		// Fall back to using the last-chunk values if re-read fails
	} else if batch != nil {
		processed = batch.Processed
		failed = batch.Failed
	}

	// Set terminal status using cumulative totals
	finalStatus := store.BatchStatusCompleted
	if failed > 0 && processed == 0 {
		finalStatus = store.BatchStatusFailed
	}
	if err := w.batchRepo.UpdateBatchJobStatus(ctx, batchID, finalStatus); err != nil {
		w.logger.ErrorContext(ctx, "Failed to set batch terminal status", "error", err)
	}

	w.logger.InfoContext(ctx, "Batch job completed",
		"batch_id", args.BatchID,
		"processed", processed,
		"failed", failed,
		"status", finalStatus)

	return nil
}

// processEventRepush re-pushes each event by loading the original record and enqueuing a new event job.
func (w *BatchJobWorker) processEventRepush(ctx context.Context, tenantID, batchID uuid.UUID, itemIDs []string) (processed, failed int) {
	for i, idStr := range itemIDs {
		// Check cancellation periodically
		if i > 0 && i%progressUpdateInterval == 0 {
			batch, err := w.batchRepo.GetBatchJob(ctx, tenantID, batchID)
			if err == nil && batch != nil && batch.Status == store.BatchStatusCancelled {
				w.logger.InfoContext(ctx, "Batch cancelled mid-processing", "batch_id", batchID, "processed_so_far", processed)
				return processed, failed
			}
			// Flush progress
			_ = w.batchRepo.UpdateBatchJobProgress(ctx, batchID, processed, failed)
			processed = 0
			failed = 0
		}

		eventID, err := uuid.Parse(idStr)
		if err != nil {
			w.logger.ErrorContext(ctx, "Invalid event ID in batch", "event_id", idStr, "error", err)
			failed++
			continue
		}

		// Load original event
		original, err := w.eventRepo.GetEventByID(ctx, tenantID, eventID)
		if err != nil || original == nil {
			w.logger.ErrorContext(ctx, "Failed to load event for repush", "event_id", idStr, "error", err)
			failed++
			continue
		}

		// Create a new event record (fresh ID, same payload/namespace/event).
		// Preserve the original TTL. TTL=0 means no expiry -- StoreEvent
		// will set ExpiresAt to the far-future sentinel automatically.
		newID := uuid.New()
		newEvent := &store.EventRecord{
			ID:          newID,
			Namespace:   original.Namespace,
			Event:       original.Event,
			Payload:     original.Payload,
			TTL:         original.TTL,
			Metadata:    original.Metadata,
			Labels:      original.Labels,
			SchemaValid: true, // Will be re-validated by event worker
			CreatedAt:   time.Now(),
		}

		if err := w.eventRepo.StoreEvent(ctx, tenantID, newEvent); err != nil {
			w.logger.ErrorContext(ctx, "Failed to store re-pushed event", "original_id", idStr, "error", err)
			failed++
			continue
		}

		// Enqueue event processing job
		_, err = w.jobInserter.Insert(ctx, EventArgs{
			TenantID:   tenantID.String(),
			EventID:    newID.String(),
			Namespace:  original.Namespace,
			Event:      original.Event,
			TTLSeconds: original.TTL,
			Metadata:   original.Metadata,
			Labels:     original.Labels,
			CreatedAt:  newEvent.CreatedAt,
		})
		if err != nil {
			w.logger.ErrorContext(ctx, "Failed to enqueue re-pushed event", "event_id", newID, "error", err)
			// Compensate: delete orphaned event record
			_ = w.eventRepo.DeleteEventByID(ctx, tenantID, newID)
			failed++
			continue
		}

		processed++
	}
	return processed, failed
}

// processDeliveryRetry resets and re-enqueues each delivery for retry.
func (w *BatchJobWorker) processDeliveryRetry(ctx context.Context, tenantID, batchID uuid.UUID, namespace string, itemIDs []string) (processed, failed int) {
	for i, idStr := range itemIDs {
		// Check cancellation periodically
		if i > 0 && i%progressUpdateInterval == 0 {
			batch, err := w.batchRepo.GetBatchJob(ctx, tenantID, batchID)
			if err == nil && batch != nil && batch.Status == store.BatchStatusCancelled {
				w.logger.InfoContext(ctx, "Batch cancelled mid-processing", "batch_id", batchID, "processed_so_far", processed)
				return processed, failed
			}
			// Flush progress
			_ = w.batchRepo.UpdateBatchJobProgress(ctx, batchID, processed, failed)
			processed = 0
			failed = 0
		}

		deliveryID, err := uuid.Parse(idStr)
		if err != nil {
			w.logger.ErrorContext(ctx, "Invalid delivery ID in batch", "delivery_id", idStr, "error", err)
			failed++
			continue
		}

		// Load delivery
		delivery, err := w.eventRepo.GetDeliveryByID(ctx, tenantID, deliveryID, namespace)
		if err != nil || delivery == nil {
			w.logger.ErrorContext(ctx, "Failed to load delivery for retry", "delivery_id", idStr, "error", err)
			failed++
			continue
		}

		// Reset delivery status
		if err := w.eventRepo.ResetDeliveryForRetry(ctx, deliveryID); err != nil {
			w.logger.ErrorContext(ctx, "Failed to reset delivery for retry", "delivery_id", idStr, "error", err)
			failed++
			continue
		}

		// Get webhook for namespace info
		webhook, err := w.webhookRepo.GetWebhookByID(ctx, tenantID, delivery.WebhookID, namespace)
		if err != nil {
			w.logger.ErrorContext(ctx, "Failed to get webhook for delivery retry", "delivery_id", idStr, "webhook_id", delivery.WebhookID, "error", err)
			failed++
			continue
		}

		// Build subscription ID string
		var subID string
		if delivery.SubscriptionID != nil {
			subID = delivery.SubscriptionID.String()
		}

		// Enqueue webhook delivery job. Manual batch retries never expire --
		// use far-future sentinel so TTL doesn't apply to explicit retries.
		_, err = w.jobInserter.Insert(ctx, &WebhookArgs{
			TenantID:       tenantID.String(),
			DeliveryID:     deliveryID.String(),
			WebhookID:      delivery.WebhookID.String(),
			SubscriptionID: subID,
			EventID:        delivery.EventID.String(),
			ExpiresAt:      store.NoExpiryTime,
			Namespace:      webhook.Namespace,
		})
		if err != nil {
			w.logger.ErrorContext(ctx, "Failed to enqueue delivery retry", "delivery_id", idStr, "error", err)
			failed++
			continue
		}

		processed++
	}
	return processed, failed
}
