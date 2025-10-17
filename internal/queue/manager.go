package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivertype"
	"github.com/sarathsp06/httpqueue/internal/jobs"
	"github.com/sarathsp06/httpqueue/internal/logger"
	"github.com/sarathsp06/httpqueue/internal/workers"
)

// Manager handles the River queue management
type Manager struct {
	client *river.Client[pgx.Tx]
	dbPool *pgxpool.Pool
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

	// Initialize River workers
	riverWorkers := river.NewWorkers()
	river.AddWorker(riverWorkers, workers.DataProcessingWorker{})
	river.AddWorker(riverWorkers, workers.WebhookWorker{})

	// Create River client
	riverClient, err := river.NewClient(riverpgxv5.New(dbPool), &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
			"priority":         {MaxWorkers: 5},
			"webhooks":         {MaxWorkers: 8}, // Dedicated queue for webhooks
		},
		Workers: riverWorkers,
	})
	if err != nil {
		dbPool.Close()
		return nil, fmt.Errorf("failed to create River client: %w", err)
	}

	return &Manager{
		client: riverClient,
		dbPool: dbPool,
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

// InsertDataProcessingJob inserts a data processing job
func (m *Manager) InsertDataProcessingJob(ctx context.Context, args jobs.DataProcessingArgs, opts *river.InsertOpts) (*rivertype.JobInsertResult, error) {
	return m.client.Insert(ctx, args, opts)
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

// InsertBatchJobs inserts a batch of webhook jobs
func (ji *JobInserter) InsertBatchJobs(ctx context.Context) error {
	log := logger.NewLogger("job-inserter")

	insertParams := make([]river.InsertManyParams, 0)
	for i := 0; i < 3; i++ {
		// Add webhook jobs
		insertParams = append(insertParams, river.InsertManyParams{
			Args: jobs.WebhookArgs{
				URL: "https://httpbin.org/post",
				Payload: map[string]interface{}{
					"event":    "batch_notification",
					"batch_id": i + 1,
					"user":     fmt.Sprintf("batch-user%d@example.com", i+1),
					"type":     "batch_webhook_notification",
				},
			},
			InsertOpts: &river.InsertOpts{
				Queue: "webhooks",
			},
		})
	}

	results, err := ji.manager.InsertManyJobs(ctx, insertParams)
	if err != nil {
		return fmt.Errorf("failed to insert batch jobs: %w", err)
	}
	log.Info("Inserted batch jobs",
		"total_jobs", len(results),
		"webhook_jobs", 3,
	)
	return nil
}

// StartPeriodicJobs starts inserting periodic jobs
func (ji *JobInserter) StartPeriodicJobs(ctx context.Context) {
	log := logger.NewLogger("periodic-jobs")

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Insert periodic data processing job
				job, err := ji.manager.InsertDataProcessingJob(ctx, jobs.DataProcessingArgs{
					DataID:   int(time.Now().Unix()),
					DataType: "periodic_cleanup",
				}, nil)
				if err != nil {
					log.Error("Failed to insert periodic job", "error", err)
				} else {
					log.Info("Inserted periodic cleanup job",
						"job_id", job.Job.ID,
						"data_type", "periodic_cleanup",
					)
				}

				// Insert periodic webhook health check
				healthJob, err := ji.manager.InsertWebhookJob(ctx, jobs.WebhookArgs{
					URL:    "https://httpbin.org/status/200",
					Method: "GET",
					Payload: map[string]interface{}{
						"timestamp":  time.Now().Unix(),
						"check_type": "periodic_health",
						"service":    "httpqueue",
					},
					Timeout: 5,
				}, &river.InsertOpts{
					Queue: "webhooks",
				})
				if err != nil {
					log.Error("Failed to insert periodic webhook health check", "error", err)
				} else {
					log.Info("Inserted periodic webhook health check",
						"job_id", healthJob.Job.ID,
						"url", "https://httpbin.org/status/200",
					)
				}
			}
		}
	}()
}
