package store

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles webhook registration storage
type Repository struct {
	db *pgxpool.Pool
}

// BeginTx starts a new transaction
func (r *Repository) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

// StoreEventTx stores an event record within a transaction
func (r *Repository) StoreEventTx(ctx context.Context, tx pgx.Tx, event *EventRecord) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
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

	_, err = tx.Exec(ctx, query,
		event.ID,
		event.Namespace,
		event.Event,
		event.Payload,
		event.TTL,
		metadataJSON,
		event.CreatedAt,
		event.ExpiresAt,
	)
	return err
}

// GetWebhooksByEventTx returns all active webhooks for a namespace/event within a transaction
func (r *Repository) GetWebhooksByEventTx(ctx context.Context, tx pgx.Tx, namespace, event string) ([]*WebhookRegistration, error) {
	query := `
		       SELECT id, namespace, events, url, headers, timeout, active, description, health, created_at, updated_at
		       FROM webhook_registrations 
		       WHERE namespace = $1 AND active = true AND events::jsonb ? $2
	       `

	rows, err := tx.Query(ctx, query, namespace, event)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*WebhookRegistration
	for rows.Next() {
		var wh WebhookRegistration
		var headersJSON []byte
		var eventsJSON []byte

		err := rows.Scan(
			&wh.ID,
			&wh.Namespace,
			&eventsJSON,
			&wh.URL,
			&headersJSON,
			&wh.Timeout,
			&wh.Active,
			&wh.Description,
			&wh.Health,
			&wh.CreatedAt,
			&wh.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(headersJSON, &wh.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}

		if err := json.Unmarshal(eventsJSON, &wh.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}

		webhooks = append(webhooks, &wh)
	}

	return webhooks, nil
}

// CreateDeliveryTx creates a webhook delivery record within a transaction
func (r *Repository) CreateDeliveryTx(ctx context.Context, tx pgx.Tx, delivery *WebhookDelivery) error {
	delivery.ID = uuid.New().String()
	delivery.CreatedAt = time.Now()
	delivery.Status = StatusPending

	query := `
		       INSERT INTO webhook_deliveries (
			       id, webhook_id, event_id, status, attempt_count, max_attempts, 
			       created_at, expires_at, response_code, response_body, error_message
		       ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	       `

	_, err := tx.Exec(ctx, query,
		delivery.ID,
		delivery.WebhookID,
		delivery.EventID,
		delivery.Status,
		delivery.AttemptCount,
		delivery.MaxAttempts,
		delivery.CreatedAt,
		delivery.ExpiresAt,
		delivery.ResponseCode,
		delivery.ResponseBody,
		delivery.ErrorMessage,
	)
	return err
}

// NewRepository creates a new webhook repository
func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// RegisterWebhook stores a new webhook registration
func (r *Repository) RegisterWebhook(ctx context.Context, registration *WebhookRegistration) error {
	registration.ID = uuid.New().String()
	registration.CreatedAt = time.Now()
	registration.UpdatedAt = time.Now()
	registration.Health = HealthUnknown // New webhooks start with unknown health

	query := `
		INSERT INTO webhook_registrations (
			id, namespace, events, url, headers, timeout, active, description, health, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	headersJSON, err := json.Marshal(registration.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	eventsJSON, err := json.Marshal(registration.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	_, err = r.db.Exec(ctx, query,
		registration.ID,
		registration.Namespace,
		eventsJSON,
		registration.URL,
		headersJSON,
		registration.Timeout,
		registration.Active,
		registration.Description,
		registration.Health,
		registration.CreatedAt,
		registration.UpdatedAt,
	)
	return err
}

// UnregisterWebhook removes a webhook registration
func (r *Repository) UnregisterWebhook(ctx context.Context, webhookID string) error {
	query := `DELETE FROM webhook_registrations WHERE id = $1`
	_, err := r.db.Exec(ctx, query, webhookID)
	return err
}

// GetWebhooksByEvent returns all active webhooks for a namespace/event
func (r *Repository) GetWebhooksByEvent(ctx context.Context, namespace, event string) ([]*WebhookRegistration, error) {
	query := `
		SELECT id, namespace, events, url, headers, timeout, active, description, health, created_at, updated_at
		FROM webhook_registrations 
		WHERE namespace = $1 AND active = true AND events::jsonb ? $2
	`

	rows, err := r.db.Query(ctx, query, namespace, event)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*WebhookRegistration
	for rows.Next() {
		var wh WebhookRegistration
		var headersJSON []byte
		var eventsJSON []byte

		err := rows.Scan(
			&wh.ID,
			&wh.Namespace,
			&eventsJSON,
			&wh.URL,
			&headersJSON,
			&wh.Timeout,
			&wh.Active,
			&wh.Description,
			&wh.Health,
			&wh.CreatedAt,
			&wh.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(headersJSON, &wh.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}

		if err := json.Unmarshal(eventsJSON, &wh.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}

		webhooks = append(webhooks, &wh)
	}

	return webhooks, nil
}

// ListWebhooks returns webhooks for a namespace
func (r *Repository) ListWebhooks(ctx context.Context, namespace string, activeOnly bool) ([]*WebhookRegistration, error) {
	query := `
		SELECT id, namespace, events, url, headers, timeout, active, description, health, created_at, updated_at
		FROM webhook_registrations 
		WHERE namespace = $1
	`
	args := []interface{}{namespace}

	if activeOnly {
		query += ` AND active = true`
	}

	query += ` ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*WebhookRegistration
	for rows.Next() {
		var wh WebhookRegistration
		var headersJSON []byte
		var eventsJSON []byte

		err := rows.Scan(
			&wh.ID,
			&wh.Namespace,
			&eventsJSON,
			&wh.URL,
			&headersJSON,
			&wh.Timeout,
			&wh.Active,
			&wh.Description,
			&wh.Health,
			&wh.CreatedAt,
			&wh.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(headersJSON, &wh.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}

		if err := json.Unmarshal(eventsJSON, &wh.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}

		webhooks = append(webhooks, &wh)
	}

	return webhooks, nil
}

// StoreEvent stores an event record
func (r *Repository) StoreEvent(ctx context.Context, event *EventRecord) error {
	if event.ID == "" {
		event.ID = uuid.New().String()
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

	_, err = r.db.Exec(ctx, query,
		event.ID,
		event.Namespace,
		event.Event,
		event.Payload,
		event.TTL,
		metadataJSON,
		event.CreatedAt,
		event.ExpiresAt,
	)
	return err
}

// CreateDelivery creates a webhook delivery record
func (r *Repository) CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error {
	delivery.ID = uuid.New().String()
	delivery.CreatedAt = time.Now()
	delivery.Status = StatusPending

	query := `
		INSERT INTO webhook_deliveries (
			id, webhook_id, event_id, status, attempt_count, max_attempts, 
			created_at, expires_at, response_code, response_body, error_message
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Exec(ctx, query,
		delivery.ID,
		delivery.WebhookID,
		delivery.EventID,
		delivery.Status,
		delivery.AttemptCount,
		delivery.MaxAttempts,
		delivery.CreatedAt,
		delivery.ExpiresAt,
		delivery.ResponseCode,
		delivery.ResponseBody,
		delivery.ErrorMessage,
	)
	return err
}

// UpdateDeliveryStatus updates the status of a webhook delivery
func (r *Repository) UpdateDeliveryStatus(ctx context.Context, deliveryID string, status WebhookDeliveryStatus, responseCode int, responseBody, errorMessage string) error {
	now := time.Now()
	query := `
		UPDATE webhook_deliveries 
		SET status = $2, last_attempted_at = $3, response_code = $4, response_body = $5, error_message = $6,
		    attempt_count = attempt_count + 1
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, deliveryID, status, now, responseCode, responseBody, errorMessage)
	return err
}

// GetDeliveriesByWebhook returns deliveries for a specific webhook
func (r *Repository) GetDeliveriesByWebhook(ctx context.Context, webhookID string) ([]*WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event_id, status, attempt_count, max_attempts, 
		       created_at, last_attempted_at, next_retry_at, expires_at,
		       response_code, response_body, error_message
		FROM webhook_deliveries 
		WHERE webhook_id = $1 
		ORDER BY created_at DESC
	`

	return r.getDeliveries(ctx, query, webhookID)
}

// GetDeliveriesByEvent returns deliveries for a specific event
func (r *Repository) GetDeliveriesByEvent(ctx context.Context, eventID string) ([]*WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event_id, status, attempt_count, max_attempts, 
		       created_at, last_attempted_at, next_retry_at, expires_at,
		       response_code, response_body, error_message
		FROM webhook_deliveries 
		WHERE event_id = $1 
		ORDER BY created_at DESC
	`

	return r.getDeliveries(ctx, query, eventID)
}

func (r *Repository) getDeliveries(ctx context.Context, query string, arg interface{}) ([]*WebhookDelivery, error) {
	rows, err := r.db.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery

		err := rows.Scan(
			&d.ID,
			&d.WebhookID,
			&d.EventID,
			&d.Status,
			&d.AttemptCount,
			&d.MaxAttempts,
			&d.CreatedAt,
			&d.LastAttemptedAt,
			&d.NextRetryAt,
			&d.ExpiresAt,
			&d.ResponseCode,
			&d.ResponseBody,
			&d.ErrorMessage,
		)
		if err != nil {
			return nil, err
		}

		deliveries = append(deliveries, &d)
	}

	return deliveries, nil
}

// Ensure we can store map[string]string as JSON in the database
func (h HeadersMap) Value() (driver.Value, error) {
	return json.Marshal(h)
}

func (h *HeadersMap) Scan(value interface{}) error {
	if value == nil {
		*h = make(map[string]string)
		return nil
	}
	return json.Unmarshal(value.([]byte), h)
}

type HeadersMap map[string]string

// GetWebhookByID gets a webhook by ID and namespace
func (r *Repository) GetWebhookByID(ctx context.Context, webhookID, namespace string) (*WebhookRegistration, error) {
	query := `
		SELECT id, namespace, events, url, headers, timeout, active, description, health, created_at, updated_at
		FROM webhook_registrations 
		WHERE id = $1 AND namespace = $2
	`

	var wh WebhookRegistration
	var headersJSON []byte
	var eventsJSON []byte

	err := r.db.QueryRow(ctx, query, webhookID, namespace).Scan(
		&wh.ID,
		&wh.Namespace,
		&eventsJSON,
		&wh.URL,
		&headersJSON,
		&wh.Timeout,
		&wh.Active,
		&wh.Description,
		&wh.Health,
		&wh.CreatedAt,
		&wh.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(headersJSON, &wh.Headers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
	}

	if err := json.Unmarshal(eventsJSON, &wh.Events); err != nil {
		return nil, fmt.Errorf("failed to unmarshal events: %w", err)
	}

	return &wh, nil
}

// GetWebhooksByNamespace gets webhooks by namespace
func (r *Repository) GetWebhooksByNamespace(ctx context.Context, namespace string, activeOnly bool) ([]*WebhookRegistration, error) {
	return r.ListWebhooks(ctx, namespace, activeOnly)
}

// UpdateWebhook updates a webhook registration
func (r *Repository) UpdateWebhook(ctx context.Context, webhook *WebhookRegistration) error {
	webhook.UpdatedAt = time.Now()

	query := `
		UPDATE webhook_registrations 
		SET events = $2, url = $3, headers = $4, timeout = $5, active = $6, 
		    description = $7, updated_at = $8
		WHERE id = $1 AND namespace = $9
	`

	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	eventsJSON, err := json.Marshal(webhook.Events)
	if err != nil {
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	_, err = r.db.Exec(ctx, query,
		webhook.ID,
		eventsJSON,
		webhook.URL,
		headersJSON,
		webhook.Timeout,
		webhook.Active,
		webhook.Description,
		webhook.UpdatedAt,
		webhook.Namespace,
	)
	return err
}

// GetDeliveryByID gets a delivery by ID and namespace
func (r *Repository) GetDeliveryByID(ctx context.Context, deliveryID, namespace string) (*WebhookDelivery, error) {
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.status, wd.attempt_count, wd.max_attempts, 
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.id = $1 AND wr.namespace = $2
	`

	var d WebhookDelivery
	err := r.db.QueryRow(ctx, query, deliveryID, namespace).Scan(
		&d.ID,
		&d.WebhookID,
		&d.EventID,
		&d.Status,
		&d.AttemptCount,
		&d.MaxAttempts,
		&d.CreatedAt,
		&d.LastAttemptedAt,
		&d.NextRetryAt,
		&d.ExpiresAt,
		&d.ResponseCode,
		&d.ResponseBody,
		&d.ErrorMessage,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	return &d, nil
}

// GetDeliveriesByWebhookID gets deliveries for a webhook with pagination
func (r *Repository) GetDeliveriesByWebhookID(ctx context.Context, webhookID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error) {
	// First get total count
	countQuery := `
		SELECT COUNT(*)
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.webhook_id = $1 AND wr.namespace = $2
	`

	var totalCount int
	err := r.db.QueryRow(ctx, countQuery, webhookID, namespace).Scan(&totalCount)
	if err != nil {
		return nil, 0, err
	}

	// Then get paginated results
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.status, wd.attempt_count, wd.max_attempts, 
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.webhook_id = $1 AND wr.namespace = $2
		ORDER BY wd.created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := r.db.Query(ctx, query, webhookID, namespace, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var deliveries []*WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		err := rows.Scan(
			&d.ID,
			&d.WebhookID,
			&d.EventID,
			&d.Status,
			&d.AttemptCount,
			&d.MaxAttempts,
			&d.CreatedAt,
			&d.LastAttemptedAt,
			&d.NextRetryAt,
			&d.ExpiresAt,
			&d.ResponseCode,
			&d.ResponseBody,
			&d.ErrorMessage,
		)
		if err != nil {
			return nil, 0, err
		}
		deliveries = append(deliveries, &d)
	}

	return deliveries, totalCount, nil
}

// RegisterEvent registers a new event type
func (r *Repository) RegisterEvent(ctx context.Context, event *EventRegistration) error {
	event.ID = uuid.New().String()
	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	query := `
		INSERT INTO event_registrations (
			id, name, description, schema, metadata, active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.Exec(ctx, query,
		event.ID,
		event.Name,
		event.Description,
		event.Schema,
		metadataJSON,
		event.Active,
		event.CreatedAt,
		event.UpdatedAt,
	)
	return err
}

// GetEventByName gets an event registration by name
func (r *Repository) GetEventByName(ctx context.Context, eventName string) (*EventRegistration, error) {
	query := `
		SELECT id, name, description, schema, metadata, active, created_at, updated_at
		FROM event_registrations 
		WHERE name = $1
	`

	var event EventRegistration
	var metadataJSON []byte

	err := r.db.QueryRow(ctx, query, eventName).Scan(
		&event.ID,
		&event.Name,
		&event.Description,
		&event.Schema,
		&metadataJSON,
		&event.Active,
		&event.CreatedAt,
		&event.UpdatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return &event, nil
}

// ListEvents returns all registered events
func (r *Repository) ListEvents(ctx context.Context, activeOnly bool) ([]*EventRegistration, error) {
	query := `
		SELECT id, name, description, schema, metadata, active, created_at, updated_at
		FROM event_registrations
	`
	args := []interface{}{}

	if activeOnly {
		query += ` WHERE active = true`
	}

	query += ` ORDER BY name ASC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*EventRegistration
	for rows.Next() {
		var event EventRegistration
		var metadataJSON []byte

		err := rows.Scan(
			&event.ID,
			&event.Name,
			&event.Description,
			&event.Schema,
			&metadataJSON,
			&event.Active,
			&event.CreatedAt,
			&event.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(metadataJSON, &event.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
		}

		events = append(events, &event)
	}

	return events, nil
}

// UpdateEvent updates an event registration
func (r *Repository) UpdateEvent(ctx context.Context, event *EventRegistration) error {
	event.UpdatedAt = time.Now()

	query := `
		UPDATE event_registrations 
		SET description = $2, schema = $3, metadata = $4, active = $5, updated_at = $6
		WHERE name = $1
	`

	metadataJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = r.db.Exec(ctx, query,
		event.Name,
		event.Description,
		event.Schema,
		metadataJSON,
		event.Active,
		event.UpdatedAt,
	)
	return err
}

// DeleteEvent deletes an event registration
func (r *Repository) DeleteEvent(ctx context.Context, eventName string) error {
	query := `DELETE FROM event_registrations WHERE name = $1`
	_, err := r.db.Exec(ctx, query, eventName)
	return err
}

// RecordWebhookHealthEvent records a health event for time-series tracking
func (r *Repository) RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID string, success bool, responseTime, responseCode int, errorMessage string) error {
	query := `
		INSERT INTO webhook_health_events (webhook_id, delivery_id, success, response_time, response_code, error_message, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`

	_, err := r.db.Exec(ctx, query, webhookID, deliveryID, success, responseTime, responseCode, errorMessage)
	if err != nil {
		return fmt.Errorf("failed to record health event: %w", err)
	}

	return nil
}

// GetWebhookHealthMetrics retrieves current health state for a webhook
func (r *Repository) GetWebhookHealthState(ctx context.Context, webhookID string) (*WebhookHealthMetrics, error) {
	query := `
		SELECT id, webhook_id, consecutive_failures, last_success_at, last_failure_at, 
		       last_event_at, created_at, updated_at
		FROM webhook_health_state
		WHERE webhook_id = $1
	`

	var state WebhookHealthMetrics
	err := r.db.QueryRow(ctx, query, webhookID).Scan(
		&state.ID,
		&state.WebhookID,
		&state.ConsecutiveFailures,
		&state.LastSuccessAt,
		&state.LastFailureAt,
		&state.LastEventAt,
		&state.CreatedAt,
		&state.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &state, nil
}

// GetWebhookHealthSummary gets aggregated health metrics for a time window
func (r *Repository) GetWebhookHealthSummary(ctx context.Context, webhookID string, hours int) (*WebhookHealthSummary, error) {
	// First try to get from pre-computed summaries
	query := `
		SELECT id, webhook_id, window_start, window_end, total_deliveries, successful_deliveries,
		       failed_deliveries, success_rate, avg_response_time, min_response_time,
		       max_response_time, p95_response_time, created_at, updated_at
		FROM webhook_health_summaries
		WHERE webhook_id = $1 
		  AND window_start >= NOW() - INTERVAL '1 hour' * $2
		ORDER BY window_start DESC
		LIMIT 1
	`

	var summary WebhookHealthSummary
	err := r.db.QueryRow(ctx, query, webhookID, hours).Scan(
		&summary.ID,
		&summary.WebhookID,
		&summary.WindowStart,
		&summary.WindowEnd,
		&summary.TotalDeliveries,
		&summary.SuccessfulDeliveries,
		&summary.FailedDeliveries,
		&summary.SuccessRate,
		&summary.AvgResponseTime,
		&summary.MinResponseTime,
		&summary.MaxResponseTime,
		&summary.P95ResponseTime,
		&summary.CreatedAt,
		&summary.UpdatedAt,
	)
	if err == nil {
		return &summary, nil
	}

	// If no pre-computed summary exists, compute on-the-fly
	realTimeQuery := `
		SELECT 
			$1 as webhook_id,
			NOW() - INTERVAL '1 hour' * $2 as window_start,
			NOW() as window_end,
			COUNT(*) as total_deliveries,
			SUM(CASE WHEN success THEN 1 ELSE 0 END) as successful_deliveries,
			SUM(CASE WHEN success THEN 0 ELSE 1 END) as failed_deliveries,
			COALESCE(AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END), 0) as success_rate,
			COALESCE(AVG(response_time), 0)::INTEGER as avg_response_time,
			COALESCE(MIN(response_time), 0) as min_response_time,
			COALESCE(MAX(response_time), 0) as max_response_time,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time), 0)::INTEGER as p95_response_time,
			NOW() as created_at,
			NOW() as updated_at
		FROM webhook_health_events
		WHERE webhook_id = $1 
		  AND timestamp >= NOW() - INTERVAL '1 hour' * $2
	`

	err = r.db.QueryRow(ctx, realTimeQuery, webhookID, hours).Scan(
		&summary.WebhookID,
		&summary.WindowStart,
		&summary.WindowEnd,
		&summary.TotalDeliveries,
		&summary.SuccessfulDeliveries,
		&summary.FailedDeliveries,
		&summary.SuccessRate,
		&summary.AvgResponseTime,
		&summary.MinResponseTime,
		&summary.MaxResponseTime,
		&summary.P95ResponseTime,
		&summary.CreatedAt,
		&summary.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Generate a synthetic ID for the real-time summary
	summary.ID = uuid.New().String()

	return &summary, nil
}

// GetWebhookHealthTimeSeries gets health events over time for analytics
func (r *Repository) GetWebhookHealthTimeSeries(ctx context.Context, webhookID string, hours int, bucketSize string) ([]*WebhookHealthEvent, error) {
	// bucketSize can be "1 minute", "5 minute", "1 hour", "1 day"
	query := `
		SELECT id, webhook_id, delivery_id, success, response_time, response_code, error_message, timestamp
		FROM webhook_health_events
		WHERE webhook_id = $1 
		  AND timestamp >= NOW() - INTERVAL '1 hour' * $2
		ORDER BY timestamp DESC
		LIMIT 1000
	`

	rows, err := r.db.Query(ctx, query, webhookID, hours)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []*WebhookHealthEvent
	for rows.Next() {
		var event WebhookHealthEvent
		err := rows.Scan(
			&event.ID,
			&event.WebhookID,
			&event.DeliveryID,
			&event.Success,
			&event.ResponseTime,
			&event.ResponseCode,
			&event.ErrorMessage,
			&event.Timestamp,
		)
		if err != nil {
			return nil, err
		}
		events = append(events, &event)
	}

	return events, nil
}

// AggregateHealthSummaries runs the aggregation function for health summaries
func (r *Repository) AggregateHealthSummaries(ctx context.Context) (int, error) {
	query := `SELECT aggregate_webhook_health_hourly()`

	var processedCount int
	err := r.db.QueryRow(ctx, query).Scan(&processedCount)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate health summaries: %w", err)
	}

	return processedCount, nil
}

// GetWebhooksByHealth retrieves webhooks filtered by health status
func (r *Repository) GetWebhooksByHealth(ctx context.Context, health WebhookHealth) ([]*WebhookRegistration, error) {
	query := `
		SELECT wr.id, wr.namespace, wr.events, wr.url, wr.headers, wr.timeout, 
		       wr.active, wr.description, wr.health, wr.created_at, wr.updated_at
		FROM webhook_registrations wr
		WHERE wr.health = $1
		ORDER BY wr.created_at DESC
	`

	rows, err := r.db.Query(ctx, query, string(health))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []*WebhookRegistration
	for rows.Next() {
		var webhook WebhookRegistration
		var eventsJSON, headersJSON []byte

		err := rows.Scan(
			&webhook.ID,
			&webhook.Namespace,
			&eventsJSON,
			&webhook.URL,
			&headersJSON,
			&webhook.Timeout,
			&webhook.Active,
			&webhook.Description,
			&webhook.Health,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := json.Unmarshal(eventsJSON, &webhook.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}

		if err := json.Unmarshal(headersJSON, &webhook.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}

		webhooks = append(webhooks, &webhook)
	}

	return webhooks, nil
}

// GetHealthSummary returns a summary of webhook health across all namespaces
func (r *Repository) GetHealthSummary(ctx context.Context) (map[WebhookHealth]int, error) {
	query := `
		SELECT health, COUNT(*) as count
		FROM webhook_registrations
		GROUP BY health
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	summary := make(map[WebhookHealth]int)
	for rows.Next() {
		var health string
		var count int

		err := rows.Scan(&health, &count)
		if err != nil {
			return nil, err
		}

		summary[WebhookHealth(health)] = count
	}

	return summary, nil
}

// GetRetriableDeliveries gets deliveries that can be retried for a webhook
func (r *Repository) GetRetriableDeliveries(ctx context.Context, webhookID, namespace string, force bool) ([]*WebhookDelivery, error) {
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.status, wd.attempt_count, wd.max_attempts, 
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.webhook_id = $1 AND wr.namespace = $2
	`
	args := []interface{}{webhookID, namespace}

	if !force {
		query += ` AND wd.status IN ('failed', 'pending', 'retrying')`
	}

	query += ` ORDER BY wd.created_at DESC`

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deliveries []*WebhookDelivery
	for rows.Next() {
		var d WebhookDelivery
		err := rows.Scan(
			&d.ID,
			&d.WebhookID,
			&d.EventID,
			&d.Status,
			&d.AttemptCount,
			&d.MaxAttempts,
			&d.CreatedAt,
			&d.LastAttemptedAt,
			&d.NextRetryAt,
			&d.ExpiresAt,
			&d.ResponseCode,
			&d.ResponseBody,
			&d.ErrorMessage,
		)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, &d)
	}

	return deliveries, nil
}

// ResetDeliveryForRetry resets a delivery status to pending for retry
func (r *Repository) ResetDeliveryForRetry(ctx context.Context, deliveryID string) error {
	query := `
		UPDATE webhook_deliveries 
		SET status = 'pending', 
		    last_attempted_at = NULL, 
		    next_retry_at = NULL,
		    response_code = 0,
		    response_body = '',
		    error_message = ''
		WHERE id = $1
	`

	_, err := r.db.Exec(ctx, query, deliveryID)
	return err
}
