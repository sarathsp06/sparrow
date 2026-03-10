package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// RegisterWebhook creates a new webhook registration with duplicate prevention.
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
func (r *Repository) UnregisterWebhook(ctx context.Context, webhookID uuid.UUID) error {
	query := `DELETE FROM webhook_registrations WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, webhookID)
	return storage.Error(err)
}

// ListWebhooks retrieves webhooks for a namespace with optional active status filtering and event filtering.
func (r *Repository) ListWebhooks(ctx context.Context, namespace string, event string, activeOnly bool) ([]*WebhookRegistration, error) {
	webhooks, _, err := r.ListWebhooksPaginated(ctx, namespace, event, activeOnly, 1000, 0)
	return webhooks, err
}

// ListWebhooksPaginated retrieves webhooks with pagination.
// When namespace is empty, returns webhooks across all namespaces.
func (r *Repository) ListWebhooksPaginated(ctx context.Context, namespace string, event string, activeOnly bool, limit, offset int) ([]*WebhookRegistration, int, error) {
	// Build WHERE clause and args dynamically based on whether namespace is provided
	var conditions []string
	var args []interface{}
	argIdx := 1

	if namespace != "" {
		conditions = append(conditions, fmt.Sprintf("wr.namespace = $%d", argIdx))
		args = append(args, namespace)
		argIdx++
	}

	conditions = append(conditions, fmt.Sprintf("($%d IS FALSE OR wr.active = true)", argIdx))
	args = append(args, activeOnly)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("($%d = '' OR es.event_name = $%d)", argIdx, argIdx))
	args = append(args, event)
	argIdx++

	whereClause := strings.Join(conditions, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(DISTINCT wr.id)
		FROM webhook_registrations wr
		LEFT JOIN event_subscriptions es ON wr.id = es.webhook_id
		WHERE %s
	`, whereClause)

	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT wr.id, wr.namespace, wr.url, wr.headers, wr.timeout, wr.active, wr.description, wr.health,
		       wr.max_retries, wr.retry_backoff_seconds, wr.capture_response_body, wr.follow_redirects,
		       wr.verify_ssl, wr.request_timeout_seconds, wr.expected_status_codes, wr.webhook_secret,
		       wr.user_agent, wr.content_type, wr.created_at, wr.updated_at
		FROM webhook_registrations wr
		LEFT JOIN event_subscriptions es ON wr.id = es.webhook_id
		WHERE %s
		ORDER BY wr.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	queryArgs := append(args, limit, offset)

	var webhooks []*WebhookRegistration
	err = r.db.SelectContext(ctx, &webhooks, query, queryArgs...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return webhooks, totalCount, nil
}

// GetNamespaceStats retrieves statistics for a namespace, or across all namespaces if namespace is empty
func (r *Repository) GetNamespaceStats(ctx context.Context, namespace string) (*NamespaceStats, error) {
	var namespaceFilter string
	var args []interface{}

	if namespace != "" {
		namespaceFilter = "WHERE namespace = $1"
		args = append(args, namespace)
	}

	// Build delivery filter based on whether namespace is provided
	var deliveryNamespaceFilter string
	if namespace != "" {
		deliveryNamespaceFilter = "WHERE wr.namespace = $1"
	}

	query := fmt.Sprintf(`
		WITH webhook_counts AS (
			SELECT
				COUNT(*) as total_webhooks,
				COUNT(*) FILTER (WHERE active = true) as active_webhooks
			FROM webhook_registrations
			%s
		),
		delivery_stats AS (
			SELECT
				COUNT(wd.id) as total_deliveries,
				COUNT(wd.id) FILTER (WHERE wd.status = 'success') as successful_deliveries,
				COUNT(wd.id) FILTER (WHERE wd.status IN ('failed', 'expired')) as failed_deliveries,
				COUNT(wd.id) FILTER (WHERE wd.status IN ('pending', 'sending', 'retrying')) as pending_deliveries
			FROM webhook_deliveries wd
			JOIN webhook_registrations wr ON wd.webhook_id = wr.id
			%s
		)
		SELECT
			wc.total_webhooks,
			wc.active_webhooks,
			COALESCE(ds.total_deliveries, 0) as total_deliveries,
			COALESCE(ds.successful_deliveries, 0) as successful_deliveries,
			COALESCE(ds.failed_deliveries, 0) as failed_deliveries,
			COALESCE(ds.pending_deliveries, 0) as pending_deliveries,
			CASE
				WHEN COALESCE(ds.total_deliveries, 0) > 0
				THEN CAST(ds.successful_deliveries AS FLOAT) / ds.total_deliveries
				ELSE 0
			END as success_rate
		FROM webhook_counts wc, delivery_stats ds
	`, namespaceFilter, deliveryNamespaceFilter)

	var stats NamespaceStats
	err := r.db.GetContext(ctx, &stats, query, args...)
	if err != nil {
		return nil, storage.Error(err)
	}

	return &stats, nil
}

