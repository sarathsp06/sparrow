package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// DeliveryRepository defines operations for the webhook delivery lifecycle:
// creation, status transitions, retries, filtering, and attempt history.
type DeliveryRepository interface {
	CreateDelivery(ctx context.Context, tenantID uuid.UUID, delivery *WebhookDelivery) error
	BatchCreateDeliveries(ctx context.Context, tenantID uuid.UUID, deliveries []*WebhookDelivery) error
	UpdateDeliveryStatus(ctx context.Context, deliveryID uuid.UUID, status WebhookDeliveryStatus, responseCode int, responseBody, errorMessage, errorCategory string) error
	RecordDeliveryAttempt(ctx context.Context, attempt DeliveryAttempt) error
	HoldDelivery(ctx context.Context, deliveryID uuid.UUID, reason string) error
	GetDeliveryByID(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID, consumer string) (*WebhookDelivery, error)
	ListDeliveriesFiltered(ctx context.Context, tenantID uuid.UUID, filter DeliveryFilter) ([]*WebhookDelivery, bool, error)
	CountDeliveries(ctx context.Context, tenantID uuid.UUID, filter DeliveryFilter) (int, error)
	GetRetriableDeliveries(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string, force bool) ([]*WebhookDelivery, error)
	ResetDeliveryForRetry(ctx context.Context, deliveryID uuid.UUID) error
	ResetDeliveriesForRetry(ctx context.Context, deliveryIDs []uuid.UUID) error
	UpdateDeliveryTemplateError(ctx context.Context, deliveryID uuid.UUID, msg string) error
	DeleteDeliveryByID(ctx context.Context, deliveryID uuid.UUID) error
	GetDeliveryAttempts(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID) ([]*WebhookHealthEvent, error)
}

// deliveryColumns is the canonical SELECT column list for webhook_deliveries (aliased as wd).
// Used by all delivery query functions to avoid repeating the same 16-column list.
const deliveryColumns = `wd.id, wd.webhook_id, wd.event_id, wd.subscription_id, wd.status, wd.attempt_count, wd.max_attempts,
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message, wd.request_body, wd.error_category,
		       COALESCE(wd.template_error, '') AS template_error`

