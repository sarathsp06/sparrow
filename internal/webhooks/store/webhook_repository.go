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

// WebhookRepository defines operations for webhook_registrations.
type WebhookRepository interface {
	RegisterWebhook(ctx context.Context, tenantID uuid.UUID, registration *WebhookRegistration) error
	UnregisterWebhook(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) error
	ListWebhooks(ctx context.Context, tenantID uuid.UUID, consumer string, event string, activeOnly bool) ([]*WebhookRegistration, error)
	ListWebhooksPaginated(ctx context.Context, tenantID uuid.UUID, consumer string, event string, activeOnly bool, health WebhookHealth, limit, offset int) ([]*WebhookRegistration, int, error)
	GetWebhookByID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string) (*WebhookRegistration, error)
	UpdateWebhook(ctx context.Context, tenantID uuid.UUID, webhook *WebhookRegistration) error
	ListConsumers(ctx context.Context, tenantID uuid.UUID, query string, limit int) ([]string, error)
}

// RateLimitRepository defines operations for per-webhook rate limiting.
type RateLimitRepository interface {
	AcquireDeliverySlot(ctx context.Context, webhookID uuid.UUID) (time.Duration, float64, error)
	UpsertRateLimitState(ctx context.Context, webhookID uuid.UUID) error
	DeleteRateLimitState(ctx context.Context, webhookID uuid.UUID) error
}

// RegisterWebhook creates a new webhook registration.
// Returns storage.ErrAlreadyExists if a webhook with the same tenant, consumer,
// and URL already exists.
func (r *Repository) RegisterWebhook(ctx context.Context, tenantID uuid.UUID, registration *WebhookRegistration) error {
	if err := checkWebhookDuplicate(ctx, r.conn, tenantID, registration); err != nil {
		return err
	}
	return insertWebhookRegistration(ctx, r.conn, tenantID, registration)
}

// UnregisterWebhook soft-deletes a webhook: it is marked deleted and inactive,
// disappears from every webhook read (lookups, lists, stats, fan-out), and its
// URL can be registered again. Its deliveries, health history and
// subscriptions stay, so nothing referencing it is touched. A hard delete
// cascaded through every delivery and health event of the webhook (8s and a
// row lock for one with 600k deliveries); this is a single-row update.
// Returns storage.ErrNotFound when the webhook does not exist or is already
// deleted.
func (r *Repository) UnregisterWebhook(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) error {
	query := `
		UPDATE webhook_registrations
		SET deleted_at = NOW(), active = false, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`
	res, err := r.conn.ExecContext(ctx, query, webhookID, tenantID)
	if err != nil {
		return storage.Error(err)
	}
	if n, err := res.RowsAffected(); err == nil && n == 0 {
		return storage.ErrNotFound
	}
	return nil
}

