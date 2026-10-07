package queue

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/sarathsp06/sparrow/internal/webhooks/client"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/crypto"
)

// Manager handles the River queue management
type Manager struct {
	client  *river.Client[pgx.Tx]
	workers *river.Workers
	dbPool  *pgxpool.Pool
	logger  *slog.Logger
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
}

// Default worker pool sizing.
const (
	DefaultEventWorkers   = 20
	DefaultWebhookWorkers = 20
	DefaultFetchCooldown  = 20 * time.Millisecond
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
	return c
}

// NewManager creates a new queue manager. retentionDays > 0 enables an hourly
// periodic job that purges events older than that many days; autoDisable
// controls when failing webhooks are paused automatically; pool sizes the
// event and delivery worker pools.
func NewManager(ctx context.Context, webhookRepo store.RepositoryInterface, cryptoSvc *crypto.Service, dbPool *pgxpool.Pool, clientConfig *client.Config, retentionDays int, autoDisable AutoDisablePolicy, pool WorkerPoolConfig) (*Manager, error) {
	pool = pool.withDefaults()
	// Initialize River workers
	riverWorkers := river.NewWorkers()

	periodicJobs := []*river.PeriodicJob{
		river.NewPeriodicJob(
			river.PeriodicInterval(batchCleanupInterval),
			func() (river.JobArgs, *river.InsertOpts) { return BatchCleanupArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true},
		),
	}
	if retentionDays > 0 {
		periodicJobs = append(periodicJobs, river.NewPeriodicJob(
			river.PeriodicInterval(time.Hour),
			func() (river.JobArgs, *river.InsertOpts) { return RetentionArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true},
		))
	}

	// Create River client first (needed for workers)
	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		PeriodicJobs:  periodicJobs,
		FetchCooldown: pool.FetchCooldown,
		Queues: map[string]river.QueueConfig{
			QueueDefault:         {MaxWorkers: 5},
			QueueEventProcessing: {MaxWorkers: pool.EventWorkers, FetchPollInterval: time.Second * 2},   // Event fan-out queue
			QueueWebhookDelivery: {MaxWorkers: pool.WebhookWorkers, FetchPollInterval: time.Second * 2}, // Webhook delivery queue
			QueueBatchJobs:       {MaxWorkers: 5, FetchPollInterval: time.Second * 5},                   // Batch job processing queue
		},
		Workers: riverWorkers,
	})
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to create River client: %w", err)
	}

	manager := &Manager{
		client:  riverClient,
		workers: riverWorkers,
		dbPool:  dbPool,
		logger:  slog.Default().With("component", "queue-manager"),
	}

	// Add workers with explicit generic types.
	// RepositoryInterface satisfies all narrow interfaces via embedding.
	river.AddWorker(riverWorkers, NewWebhookWorker(webhookRepo, webhookRepo, webhookRepo, webhookRepo, webhookRepo, webhookRepo, webhookRepo, manager.GetJobInserter(), cryptoSvc, clientConfig, autoDisable))
	river.AddWorker(riverWorkers, NewEventProcessingWorker(webhookRepo, webhookRepo, webhookRepo, manager.GetJobInserter()))
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

// Stop stops the queue processing
func (m *Manager) Stop(ctx context.Context) error {
	_ = m.client.Stop(ctx)
	m.dbPool.Close()
	return nil
}

func (m *Manager) GetJobInserter() JobInserter {
	return NewJobInserterWithTracing(NewJobInserter(m.client), "")
}
