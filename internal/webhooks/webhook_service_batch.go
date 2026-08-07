package webhooks

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/queue"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// --- Batch Operations ---

// loadAndValidateBatch parses a batch ID, loads the job, and verifies it matches
// the expected job type. This eliminates the repeated parse+get+nil-check+type-check
// pattern across all batch operations.
func (s *WebhookService) loadAndValidateBatch(ctx context.Context, batchID string, expectedType store.BatchJobType) (*store.BatchJob, error) {
	batchUUID, err := parseUUID(batchID, "batch ID")
	if err != nil {
		return nil, err
	}

	batch, err := s.webhookRepo.GetBatchJob(ctx, tenant.DefaultTenantID, batchUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch job: %w", err)
	}
	if batch == nil {
		return nil, svcerrors.Error(codes.NotFound, "batch job not found")
	}
	if batch.JobType != expectedType {
		return nil, svcerrors.Errorf(codes.FailedPrecondition, "batch job is not a %s", expectedType)
	}
	return batch, nil
}

// startBatch validates a pending batch job, transitions it to processing, and
// enqueues a River job for async execution. Used by RePushEvents and RetryDeliveries.
func (s *WebhookService) startBatch(ctx context.Context, batchID string, jobType store.BatchJobType) error {
	batch, err := s.loadAndValidateBatch(ctx, batchID, jobType)
	if err != nil {
		return err
	}
	if batch.Status != store.BatchStatusPending {
		return svcerrors.Errorf(codes.FailedPrecondition, "batch job is not in pending status (current: %s)", batch.Status)
	}
	if time.Now().After(batch.ExpiresAt) {
		return svcerrors.Error(codes.FailedPrecondition, "batch job has expired")
	}

	if err := s.webhookRepo.UpdateBatchJobStatus(ctx, batch.ID, store.BatchStatusProcessing); err != nil {
		return fmt.Errorf("failed to update batch status: %w", err)
	}

	_, err = s.jobInserter.Insert(ctx, &queue.BatchJobArgs{
		TenantID: tenant.DefaultTenantID.String(),
		BatchID:  batchID,
	})
	if err != nil {
		// Roll back status on enqueue failure
		_ = s.webhookRepo.UpdateBatchJobStatus(ctx, batch.ID, store.BatchStatusPending)
		return fmt.Errorf("failed to enqueue batch job: %w", err)
	}

	s.logger.InfoContext(ctx, "Batch job enqueued", "batch_id", batchID, "job_type", jobType, "total", batch.Total)
	return nil
}

// cancelBatch validates a batch job and transitions it to cancelled.
// Used by CancelRepush and CancelRetry.
func (s *WebhookService) cancelBatch(ctx context.Context, batchID string, jobType store.BatchJobType) error {
	batch, err := s.loadAndValidateBatch(ctx, batchID, jobType)
	if err != nil {
		return err
	}
	if batch.Status == store.BatchStatusCompleted || batch.Status == store.BatchStatusCancelled {
		return svcerrors.Errorf(codes.FailedPrecondition, "batch job is already in terminal state: %s", batch.Status)
	}

	if err := s.webhookRepo.UpdateBatchJobStatus(ctx, batch.ID, store.BatchStatusCancelled); err != nil {
		return fmt.Errorf("failed to cancel batch job: %w", err)
	}

	s.logger.InfoContext(ctx, "Batch job cancelled", "batch_id", batchID, "job_type", jobType)
	return nil
}

// RePushEvents starts async processing of a batch re-push.
// The batch must exist, belong to the tenant, be of type event_repush, and be in pending status.
func (s *WebhookService) RePushEvents(ctx context.Context, repushID string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.RePushEvents")
	defer span.End()
	return s.startBatch(ctx, repushID, store.BatchTypeEventRepush)
}

// GetRepushStatus returns the current state of a batch re-push.
func (s *WebhookService) GetRepushStatus(ctx context.Context, repushID string) (*store.BatchJob, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetRepushStatus")
	defer span.End()
	return s.loadAndValidateBatch(ctx, repushID, store.BatchTypeEventRepush)
}

// CancelRepush aborts a pending or in-progress batch re-push.
func (s *WebhookService) CancelRepush(ctx context.Context, repushID string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.CancelRepush")
	defer span.End()
	return s.cancelBatch(ctx, repushID, store.BatchTypeEventRepush)
}

// RetryDeliveries starts async processing of a batch delivery retry.
func (s *WebhookService) RetryDeliveries(ctx context.Context, retryID string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.RetryDeliveries")
	defer span.End()
	return s.startBatch(ctx, retryID, store.BatchTypeDeliveryRetry)
}

// GetRetryStatus returns the current state of a batch delivery retry.
func (s *WebhookService) GetRetryStatus(ctx context.Context, retryID string) (*store.BatchJob, error) {
	ctx, span := s.tracer.Start(ctx, "WebhookService.GetRetryStatus")
	defer span.End()
	return s.loadAndValidateBatch(ctx, retryID, store.BatchTypeDeliveryRetry)
}

// CancelRetry aborts a pending or in-progress batch delivery retry.
func (s *WebhookService) CancelRetry(ctx context.Context, retryID string) error {
	ctx, span := s.tracer.Start(ctx, "WebhookService.CancelRetry")
	defer span.End()
	return s.cancelBatch(ctx, retryID, store.BatchTypeDeliveryRetry)
}
