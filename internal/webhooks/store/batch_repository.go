package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// BatchRepository defines operations for batch jobs.
type BatchRepository interface {
	CreateBatchJob(ctx context.Context, tenantID uuid.UUID, consumer string, jobType BatchJobType, data *BatchJobData) (*BatchJob, error)
	GetBatchJob(ctx context.Context, tenantID uuid.UUID, batchID uuid.UUID) (*BatchJob, error)
	UpdateBatchJobStatus(ctx context.Context, batchID uuid.UUID, from, to BatchJobStatus) error
	UpdateBatchJobProgress(ctx context.Context, batchID uuid.UUID, processedDelta, failedDelta int) error
	CleanupExpiredBatchJobs(ctx context.Context) (int, error)
	SnapshotEventIDs(ctx context.Context, tenantID uuid.UUID, filter EventReportFilter) ([]string, error)
	SnapshotDeliveryIDs(ctx context.Context, tenantID uuid.UUID, filter DeliveryFilter) ([]string, error)
}

// CreateBatchJob inserts a new batch job with snapshotted item IDs.
func (r *Repository) CreateBatchJob(ctx context.Context, tenantID uuid.UUID, consumer string, jobType BatchJobType, data *BatchJobData) (*BatchJob, error) {
	if len(data.ItemIDs) > MaxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum of %d", len(data.ItemIDs), MaxBatchSize)
	}

	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal batch data: %w", err)
	}

	now := time.Now()
	job := &BatchJob{
		ID:         uuid.New(),
		TenantID:   tenantID,
		Consumer:   consumer,
		JobType:    jobType,
		Status:     BatchStatusPending,
		Data:       dataJSON,
		Total:      len(data.ItemIDs),
		Processed:  0,
		Failed:     0,
		TTLSeconds: DefaultBatchTTLSeconds,
		CreatedAt:  now,
		ExpiresAt:  now.Add(time.Duration(DefaultBatchTTLSeconds) * time.Second),
		UpdatedAt:  now,
	}

	query := `
		INSERT INTO batch_jobs (id, tenant_id, consumer, job_type, status, data, total, processed, failed, ttl_seconds, created_at, expires_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.conn.ExecContext(ctx, query,
		job.ID, job.TenantID, job.Consumer, job.JobType, job.Status,
		job.Data, job.Total, job.Processed, job.Failed, job.TTLSeconds,
		job.CreatedAt, job.ExpiresAt, job.UpdatedAt,
	)
	if err != nil {
		return nil, storage.Error(err)
	}

	return job, nil
}

// GetBatchJob retrieves a batch job by ID within a tenant.
// Returns nil, nil if not found.
func (r *Repository) GetBatchJob(ctx context.Context, tenantID uuid.UUID, batchID uuid.UUID) (*BatchJob, error) {
	query := `
		SELECT id, tenant_id, consumer, job_type, status, data, total, processed, failed,
		       ttl_seconds, created_at, expires_at, updated_at
		FROM batch_jobs
		WHERE id = $1 AND tenant_id = $2
	`

	var job BatchJob
	err := r.conn.GetContext(ctx, &job, query, batchID, tenantID)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &job, nil
}

// UpdateBatchJobStatus atomically transitions a batch job from one status to
// another (compare-and-set). Returns storage.ErrNotFound when no row matched,
// i.e. the job does not exist or is no longer in the expected `from` status.
func (r *Repository) UpdateBatchJobStatus(ctx context.Context, batchID uuid.UUID, from, to BatchJobStatus) error {
	query := `
		UPDATE batch_jobs
		SET status = $3, updated_at = NOW()
		WHERE id = $1 AND status = $2
	`

	res, err := r.conn.ExecContext(ctx, query, batchID, from, to)
	if err != nil {
		return storage.Error(err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return storage.Error(err)
	}
	if rows == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// UpdateBatchJobProgress atomically increments the processed/failed counters.
// Uses atomic SQL increments to support concurrent updates from the worker.
func (r *Repository) UpdateBatchJobProgress(ctx context.Context, batchID uuid.UUID, processedDelta, failedDelta int) error {
	query := `
		UPDATE batch_jobs
		SET processed = processed + $2,
		    failed = failed + $3,
		    updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.conn.ExecContext(ctx, query, batchID, processedDelta, failedDelta)
	return storage.Error(err)
}

// CleanupExpiredBatchJobs deletes expired batch jobs that were never started
// (prepared snapshots the user abandoned). Processing jobs are left alone even
// past expiry: a large batch can outlive its TTL and the worker still owns it.
// Returns the number of deleted rows.
func (r *Repository) CleanupExpiredBatchJobs(ctx context.Context) (int, error) {
	query := `
		DELETE FROM batch_jobs
		WHERE expires_at < NOW()
		  AND status = 'pending'
	`

	result, err := r.conn.ExecContext(ctx, query)
	if err != nil {
		return 0, storage.Error(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rows), nil
}

// SnapshotEventIDs runs the event report filter query WITHOUT pagination to capture
// all matching event IDs (up to MaxBatchSize). Used by prepare_repush.
func (r *Repository) SnapshotEventIDs(ctx context.Context, tenantID uuid.UUID, filter EventReportFilter) ([]string, error) {
	args, err := eventReportFilterArgs(tenantID, filter)
	if err != nil {
		return nil, err
	}
	args = append(args, MaxBatchSize+1)
	query := `SELECT er.id::text FROM event_records er ` + eventReportFilterWhere + `
		ORDER BY er.created_at DESC
		LIMIT $8`

	var ids []string
	if err := r.conn.SelectContext(ctx, &ids, query, args...); err != nil {
		return nil, storage.Error(err)
	}

	if len(ids) > MaxBatchSize {
		return nil, fmt.Errorf("filter matches more than %d events; narrow your filter criteria", MaxBatchSize)
	}

	return ids, nil
}

// SnapshotDeliveryIDs runs the delivery filter query WITHOUT pagination to capture
// all matching delivery IDs (up to MaxBatchSize). Used by prepare_retry.
func (r *Repository) SnapshotDeliveryIDs(ctx context.Context, tenantID uuid.UUID, filter DeliveryFilter) ([]string, error) {
	args := append(deliveryFilterArgs(tenantID, filter), MaxBatchSize+1)
	query := `SELECT wd.id::text ` + deliveryFilterFrom + `
		ORDER BY wd.created_at DESC
		LIMIT $11`

	var ids []string
	err := r.conn.SelectContext(ctx, &ids, query, args...)
	if err != nil {
		return nil, storage.Error(err)
	}

	if len(ids) > MaxBatchSize {
		return nil, fmt.Errorf("filter matches more than %d deliveries; narrow your filter criteria", MaxBatchSize)
	}

	return ids, nil
}
