package queue

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/sarathsp06/sparrow/internal/observability"
	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// workerMetrics are the delivery instruments the webhook worker records.
// A nil instrument (the meter refused it) is skipped.
type workerMetrics struct {
	attempts        metric.Int64Counter
	attemptDuration metric.Float64Histogram
	autoDisabled    metric.Int64Counter
}

func newWorkerMetrics() workerMetrics {
	meter := observability.GetMeter("sparrow")
	var m workerMetrics
	if c, err := meter.Int64Counter("sparrow_delivery_attempts_total",
		metric.WithDescription("Webhook delivery attempts that reached the receiver (or failed to), by result and error_category"),
	); err == nil {
		m.attempts = c
	}
	if h, err := meter.Float64Histogram("sparrow_delivery_attempt_duration",
		metric.WithUnit("s"),
		metric.WithDescription("Time from sending a webhook delivery attempt to the receiver's response, by result"),
		metric.WithExplicitBucketBoundaries(0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30),
	); err == nil {
		m.attemptDuration = h
	}
	if c, err := meter.Int64Counter("sparrow_webhooks_auto_disabled_total",
		metric.WithDescription("Webhooks Sparrow paused automatically because their receiver kept failing"),
	); err == nil {
		m.autoDisabled = c
	}
	return m
}

// recordAttempt records one delivery attempt outcome.
func (m workerMetrics) recordAttempt(ctx context.Context, success bool, durationMs int, errorCategory string) {
	result := "failure"
	if success {
		result = "success"
		errorCategory = ""
	}
	if m.attempts != nil {
		m.attempts.Add(ctx, 1, metric.WithAttributes(
			attribute.String("result", result),
			attribute.String("error_category", errorCategory),
		))
	}
	if m.attemptDuration != nil {
		m.attemptDuration.Record(ctx, float64(durationMs)/1000, metric.WithAttributes(attribute.String("result", result)))
	}
}

// stateMetricsTimeout bounds the queries behind one collection of the state
// gauges, so a slow database never stalls a scrape.
const stateMetricsTimeout = 5 * time.Second

// RegisterStateMetrics registers gauges read from the database at each
// collection: sparrow_queue_jobs (River jobs by queue and state) and
// sparrow_webhooks (webhooks by health and status).
func (m *Manager) RegisterStateMetrics(repo store.HealthRepository) error {
	meter := observability.GetMeter("sparrow")
	queueJobs, err := meter.Int64ObservableGauge("sparrow_queue_jobs",
		metric.WithDescription("River jobs waiting or running, by queue and state"))
	if err != nil {
		return fmt.Errorf("create sparrow_queue_jobs gauge: %w", err)
	}
	webhooks, err := meter.Int64ObservableGauge("sparrow_webhooks",
		metric.WithDescription("Webhooks by health and status (active, paused, auto_disabled)"))
	if err != nil {
		return fmt.Errorf("create sparrow_webhooks gauge: %w", err)
	}

	_, err = meter.RegisterCallback(func(ctx context.Context, o metric.Observer) error {
		ctx, cancel := context.WithTimeout(ctx, stateMetricsTimeout)
		defer cancel()

		// Queue depth: jobs waiting to run, waiting for a retry, or running.
		rows, err := m.dbPool.Query(ctx, `
			SELECT queue, state::text, COUNT(*)
			FROM river_job
			WHERE state IN ('available', 'scheduled', 'retryable', 'running')
			GROUP BY 1, 2`)
		if err != nil {
			return fmt.Errorf("count river jobs: %w", err)
		}
		for rows.Next() {
			var queue, state string
			var count int64
			if err := rows.Scan(&queue, &state, &count); err != nil {
				rows.Close()
				return fmt.Errorf("scan river job count: %w", err)
			}
			o.ObserveInt64(queueJobs, count, metric.WithAttributes(
				attribute.String("queue", queue), attribute.String("state", state)))
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return fmt.Errorf("count river jobs: %w", err)
		}

		counts, err := repo.CountWebhooksByState(ctx, tenant.DefaultTenantID)
		if err != nil {
			return fmt.Errorf("count webhooks: %w", err)
		}
		for _, c := range counts {
			o.ObserveInt64(webhooks, c.Count, metric.WithAttributes(
				attribute.String("health", c.Health), attribute.String("status", c.Status)))
		}
		return nil
	}, queueJobs, webhooks)
	if err != nil {
		return fmt.Errorf("register state metrics callback: %w", err)
	}
	return nil
}
