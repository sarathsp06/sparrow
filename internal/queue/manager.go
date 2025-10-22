package queue

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/sarathsp06/sparrow/internal/jobs"
	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/workers"
)

// Manager handles the River queue management
type Manager struct {
	client      *river.Client[pgx.Tx]
	dbPool      *pgxpool.Pool
	webhookRepo *webhooks.Repository
}

// NewManager creates a new queue manager
func NewManager(ctx context.Context, databaseURL string) (*Manager, error) {
	// Create database connection pool
	dbPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	// Test database connection
	if err := dbPool.Ping(ctx); err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Create webhook repository
	webhookRepo := webhooks.NewRepository(dbPool)

	// Initialize River workers
	riverWorkers := river.NewWorkers()

	// Create River client first (needed for workers)
	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
			"events":           {MaxWorkers: 5}, // Event processing queue
			"webhooks":         {MaxWorkers: 8}, // Webhook delivery queue
		},
		Workers: riverWorkers,
	})
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to create River client: %w", err)
	}

	// Add workers that need dependencies
	river.AddWorker(riverWorkers, workers.NewWebhookWorker(webhookRepo))
	river.AddWorker(riverWorkers, workers.NewEventProcessingWorker(webhookRepo, riverClient))

	return &Manager{
		client:      riverClient,
		dbPool:      dbPool,
		webhookRepo: webhookRepo,
	}, nil
}

// Start starts the queue processing
func (m *Manager) Start(ctx context.Context) error {
	log := logger.NewLogger("queue-manager")

	if err := m.client.Start(ctx); err != nil {
		log.Error("Failed to start River client", "error", err)
		return fmt.Errorf("failed to start River client: %w", err)
	}

	log.Info("Connected to database")
	log.Info("River queue started successfully")
	return nil
}

// Stop stops the queue processing
func (m *Manager) Stop(ctx context.Context) error {
	m.client.Stop(ctx)
	m.dbPool.Close()
	return nil
}

// GetClient returns the River client
func (m *Manager) GetClient() *river.Client[pgx.Tx] {
	return m.client
}

// GetWebhookRepo returns the webhook repository
func (m *Manager) GetWebhookRepo() *webhooks.Repository {
	return m.webhookRepo
}

// InsertWebhookJob inserts a webhook job
func (m *Manager) InsertWebhookJob(ctx context.Context, args jobs.WebhookArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return m.client.Insert(ctx, args, opts)
}

// InsertManyJobs inserts multiple jobs at once
func (m *Manager) InsertManyJobs(ctx context.Context, params []river.InsertManyParams) ([]*rivertype.JobInsertResult, error) {
	return m.client.InsertMany(ctx, params)
}

// JobInserter provides methods to insert jobs with examples
type JobInserter struct {
	manager *Manager
}

// NewJobInserter creates a new job inserter
func (m *Manager) NewJobInserter() *JobInserter {
	return &JobInserter{manager: m}
}
