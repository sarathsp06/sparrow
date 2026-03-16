package store

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// CreateDelivery creates a new webhook delivery record for tracking delivery attempts.
// tenantID is accepted for interface consistency; the delivery is implicitly tenant-scoped via webhook_id.
func (r *Repository) CreateDelivery(ctx context.Context, tenantID uuid.UUID, delivery *WebhookDelivery) error {
	query := `
		INSERT INTO webhook_deliveries (id, webhook_id, event_id, subscription_id, status, attempt_count, max_attempts, expires_at, response_code, response_body, error_message, request_body)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.ExecContext(ctx, query,
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

// UpdateDeliveryStatus records the outcome of a webhook delivery attempt.
func (r *Repository) UpdateDeliveryStatus(ctx context.Context, deliveryID uuid.UUID, status WebhookDeliveryStatus, responseCode int, responseBody, errorMessage, errorCategory string) error {
	now := time.Now()
	attemptIncrement := 0
	if status == StatusFailed || status == StatusSuccess || status == StatusExpired {
		attemptIncrement = 1
	}

	query := `
		UPDATE webhook_deliveries
		SET status = $2, last_attempted_at = $3, response_code = $4, response_body = $5, error_message = $6,
		    attempt_count = attempt_count + $7::integer, error_category = $8
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, deliveryID, status, now, responseCode, responseBody, errorMessage, attemptIncrement, errorCategory)
	return storage.Error(err)
}

