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
//   - webhook_health_buckets: per-minute attempt/failure counts, so the 24h
//     success rate behind the label is a sum over at most 1,440 small rows
//     however busy the webhook is;
//   - webhook_health_state: consecutive_failures, failing_since and the
//     last_*_at timestamps, derived from the ordered outcome stream;
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
		maxEvents = 50000
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

	// 3. Per-minute rollups.
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

	// 4. Failure-run state from the ordered outcome stream. For each touched
	// webhook: the trailing failures after its last success in the slice
	// (or all of them if the slice has no success), and when that run began.
	type stateRow struct {
		WebhookID           uuid.UUID  `db:"webhook_id"`
		TenantID            uuid.UUID  `db:"tenant_id"`
		Consumer            string     `db:"consumer"`
		URL                 string     `db:"url"`
		Health              string     `db:"health"`
		ConsecutiveFailures int        `db:"consecutive_failures"`
		FailingSince        *time.Time `db:"failing_since"`
		Events              int        `db:"events"`
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
		)
		SELECT u.webhook_id, wr.tenant_id, wr.consumer, wr.url, wr.health, u.consecutive_failures, u.failing_since, a.events
		FROM upserted u
		JOIN agg a USING (webhook_id)
		JOIN webhook_registrations wr ON wr.id = u.webhook_id`, from, to); err != nil {
		return nil, storage.Error(err)
	}
	res.Webhooks = len(touched)
	if len(touched) == 0 {
		return res, r.advanceHealthWatermark(ctx, to)
	}

	// 5. Labels for the touched webhooks from the 24h rollup window.
	ids := make([]uuid.UUID, 0, len(touched))
	for _, t := range touched {
		ids = append(ids, t.WebhookID)
		res.Events += t.Events
	}
	type winRow struct {
		WebhookID uuid.UUID `db:"webhook_id"`
		Attempts  int       `db:"attempts"`
		Failures  int       `db:"failures"`
	}
	var wins []winRow
	if err := r.conn.SelectContext(ctx, &wins, `
		SELECT webhook_id, SUM(attempts) AS attempts, SUM(failures) AS failures
		FROM webhook_health_buckets
		WHERE webhook_id = ANY($1) AND bucket_start >= date_trunc('minute', $2::timestamptz) - make_interval(hours => $3)
		GROUP BY webhook_id`, pq.Array(ids), to, DefaultHealthRules.WindowHours); err != nil {
		return nil, storage.Error(err)
	}
	window := make(map[uuid.UUID]winRow, len(wins))
	for _, w := range wins {
		window[w.WebhookID] = w
	}

	for _, t := range touched {
		w := window[t.WebhookID]
		rate := 0.0
		if w.Attempts > 0 {
			rate = float64(w.Attempts-w.Failures) / float64(w.Attempts)
		}
		label := healthLabel(w.Attempts, rate, t.ConsecutiveFailures)
		if label != t.Health {
			if _, err := r.conn.ExecContext(ctx, `UPDATE webhook_registrations SET health = $1, updated_at = NOW() WHERE id = $2`, label, t.WebhookID); err != nil {
				return nil, storage.Error(err)
			}
			res.Changes = append(res.Changes, HealthLabelChange{WebhookID: t.WebhookID, TenantID: t.TenantID, Consumer: t.Consumer, URL: t.URL, OldHealth: t.Health, NewHealth: label})
		}
		if t.ConsecutiveFailures > 0 {
			res.Failing = append(res.Failing, FailingWebhook{WebhookID: t.WebhookID, TenantID: t.TenantID, Consumer: t.Consumer, URL: t.URL, ConsecutiveFailures: t.ConsecutiveFailures, FailingSince: t.FailingSince})
		}
	}

	// 6. Drop rollups outside the window.
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
