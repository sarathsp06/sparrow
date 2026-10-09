package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// EventRepository defines operations for event records and event report listings.
type EventRepository interface {
	StoreEvent(ctx context.Context, tenantID uuid.UUID, event *EventRecord) error
	GetEventByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, consumer, idempotencyKey string) (*EventRecord, error)
	GetEventByID(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) (*EventRecord, error)
	GetEventDeliveryStats(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) (int32, int32, int32, int32, error)
	DeleteEventByID(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) error
	DeleteEventsBefore(ctx context.Context, cutoff time.Time) (int64, error)

	ListEventReports(ctx context.Context, tenantID uuid.UUID, consumer string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error)
	ListEventReportsWithStats(ctx context.Context, tenantID uuid.UUID, consumer string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error)
	ListEventReportsFiltered(ctx context.Context, tenantID uuid.UUID, filter EventReportFilter) ([]*EventReportWithStats, bool, error)
}

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
			id, tenant_id, consumer, event, payload, ttl, metadata, labels, schema_valid, idempotency_key, created_at, expires_at, event_version
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
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
		event.Consumer,
		event.Event,
		event.Payload,
		event.TTL,
		metadataJSON,
		labelsJSON,
		event.SchemaValid,
		event.IdempotencyKey,
		event.CreatedAt,
		event.ExpiresAt,
		eventVersionOrDefault(event.EventVersion),
	)
	return storage.Error(err)
}

// eventVersionOrDefault stores version 1 for callers that predate versioning
// and leave EventVersion unset.
func eventVersionOrDefault(v int) int {
	if v <= 0 {
		return 1
	}
	return v
}

