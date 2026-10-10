package store

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// HealthRepository defines operations for webhook health tracking.
type HealthRepository interface {
	RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID uuid.UUID, success bool, responseTime, responseCode int, errorMessage string, errorCategory string) error
	GetWebhookHealthState(ctx context.Context, webhookID uuid.UUID) (*WebhookHealthMetrics, error)
	GetWebhookHealthSummary(ctx context.Context, webhookID uuid.UUID, hours int) (*WebhookHealthSummary, error)
	GetHealthSummary(ctx context.Context, tenantID uuid.UUID, consumer string) (map[WebhookHealth]int, error)
	GetConsumerStats(ctx context.Context, tenantID uuid.UUID, consumer string) (*ConsumerStats, error)
	AutoDisableWebhook(ctx context.Context, webhookID uuid.UUID, minFailures int, failingFor time.Duration) (*AutoDisableResult, error)
	CountWebhooksByState(ctx context.Context, tenantID uuid.UUID) ([]WebhookStateCount, error)
}

// WebhookStateCount is the number of webhooks with one health and status,
// where status is "active", "paused" (by an operator) or "auto_disabled".
type WebhookStateCount struct {
	Health string `db:"health"`
	Status string `db:"status"`
	Count  int64  `db:"count"`
}

// AutoDisableResult describes a webhook AutoDisableWebhook just paused.
type AutoDisableResult struct {
	Reason              string    `db:"reason"`
	ConsecutiveFailures int       `db:"consecutive_failures"`
	FailingSince        time.Time `db:"failing_since"`
}

// AutoDisableWebhook pauses an active webhook whose receiver has failed at
// least minFailures attempts in a row, with no success for failingFor. The
// check and the pause are one statement, so concurrent delivery jobs pause a
// webhook at most once. It returns nil when nothing was disabled.
func (r *Repository) AutoDisableWebhook(ctx context.Context, webhookID uuid.UUID, minFailures int, failingFor time.Duration) (*AutoDisableResult, error) {
	var result AutoDisableResult
	err := r.conn.GetContext(ctx, &result, `
		UPDATE webhook_registrations wr
		SET active = false,
		    auto_disabled_at = NOW(),
		    auto_disabled_reason = format('auto-disabled: %s failed attempts in a row since %s with no success',
		        hs.consecutive_failures, to_char(hs.failing_since AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')),
		    updated_at = NOW()
		FROM webhook_health_state hs
		WHERE wr.id = $1 AND hs.webhook_id = wr.id AND wr.active
		  AND hs.consecutive_failures >= $2
		  AND hs.failing_since <= NOW() - make_interval(secs => $3)
		RETURNING wr.auto_disabled_reason AS reason, hs.consecutive_failures, hs.failing_since
	`, webhookID, minFailures, failingFor.Seconds())
	if err != nil {
		if storage.IsNotFound(storage.Error(err)) {
			return nil, nil
		}
		return nil, storage.Error(err)
	}
	return &result, nil
}

// HealthRules are the thresholds a webhook's health label is derived from.
// The API serves them so clients describe the labels without copying the
// numbers.
type HealthRules struct {
	// WindowHours is the lookback window the attempt counts and success rate
	// cover.
	WindowHours int
	// UnhealthyConsecutiveFailures failed attempts in a row make a webhook
	// unhealthy regardless of its success rate.
	UnhealthyConsecutiveFailures int
	// A success rate below UnhealthySuccessRate over at least
	// UnhealthyMinAttempts attempts is unhealthy.
	UnhealthySuccessRate float64
	UnhealthyMinAttempts int
	// A success rate below DegradedSuccessRate over at least
	// DegradedMinAttempts attempts is degraded.
	DegradedSuccessRate float64
	DegradedMinAttempts int
	// HealthyMinAttempts attempts at or above DegradedSuccessRate are needed
	// to call a webhook healthy; fewer leave it unknown.
	HealthyMinAttempts int
}

// DefaultHealthRules are the rules the health evaluator applies.
var DefaultHealthRules = HealthRules{
	WindowHours:                  24,
	UnhealthyConsecutiveFailures: 5,
	UnhealthySuccessRate:         0.8,
	UnhealthyMinAttempts:         10,
	DegradedSuccessRate:          0.9,
	DegradedMinAttempts:          5,
	HealthyMinAttempts:           3,
}

// healthLabel is the health classification: attempts in the lookback
// window, their success rate, and the current run of failures.
func healthLabel(recentEvents int, successRate float64, consecutiveFailures int) string {
	rules := DefaultHealthRules
	switch {
	case recentEvents == 0:
		return "unknown"
	case consecutiveFailures >= rules.UnhealthyConsecutiveFailures:
		return "unhealthy"
	case successRate < rules.UnhealthySuccessRate && recentEvents >= rules.UnhealthyMinAttempts:
		return "unhealthy"
	case successRate < rules.DegradedSuccessRate && recentEvents >= rules.DegradedMinAttempts:
		return "degraded"
	case successRate >= rules.DegradedSuccessRate && recentEvents >= rules.HealthyMinAttempts:
		return "healthy"
	default:
		return "unknown"
	}
}

