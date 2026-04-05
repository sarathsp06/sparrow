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

// CreateSubscription creates a new event subscription within a tenant
func (r *Repository) CreateSubscription(ctx context.Context, tenantID uuid.UUID, sub *EventSubscription) error {
	if sub.ID == uuid.Nil {
		sub.ID = uuid.New()
	}
	sub.TenantID = tenantID
	now := time.Now()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now

	query := `
		INSERT INTO event_subscriptions (
			id, tenant_id, webhook_id, event_name, namespace, headers, method,
			transform_enabled, transform_template, timeout, label_filters, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	headersJSON, err := json.Marshal(sub.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	labelFiltersJSON, err := json.Marshal(sub.LabelFilters)
	if err != nil {
		return fmt.Errorf("failed to marshal label_filters: %w", err)
	}

	_, err = r.conn.ExecContext(ctx, query,
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

// GetSubscription gets a subscription by ID within a tenant
func (r *Repository) GetSubscription(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) (*EventSubscription, error) {
	query := `
		SELECT id, tenant_id, webhook_id, event_name, namespace, headers, method,
		       transform_enabled, transform_template, timeout, label_filters, created_at, updated_at
		FROM event_subscriptions
		WHERE tenant_id = $1 AND id = $2
	`
	var sub EventSubscription
	err := r.conn.GetContext(ctx, &sub, query, tenantID, id)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &sub, nil
}

// UpdateSubscription updates a subscription within a tenant
func (r *Repository) UpdateSubscription(ctx context.Context, tenantID uuid.UUID, sub *EventSubscription) error {
	sub.UpdatedAt = time.Now()

	headersJSON, err := json.Marshal(sub.Headers)
	if err != nil {
		return fmt.Errorf("failed to marshal headers: %w", err)
	}

	labelFiltersJSON, err := json.Marshal(sub.LabelFilters)
	if err != nil {
		return fmt.Errorf("failed to marshal label_filters: %w", err)
	}

	query := `
		UPDATE event_subscriptions
		SET headers = $3, method = $4, transform_enabled = $5,
		    transform_template = $6, timeout = $7, label_filters = $8, updated_at = $9
		WHERE tenant_id = $1 AND id = $2
	`

	_, err = r.conn.ExecContext(ctx, query,
		tenantID,
		sub.ID,
		headersJSON,
		sub.Method,
		sub.TransformEnabled,
		sub.TransformTemplate,
		sub.Timeout,
		labelFiltersJSON,
		sub.UpdatedAt,
	)
	return storage.Error(err)
}

// DeleteSubscription deletes a subscription within a tenant
func (r *Repository) DeleteSubscription(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	query := `DELETE FROM event_subscriptions WHERE tenant_id = $1 AND id = $2`
	_, err := r.conn.ExecContext(ctx, query, tenantID, id)
	return storage.Error(err)
}

// ListSubscriptions lists subscriptions for a webhook within a tenant
func (r *Repository) ListSubscriptions(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID) ([]*EventSubscription, error) {
	query := `
		SELECT id, tenant_id, webhook_id, event_name, namespace, headers, method,
		       transform_enabled, transform_template, timeout, label_filters, created_at, updated_at
		FROM event_subscriptions
		WHERE tenant_id = $1 AND webhook_id = $2
		ORDER BY created_at DESC
	`
	var subs []*EventSubscription
	err := r.conn.SelectContext(ctx, &subs, query, tenantID, webhookID)
	if err != nil {
		return nil, storage.Error(err)
	}
	return subs, nil
}

// ListSubscriptionsByNamespace lists all subscriptions in a namespace within a tenant with pagination.
func (r *Repository) ListSubscriptionsByNamespace(ctx context.Context, tenantID uuid.UUID, namespace string, limit, offset int) ([]*EventSubscription, int, error) {
	countQuery := `SELECT COUNT(*) FROM event_subscriptions WHERE tenant_id = $1 AND namespace = $2`
	var totalCount int
	if err := r.conn.GetContext(ctx, &totalCount, countQuery, tenantID, namespace); err != nil {
		return nil, 0, storage.Error(err)
	}

	query := `
		SELECT id, tenant_id, webhook_id, event_name, namespace, headers, method,
		       transform_enabled, transform_template, timeout, label_filters, created_at, updated_at
		FROM event_subscriptions
		WHERE tenant_id = $1 AND namespace = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	var subs []*EventSubscription
	err := r.conn.SelectContext(ctx, &subs, query, tenantID, namespace, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}
	return subs, totalCount, nil
}

// GetSubscriptionsByEvent finds all active subscriptions for a specific event in a namespace within a tenant.
// Also includes catch-all subscriptions (event_name = '*') for the same namespace.
func (r *Repository) GetSubscriptionsByEvent(ctx context.Context, tenantID uuid.UUID, namespace, event string, labels map[string]string) ([]*EventSubscription, error) {
	query := `
		SELECT es.id, es.tenant_id, es.webhook_id, es.event_name, es.namespace, es.headers, es.method, 
		       es.transform_enabled, es.transform_template, es.timeout, es.label_filters, es.created_at, es.updated_at
		FROM event_subscriptions es
		JOIN webhook_registrations wr ON es.webhook_id = wr.id
		WHERE es.tenant_id = $1 AND es.namespace = $2
		  AND (es.event_name = $3 OR es.event_name = '*')
		  AND wr.active = true
		  AND (es.label_filters = '{}' OR es.label_filters <@ $4::jsonb)
	`

	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal labels: %w", err)
	}

	var subscriptions []*EventSubscription

	err = r.conn.SelectContext(ctx, &subscriptions, query, tenantID, namespace, event, labelsJSON)
	if err != nil {
		return nil, storage.Error(err)
	}
	return subscriptions, nil
}

