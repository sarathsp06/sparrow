package audit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// Repository defines the data access interface for audit logs.
type Repository interface {
	// WithConn returns a repository that executes queries against the given
	// connection (e.g. a transaction from storage.WithTransaction).
	WithConn(conn storage.DBTX) Repository

	// Insert persists a new audit log entry.
	Insert(ctx context.Context, entry *Entry) error

	// List retrieves audit log entries matching the given filter.
	List(ctx context.Context, filter ListFilter) ([]*Entry, int, error)
}

// pgRepository implements Repository using PostgreSQL via sqlx.
type pgRepository struct {
	db   storage.DB
	conn storage.DBTX
}

// NewRepository creates a new PostgreSQL-backed audit repository.
func NewRepository(db storage.DB) Repository {
	return &pgRepository{db: db, conn: db}
}

// WithConn returns a shallow copy that runs queries against conn.
func (r *pgRepository) WithConn(conn storage.DBTX) Repository {
	return &pgRepository{db: r.db, conn: conn}
}

// Insert persists a new audit log entry.
func (r *pgRepository) Insert(ctx context.Context, entry *Entry) error {
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now()
	}
	if entry.Metadata == "" {
		entry.Metadata = "{}"
	}

	query := `
		INSERT INTO audit_logs (id, tenant_id, actor_id, actor_type, action, resource_type, resource_id, namespace, metadata, ip_address, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb, $10, $11)
	`
	_, err := r.conn.ExecContext(ctx, query,
		entry.ID,
		entry.TenantID,
		entry.ActorID,
		string(entry.ActorType),
		string(entry.Action),
		string(entry.ResourceType),
		entry.ResourceID,
		entry.Namespace,
		entry.Metadata,
		entry.IPAddress,
		entry.CreatedAt,
	)
	return storage.Error(err)
}

// List retrieves audit log entries matching the given filter.
func (r *pgRepository) List(ctx context.Context, filter ListFilter) ([]*Entry, int, error) {
	// Build WHERE clause dynamically
	var conditions []string
	var args []interface{}
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argIdx))
	args = append(args, filter.TenantID)
	argIdx++

	if filter.Namespace != "" {
		conditions = append(conditions, fmt.Sprintf("namespace = $%d", argIdx))
		args = append(args, filter.Namespace)
		argIdx++
	}
	if filter.ResourceType != "" {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", argIdx))
		args = append(args, string(filter.ResourceType))
		argIdx++
	}
	if filter.ResourceID != "" {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", argIdx))
		args = append(args, filter.ResourceID)
		argIdx++
	}
	if filter.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, string(filter.Action))
		argIdx++
	}
	if filter.ActorID != "" {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, filter.ActorID)
		argIdx++
	}

	where := strings.Join(conditions, " AND ")

	// Count total
	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs WHERE %s", where)
	if err := r.conn.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, storage.Error(err)
	}

	// Apply pagination defaults
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	// Fetch rows
	dataQuery := fmt.Sprintf(`
		SELECT id, tenant_id, actor_id, actor_type, action, resource_type, resource_id, namespace, metadata, ip_address, created_at
		FROM audit_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)
	args = append(args, limit, offset)

	var entries []*Entry
	if err := r.conn.SelectContext(ctx, &entries, dataQuery, args...); err != nil {
		return nil, 0, storage.Error(err)
	}

	return entries, total, nil
}

// Compile-time check
var _ Repository = (*pgRepository)(nil)