// CreateDelivery creates a new webhook delivery record for tracking delivery attempts.
// tenantID is accepted for interface consistency; the delivery is implicitly tenant-scoped via webhook_id.
func (r *Repository) CreateDelivery(ctx context.Context, tenantID uuid.UUID, delivery *WebhookDelivery) error {
	query := `
		INSERT INTO webhook_deliveries (id, webhook_id, event_id, subscription_id, status, attempt_count, max_attempts, expires_at, response_code, response_body, error_message, request_body)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.conn.ExecContext(ctx, query,
		delivery.ID,
		delivery.WebhookID,
		delivery.EventID,
		delivery.SubscriptionID,
		delivery.Status,
		delivery.AttemptCount,
		delivery.MaxAttempts,
		delivery.ExpiresAt,
		delivery.ResponseCode,
		delivery.ResponseBody,
		delivery.ErrorMessage,
		delivery.RequestBody,
	)
	return storage.Error(err)
}

// BatchCreateDeliveries inserts multiple webhook delivery records in a single
// multi-row INSERT statement. This is the batch counterpart of CreateDelivery
// and is used by the fan-out loop to avoid N sequential round-trips.
// tenantID is accepted for interface consistency.
func (r *Repository) BatchCreateDeliveries(ctx context.Context, tenantID uuid.UUID, deliveries []*WebhookDelivery) error {
	if len(deliveries) == 0 {
		return nil
	}

	// Single delivery — fall through to the simple path to avoid builder overhead.
	if len(deliveries) == 1 {
		return r.CreateDelivery(ctx, tenantID, deliveries[0])
	}

	// Build a multi-row INSERT:
	//   INSERT INTO webhook_deliveries (id, webhook_id, event_id, subscription_id, status, attempt_count, max_attempts, expires_at, response_code, response_body, error_message, request_body)
	//   VALUES ($1,$2,...,$12), ($13,$14,...,$24), ...
	const cols = 12
	valueGroups := make([]string, 0, len(deliveries))
	args := make([]any, 0, len(deliveries)*cols)

	for i, d := range deliveries {
		base := i * cols
		placeholders := make([]string, cols)
		for j := 0; j < cols; j++ {
			placeholders[j] = fmt.Sprintf("$%d", base+j+1)
		}
		valueGroups = append(valueGroups, "("+strings.Join(placeholders, ",")+")")

		args = append(args,
			d.ID,
			d.WebhookID,
			d.EventID,
			d.SubscriptionID,
			d.Status,
			d.AttemptCount,
			d.MaxAttempts,
			d.ExpiresAt,
			d.ResponseCode,
			d.ResponseBody,
			d.ErrorMessage,
			d.RequestBody,
		)
	}

	query := `INSERT INTO webhook_deliveries (id, webhook_id, event_id, subscription_id, status, attempt_count, max_attempts, expires_at, response_code, response_body, error_message, request_body)
		VALUES ` + strings.Join(valueGroups, ",")

	_, err := r.conn.ExecContext(ctx, query, args...)
	return storage.Error(err)
}

// DeleteDeliveryByID removes a delivery record by its ID. Used as a compensation
// action when a River job insert fails after the delivery was already created.
func (r *Repository) DeleteDeliveryByID(ctx context.Context, deliveryID uuid.UUID) error {
	query := `DELETE FROM webhook_deliveries WHERE id = $1`
	_, err := r.conn.ExecContext(ctx, query, deliveryID)
	return storage.Error(err)
}

// UpdateDeliveryStatus records the outcome of a webhook delivery attempt.
func (r *Repository) UpdateDeliveryStatus(ctx context.Context, deliveryID uuid.UUID, status WebhookDeliveryStatus, responseCode int, responseBody, errorMessage, errorCategory string) error {
	now := time.Now()
	// Each call that records an attempt outcome (success, terminal failure, or a
	// failed attempt that will be retried) counts one attempt. Expiry is not an
	// attempt: no request was made.
	attemptIncrement := 0
	if status == StatusFailed || status == StatusSuccess || status == StatusRetrying {
		attemptIncrement = 1
	}

	query := `
		UPDATE webhook_deliveries
		SET status = $2, last_attempted_at = $3, response_code = $4, response_body = $5, error_message = $6,
		    attempt_count = attempt_count + $7::integer, error_category = $8
		WHERE id = $1
	`

	_, err := r.conn.ExecContext(ctx, query, deliveryID, status, now, responseCode, responseBody, errorMessage, attemptIncrement, errorCategory)
	return storage.Error(err)
}

// HoldDelivery marks a delivery paused without recording an attempt: its
// webhook or subscription was paused before it could be sent. Like a delivery
// fanned out while paused, it waits to be retried.
func (r *Repository) HoldDelivery(ctx context.Context, deliveryID uuid.UUID, reason string) error {
	_, err := r.conn.ExecContext(ctx,
		`UPDATE webhook_deliveries SET status = $2, error_message = $3 WHERE id = $1`,
		deliveryID, StatusPaused, reason)
	return storage.Error(err)
}

// GetDeliveryByID gets a delivery by ID, optionally filtered by consumer, within a tenant.
// When consumer is empty, looks up by delivery ID alone (still tenant-scoped).
func (r *Repository) GetDeliveryByID(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID, consumer string) (*WebhookDelivery, error) {
	var query string
	var args []any

	if consumer != "" {
		query = fmt.Sprintf(`
			SELECT %s
			FROM webhook_deliveries wd
			JOIN webhook_registrations wr ON wd.webhook_id = wr.id
			WHERE wd.id = $1 AND wr.tenant_id = $2 AND wr.consumer = $3
		`, deliveryColumns)
		args = []any{deliveryID, tenantID, consumer}
	} else {
		query = fmt.Sprintf(`
			SELECT %s
			FROM webhook_deliveries wd
			JOIN webhook_registrations wr ON wd.webhook_id = wr.id
			WHERE wd.id = $1 AND wr.tenant_id = $2
		`, deliveryColumns)
		args = []any{deliveryID, tenantID}
	}

	var d WebhookDelivery
	err := r.conn.GetContext(ctx, &d, query, args...)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}

	return &d, nil
}

// deliveryFilterFrom is the FROM and WHERE of every DeliveryFilter query
// (count, page, retry snapshot), with the filter in deliveryFilterArgs order
// as $1..$10. Unset filter fields bind NULL, so each ($N IS NULL OR ...)
// guard becomes a no-op: no dynamic SQL.
//
// Status binds as a webhook_delivery_status parameter. Comparing status::text
// hid the column from every index, and casting a text parameter to the enum
// is not constant-folded, so the planner could not match the partial
// idx_webhook_deliveries_unsuccessful (measured on 6M deliveries: a
// failed-only retry snapshot 4.8s with the cast, 154ms with the enum
// parameter; paused counts 185ms -> 0.05ms). The REST layer validates the
// value, so an unknown status is a 422, not a database error.
//
// The event type filter is an EXISTS behind its guard, not a join on
// event_records: measured on 300k deliveries with pgx's cached (generic)
// plans, a join made every count 5-20x slower, filtered or not, while the
// EXISTS costs nothing when the filter is unset.
const deliveryFilterFrom = `
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wr.tenant_id = $1
		  AND ($2::text IS NULL OR wr.consumer = $2)` + deliveryFilterConds

// deliveryFilterConds is the delivery-side half of deliveryFilterFrom
// ($3..$10), shared with the per-webhook page in ListDeliveriesFiltered.
const deliveryFilterConds = `
		  AND ($3::uuid IS NULL OR wd.webhook_id = $3)
		  AND ($4::uuid IS NULL OR wd.event_id = $4)
		  AND ($5::webhook_delivery_status IS NULL OR wd.status = $5)
		  AND ($6::text IS NULL OR wd.error_category = $6)
		  AND ($7::uuid IS NULL OR wd.subscription_id = $7)
		  AND ($8::timestamptz IS NULL OR wd.created_at >= $8)
		  AND ($9::timestamptz IS NULL OR wd.created_at <= $9)
		  AND ($10::text IS NULL OR EXISTS (
		        SELECT 1 FROM event_records er WHERE er.id = wd.event_id AND er.event = $10))`

// deliveryFilterArgs binds a DeliveryFilter to deliveryFilterFrom's $1..$10.
func deliveryFilterArgs(tenantID uuid.UUID, filter DeliveryFilter) []any {
	var ns any
	if filter.Consumer != "" {
		ns = filter.Consumer
	}
	return []any{tenantID, ns, filter.WebhookID, filter.EventID, filter.Status, filter.ErrorCategory, filter.SubscriptionID, filter.CreatedAfter, filter.CreatedBefore, filter.EventName}
}

// CountDeliveries counts the deliveries matching filter (its cursor, limit
// and offset are ignored). For the few places that report a number, e.g.
// the deliveries held by a paused webhook; lists use has_more instead.
func (r *Repository) CountDeliveries(ctx context.Context, tenantID uuid.UUID, filter DeliveryFilter) (int, error) {
	var n int
	if err := r.conn.GetContext(ctx, &n, "SELECT COUNT(*) "+deliveryFilterFrom, deliveryFilterArgs(tenantID, filter)...); err != nil {
		return 0, storage.Error(err)
	}
	return n, nil
}

// ListDeliveriesFiltered returns one page of the deliveries matching filter,
// newest first, and whether more follow. There is no total: counting every
// match cost more than the page itself on a large table, on every load.
// filter.After continues after a row (keyset, constant cost at any depth).
func (r *Repository) ListDeliveriesFiltered(ctx context.Context, tenantID uuid.UUID, filter DeliveryFilter) ([]*WebhookDelivery, bool, error) {
	afterAt, afterID := cursorArgs(filter.After)
	args := append(deliveryFilterArgs(tenantID, filter), afterAt, afterID, filter.Limit+1)

	// The page is cut first, then joined to its event names, so the lookup
	// runs once per returned row.
	query := fmt.Sprintf(deliveryPageQuery, fmt.Sprintf(`
			SELECT %s, wr.consumer
			%s
			  AND wd.created_at <= $11 AND (wd.created_at < $11 OR wd.id < $12)
			ORDER BY wd.created_at DESC, wd.id DESC
			LIMIT $13`, deliveryColumns, deliveryFilterFrom))
	if filter.Consumer != "" && filter.WebhookID == nil && filter.EventID == nil {
		// A consumer's deliveries have no index in (consumer, created_at)
		// order: consumer is on the webhook. Walking all deliveries newest
		// first took 1.56s for a consumer with few of them on 6M rows, and
		// the opposite plan was slow for the busiest one. Taking the newest
		// page of each of the consumer's webhooks (webhook_id, created_at
		// index) and merging them reads at most webhooks x page rows:
		// 0.5-3.5ms for every consumer shape measured.
		query = fmt.Sprintf(deliveryPageQuery, fmt.Sprintf(`
			SELECT d.*
			FROM webhook_registrations wr
			CROSS JOIN LATERAL (
				SELECT %s, wr.consumer
				FROM webhook_deliveries wd
				WHERE wd.webhook_id = wr.id%s
				  AND wd.created_at <= $11 AND (wd.created_at < $11 OR wd.id < $12)
				ORDER BY wd.created_at DESC, wd.id DESC
				LIMIT $13
			) d
			WHERE wr.tenant_id = $1 AND wr.consumer = $2
			ORDER BY d.created_at DESC, d.id DESC
			LIMIT $13`, deliveryColumns, deliveryFilterConds))
	}

	var deliveries []*WebhookDelivery
	if err := r.conn.SelectContext(ctx, &deliveries, query, args...); err != nil {
		return nil, false, storage.Error(err)
	}
	if len(deliveries) > filter.Limit {
		return deliveries[:filter.Limit], true, nil
	}
	return deliveries, false, nil
}

// deliveryPageQuery wraps one page of deliveries (%s, ordered and limited)
// with the event name of each.
const deliveryPageQuery = `
		SELECT page.*, er.event AS event_name
		FROM (%s
		) page
		LEFT JOIN event_records er ON er.id = page.event_id
		ORDER BY page.created_at DESC, page.id DESC`

// GetRetriableDeliveries finds webhook deliveries eligible for retry attempts within a tenant.
func (r *Repository) GetRetriableDeliveries(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string, force bool) ([]*WebhookDelivery, error) {
	// Only what a retry needs to re-queue each delivery, never the bodies: a
	// forced retry can cover every delivery of a webhook. subscription_id
	// must stay: without it the worker finds no transform template (sending
	// the payload untransformed, or failing a webhook that requires one).
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.subscription_id, wd.status, wd.max_attempts,
		       wd.created_at, wd.expires_at
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.webhook_id = $1
		  AND wr.tenant_id = $2
		  AND wr.consumer = $3
		  AND ($4 IS TRUE OR wd.status IN ('failed', 'pending', 'retrying', 'paused'))
		ORDER BY wd.created_at DESC
	`

	var deliveries []*WebhookDelivery
	err := r.conn.SelectContext(ctx, &deliveries, query, webhookID, tenantID, consumer, force)
	if err != nil {
		return nil, storage.Error(err)
	}

	return deliveries, nil
}

