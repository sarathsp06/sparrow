package namespace

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/auth"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// Repository defines the data access interface for namespaces and memberships.
type Repository interface {
	// WithConn returns a repository that executes queries against the given
	// connection (e.g. a transaction from storage.WithTransaction).
	WithConn(conn storage.DBTX) Repository

	// Namespace operations
	CreateNamespace(ctx context.Context, ns *Namespace) error
	GetNamespaceByID(ctx context.Context, id uuid.UUID) (*Namespace, error)
	GetNamespaceByName(ctx context.Context, tenantID uuid.UUID, name string) (*Namespace, error)
	ListNamespaces(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Namespace, int, error)
	UpdateNamespace(ctx context.Context, ns *Namespace) error
	DeleteNamespace(ctx context.Context, id uuid.UUID) error

	// Membership operations
	UpsertMembership(ctx context.Context, m *Membership) error
	GetMembership(ctx context.Context, tenantID uuid.UUID, subjectID, namespace string) (*Membership, error)
	DeleteMembership(ctx context.Context, tenantID uuid.UUID, subjectID, namespace string) error
	ListMembersByNamespace(ctx context.Context, tenantID uuid.UUID, namespace string, limit, offset int) ([]*Membership, int, error)
	ListNamespacesBySubject(ctx context.Context, tenantID uuid.UUID, subjectID string) ([]*Membership, error)

	// Implements auth.MembershipResolver for the JWT authenticator
	auth.MembershipResolver
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

// ---- Membership operations ----

func (r *pgRepository) UpsertMembership(ctx context.Context, m *Membership) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	now := time.Now()
	m.CreatedAt = now
	m.UpdatedAt = now

	query := `
		INSERT INTO namespace_memberships (id, tenant_id, subject_id, namespace, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (tenant_id, subject_id, namespace)
		DO UPDATE SET role = EXCLUDED.role, updated_at = EXCLUDED.updated_at
	`
	_, err := r.conn.ExecContext(ctx, query,
		m.ID, m.TenantID, m.SubjectID, m.Namespace, string(m.Role), m.CreatedAt, m.UpdatedAt,
	)
	return storage.Error(err)
}

func (r *pgRepository) GetMembership(ctx context.Context, tenantID uuid.UUID, subjectID, namespace string) (*Membership, error) {
	var m Membership
	query := `
		SELECT id, tenant_id, subject_id, namespace, role, created_at, updated_at
		FROM namespace_memberships
		WHERE tenant_id = $1 AND subject_id = $2 AND namespace = $3
	`
	err := r.conn.GetContext(ctx, &m, query, tenantID, subjectID, namespace)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &m, nil
}

func (r *pgRepository) DeleteMembership(ctx context.Context, tenantID uuid.UUID, subjectID, namespace string) error {
	query := `DELETE FROM namespace_memberships WHERE tenant_id = $1 AND subject_id = $2 AND namespace = $3`
	_, err := r.conn.ExecContext(ctx, query, tenantID, subjectID, namespace)
	return storage.Error(err)
}

func (r *pgRepository) ListMembersByNamespace(ctx context.Context, tenantID uuid.UUID, namespace string, limit, offset int) ([]*Membership, int, error) {
	var total int
	err := r.conn.GetContext(ctx, &total,
		`SELECT COUNT(*) FROM namespace_memberships WHERE tenant_id = $1 AND namespace = $2`,
		tenantID, namespace,
	)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	var members []*Membership
	query := `
		SELECT id, tenant_id, subject_id, namespace, role, created_at, updated_at
		FROM namespace_memberships
		WHERE tenant_id = $1 AND namespace = $2
		ORDER BY created_at ASC LIMIT $3 OFFSET $4
	`
	err = r.conn.SelectContext(ctx, &members, query, tenantID, namespace, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}
	return members, total, nil
}

func (r *pgRepository) ListNamespacesBySubject(ctx context.Context, tenantID uuid.UUID, subjectID string) ([]*Membership, error) {
	var memberships []*Membership
	query := `
		SELECT id, tenant_id, subject_id, namespace, role, created_at, updated_at
		FROM namespace_memberships
		WHERE tenant_id = $1 AND subject_id = $2
		ORDER BY namespace ASC
	`
	err := r.conn.SelectContext(ctx, &memberships, query, tenantID, subjectID)
	if err != nil {
		return nil, storage.Error(err)
	}
	return memberships, nil
}

// ---- auth.MembershipResolver implementation ----

// ResolveNamespaceMemberships returns the namespace roles for a user within a tenant.
// Returns nil map if the user has no namespace memberships (tenant-wide access).
func (r *pgRepository) ResolveNamespaceMemberships(ctx context.Context, tenantID uuid.UUID, subjectID string) (map[string]auth.Role, error) {
	memberships, err := r.ListNamespacesBySubject(ctx, tenantID, subjectID)
	if err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return nil, nil
	}
	roles := make(map[string]auth.Role, len(memberships))
	for _, m := range memberships {
		roles[m.Namespace] = m.Role
	}
	return roles, nil
}

// Compile-time checks
var _ Repository = (*pgRepository)(nil)
var _ auth.MembershipResolver = (*pgRepository)(nil)
