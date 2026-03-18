package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
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
		event.ExpiresAt = time.Now().Add(time.Duration(event.TTL) * time.Second)
	}

	query := `
		INSERT INTO event_records (
			id, tenant_id, namespace, event, payload, ttl, metadata, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.conn.ExecContext(ctx, query,
		event.ID,
		event.TenantID,
		event.Namespace,
		event.Event,
		event.Payload,
		event.TTL,
		metadataJSON,
		event.CreatedAt,
		event.ExpiresAt,
	)
	return storage.Error(err)
}

// GetEventByID gets an event record by ID within a tenant
func (r *Repository) GetEventByID(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID) (*EventRecord, error) {
	query := `
		SELECT id, tenant_id, namespace, event, payload, ttl, metadata, created_at, expires_at
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

// ListEventReports gets event records in descending order by creation time.
// When namespace is empty, returns reports across all namespaces within the tenant.
func (r *Repository) ListEventReports(ctx context.Context, tenantID uuid.UUID, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	// Always filter by tenant
	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if namespace != "" {
		conditions = append(conditions, fmt.Sprintf("namespace = $%d", argIdx))
		args = append(args, namespace)
		argIdx++
	}

	eventArgIdx := argIdx
	args = append(args, eventName)
	argIdx++

	whereClause := strings.Join(conditions, " AND ")

	// Build base query
	baseQuery := fmt.Sprintf(`
		SELECT
			id, tenant_id, namespace, event, payload, ttl, metadata, created_at, expires_at
		FROM event_records
		WHERE %s
		  AND ($%d IS NULL OR event = $%d)
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, eventArgIdx, eventArgIdx, argIdx, argIdx+1)

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM event_records
		WHERE %s
		  AND ($%d IS NULL OR event = $%d)
	`, whereClause, eventArgIdx, eventArgIdx)

	queryArgs := append(args, limit, offset)

	// Execute main query
	var eventRows []EventRecord
	err := r.conn.SelectContext(ctx, &eventRows, baseQuery, queryArgs...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Get total count
	var totalCount int
	err = r.conn.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Convert to EventReportWithStats format
	var events []*EventReportWithStats
	for _, row := range eventRows {
		event := &EventReportWithStats{
			EventRecord: row,
			// Delivery stats can be loaded separately if needed
			WebhookCount:         0,
			SuccessfulDeliveries: 0,
			FailedDeliveries:     0,
			PendingDeliveries:    0,
		}

		events = append(events, event)
	}

	return events, totalCount, nil
}

// ListEventReportsWithStats retrieves event records enriched with delivery statistics.
// When namespace is empty, returns reports across all namespaces within the tenant.
func (r *Repository) ListEventReportsWithStats(ctx context.Context, tenantID uuid.UUID, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error) {
	var conditions []string
	var args []any
	argIdx := 1

	// Always filter by tenant
	conditions = append(conditions, fmt.Sprintf("er.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if namespace != "" {
		conditions = append(conditions, fmt.Sprintf("er.namespace = $%d", argIdx))
		args = append(args, namespace)
		argIdx++
	}

	eventArgIdx := argIdx
	args = append(args, eventName)
	argIdx++

	whereClause := strings.Join(conditions, " AND ")

	// Build base query with delivery stats from health events
	baseQuery := fmt.Sprintf(`
		SELECT
			er.id, er.tenant_id, er.namespace, er.event, er.payload, er.ttl, er.metadata, er.created_at, er.expires_at,
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
		WHERE %s
		  AND ($%d::text IS NULL OR er.event = $%d::text)
		ORDER BY er.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, eventArgIdx, eventArgIdx, argIdx, argIdx+1)

	// Build count query — use column names without alias
	var countConditions []string
	countConditions = append(countConditions, fmt.Sprintf("tenant_id = $1"))
	countArgIdx := 2
	if namespace != "" {
		countConditions = append(countConditions, fmt.Sprintf("namespace = $%d", countArgIdx))
		countArgIdx++
	}
	countWhereClause := strings.Join(countConditions, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM event_records
		WHERE %s
		  AND ($%d::text IS NULL OR event = $%d::text)
	`, countWhereClause, eventArgIdx, eventArgIdx)

	queryArgs := append(args, limit, offset)

	// Execute main query
	var events []*EventReportWithStats
	err := r.conn.SelectContext(ctx, &events, baseQuery, queryArgs...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Get total count
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
