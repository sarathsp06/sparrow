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

	"github.com/sarathsp06/sparrow/internal/logger"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// Manager handles the River queue management
type Manager struct {
	client *river.Client[pgx.Tx]
	dbPool *pgxpool.Pool
	logger *slog.Logger
}

// NewManager creates a new queue manager
func NewManager(ctx context.Context, webhookRepo store.RepositoryInterface, dbPool *pgxpool.Pool) (*Manager, error) {
	// Initialize River workers
	riverWorkers := river.NewWorkers()

	// Create River client first (needed for workers)
	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		Queues: map[string]river.QueueConfig{
			QueueDefault:         {MaxWorkers: 5},
			QueueEventProcessing: {MaxWorkers: 20, FetchPollInterval: time.Second * 30}, // Event processing queue
			QueueWebhookDelivery: {MaxWorkers: 20, FetchPollInterval: time.Second * 30}, // Webhook delivery queue
		},
		Workers: riverWorkers,
	})
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to create River client: %w", err)
	}

	manager := &Manager{
		client: riverClient,
		dbPool: dbPool,
		logger: logger.NewLogger("queue-manager"),
	}

	// Add workers with explicit generic types
	river.AddWorker(riverWorkers, NewWebhookWorker(webhookRepo))
	river.AddWorker(riverWorkers, NewEventProcessingWorker(webhookRepo, manager.GetJobInserter()))

	return manager, nil
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
	_ = m.client.Stop(ctx)
	m.dbPool.Close()
	return nil
}

func (m *Manager) GetClient() *river.Client[pgx.Tx] {
	return m.client
}

func (m *Manager) GetJobInserter() JobInserter {
	return NewJobInserterWithTracing(NewJobInserter(m.client), "")
}
