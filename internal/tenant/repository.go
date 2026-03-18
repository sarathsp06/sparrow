package tenant

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/auth"
	"github.com/sarathsp06/sparrow/pkg/storage"
)

// Repository defines the data access interface for tenants and API keys.
type Repository interface {
	// WithConn returns a repository that executes queries against the given
	// connection (e.g. a transaction from storage.WithTransaction).
	WithConn(conn storage.DBTX) Repository

	// Tenant operations
	CreateTenant(ctx context.Context, t *Tenant) error
	GetTenantByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error)
	GetTenantByExternalID(ctx context.Context, externalID string) (*Tenant, error)
	CountTenantsByCreator(ctx context.Context, createdBy string) (int, error)
	ListTenants(ctx context.Context, limit, offset int) ([]*Tenant, int, error)
	UpdateTenant(ctx context.Context, t *Tenant) error
	DeleteTenant(ctx context.Context, id uuid.UUID) error

	// API key operations
	CreateAPIKey(ctx context.Context, key *APIKey) error
	GetAPIKeyByID(ctx context.Context, id uuid.UUID) (*APIKey, error)
	ListAPIKeys(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*APIKey, int, error)
	RevokeAPIKey(ctx context.Context, id uuid.UUID) error

	// Implements auth.APIKeyStore for the authenticator
	auth.APIKeyStore

	// Implements auth.TenantLookup for JWT tenant resolution
	auth.TenantLookup
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
		INSERT INTO tenants (id, name, slug, status, settings, external_id, created_by, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5::jsonb, $6, $7, $8, $9)
	`
	_, err := r.conn.ExecContext(ctx, query,
		t.ID, t.Name, t.Slug, t.Status, t.Settings, t.ExternalID, t.CreatedBy, t.CreatedAt, t.UpdatedAt,
	)
	return storage.Error(err)
}

func (r *pgRepository) GetTenantByID(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	var t Tenant
	query := `SELECT id, name, slug, status, settings, external_id, created_by, created_at, updated_at FROM tenants WHERE id = $1`
	err := r.conn.GetContext(ctx, &t, query, id)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &t, nil
}

func (r *pgRepository) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	var t Tenant
	query := `SELECT id, name, slug, status, settings, external_id, created_by, created_at, updated_at FROM tenants WHERE slug = $1`
	err := r.conn.GetContext(ctx, &t, query, slug)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &t, nil
}

func (r *pgRepository) GetTenantByExternalID(ctx context.Context, externalID string) (*Tenant, error) {
	var t Tenant
	query := `SELECT id, name, slug, status, settings, external_id, created_by, created_at, updated_at FROM tenants WHERE external_id = $1`
	err := r.conn.GetContext(ctx, &t, query, externalID)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &t, nil
}

func (r *pgRepository) CountTenantsByCreator(ctx context.Context, createdBy string) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM tenants WHERE created_by = $1`
	err := r.conn.GetContext(ctx, &count, query, createdBy)
	if err != nil {
		return 0, storage.Error(err)
	}
	return count, nil
}

