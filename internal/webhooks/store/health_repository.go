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
	RecordDeliveryOutcome(ctx context.Context, outcome DeliveryOutcome) (oldHealth, newHealth string, err error)
	UpdateWebhookHealthState(ctx context.Context, webhookID uuid.UUID, success bool, eventTimestamp time.Time) (oldHealth, newHealth string, err error)
	CalculateWebhookHealth(ctx context.Context, webhookID uuid.UUID, lookbackHours int) (string, error)
	RecordWebhookHealthEvent(ctx context.Context, webhookID, deliveryID uuid.UUID, success bool, responseTime, responseCode int, errorMessage string, errorCategory string) error
	GetWebhookHealthState(ctx context.Context, webhookID uuid.UUID) (*WebhookHealthMetrics, error)
	GetWebhookHealthSummary(ctx context.Context, webhookID uuid.UUID, hours int) (*WebhookHealthSummary, error)
	GetWebhookHealthTimeSeries(ctx context.Context, webhookID uuid.UUID, hours int, bucketSize string) ([]*WebhookHealthEvent, error)
	AggregateHealthSummaries(ctx context.Context) (int, error)
	GetHealthSummary(ctx context.Context, tenantID uuid.UUID) (map[WebhookHealth]int, error)
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

// UpdateWebhookHealthState records a webhook delivery outcome and updates health metrics.
// For successful deliveries, it resets consecutive failures to 0 and updates last success timestamp.
// For failed deliveries, it increments consecutive failures and updates last failure timestamp.
// After updating health state, it recalculates the overall webhook health status (healthy/degraded/unhealthy)
// and returns the health value from immediately before and after this call, so callers can detect a
// transition (e.g. to emit a health-change notification) without a separate read.
// This function performs upsert operations to handle both new webhooks and existing ones.
func (r *Repository) UpdateWebhookHealthState(ctx context.Context, webhookID uuid.UUID, success bool, eventTimestamp time.Time) (string, string, error) {
	var oldHealth string
	if err := r.conn.GetContext(ctx, &oldHealth, `SELECT health FROM webhook_registrations WHERE id = $1`, webhookID); err != nil {
		return "", "", storage.Error(err)
	}

	var lastSuccessAt, lastFailureAt *time.Time
	if success {
		lastSuccessAt = &eventTimestamp
	} else {
		lastFailureAt = &eventTimestamp
	}

	// Atomic upsert: use a single SQL statement to avoid read-then-write race conditions.
	// For failures, increment consecutive_failures atomically in the ON CONFLICT clause.
	// For successes, reset to 0.
	var consecutiveFailures int
	if success {
		consecutiveFailures = 0
	} else {
		consecutiveFailures = 1
	}

	_, err := r.conn.ExecContext(ctx, `
		INSERT INTO webhook_health_state (webhook_id, consecutive_failures, last_success_at, last_failure_at, failing_since, last_event_at, updated_at)
		VALUES ($1, $2, $3, $4, $4, $5, NOW())
		ON CONFLICT (webhook_id) DO UPDATE SET
			consecutive_failures = CASE WHEN $6 THEN 0 ELSE webhook_health_state.consecutive_failures + 1 END,
			failing_since = CASE WHEN $6 THEN NULL ELSE COALESCE(webhook_health_state.failing_since, $5) END,
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
		success,
	)
	if err != nil {
		return "", "", storage.Error(err)
	}

	// Calculate health status
	newHealth, err := r.CalculateWebhookHealth(ctx, webhookID, 24)
	if err != nil {
		return "", "", storage.Error(err)
	}

	// Update webhook_registrations health field
	_, err = r.conn.ExecContext(ctx, `UPDATE webhook_registrations SET health = $1, updated_at = NOW() WHERE id = $2`, newHealth, webhookID)
	if err != nil {
		return "", "", storage.Error(err)
	}
	return oldHealth, newHealth, nil
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

// CalculateWebhookHealth determines webhook health status based on delivery patterns.
// Health calculation considers: recent success rate within lookbackHours window, consecutive failures,
// and minimum event threshold for statistical significance.
// Returns: "healthy" (>90% success, <5 failures), "degraded" (80-90% success),
//
//	"unhealthy" (<80% success or >=5 consecutive failures), "unknown" (insufficient data).
func (r *Repository) CalculateWebhookHealth(ctx context.Context, webhookID uuid.UUID, lookbackHours int) (string, error) {
	// Get recent delivery statistics (count unique deliveries, not attempts)
	query := `
		SELECT 
			COUNT(DISTINCT delivery_id),
			COALESCE(
				CASE WHEN COUNT(DISTINCT delivery_id) > 0
				     THEN COUNT(DISTINCT CASE WHEN success THEN delivery_id END)::FLOAT / COUNT(DISTINCT delivery_id)
				     ELSE 0
				END, 0)
		FROM webhook_health_events
		WHERE webhook_id = $1 AND timestamp >= NOW() - INTERVAL '1 hour' * $2
	`
	var result struct {
		EventsCount int     `db:"count"`
		SuccessRate float64 `db:"coalesce"`
	}
	err := r.conn.GetContext(ctx, &result, query, webhookID, lookbackHours)
	if err != nil {
		return "unknown", storage.Error(err)
	}
	// Get consecutive failures
	var consecutiveFailuresCount int
	err = r.conn.GetContext(ctx, &consecutiveFailuresCount, `SELECT COALESCE(consecutive_failures, 0) FROM webhook_health_state WHERE webhook_id = $1`, webhookID)
	if err != nil && !storage.IsNotFound(storage.Error(err)) {
		return "", storage.Error(err)
	}

	return healthLabel(result.EventsCount, result.SuccessRate, consecutiveFailuresCount), nil
}

// healthLabel is the health classification: recent distinct deliveries in
// the lookback window, their success rate, and the current run of failures.
func healthLabel(recentEvents int, successRate float64, consecutiveFailures int) string {
	switch {
	case recentEvents == 0:
		return "unknown"
	case consecutiveFailures >= 5:
		return "unhealthy"
	case successRate < 0.8 && recentEvents >= 10:
		return "unhealthy"
	case successRate < 0.9 && recentEvents >= 5:
		return "degraded"
	case successRate >= 0.9 && recentEvents >= 3:
		return "healthy"
	default:
		return "unknown"
	}
}

// healthLabelRecomputeInterval bounds how often a webhook's health label is
// recomputed from its 24h event history while deliveries keep producing the
// same outcome. A flip between success and failure, or crossing the
// consecutive-failure threshold, recomputes immediately.
const healthLabelRecomputeInterval = 5 * time.Second

// healthLabelDue reports whether the 24h recomputation should run now.
// prevComputedAt is nil when the label has never been computed; prevFailures
// and prevLastWasFailure describe the state before this outcome.
func healthLabelDue(now time.Time, prevComputedAt *time.Time, prevFailures, newFailures int, prevLastWasFailure, prevKnown, success bool) bool {
	switch {
	case !prevKnown, prevComputedAt == nil:
		return true
	case now.Sub(*prevComputedAt) >= healthLabelRecomputeInterval:
		return true
	case prevLastWasFailure == success: // outcome flipped
		return true
	case (prevFailures >= 5) != (newFailures >= 5):
		return true
	}
	return false
}

// DeliveryOutcome is one delivery attempt's result, as recorded for health.
type DeliveryOutcome struct {
	WebhookID     uuid.UUID
	DeliveryID    uuid.UUID
	Success       bool
	ResponseTime  int
	ResponseCode  int
	ErrorMessage  string
	ErrorCategory string
	At            time.Time
}

// RecordDeliveryOutcome records a delivery attempt for health tracking in
// one transaction: the health event, the webhook's health state, and the
// health label on the registration when it changed. It returns the label
// before and after.
//
// It replaces the pair RecordWebhookHealthEvent + UpdateWebhookHealthState,
// which issued four to five autocommit statements per delivery. Two of them
// wrote rows owned by the webhook, each holding its row lock through its own
// commit, so deliveries to one busy webhook serialized at roughly one per
// commit latency however many workers ran. Here the webhook-owned rows are
// touched last, the registration is written only when the label changes,
// and the 24h event scan behind the label runs at most every
// healthLabelRecomputeInterval unless the outcome flips.
func (r *Repository) RecordDeliveryOutcome(ctx context.Context, o DeliveryOutcome) (string, string, error) {
	if o.At.IsZero() {
		o.At = time.Now()
	}
	var oldHealth, newHealth string
	err := storage.WithTransaction(r.db, func(tx storage.DBTX) error {
		var err error
		oldHealth, newHealth, err = r.WithConn(tx).recordDeliveryOutcome(ctx, o)
		return err
	})
	return oldHealth, newHealth, err
}

func (r *Repository) recordDeliveryOutcome(ctx context.Context, o DeliveryOutcome) (string, string, error) {
	var oldHealth string
	if err := r.conn.GetContext(ctx, &oldHealth, `SELECT health FROM webhook_registrations WHERE id = $1`, o.WebhookID); err != nil {
		return "", "", storage.Error(err)
	}

	// Append-only, no contention: do it before taking the webhook's state row.
	if _, err := r.conn.ExecContext(ctx, `
		INSERT INTO webhook_health_events (webhook_id, delivery_id, success, response_time, response_code, error_message, error_category, timestamp)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, o.WebhookID, o.DeliveryID, o.Success, o.ResponseTime, o.ResponseCode, o.ErrorMessage, o.ErrorCategory, o.At); err != nil {
		return "", "", storage.Error(err)
	}

	var lastSuccessAt, lastFailureAt *time.Time
	initialFailures := 0
	if o.Success {
		lastSuccessAt = &o.At
	} else {
		lastFailureAt = &o.At
		initialFailures = 1
	}

	// Upsert the state row and read back what it looked like before, so the
	// label decision below needs no extra round trip.
	var st struct {
		Failures          int        `db:"consecutive_failures"`
		PrevKnown         bool       `db:"prev_known"`
		PrevFailures      int        `db:"prev_failures"`
		PrevLastSuccessAt *time.Time `db:"prev_last_success_at"`
		PrevLastFailureAt *time.Time `db:"prev_last_failure_at"`
		PrevLabelComputed *time.Time `db:"prev_label_computed_at"`
	}
	if err := r.conn.GetContext(ctx, &st, `
		WITH prev AS (
			SELECT consecutive_failures, last_success_at, last_failure_at, health_label_computed_at
			FROM webhook_health_state WHERE webhook_id = $1
		)
		INSERT INTO webhook_health_state (webhook_id, consecutive_failures, last_success_at, last_failure_at, failing_since, last_event_at, updated_at)
		VALUES ($1, $2, $3, $4, $4, $5, NOW())
		ON CONFLICT (webhook_id) DO UPDATE SET
			consecutive_failures = CASE WHEN $6 THEN 0 ELSE webhook_health_state.consecutive_failures + 1 END,
			failing_since = CASE WHEN $6 THEN NULL ELSE COALESCE(webhook_health_state.failing_since, $5) END,
			last_success_at = COALESCE($3, webhook_health_state.last_success_at),
			last_failure_at = COALESCE($4, webhook_health_state.last_failure_at),
			last_event_at = $5,
			updated_at = NOW()
		RETURNING
			webhook_health_state.consecutive_failures,
			EXISTS (SELECT 1 FROM prev) AS prev_known,
			COALESCE((SELECT consecutive_failures FROM prev), 0) AS prev_failures,
			(SELECT last_success_at FROM prev) AS prev_last_success_at,
			(SELECT last_failure_at FROM prev) AS prev_last_failure_at,
			(SELECT health_label_computed_at FROM prev) AS prev_label_computed_at
	`, o.WebhookID, initialFailures, lastSuccessAt, lastFailureAt, o.At, o.Success); err != nil {
		return "", "", storage.Error(err)
	}

	prevLastWasFailure := st.PrevLastFailureAt != nil && (st.PrevLastSuccessAt == nil || st.PrevLastFailureAt.After(*st.PrevLastSuccessAt))
	newHealth := oldHealth
	if healthLabelDue(o.At, st.PrevLabelComputed, st.PrevFailures, st.Failures, prevLastWasFailure, st.PrevKnown, o.Success) {
		var agg struct {
			Count int     `db:"count"`
			Rate  float64 `db:"coalesce"`
		}
		if err := r.conn.GetContext(ctx, &agg, `
			SELECT
				COUNT(DISTINCT delivery_id),
				COALESCE(
					CASE WHEN COUNT(DISTINCT delivery_id) > 0
					     THEN COUNT(DISTINCT CASE WHEN success THEN delivery_id END)::FLOAT / COUNT(DISTINCT delivery_id)
					     ELSE 0
					END, 0)
			FROM webhook_health_events
			WHERE webhook_id = $1 AND timestamp >= NOW() - INTERVAL '24 hours'
		`, o.WebhookID); err != nil {
			return "", "", storage.Error(err)
		}
		newHealth = healthLabel(agg.Count, agg.Rate, st.Failures)
		if _, err := r.conn.ExecContext(ctx, `UPDATE webhook_health_state SET health_label_computed_at = $2 WHERE webhook_id = $1`, o.WebhookID, o.At); err != nil {
			return "", "", storage.Error(err)
		}
	}

	if newHealth != oldHealth {
		if _, err := r.conn.ExecContext(ctx, `UPDATE webhook_registrations SET health = $1, updated_at = NOW() WHERE id = $2`, newHealth, o.WebhookID); err != nil {
			return "", "", storage.Error(err)
		}
	}
	return oldHealth, newHealth, nil
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

// GetWebhookHealthTimeSeries gets health events over time for analytics.
// bucketSize controls time bucketing: valid values are "1 minute", "5 minutes", "1 hour", "1 day".
// If empty, raw events are returned (up to 1000).
func (r *Repository) GetWebhookHealthTimeSeries(ctx context.Context, webhookID uuid.UUID, hours int, bucketSize string) ([]*WebhookHealthEvent, error) {
	if bucketSize == "" {
		// Return raw events when no bucket size specified
		query := `
			SELECT id, webhook_id, delivery_id, success, response_time, response_code, error_message, timestamp
			FROM webhook_health_events
			WHERE webhook_id = $1 
			  AND timestamp >= NOW() - INTERVAL '1 hour' * $2
			ORDER BY timestamp DESC
			LIMIT 1000
		`
		var events []*WebhookHealthEvent
		err := r.conn.SelectContext(ctx, &events, query, webhookID, hours)
		if err != nil {
			return nil, storage.Error(err)
		}
		return events, nil
	}

	// Validate and map bucketSize to a date_trunc precision to prevent SQL injection.
	var truncPrecision string
	switch bucketSize {
	case "1 minute":
		truncPrecision = "minute"
	case "5 minutes":
		truncPrecision = "minute" // We'll use 5-minute flooring below
	case "1 hour":
		truncPrecision = "hour"
	case "1 day":
		truncPrecision = "day"
	default:
		return nil, fmt.Errorf("invalid bucket size: %q (valid: \"1 minute\", \"5 minutes\", \"1 hour\", \"1 day\")", bucketSize)
	}

	// For 5-minute buckets, floor to 5-minute intervals using epoch arithmetic.
	// For all others, date_trunc with the precision is sufficient.
	var bucketExpr string
	if bucketSize == "5 minutes" {
		bucketExpr = "to_timestamp(floor(extract(epoch from timestamp) / 300) * 300)"
	} else {
		bucketExpr = fmt.Sprintf("date_trunc('%s', timestamp)", truncPrecision)
	}

	query := fmt.Sprintf(`
		SELECT 
			gen_random_uuid() AS id,
			webhook_id,
			'00000000-0000-0000-0000-000000000000'::uuid AS delivery_id,
			(AVG(CASE WHEN success THEN 1.0 ELSE 0.0 END) >= 0.5) AS success,
			COALESCE(AVG(response_time), 0)::INTEGER AS response_time,
			0 AS response_code,
			'' AS error_message,
			%s AS timestamp
		FROM webhook_health_events
		WHERE webhook_id = $1
		  AND timestamp >= NOW() - INTERVAL '1 hour' * $2
		GROUP BY %s, webhook_id
		ORDER BY %s DESC
		LIMIT 1000
	`, bucketExpr, bucketExpr, bucketExpr)

	var events []*WebhookHealthEvent
	err := r.conn.SelectContext(ctx, &events, query, webhookID, hours)
	if err != nil {
		return nil, storage.Error(err)
	}

	return events, nil
}

// AggregateHealthSummaries computes hourly health summaries from raw health events
// and inserts them into the webhook_health_summaries table.
// Includes error category breakdown (client_errors, server_errors, timeout_errors, network_errors, unexpected_status_errors).
// Returns the number of summaries processed.
func (r *Repository) AggregateHealthSummaries(ctx context.Context) (int, error) {
	query := `
		INSERT INTO webhook_health_summaries (
			id, webhook_id, window_start, window_end,
			total_deliveries, successful_deliveries, failed_deliveries,
			success_rate, avg_response_time, min_response_time, max_response_time, p95_response_time,
			client_errors, server_errors, timeout_errors, network_errors, unexpected_status_errors,
			created_at, updated_at
		)
		SELECT
			gen_random_uuid(),
			webhook_id,
			date_trunc('hour', timestamp) AS window_start,
			date_trunc('hour', timestamp) + INTERVAL '1 hour' AS window_end,
			COUNT(DISTINCT delivery_id) AS total_deliveries,
			COUNT(DISTINCT CASE WHEN success THEN delivery_id END) AS successful_deliveries,
			COUNT(DISTINCT delivery_id) - COUNT(DISTINCT CASE WHEN success THEN delivery_id END) AS failed_deliveries,
			COALESCE(
				CASE WHEN COUNT(DISTINCT delivery_id) > 0
				     THEN COUNT(DISTINCT CASE WHEN success THEN delivery_id END)::FLOAT / COUNT(DISTINCT delivery_id)
				     ELSE 0
				END, 0) AS success_rate,
			COALESCE(AVG(response_time), 0)::INTEGER AS avg_response_time,
			COALESCE(MIN(response_time), 0) AS min_response_time,
			COALESCE(MAX(response_time), 0) AS max_response_time,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY response_time), 0)::INTEGER AS p95_response_time,
			SUM(CASE WHEN error_category = 'client_error' THEN 1 ELSE 0 END) AS client_errors,
			SUM(CASE WHEN error_category = 'server_error' THEN 1 ELSE 0 END) AS server_errors,
			SUM(CASE WHEN error_category = 'timeout' THEN 1 ELSE 0 END) AS timeout_errors,
			SUM(CASE WHEN error_category IN ('network_error', 'dns_error', 'tls_error', 'connection_refused') THEN 1 ELSE 0 END) AS network_errors,
			SUM(CASE WHEN error_category = 'unexpected_status' THEN 1 ELSE 0 END) AS unexpected_status_errors,
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
			client_errors = EXCLUDED.client_errors,
			server_errors = EXCLUDED.server_errors,
			timeout_errors = EXCLUDED.timeout_errors,
			network_errors = EXCLUDED.network_errors,
			unexpected_status_errors = EXCLUDED.unexpected_status_errors,
			updated_at = NOW()
	`

	result, err := r.conn.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("failed to aggregate health summaries: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
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
		WHERE tenant_id = $1
		GROUP BY 1, 2
	`, tenantID)
	if err != nil {
		return nil, storage.Error(err)
	}
	return counts, nil
}

// GetHealthSummary returns a summary of webhook health within a tenant
func (r *Repository) GetHealthSummary(ctx context.Context, tenantID uuid.UUID) (map[WebhookHealth]int, error) {
	query := `
		SELECT health, COUNT(*) as count
		FROM webhook_registrations
		WHERE tenant_id = $1
		GROUP BY health
	`

	type healthCount struct {
		Health string `db:"health"`
		Count  int    `db:"count"`
	}

	var results []healthCount
	err := r.conn.SelectContext(ctx, &results, query, tenantID)
	if err != nil {
		return nil, storage.Error(err)
	}

	summary := make(map[WebhookHealth]int)
	for _, result := range results {
		summary[WebhookHealth(result.Health)] = result.Count
	}

	return summary, nil
}
