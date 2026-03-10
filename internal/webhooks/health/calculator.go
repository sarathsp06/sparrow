package health

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// WebhookHealthStatus represents the health status of a webhook
type WebhookHealthStatus string

const (
	StatusHealthy   WebhookHealthStatus = "healthy"
	StatusDegraded  WebhookHealthStatus = "degraded"
	StatusUnhealthy WebhookHealthStatus = "unhealthy"
	StatusUnknown   WebhookHealthStatus = "unknown"
)

// WebhookHealthEvent represents a single health event for a webhook
type WebhookHealthEvent struct {
	ID            uuid.UUID `db:"id"`
	WebhookID     string    `db:"webhook_id"`
	DeliveryID    uuid.UUID `db:"delivery_id"`
	Success       bool      `db:"success"`
	ResponseTime  int       `db:"response_time"` // milliseconds
	ResponseCode  int       `db:"response_code"`
	ErrorMessage  string    `db:"error_message"`
	ErrorCategory string    `db:"error_category"` // success, client_error, server_error, timeout, dns_error, tls_error, connection_refused, network_error, unknown
	Timestamp     time.Time `db:"timestamp"`
}

// WebhookHealthState represents the current health state of a webhook
type WebhookHealthState struct {
	ID                  uuid.UUID  `db:"id"`
	WebhookID           string     `db:"webhook_id"`
	ConsecutiveFailures int        `db:"consecutive_failures"`
	LastSuccessAt       *time.Time `db:"last_success_at"`
	LastFailureAt       *time.Time `db:"last_failure_at"`
	LastEventAt         *time.Time `db:"last_event_at"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}

// WebhookHealthSummary represents aggregated health metrics for a time window
type WebhookHealthSummary struct {
	ID                   uuid.UUID `db:"id"`
	WebhookID            string    `db:"webhook_id"`
	WindowStart          time.Time `db:"window_start"`
	WindowEnd            time.Time `db:"window_end"`
	TotalDeliveries      int       `db:"total_deliveries"`
	SuccessfulDeliveries int       `db:"successful_deliveries"`
	FailedDeliveries     int       `db:"failed_deliveries"`
	SuccessRate          float64   `db:"success_rate"`
	AvgResponseTime      int       `db:"avg_response_time"`
	MinResponseTime      int       `db:"min_response_time"`
	MaxResponseTime      int       `db:"max_response_time"`
	P95ResponseTime      int       `db:"p95_response_time"`
	CreatedAt            time.Time `db:"created_at"`
	UpdatedAt            time.Time `db:"updated_at"`
}

// HealthCalculator handles webhook health calculations and state management
type HealthCalculator struct {
	db     *sqlx.DB
	logger *slog.Logger
}

// NewHealthCalculator creates a new health calculator instance
func NewHealthCalculator(db *sqlx.DB, logger *slog.Logger) *HealthCalculator {
	return &HealthCalculator{
		db:     db,
		logger: logger,
	}
}

// RecordHealthEvent records a new webhook health event and updates health state
func (hc *HealthCalculator) RecordHealthEvent(ctx context.Context, event *WebhookHealthEvent) error {
	tx, err := hc.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Insert health event
	if err := hc.insertHealthEvent(ctx, tx, event); err != nil {
		return fmt.Errorf("failed to insert health event: %w", err)
	}

	// Update health state
	if err := hc.updateHealthState(ctx, tx, event); err != nil {
		return fmt.Errorf("failed to update health state: %w", err)
	}

	// Calculate and update webhook health status
	newStatus, err := hc.calculateWebhookHealth(ctx, tx, event.WebhookID, 24)
	if err != nil {
		return fmt.Errorf("failed to calculate webhook health: %w", err)
	}

	if err := hc.updateWebhookHealth(ctx, tx, event.WebhookID, newStatus); err != nil {
		return fmt.Errorf("failed to update webhook health: %w", err)
	}

	return tx.Commit()
}

// insertHealthEvent inserts a new health event record
func (hc *HealthCalculator) insertHealthEvent(ctx context.Context, tx *sqlx.Tx, event *WebhookHealthEvent) error {
	query := `
		INSERT INTO webhook_health_events (id, webhook_id, delivery_id, success, response_time, response_code, error_message, error_category, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	if event.ErrorCategory == "" {
		if event.Success {
			event.ErrorCategory = "success"
		} else {
			event.ErrorCategory = "unknown"
		}
	}

	_, err := tx.ExecContext(ctx, query,
		event.ID, event.WebhookID, event.DeliveryID, event.Success,
		event.ResponseTime, event.ResponseCode, event.ErrorMessage, event.ErrorCategory, event.Timestamp)
	return err
}

// updateHealthState updates or creates the health state for a webhook
func (hc *HealthCalculator) updateHealthState(ctx context.Context, tx *sqlx.Tx, event *WebhookHealthEvent) error {
	query := `
		INSERT INTO webhook_health_state (webhook_id, consecutive_failures, last_success_at, last_failure_at, last_event_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW())
		ON CONFLICT (webhook_id) DO UPDATE SET
			consecutive_failures = CASE 
				WHEN $2 = 0 THEN 0 
				ELSE webhook_health_state.consecutive_failures + 1 
			END,
			last_success_at = CASE 
				WHEN $2 = 0 THEN $3 
				ELSE webhook_health_state.last_success_at 
			END,
			last_failure_at = CASE 
				WHEN $2 != 0 THEN $4 
				ELSE webhook_health_state.last_failure_at 
			END,
			last_event_at = $5,
			updated_at = NOW()`

	var consecutiveFailures int
	var lastSuccessAt, lastFailureAt *time.Time

	if event.Success {
		consecutiveFailures = 0
		lastSuccessAt = &event.Timestamp
	} else {
		consecutiveFailures = 1 // Will be incremented by the query if there are existing failures
		lastFailureAt = &event.Timestamp
	}

	_, err := tx.ExecContext(ctx, query, event.WebhookID, consecutiveFailures, lastSuccessAt, lastFailureAt, event.Timestamp)
	return err
}

// calculateWebhookHealth calculates the health status for a webhook based on recent events
func (hc *HealthCalculator) calculateWebhookHealth(ctx context.Context, tx *sqlx.Tx, webhookID string, lookbackHours int) (WebhookHealthStatus, error) {
	// Get recent event statistics
	var recentEventsCount int
	var recentSuccessRate float64

	eventQuery := `
		SELECT 
			COUNT(*),
			COALESCE(AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END), 0)
		FROM webhook_health_events 
		WHERE webhook_id = $1 
		  AND timestamp >= NOW() - INTERVAL '1 hour' * $2`

	err := tx.QueryRowContext(ctx, eventQuery, webhookID, lookbackHours).
		Scan(&recentEventsCount, &recentSuccessRate)
	if err != nil {
		return StatusUnknown, fmt.Errorf("failed to get recent events: %w", err)
	}

	// Get consecutive failures
	var consecutiveFailures int
	stateQuery := `SELECT COALESCE(consecutive_failures, 0) FROM webhook_health_state WHERE webhook_id = $1`
	err = tx.QueryRowContext(ctx, stateQuery, webhookID).Scan(&consecutiveFailures)
	if err != nil && err != sql.ErrNoRows {
		return StatusUnknown, fmt.Errorf("failed to get consecutive failures: %w", err)
	}

	// Calculate health status using the same logic as the original trigger
	if recentEventsCount == 0 {
		return StatusUnknown, nil
	} else if consecutiveFailures >= 5 {
		return StatusUnhealthy, nil
	} else if recentSuccessRate < 0.8000 && recentEventsCount >= 10 {
		return StatusUnhealthy, nil
	} else if recentSuccessRate < 0.9000 && recentEventsCount >= 5 {
		return StatusDegraded, nil
	} else if recentSuccessRate >= 0.9000 && recentEventsCount >= 3 {
		return StatusHealthy, nil
	} else {
		return StatusUnknown, nil
	}
}

// updateWebhookHealth updates the health status in the webhook_registrations table
func (hc *HealthCalculator) updateWebhookHealth(ctx context.Context, tx *sqlx.Tx, webhookID string, status WebhookHealthStatus) error {
	query := `UPDATE webhook_registrations SET health = $1, updated_at = NOW() WHERE id = $2`
	_, err := tx.ExecContext(ctx, query, string(status), webhookID)
	return err
}

// AggregateHealthHourly aggregates webhook health data for the specified time range.
// Uses a single bulk SQL statement to compute and upsert all hourly summaries at once,
// avoiding the N+1 query problem of iterating per-webhook and per-hour.
// Includes error category breakdown (client_errors, server_errors, timeout_errors, network_errors).
func (hc *HealthCalculator) AggregateHealthHourly(ctx context.Context, lookbackHours int) (int, error) {
	query := `
		INSERT INTO webhook_health_summaries (
			webhook_id, window_start, window_end,
			total_deliveries, successful_deliveries, failed_deliveries,
			success_rate, avg_response_time, min_response_time,
			max_response_time, p95_response_time,
			client_errors, server_errors, timeout_errors, network_errors,
			updated_at
		)
		SELECT
			webhook_id,
			date_trunc('hour', timestamp) AS window_start,
			date_trunc('hour', timestamp) + INTERVAL '1 hour' AS window_end,
			COUNT(*) AS total_deliveries,
			SUM(CASE WHEN success THEN 1 ELSE 0 END) AS successful_deliveries,
			SUM(CASE WHEN success THEN 0 ELSE 1 END) AS failed_deliveries,
			AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END) AS success_rate,
			AVG(response_time)::INTEGER AS avg_response_time,
			MIN(response_time) AS min_response_time,
			MAX(response_time) AS max_response_time,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time)::INTEGER AS p95_response_time,
			SUM(CASE WHEN error_category = 'client_error' THEN 1 ELSE 0 END) AS client_errors,
			SUM(CASE WHEN error_category = 'server_error' THEN 1 ELSE 0 END) AS server_errors,
			SUM(CASE WHEN error_category = 'timeout' THEN 1 ELSE 0 END) AS timeout_errors,
			SUM(CASE WHEN error_category IN ('network_error', 'dns_error', 'tls_error', 'connection_refused') THEN 1 ELSE 0 END) AS network_errors,
			NOW()
		FROM webhook_health_events
		WHERE timestamp >= NOW() - INTERVAL '1 hour' * $1
		GROUP BY webhook_id, date_trunc('hour', timestamp)
		ON CONFLICT (webhook_id, window_start, window_end) DO UPDATE SET
			total_deliveries = EXCLUDED.total_deliveries,
			successful_deliveries = EXCLUDED.successful_deliveries,
			failed_deliveries = EXCLUDED.failed_deliveries,
			success_rate = EXCLUDED.success_rate,
			avg_response_time = EXCLUDED.avg_response_time,
			min_response_time = EXCLUDED.min_response_time,
			max_response_time = EXCLUDED.max_response_time,
			p95_response_time = EXCLUDED.p95_response_time,
			client_errors = EXCLUDED.client_errors,
			server_errors = EXCLUDED.server_errors,
			timeout_errors = EXCLUDED.timeout_errors,
			network_errors = EXCLUDED.network_errors,
			updated_at = NOW()`

	result, err := hc.db.ExecContext(ctx, query, lookbackHours)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate hourly health summaries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// GetWebhookHealthState gets the current health state for a webhook
func (hc *HealthCalculator) GetWebhookHealthState(ctx context.Context, webhookID string) (*WebhookHealthState, error) {
	var state WebhookHealthState
	query := `SELECT * FROM webhook_health_state WHERE webhook_id = $1`

	err := hc.db.GetContext(ctx, &state, query, webhookID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get webhook health state: %w", err)
	}

	return &state, nil
}

// InitializeWebhookHealthState creates initial health state for all webhooks that don't have one
func (hc *HealthCalculator) InitializeWebhookHealthState(ctx context.Context) error {
	query := `
		INSERT INTO webhook_health_state (webhook_id)
		SELECT id FROM webhook_registrations
		ON CONFLICT (webhook_id) DO NOTHING`

	_, err := hc.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to initialize webhook health states: %w", err)
	}

	return nil
}
