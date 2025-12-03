package store

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// CreateDelivery creates a new webhook delivery record for tracking delivery attempts.
// Records initial delivery state including webhook_id, event_id, retry configuration,
// and expiration time for delivery attempts. Used by the job queue system to track
// webhook delivery lifecycle from creation through completion or failure.
func (r *Repository) CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error {
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
// Updates status (pending/success/failed/expired), captures HTTP response details,
// and increments attempt_count for failed/success/expired statuses.
// Sets last_attempted_at timestamp and preserves complete response for audit trail.
func (r *Repository) UpdateDeliveryStatus(ctx context.Context, deliveryID uuid.UUID, status WebhookDeliveryStatus, responseCode int, responseBody, errorMessage string) error {
	now := time.Now()
	var attemptIncrement int = 0
	if status == StatusFailed || status == StatusSuccess || status == StatusExpired {
		attemptIncrement = 1
	}

	query := `
		UPDATE webhook_deliveries 
		SET status = $2, last_attempted_at = $3, response_code = $4, response_body = $5, error_message = $6,
		    attempt_count = attempt_count + $7::integer
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, deliveryID, status, now, responseCode, responseBody, errorMessage, attemptIncrement)
	return storage.Error(err)
}

// UpdateDeliveryRequestBody updates the request body for a delivery
func (r *Repository) UpdateDeliveryRequestBody(ctx context.Context, deliveryID uuid.UUID, requestBody string) error {
	query := `UPDATE webhook_deliveries SET request_body = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, deliveryID, requestBody)
	return storage.Error(err)
}

// GetDeliveriesByWebhook returns deliveries for a specific webhook
func (r *Repository) GetDeliveriesByWebhook(ctx context.Context, webhookID uuid.UUID) ([]*WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event_id, subscription_id, status, attempt_count, max_attempts, 
		       created_at, last_attempted_at, next_retry_at, expires_at,
		       response_code, response_body, error_message, request_body
		FROM webhook_deliveries 
		WHERE webhook_id = $1 
		ORDER BY created_at DESC
	`

	var deliveries []*WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, query, webhookID)
	if err != nil {
		return nil, storage.Error(err)
	}

	return deliveries, nil
}

// GetDeliveriesByEvent returns deliveries for a specific event
func (r *Repository) GetDeliveriesByEvent(ctx context.Context, eventID uuid.UUID) ([]*WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event_id, subscription_id, status, attempt_count, max_attempts, 
		       created_at, last_attempted_at, next_retry_at, expires_at,
		       response_code, response_body, error_message, request_body
		FROM webhook_deliveries 
		WHERE event_id = $1 
		ORDER BY created_at DESC
	`

	var deliveries []*WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, query, eventID)
	if err != nil {
		return nil, storage.Error(err)
	}

	return deliveries, nil
}

// GetDeliveryByID gets a delivery by ID and namespace
func (r *Repository) GetDeliveryByID(ctx context.Context, deliveryID uuid.UUID, namespace string) (*WebhookDelivery, error) {
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.subscription_id, wd.status, wd.attempt_count, wd.max_attempts, 
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message, wd.request_body
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.id = $1 AND wr.namespace = $2
	`

	var d WebhookDelivery
	err := r.db.GetContext(ctx, &d, query, deliveryID, namespace)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}

	return &d, nil
}

// GetDeliveriesByWebhookID retrieves webhook delivery records for a specific webhook
// Supports pagination via limit and offset parameters. Returns a total count
// for UI pagination and the paginated results.
func (r *Repository) GetDeliveriesByWebhookID(ctx context.Context, webhookID uuid.UUID, namespace string, limit, offset int) ([]*WebhookDelivery, int, error) {
	// First get total count
	countQuery := `
		SELECT COUNT(*)
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.webhook_id = $1 AND wr.namespace = $2
	`

	var totalCount int
	err := r.db.GetContext(ctx, &totalCount, countQuery, webhookID, namespace)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Then get paginated results
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.subscription_id, wd.status, wd.attempt_count, wd.max_attempts, 
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message, wd.request_body
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.webhook_id = $1 AND wr.namespace = $2
		ORDER BY wd.created_at DESC
		LIMIT $3 OFFSET $4
	`

	var deliveries []*WebhookDelivery
	err = r.db.SelectContext(ctx, &deliveries, query, webhookID, namespace, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return deliveries, totalCount, nil
}

// GetRetriableDeliveries finds webhook deliveries eligible for retry attempts.
// When force=false, only returns deliveries with status 'failed', 'pending', or 'retrying'.
// When force=true, returns all deliveries regardless of status for administrative retry operations.
// Results are namespace-isolated and ordered by creation time for consistent processing.
func (r *Repository) GetRetriableDeliveries(ctx context.Context, webhookID uuid.UUID, namespace string, force bool) ([]*WebhookDelivery, error) {
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.status, wd.attempt_count, wd.max_attempts, 
		       wd.created_at, wd.last_attempted_at, wd.next_retry_at, wd.expires_at,
		       wd.response_code, wd.response_body, wd.error_message
		FROM webhook_deliveries wd
		JOIN webhook_registrations wr ON wd.webhook_id = wr.id
		WHERE wd.webhook_id = $1 
		  AND wr.namespace = $2
		  AND ($3 IS TRUE OR wd.status IN ('failed', 'pending', 'retrying'))
		ORDER BY wd.created_at DESC
	`

	var deliveries []*WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, query, webhookID, namespace, force)
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
		    error_message = ''
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query, deliveryID)
	return storage.Error(err)
}
