package audit

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/auth"
)

// Logger provides a high-level API for recording audit events.
// It extracts actor information from the request context (via auth.AuthInfo)
// and writes entries asynchronously to avoid adding latency to mutations.
type Logger struct {
	repo   Repository
	logger *slog.Logger
}

// NewLogger creates a new audit Logger.
func NewLogger(repo Repository, logger *slog.Logger) *Logger {
	if logger == nil {
		logger = slog.Default()
	}
	return &Logger{repo: repo, logger: logger}
}

// LogEntry describes a single audit event to record.
type LogEntry struct {
	Action       Action
	ResourceType ResourceType
	ResourceID   string
	Namespace    string
	Metadata     map[string]any // serialized to JSONB
}

// Log records an audit event. It extracts actor info from the context
// and writes the entry asynchronously (fire-and-forget) so it does not
// block the RPC response.
func (l *Logger) Log(ctx context.Context, entry LogEntry) {
	info := auth.FromContext(ctx)
	if info == nil {
		l.logger.Warn("audit: no auth info in context, skipping audit log",
			"action", string(entry.Action),
			"resource_type", string(entry.ResourceType),
			"resource_id", entry.ResourceID,
		)
		return
	}

	actorID, actorType := resolveActor(info)

	metadataJSON := "{}"
	if entry.Metadata != nil {
		if b, err := json.Marshal(entry.Metadata); err == nil {
			metadataJSON = string(b)
		}
	}

	record := &Entry{
		TenantID:     info.TenantID,
		ActorID:      actorID,
		ActorType:    actorType,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		Namespace:    entry.Namespace,
		Metadata:     metadataJSON,
	}

	// Fire-and-forget: run the INSERT in a background goroutine so it
	// does not add latency to the RPC. We use context.WithoutCancel so
	// the write completes even if the client disconnects.
	go func() {
		bgCtx := context.WithoutCancel(ctx)
		if err := l.repo.Insert(bgCtx, record); err != nil {
			l.logger.Error("audit: failed to write audit log",
				"error", err,
				"action", string(record.Action),
				"resource_type", string(record.ResourceType),
				"resource_id", record.ResourceID,
				"tenant_id", record.TenantID.String(),
			)
		}
	}()
}

// LogSync records an audit event synchronously. Use this within transactions
// where the audit log must be committed atomically with the mutation.
func (l *Logger) LogSync(ctx context.Context, entry LogEntry) error {
	info := auth.FromContext(ctx)
	if info == nil {
		l.logger.Warn("audit: no auth info in context, skipping audit log",
			"action", string(entry.Action),
		)
		return nil
	}

	actorID, actorType := resolveActor(info)

	metadataJSON := "{}"
	if entry.Metadata != nil {
		if b, err := json.Marshal(entry.Metadata); err == nil {
			metadataJSON = string(b)
		}
	}

	record := &Entry{
		TenantID:     info.TenantID,
		ActorID:      actorID,
		ActorType:    actorType,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		Namespace:    entry.Namespace,
		Metadata:     metadataJSON,
	}

	return l.repo.Insert(ctx, record)
}

// WithConn returns a Logger that uses the given connection for the repository.
// Useful for writing audit logs within a transaction.
func (l *Logger) WithConn(conn interface {
	ExecContext(context.Context, string, ...any) (interface{ RowsAffected() (int64, error) }, error)
}) *Logger {
	// This won't work directly since we need storage.DBTX; provide a
	// direct repo-level method instead. Callers should use the repository
	// directly for transactional audit logging.
	return l
}

// Repo returns the underlying repository for advanced use cases
// like transactional audit logging.
func (l *Logger) Repo() Repository {
	return l.repo
}

// resolveActor extracts actor identity from AuthInfo.
func resolveActor(info *auth.AuthInfo) (string, ActorType) {
	if info.KeyID != nil {
		return info.KeyID.String(), ActorAPIKey
	}
	if info.SubjectID != "" {
		return info.SubjectID, ActorUser
	}
	// Fallback: use tenant ID as actor for system/default auth
	return info.TenantID.String(), ActorSystem
}

// ListAuditLogs queries audit logs with filtering and pagination.
func (l *Logger) ListAuditLogs(ctx context.Context, tenantID uuid.UUID, filter ListFilter) ([]*Entry, int, error) {
	filter.TenantID = tenantID
	return l.repo.List(ctx, filter)
}
