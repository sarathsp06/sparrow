package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"

	"github.com/sarathsp06/sparrow/internal/storage"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// progressUpdateInterval controls how often the worker flushes progress counters to the DB.
const progressUpdateInterval = 25

// batchJobTimeout bounds one run of a batch job. A batch holds up to
// MaxBatchSize items, each a few queries and a job insert; River's 60s
// default cut large batches short. A run that is cut short anyway (timeout
// or shutdown) records its progress and the retry resumes after it. It stays
// well below River's RescueStuckJobsAfter (1h), past which a running job is
// considered stuck and queued again.
const batchJobTimeout = 30 * time.Minute

// batchFinalWriteTimeout bounds the progress and status writes made after
// the job context is done, so they still land when the job was cancelled.
const batchFinalWriteTimeout = 10 * time.Second

// BatchJobWorker processes batch jobs (event re-push and delivery retry).
// It reads item IDs from the batch_jobs row and dispatches each one.
type BatchJobWorker struct {
	river.WorkerDefaults[BatchJobArgs]
	logger       *slog.Logger
	batchRepo    store.BatchRepository
	eventRepo    store.EventRepository
	deliveryRepo store.DeliveryRepository
	webhookRepo  store.WebhookRepository
	jobInserter  JobInserter
}

// NewBatchJobWorker creates a new batch job worker.
func NewBatchJobWorker(batchRepo store.BatchRepository, eventRepo store.EventRepository, deliveryRepo store.DeliveryRepository, webhookRepo store.WebhookRepository, jobInserter JobInserter) *BatchJobWorker {
	return &BatchJobWorker{
		batchRepo:    batchRepo,
		eventRepo:    eventRepo,
		deliveryRepo: deliveryRepo,
		webhookRepo:  webhookRepo,
		logger:       slog.Default().With("component", "batch-job-worker"),
		jobInserter:  jobInserter,
	}
}

// Timeout overrides River's 60s default; see batchJobTimeout.
func (w *BatchJobWorker) Timeout(*river.Job[BatchJobArgs]) time.Duration {
	return batchJobTimeout
}

