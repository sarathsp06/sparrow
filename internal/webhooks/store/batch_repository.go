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
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Namespace != "" {
		conditions = append(conditions, fmt.Sprintf("namespace = $%d", argIdx))
		args = append(args, filter.Namespace)
		argIdx++
	}

	if filter.EventName != nil {
		conditions = append(conditions, fmt.Sprintf("event = $%d", argIdx))
		args = append(args, *filter.EventName)
		argIdx++
	}

	if filter.SchemaValid != nil {
		conditions = append(conditions, fmt.Sprintf("schema_valid = $%d", argIdx))
		args = append(args, *filter.SchemaValid)
		argIdx++
	}

	if len(filter.Labels) > 0 {
		labelsJSON, err := json.Marshal(filter.Labels)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal label filter: %w", err)
		}
		conditions = append(conditions, fmt.Sprintf("labels @> $%d::jsonb", argIdx))
		args = append(args, string(labelsJSON))
		argIdx++
	}

	if filter.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.CreatedAfter)
		argIdx++
	}

	if filter.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.CreatedBefore)
		argIdx++
	}

	whereClause := "WHERE " + joinConditions(conditions)

	// Limit to MaxBatchSize + 1 to detect overflow
	query := fmt.Sprintf(`
		SELECT id::text FROM event_records
		%s
		ORDER BY created_at DESC
		LIMIT $%d
	`, whereClause, argIdx)
	args = append(args, MaxBatchSize+1)

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
	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("wr.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if filter.Namespace != "" {
		conditions = append(conditions, fmt.Sprintf("wr.namespace = $%d", argIdx))
		args = append(args, filter.Namespace)
		argIdx++
	}

	if filter.WebhookID != nil {
		conditions = append(conditions, fmt.Sprintf("wd.webhook_id = $%d", argIdx))
		args = append(args, *filter.WebhookID)
		argIdx++
	}

	if filter.EventID != nil {
		conditions = append(conditions, fmt.Sprintf("wd.event_id = $%d", argIdx))
		args = append(args, *filter.EventID)
		argIdx++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("wd.status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.ErrorCategory != nil {
		conditions = append(conditions, fmt.Sprintf("wd.error_category = $%d", argIdx))
		args = append(args, *filter.ErrorCategory)
		argIdx++
	}

	if filter.SubscriptionID != nil {
		conditions = append(conditions, fmt.Sprintf("wd.subscription_id = $%d", argIdx))
		args = append(args, *filter.SubscriptionID)
		argIdx++
	}

	if filter.CreatedAfter != nil {
		conditions = append(conditions, fmt.Sprintf("wd.created_at >= $%d", argIdx))
		args = append(args, *filter.CreatedAfter)
		argIdx++
	}

	if filter.CreatedBefore != nil {
		conditions = append(conditions, fmt.Sprintf("wd.created_at <= $%d", argIdx))
		args = append(args, *filter.CreatedBefore)
		argIdx++
	}

	whereClause := "WHERE " + joinConditions(conditions)

	query := fmt.Sprintf(`
		SELECT wd.id::text
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		%s
		ORDER BY wd.created_at DESC
		LIMIT $%d
	`, whereClause, argIdx)
	args = append(args, MaxBatchSize+1)

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

// joinConditions joins SQL conditions with AND.
func joinConditions(conditions []string) string {
	result := ""
	for i, c := range conditions {
		if i > 0 {
			result += " AND "
		}
		result += c
	}
	return result
}
