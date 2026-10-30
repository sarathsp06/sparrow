package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"

	"github.com/sarathsp06/sparrow/internal/crypto"
	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// Manager handles the River queue management
type Manager struct {
	client   *river.Client[pgx.Tx]
	txClient *river.Client[*sql.Tx]
	workers  *river.Workers
	dbPool   *pgxpool.Pool
	logger   *slog.Logger

	tenantLimits TenantLimiter
	extraQueues  map[string]int
}

// WorkerPoolConfig sizes the River queues. Zero values take the defaults.
//
// Throughput per queue is bounded by MaxWorkers / FetchCooldown: River fetches
// at most once per cooldown, taking as many jobs as there are free worker
// slots, so 20 workers with River's default 100 ms cooldown cap a queue at
// about 200 jobs/s however fast the jobs are. The default cooldown here is
// 20 ms (about 1000 jobs/s per queue with 20 workers).
type WorkerPoolConfig struct {
	EventWorkers   int
	WebhookWorkers int
	FetchCooldown  time.Duration
	// RescueAfter is how long a job may stay running before River treats its
	// worker as dead and retries it (River's RescueStuckJobsAfter). After a
	// crash, jobs that were running wait this long. River never rescues a job
	// still inside its own timeout, so a 30 minute batch job is safe with a
	// shorter value.
	RescueAfter time.Duration

	// ExtraQueues are additional queues, by name, with their worker counts.
	// TenantLimits.Queue routes a tenant's jobs to one of them. Self-hosted
	// Sparrow sets none.
	ExtraQueues map[string]int
	// TenantLimits, when set, caps each tenant's running jobs and routes its
	// jobs to its queue. Nil means no per-tenant limits (self-hosted).
	TenantLimits TenantLimiter
}

// Default worker pool sizing.
const (
	DefaultEventWorkers   = 20
	DefaultWebhookWorkers = 20
	DefaultFetchCooldown  = 20 * time.Millisecond
	// DefaultRescueAfter is just above the delivery job timeout. River's own
	// default is an hour, which held deliveries that were running when the
	// process died for that long.
	DefaultRescueAfter = deliveryJobTimeout + time.Minute
)

func (c WorkerPoolConfig) withDefaults() WorkerPoolConfig {
	if c.EventWorkers <= 0 {
		c.EventWorkers = DefaultEventWorkers
	}
	if c.WebhookWorkers <= 0 {
		c.WebhookWorkers = DefaultWebhookWorkers
	}
	if c.FetchCooldown <= 0 {
		c.FetchCooldown = DefaultFetchCooldown
	}
	if c.RescueAfter <= 0 {
		c.RescueAfter = DefaultRescueAfter
	}
	return c
}

// NewManager creates a new queue manager. retentionDays > 0 enables an hourly
// periodic job that purges events older than that many days; autoDisable
// controls when failing webhooks are paused automatically; pool sizes the
// event and delivery worker pools.
func NewManager(ctx context.Context, webhookRepo store.RepositoryInterface, cryptoSvc *crypto.Service, dbPool *pgxpool.Pool, clientConfig *client.Config, retentionDays int, autoDisable AutoDisablePolicy, pool WorkerPoolConfig, health HealthEvaluatorConfig) (*Manager, error) {
	pool = pool.withDefaults()
	health = health.withDefaults()
	// Initialize River workers
	riverWorkers := river.NewWorkers()

	periodicJobs := []*river.PeriodicJob{
		river.NewPeriodicJob(
			river.PeriodicInterval(batchCleanupInterval),
			func() (river.JobArgs, *river.InsertOpts) { return BatchCleanupArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true},
		),
		river.NewPeriodicJob(
			river.PeriodicInterval(health.Interval),
			func() (river.JobArgs, *river.InsertOpts) { return HealthEvaluateArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true},
		),
	}
	if retentionDays > 0 {
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(retentionInterval),
			func() (river.JobArgs, *river.InsertOpts) { return RetentionArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true},
		))
	}

	queues := map[string]river.QueueConfig{
		QueueDefault:         {MaxWorkers: 5},
		QueueEventProcessing: {MaxWorkers: pool.EventWorkers, FetchPollInterval: time.Second * 2},   // Event fan-out queue
		QueueWebhookDelivery: {MaxWorkers: pool.WebhookWorkers, FetchPollInterval: time.Second * 2}, // Webhook delivery queue
		QueueBatchJobs:       {MaxWorkers: 5, FetchPollInterval: time.Second * 5},                   // Batch job processing queue
	}
	for name, workers := range pool.ExtraQueues {
		if _, exists := queues[name]; exists {
			dbPool.Close()
			return nil, fmt.Errorf("extra queue %q clashes with a built-in queue", name)
		}
		if workers <= 0 {
			dbPool.Close()
			return nil, fmt.Errorf("extra queue %q needs at least one worker", name)
		}
		queues[name] = river.QueueConfig{MaxWorkers: workers, FetchPollInterval: time.Second * 2}
	}

	// Create River client first (needed for workers)
	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		PeriodicJobs:         periodicJobs,
		FetchCooldown:        pool.FetchCooldown,
		RescueStuckJobsAfter: pool.RescueAfter,
		Queues:               queues,
		Workers:              riverWorkers,
	})
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to create River client: %w", err)
	}

	txClient, err := NewTxInsertClient()
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to create River insert client: %w", err)
	}

	manager := &Manager{
		client:       riverClient,
		txClient:     txClient,
		workers:      riverWorkers,
		dbPool:       dbPool,
		logger:       slog.Default().With("component", "queue-manager"),
		tenantLimits: pool.TenantLimits,
		extraQueues:  pool.ExtraQueues,
	}
	gate := newTenantGate(pool.TenantLimits)

	// Add workers with explicit generic types.
	// RepositoryInterface satisfies all narrow interfaces via embedding.
	webhookWorker := NewWebhookWorker(webhookRepo, webhookRepo, webhookRepo, webhookRepo, webhookRepo, webhookRepo, manager.GetJobInserter(), cryptoSvc, clientConfig)
	webhookWorker.gate = gate
	river.AddWorker(riverWorkers, webhookWorker)
	river.AddWorker(riverWorkers, NewHealthEvaluatorWorker(webhookRepo, webhookRepo, webhookRepo, webhookRepo, manager.GetJobInserter(), autoDisable, health))
	eventWorker := NewEventProcessingWorker(webhookRepo, webhookRepo, webhookRepo, manager.GetJobInserter())
	eventWorker.gate = gate
	river.AddWorker(riverWorkers, eventWorker)
	river.AddWorker(riverWorkers, NewBatchJobWorker(webhookRepo, webhookRepo, webhookRepo, webhookRepo, manager.GetJobInserter()))
	river.AddWorker(riverWorkers, NewRetentionWorker(webhookRepo, retentionDays))
	river.AddWorker(riverWorkers, NewBatchCleanupWorker(webhookRepo))

	return manager, nil
}