// GetEventByID gets an event record by ID within a tenant
func (r *Repository) GetEventByID(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) (*EventRecord, error) {
	query := `
		SELECT id, tenant_id, consumer, event, payload, ttl, metadata, labels, schema_valid, COALESCE(event_version, 1) AS event_version, idempotency_key, created_at, expires_at
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
func (r *Repository) GetEventByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, consumer, idempotencyKey string) (*EventRecord, error) {
	query := `
		SELECT id, tenant_id, consumer, event, payload, ttl, metadata, labels, schema_valid, COALESCE(event_version, 1) AS event_version, idempotency_key, created_at, expires_at
		FROM event_records
		WHERE tenant_id = $1 AND consumer = $2 AND idempotency_key = $3
	`

	var eventRow EventRecord
	err := r.conn.GetContext(ctx, &eventRow, query, tenantID, consumer, idempotencyKey)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &eventRow, nil
}

// DeleteEventsBefore purges event records created before cutoff, in batches
// of 10k to keep transactions and lock windows small. Deliveries cascade via
// FK; webhook_health_events rows older than cutoff are purged too (they have
// no FK to deliveries). Server-wide by design: retention is an operator
// policy, not a tenant-scoped API operation. Returns events deleted.
func (r *Repository) DeleteEventsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	const batch = `
		DELETE FROM event_records
		WHERE id IN (SELECT id FROM event_records WHERE created_at < $1 LIMIT 10000)
	`
	var total int64
	for {
		res, err := r.conn.ExecContext(ctx, batch, cutoff)
		if err != nil {
			return total, storage.Error(err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return total, storage.Error(err)
		}
		total += n
		if n < 10000 {
			break
		}
	}
	_, err := r.conn.ExecContext(ctx, `DELETE FROM webhook_health_events WHERE timestamp < $1`, cutoff)
	if err != nil {
		return total, storage.Error(err)
	}
	return total, nil
}

// ListEventReports gets event records in descending order by creation time.
// Uses ($N::type IS NULL OR col = $N) guards so unset filters become no-op.
func (r *Repository) ListEventReports(ctx context.Context, tenantID uuid.UUID, consumer string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error) {
	var ns any
	if consumer != "" {
		ns = consumer
	}

	args := []any{tenantID, ns, eventName}

	baseQuery := `
		SELECT
			id, tenant_id, consumer, event, payload, ttl, metadata, labels, schema_valid, COALESCE(event_version, 1) AS event_version, idempotency_key, created_at, expires_at
		FROM event_records
		WHERE tenant_id = $1
		  AND ($2::text IS NULL OR consumer = $2)
		  AND ($3::text IS NULL OR event = $3)
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`

	countQuery := `
		SELECT COUNT(*)
		FROM event_records
		WHERE tenant_id = $1
		  AND ($2::text IS NULL OR consumer = $2)
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
func (r *Repository) ListEventReportsWithStats(ctx context.Context, tenantID uuid.UUID, consumer string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error) {
	var ns any
	if consumer != "" {
		ns = consumer
	}

	args := []any{tenantID, ns, eventName}

	baseQuery := `
		SELECT
			er.id, er.tenant_id, er.consumer, er.event, er.payload, er.ttl,
			er.metadata, er.labels, er.schema_valid, COALESCE(er.event_version, 1) AS event_version, er.created_at, er.expires_at,
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
		  AND ($2::text IS NULL OR er.consumer = $2)
		  AND ($3::text IS NULL OR er.event = $3::text)
		ORDER BY er.created_at DESC
		LIMIT $4 OFFSET $5
	`

	countQuery := `
		SELECT COUNT(*)
		FROM event_records
		WHERE tenant_id = $1
		  AND ($2::text IS NULL OR consumer = $2)
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

// eventReportFilterWhere is the WHERE of every EventReportFilter query (page,
// re-push snapshot) over event_records er, with the filter in
// eventReportFilterArgs order as $1..$7.
const eventReportFilterWhere = `
		WHERE er.tenant_id = $1
		  AND ($2::text IS NULL OR er.consumer = $2)
		  AND ($3::text IS NULL OR er.event = $3)
		  AND ($4::boolean IS NULL OR er.schema_valid = $4)
		  AND ($5::jsonb IS NULL OR er.labels @> $5::jsonb)
		  AND ($6::timestamptz IS NULL OR er.created_at >= $6)
		  AND ($7::timestamptz IS NULL OR er.created_at <= $7)`

// eventReportFilterArgs binds an EventReportFilter to eventReportFilterWhere.
func eventReportFilterArgs(tenantID uuid.UUID, filter EventReportFilter) ([]any, error) {
	var ns any
	if filter.Consumer != "" {
		ns = filter.Consumer
	}
	var labelsJSON any
	if len(filter.Labels) > 0 {
		b, err := json.Marshal(filter.Labels)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal label filter: %w", err)
		}
		labelsJSON = string(b)
	}
	return []any{tenantID, ns, filter.EventName, filter.SchemaValid, labelsJSON, filter.CreatedAfter, filter.CreatedBefore}, nil
}

// ListEventReportsFiltered returns one page of event records, newest first,
// with their delivery statistics, and whether more follow. The page is cut
// first and statistics are computed for its rows only: aggregating every
// delivery and then joining scanned the whole deliveries table per page.
// There is no total, and filter.After continues after a row (keyset).
func (r *Repository) ListEventReportsFiltered(ctx context.Context, tenantID uuid.UUID, filter EventReportFilter) ([]*EventReportWithStats, bool, error) {
	args, err := eventReportFilterArgs(tenantID, filter)
	if err != nil {
		return nil, false, err
	}
	afterAt, afterID := cursorArgs(filter.After)
	args = append(args, afterAt, afterID, filter.Limit+1, filter.Offset)

	query := `
		SELECT
			page.id, page.tenant_id, page.consumer, page.event, page.payload, page.ttl,
			page.metadata, page.labels, page.schema_valid, page.event_version, page.created_at, page.expires_at,
			COALESCE(ds.webhook_count, 0) AS webhook_count,
			COALESCE(ds.successful_deliveries, 0) AS successful_deliveries,
			COALESCE(ds.failed_deliveries, 0) AS failed_deliveries,
			COALESCE(ds.pending_deliveries, 0) AS pending_deliveries
		FROM (
			SELECT er.id, er.tenant_id, er.consumer, er.event, er.payload, er.ttl,
			       er.metadata, er.labels, er.schema_valid, COALESCE(er.event_version, 1) AS event_version,
			       er.created_at, er.expires_at
			FROM event_records er
			` + eventReportFilterWhere + `
			  AND er.created_at <= $8 AND (er.created_at < $8 OR er.id < $9)
			ORDER BY er.created_at DESC, er.id DESC
			LIMIT $10 OFFSET $11
		) page
		LEFT JOIN LATERAL (
			SELECT
				COUNT(DISTINCT wd.webhook_id) AS webhook_count,
				SUM(CASE WHEN wh.success = true THEN 1 ELSE 0 END) AS successful_deliveries,
				SUM(CASE WHEN wh.success = false THEN 1 ELSE 0 END) AS failed_deliveries,
				COUNT(CASE WHEN wd.status IN ('pending', 'sending', 'retrying') THEN 1 END) AS pending_deliveries
			FROM webhook_deliveries wd
			LEFT JOIN webhook_health_events wh ON wd.id = wh.delivery_id
			WHERE wd.event_id = page.id
		) ds ON true
		ORDER BY page.created_at DESC, page.id DESC
	`

	var events []*EventReportWithStats
	if err := r.conn.SelectContext(ctx, &events, query, args...); err != nil {
		return nil, false, storage.Error(err)
	}
	if len(events) > filter.Limit {
		return events[:filter.Limit], true, nil
	}
	return events, false, nil
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
			COALESCE(SUM(CASE WHEN wh.success = true THEN 1 ELSE 0 END), 0) as successful_deliveries,
			COALESCE(SUM(CASE WHEN wh.success = false THEN 1 ELSE 0 END), 0) as failed_deliveries,
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
