package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// batchCleanupInterval is how often abandoned batch snapshots are purged.
// Snapshots expire after their TTL (15 min by default), so hourly is plenty.
const batchCleanupInterval = time.Hour

// BatchCleanupArgs is the periodic job that deletes expired, never-started
// batch jobs: snapshots prepared with prepare_retry/prepare_repush that the
// user never confirmed (e.g. closed the tab with the dialog open).
type BatchCleanupArgs struct{}

// Kind returns the job kind for River queue
func (BatchCleanupArgs) Kind() string {
	return "batch_cleanup"
}

var _ river.JobArgsWithInsertOpts = (*BatchCleanupArgs)(nil)

func (BatchCleanupArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueDefault,
		// ByPeriod is required: River's default unique states include
		// completed, so without it the next interval's insert is deduped
		// against the previous run until the job cleaner removes it (~24h).
		UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: batchCleanupInterval},
	}
}

// BatchCleanupWorker deletes expired pending batch jobs.
type BatchCleanupWorker struct {
	river.WorkerDefaults[BatchCleanupArgs]
	repo   store.BatchRepository
	logger *slog.Logger
}

// NewBatchCleanupWorker creates a batch cleanup worker.
func NewBatchCleanupWorker(repo store.BatchRepository) *BatchCleanupWorker {
	return &BatchCleanupWorker{
		repo:   repo,
		logger: slog.Default().With("component", "batch-cleanup-worker"),
	}
}

// Work deletes expired batch snapshots that were never started.
func (w *BatchCleanupWorker) Work(ctx context.Context, job *river.Job[BatchCleanupArgs]) error {
	deleted, err := w.repo.CleanupExpiredBatchJobs(ctx)
	if err != nil {
		w.logger.ErrorContext(ctx, "Batch cleanup failed", "error", err)
		return err
	}
	if deleted > 0 {
		w.logger.InfoContext(ctx, "Batch cleanup completed", "batch_jobs_deleted", deleted)
	}
	return nil
}