// GetWebhookByID gets a webhook by ID, optionally filtered by namespace.
// When namespace is empty, looks up by webhook ID alone.
func (r *Repository) GetWebhookByID(ctx context.Context, webhookID uuid.UUID, namespace string) (*WebhookRegistration, error) {
	var query string
	var args []interface{}

	if namespace != "" {
		query = `
			SELECT id, namespace, url, headers, timeout, active, description, health,
			       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			       user_agent, content_type, created_at, updated_at
			FROM webhook_registrations 
			WHERE id = $1 AND namespace = $2
		`
		args = []interface{}{webhookID, namespace}
	} else {
		query = `
			SELECT id, namespace, url, headers, timeout, active, description, health,
			       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			       user_agent, content_type, created_at, updated_at
			FROM webhook_registrations 
			WHERE id = $1
		`
		args = []interface{}{webhookID}
	}

	var result WebhookRegistration
	err := r.db.GetContext(ctx, &result, query, args...)
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
	return r.ListWebhooks(ctx, namespace, "", activeOnly)
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
	webhooks, _, err := r.GetWebhooksByHealthPaginated(ctx, health, 1000, 0)
	return webhooks, err
}

// GetWebhooksByHealthPaginated retrieves webhooks filtered by health status with pagination
func (r *Repository) GetWebhooksByHealthPaginated(ctx context.Context, health WebhookHealth, limit, offset int) ([]*WebhookRegistration, int, error) {
	countQuery := `SELECT COUNT(*) FROM webhook_registrations WHERE health = $1`
	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, string(health))
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	query := `
		SELECT id, namespace, url, headers, timeout,
		       active, description, health,
		       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
		       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
		       user_agent, content_type, created_at, updated_at
		FROM webhook_registrations
		WHERE health = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	type webhookRow struct {
		ID                    uuid.UUID     `db:"id"`
		Namespace             string        `db:"namespace"`
		URL                   string        `db:"url"`
		HeadersJSON           []byte        `db:"headers"`
		Timeout               int           `db:"timeout"`
		Active                bool          `db:"active"`
		Description           string        `db:"description"`
		Health                string        `db:"health"`
		MaxRetries            int           `db:"max_retries"`
		RetryBackoffSeconds   int           `db:"retry_backoff_seconds"`
		CaptureResponseBody   bool          `db:"capture_response_body"`
		FollowRedirects       bool          `db:"follow_redirects"`
		VerifySSL             bool          `db:"verify_ssl"`
		RequestTimeoutSeconds int           `db:"request_timeout_seconds"`
		ExpectedStatusCodes   pq.Int64Array `db:"expected_status_codes"`
		WebhookSecret         string        `db:"webhook_secret"`
		UserAgent             string        `db:"user_agent"`
		ContentType           string        `db:"content_type"`
		CreatedAt             time.Time     `db:"created_at"`
		UpdatedAt             time.Time     `db:"updated_at"`
	}

	var rows []webhookRow
	err = r.db.SelectContext(ctx, &rows, query, string(health), limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
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
			return nil, 0, fmt.Errorf("failed to unmarshal headers: %w", err)
		}

		webhook.ExpectedStatusCodes = row.ExpectedStatusCodes

		webhooks = append(webhooks, webhook)
	}

	return webhooks, totalCount, nil
}
