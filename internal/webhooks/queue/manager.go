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

// NewManager creates a new queue manager. retentionDays > 0 enables an hourly
// periodic job that purges events older than that many days; autoDisable
// controls when failing webhooks are paused automatically.
func NewManager(ctx context.Context, webhookRepo store.RepositoryInterface, cryptoSvc *crypto.Service, dbPool *pgxpool.Pool, clientConfig *client.Config, retentionDays int, autoDisable AutoDisablePolicy) (*Manager, error) {
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
			river.PeriodicInterval(retentionInterval),
			func() (river.JobArgs, *river.InsertOpts) { return RetentionArgs{}, nil },
			&river.PeriodicJobOpts{RunOnStart: true},
		))
	}

	// Create River client first (needed for workers)
	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		PeriodicJobs: periodicJobs,
		Queues: map[string]river.QueueConfig{
			QueueDefault:         {MaxWorkers: 5},
			QueueEventProcessing: {MaxWorkers: 20, FetchPollInterval: time.Second * 2}, // Event processing queue
			QueueWebhookDelivery: {MaxWorkers: 20, FetchPollInterval: time.Second * 2}, // Webhook delivery queue
			QueueBatchJobs:       {MaxWorkers: 5, FetchPollInterval: time.Second * 5},  // Batch job processing queue
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
