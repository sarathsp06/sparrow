package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// CreateBatchJob inserts a new batch job with snapshotted item IDs.
func (r *Repository) CreateBatchJob(ctx context.Context, tenantID uuid.UUID, namespace string, jobType BatchJobType, data *BatchJobData) (*BatchJob, error) {
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
		Namespace:  namespace,
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
		INSERT INTO batch_jobs (id, tenant_id, namespace, job_type, status, data, total, processed, failed, ttl_seconds, created_at, expires_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.conn.ExecContext(ctx, query,
		job.ID, job.TenantID, job.Namespace, job.JobType, job.Status,
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
		SELECT id, tenant_id, namespace, job_type, status, data, total, processed, failed,
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

// UpdateBatchJobStatus atomically updates the status of a batch job.
// Returns the updated job or an error if the status transition is invalid.
func (r *Repository) UpdateBatchJobStatus(ctx context.Context, batchID uuid.UUID, status BatchJobStatus) error {
	query := `
		UPDATE batch_jobs
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`

	_, err := r.conn.ExecContext(ctx, query, batchID, status)
	return storage.Error(err)
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

// CleanupExpiredBatchJobs deletes batch jobs that have expired and are not in a terminal state.
// Returns the number of deleted rows.
func (r *Repository) CleanupExpiredBatchJobs(ctx context.Context) (int, error) {
	query := `
		DELETE FROM batch_jobs
		WHERE expires_at < NOW()
		  AND status NOT IN ('completed', 'cancelled')
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
	var ns any
	if filter.Namespace != "" {
		ns = filter.Namespace
	}

	var labelsJSON any
	if len(filter.Labels) > 0 {
		b, err := json.Marshal(filter.Labels)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal label filter: %w", err)
		}
		labelsJSON = string(b)
	}

	args := []any{tenantID, ns, filter.EventName, filter.SchemaValid, labelsJSON, filter.CreatedAfter, filter.CreatedBefore, MaxBatchSize + 1}

	query := `
		SELECT id::text FROM event_records
		WHERE tenant_id = $1
		  AND ($2::text IS NULL OR namespace = $2)
		  AND ($3::text IS NULL OR event = $3)
		  AND ($4::boolean IS NULL OR schema_valid = $4)
		  AND ($5::jsonb IS NULL OR labels @> $5::jsonb)
		  AND ($6::timestamptz IS NULL OR created_at >= $6)
		  AND ($7::timestamptz IS NULL OR created_at <= $7)
		ORDER BY created_at DESC
		LIMIT $8
	`

	var ids []string
	err := r.conn.SelectContext(ctx, &ids, query, args...)
	if err != nil {
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
	var ns any
	if filter.Namespace != "" {
		ns = filter.Namespace
	}

	args := []any{tenantID, ns, filter.WebhookID, filter.EventID, filter.Status, filter.ErrorCategory, filter.SubscriptionID, filter.CreatedAfter, filter.CreatedBefore, MaxBatchSize + 1}

	query := `
		SELECT wd.id::text
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wr.tenant_id = $1
		  AND ($2::text IS NULL OR wr.namespace = $2)
		  AND ($3::uuid IS NULL OR wd.webhook_id = $3)
		  AND ($4::uuid IS NULL OR wd.event_id = $4)
		  AND ($5::text IS NULL OR wd.status::text = $5)
		  AND ($6::text IS NULL OR wd.error_category = $6)
		  AND ($7::uuid IS NULL OR wd.subscription_id = $7)
		  AND ($8::timestamptz IS NULL OR wd.created_at >= $8)
		  AND ($9::timestamptz IS NULL OR wd.created_at <= $9)
		ORDER BY wd.created_at DESC
		LIMIT $10
	`

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


