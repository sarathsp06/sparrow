package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// RegisterWebhook creates a new webhook registration with duplicate prevention.
// Checks for existing webhook with same namespace+URL combination to prevent duplicates.
// Generates a new UUID v4 for the webhook ID and initializes health status as "unknown".
// Sets created_at and updated_at timestamps automatically.
// Returns nil if webhook already exists (idempotent operation) or on successful creation.
func (r *Repository) RegisterWebhook(ctx context.Context, registration *WebhookRegistration) error {
	// Check for existing webhook with same namespace and url
	checkQuery := `SELECT id FROM webhook_registrations WHERE namespace = $1 AND url = $2 LIMIT 1`
	var existingID uuid.UUID
	err := r.db.GetContext(ctx, &existingID, checkQuery, registration.Namespace, registration.URL)
	if err == nil && existingID != uuid.Nil {
		// Already exists, treat as success
		return nil
	} else if err != nil && !storage.IsNotFound(storage.Error(err)) {
		// DB error
		return storage.Error(err)
	}

	registration.ID = uuid.New()
	registration.Health = HealthUnknown // New webhooks start with unknown health

	query := `
		INSERT INTO webhook_registrations (
			id, namespace, url, headers, timeout, active, description, health,
			max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			user_agent, content_type, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
	`

	headersJSON, err := json.Marshal(registration.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	// Set timestamps
	now := time.Now()
	if registration.CreatedAt.IsZero() {
		registration.CreatedAt = now
	}
	registration.UpdatedAt = now

	_, err = r.db.ExecContext(ctx, query,
		registration.ID,
		registration.Namespace,
		registration.URL,
		headersJSON,
		registration.Timeout,
		registration.Active,
		registration.Description,
		registration.Health,
		registration.MaxRetries,
		registration.RetryBackoffSeconds,
		registration.CaptureResponseBody,
		registration.FollowRedirects,
		registration.VerifySSL,
		registration.RequestTimeoutSeconds,
		pq.Array(registration.ExpectedStatusCodes),
		registration.WebhookSecret,
		registration.UserAgent,
		registration.ContentType,
		registration.CreatedAt,
		registration.UpdatedAt,
	)
	return storage.Error(err)
}

// UnregisterWebhook permanently deletes a webhook registration and all associated data.
// This is a hard delete that removes the webhook from the database entirely.
// Related delivery records and health events may be retained based on retention policies.
func (r *Repository) UnregisterWebhook(ctx context.Context, webhookID uuid.UUID) error {
	query := `DELETE FROM webhook_registrations WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, webhookID)
	return storage.Error(err)
}

// ListWebhooks retrieves webhooks for a namespace with optional active status filtering.
// When activeOnly=true, returns only webhooks with active=true (excludes paused webhooks).
// When activeOnly=false, returns all webhooks regardless of active status for management purposes.
// Results are ordered by created_at DESC to show newest webhooks first.
func (r *Repository) ListWebhooks(ctx context.Context, namespace string, activeOnly bool) ([]*WebhookRegistration, error) {
	query := `
		SELECT id, namespace, url, headers, timeout, active, description, health,
		       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
		       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
		       user_agent, content_type, created_at, updated_at
		FROM webhook_registrations 
		WHERE namespace = $1 
		  AND ($2 IS FALSE OR active = true)
		ORDER BY created_at DESC
	`

	var webhooks []*WebhookRegistration
	err := r.db.SelectContext(ctx, &webhooks, query, namespace, activeOnly)
	if err != nil {
		return nil, storage.Error(err)
	}

	return webhooks, nil
}

// GetWebhookByID gets a webhook by ID and namespace
func (r *Repository) GetWebhookByID(ctx context.Context, webhookID uuid.UUID, namespace string) (*WebhookRegistration, error) {
	query := `
		SELECT id, namespace, url, headers, timeout, active, description, health,
		       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
		       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
		       user_agent, content_type, created_at, updated_at
		FROM webhook_registrations 
		WHERE id = $1 AND namespace = $2
	`

	var result WebhookRegistration
	err := r.db.GetContext(ctx, &result, query, webhookID, namespace)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}

	return &result, nil
}

// GetWebhooksByNamespace gets webhooks by namespace
func (r *Repository) GetWebhooksByNamespace(ctx context.Context, namespace string, activeOnly bool) ([]*WebhookRegistration, error) {
	return r.ListWebhooks(ctx, namespace, activeOnly)
}

// UpdateWebhook updates a webhook registration
func (r *Repository) UpdateWebhook(ctx context.Context, webhook *WebhookRegistration) error {
	webhook.UpdatedAt = time.Now()

	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
		UPDATE webhook_registrations 
		SET url = $3, headers = $4, timeout = $5, active = $6, 
		    description = $7, updated_at = NOW()
		WHERE id = $1 AND namespace = $2
	`

	_, err = r.db.ExecContext(ctx, query,
		webhook.ID, webhook.Namespace, webhook.URL, headersJSON,
		webhook.Timeout, webhook.Active, webhook.Description)
	return storage.Error(err)
}

// GetWebhooksByHealth retrieves webhooks filtered by health status
func (r *Repository) GetWebhooksByHealth(ctx context.Context, health WebhookHealth) ([]*WebhookRegistration, error) {
	query := `
		SELECT wr.id, wr.namespace, wr.events, wr.url, wr.headers, wr.timeout, 
		       wr.active, wr.description, wr.health,
		       wr.max_retries, wr.retry_backoff_seconds, wr.capture_response_body, wr.follow_redirects,
		       wr.verify_ssl, wr.request_timeout_seconds, wr.expected_status_codes, wr.webhook_secret,
		       wr.user_agent, wr.content_type, wr.created_at, wr.updated_at
		FROM webhook_registrations wr
		WHERE wr.health = $1
		ORDER BY wr.created_at DESC
	`

	type webhookRow struct {
		ID                    uuid.UUID `db:"id"`
		Namespace             string    `db:"namespace"`
		EventsJSON            []byte    `db:"events"`
		URL                   string    `db:"url"`
		HeadersJSON           []byte    `db:"headers"`
		Timeout               int       `db:"timeout"`
		Active                bool      `db:"active"`
		Description           string    `db:"description"`
		Health                string    `db:"health"`
		MaxRetries            int       `db:"max_retries"`
		RetryBackoffSeconds   int       `db:"retry_backoff_seconds"`
		CaptureResponseBody   bool      `db:"capture_response_body"`
		FollowRedirects       bool      `db:"follow_redirects"`
		VerifySSL             bool      `db:"verify_ssl"`
		RequestTimeoutSeconds int       `db:"request_timeout_seconds"`
		ExpectedStatusCodes   []byte    `db:"expected_status_codes"`
		WebhookSecret         string    `db:"webhook_secret"`
		UserAgent             string    `db:"user_agent"`
		ContentType           string    `db:"content_type"`
		CreatedAt             time.Time `db:"created_at"`
		UpdatedAt             time.Time `db:"updated_at"`
	}

	var rows []webhookRow
	err := r.db.SelectContext(ctx, &rows, query, string(health))
	if err != nil {
		return nil, storage.Error(err)
	}

	var webhooks []*WebhookRegistration
	for _, row := range rows {
		webhook := &WebhookRegistration{
			ID:                    row.ID,
			Namespace:             row.Namespace,
			URL:                   row.URL,
			Timeout:               row.Timeout,
			Active:                row.Active,
			Description:           row.Description,
			Health:                WebhookHealth(row.Health),
			MaxRetries:            row.MaxRetries,
			RetryBackoffSeconds:   row.RetryBackoffSeconds,
			CaptureResponseBody:   row.CaptureResponseBody,
			FollowRedirects:       row.FollowRedirects,
			VerifySSL:             row.VerifySSL,
			RequestTimeoutSeconds: row.RequestTimeoutSeconds,
			WebhookSecret:         row.WebhookSecret,
			UserAgent:             row.UserAgent,
			ContentType:           row.ContentType,
			CreatedAt:             row.CreatedAt,
			UpdatedAt:             row.UpdatedAt,
		}

		if err := json.Unmarshal(row.HeadersJSON, &webhook.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}

		if err := json.Unmarshal(row.ExpectedStatusCodes, &webhook.ExpectedStatusCodes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal expected status codes: %w", err)
		}

		webhooks = append(webhooks, webhook)
	}

	return webhooks, nil
}
