package queue

import (
	"context"
	"log/slog"
	"time"

	"github.com/riverqueue/river"
)

// tokenPurgeInterval is how often expired and revoked access tokens are purged.
const tokenPurgeInterval = 24 * time.Hour

// TokenPurger deletes access tokens that expired or were revoked more than
// retention ago (access.Service satisfies it).
type TokenPurger interface {
	PurgeTokens(ctx context.Context, retention time.Duration) (int64, error)
}

// TokenPurgeArgs is the daily job that deletes long-dead access tokens, so
// the table grows with live tokens (portal links are minted often), not with
// every token ever issued.
type TokenPurgeArgs struct{}

// Kind returns the job kind for River queue
func (TokenPurgeArgs) Kind() string {
	return "access_token_purge"
}

var _ river.JobArgsWithInsertOpts = (*TokenPurgeArgs)(nil)

// InsertOpts deliberately sets no UniqueOpts: River's default unique states
// include completed jobs until its cleaner removes them (24h), which would
// silently skip the next day's run. The purge is an idempotent DELETE, and
// periodic jobs are only enqueued by the leader, so duplicates are harmless.
func (TokenPurgeArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: QueueDefault}
}

// TokenPurgeWorker runs TokenPurgeArgs jobs.
type TokenPurgeWorker struct {
	river.WorkerDefaults[TokenPurgeArgs]
	purger    TokenPurger
	retention time.Duration
	logger    *slog.Logger
}

// NewTokenPurgeWorker creates a worker that purges tokens dead for longer than retention.
func NewTokenPurgeWorker(purger TokenPurger, retention time.Duration) *TokenPurgeWorker {
	return &TokenPurgeWorker{
		purger:    purger,
		retention: retention,
		logger:    slog.Default().With("component", "token-purge-worker"),
	}
}

// Work purges expired and revoked tokens past the retention window.
func (w *TokenPurgeWorker) Work(ctx context.Context, _ *river.Job[TokenPurgeArgs]) error {
	deleted, err := w.purger.PurgeTokens(ctx, w.retention)
	if err != nil {
		w.logger.ErrorContext(ctx, "Access token purge failed", "error", err)
		return err
	}
	if deleted > 0 {
		w.logger.InfoContext(ctx, "Access token purge completed", "tokens_deleted", deleted, "retention", w.retention)
	}
	return nil
}
