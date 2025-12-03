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
// Generates UUID v4 for event.ID if not provided and sets created_at to current time.
// Calculates expires_at based on TTL (time-to-live) seconds from creation time.
// Marshals metadata map to JSON for database storage in JSONB column.
func (r *Repository) StoreEvent(ctx context.Context, event *EventRecord) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}
	if event.ExpiresAt.IsZero() {
		event.ExpiresAt = time.Now().Add(time.Duration(event.TTL) * time.Second)
	}

	query := `
		INSERT INTO event_records (
			id, namespace, event, payload, ttl, metadata, created_at, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		event.ID,
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

// GetEventByID gets an event record by ID
func (r *Repository) GetEventByID(ctx context.Context, eventID uuid.UUID) (*EventRecord, error) {
	query := `
		SELECT id, namespace, event, payload, ttl, metadata, created_at, expires_at
		FROM event_records
		WHERE id = $1
	`

	var eventRow EventRecord

	err := r.db.GetContext(ctx, &eventRow, query, eventID)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &eventRow, nil
}

// ListEventReports gets event records in descending order by creation time
func (r *Repository) ListEventReports(ctx context.Context, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error) {
	// Build base query
	baseQuery := `
		SELECT 
			id, namespace, event, payload, ttl, metadata, created_at, expires_at
		FROM event_records 
		WHERE namespace = $1
		  AND ($2 IS NULL OR event = $2)
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	countQuery := `
		SELECT COUNT(*) 
		FROM event_records 
		WHERE namespace = $1
		  AND ($2 IS NULL OR event = $2)
	`

	// Execute main query
	var eventRows []EventRecord

	err := r.db.SelectContext(ctx, &eventRows, baseQuery, namespace, eventName, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Get total count
	var totalCount int
	err = r.db.GetContext(ctx, &totalCount, countQuery, namespace, eventName)
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
// Joins event_records with webhook_deliveries and webhook_health_events to calculate:
// - webhook_count: number of unique webhooks that received the event
// - successful/failed/pending delivery counts based on health event outcomes
// Supports optional event name filtering and pagination. Returns total count for UI pagination.
func (r *Repository) ListEventReportsWithStats(ctx context.Context, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error) {
	// Build base query with delivery stats from health events
	baseQuery := `
		SELECT 
			er.id, er.namespace, er.event, er.payload, er.ttl, er.metadata, er.created_at, er.expires_at,
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
		WHERE er.namespace = $1
		  AND ($2::text IS NULL OR er.event = $2::text)
		ORDER BY er.created_at DESC
		LIMIT $3 OFFSET $4
	`

	countQuery := `
		SELECT COUNT(*) 
		FROM event_records 
		WHERE namespace = $1
		  AND ($2::text IS NULL OR event = $2::text)
	`

	// Execute main query
	var events []*EventReportWithStats
	err := r.db.SelectContext(ctx, &events, baseQuery, namespace, eventName, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Get total count
	var totalCount int
	err = r.db.GetContext(ctx, &totalCount, countQuery, namespace, eventName)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return events, totalCount, nil
}

// GetEventDeliveryStats gets delivery statistics for a specific event using health events
func (r *Repository) GetEventDeliveryStats(ctx context.Context, eventID uuid.UUID) (int32, int32, int32, int32, error) {
	query := `
		SELECT 
			COUNT(DISTINCT wd.webhook_id) as webhook_count,
			SUM(CASE WHEN wh.success = true THEN 1 ELSE 0 END) as successful_deliveries,
			SUM(CASE WHEN wh.success = false THEN 1 ELSE 0 END) as failed_deliveries,
			COUNT(CASE WHEN wd.status IN ('pending', 'sending', 'retrying') THEN 1 END) as pending_deliveries
		FROM webhook_deliveries wd
		LEFT JOIN webhook_health_events wh ON wd.id = wh.delivery_id
		WHERE wd.event_id = $1
	`

	var result struct {
		WebhookCount         int32 `db:"webhook_count"`
		SuccessfulDeliveries int32 `db:"successful_deliveries"`
		FailedDeliveries     int32 `db:"failed_deliveries"`
		PendingDeliveries    int32 `db:"pending_deliveries"`
	}

	err := r.db.GetContext(ctx, &result, query, eventID)
	if err != nil {
		return 0, 0, 0, 0, storage.Error(err)
	}

	return result.WebhookCount, result.SuccessfulDeliveries, result.FailedDeliveries, result.PendingDeliveries, nil
}