// RecordWebhookHealthEvent creates a health tracking record for analytics and monitoring.
// Captures delivery outcome, response time metrics, HTTP status codes, error details,
// and error category for classifying failures (client_error, server_error, timeout, etc.).
// Timestamp is set to NOW() for accurate time-series data collection.
func (r *Repository) RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID uuid.UUID, success bool, responseTime, responseCode int, errorMessage string, errorCategory string) error {
	query := `
		INSERT INTO webhook_health_events (webhook_id, delivery_id, success, response_time, response_code, error_message, error_category, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
	`
	_, err := r.conn.ExecContext(ctx, query, webhookID, deliveryID, success, responseTime, responseCode, errorMessage, errorCategory)
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
	err := r.conn.GetContext(ctx, &state, query, webhookID)
	if err != nil {
		return nil, storage.Error(err)
	}

	return &state, nil
}

// GetWebhookHealthSummary provides aggregated performance metrics over a time window.
// Always computes metrics in real-time from webhook_health_events for accuracy.
// Uses COUNT(DISTINCT delivery_id) for delivery counts to avoid inflating numbers
// when a single delivery has multiple attempts (retries).
// Includes delivery counts, success rates, response time percentiles, and error breakdown.
func (r *Repository) GetWebhookHealthSummary(ctx context.Context, webhookID uuid.UUID, hours int) (*WebhookHealthSummary, error) {
	query := `
		SELECT 
			$1::uuid as webhook_id,
			NOW() - INTERVAL '1 hour' * $2 as window_start,
			NOW() as window_end,
			COUNT(DISTINCT delivery_id) as total_deliveries,
			COUNT(DISTINCT CASE WHEN success THEN delivery_id END) as successful_deliveries,
			COUNT(DISTINCT delivery_id) - COUNT(DISTINCT CASE WHEN success THEN delivery_id END) as failed_deliveries,
			COALESCE(
				CASE WHEN COUNT(DISTINCT delivery_id) > 0
				     THEN COUNT(DISTINCT CASE WHEN success THEN delivery_id END)::FLOAT / COUNT(DISTINCT delivery_id)
				     ELSE 0
				END, 0) as success_rate,
			COALESCE(AVG(response_time), 0)::INTEGER as avg_response_time,
			COALESCE(MIN(response_time), 0) as min_response_time,
			COALESCE(MAX(response_time), 0) as max_response_time,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time), 0)::INTEGER as p95_response_time,
			SUM(CASE WHEN error_category = 'client_error' THEN 1 ELSE 0 END) as client_errors,
			SUM(CASE WHEN error_category = 'server_error' THEN 1 ELSE 0 END) as server_errors,
			SUM(CASE WHEN error_category = 'timeout' THEN 1 ELSE 0 END) as timeout_errors,
			SUM(CASE WHEN error_category IN ('network_error', 'dns_error', 'tls_error', 'connection_refused') THEN 1 ELSE 0 END) as network_errors,
			SUM(CASE WHEN error_category = 'unexpected_status' THEN 1 ELSE 0 END) as unexpected_status_errors,
			NOW() as created_at,
			NOW() as updated_at
		FROM webhook_health_events
		WHERE webhook_id = $1 
		  AND timestamp >= NOW() - INTERVAL '1 hour' * $2
	`

	var summary WebhookHealthSummary
	err := r.conn.GetContext(ctx, &summary, query, webhookID, hours)
	if err != nil {
		return nil, storage.Error(err)
	}

	summary.ID = uuid.New()
	return &summary, nil
}

// CountWebhooksByState counts a tenant's webhooks by health and status, for
// the sparrow_webhooks gauge.
func (r *Repository) CountWebhooksByState(ctx context.Context, tenantID uuid.UUID) ([]WebhookStateCount, error) {
	var counts []WebhookStateCount
	err := r.conn.SelectContext(ctx, &counts, `
		SELECT health,
		       CASE WHEN active THEN 'active'
		            WHEN auto_disabled_at IS NOT NULL THEN 'auto_disabled'
		            ELSE 'paused' END AS status,
		       COUNT(*) AS count
		FROM webhook_registrations
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY 1, 2
	`, tenantID)
	if err != nil {
		return nil, storage.Error(err)
	}
	return counts, nil
}

// GetHealthSummary counts a consumer's webhooks by health, or every
// consumer's when consumer is empty.
func (r *Repository) GetHealthSummary(ctx context.Context, tenantID uuid.UUID, consumer string) (map[WebhookHealth]int, error) {
	var ns any
	if consumer != "" {
		ns = consumer
	}
	query := `
		SELECT health, COUNT(*) as count
		FROM webhook_registrations
		WHERE tenant_id = $1 AND deleted_at IS NULL
		  AND ($2::text IS NULL OR consumer = $2)
		GROUP BY health
	`

	type healthCount struct {
		Health string `db:"health"`
		Count  int    `db:"count"`
	}

	var results []healthCount
	err := r.conn.SelectContext(ctx, &results, query, tenantID, ns)
	if err != nil {
		return nil, storage.Error(err)
	}

	summary := make(map[WebhookHealth]int)
	for _, result := range results {
		summary[WebhookHealth(result.Health)] = result.Count
	}

	return summary, nil
}
