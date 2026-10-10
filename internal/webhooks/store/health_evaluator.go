package store

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// healthBucketRetention is how long per-minute rollups are kept; the label
// window is 24h.
const healthBucketRetention = 25 * time.Hour

// HealthLabelChange is a webhook whose health label changed in an evaluation.
type HealthLabelChange struct {
	WebhookID uuid.UUID
	TenantID  uuid.UUID
	Consumer  string
	URL       string
	OldHealth string
	NewHealth string
}

// FailingWebhook is a webhook touched by an evaluation that is in a failure
// run, for the auto-disable check.
type FailingWebhook struct {
	WebhookID           uuid.UUID
	TenantID            uuid.UUID
	Consumer            string
	URL                 string
	ConsecutiveFailures int
	FailingSince        *time.Time
}

// HealthEvaluation reports one evaluator pass.
type HealthEvaluation struct {
	From     time.Time
	To       time.Time
	Events   int
	Webhooks int
	Changes  []HealthLabelChange
	Failing  []FailingWebhook
	// More is set when the pass stopped at maxEvents and events remain
	// before upTo; the caller should run another pass.
	More bool
}

// HealthEvaluationRepository folds new delivery outcomes into webhook health.
type HealthEvaluationRepository interface {
	EvaluateHealth(ctx context.Context, upTo time.Time, maxEvents int) (*HealthEvaluation, error)
}

// EvaluateHealth folds every webhook_health_events row recorded after the
// watermark and up to upTo (at most about maxEvents of them) into:
//
//   - webhook_health_buckets: per-minute attempt/failure counts;
//   - webhook_health_state: consecutive_failures, failing_since and the
//     last_*_at timestamps, derived from the ordered outcome stream;
//   - webhook_metrics: running counters, never recounted from history:
//     total_attempts/total_failures (all time, for consumer and global
//     stats) and window_attempts/window_failures (the label's 24h window:
//     each pass adds its new outcomes and subtracts the buckets that slid out
//     of the window, so a pass reads about a minute of buckets however many
//     webhooks are busy);
//   - webhook_registrations.health: rewritten only where the label changed.
//
// Everything, including advancing the watermark, happens in one transaction
// that holds the watermark row, so concurrent instances serialize and a crash
// mid-pass replays cleanly. Delivery workers never write any of these rows:
// they only append to webhook_health_events, so a webhook receiving thousands
// of events a minute costs the evaluator one aggregate over its new rows and
// one write per touched webhook per pass.
//
// upTo should trail the clock by a few seconds: an event stamped earlier than
// a later one can still commit after it, and the watermark must not pass it.
func (r *Repository) EvaluateHealth(ctx context.Context, upTo time.Time, maxEvents int) (*HealthEvaluation, error) {
	if maxEvents <= 0 {
		maxEvents = 10000
	}
	var out *HealthEvaluation
	err := storage.WithTransaction(r.db, func(tx storage.DBTX) error {
		var err error
		out, err = r.WithConn(tx).evaluateHealth(ctx, upTo, maxEvents)
		return err
	})
	return out, err
}