// UpdateDeliveryRequestBody updates the request body for a delivery
func (r *Repository) UpdateDeliveryRequestBody(ctx context.Context, deliveryID uuid.UUID, requestBody string) error {
	query := `UPDATE webhook_deliveries SET request_body = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, deliveryID, requestBody)
	return storage.Error(err)
}

// GetDeliveryByID gets a delivery by ID, optionally filtered by namespace, within a tenant.
// When namespace is empty, looks up by delivery ID alone (still tenant-scoped).
func (r *Repository) GetDeliveryByID(ctx context.Context, tenantID uuid.UUID, deliveryID uuid.UUID, namespace string) (*WebhookDelivery, error) {
	var query string
	var args []any

	if namespace != "" {
		query = `
			SELECT wd.id, wd.webhook_id, wd.event_id, wd.subscription_id, wd.status, wd.attempt_count, wd.max_attempts,
			       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
			       wd.response_code, wd.response_body, wd.error_message, wd.request_body, wd.error_category
			FROM webhook_deliveries wd
			JOIN webhook_registrations wr ON wd.webhook_id = wr.id
			WHERE wd.id = $1 AND wr.tenant_id = $2 AND wr.namespace = $3
		`
		args = []any{deliveryID, tenantID, namespace}
	} else {
		query = `
			SELECT wd.id, wd.webhook_id, wd.event_id, wd.subscription_id, wd.status, wd.attempt_count, wd.max_attempts,
			       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
			       wd.response_code, wd.response_body, wd.error_message, wd.request_body, wd.error_category
			FROM webhook_deliveries wd
			JOIN webhook_registrations wr ON wd.webhook_id = wr.id
			WHERE wd.id = $1 AND wr.tenant_id = $2
		`
		args = []any{deliveryID, tenantID}
	}

	var d WebhookDelivery
	err := r.db.GetContext(ctx, &d, query, args...)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}

	return &d, nil
}

// GetDeliveriesByWebhookID retrieves webhook delivery records for a specific webhook within a tenant
func (r *Repository) GetDeliveriesByWebhookID(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error) {
	// First get total count
	countQuery := `
		SELECT COUNT(*)
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.webhook_id = $1 AND wr.tenant_id = $2 AND wr.namespace = $3
	`

	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, webhookID, tenantID, namespace)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Then get paginated results
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.subscription_id, wd.status, wd.attempt_count, wd.max_attempts,
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message, wd.request_body, wd.error_category
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.webhook_id = $1 AND wr.tenant_id = $2 AND wr.namespace = $3
		ORDER BY wd.created_at DESC
		LIMIT $4 OFFSET $5
	`

	var deliveries []*WebhookDelivery
	err = r.db.SelectContext(ctx, &deliveries, query, webhookID, tenantID, namespace, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return deliveries, totalCount, nil
}

// GetDeliveriesByEventPaginated retrieves webhook delivery records for a specific event within a tenant
func (r *Repository) GetDeliveriesByEventPaginated(ctx context.Context, tenantID uuid.UUID, eventID uuid.UUID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error) {
	// First get total count
	countQuery := `
		SELECT COUNT(*)
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.event_id = $1 AND wr.tenant_id = $2 AND wr.namespace = $3
	`

	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, eventID, tenantID, namespace)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Then get paginated results
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.subscription_id, wd.status, wd.attempt_count, wd.max_attempts,
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message, wd.request_body, wd.error_category
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.event_id = $1 AND wr.tenant_id = $2 AND wr.namespace = $3
		ORDER BY wd.created_at DESC
		LIMIT $4 OFFSET $5
	`

	var deliveries []*WebhookDelivery
	err = r.db.SelectContext(ctx, &deliveries, query, eventID, tenantID, namespace, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return deliveries, totalCount, nil
}

// ListDeliveriesPaginated retrieves webhook delivery records within a tenant, optionally filtered by namespace.
// When namespace is empty, returns deliveries across all namespaces for the tenant.
func (r *Repository) ListDeliveriesPaginated(ctx context.Context, tenantID uuid.UUID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error) {
	var countQuery, query string
	var args []any

	if namespace != "" {
		// First get total count
		countQuery = `
			SELECT COUNT(*)
			FROM webhook_deliveries wd
			JOIN webhook_registrations wr ON wd.webhook_id = wr.id
			WHERE wr.tenant_id = $1 AND wr.namespace = $2
		`
		args = []any{tenantID, namespace}

		var totalCount int
		err := r.db.GetContext(ctx, &totalCount, countQuery, args...)
		if err != nil {
			return nil, 0, storage.Error(err)
		}

		// Then get paginated results
		query = `
			SELECT wd.id, wd.webhook_id, wd.event_id, wd.subscription_id, wd.status, wd.attempt_count, wd.max_attempts,
			       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
			       wd.response_code, wd.response_body, wd.error_message, wd.request_body, wd.error_category
			FROM webhook_deliveries wd
			JOIN webhook_registrations wr ON wd.webhook_id = wr.id
			WHERE wr.tenant_id = $1 AND wr.namespace = $2
			ORDER BY wd.created_at DESC
			LIMIT $3 OFFSET $4
		`

		var deliveries []*WebhookDelivery
		err = r.db.SelectContext(ctx, &deliveries, query, tenantID, namespace, limit, offset)
		if err != nil {
			return nil, 0, storage.Error(err)
		}

		return deliveries, totalCount, nil
	}

	// No namespace filter - return all deliveries for the tenant
	countQuery = `
		SELECT COUNT(*)
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wr.tenant_id = $1
	`
	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, tenantID)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	query = `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.subscription_id, wd.status, wd.attempt_count, wd.max_attempts,
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message, wd.request_body, wd.error_category
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wr.tenant_id = $1
		ORDER BY wd.created_at DESC
		LIMIT $2 OFFSET $3
	`

	var deliveries []*WebhookDelivery
	err = r.db.SelectContext(ctx, &deliveries, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return deliveries, totalCount, nil
}

// GetRetriableDeliveries finds webhook deliveries eligible for retry attempts within a tenant.
func (r *Repository) GetRetriableDeliveries(ctx context.Context, tenantID uuid.UUID, webhookID uuid.UUID, namespace string, force bool) ([]*WebhookDelivery, error) {
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.status, wd.attempt_count, wd.max_attempts,
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message, wd.error_category
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.webhook_id = $1
		  AND wr.tenant_id = $2
		  AND wr.namespace = $3
		  AND ($4 IS TRUE OR wd.status IN ('failed', 'pending', 'retrying'))
		ORDER BY wd.created_at DESC
	`

	var deliveries []*WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, query, webhookID, tenantID, namespace, force)
	if err != nil {
		return nil, storage.Error(err)
	}

	return deliveries, nil
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
		    error_category = ''
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, deliveryID)
	return storage.Error(err)
}