// checkWebhookDuplicate checks if a webhook with the same tenant, consumer, and URL
// already exists. If found, sets registration.ID to the existing ID and returns
// storage.ErrAlreadyExists. Used by RegisterWebhook and RegisterWebhookWithSubscriptions.
func checkWebhookDuplicate(ctx context.Context, conn storage.DBTX, tenantID uuid.UUID, registration *WebhookRegistration) error {
	checkQuery := `SELECT id FROM webhook_registrations WHERE tenant_id = $1 AND consumer = $2 AND url = $3 AND deleted_at IS NULL LIMIT 1`
	var existingID uuid.UUID
	err := conn.GetContext(ctx, &existingID, checkQuery, tenantID, registration.Consumer, registration.URL)
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
			id, tenant_id, consumer, url, headers, timeout, active, description, health,
			max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			user_agent, content_type, secret_headers, rate_limit_rps, ed25519_private_key, signature_type, requires_transform, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26)
	`

	_, err = conn.ExecContext(ctx, query,
		registration.ID,
		registration.TenantID,
		registration.Consumer,
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
		registration.Ed25519PrivateKey,
		registration.SignatureType,
		registration.RequiresTransform,
		registration.CreatedAt,
		registration.UpdatedAt,
	)
	return storage.Error(err)
}

// ListWebhooks retrieves webhooks for a consumer with optional active status filtering and event filtering.
func (r *Repository) ListWebhooks(ctx context.Context, tenantID uuid.UUID, consumer string, event string, activeOnly bool) ([]*WebhookRegistration, error) {
	webhooks, _, err := r.ListWebhooksPaginated(ctx, tenantID, consumer, event, activeOnly, "", 1000, 0)
	return webhooks, err
}

// ListWebhooksPaginated retrieves webhooks with pagination.
// When consumer is empty, returns webhooks across all consumers within the tenant.
// When health is non-empty, only webhooks with that health status are returned.
func (r *Repository) ListWebhooksPaginated(ctx context.Context, tenantID uuid.UUID, consumer string, event string, activeOnly bool, health WebhookHealth, limit, offset int) ([]*WebhookRegistration, int, error) {
	var ns any
	if consumer != "" {
		ns = consumer
	}

	args := []any{tenantID, ns, activeOnly, event, string(health)}

	countQuery := `
		SELECT COUNT(DISTINCT wr.id)
		FROM webhook_registrations wr
		LEFT JOIN event_subscriptions es ON wr.id = es.webhook_id
		WHERE wr.tenant_id = $1
		  AND wr.deleted_at IS NULL
		  AND ($2::text IS NULL OR wr.consumer = $2)
		  AND ($3 IS FALSE OR wr.active = true)
		  AND ($4 = '' OR es.event_name = $4)
		  AND ($5 = '' OR wr.health = $5)
	`

	var totalCount int
	err := r.conn.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	query := `
		SELECT DISTINCT wr.id, wr.tenant_id, wr.consumer, wr.url, wr.headers, wr.timeout, wr.active, wr.description, wr.health,
		       wr.max_retries, wr.retry_backoff_seconds, wr.capture_response_body, wr.follow_redirects,
		       wr.verify_ssl, wr.request_timeout_seconds, wr.expected_status_codes, wr.webhook_secret,
		       wr.user_agent, wr.content_type, wr.secret_headers, wr.rate_limit_rps, wr.ed25519_private_key, wr.signature_type, wr.requires_transform,
		       wr.auto_disabled_at, wr.auto_disabled_reason, wr.created_at, wr.updated_at
		FROM webhook_registrations wr
		LEFT JOIN event_subscriptions es ON wr.id = es.webhook_id
		WHERE wr.tenant_id = $1
		  AND wr.deleted_at IS NULL
		  AND ($2::text IS NULL OR wr.consumer = $2)
		  AND ($3 IS FALSE OR wr.active = true)
		  AND ($4 = '' OR es.event_name = $4)
		  AND ($5 = '' OR wr.health = $5)
		ORDER BY wr.created_at DESC
		LIMIT $6 OFFSET $7
	`

	queryArgs := append(args, limit, offset)
	var webhooks []*WebhookRegistration
	err = r.conn.SelectContext(ctx, &webhooks, query, queryArgs...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return webhooks, totalCount, nil
}

// GetConsumerStats retrieves statistics for a consumer, or across all consumers within the tenant if consumer is empty
func (r *Repository) GetConsumerStats(ctx context.Context, tenantID uuid.UUID, consumer string) (*ConsumerStats, error) {
	var ns any
	if consumer != "" {
		ns = consumer
	}

	args := []any{tenantID, ns}

	// Attempt counts come from the running totals the health evaluator keeps
	// in webhook_metrics (one row per webhook), so they lag by at most
	// one evaluator pass. Counting webhook_deliveries here instead took 4.8s
	// for a busy consumer on 6M deliveries. Pending is a live, index-only
	// count over the small partial idx_webhook_deliveries_unsuccessful.
	query := `
		WITH wh AS (
			SELECT id, active
			FROM webhook_registrations
			WHERE tenant_id = $1
			  AND deleted_at IS NULL
			  AND ($2::text IS NULL OR consumer = $2)
		),
		webhook_counts AS (
			SELECT COUNT(*) AS total_webhooks, COUNT(*) FILTER (WHERE active) AS active_webhooks
			FROM wh
		),
		attempt_stats AS (
			SELECT COALESCE(SUM(m.total_attempts), 0) AS attempts,
			       COALESCE(SUM(m.total_failures), 0) AS failures
			FROM webhook_metrics m
			WHERE m.webhook_id IN (SELECT id FROM wh)
		),
		in_flight AS (
			SELECT COUNT(*) AS pending
			FROM webhook_deliveries wd
			WHERE wd.webhook_id IN (SELECT id FROM wh)
			  AND wd.status IN ('pending', 'sending', 'retrying')
		)
		SELECT
			wc.total_webhooks,
			wc.active_webhooks,
			a.attempts AS total_deliveries,
			a.attempts - a.failures AS successful_deliveries,
			a.failures AS failed_deliveries,
			f.pending AS pending_deliveries,
			CASE WHEN a.attempts > 0
			     THEN CAST(a.attempts - a.failures AS FLOAT) / a.attempts
			     ELSE 0
			END AS success_rate
		FROM webhook_counts wc, attempt_stats a, in_flight f
	`

	var stats ConsumerStats
	err := r.conn.GetContext(ctx, &stats, query, args...)
	if err != nil {
		return nil, storage.Error(err)
	}

	return &stats, nil
}

// LockWebhook reads a webhook and row-locks it until the surrounding
// transaction ends. Writers that depend on requires_transform serialize on
// it: changing the flag takes an exclusive lock, subscription writes a shared
// one, so a subscription can never be saved against a stale flag.
func (r *Repository) LockWebhook(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string, exclusive bool) (*WebhookRegistration, error) {
	lock := "FOR SHARE"
	if exclusive {
		lock = "FOR UPDATE"
	}
	query := `
		SELECT id, tenant_id, consumer, url, headers, timeout, active, description, health,
		       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
		       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
		       user_agent, content_type, secret_headers, rate_limit_rps, ed25519_private_key, signature_type, requires_transform,
		       auto_disabled_at, auto_disabled_reason, created_at, updated_at
		FROM webhook_registrations
		WHERE id = $1 AND tenant_id = $2 AND consumer = $3 AND deleted_at IS NULL
		` + lock
	var result WebhookRegistration
	if err := r.conn.GetContext(ctx, &result, query, webhookID, tenantID, consumer); err != nil {
		return nil, storage.Error(err)
	}
	return &result, nil
}

// RegisterWebhookWithSubscriptions creates a webhook and its subscriptions atomically.
// Both the webhook registration and all subscriptions are created within a single
// database transaction. If any subscription fails, the entire operation is rolled back.
// A subscription created with a template gets it as its first saved version,
// labelled with firstVersion's Source, Notes and SavedBy.
func (r *Repository) RegisterWebhookWithSubscriptions(ctx context.Context, tenantID uuid.UUID, registration *WebhookRegistration, subscriptions []*EventSubscription, firstVersion SubscriptionTemplateVersion) error {
	return storage.WithTransaction(r.db, func(tx storage.DBTX) error {
		if err := checkWebhookDuplicate(ctx, tx, tenantID, registration); err != nil {
			return err
		}
		if err := insertWebhookRegistration(ctx, tx, tenantID, registration); err != nil {
			return err
		}

		// Create all subscriptions within the same transaction.
		for _, sub := range subscriptions {
			sub.WebhookID = registration.ID
			if err := insertSubscription(ctx, tx, tenantID, sub); err != nil {
				return fmt.Errorf("failed to create subscription for event %s: %w", sub.EventName, err)
			}
			if strings.TrimSpace(sub.TransformTemplate) != "" {
				v := firstVersion
				v.ID, v.SubscriptionID, v.Template = uuid.Nil, sub.ID, sub.TransformTemplate
				if err := r.WithConn(tx).InsertTemplateVersion(ctx, tenantID, &v); err != nil {
					return err
				}
			}
		}

		return nil
	})
}

// ReplaceWebhookSubscriptions atomically deletes all existing subscriptions for a webhook
// and creates new ones. This prevents the partial-update bug where old subscriptions
// could be deleted but new ones fail to create.
func (r *Repository) ReplaceWebhookSubscriptions(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string, newSubscriptions []*EventSubscription) error {
	// When called through RunInTransaction, r.conn is already a tx.
	// When called standalone, wrap in a new transaction for atomicity.
	if _, isTx := r.conn.(*sqlx.Tx); isTx {
		return r.replaceWebhookSubscriptions(ctx, r.conn, tenantID, webhookID, consumer, newSubscriptions)
	}
	return storage.WithTransaction(r.db, func(tx storage.DBTX) error {
		return r.replaceWebhookSubscriptions(ctx, tx, tenantID, webhookID, consumer, newSubscriptions)
	})
}

func (r *Repository) replaceWebhookSubscriptions(ctx context.Context, conn storage.DBTX, tenantID uuid.UUID, webhookID uuid.UUID, consumer string, newSubscriptions []*EventSubscription) error {
	// Delete all existing subscriptions for this webhook
	deleteQuery := `DELETE FROM event_subscriptions WHERE tenant_id = $1 AND webhook_id = $2`
	_, err := conn.ExecContext(ctx, deleteQuery, tenantID, webhookID)
	if err != nil {
		return fmt.Errorf("failed to delete existing subscriptions: %w", storage.Error(err))
	}

	// Create new subscriptions
	for _, sub := range newSubscriptions {
		sub.WebhookID = webhookID
		sub.Consumer = consumer
		if err := insertSubscription(ctx, conn, tenantID, sub); err != nil {
			return fmt.Errorf("failed to create subscription for event %s: %w", sub.EventName, err)
		}
	}

	return nil
}

// GetWebhookByID gets a webhook by ID within a tenant, optionally filtered by consumer.
// When consumer is empty, looks up by webhook ID within the tenant. A deleted
// webhook is not found.
func (r *Repository) GetWebhookByID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, consumer string) (*WebhookRegistration, error) {
	var query string
	var args []any

	if consumer != "" {
		query = `
			SELECT id, tenant_id, consumer, url, headers, timeout, active, description, health,
			       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			       user_agent, content_type, secret_headers, rate_limit_rps, ed25519_private_key, signature_type, requires_transform,
			       auto_disabled_at, auto_disabled_reason, created_at, updated_at
			FROM webhook_registrations
			WHERE id = $1 AND tenant_id = $2 AND consumer = $3 AND deleted_at IS NULL
		`
		args = []any{webhookID, tenantID, consumer}
	} else {
		query = `
			SELECT id, tenant_id, consumer, url, headers, timeout, active, description, health,
			       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
			       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
			       user_agent, content_type, secret_headers, rate_limit_rps, ed25519_private_key, signature_type, requires_transform,
			       auto_disabled_at, auto_disabled_reason, created_at, updated_at
			FROM webhook_registrations
			WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL
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
// Saving it as active clears an automatic disable: the auto_disabled_* columns
// are reset and the receiver's failure run restarts, so a resumed webhook gets
// a full window before it can be auto-disabled again. Both statements of the
// CTE read the row as it was before the update.
func (r *Repository) UpdateWebhook(ctx context.Context, tenantID uuid.UUID, webhook *WebhookRegistration) error {
	webhook.UpdatedAt = time.Now()

	headersJSON, err := json.Marshal(webhook.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	query := `
		WITH resumed AS (
			UPDATE webhook_health_state SET failing_since = NULL
			WHERE $7 AND webhook_id = (
				SELECT id FROM webhook_registrations
				WHERE id = $1 AND tenant_id = $2 AND consumer = $3 AND auto_disabled_at IS NOT NULL AND deleted_at IS NULL
			)
		)
		UPDATE webhook_registrations
		SET url = $4, headers = $5, timeout = $6, active = $7,
		    description = $8,
		    max_retries = $9, retry_backoff_seconds = $10,
		    capture_response_body = $11, follow_redirects = $12,
		    verify_ssl = $13, request_timeout_seconds = $14,
		    expected_status_codes = $15, webhook_secret = $16,
		    user_agent = $17, content_type = $18,
		    secret_headers = $19, rate_limit_rps = $20,
		    ed25519_private_key = $21, signature_type = $22, requires_transform = $23,
		    auto_disabled_at = CASE WHEN $7 THEN NULL ELSE auto_disabled_at END,
		    auto_disabled_reason = CASE WHEN $7 THEN NULL ELSE auto_disabled_reason END,
		    updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2 AND consumer = $3 AND deleted_at IS NULL
	`

	_, err = r.conn.ExecContext(ctx, query,
		webhook.ID, tenantID, webhook.Consumer,
		webhook.URL, headersJSON, webhook.Timeout, webhook.Active, webhook.Description,
		webhook.MaxRetries, webhook.RetryBackoffSeconds,
		webhook.CaptureResponseBody, webhook.FollowRedirects,
		webhook.VerifySSL, webhook.RequestTimeoutSeconds,
		pq.Array(webhook.ExpectedStatusCodes), webhook.WebhookSecret,
		webhook.UserAgent, webhook.ContentType,
		webhook.SecretHeaders, webhook.RateLimitRPS,
		webhook.Ed25519PrivateKey, webhook.SignatureType, webhook.RequiresTransform,
	)
	return storage.Error(err)
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
	if sub.LabelFilters == nil {
		// A nil map marshals to jsonb null, which never matches the
		// subscription lookup predicate (label_filters = '{}' OR <@ labels).
		sub.LabelFilters = JSONStringMap{}
	}

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
			id, tenant_id, webhook_id, event_name, consumer, headers, method,
			transform_enabled, transform_template, timeout, label_filters, created_at, updated_at,
			on_transform_error, template_missing_key
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	sub.ApplyTemplateDefaults()
	_, err = conn.ExecContext(ctx, query,
		sub.ID,
		sub.TenantID,
		sub.WebhookID,
		sub.EventName,
		sub.Consumer,
		headersJSON,
		sub.Method,
		sub.TransformEnabled,
		sub.TransformTemplate,
		sub.Timeout,
		labelFiltersJSON,
		sub.CreatedAt,
		sub.UpdatedAt,
		sub.OnTransformError,
		sub.TemplateMissingKey,
	)
	return storage.Error(err)
}

// AcquireDeliverySlot implements a leaky bucket for a webhook. When the
// bucket is free (next_delivery_at <= NOW()), it atomically claims the slot,
// advances the bucket by one interval and returns a zero wait: "send now".
// When the bucket is busy, NOTHING is consumed: the wait until the current
// tail is returned so the caller can sleep and try again — a retry after the
// wait does not burn slots.
// The wait is computed on the database clock, never compared against the
// caller's clock: a worker whose clock is behind or ahead of Postgres would
// otherwise snooze a slot it was just granted (burning it) or send while the
// bucket is busy.
// If the webhook has no rate limit state row, returns (0, 0, nil) meaning
// "no rate limit configured — send immediately".
// ponytail: under READ COMMITTED a concurrent waiter can read a stale tail in
// the SELECT branch and send one extra delivery; bounded by worker concurrency.
// Upgrade path: SELECT ... FOR UPDATE in a transaction if strict RPS matters.
func (r *Repository) AcquireDeliverySlot(ctx context.Context, webhookID uuid.UUID) (time.Duration, float64, error) {
	query := `
		WITH granted AS (
			UPDATE webhook_rate_limit_state rls
			SET next_delivery_at = NOW() + (interval '1 second' / wr.rate_limit_rps)
			FROM webhook_registrations wr
			WHERE rls.webhook_id = wr.id AND rls.webhook_id = $1
			  AND wr.rate_limit_rps > 0
			  AND rls.next_delivery_at <= NOW()
			RETURNING wr.rate_limit_rps
		)
		SELECT 0::float8 AS wait_seconds, rate_limit_rps FROM granted
		UNION ALL
		SELECT GREATEST(EXTRACT(EPOCH FROM rls.next_delivery_at - NOW()), 0)::float8, wr.rate_limit_rps
		FROM webhook_rate_limit_state rls
		JOIN webhook_registrations wr ON rls.webhook_id = wr.id
		WHERE rls.webhook_id = $1
		  AND wr.rate_limit_rps > 0
		  AND NOT EXISTS (SELECT 1 FROM granted)
	`

	var result struct {
		WaitSeconds  float64 `db:"wait_seconds"`
		RateLimitRPS float64 `db:"rate_limit_rps"`
	}
	err := r.conn.GetContext(ctx, &result, query, webhookID)
	if err != nil {
		err = storage.Error(err)
		// No rate limit state row — no limit configured.
		if storage.IsNotFound(err) {
			return 0, 0, nil
		}
		return 0, 0, err
	}

	return time.Duration(result.WaitSeconds * float64(time.Second)), result.RateLimitRPS, nil
}

// UpsertRateLimitState creates the rate limit state row for a webhook if
// absent. An existing row keeps its future backlog (GREATEST) so config
// updates cannot rewind the bucket and let a burst exceed the configured RPS.
func (r *Repository) UpsertRateLimitState(ctx context.Context, webhookID uuid.UUID) error {
	query := `
		INSERT INTO webhook_rate_limit_state (webhook_id, next_delivery_at)
		VALUES ($1, NOW())
		ON CONFLICT (webhook_id) DO UPDATE
		SET next_delivery_at = GREATEST(webhook_rate_limit_state.next_delivery_at, NOW())
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