func (r *Repository) evaluateHealth(ctx context.Context, upTo time.Time, maxEvents int) (*HealthEvaluation, error) {
	// 1. Watermark, locked for the pass. The first pass starts a label
	// window (24h) back so existing history is picked up.
	var from time.Time
	if err := r.conn.GetContext(ctx, &from, `
		INSERT INTO webhook_health_evaluator (singleton, watermark) VALUES (TRUE, $1)
		ON CONFLICT (singleton) DO UPDATE SET singleton = TRUE
		RETURNING watermark`, upTo.Add(-24*time.Hour)); err != nil {
		return nil, storage.Error(err)
	}
	if !upTo.After(from) {
		return &HealthEvaluation{From: from, To: from}, nil
	}

	// 2. Cap the slice: if more than maxEvents are pending, stop at the
	// timestamp of the maxEvents-th one (events sharing it are included, so
	// the half-open (from, to] boundary never splits a timestamp).
	to := upTo
	more := false
	var capTS time.Time
	err := r.conn.GetContext(ctx, &capTS, `
		SELECT timestamp FROM webhook_health_events
		WHERE timestamp > $1 AND timestamp <= $2
		ORDER BY timestamp OFFSET $3 LIMIT 1`, from, upTo, maxEvents-1)
	switch {
	case err == nil:
		if capTS.Before(upTo) {
			to, more = capTS, true
		}
	case storage.IsNotFound(storage.Error(err)):
	default:
		return nil, storage.Error(err)
	}

	res := &HealthEvaluation{From: from, To: to, More: more}
	windowHours := DefaultHealthRules.WindowHours

	// 3. Slide the label window from (from - 24h) to (to - 24h): subtract the
	// buckets that left it. This runs before the rollup below, so a bucket
	// leaves with exactly the counts that were added while it was inside.
	if _, err := r.conn.ExecContext(ctx, `
		UPDATE webhook_metrics m
		SET window_attempts = m.window_attempts - x.attempts,
		    window_failures = m.window_failures - x.failures
		FROM (
			SELECT webhook_id, SUM(attempts) AS attempts, SUM(failures) AS failures
			FROM webhook_health_buckets
			WHERE bucket_start >= date_trunc('minute', $1::timestamptz) - make_interval(hours => $3)
			  AND bucket_start <  date_trunc('minute', $2::timestamptz) - make_interval(hours => $3)
			GROUP BY webhook_id
		) x
		WHERE m.webhook_id = x.webhook_id`, from, to, windowHours); err != nil {
		return nil, storage.Error(err)
	}

	// 4. Per-minute rollups.
	bucketRes, err := r.conn.ExecContext(ctx, `
		INSERT INTO webhook_health_buckets (webhook_id, bucket_start, attempts, failures)
		SELECT e.webhook_id, date_trunc('minute', e.timestamp), COUNT(*), COUNT(*) FILTER (WHERE NOT e.success)
		FROM webhook_health_events e
		JOIN webhook_registrations wr ON wr.id = e.webhook_id
		WHERE e.timestamp > $1 AND e.timestamp <= $2
		GROUP BY 1, 2
		ON CONFLICT (webhook_id, bucket_start) DO UPDATE SET
			attempts = webhook_health_buckets.attempts + EXCLUDED.attempts,
			failures = webhook_health_buckets.failures + EXCLUDED.failures`, from, to)
	if err != nil {
		return nil, storage.Error(err)
	}
	_ = bucketRes

	// 5. Failure-run state from the ordered outcome stream. For each touched
	// webhook: the trailing failures after its last success in the slice
	// (or all of them if the slice has no success), and when that run began.
	// The slice's outcomes are added to the webhook's counters; only those
	// inside the label window (all of them, unless the slice is a backlog
	// older than the window) count toward the window.
	type stateRow struct {
		WebhookID           uuid.UUID  `db:"webhook_id"`
		TenantID            uuid.UUID  `db:"tenant_id"`
		Consumer            string     `db:"consumer"`
		URL                 string     `db:"url"`
		Health              string     `db:"health"`
		ConsecutiveFailures int        `db:"consecutive_failures"`
		FailingSince        *time.Time `db:"failing_since"`
		Events              int        `db:"events"`
		WindowAttempts      int        `db:"window_attempts"`
		WindowFailures      int        `db:"window_failures"`
	}
	var touched []stateRow
	if err := r.conn.SelectContext(ctx, &touched, `
		WITH slice AS (
			SELECT e.webhook_id, e.timestamp, e.success
			FROM webhook_health_events e
			JOIN webhook_registrations wr ON wr.id = e.webhook_id
			WHERE e.timestamp > $1 AND e.timestamp <= $2
		),
		agg AS (
			SELECT webhook_id,
			       COUNT(*)                                   AS events,
			       COUNT(*) FILTER (WHERE NOT success)        AS failures,
			       COUNT(*) FILTER (WHERE timestamp >= date_trunc('minute', $2::timestamptz) - make_interval(hours => $3))                 AS window_events,
			       COUNT(*) FILTER (WHERE NOT success AND timestamp >= date_trunc('minute', $2::timestamptz) - make_interval(hours => $3)) AS window_failures,
			       MAX(timestamp) FILTER (WHERE success)      AS last_success_at,
			       MAX(timestamp) FILTER (WHERE NOT success)  AS last_failure_at,
			       MAX(timestamp)                             AS last_event_at
			FROM slice GROUP BY webhook_id
		),
		run AS (
			SELECT s.webhook_id, COUNT(*) AS trailing_failures, MIN(s.timestamp) AS trailing_since
			FROM slice s JOIN agg a USING (webhook_id)
			WHERE NOT s.success AND s.timestamp > COALESCE(a.last_success_at, '-infinity'::timestamptz)
			GROUP BY s.webhook_id
		),
		upserted AS (
			INSERT INTO webhook_health_state (webhook_id, consecutive_failures, last_success_at, last_failure_at, failing_since, last_event_at, updated_at)
			SELECT a.webhook_id, COALESCE(run.trailing_failures, 0), a.last_success_at, a.last_failure_at, run.trailing_since, a.last_event_at, NOW()
			FROM agg a LEFT JOIN run USING (webhook_id)
			ON CONFLICT (webhook_id) DO UPDATE SET
				consecutive_failures = CASE WHEN EXCLUDED.last_success_at IS NOT NULL
				                            THEN EXCLUDED.consecutive_failures
				                            ELSE webhook_health_state.consecutive_failures + EXCLUDED.consecutive_failures END,
				failing_since        = CASE WHEN EXCLUDED.last_success_at IS NOT NULL
				                            THEN EXCLUDED.failing_since
				                            ELSE COALESCE(webhook_health_state.failing_since, EXCLUDED.failing_since) END,
				last_success_at = GREATEST(webhook_health_state.last_success_at, EXCLUDED.last_success_at),
				last_failure_at = GREATEST(webhook_health_state.last_failure_at, EXCLUDED.last_failure_at),
				last_event_at   = GREATEST(webhook_health_state.last_event_at, EXCLUDED.last_event_at),
				updated_at      = NOW()
			RETURNING webhook_id, consecutive_failures, failing_since
		),
		counted AS (
			INSERT INTO webhook_metrics (webhook_id, total_attempts, total_failures, window_attempts, window_failures, updated_at)
			SELECT a.webhook_id, a.events, a.failures, a.window_events, a.window_failures, NOW()
			FROM agg a
			ON CONFLICT (webhook_id) DO UPDATE SET
				total_attempts  = webhook_metrics.total_attempts + EXCLUDED.total_attempts,
				total_failures  = webhook_metrics.total_failures + EXCLUDED.total_failures,
				window_attempts = webhook_metrics.window_attempts + EXCLUDED.window_attempts,
				window_failures = webhook_metrics.window_failures + EXCLUDED.window_failures,
				updated_at      = NOW()
			RETURNING webhook_id, window_attempts, window_failures
		)
		SELECT u.webhook_id, wr.tenant_id, wr.consumer, wr.url, wr.health, u.consecutive_failures, u.failing_since, a.events,
		       c.window_attempts, c.window_failures
		FROM upserted u
		JOIN counted c USING (webhook_id)
		JOIN agg a USING (webhook_id)
		JOIN webhook_registrations wr ON wr.id = u.webhook_id`, from, to, windowHours); err != nil {
		return nil, storage.Error(err)
	}
	res.Webhooks = len(touched)
	if len(touched) == 0 {
		return res, r.advanceHealthWatermark(ctx, to)
	}

	// 6. Labels for the touched webhooks, from their window counters,
	// written in one statement for all that changed.
	var changedIDs, changedLabels []string
	for _, t := range touched {
		res.Events += t.Events
		rate := 0.0
		if t.WindowAttempts > 0 {
			rate = float64(t.WindowAttempts-t.WindowFailures) / float64(t.WindowAttempts)
		}
		label := healthLabel(t.WindowAttempts, rate, t.ConsecutiveFailures)
		if label != t.Health {
			changedIDs = append(changedIDs, t.WebhookID.String())
			changedLabels = append(changedLabels, label)
			res.Changes = append(res.Changes, HealthLabelChange{WebhookID: t.WebhookID, TenantID: t.TenantID, Consumer: t.Consumer, URL: t.URL, OldHealth: t.Health, NewHealth: label})
		}
		if t.ConsecutiveFailures > 0 {
			res.Failing = append(res.Failing, FailingWebhook{WebhookID: t.WebhookID, TenantID: t.TenantID, Consumer: t.Consumer, URL: t.URL, ConsecutiveFailures: t.ConsecutiveFailures, FailingSince: t.FailingSince})
		}
	}
	if len(changedIDs) > 0 {
		if _, err := r.conn.ExecContext(ctx, `
			UPDATE webhook_registrations wr SET health = c.health, updated_at = NOW()
			FROM unnest($1::uuid[], $2::text[]) AS c(id, health)
			WHERE wr.id = c.id`, pq.Array(changedIDs), pq.Array(changedLabels)); err != nil {
			return nil, storage.Error(err)
		}
	}

	// 7. Drop rollups outside the window.
	if _, err := r.conn.ExecContext(ctx, `DELETE FROM webhook_health_buckets WHERE bucket_start < $1`, to.Add(-healthBucketRetention)); err != nil {
		return nil, storage.Error(err)
	}

	return res, r.advanceHealthWatermark(ctx, to)
}

