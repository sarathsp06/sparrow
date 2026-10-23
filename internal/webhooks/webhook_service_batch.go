package webhooks

import (
	"context"
	"fmt"
	"time"

	svcerrors "github.com/sarathsp06/sparrow/internal/errors"
	"github.com/sarathsp06/sparrow/internal/storage"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// --- Batch Operations ---

// BatchScope says whose batch jobs a caller may reach. Consumer is the
// consumer in the request path. A job snapshotted for one consumer is
// reachable only under that consumer; a job snapshotted across all consumers
// (Consumer "") is reachable under any consumer path unless ConsumerOnly is
// set, which consumer-scoped callers (portal tokens) always set. A job outside
// the scope answers not found, so its existence is not revealed.
type BatchScope struct {
	Consumer     string
	ConsumerOnly bool
}

func (b BatchScope) allows(job *store.BatchJob) bool {
	if job.Consumer == b.Consumer {
		return true
	}
	return job.Consumer == "" && !b.ConsumerOnly
}

// loadAndValidateBatch parses a batch ID, loads the job, and verifies it is in
// scope and matches the expected job type. This eliminates the repeated
// parse+get+nil-check+type-check pattern across all batch operations.
func (s *WebhookService) loadAndValidateBatch(ctx context.Context, scope BatchScope, batchID string, expectedType store.BatchJobType) (*store.BatchJob, error) {
	batchUUID, err := parseUUID(batchID, "batch ID")
	if err != nil {
		return nil, err
	}

	batch, err := s.webhookRepo.GetBatchJob(ctx, tenant.DefaultTenantID, batchUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch job: %w", err)
	}
	if batch == nil || !scope.allows(batch) {
		return nil, svcerrors.Error(svcerrors.NotFound, "batch job not found")
	}
	if batch.JobType != expectedType {
		return nil, svcerrors.Errorf(svcerrors.FailedPrecondition, "batch job is not a %s", expectedType)
	}
	return batch, nil
}

// startBatch validates a pending batch job, transitions it to processing, and
// enqueues a River job for async execution. Used by RePushEvents and RetryDeliveries.
func (s *WebhookService) startBatch(ctx context.Context, scope BatchScope, batchID string, jobType store.BatchJobType) error {
	batch, err := s.loadAndValidateBatch(ctx, scope, batchID, jobType)
	if err != nil {
		return err
	}
	if batch.Status != store.BatchStatusPending {
		return svcerrors.Errorf(svcerrors.FailedPrecondition, "batch job is not in pending status (current: %s)", batch.Status)
	}
	if time.Now().After(batch.ExpiresAt) {
		return svcerrors.Error(svcerrors.FailedPrecondition, "batch job has expired")
	}

	// CAS pending→processing: only one caller wins; a concurrent start loses the race.
	if err := s.webhookRepo.UpdateBatchJobStatus(ctx, batch.TenantID, batch.ID, store.BatchStatusPending, store.BatchStatusProcessing); err != nil {
		if storage.IsNotFound(err) {
			return svcerrors.Error(svcerrors.FailedPrecondition, "batch job is no longer in pending status")
		}
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	_, err = s.jobInserter.Insert(ctx, &queue.BatchJobArgs{
		TenantID: batch.TenantID.String(),
		BatchID:  batchID,
	})
	if err != nil {
		// Roll back status on enqueue failure
		_ = s.webhookRepo.UpdateBatchJobStatus(ctx, batch.TenantID, batch.ID, store.BatchStatusProcessing, store.BatchStatusPending)
		return fmt.Errorf("failed to enqueue batch job: %w", err)
	}

	s.logger.InfoContext(ctx, "Batch job enqueued", "batch_id", batchID, "job_type", jobType, "total", batch.Total)
	return nil
}

// cancelBatch validates a batch job and transitions it to cancelled.
// Used by CancelRepush and CancelRetry.
func (s *WebhookService) cancelBatch(ctx context.Context, scope BatchScope, batchID string, jobType store.BatchJobType) error {
	batch, err := s.loadAndValidateBatch(ctx, scope, batchID, jobType)
	if err != nil {
		return err
	}
	if batch.Status == store.BatchStatusCompleted || batch.Status == store.BatchStatusCancelled {
		return svcerrors.Errorf(svcerrors.FailedPrecondition, "batch job is already in terminal state: %s", batch.Status)
	}

	// CAS from the observed status: if it changed concurrently (e.g. completed), fail rather than clobber.
	if err := s.webhookRepo.UpdateBatchJobStatus(ctx, batch.TenantID, batch.ID, batch.Status, store.BatchStatusCancelled); err != nil {
		if storage.IsNotFound(err) {
			return svcerrors.Error(svcerrors.FailedPrecondition, "batch job status changed concurrently; retry cancel")
		}
		return fmt.Errorf("failed to cancel batch job: %w", err)
	}

	s.logger.InfoContext(ctx, "Batch job cancelled", "batch_id", batchID, "job_type", jobType)
	return nil
}

// RePushEvents starts async processing of a batch re-push.
// The batch must exist, belong to the tenant, be of type event_repush, and be in pending status.
func (s *WebhookService) RePushEvents(ctx context.Context, scope BatchScope, repushID string) error {
	return s.startBatch(ctx, scope, repushID, store.BatchTypeEventRepush)
}

// GetRepushStatus returns the current state of a batch re-push.
func (s *WebhookService) GetRepushStatus(ctx context.Context, scope BatchScope, repushID string) (*store.BatchJob, error) {
	return s.loadAndValidateBatch(ctx, scope, repushID, store.BatchTypeEventRepush)
}

// CancelRepush aborts a pending or in-progress batch re-push.
func (s *WebhookService) CancelRepush(ctx context.Context, scope BatchScope, repushID string) error {
	return s.cancelBatch(ctx, scope, repushID, store.BatchTypeEventRepush)
}

// RetryDeliveries starts async processing of a batch delivery retry.
func (s *WebhookService) RetryDeliveries(ctx context.Context, scope BatchScope, retryID string) error {
	return s.startBatch(ctx, scope, retryID, store.BatchTypeDeliveryRetry)
}

// GetRetryStatus returns the current state of a batch delivery retry.
func (s *WebhookService) GetRetryStatus(ctx context.Context, scope BatchScope, retryID string) (*store.BatchJob, error) {
	return s.loadAndValidateBatch(ctx, scope, retryID, store.BatchTypeDeliveryRetry)
}

// CancelRetry aborts a pending or in-progress batch delivery retry.
func (s *WebhookService) CancelRetry(ctx context.Context, scope BatchScope, retryID string) error {
	return s.cancelBatch(ctx, scope, retryID, store.BatchTypeDeliveryRetry)
}