// setTerminalStatus CAS-transitions the batch from processing to a terminal
// status. A lost race (someone else already transitioned it, e.g. a
// cancellation) is logged and swallowed — the other writer's status wins.
func (w *BatchJobWorker) setTerminalStatus(ctx context.Context, batchID uuid.UUID, to store.BatchJobStatus) error {
	err := w.batchRepo.UpdateBatchJobStatus(ctx, batchID, store.BatchStatusProcessing, to)
	if storage.IsNotFound(err) {
		w.logger.InfoContext(ctx, "Batch terminal status transition lost race, keeping existing status",
			"batch_id", batchID, "to", to)
		return nil
	}
	return err
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

	// Parse job-arg IDs up front. Malformed args are permanent — cancel
	// instead of burning retries.
	tenantID, err := uuid.Parse(args.TenantID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("invalid tenant ID %q in job args: %w", args.TenantID, err))
	}
	batchID, err := uuid.Parse(args.BatchID)
	if err != nil {
		return river.JobCancel(fmt.Errorf("invalid batch ID %q in job args: %w", args.BatchID, err))
	}

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
		if statusErr := w.setTerminalStatus(ctx, batchID, store.BatchStatusFailed); statusErr != nil {
			w.logger.ErrorContext(ctx, "Failed to mark batch failed", "error", statusErr, "batch_id", args.BatchID)
		}
		return fmt.Errorf("failed to parse batch data: %w", err)
	}

	w.logger.InfoContext(ctx, "Batch job loaded",
		"batch_id", args.BatchID,
		"job_type", batch.JobType,
		"total_items", len(data.ItemIDs))

	// Items already counted belong to an earlier run of this job that was
	// cut short (timeout, shutdown); resume after them instead of handling
	// them twice. Progress is flushed in order, so the count is a prefix.
	items := data.ItemIDs[min(batch.Processed+batch.Failed, len(data.ItemIDs)):]

	var processed, failed int

	switch batch.JobType {
	case store.BatchTypeEventRepush:
		processed, failed = w.processEventRepush(ctx, tenantID, batchID, items)
	case store.BatchTypeDeliveryRetry:
		processed, failed = w.processDeliveryRetry(ctx, tenantID, batchID, batch.Consumer, items)
	default:
		if statusErr := w.setTerminalStatus(ctx, batchID, store.BatchStatusFailed); statusErr != nil {
			w.logger.ErrorContext(ctx, "Failed to mark batch failed", "error", statusErr, "batch_id", args.BatchID)
		}
		return fmt.Errorf("unknown batch job type: %s", batch.JobType)
	}

	// The job context may be done (timeout, shutdown): the remaining writes
	// still have to land, or the batch stays processing with lost progress.
	jobErr := ctx.Err()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), batchFinalWriteTimeout)
	defer cancel()

	// Final progress flush
	if err := w.batchRepo.UpdateBatchJobProgress(ctx, batchID, processed, failed); err != nil {
		w.logger.ErrorContext(ctx, "Failed to update final batch progress", "error", err)
	}
	if jobErr != nil {
		// Cut short: River retries the job, which resumes after the
		// flushed progress.
		w.logger.WarnContext(ctx, "Batch job interrupted, will resume", "batch_id", args.BatchID, "error", jobErr)
		return fmt.Errorf("batch job %s interrupted: %w", args.BatchID, jobErr)
	}

	// Re-read the batch from DB to get cumulative totals for the terminal
	// status decision. The local processed/failed counters only reflect
	// the last chunk (they're reset after each periodic flush).
	batch, err = w.batchRepo.GetBatchJob(ctx, tenantID, batchID)
	if err != nil {
		w.logger.ErrorContext(ctx, "Failed to re-read batch for terminal status", "error", err)
		// Fall back to using the last-chunk values if re-read fails
	} else if batch != nil {
		// The batch was cancelled while we were processing — don't overwrite
		// the cancelled status with a terminal one.
		if batch.Status == store.BatchStatusCancelled {
			w.logger.InfoContext(ctx, "Batch cancelled during processing, keeping cancelled status", "batch_id", args.BatchID)
			return nil
		}
		processed = batch.Processed
		failed = batch.Failed
	}

	// Set terminal status using cumulative totals
	finalStatus := store.BatchStatusCompleted
	if failed > 0 && processed == 0 {
		finalStatus = store.BatchStatusFailed
	}
	if err := w.setTerminalStatus(ctx, batchID, finalStatus); err != nil {
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
		if ctx.Err() != nil {
			return processed, failed // job timed out or is shutting down
		}
		// Check cancellation periodically
		if i > 0 && i%progressUpdateInterval == 0 {
			batch, err := w.batchRepo.GetBatchJob(ctx, tenantID, batchID)
			if err == nil && batch != nil && batch.Status == store.BatchStatusCancelled {
				w.logger.InfoContext(ctx, "Batch cancelled mid-processing", "batch_id", batchID, "processed_so_far", processed)
				return processed, failed
			}
			// Flush progress. On error keep counting: the counters carry
			// over to the next flush, so the resume point stays exact.
			if err := w.batchRepo.UpdateBatchJobProgress(ctx, batchID, processed, failed); err == nil {
				processed, failed = 0, 0
			}
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
		// Only Sparrow emits its own sparrow.* events: the single re-push
		// refuses them, and replaying one here would resend its alert
		// emails. Counted as failed so the job shows it.
		if strings.HasPrefix(strings.ToLower(original.Event), store.SystemEventPrefix) {
			w.logger.WarnContext(ctx, "Skipping re-push of a Sparrow system event", "event_id", idStr, "event", original.Event)
			failed++
			continue
		}

		// Create a new event record (fresh ID, same payload/consumer/event).
		// Preserve the original TTL. TTL=0 means no expiry -- StoreEvent
		// will set ExpiresAt to the far-future sentinel automatically.
		// A batch replay does not re-validate the payload, so it keeps the
		// event type version and validation result the original was
		// accepted under.
		newID := uuid.New()
		newEvent := &store.EventRecord{
			ID:           newID,
			Consumer:     original.Consumer,
			Event:        original.Event,
			Payload:      original.Payload,
			TTL:          original.TTL,
			Metadata:     original.Metadata,
			Labels:       original.Labels,
			SchemaValid:  original.SchemaValid,
			EventVersion: original.EventVersion,
			CreatedAt:    time.Now(),
		}

		// The copy and its processing job commit together, so a cancel or
		// crash in between leaves neither.
		job := EventArgs{
			TenantID:   tenantID.String(),
			EventID:    newID.String(),
			Consumer:   original.Consumer,
			Event:      original.Event,
			TTLSeconds: original.TTL,
			Metadata:   original.Metadata,
			Labels:     original.Labels,
			CreatedAt:  newEvent.CreatedAt,
		}
		err = w.eventRepo.StoreEventAndEnqueue(ctx, tenantID, newEvent, func(ctx context.Context, tx *sql.Tx) error {
			return w.jobInserter.BatchInsertTx(ctx, tx, []river.JobArgs{job})
		})
		if err != nil {
			w.logger.ErrorContext(ctx, "Failed to re-push event", "original_id", idStr, "error", err)
			failed++
			continue
		}

		processed++
	}
	return processed, failed
}