func (r *Repository) advanceHealthWatermark(ctx context.Context, to time.Time) error {
	_, err := r.conn.ExecContext(ctx, `UPDATE webhook_health_evaluator SET watermark = $1, updated_at = NOW() WHERE singleton`, to)
	return storage.Error(err)
}

// DeliveryAttempt is the outcome of one send, recorded in a single
// transaction: the delivery row (status, response, the request body that
// was sent) and the append-only health event the evaluator folds in later.
type DeliveryAttempt struct {
	DeliveryID uuid.UUID
	WebhookID  uuid.UUID

	// UpdateStatus is false for outcomes that leave the delivery row as it
	// is (a 429 snooze), while still recording a health event.
	UpdateStatus  bool
	Status        WebhookDeliveryStatus
	ResponseCode  int
	ResponseBody  string
	ErrorMessage  string
	ErrorCategory string
	// RequestBody, when non-nil, is stored on the delivery row.
	RequestBody *string

	Success        bool
	ResponseTimeMs int
	// HealthErrorMessage is what the health event records; it may differ
	// from ErrorMessage (e.g. the raw transport error).
	HealthErrorMessage string
}

// RecordDeliveryAttempt writes a DeliveryAttempt in one transaction, so a
// delivery costs one commit. Health state and labels are not touched here;
// see EvaluateHealth.
func (r *Repository) RecordDeliveryAttempt(ctx context.Context, a DeliveryAttempt) error {
	return storage.WithTransaction(r.db, func(tx storage.DBTX) error {
		rr := r.WithConn(tx)
		if a.UpdateStatus {
			attemptIncrement := 0
			if a.Status == StatusFailed || a.Status == StatusSuccess || a.Status == StatusRetrying {
				attemptIncrement = 1
			}
			if _, err := rr.conn.ExecContext(ctx, `
				UPDATE webhook_deliveries
				SET status = $2, last_attempted_at = $3, response_code = $4, response_body = $5, error_message = $6,
				    attempt_count = attempt_count + $7::integer, error_category = $8,
				    request_body = COALESCE($9, request_body)
				WHERE id = $1`,
				a.DeliveryID, a.Status, time.Now(), a.ResponseCode, a.ResponseBody, a.ErrorMessage, attemptIncrement, a.ErrorCategory, a.RequestBody); err != nil {
				return storage.Error(err)
			}
		}
		return rr.RecordWebhookHealthEvent(ctx, a.WebhookID, a.DeliveryID, a.Success, a.ResponseTimeMs, a.ResponseCode, a.HealthErrorMessage, a.ErrorCategory)
	})
}
