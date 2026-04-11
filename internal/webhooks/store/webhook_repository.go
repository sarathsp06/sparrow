package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// RegisterWebhook creates a new webhook registration.
// Returns storage.ErrAlreadyExists if a webhook with the same tenant, namespace,
// and URL already exists.
func (r *Repository) RegisterWebhook(ctx context.Context, tenantID uuid.UUID, registration *WebhookRegistration) error {
	if err := checkWebhookDuplicate(ctx, r.conn, tenantID, registration); err != nil {
		return err
	}
	return insertWebhookRegistration(ctx, r.conn, tenantID, registration)
}

// UnregisterWebhook permanently deletes a webhook registration and all associated data.
func (r *Repository) UnregisterWebhook(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) error {
	query := `DELETE FROM webhook_registrations WHERE id = $1 AND tenant_id = $2`
	_, err := r.conn.ExecContext(ctx, query, webhookID, tenantID)
	return storage.Error(err)
}

// checkWebhookDuplicate checks if a webhook with the same tenant, namespace, and URL
// already exists. If found, sets registration.ID to the existing ID and returns
// storage.ErrAlreadyExists. Used by RegisterWebhook and RegisterWebhookWithSubscriptions.
func checkWebhookDuplicate(ctx context.Context, conn storage.DBTX, tenantID uuid.UUID, registration *WebhookRegistration) error {
	checkQuery := `SELECT id FROM webhook_registrations WHERE tenant_id = $1 AND namespace = $2 AND url = $3 LIMIT 1`
	var existingID uuid.UUID
	err := conn.GetContext(ctx, &existingID, checkQuery, tenantID, registration.Namespace, registration.URL)
	if err == nil && existingID != uuid.Nil {
		registration.ID = existingID
		return storage.ErrAlreadyExists
	} else if err != nil && !storage.IsNotFound(storage.Error(err)) {
		return storage.Error(err)
	}
	return nil
}

// insertWebhookRegistration is the single canonical INSERT for webhook_registrations.
// It handles ID generation, default health, headers marshalling, timestamps, and the INSERT.
// Used by RegisterWebhook and RegisterWebhookWithSubscriptions.
func insertWebhookRegistration(ctx context.Context, conn storage.DBTX, tenantID uuid.UUID, registration *WebhookRegistration) error {
	if registration.ID == uuid.Nil {
		registration.ID = uuid.New()
	}
	registration.TenantID = tenantID
	registration.Health = HealthUnknown

	headersJSON, err := json.Marshal(registration.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	now := time.Now()
	if registration.CreatedAt.IsZero() {
		registration.CreatedAt = now
	}
	registration.UpdatedAt = now

	query := `
		INSERT INTO webhook_registrations (
			id, tenant_id, namespace, url, headers, timeout, active, description, health,
			max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			user_agent, content_type, secret_headers, rate_limit_rps, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`

	_, err = conn.ExecContext(ctx, query,
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
		registration.SecretHeaders,
		registration.RateLimitRPS,
		registration.CreatedAt,
		registration.UpdatedAt,
	)
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
	err := r.conn.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT wr.id, wr.tenant_id, wr.namespace, wr.url, wr.headers, wr.timeout, wr.active, wr.description, wr.health,
		       wr.max_retries, wr.retry_backoff_seconds, wr.capture_response_body, wr.follow_redirects,
		       wr.verify_ssl, wr.request_timeout_seconds, wr.expected_status_codes, wr.webhook_secret,
		       wr.user_agent, wr.content_type, wr.secret_headers, wr.rate_limit_rps, wr.created_at, wr.updated_at
		FROM webhook_registrations wr
		LEFT JOIN event_subscriptions es ON wr.id = es.webhook_id
		WHERE %s
		ORDER BY wr.created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)

	queryArgs := append(args, limit, offset)

	var webhooks []*WebhookRegistration
	err = r.conn.SelectContext(ctx, &webhooks, query, queryArgs...)
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
		argIdx++ //nolint:ineffassign // kept for clarity and future extensibility
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
				COUNT(wd.id) FILTER (WHERE wd.status = 'failed') as failed_deliveries,
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
	err := r.conn.GetContext(ctx, &stats, query, args...)
	if err != nil {
		return nil, storage.Error(err)
	}

	return &stats, nil
}