// GetSubscriptionsWithWebhooksByEvent finds all active subscriptions for a specific event in a namespace within a tenant,
// including the webhook configuration for each subscription.
func (r *Repository) GetSubscriptionsWithWebhooksByEvent(ctx context.Context, tenantID uuid.UUID, namespace, event string, labels map[string]string) ([]*SubscriptionWithWebhook, error) {
	query := `
		SELECT
			es.id, es.webhook_id, es.event_name, es.namespace, es.headers as es_headers, es.method,
			es.transform_enabled, es.transform_template, es.timeout, es.label_filters, es.created_at, es.updated_at,
			wr.id as wr_id, wr.namespace as wr_namespace, wr.url, wr.headers as wr_headers,
			wr.timeout as wr_timeout, wr.active, wr.description, wr.health,
			wr.max_retries, wr.retry_backoff_seconds, wr.capture_response_body, wr.follow_redirects,
			wr.verify_ssl, wr.request_timeout_seconds, wr.expected_status_codes, wr.webhook_secret,
			wr.user_agent, wr.content_type, wr.secret_headers, wr.created_at as wr_created_at, wr.updated_at as wr_updated_at
		FROM event_subscriptions es
		JOIN webhook_registrations wr ON es.webhook_id = wr.id
		WHERE es.tenant_id = $1 AND es.namespace = $2
		  AND (es.event_name = $3 OR es.event_name = '*')
		  AND wr.active = true
		  AND (es.label_filters = '{}' OR es.label_filters <@ $4::jsonb)
	`

	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal labels: %w", err)
	}

	type rowStruct struct {
		// Subscription fields
		ID                uuid.UUID `db:"id"`
		WebhookID         uuid.UUID `db:"webhook_id"`
		EventName         string    `db:"event_name"`
		Namespace         string    `db:"namespace"`
		HeadersJSON       []byte    `db:"es_headers"`
		Method            string    `db:"method"`
		TransformEnabled  bool      `db:"transform_enabled"`
		TransformTemplate string    `db:"transform_template"`
		Timeout           int       `db:"timeout"`
		LabelFiltersJSON  []byte    `db:"label_filters"`
		CreatedAt         time.Time `db:"created_at"`
		UpdatedAt         time.Time `db:"updated_at"`

		// Webhook fields
		WRID                    uuid.UUID     `db:"wr_id"`
		WRNamespace             string        `db:"wr_namespace"`
		URL                     string        `db:"url"`
		WRHeadersJSON           []byte        `db:"wr_headers"`
		WRTimeout               int           `db:"wr_timeout"`
		Active                  bool          `db:"active"`
		Description             string        `db:"description"`
		Health                  string        `db:"health"`
		MaxRetries              int           `db:"max_retries"`
		RetryBackoffSeconds     int           `db:"retry_backoff_seconds"`
		CaptureResponseBody     bool          `db:"capture_response_body"`
		FollowRedirects         bool          `db:"follow_redirects"`
		VerifySSL               bool          `db:"verify_ssl"`
		RequestTimeoutSeconds   int           `db:"request_timeout_seconds"`
		ExpectedStatusCodesJSON pq.Int64Array `db:"expected_status_codes"`
		WebhookSecret           []byte        `db:"webhook_secret"`
		UserAgent               string        `db:"user_agent"`
		ContentType             string        `db:"content_type"`
		SecretHeaders           []byte        `db:"secret_headers"`
		WRCreatedAt             time.Time     `db:"wr_created_at"`
		WRUpdatedAt             time.Time     `db:"wr_updated_at"`
	}

	var rows []rowStruct
	err = r.conn.SelectContext(ctx, &rows, query, tenantID, namespace, event, labelsJSON)
	if err != nil {
		return nil, storage.Error(err)
	}

	var results []*SubscriptionWithWebhook
	for _, row := range rows {
		sub := &EventSubscription{
			ID:                row.ID,
			TenantID:          tenantID,
			WebhookID:         row.WebhookID,
			EventName:         row.EventName,
			Namespace:         row.Namespace,
			Method:            row.Method,
			TransformEnabled:  row.TransformEnabled,
			TransformTemplate: row.TransformTemplate,
			Timeout:           row.Timeout,
			CreatedAt:         row.CreatedAt,
			UpdatedAt:         row.UpdatedAt,
		}

		wh := &WebhookRegistration{
			ID:                    row.WRID,
			TenantID:              tenantID,
			Namespace:             row.WRNamespace,
			URL:                   row.URL,
			Timeout:               row.WRTimeout,
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
			SecretHeaders:         row.SecretHeaders,
			CreatedAt:             row.WRCreatedAt,
			UpdatedAt:             row.WRUpdatedAt,
		}

		if err := json.Unmarshal(row.HeadersJSON, &sub.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal subscription headers: %w", err)
		}
		if err := json.Unmarshal(row.LabelFiltersJSON, &sub.LabelFilters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal subscription label_filters: %w", err)
		}
		if err := json.Unmarshal(row.WRHeadersJSON, &wh.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal webhook headers: %w", err)
		}
		wh.ExpectedStatusCodes = row.ExpectedStatusCodesJSON

		results = append(results, &SubscriptionWithWebhook{
			Subscription: sub,
			Webhook:      wh,
		})
	}

	return results, nil
}