func (r *pgRepository) ListTenants(ctx context.Context, limit, offset int) ([]*Tenant, int, error) {
	var total int
	err := r.conn.GetContext(ctx, &total, `SELECT COUNT(*) FROM tenants`)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	var tenants []*Tenant
	query := `SELECT id, name, slug, status, settings, external_id, created_by, created_at, updated_at FROM tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	err = r.conn.SelectContext(ctx, &tenants, query, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}
	return tenants, total, nil
}

func (r *pgRepository) UpdateTenant(ctx context.Context, t *Tenant) error {
	t.UpdatedAt = time.Now()
	query := `
		UPDATE tenants SET name = $1, slug = $2, status = $3, settings = $4::jsonb, external_id = $5, created_by = $6, updated_at = $7
		WHERE id = $8
	`
	_, err := r.conn.ExecContext(ctx, query, t.Name, t.Slug, t.Status, t.Settings, t.ExternalID, t.CreatedBy, t.UpdatedAt, t.ID)
	return storage.Error(err)
}

func (r *pgRepository) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	_, err := r.conn.ExecContext(ctx, `DELETE FROM tenants WHERE id = $1`, id)
	return storage.Error(err)
}

// ---- API Key operations ----

func (r *pgRepository) CreateAPIKey(ctx context.Context, key *APIKey) error {
	if key.ID == uuid.Nil {
		key.ID = uuid.New()
	}
	key.CreatedAt = time.Now()

	query := `
		INSERT INTO api_keys (id, tenant_id, name, key_prefix, key_hash, role, namespace_scope, is_platform_admin, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.conn.ExecContext(ctx, query,
		key.ID, key.TenantID, key.Name, key.KeyPrefix, key.KeyHash,
		string(key.Role), key.NamespaceScope, key.IsPlatformAdmin,
		key.ExpiresAt, key.CreatedAt,
	)
	return storage.Error(err)
}

func (r *pgRepository) GetAPIKeyByID(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	var key APIKey
	query := `
		SELECT id, tenant_id, name, key_prefix, key_hash, role, namespace_scope,
		       is_platform_admin, expires_at, last_used_at, created_at, revoked_at
		FROM api_keys WHERE id = $1
	`
	err := r.conn.GetContext(ctx, &key, query, id)
	if err != nil {
		return nil, storage.Error(err)
	}
	return &key, nil
}

func (r *pgRepository) ListAPIKeys(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*APIKey, int, error) {
	var total int
	err := r.conn.GetContext(ctx, &total, `SELECT COUNT(*) FROM api_keys WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, 0, storage.Error(err)
	}

	var keys []*APIKey
	query := `
		SELECT id, tenant_id, name, key_prefix, key_hash, role, namespace_scope,
		       is_platform_admin, expires_at, last_used_at, created_at, revoked_at
		FROM api_keys WHERE tenant_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`
	err = r.conn.SelectContext(ctx, &keys, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, storage.Error(err)
	}
	return keys, total, nil
}

func (r *pgRepository) RevokeAPIKey(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_keys SET revoked_at = $1 WHERE id = $2 AND revoked_at IS NULL`
	_, err := r.conn.ExecContext(ctx, query, time.Now(), id)
	return storage.Error(err)
}

// ---- auth.APIKeyStore implementation ----

func (r *pgRepository) GetAPIKeyByHash(ctx context.Context, keyHash string) (*auth.APIKeyRecord, error) {
	var key APIKey
	query := `
		SELECT id, tenant_id, key_hash, role, namespace_scope,
		       is_platform_admin, expires_at, revoked_at
		FROM api_keys WHERE key_hash = $1
	`
	err := r.conn.GetContext(ctx, &key, query, keyHash)
	if err != nil {
		return nil, storage.Error(err)
	}

	return &auth.APIKeyRecord{
		ID:              key.ID,
		TenantID:        key.TenantID,
		KeyHash:         key.KeyHash,
		Role:            key.Role,
		NamespaceScope:  key.NamespaceScope,
		IsPlatformAdmin: key.IsPlatformAdmin,
		ExpiresAt:       key.ExpiresAt,
		RevokedAt:       key.RevokedAt,
	}, nil
}

func (r *pgRepository) UpdateAPIKeyLastUsed(ctx context.Context, keyID uuid.UUID, usedAt time.Time) error {
	query := `UPDATE api_keys SET last_used_at = $1 WHERE id = $2`
	_, err := r.conn.ExecContext(ctx, query, usedAt, keyID)
	return storage.Error(err)
}

// ---- auth.TenantLookup implementation ----

func (r *pgRepository) LookupTenantIDByExternalID(ctx context.Context, externalID string) (uuid.UUID, error) {
	var tenantID uuid.UUID
	query := `SELECT id FROM tenants WHERE external_id = $1 AND status = 'active'`
	err := r.conn.GetContext(ctx, &tenantID, query, externalID)
	if err != nil {
		return uuid.Nil, storage.Error(err)
	}
	return tenantID, nil
}

// Compile-time checks
var _ Repository = (*pgRepository)(nil)
var _ auth.APIKeyStore = (*pgRepository)(nil)
var _ auth.TenantLookup = (*pgRepository)(nil)