// RegisterWebhookWithSubscriptions creates a webhook and its subscriptions atomically.
// Both the webhook registration and all subscriptions are created within a single
// database transaction. If any subscription fails, the entire operation is rolled back.
func (r *Repository) RegisterWebhookWithSubscriptions(ctx context.Context, tenantID uuid.UUID, registration *WebhookRegistration, subscriptions []*EventSubscription) error {
	return storage.WithTransaction(r.db, func(tx storage.DBTX) error {
		if err := checkWebhookDuplicate(ctx, tx, tenantID, registration); err != nil {
			return err
		}
		if err := insertWebhookRegistration(ctx, tx, tenantID, registration); err != nil {
			return err
		}

		// Create all subscriptions within the same transaction
		for _, sub := range subscriptions {
			sub.WebhookID = registration.ID
			if err := insertSubscription(ctx, tx, tenantID, sub); err != nil {
				return fmt.Errorf("failed to create subscription for event %s: %w", sub.EventName, err)
			}
		}

		return nil
	})
}

// ReplaceWebhookSubscriptions atomically deletes all existing subscriptions for a webhook
// and creates new ones. This prevents the partial-update bug where old subscriptions
// could be deleted but new ones fail to create.
func (r *Repository) ReplaceWebhookSubscriptions(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, newSubscriptions []*EventSubscription) error {
	// When called through RunInTransaction, r.conn is already a tx.
	// When called standalone, wrap in a new transaction for atomicity.
	if _, isTx := r.conn.(*sqlx.Tx); isTx {
		return r.replaceWebhookSubscriptions(ctx, r.conn, tenantID, webhookID, namespace, newSubscriptions)
	}
	return storage.WithTransaction(r.db, func(tx storage.DBTX) error {
		return r.replaceWebhookSubscriptions(ctx, tx, tenantID, webhookID, namespace, newSubscriptions)
	})
}