// EnableTokenPurge schedules a daily job deleting access tokens that expired
// or were revoked more than retention ago. Call it before Start.
func (m *Manager) EnableTokenPurge(purger TokenPurger, retention time.Duration) {
	river.AddWorker(m.workers, NewTokenPurgeWorker(purger, retention))
	m.client.PeriodicJobs().Add(river.NewPeriodicJob(
		river.PeriodicInterval(tokenPurgeInterval),
		func() (river.JobArgs, *river.InsertOpts) { return TokenPurgeArgs{}, nil },
		&river.PeriodicJobOpts{RunOnStart: true},
	))
}

// Start starts the queue processing
func (m *Manager) Start(ctx context.Context) error {
	log := slog.Default().With("component", "queue-manager")

	if err := m.client.Start(ctx); err != nil {
		log.ErrorContext(ctx, "Failed to start River client", "error", err)
		return fmt.Errorf("failed to start River client: %w", err)
	}

	log.InfoContext(ctx, "Connected to database")
	log.InfoContext(ctx, "River queue started successfully")
	return nil
}

// Stop shuts the queue down within ctx's deadline. It first stops fetching
// and waits for running jobs (a soft stop) for most of the budget, then
// cancels the jobs still running and waits for them to return (River retries
// them later; a cancelled delivery records no attempt). The pool is closed
// last, once no job can use it.
func (m *Manager) Stop(ctx context.Context) error {
	softCtx, cancelSoft := context.WithCancel(ctx)
	if deadline, ok := ctx.Deadline(); ok {
		// Leave a fifth of the budget for cancelled jobs to unwind.
		softCtx, cancelSoft = context.WithDeadline(ctx, deadline.Add(-time.Until(deadline)/5))
	}
	err := m.client.Stop(softCtx)
	cancelSoft()
	if err != nil {
		m.logger.WarnContext(ctx, "Jobs still running at the end of the soft stop, cancelling them", "error", err)
		err = m.client.StopAndCancel(ctx)
		if err != nil {
			m.logger.ErrorContext(ctx, "Queue did not stop before the shutdown deadline", "error", err)
		}
	}
	m.dbPool.Close()
	return err
}

// CancelTenantJobs cancels every unfinished job whose args belong to
// tenantID (event fan-out, delivery and batch jobs carry a tenant_id) and
// returns how many it cancelled. A job already running is cancelled through
// its context. Used before erasing a tenant.
func (m *Manager) CancelTenantJobs(ctx context.Context, tenantID uuid.UUID) (int, error) {
	filter, err := json.Marshal(map[string]string{"tenant_id": tenantID.String()})
	if err != nil {
		return 0, err
	}
	// The containment form uses river_job's GIN index on args.
	rows, err := m.dbPool.Query(ctx, `
		SELECT id FROM river_job
		WHERE args @> $1::jsonb
		  AND state NOT IN ('completed', 'cancelled', 'discarded')`, filter)
	if err != nil {
		return 0, fmt.Errorf("list tenant jobs: %w", err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[int64])
	if err != nil {
		return 0, fmt.Errorf("list tenant jobs: %w", err)
	}
	for _, id := range ids {
		if _, err := m.client.JobCancel(ctx, id); err != nil && !errors.Is(err, rivertype.ErrNotFound) {
			return 0, fmt.Errorf("cancel job %d: %w", id, err)
		}
	}
	return len(ids), nil
}

func (m *Manager) GetJobInserter() JobInserter {
	ji := NewJobInserter(m.client, m.txClient)
	ji.tenantLimits = m.tenantLimits
	ji.extraQueues = m.extraQueues
	return NewJobInserterWithTracing(ji, "")
}
