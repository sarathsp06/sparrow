package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// retentionInterval is how often the retention purge runs.
const retentionInterval = time.Hour

// RetentionArgs is the periodic job that purges events older than the
// configured retention window. Carries no payload; the window comes from
// server config at worker construction.
type RetentionArgs struct{}

// Kind returns the job kind for River queue
func (RetentionArgs) Kind() string {
	return "retention_cleanup"
}

var _ river.JobArgsWithInsertOpts = (*RetentionArgs)(nil)

func (RetentionArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue: QueueDefault,
		// One retention job per interval is plenty. ByPeriod is required:
		// River's default unique states include completed, so without it the
		// next interval's insert is deduped against the previous run until
		// the job cleaner removes it (~24h).
		UniqueOpts: river.UniqueOpts{ByArgs: true, ByPeriod: retentionInterval},
	}
}

// RetentionWorker deletes events (and, via FK cascade, their deliveries)
// created before the retention cutoff.
type RetentionWorker struct {
	river.WorkerDefaults[RetentionArgs]
	repo   store.EventRepository
	days   int
	logger *slog.Logger
}

// NewRetentionWorker creates a retention worker purging data older than days.
func NewRetentionWorker(repo store.EventRepository, days int) *RetentionWorker {
	return &RetentionWorker{
		repo:   repo,
		days:   days,
		logger: slog.Default().With("component", "retention-worker"),
	}
}

// Work purges events older than the retention window.
func (w *RetentionWorker) Work(ctx context.Context, job *river.Job[RetentionArgs]) error {
	if w.days <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -w.days)
	deleted, err := w.repo.DeleteEventsBefore(ctx, cutoff)
	if err != nil {
		w.logger.ErrorContext(ctx, "Retention purge failed", "cutoff", cutoff, "deleted", deleted, "error", err)
		return err
	}
	if deleted > 0 {
		w.logger.InfoContext(ctx, "Retention purge completed", "cutoff", cutoff, "events_deleted", deleted)
	}
	return nil
}
