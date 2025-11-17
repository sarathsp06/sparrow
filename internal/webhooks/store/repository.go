package store

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/lib/pq"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// Repository provides data access layer for webhook operations.
// It handles CRUD operations for webhooks, events, deliveries, and health tracking.
// All database interactions are performed through the storage.DB interface for testability.
type Repository struct {
	db storage.DB
}

// UpdateWebhookHealthState records a webhook delivery outcome and updates health metrics.
// For successful deliveries, it resets consecutive failures to 0 and updates last success timestamp.
// For failed deliveries, it increments consecutive failures and updates last failure timestamp.
// After updating health state, it recalculates the overall webhook health status (healthy/degraded/unhealthy).
// This function performs upsert operations to handle both new webhooks and existing ones.
func (r *Repository) UpdateWebhookHealthState(ctx context.Context, webhookID string, success bool, eventTimestamp time.Time) error {
	// Upsert health state
	var err error
	var consecutiveFailures int
	if success {
		consecutiveFailures = 0
	} else {
		// Get current consecutive failures
		err = r.db.GetContext(ctx, &consecutiveFailures, `SELECT COALESCE(consecutive_failures, 0) FROM webhook_health_state WHERE webhook_id = $1`, webhookID)
		if err != nil && !storage.IsNotFound(storage.Error(err)) {
			return storage.Error(err)
		}
		consecutiveFailures++
	}

	var lastSuccessAt, lastFailureAt *time.Time
	if success {
		lastSuccessAt = &eventTimestamp
	} else {
		lastFailureAt = &eventTimestamp
	}
	// Upsert health state
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO webhook_health_state (webhook_id, consecutive_failures, last_success_at, last_failure_at, last_event_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (webhook_id) DO UPDATE SET
			consecutive_failures = $2,
			last_success_at = COALESCE($3, webhook_health_state.last_success_at),
			last_failure_at = COALESCE($4, webhook_health_state.last_failure_at),
			last_event_at = $5,
			updated_at = NOW()
	`,
		webhookID,
		consecutiveFailures,
		lastSuccessAt,
		lastFailureAt,
		eventTimestamp,
	)
	if err != nil {
		return storage.Error(err)
	}

	// Calculate health status
	healthStatus, err := r.CalculateWebhookHealth(ctx, webhookID, 24)
	if err != nil {
		return storage.Error(err)
	}

	// Update webhook_registrations health field
	_, err = r.db.ExecContext(ctx, `UPDATE webhook_registrations SET health = $1, updated_at = NOW() WHERE id = $2`, healthStatus, webhookID)
	return storage.Error(err)
}

// CalculateWebhookHealth determines webhook health status based on delivery patterns.
// Health calculation considers: recent success rate within lookbackHours window, consecutive failures,
// and minimum event threshold for statistical significance.
// Returns: "healthy" (>90% success, <5 failures), "degraded" (80-90% success),
//
//	"unhealthy" (<80% success or >=5 consecutive failures), "unknown" (insufficient data).
func (r *Repository) CalculateWebhookHealth(ctx context.Context, webhookID string, lookbackHours int) (string, error) {
	// Get recent event statistics
	query := `
		SELECT COUNT(*), COALESCE(AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END), 0)
		FROM webhook_health_events
		WHERE webhook_id = $1 AND timestamp >= NOW() - INTERVAL '1 hour' * $2
	`
	var result struct {
		EventsCount int     `db:"count"`
		SuccessRate float64 `db:"coalesce"`
	}
	err := r.db.GetContext(ctx, &result, query, webhookID, lookbackHours)
	if err != nil {
		return "unknown", storage.Error(err)
	}
	recentEventsCount := result.EventsCount
	recentSuccessRate := result.SuccessRate

	// Get consecutive failures
	var consecutiveFailuresCount int
	err = r.db.GetContext(ctx, &consecutiveFailuresCount, `SELECT COALESCE(consecutive_failures, 0) FROM webhook_health_state WHERE webhook_id = $1`, webhookID)
	if err != nil && !storage.IsNotFound(storage.Error(err)) {
		return "", storage.Error(err)
	}

	// Calculate health status
	switch {
	case recentEventsCount == 0:
		return "unknown", nil
	case consecutiveFailuresCount >= 5:
		return "unhealthy", nil
	case recentSuccessRate < 0.8 && recentEventsCount >= 10:
		return "unhealthy", nil
	case recentSuccessRate < 0.9 && recentEventsCount >= 5:
		return "degraded", nil
	case recentSuccessRate >= 0.9 && recentEventsCount >= 3:
		return "healthy", nil
	default:
		return "unknown", nil
	}
}

// StoreEventTx persists an event record within an existing database transaction.
// Automatically generates UUID if event.ID is empty and sets created_at/expires_at timestamps.
// The expires_at is calculated from TTL (time-to-live) in seconds from creation time.
// This transactional version ensures atomic operations when creating events with related deliveries.
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
	return storage.Error(err)
}

// GetWebhooksByEventTx retrieves all active webhooks subscribed to a specific event within a transaction.
// Uses JSONB ? operator to efficiently query webhooks where the events array contains the specified event.
// Only returns webhooks that are active=true and match the namespace for tenant isolation.
// Includes complete webhook configuration including HTTP settings for delivery customization.
func (r *Repository) GetWebhooksByEventTx(ctx context.Context, tx pgx.Tx, namespace, event string) ([]*WebhookRegistration, error) {
	query := `
	       SELECT id, namespace, events, url, headers, timeout, active, description, health,
	              max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
	              verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
	              user_agent, content_type, created_at, updated_at
	       FROM webhook_registrations 
	       WHERE namespace = $1 AND active = true AND events::jsonb ? $2
	       `

	rows, err := tx.Query(ctx, query, namespace, event)
	if err != nil {
		return nil, storage.Error(err)
	}
	defer rows.Close()

	var webhooks []*WebhookRegistration
	for rows.Next() {
		var wh WebhookRegistration
		var headersJSON []byte
		var eventsJSON []byte
		var expectedStatusCodesJSON []byte

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
			&wh.MaxRetries,
			&wh.RetryBackoffSeconds,
			&wh.CaptureResponseBody,
			&wh.FollowRedirects,
			&wh.VerifySSL,
			&wh.RequestTimeoutSeconds,
			&expectedStatusCodesJSON,
			&wh.WebhookSecret,
			&wh.UserAgent,
			&wh.ContentType,
			&wh.CreatedAt,
			&wh.UpdatedAt,
		)
		if err != nil {
			return nil, storage.Error(err)
		}

		if err := json.Unmarshal(headersJSON, &wh.Headers); err != nil {
			return nil, fmt.Errorf("failed to unmarshal headers: %w", err)
		}

		if err := json.Unmarshal(eventsJSON, &wh.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}

		if err := json.Unmarshal(expectedStatusCodesJSON, &wh.ExpectedStatusCodes); err != nil {
			return nil, fmt.Errorf("failed to unmarshal expected status codes: %w", err)
		}

		if err := json.Unmarshal(eventsJSON, &wh.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
		}

		webhooks = append(webhooks, &wh)
	}

	return webhooks, nil
}

// NewRepository creates a new Repository instance with the provided database connection.
// The storage.DB interface allows for dependency injection and easier testing with mock implementations.
func NewRepository(db storage.DB) *Repository {
	return &Repository{
		db: db,
	}
}

// RegisterWebhook creates a new webhook registration with duplicate prevention.
// Checks for existing webhook with same namespace+URL combination to prevent duplicates.
// Generates a new UUID v4 for the webhook ID and initializes health status as "unknown".
// Sets created_at and updated_at timestamps automatically.
// Returns nil if webhook already exists (idempotent operation) or on successful creation.
func (r *Repository) RegisterWebhook(ctx context.Context, registration *WebhookRegistration) error {
	// Check for existing webhook with same namespace and url
	checkQuery := `SELECT id FROM webhook_registrations WHERE namespace = $1 AND url = $2 LIMIT 1`
	var existingID string
	err := r.db.GetContext(ctx, &existingID, checkQuery, registration.Namespace, registration.URL)
	if err == nil && existingID != "" {
		// Already exists, treat as success
		return nil
	} else if err != nil && !storage.IsNotFound(storage.Error(err)) {
		// DB error
		return storage.Error(err)
	}

	registration.ID = uuid.New().String()
	registration.Health = HealthUnknown // New webhooks start with unknown health

	query := `
		INSERT INTO webhook_registrations (
			id, namespace, events, url, headers, timeout, active, description, health,
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
		registration.Namespace,
		pq.Array(registration.Events),
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
func (r *Repository) UnregisterWebhook(ctx context.Context, webhookID string) error {
	query := `DELETE FROM webhook_registrations WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, webhookID)
	return storage.Error(err)
}

// GetWebhooksByEvent finds all active webhooks subscribed to a specific event in a namespace.
// Uses PostgreSQL array containment operator (= ANY) for efficient event subscription lookup.
// Only returns webhooks with active=true to exclude paused or disabled webhooks.
// Results include complete webhook configuration for immediate delivery processing.
func (r *Repository) GetWebhooksByEvent(ctx context.Context, namespace, event string) ([]*WebhookRegistration, error) {
	query := `
		SELECT id, namespace, events, url, headers, timeout, active, description, health,
		       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
		       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
		       user_agent, content_type, created_at, updated_at
		FROM webhook_registrations 
		WHERE namespace = $1 AND active = true AND $2 = ANY(events)
	`
	var webhooks []*WebhookRegistration

	err := r.db.SelectContext(ctx, &webhooks, query, namespace, event)
	if err != nil {
		return nil, storage.Error(err)
	}
	return webhooks, nil
}

// ListWebhooks retrieves webhooks for a namespace with optional active status filtering.
// When activeOnly=true, returns only webhooks with active=true (excludes paused webhooks).
// When activeOnly=false, returns all webhooks regardless of active status for management purposes.
// Results are ordered by created_at DESC to show newest webhooks first.
func (r *Repository) ListWebhooks(ctx context.Context, namespace string, activeOnly bool) ([]*WebhookRegistration, error) {
	query := `
		SELECT id, namespace, events, url, headers, timeout, active, description, health,
		       max_retries, retry_backoff_seconds, capture_response_body, follow_redirects,
		       verify_ssl, request_timeout_seconds, expected_status_codes, webhook_secret,
		       user_agent, content_type, created_at, updated_at
		FROM webhook_registrations 
		WHERE namespace = $1 AND ($2::boolean IS FALSE OR active = true)
		ORDER BY created_at DESC
	`

	var webhooks []*WebhookRegistration
	err := r.db.SelectContext(ctx, &webhooks, query, namespace, activeOnly)
	if err != nil {
		return nil, storage.Error(err)
	}

	return webhooks, nil
}

// StoreEvent persists an event record with automatic ID generation and timestamp management.
// Generates UUID v4 for event.ID if not provided and sets created_at to current time.
// Calculates expires_at based on TTL (time-to-live) seconds from creation time.
// Marshals metadata map to JSON for database storage in JSONB column.
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

	_, err = r.db.ExecContext(ctx, query,
		event.ID,
		event.Namespace,
		event.Event,
		event.Payload,
		event.TTL,
		metadataJSON,
		event.CreatedAt,
		event.ExpiresAt,
	)
	return storage.Error(err)
}

// CreateDelivery creates a new webhook delivery record for tracking delivery attempts.
// Records initial delivery state including webhook_id, event_id, retry configuration,
// and expiration time for delivery attempts. Used by the job queue system to track
// webhook delivery lifecycle from creation through completion or failure.
func (r *Repository) CreateDelivery(ctx context.Context, delivery *WebhookDelivery) error {
	query := `
		INSERT INTO webhook_deliveries (id, webhook_id, event_id, status, attempt_count, max_attempts, expires_at, response_code, response_body, error_message, request_body)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.ExecContext(ctx, query,
		delivery.ID,
		delivery.WebhookID,
		delivery.EventID,
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
func (r *Repository) UpdateDeliveryStatus(ctx context.Context, deliveryID string, status WebhookDeliveryStatus, responseCode int, responseBody, errorMessage string) error {
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
func (r *Repository) UpdateDeliveryRequestBody(ctx context.Context, deliveryID string, requestBody string) error {
	query := `UPDATE webhook_deliveries SET request_body = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, deliveryID, requestBody)
	return storage.Error(err)
}

// GetDeliveriesByWebhook returns deliveries for a specific webhook
func (r *Repository) GetDeliveriesByWebhook(ctx context.Context, webhookID string) ([]*WebhookDelivery, error) {
	query := `
		SELECT id, webhook_id, event_id, status, attempt_count, max_attempts, 
		       created_at, last_attempted_at, next_retry_at, expires_at,
		       response_code, response_body, error_message, request_body
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
		       response_code, response_body, error_message, request_body
		FROM webhook_deliveries 
		WHERE event_id = $1 
		ORDER BY created_at DESC
	`

	return r.getDeliveries(ctx, query, eventID)
}

func (r *Repository) getDeliveries(ctx context.Context, query string, arg interface{}) ([]*WebhookDelivery, error) {
	var deliveries []*WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, query, arg)
	if err != nil {
		return nil, storage.Error(err)
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
		SELECT id, namespace, events, url, headers, timeout, active, description, health,
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

	query := `
		UPDATE webhook_registrations 
		SET events = :events, url = :url, headers = :headers, timeout = :timeout, active = :active, 
		    description = :description, updated_at = NOW()
		WHERE id = :id AND namespace = :namespace
	`

	_, err := r.db.NamedExecContext(ctx, query, webhook)
	return storage.Error(err)
}

// GetDeliveryByID gets a delivery by ID and namespace
func (r *Repository) GetDeliveryByID(ctx context.Context, deliveryID, namespace string) (*WebhookDelivery, error) {
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.status, wd.attempt_count, wd.max_attempts, 
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
	err := r.db.GetContext(ctx, &totalCount, countQuery, webhookID, namespace)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Then get paginated results
	query := `
		SELECT wd.id, wd.webhook_id, wd.event_id, wd.status, wd.attempt_count, wd.max_attempts, 
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

// RegisterEvent registers a new event type
func (r *Repository) RegisterEvent(ctx context.Context, event *EventRegistration) error {
	event.ID = uuid.New().String()
	query := `
		INSERT INTO event_registrations (
			id, name, description, schema, metadata, active
		) VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.db.ExecContext(ctx, query,
		event.ID,
		event.Name,
		event.Description,
		event.Schema,
		event.Metadata,
		event.Active,
		event.CreatedAt,
		event.UpdatedAt,
	)
	return storage.Error(err)
}

// GetEventByName gets an event registration by name
func (r *Repository) GetEventByName(ctx context.Context, eventName string) (*EventRegistration, error) {
	query := `
		SELECT id, name, description, schema, metadata, active, created_at, updated_at
		FROM event_registrations 
		WHERE name = $1
	`
	var event EventRegistration
	err := r.db.GetContext(ctx, &event, query, eventName)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &event, nil
}

// ListEvents returns all registered events
func (r *Repository) ListEvents(ctx context.Context, activeOnly bool) ([]*EventRegistration, error) {
	query := `
		SELECT id, name, description, schema, metadata, active, created_at, updated_at
		FROM event_registrations
		WHERE ( $1::boolean IS FALSE OR active = true )
		ORDER BY name ASC
	`
	var events []*EventRegistration
	err := r.db.SelectContext(ctx, &events, query, activeOnly)
	if err != nil {
		return nil, storage.Error(err)
	}
	return events, nil
}

// UpdateEvent updates an event registration
func (r *Repository) UpdateEvent(ctx context.Context, event *EventRegistration) error {
	query := `
		UPDATE event_registrations 
		SET description = $2, schema = $3, metadata = $4, active = $5
		WHERE name = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		event.Name,
		event.Description,
		event.Schema,
		event.Metadata,
		event.Active,
		event.UpdatedAt,
	)
	return storage.Error(err)
}

// DeleteEvent deletes an event registration
func (r *Repository) DeleteEvent(ctx context.Context, eventName string) error {
	query := `DELETE FROM event_registrations WHERE name = $1`
	_, err := r.db.ExecContext(ctx, query, eventName)
	return storage.Error(err)
}

// RecordWebhookHealthEvent creates a health tracking record for analytics and monitoring.
// Captures delivery outcome, response time metrics, HTTP status codes, and error details.
// These events feed into webhook health calculations and performance analytics.
// Timestamp is set to NOW() for accurate time-series data collection.
func (r *Repository) RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID string, success bool, responseTime, responseCode int, errorMessage string) error {
	query := `
		INSERT INTO webhook_health_events (webhook_id, delivery_id, success, response_time, response_code, error_message, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
	`
	_, err := r.db.ExecContext(ctx, query, webhookID, deliveryID, success, responseTime, responseCode, errorMessage)
	if err != nil {
		return fmt.Errorf("failed to record health event: %w", err)
	}

	return nil
}

// GetWebhookHealthState retrieves the current health tracking state for a webhook.
// Returns metrics including consecutive failure count, timestamps of last success/failure,
// and when the last delivery event occurred. Used for health status calculations
// and determining when webhooks should be automatically disabled.
func (r *Repository) GetWebhookHealthState(ctx context.Context, webhookID string) (*WebhookHealthMetrics, error) {
	query := `
		SELECT id, webhook_id, consecutive_failures, last_success_at, last_failure_at, 
		       last_event_at, created_at, updated_at
		FROM webhook_health_state
		WHERE webhook_id = $1
	`

	var state WebhookHealthMetrics
	err := r.db.GetContext(ctx, &state, query, webhookID)
	if err != nil {
		return nil, storage.Error(err)
	}

	return &state, nil
}

// GetWebhookHealthSummary provides aggregated performance metrics over a time window.
// First attempts to retrieve pre-computed summaries from webhook_health_summaries table.
// If no pre-computed data exists, calculates metrics on-the-fly from webhook_health_events.
// Includes delivery counts, success rates, and response time percentiles (avg, min, max, p95).
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
	err := r.db.GetContext(ctx, &summary, query, webhookID, hours)
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

	err = r.db.GetContext(ctx, &summary, realTimeQuery, webhookID, hours)
	if err != nil {
		return nil, storage.Error(err)
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

	var events []*WebhookHealthEvent
	err := r.db.SelectContext(ctx, &events, query, webhookID, hours)
	if err != nil {
		return nil, storage.Error(err)
	}

	return events, nil
}

// AggregateHealthSummaries runs the aggregation function for health summaries
func (r *Repository) AggregateHealthSummaries(ctx context.Context) (int, error) {
	query := `SELECT aggregate_webhook_health_hourly()`

	var processedCount int
	err := r.db.GetContext(ctx, &processedCount, query)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate health summaries: %w", err)
	}

	return processedCount, nil
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
		ID                    string    `db:"id"`
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

		if err := json.Unmarshal(row.EventsJSON, &webhook.Events); err != nil {
			return nil, fmt.Errorf("failed to unmarshal events: %w", err)
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

// GetHealthSummary returns a summary of webhook health across all namespaces
func (r *Repository) GetHealthSummary(ctx context.Context) (map[WebhookHealth]int, error) {
	query := `
		SELECT health, COUNT(*) as count
		FROM webhook_registrations
		GROUP BY health
	`

	type healthCount struct {
		Health string `db:"health"`
		Count  int    `db:"count"`
	}

	var results []healthCount
	err := r.db.SelectContext(ctx, &results, query)
	if err != nil {
		return nil, storage.Error(err)
	}

	summary := make(map[WebhookHealth]int)
	for _, result := range results {
		summary[WebhookHealth(result.Health)] = result.Count
	}

	return summary, nil
}

// GetRetriableDeliveries finds webhook deliveries eligible for retry attempts.
// When force=false, only returns deliveries with status 'failed', 'pending', or 'retrying'.
// When force=true, returns all deliveries regardless of status for administrative retry operations.
// Results are namespace-isolated and ordered by creation time for consistent processing.
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

	var deliveries []*WebhookDelivery
	err := r.db.SelectContext(ctx, &deliveries, query, args...)
	if err != nil {
		return nil, storage.Error(err)
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

	_, err := r.db.ExecContext(ctx, query, deliveryID)
	return storage.Error(err)
}

// GetEventByID gets an event record by ID
func (r *Repository) GetEventByID(ctx context.Context, eventID string) (*EventRecord, error) {
	query := `
		SELECT id, namespace, event, payload, ttl, metadata, created_at, expires_at
		FROM event_records
		WHERE id = $1
	`

	var eventRow EventRecord

	err := r.db.GetContext(ctx, &eventRow, query, eventID)
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &eventRow, nil
}

// ListEventReports gets event records in descending order by creation time
func (r *Repository) ListEventReports(ctx context.Context, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error) {
	// Build base query
	baseQuery := `
		SELECT 
			id, namespace, event, payload, ttl, metadata, created_at, expires_at
		FROM event_records 
		WHERE namespace = $1
	`

	countQuery := `
		SELECT COUNT(*) 
		FROM event_records 
		WHERE namespace = $1
	`

	args := []interface{}{namespace}

	// Add event name filter if provided
	if eventName != nil && *eventName != "" {
		baseQuery += ` AND event = $2`
		countQuery += ` AND event = $2`
		args = append(args, *eventName)
	}

	// Add ordering and pagination
	baseQuery += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, limit, offset)

	// Execute main query
	var eventRows []EventRecord

	err := r.db.SelectContext(ctx, &eventRows, baseQuery, args...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Get total count
	var totalCount int
	countArgs := args[:len(args)-2] // Remove LIMIT and OFFSET
	err = r.db.GetContext(ctx, &totalCount, countQuery, countArgs...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Convert to EventReportWithStats format
	var events []*EventReportWithStats
	for _, row := range eventRows {
		event := &EventReportWithStats{
			EventRecord: row,
			// Delivery stats can be loaded separately if needed
			WebhookCount:         0,
			SuccessfulDeliveries: 0,
			FailedDeliveries:     0,
			PendingDeliveries:    0,
		}

		events = append(events, event)
	}

	return events, totalCount, nil
}

// ListEventReportsWithStats retrieves event records enriched with delivery statistics.
// Joins event_records with webhook_deliveries and webhook_health_events to calculate:
// - webhook_count: number of unique webhooks that received the event
// - successful/failed/pending delivery counts based on health event outcomes
// Supports optional event name filtering and pagination. Returns total count for UI pagination.
func (r *Repository) ListEventReportsWithStats(ctx context.Context, namespace string, eventName *string, limit, offset int) ([]*EventReportWithStats, int, error) {
	// Build base query with delivery stats from health events
	baseQuery := `
		SELECT 
			er.id, er.namespace, er.event, er.payload, er.ttl, er.metadata, er.created_at, er.expires_at,
			COALESCE(ds.webhook_count, 0) as webhook_count,
			COALESCE(ds.successful_deliveries, 0) as successful_deliveries,
			COALESCE(ds.failed_deliveries, 0) as failed_deliveries,
			COALESCE(ds.pending_deliveries, 0) as pending_deliveries
		FROM event_records er
		LEFT JOIN (
			SELECT 
				wd.event_id,
				COUNT(DISTINCT wd.webhook_id) as webhook_count,
				SUM(CASE WHEN wh.success = true THEN 1 ELSE 0 END) as successful_deliveries,
				SUM(CASE WHEN wh.success = false THEN 1 ELSE 0 END) as failed_deliveries,
				COUNT(CASE WHEN wd.status IN ('pending', 'sending', 'retrying') THEN 1 END) as pending_deliveries
			FROM webhook_deliveries wd
			LEFT JOIN webhook_health_events wh ON wd.id = wh.delivery_id::text
			GROUP BY wd.event_id
		) ds ON er.id = ds.event_id
		WHERE er.namespace = $1
	`

	countQuery := `
		SELECT COUNT(*) 
		FROM event_records 
		WHERE namespace = $1
	`

	args := []interface{}{namespace}

	// Add event name filter if provided
	if eventName != nil && *eventName != "" {
		baseQuery += ` AND er.event = $2`
		countQuery += ` AND event = $2`
		args = append(args, *eventName)
	}

	// Add ordering and pagination
	baseQuery += ` ORDER BY er.created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, limit, offset)

	// Execute main query
	var events []*EventReportWithStats
	err := r.db.SelectContext(ctx, &events, baseQuery, args...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	// Get total count
	var totalCount int
	countArgs := args[:len(args)-2] // Remove LIMIT and OFFSET
	err = r.db.GetContext(ctx, &totalCount, countQuery, countArgs...)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	return events, totalCount, nil
}

// GetEventDeliveryStats gets delivery statistics for a specific event using health events
func (r *Repository) GetEventDeliveryStats(ctx context.Context, eventID string) (int32, int32, int32, int32, error) {
	query := `
		SELECT 
			COUNT(DISTINCT wd.webhook_id) as webhook_count,
			SUM(CASE WHEN wh.success = true THEN 1 ELSE 0 END) as successful_deliveries,
			SUM(CASE WHEN wh.success = false THEN 1 ELSE 0 END) as failed_deliveries,
			COUNT(CASE WHEN wd.status IN ('pending', 'sending', 'retrying') THEN 1 END) as pending_deliveries
		FROM webhook_deliveries wd
		LEFT JOIN webhook_health_events wh ON wd.id = wh.delivery_id
		WHERE wd.event_id = $1
	`

	var result struct {
		WebhookCount         int32 `db:"webhook_count"`
		SuccessfulDeliveries int32 `db:"successful_deliveries"`
		FailedDeliveries     int32 `db:"failed_deliveries"`
		PendingDeliveries    int32 `db:"pending_deliveries"`
	}

	err := r.db.GetContext(ctx, &result, query, eventID)
	if err != nil {
		return 0, 0, 0, 0, storage.Error(err)
	}

	return result.WebhookCount, result.SuccessfulDeliveries, result.FailedDeliveries, result.PendingDeliveries, nil
}
