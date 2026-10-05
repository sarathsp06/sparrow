package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// listenCleanupInterval is how often expired listen sessions are deleted.
// An expired session already gets no deliveries (they fail as offline), so
// this only bounds how long its webhook row lingers.
const listenCleanupInterval = 5 * time.Minute

// ListenCleanupArgs is the periodic job that deletes listen sessions whose
// CLI never deleted them (killed terminal, lost network) once they expire.
type ListenCleanupArgs struct{}

// Kind returns the job kind for River queue
func (ListenCleanupArgs) Kind() string {
	return "listen_session_cleanup"
}

var _ river.JobArgsWithInsertOpts = (*ListenCleanupArgs)(nil)

// InsertOpts sets no UniqueOpts for the same reason as TokenPurgeArgs: the
// cleanup is an idempotent DELETE and only the leader enqueues it.
func (ListenCleanupArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueDefault}
}

// ListenCleanupWorker runs ListenCleanupArgs jobs.
type ListenCleanupWorker struct {
	river.WorkerDefaults[ListenCleanupArgs]
	repo   store.ListenRepository
	logger *slog.Logger
}

// NewListenCleanupWorker creates a listen session cleanup worker.
func NewListenCleanupWorker(repo store.ListenRepository) *ListenCleanupWorker {
	return &ListenCleanupWorker{repo: repo, logger: slog.Default().With("component", "listen-cleanup-worker")}
}

// Work deletes expired listen sessions and their webhooks.
func (w *ListenCleanupWorker) Work(ctx context.Context, _ *river.Job[ListenCleanupArgs]) error {
	deleted, err := w.repo.DeleteExpiredListenSessions(ctx)
	if err != nil {
		w.logger.ErrorContext(ctx, "Listen session cleanup failed", "error", err)
		return err
	}
	if deleted > 0 {
		w.logger.InfoContext(ctx, "Deleted expired listen sessions", "sessions_deleted", deleted)
	}
	return nil
}

// EnableListenSessions schedules the cleanup of expired listen sessions.
// Call it before Start.
func (m *Manager) EnableListenSessions(repo store.ListenRepository) {
	river.AddWorker(m.workers, NewListenCleanupWorker(repo))
	m.client.PeriodicJobs().Add(river.NewPeriodicJob(
		river.PeriodicInterval(listenCleanupInterval),
		func() (river.JobArgs, *river.InsertOpts) { return ListenCleanupArgs{}, nil },
		&river.PeriodicJobOpts{RunOnStart: true},
	))
}
