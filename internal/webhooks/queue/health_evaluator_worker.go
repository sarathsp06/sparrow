package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// HealthEvaluateArgs is the periodic job that folds new delivery outcomes
// into webhook health state, labels, alerts and auto-disable.
type HealthEvaluateArgs struct{}

// Kind returns the job kind for River queue
func (HealthEvaluateArgs) Kind() string { return "health_evaluate" }

var _ river.JobArgsWithInsertOpts = (*HealthEvaluateArgs)(nil)

// InsertOpts keeps a single evaluation queued or running at a time. ByState
// is set explicitly: River's default unique states include completed jobs
// until its cleaner removes them (24h), which would silently drop every
// periodic insert after the first run.
func (HealthEvaluateArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueDefault,
		UniqueOpts: river.UniqueOpts{
			ByArgs: true,
			ByState: []rivertype.JobState{
				rivertype.JobStateAvailable, rivertype.JobStatePending, rivertype.JobStateRunning,
				rivertype.JobStateRetryable, rivertype.JobStateScheduled,
			},
		},
	}
}

// HealthEvaluatorConfig tunes the periodic health evaluation.
type HealthEvaluatorConfig struct {
	// Interval between evaluations. Health labels, health_changed alerts and
	// auto-disable lag delivery outcomes by at most this much. Default 1m.
	Interval time.Duration
	// Lag is how far behind the clock the evaluation stops, so an event that
	// commits late is not skipped. Default 5s.
	Lag time.Duration
	// MaxEvents bounds one pass (one transaction); a backlog is worked
	// through in several passes within the same job. Default 10000: on 6M
	// deliveries a 10k-event pass takes ~85ms of rollup with a 1.4MB hash,
	// where 50k took ~380ms and 6MB while holding the evaluator's locks.
	MaxEvents int
}

// Default evaluator settings.
const (
	DefaultHealthEvalInterval = time.Minute
	defaultHealthEvalLag      = 5 * time.Second
	defaultHealthEvalMax      = 10000
	maxHealthEvalPasses       = 100
)

func (c HealthEvaluatorConfig) withDefaults() HealthEvaluatorConfig {
	if c.Interval <= 0 {
		c.Interval = DefaultHealthEvalInterval
	}
	if c.Lag <= 0 {
		c.Lag = defaultHealthEvalLag
	}
	if c.MaxEvents <= 0 {
		c.MaxEvents = defaultHealthEvalMax
	}
	return c
}

// HealthEvaluatorWorker runs store.EvaluateHealth and acts on what changed:
// health_changed system events for label transitions and the auto-disable
// check for webhooks in a failure run.
type HealthEvaluatorWorker struct {
	river.WorkerDefaults[HealthEvaluateArgs]
	evalRepo    store.HealthEvaluationRepository
	healthRepo  store.HealthRepository
	events      systemEventDeps
	autoDisable AutoDisablePolicy
	cfg         HealthEvaluatorConfig
	metrics     workerMetrics
	logger      *slog.Logger
}

// NewHealthEvaluatorWorker creates the evaluator worker.
func NewHealthEvaluatorWorker(evalRepo store.HealthEvaluationRepository, healthRepo store.HealthRepository, eventRepo systemEventRepo, alertConfigRepo store.AlertConfigRepository, jobInserter JobInserter, autoDisable AutoDisablePolicy, cfg HealthEvaluatorConfig) *HealthEvaluatorWorker {
	return &HealthEvaluatorWorker{
		evalRepo:    evalRepo,
		healthRepo:  healthRepo,
		events:      systemEventDeps{eventRepo: eventRepo, jobInserter: jobInserter, alertConfigRepo: alertConfigRepo},
		autoDisable: autoDisable,
		cfg:         cfg.withDefaults(),
		metrics:     newWorkerMetrics(),
		logger:      slog.Default().With("component", "health-evaluator"),
	}
}

// Work runs evaluation passes until the backlog before now-Lag is folded in
// (or maxHealthEvalPasses is reached; the next run continues).
func (w *HealthEvaluatorWorker) Work(ctx context.Context, _ *river.Job[HealthEvaluateArgs]) error {
	return w.evaluate(ctx)
}

func (w *HealthEvaluatorWorker) evaluate(ctx context.Context) error {
	log := w.logger
	for pass := 0; pass < maxHealthEvalPasses; pass++ {
		res, err := w.evalRepo.EvaluateHealth(ctx, time.Now().Add(-w.cfg.Lag), w.cfg.MaxEvents)
		if err != nil {
			return err
		}
		if res.Events > 0 {
			log.DebugContext(ctx, "Evaluated webhook health",
				"events", res.Events, "webhooks", res.Webhooks, "label_changes", len(res.Changes),
				"from", res.From, "to", res.To, "more", res.More)
		}
		for _, c := range res.Changes {
			emitHealthChangedEvent(ctx, log, w.events, c.TenantID, c.Consumer, c.WebhookID, c.URL, c.OldHealth, c.NewHealth)
		}
		for _, f := range res.Failing {
			w.maybeAutoDisable(ctx, log, f)
		}
		if !res.More {
			return nil
		}
	}
	return nil
}

// maybeAutoDisable pauses f.WebhookID if its receiver has been failing long
// enough under w.autoDisable, then announces it with a
// sparrow.webhook.disabled system event. _sparrow's own webhooks are never
// auto-disabled: they carry the alerts that would report it.
func (w *HealthEvaluatorWorker) maybeAutoDisable(ctx context.Context, log *slog.Logger, f store.FailingWebhook) {
	if f.Consumer == SystemEventConsumer || !w.autoDisable.enabled() || f.ConsecutiveFailures < w.autoDisable.MinFailures {
		return
	}
	disabled, err := w.healthRepo.AutoDisableWebhook(ctx, f.WebhookID, w.autoDisable.MinFailures, w.autoDisable.After)
	if err != nil {
		log.ErrorContext(ctx, "Failed to check webhook for auto-disable", "error", err, "webhook_id", f.WebhookID)
		return
	}
	if disabled == nil {
		return
	}
	log.WarnContext(ctx, "Webhook auto-disabled after repeated failures",
		"webhook_id", f.WebhookID,
		"reason", disabled.Reason,
		"consecutive_failures", disabled.ConsecutiveFailures,
		"failing_since", disabled.FailingSince)
	if w.metrics.autoDisabled != nil {
		w.metrics.autoDisabled.Add(ctx, 1)
	}
	emitWebhookDisabledEvent(ctx, log, w.events, f.TenantID, f.Consumer, f.WebhookID, f.URL, disabled)
}
