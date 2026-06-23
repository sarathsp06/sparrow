package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// StoreEvent persists an event record with automatic ID generation and timestamp management.
func (r *Repository) StoreEvent(ctx context.Context, tenantID uuid.UUID, event *EventRecord) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	event.TenantID = tenantID
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if event.ExpiresAt.IsZero() {
		if event.TTL <= 0 {
			event.ExpiresAt = NoExpiryTime
		} else {
			event.ExpiresAt = time.Now().Add(time.Duration(event.TTL) * time.Second)
		}
	}

	query := `
		INSERT INTO event_records (
			id, tenant_id, namespace, event, payload, ttl, metadata, labels, schema_valid, idempotency_key, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	labelsJSON, err := json.Marshal(event.Labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	_, err = r.conn.ExecContext(ctx, query,
		event.ID,
		event.TenantID,
		event.Namespace,
		event.Event,
		event.Payload,
		event.TTL,
		metadataJSON,
		labelsJSON,
		event.SchemaValid,
		event.IdempotencyKey,
		event.CreatedAt,
		event.ExpiresAt,
	)
	return storage.Error(err)
}

// GetEventByID gets an event record by ID within a tenant
func (r *Repository) GetEventByID(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) (*EventRecord, error) {
	query := `
		SELECT id, tenant_id, namespace, event, payload, ttl, metadata, labels, schema_valid, idempotency_key, created_at, expires_at
		FROM event_records
		WHERE id = $1 AND tenant_id = $2
	`

	var eventRow EventRecord

	err := r.conn.GetContext(ctx, &eventRow, query, eventID, tenantID)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &eventRow, nil
}

// GetEventByIdempotencyKey looks up an event record by its client-provided idempotency key.
// Returns nil, nil when no matching record exists.
func (r *Repository) GetEventByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, namespace, idempotencyKey string) (*EventRecord, error) {
	query := `
		SELECT id, tenant_id, namespace, event, payload, ttl, metadata, labels, schema_valid, idempotency_key, created_at, expires_at
		FROM event_records
		WHERE tenant_id = $1 AND namespace = $2 AND idempotency_key = $3
	`

	var eventRow EventRecord
	err := r.conn.GetContext(ctx, &eventRow, query, tenantID, namespace, idempotencyKey)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &eventRow, nil
}

// ListEventReports gets event records in descending order by creation time.
// Uses ($N::type IS NULL OR col = $N) guards so unset filters become no-op.
func (r *Repository) ListEventReports(ctx context.Context, tenantID uuid.UUID, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error) {
	var ns any
	if namespace != "" {
		ns = namespace
	}

	args := []any{tenantID, ns, eventName}

	baseQuery := `
		SELECT
			id, tenant_id, namespace, event, payload, ttl, metadata, labels, schema_valid, idempotency_key, created_at, expires_at
		FROM event_records
		WHERE tenant_id = $1
		  AND ($2::text IS NULL OR namespace = $2)
		  AND ($3::text IS NULL OR event = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`

	countQuery := `
		SELECT COUNT(*)
		FROM event_records
		WHERE tenant_id = $1
		  AND ($2::text IS NULL OR namespace = $2)
		  AND ($3::text IS NULL OR event = $3)
	`

	queryArgs := append(args, limit, offset)

	var eventRows []EventRecord
	err := r.conn.SelectContext(ctx, &eventRows, baseQuery, queryArgs...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	var totalCount int
	err = r.conn.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	var events []*EventReportWithStats
	for _, row := range eventRows {
		events = append(events, &EventReportWithStats{
			EventRecord: row,
		})
	}

	return events, totalCount, nil
}

// ListEventReportsWithStats retrieves event records enriched with delivery statistics.
// Uses ($N::type IS NULL OR col = $N) guards so unset filters become no-op.
func (r *Repository) ListEventReportsWithStats(ctx context.Context, tenantID uuid.UUID, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error) {
	var ns any
	if namespace != "" {
		ns = namespace
	}

	args := []any{tenantID, ns, eventName}

	baseQuery := `
		SELECT
			er.id, er.tenant_id, er.namespace, er.event, er.payload, er.ttl,
			er.metadata, er.labels, er.schema_valid, er.created_at, er.expires_at,
			COALESCE(ds.webhook_count, 0) as webhook_count,
			COALESCE(ds.successful_deliveries, 0) as successful_deliveries,
			COALESCE(ds.failed_deliveries, 0) as failed_deliveries,
			COALESCE(ds.pending_deliveries, 0) as pending_deliveries
		FROM event_records er
		LEFT JOIN (
			SELECT
				wd.event_id,
				COUNT(DISTINCT wd.webhook_id) as webhook_count,
				SUM(CASE WHEN wh.success = true THEN 1 ELSE 0 END) as successful_deliveries,
				SUM(CASE WHEN wh.success = false THEN 1 ELSE 0 END) as failed_deliveries,
				COUNT(CASE WHEN wd.status IN ('pending', 'sending', 'retrying') THEN 1 END) as pending_deliveries
			FROM webhook_deliveries wd
			LEFT JOIN webhook_health_events wh ON wd.id = wh.delivery_id
			GROUP BY wd.event_id
		) ds ON er.id = ds.event_id
		WHERE er.tenant_id = $1
		  AND ($2::text IS NULL OR er.namespace = $2)
		  AND ($3::text IS NULL OR er.event = $3::text)
		ORDER BY er.created_at DESC
		LIMIT $4 OFFSET $5
	`

	countQuery := `
		SELECT COUNT(*)
		FROM event_records
		WHERE tenant_id = $1
		  AND ($2::text IS NULL OR namespace = $2)
		  AND ($3::text IS NULL OR event = $3::text)
	`

	queryArgs := append(args, limit, offset)

	var events []*EventReportWithStats
	err := r.conn.SelectContext(ctx, &events, baseQuery, queryArgs...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	var totalCount int
	err = r.conn.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return events, totalCount, nil
}

// ListEventReportsFiltered retrieves event records with delivery statistics using
// fixed IS NULL guard conditions. Labels filter uses @> operator; time range,
// schema_valid, and event_name are optional via ($N::type IS NULL OR col = $N).
func (r *Repository) ListEventReportsFiltered(ctx context.Context, tenantID uuid.UUID, filter EventReportFilter) ([]*EventReportWithStats, int, error) {
	var ns any
	if filter.Namespace != "" {
		ns = filter.Namespace
	}

	var labelsJSON any
	if len(filter.Labels) > 0 {
		b, err := json.Marshal(filter.Labels)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to marshal label filter: %w", err)
		}
		labelsJSON = string(b)
	}

	args := []any{tenantID, ns, filter.EventName, filter.SchemaValid, labelsJSON, filter.CreatedAfter, filter.CreatedBefore}

	baseQuery := `
		SELECT
			er.id, er.tenant_id, er.namespace, er.event, er.payload, er.ttl,
			er.metadata, er.labels, er.schema_valid, er.created_at, er.expires_at,
			COALESCE(ds.webhook_count, 0) as webhook_count,
			COALESCE(ds.successful_deliveries, 0) as successful_deliveries,
			COALESCE(ds.failed_deliveries, 0) as failed_deliveries,
			COALESCE(ds.pending_deliveries, 0) as pending_deliveries
		FROM event_records er
		LEFT JOIN (
			SELECT
				wd.event_id,
				COUNT(DISTINCT wd.webhook_id) as webhook_count,
				SUM(CASE WHEN wh.success = true THEN 1 ELSE 0 END) as successful_deliveries,
				SUM(CASE WHEN wh.success = false THEN 1 ELSE 0 END) as failed_deliveries,
				COUNT(CASE WHEN wd.status IN ('pending', 'sending', 'retrying') THEN 1 END) as pending_deliveries
			FROM webhook_deliveries wd
			LEFT JOIN webhook_health_events wh ON wd.id = wh.delivery_id
			GROUP BY wd.event_id
		) ds ON er.id = ds.event_id
		WHERE er.tenant_id = $1
		  AND ($2::text IS NULL OR er.namespace = $2)
		  AND ($3::text IS NULL OR er.event = $3)
		  AND ($4::boolean IS NULL OR er.schema_valid = $4)
		  AND ($5::jsonb IS NULL OR er.labels @> $5::jsonb)
		  AND ($6::timestamptz IS NULL OR er.created_at >= $6)
		  AND ($7::timestamptz IS NULL OR er.created_at <= $7)
		ORDER BY er.created_at DESC
		LIMIT $8 OFFSET $9
	`

	countQuery := `
		SELECT COUNT(*)
		FROM event_records
		WHERE tenant_id = $1
		  AND ($2::text IS NULL OR namespace = $2)
		  AND ($3::text IS NULL OR event = $3)
		  AND ($4::boolean IS NULL OR schema_valid = $4)
		  AND ($5::jsonb IS NULL OR labels @> $5::jsonb)
		  AND ($6::timestamptz IS NULL OR created_at >= $6)
		  AND ($7::timestamptz IS NULL OR created_at <= $7)
	`

	queryArgs := append(args, filter.Limit, filter.Offset)
	var events []*EventReportWithStats
	err := r.conn.SelectContext(ctx, &events, baseQuery, queryArgs...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	var totalCount int
	err = r.conn.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return events, totalCount, nil
}

// DeleteEventByID deletes an event record by its ID within a tenant.
// Used as a compensation action when downstream operations (e.g. job insertion) fail
// after the event has already been stored.
func (r *Repository) DeleteEventByID(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) error {
	query := `DELETE FROM event_records WHERE id = $1 AND tenant_id = $2`
	_, err := r.conn.ExecContext(ctx, query, eventID, tenantID)
	return storage.Error(err)
}

// GetEventDeliveryStats gets delivery statistics for a specific event within a tenant
func (r *Repository) GetEventDeliveryStats(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) (int32, int32, int32, int32, error) {
	query := `
		SELECT
			COUNT(DISTINCT wd.webhook_id) as webhook_count,
			SUM(CASE WHEN wh.success = true THEN 1 ELSE 0 END) as successful_deliveries,
			SUM(CASE WHEN wh.success = false THEN 1 ELSE 0 END) as failed_deliveries,
			COUNT(CASE WHEN wd.status IN ('pending', 'sending', 'retrying') THEN 1 END) as pending_deliveries
		FROM webhook_deliveries wd
		LEFT JOIN webhook_health_events wh ON wd.id = wh.delivery_id
		JOIN event_records er ON wd.event_id = er.id
		WHERE wd.event_id = $1 AND er.tenant_id = $2
	`

	var result struct {
		WebhookCount         int32 `db:"webhook_count"`
		SuccessfulDeliveries int32 `db:"successful_deliveries"`
		FailedDeliveries     int32 `db:"failed_deliveries"`
		PendingDeliveries    int32 `db:"pending_deliveries"`
	}

	err := r.conn.GetContext(ctx, &result, query, eventID, tenantID)
	if err != nil {
		return 0, 0, 0, 0, storage.Error(err)
	}

	return result.WebhookCount, result.SuccessfulDeliveries, result.FailedDeliveries, result.PendingDeliveries, nil
}
