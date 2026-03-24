package namespace

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// Repository defines the data access interface for namespaces.
type Repository interface {
	// WithConn returns a repository that executes queries against the given
	// connection (e.g. a transaction from storage.WithTransaction).
	WithConn(conn storage.DBTX) Repository

	CreateNamespace(ctx context.Context, ns *Namespace) error
	GetNamespaceByID(ctx context.Context, id uuid.UUID) (*Namespace, error)
	GetNamespaceByName(ctx context.Context, tenantID uuid.UUID, name string) (*Namespace, error)
	ListNamespaces(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Namespace, int, error)
	UpdateNamespace(ctx context.Context, ns *Namespace) error
	DeleteNamespace(ctx context.Context, id uuid.UUID) error
}

// pgRepository implements Repository using PostgreSQL.
type pgRepository struct {
	db   storage.DB   // full connection — used for Beginx/Ping/Close
	conn storage.DBTX // query/exec target — either db or a transaction
}

// NewRepository creates a new PostgreSQL-backed namespace repository.
func NewRepository(db storage.DB) Repository {
	return &pgRepository{db: db, conn: db}
}

// WithConn returns a shallow copy that runs queries against conn.
func (r *pgRepository) WithConn(conn storage.DBTX) Repository {
	return &pgRepository{db: r.db, conn: conn}
}

// ---- Namespace operations ----

func (r *pgRepository) CreateNamespace(ctx context.Context, ns *Namespace) error {
	if ns.ID == uuid.Nil {
		ns.ID = uuid.New()
	}
	now := time.Now()
	ns.CreatedAt = now
	ns.UpdatedAt = now

	query := `
		INSERT INTO namespaces (id, tenant_id, name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := r.conn.ExecContext(ctx, query,
		ns.ID, ns.TenantID, ns.Name, ns.Description, ns.CreatedAt, ns.UpdatedAt,
	)
	return storage.Error(err)
}

func (r *pgRepository) GetNamespaceByID(ctx context.Context, id uuid.UUID) (*Namespace, error) {
	var ns Namespace
	query := `SELECT id, tenant_id, name, description, created_at, updated_at FROM namespaces WHERE id = $1`
	err := r.conn.GetContext(ctx, &ns, query, id)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &ns, nil
}

func (r *pgRepository) GetNamespaceByName(ctx context.Context, tenantID uuid.UUID, name string) (*Namespace, error) {
	var ns Namespace
	query := `SELECT id, tenant_id, name, description, created_at, updated_at FROM namespaces WHERE tenant_id = $1 AND name = $2`
	err := r.conn.GetContext(ctx, &ns, query, tenantID, name)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &ns, nil
}

func (r *pgRepository) ListNamespaces(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Namespace, int, error) {
	var total int
	err := r.conn.GetContext(ctx, &total, `SELECT COUNT(*) FROM namespaces WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	var namespaces []*Namespace
	query := `
		SELECT id, tenant_id, name, description, created_at, updated_at
		FROM namespaces WHERE tenant_id = $1
		ORDER BY name ASC LIMIT $2 OFFSET $3
	`
	err = r.conn.SelectContext(ctx, &namespaces, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}
	return namespaces, total, nil
}

func (r *pgRepository) UpdateNamespace(ctx context.Context, ns *Namespace) error {
	ns.UpdatedAt = time.Now()
	query := `
		UPDATE namespaces SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
	`
	_, err := r.conn.ExecContext(ctx, query, ns.Name, ns.Description, ns.UpdatedAt, ns.ID)
	return storage.Error(err)
}

func (r *pgRepository) DeleteNamespace(ctx context.Context, id uuid.UUID) error {
	_, err := r.conn.ExecContext(ctx, `DELETE FROM namespaces WHERE id = $1`, id)
	return storage.Error(err)
}

// Compile-time checks
var _ Repository = (*pgRepository)(nil)