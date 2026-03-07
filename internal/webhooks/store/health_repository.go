package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// UpdateWebhookHealthState records a webhook delivery outcome and updates health metrics.
// For successful deliveries, it resets consecutive failures to 0 and updates last success timestamp.
// For failed deliveries, it increments consecutive failures and updates last failure timestamp.
// After updating health state, it recalculates the overall webhook health status (healthy/degraded/unhealthy).
// This function performs upsert operations to handle both new webhooks and existing ones.
func (r *Repository) UpdateWebhookHealthState(ctx context.Context, webhookID uuid.UUID, success bool, eventTimestamp time.Time) error {
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
func (r *Repository) CalculateWebhookHealth(ctx context.Context, webhookID uuid.UUID, lookbackHours int) (string, error) {
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

// RecordWebhookHealthEvent creates a health tracking record for analytics and monitoring.
// Captures delivery outcome, response time metrics, HTTP status codes, and error details.
// These events feed into webhook health calculations and performance analytics.
// Timestamp is set to NOW() for accurate time-series data collection.
func (r *Repository) RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID uuid.UUID, success bool, responseTime, responseCode int, errorMessage string) error {
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
func (r *Repository) GetWebhookHealthState(ctx context.Context, webhookID uuid.UUID) (*WebhookHealthMetrics, error) {
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
func (r *Repository) GetWebhookHealthSummary(ctx context.Context, webhookID uuid.UUID, hours int) (*WebhookHealthSummary, error) {
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
	summary.ID = uuid.New()

	return &summary, nil
}

// GetWebhookHealthTimeSeries gets health events over time for analytics
func (r *Repository) GetWebhookHealthTimeSeries(ctx context.Context, webhookID uuid.UUID, hours int, bucketSize string) ([]*WebhookHealthEvent, error) {
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

// AggregateHealthSummaries computes hourly health summaries from raw health events
// and inserts them into the webhook_health_summaries table.
// Returns the number of summaries processed.
func (r *Repository) AggregateHealthSummaries(ctx context.Context) (int, error) {
	query := `
		INSERT INTO webhook_health_summaries (
			id, webhook_id, window_start, window_end,
			total_deliveries, successful_deliveries, failed_deliveries,
			success_rate, avg_response_time, min_response_time, max_response_time, p95_response_time,
			created_at, updated_at
		)
		SELECT
			gen_random_uuid(),
			webhook_id,
			date_trunc('hour', timestamp) AS window_start,
			date_trunc('hour', timestamp) + INTERVAL '1 hour' AS window_end,
			COUNT(*) AS total_deliveries,
			SUM(CASE WHEN success THEN 1 ELSE 0 END) AS successful_deliveries,
			SUM(CASE WHEN success THEN 0 ELSE 1 END) AS failed_deliveries,
			COALESCE(AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END), 0) AS success_rate,
			COALESCE(AVG(response_time), 0)::INTEGER AS avg_response_time,
			COALESCE(MIN(response_time), 0) AS min_response_time,
			COALESCE(MAX(response_time), 0) AS max_response_time,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time), 0)::INTEGER AS p95_response_time,
			NOW(),
			NOW()
		FROM webhook_health_events
		WHERE timestamp >= NOW() - INTERVAL '24 hours'
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
			updated_at = NOW()
	`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate health summaries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
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