func (r *Repository) replaceWebhookSubscriptions(ctx context.Context, conn storage.DBTX, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, newSubscriptions []*EventSubscription) error {
	// Delete all existing subscriptions for this webhook
	deleteQuery := `DELETE FROM event_subscriptions WHERE tenant_id = $1 AND webhook_id = $2`
	_, err := conn.ExecContext(ctx, deleteQuery, tenantID, webhookID)
	if err != nil {
		return fmt.Errorf("failed to delete existing subscriptions: %w", storage.Error(err))
	}

	// Create new subscriptions
	for _, sub := range newSubscriptions {
		sub.WebhookID = webhookID
		sub.Namespace = namespace
		if err := insertSubscription(ctx, conn, tenantID, sub); err != nil {
			return fmt.Errorf("failed to create subscription for event %s: %w", sub.EventName, err)
		}
	}

	return nil
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
			       user_agent, content_type, secret_headers, rate_limit_rps, created_at, updated_at
			FROM webhook_registrations
			WHERE id = $1 AND tenant_id = $2 AND namespace = $3
		`
		args = []any{webhookID, tenantID, namespace}
	} else {
		query = `
			SELECT id, tenant_id, namespace, url, headers, timeout, active, description, health,
			       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			       user_agent, content_type, secret_headers, rate_limit_rps, created_at, updated_at
			FROM webhook_registrations
			WHERE id = $1 AND tenant_id = $2
		`
		args = []any{webhookID, tenantID}
	}

	var result WebhookRegistration
	err := r.conn.GetContext(ctx, &result, query, args...)
	if err != nil {
		return nil, storage.Error(err)
	}

	return &result, nil
}

// UpdateWebhook updates a webhook registration within a tenant.
// This persists ALL mutable fields including HTTP config, secret headers, and webhook secret.
func (r *Repository) UpdateWebhook(ctx context.Context, tenantID uuid.UUID, webhook *WebhookRegistration) error {
	webhook.UpdatedAt = time.Now()

	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
		UPDATE webhook_registrations
		SET url = $4, headers = $5, timeout = $6, active = $7,
		    description = $8,
		    max_retries = $9, retry_backoff_seconds = $10,
		    capture_response_body = $11, follow_redirects = $12,
		    verify_ssl = $13, request_timeout_seconds = $14,
		    expected_status_codes = $15, webhook_secret = $16,
		    user_agent = $17, content_type = $18,
		    secret_headers = $19, rate_limit_rps = $20, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND namespace = $3
	`

	_, err = r.conn.ExecContext(ctx, query,
		webhook.ID, tenantID, webhook.Namespace,
		webhook.URL, headersJSON, webhook.Timeout, webhook.Active, webhook.Description,
		webhook.MaxRetries, webhook.RetryBackoffSeconds,
		webhook.CaptureResponseBody, webhook.FollowRedirects,
		webhook.VerifySSL, webhook.RequestTimeoutSeconds,
		pq.Array(webhook.ExpectedStatusCodes), webhook.WebhookSecret,
		webhook.UserAgent, webhook.ContentType,
		webhook.SecretHeaders, webhook.RateLimitRPS,
	)
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
	err := r.conn.GetContext(ctx, &totalCount, countQuery, tenantID, string(health))
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	query := `
		SELECT id, tenant_id, namespace, url, headers, timeout,
		       active, description, health,
		       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
		       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
		       user_agent, content_type, secret_headers, rate_limit_rps, created_at, updated_at
		FROM webhook_registrations
		WHERE tenant_id = $1 AND health = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	var webhooks []*WebhookRegistration
	err = r.conn.SelectContext(ctx, &webhooks, query, tenantID, string(health), limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return webhooks, totalCount, nil
}

// insertSubscription is the single canonical INSERT for event_subscriptions.
// It handles ID generation, timestamps, JSON marshalling of headers and label_filters,
// and is used by CreateSubscription, RegisterWebhookWithSubscriptions, and
// ReplaceWebhookSubscriptions to avoid duplication and ensure all columns are included.
func insertSubscription(ctx context.Context, conn storage.DBTX, tenantID uuid.UUID, sub *EventSubscription) error {
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	sub.TenantID = tenantID
	now := time.Now()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now

	headersJSON, err := json.Marshal(sub.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	labelFiltersJSON, err := json.Marshal(sub.LabelFilters)
	if err != nil {
		return fmt.Errorf("failed to marshal label_filters: %w", err)
	}

	query := `
		INSERT INTO event_subscriptions (
			id, tenant_id, webhook_id, event_name, namespace, headers, method,
			transform_enabled, transform_template, timeout, label_filters, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = conn.ExecContext(ctx, query,
		sub.ID,
		sub.TenantID,
		sub.WebhookID,
		sub.EventName,
		sub.Namespace,
		headersJSON,
		sub.Method,
		sub.TransformEnabled,
		sub.TransformTemplate,
		sub.Timeout,
		labelFiltersJSON,
		sub.CreatedAt,
		sub.UpdatedAt,
	)
	return storage.Error(err)
}

// AcquireDeliverySlot atomically advances the leaky bucket for a webhook and
// returns the slot time assigned to this delivery plus the configured rate.
// If the webhook has no rate limit state row, returns (zero time, 0, nil)
// meaning "no rate limit configured — send immediately".
func (r *Repository) AcquireDeliverySlot(ctx context.Context, webhookID uuid.UUID) (time.Time, float64, error) {
	// Atomic UPDATE that advances next_delivery_at by 1/rate_limit_rps.
	// If the bucket has drained (next_delivery_at <= NOW()), the next slot
	// starts from NOW() + interval. Otherwise it extends from the current
	// next_delivery_at. The RETURNING clause gives us the NEW next_delivery_at
	// (the slot AFTER ours) and the rate. Our slot = returned - interval.
	query := `
		UPDATE webhook_rate_limit_state rls
		SET next_delivery_at =
			CASE
				WHEN rls.next_delivery_at <= NOW()
				THEN NOW() + (interval '1 second' / wr.rate_limit_rps)
				ELSE rls.next_delivery_at + (interval '1 second' / wr.rate_limit_rps)
			END
		FROM webhook_registrations wr
		WHERE rls.webhook_id = wr.id AND rls.webhook_id = $1
		RETURNING rls.next_delivery_at, wr.rate_limit_rps
	`

	var result struct {
		NextDeliveryAt time.Time `db:"next_delivery_at"`
		RateLimitRPS   float64   `db:"rate_limit_rps"`
	}
	err := r.conn.GetContext(ctx, &result, query, webhookID)
	if err != nil {
		// sql.ErrNoRows means no rate limit state row — no limit configured
		if storage.IsNotFound(err) {
			return time.Time{}, 0, nil
		}
		return time.Time{}, 0, storage.Error(err)
	}

	return result.NextDeliveryAt, result.RateLimitRPS, nil
}

// UpsertRateLimitState creates or resets the rate limit state row for a webhook.
// Called when a webhook is created or updated with a rate limit.
func (r *Repository) UpsertRateLimitState(ctx context.Context, webhookID uuid.UUID) error {
	query := `
		INSERT INTO webhook_rate_limit_state (webhook_id, next_delivery_at)
		VALUES ($1, NOW())
		ON CONFLICT (webhook_id) DO UPDATE SET next_delivery_at = NOW()
	`
	_, err := r.conn.ExecContext(ctx, query, webhookID)
	return storage.Error(err)
}

// DeleteRateLimitState removes the rate limit state row for a webhook.
// Called when rate limiting is removed from a webhook.
func (r *Repository) DeleteRateLimitState(ctx context.Context, webhookID uuid.UUID) error {
	query := `DELETE FROM webhook_rate_limit_state WHERE webhook_id = $1`
	_, err := r.conn.ExecContext(ctx, query, webhookID)
	return storage.Error(err)
}
