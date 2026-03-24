package tenant

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/pkg/storage"
)

// Repository defines the data access interface for tenants.
type Repository interface {
	// WithConn returns a repository that executes queries against the given
	// connection (e.g. a transaction from storage.WithTransaction).
	WithConn(conn storage.DBTX) Repository

	// Tenant operations
	CreateTenant(ctx context.Context, t *Tenant) error
	GetTenantByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error)
	ListTenants(ctx context.Context, limit, offset int) ([]*Tenant, int, error)
	UpdateTenant(ctx context.Context, t *Tenant) error
	DeleteTenant(ctx context.Context, id uuid.UUID) error
}

// pgRepository implements Repository using PostgreSQL via sqlx.
type pgRepository struct {
	db   storage.DB   // full connection — used for Beginx/Ping/Close
	conn storage.DBTX // query/exec target — either db or a transaction
}

// NewRepository creates a new PostgreSQL-backed tenant repository.
func NewRepository(db storage.DB) Repository {
	return &pgRepository{db: db, conn: db}
}

// WithConn returns a shallow copy that runs queries against conn.
func (r *pgRepository) WithConn(conn storage.DBTX) Repository {
	return &pgRepository{db: r.db, conn: conn}
}

// ---- Tenant operations ----

func (r *pgRepository) CreateTenant(ctx context.Context, t *Tenant) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Status == "" {
		t.Status = StatusActive
	}
	if t.Settings == "" {
		t.Settings = "{}"
	}
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now

	query := `
		INSERT INTO tenants (id, name, slug, status, settings, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7)
	`
	_, err := r.conn.ExecContext(ctx, query,
		t.ID, t.Name, t.Slug, t.Status, t.Settings, t.CreatedAt, t.UpdatedAt,
	)
	return storage.Error(err)
}

func (r *pgRepository) GetTenantByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	var t Tenant
	query := `SELECT id, name, slug, status, settings, created_at, updated_at FROM tenants WHERE id = $1`
	err := r.conn.GetContext(ctx, &t, query, id)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &t, nil
}

func (r *pgRepository) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	var t Tenant
	query := `SELECT id, name, slug, status, settings, created_at, updated_at FROM tenants WHERE slug = $1`
	err := r.conn.GetContext(ctx, &t, query, slug)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &t, nil
}

func (r *pgRepository) ListTenants(ctx context.Context, limit, offset int) ([]*Tenant, int, error) {
	var total int
	err := r.conn.GetContext(ctx, &total, `SELECT COUNT(*) FROM tenants`)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	var tenants []*Tenant
	query := `SELECT id, name, slug, status, settings, created_at, updated_at FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	err = r.conn.SelectContext(ctx, &tenants, query, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}
	return tenants, total, nil
}

func (r *pgRepository) UpdateTenant(ctx context.Context, t *Tenant) error {
	t.UpdatedAt = time.Now()
	query := `
		UPDATE tenants SET name = $1, slug = $2, status = $3, settings = $4::jsonb, updated_at = $5
		WHERE id = $6
	`
	_, err := r.conn.ExecContext(ctx, query, t.Name, t.Slug, t.Status, t.Settings, t.UpdatedAt, t.ID)
	return storage.Error(err)
}

func (r *pgRepository) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	_, err := r.conn.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	return storage.Error(err)
}

// Compile-time check
var _ Repository = (*pgRepository)(nil)
