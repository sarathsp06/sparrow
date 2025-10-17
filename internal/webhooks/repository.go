package webhooks

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository handles webhook registration storage
type Repository struct {
	db *pgxpool.Pool
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

	query := `
		INSERT INTO webhook_registrations (
			id, namespace, events, url, headers, timeout, active, description, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
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
		SELECT id, namespace, events, url, headers, timeout, active, description, created_at, updated_at
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
		SELECT id, namespace, events, url, headers, timeout, active, description, created_at, updated_at
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
	event.ID = uuid.New().String()
	event.CreatedAt = time.Now()
	event.ExpiresAt = time.Now().Add(time.Duration(event.TTL) * time.Second)

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
