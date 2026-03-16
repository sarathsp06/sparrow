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
func (r *Repository) RegisterWebhook(ctx context.Context, tenantID uuid.UUID, registration *WebhookRegistration) error {
	// Check for existing webhook with same tenant, namespace and url
	checkQuery := `SELECT id FROM webhook_registrations WHERE tenant_id = $1 AND namespace = $2 AND url = $3 LIMIT 1`
	var existingID uuid.UUID
	err := r.db.GetContext(ctx, &existingID, checkQuery, tenantID, registration.Namespace, registration.URL)
	if err == nil && existingID != uuid.Nil {
		// Already exists, treat as success
		return nil
	} else if err != nil && !storage.IsNotFound(storage.Error(err)) {
		// DB error
		return storage.Error(err)
	}

	registration.ID = uuid.New()
	registration.TenantID = tenantID
	registration.Health = HealthUnknown // New webhooks start with unknown health

	query := `
		INSERT INTO webhook_registrations (
			id, tenant_id, namespace, url, headers, timeout, active, description, health,
			max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			user_agent, content_type, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)
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
		registration.TenantID,
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
func (r *Repository) UnregisterWebhook(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) error {
	query := `DELETE FROM webhook_registrations WHERE id = $1 AND tenant_id = $2`
	_, err := r.db.ExecContext(ctx, query, webhookID, tenantID)
	return storage.Error(err)
}

// ListWebhooks retrieves webhooks for a namespace with optional active status filtering and event filtering.
func (r *Repository) ListWebhooks(ctx context.Context, tenantID uuid.UUID, namespace string, event string, activeOnly bool) ([]*WebhookRegistration, error) {
	webhooks, _, err := r.ListWebhooksPaginated(ctx, tenantID, namespace, event, activeOnly, 1000, 0)
	return webhooks, err
}

// ListWebhooksPaginated retrieves webhooks with pagination.
// When namespace is empty, returns webhooks across all namespaces within the tenant.
func (r *Repository) ListWebhooksPaginated(ctx context.Context, tenantID uuid.UUID, namespace string, event string, activeOnly bool, limit, offset int) ([]*WebhookRegistration, int, error) {
	// Build WHERE clause and args dynamically
	var conditions []string
	var args []any
	argIdx := 1

	// Always filter by tenant
	conditions = append(conditions, fmt.Sprintf("wr.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

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
		SELECT DISTINCT wr.id, wr.tenant_id, wr.namespace, wr.url, wr.headers, wr.timeout, wr.active, wr.description, wr.health,
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

// GetNamespaceStats retrieves statistics for a namespace, or across all namespaces within the tenant if namespace is empty
func (r *Repository) GetNamespaceStats(ctx context.Context, tenantID uuid.UUID, namespace string) (*NamespaceStats, error) {
	var conditions []string
	var deliveryConditions []string
	var args []any
	argIdx := 1

	// Always filter by tenant
	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	deliveryConditions = append(deliveryConditions, fmt.Sprintf("wr.tenant_id = $%d", argIdx))
	args = append(args, tenantID)
	argIdx++

	if namespace != "" {
		conditions = append(conditions, fmt.Sprintf("namespace = $%d", argIdx))
		deliveryConditions = append(deliveryConditions, fmt.Sprintf("wr.namespace = $%d", argIdx))
		args = append(args, namespace)
		argIdx++
	}

	webhookFilter := "WHERE " + strings.Join(conditions, " AND ")
	deliveryFilter := "WHERE " + strings.Join(deliveryConditions, " AND ")

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
	`, webhookFilter, deliveryFilter)

	var stats NamespaceStats
	err := r.db.GetContext(ctx, &stats, query, args...)
	if err != nil {
		return nil, storage.Error(err)
	}

	return &stats, nil
}

// GetWebhookByID gets a webhook by ID within a tenant, optionally filtered by namespace.
// When namespace is empty, looks up by webhook ID within the tenant.
func (r *Repository) GetWebhookByID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string) (*WebhookRegistration, error) {
	var query string
	var args []any

	if namespace != "" {
		query = `
			SELECT id, tenant_id, namespace, url, headers, timeout, active, description, health,
			       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			       user_agent, content_type, created_at, updated_at
			FROM webhook_registrations
			WHERE id = $1 AND tenant_id = $2 AND namespace = $3
		`
		args = []any{webhookID, tenantID, namespace}
	} else {
		query = `
			SELECT id, tenant_id, namespace, url, headers, timeout, active, description, health,
			       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			       user_agent, content_type, created_at, updated_at
			FROM webhook_registrations
			WHERE id = $1 AND tenant_id = $2
		`
		args = []any{webhookID, tenantID}
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

// UpdateWebhook updates a webhook registration within a tenant
func (r *Repository) UpdateWebhook(ctx context.Context, tenantID uuid.UUID, webhook *WebhookRegistration) error {
	webhook.UpdatedAt = time.Now()

	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
		UPDATE webhook_registrations
		SET url = $4, headers = $5, timeout = $6, active = $7,
		    description = $8, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND namespace = $3
	`

	_, err = r.db.ExecContext(ctx, query,
		webhook.ID, tenantID, webhook.Namespace, webhook.URL, headersJSON,
		webhook.Timeout, webhook.Active, webhook.Description)
	return storage.Error(err)
}

// GetWebhooksByHealth retrieves webhooks filtered by health status within a tenant
func (r *Repository) GetWebhooksByHealth(ctx context.Context, tenantID uuid.UUID, health WebhookHealth) ([]*WebhookRegistration, error) {
	webhooks, _, err := r.GetWebhooksByHealthPaginated(ctx, tenantID, health, 1000, 0)
	return webhooks, err
}

// GetWebhooksByHealthPaginated retrieves webhooks filtered by health status with pagination within a tenant
func (r *Repository) GetWebhooksByHealthPaginated(ctx context.Context, tenantID uuid.UUID, health WebhookHealth, limit, offset int) ([]*WebhookRegistration, int, error) {
	countQuery := `SELECT COUNT(*) FROM webhook_registrations WHERE tenant_id = $1 AND health = $2`
	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, tenantID, string(health))
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	query := `
		SELECT id, tenant_id, namespace, url, headers, timeout,
		       active, description, health,
		       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
		       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
		       user_agent, content_type, created_at, updated_at
		FROM webhook_registrations
		WHERE tenant_id = $1 AND health = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	type webhookRow struct {
		ID                    uuid.UUID     `db:"id"`
		TenantID              uuid.UUID     `db:"tenant_id"`
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
	err = r.db.SelectContext(ctx, &rows, query, tenantID, string(health), limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	var webhooks []*WebhookRegistration
	for _, row := range rows {
		webhook := &WebhookRegistration{
			ID:                    row.ID,
			TenantID:              row.TenantID,
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
