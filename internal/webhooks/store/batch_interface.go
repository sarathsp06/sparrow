package store

import (
	"context"

	"github.com/google/uuid"
)

// BatchRepository defines operations for batch jobs.
type BatchRepository interface {
	CreateBatchJob(ctx context.Context, tenantID uuid.UUID, namespace string, jobType BatchJobType, data *BatchJobData) (*BatchJob, error)
	GetBatchJob(ctx context.Context, tenantID uuid.UUID, batchID uuid.UUID) (*BatchJob, error)
	UpdateBatchJobStatus(ctx context.Context, batchID uuid.UUID, status BatchJobStatus) error
	UpdateBatchJobProgress(ctx context.Context, batchID uuid.UUID, processedDelta, failedDelta int) error
	CleanupExpiredBatchJobs(ctx context.Context) (int, error)
	SnapshotEventIDs(ctx context.Context, tenantID uuid.UUID, filter EventReportFilter) ([]string, error)
	SnapshotDeliveryIDs(ctx context.Context, tenantID uuid.UUID, filter DeliveryFilter) ([]string, error)
}