// processDeliveryRetry resets and re-enqueues each delivery for retry.
func (w *BatchJobWorker) processDeliveryRetry(ctx context.Context, tenantID, batchID uuid.UUID, consumer string, itemIDs []string) (processed, failed int) {
	for i, idStr := range itemIDs {
		if ctx.Err() != nil {
			return processed, failed // job timed out or is shutting down
		}
		// Check cancellation periodically
		if i > 0 && i%progressUpdateInterval == 0 {
			batch, err := w.batchRepo.GetBatchJob(ctx, tenantID, batchID)
			if err == nil && batch != nil && batch.Status == store.BatchStatusCancelled {
				w.logger.InfoContext(ctx, "Batch cancelled mid-processing", "batch_id", batchID, "processed_so_far", processed)
				return processed, failed
			}
			// Flush progress. On error keep counting: the counters carry
			// over to the next flush, so the resume point stays exact.
			if err := w.batchRepo.UpdateBatchJobProgress(ctx, batchID, processed, failed); err == nil {
				processed, failed = 0, 0
			}
		}

		deliveryID, err := uuid.Parse(idStr)
		if err != nil {
			w.logger.ErrorContext(ctx, "Invalid delivery ID in batch", "delivery_id", idStr, "error", err)
			failed++
			continue
		}

		// Load delivery
		delivery, err := w.deliveryRepo.GetDeliveryByID(ctx, tenantID, deliveryID, consumer)
		if err != nil || delivery == nil {
			w.logger.ErrorContext(ctx, "Failed to load delivery for retry", "delivery_id", idStr, "error", err)
			failed++
			continue
		}

		// Get webhook (consumer + retry config) BEFORE mutating the delivery,
		// so a failed fetch doesn't strand the delivery in a reset state with
		// no job enqueued.
		webhook, err := w.webhookRepo.GetWebhookByID(ctx, tenantID, delivery.WebhookID, consumer)
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

		// Reset the delivery and queue its job in one transaction, so a
		// cancel or crash in between leaves neither. Manual batch retries
		// never expire -- use far-future sentinel so TTL doesn't apply.
		job := &WebhookArgs{
			TenantID:            tenantID.String(),
			DeliveryID:          deliveryID.String(),
			WebhookID:           delivery.WebhookID.String(),
			SubscriptionID:      subID,
			EventID:             delivery.EventID.String(),
			ExpiresAt:           store.NoExpiryTime,
			Consumer:            webhook.Consumer,
			MaxAttempts:         webhook.MaxDeliveryAttempts(),
			RetryBackoffSeconds: webhook.RetryBackoffSeconds,
		}
		reset, err := w.deliveryRepo.ResetDeliveriesForRetry(ctx, []uuid.UUID{deliveryID}, func(ctx context.Context, tx *sql.Tx, _ []uuid.UUID) error {
			return w.jobInserter.BatchInsertTx(ctx, tx, []river.JobArgs{job})
		})
		if err != nil {
			w.logger.ErrorContext(ctx, "Failed to retry delivery", "delivery_id", idStr, "error", err)
			failed++
			continue
		}
		if len(reset) == 0 {
			// Still in flight: its own job will deliver it.
			w.logger.InfoContext(ctx, "Skipped retrying a delivery still in flight", "delivery_id", idStr)
			failed++
			continue
		}

		processed++
	}
	return processed, failed
}