// UpdateDeliveryTemplateError records the payload transform error for a
// delivery, or clears it when msg is empty.
func (r *Repository) UpdateDeliveryTemplateError(ctx context.Context, deliveryID uuid.UUID, msg string) error {
	var v any
	if msg != "" {
		v = msg
	}
	_, err := r.conn.ExecContext(ctx, `UPDATE webhook_deliveries SET template_error = $2 WHERE id = $1`, deliveryID, v)
	return storage.Error(err)
}

// ResetDeliveryForRetry resets a delivery status to pending for retry
func (r *Repository) ResetDeliveryForRetry(ctx context.Context, deliveryID uuid.UUID) error {
	query := `
		UPDATE webhook_deliveries
		SET status = 'pending',
		    last_attempted_at = NULL,
		    next_retry_at = NULL,
		    response_code = 0,
		    response_body = '',
		    error_message = '',
		    error_category = '',
		    template_error = NULL
		WHERE id = $1
	`

	_, err := r.conn.ExecContext(ctx, query, deliveryID)
	return storage.Error(err)
}

// ResetDeliveriesForRetry is ResetDeliveryForRetry for many deliveries in
// one statement.
func (r *Repository) ResetDeliveriesForRetry(ctx context.Context, deliveryIDs []uuid.UUID) error {
	if len(deliveryIDs) == 0 {
		return nil
	}
	query := `
		UPDATE webhook_deliveries
		SET status = 'pending',
		    last_attempted_at = NULL,
		    next_retry_at = NULL,
		    response_code = 0,
		    response_body = '',
		    error_message = '',
		    error_category = '',
		    template_error = NULL
		WHERE id = ANY($1)
	`

	_, err := r.conn.ExecContext(ctx, query, pq.Array(deliveryIDs))
	return storage.Error(err)
}

// GetDeliveryAttempts retrieves all health events for a specific delivery, ordered by timestamp.
// Each health event represents an individual delivery attempt with response details.
// Filters by tenant_id via a JOIN on webhook_registrations to enforce tenant isolation.
func (r *Repository) GetDeliveryAttempts(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID) ([]*WebhookHealthEvent, error) {
	query := `
		SELECT whe.id, whe.webhook_id, whe.delivery_id, whe.success, whe.response_time, whe.response_code, whe.error_message, whe.error_category, whe.timestamp
		FROM webhook_health_events whe
		JOIN webhook_registrations wr ON wr.id = whe.webhook_id
		WHERE whe.delivery_id = $1
		  AND wr.tenant_id = $2
		ORDER BY whe.timestamp ASC
	`

	var events []*WebhookHealthEvent
	err := r.conn.SelectContext(ctx, &events, query, deliveryID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery attempts: %w", err)
	}

	return events, nil
}
