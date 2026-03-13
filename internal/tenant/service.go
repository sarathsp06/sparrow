package tenant

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/auth"
)

// Service provides business logic for tenant and API key management.
type Service struct {
	repo Repository
}

// NewService creates a new tenant service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ---- Tenant operations ----

// CreateTenantOpts holds optional parameters for tenant creation.
type CreateTenantOpts struct {
	ExternalID *string // External identity provider org ID (e.g., Clerk org_id)
	CreatedBy  *string // Identity provider user ID (JWT sub) who is creating this tenant
}

// CreateTenant creates a new tenant with the given name.
// The slug is derived from the name (lowercased, non-alphanumeric replaced with hyphens).
func (s *Service) CreateTenant(ctx context.Context, name string, opts ...CreateTenantOpts) (*Tenant, error) {
	if name == "" {
		return nil, fmt.Errorf("tenant name is required")
	}

	slug := slugify(name)
	if slug == "" {
		return nil, fmt.Errorf("tenant name must contain at least one alphanumeric character")
	}

	t := &Tenant{
		Name:   name,
		Slug:   slug,
		Status: StatusActive,
	}

	if len(opts) > 0 {
		if opts[0].ExternalID != nil {
			t.ExternalID = opts[0].ExternalID
		}
		if opts[0].CreatedBy != nil {
			t.CreatedBy = opts[0].CreatedBy
		}
	}

	if err := s.repo.CreateTenant(ctx, t); err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}
	return t, nil
}

// GetTenant retrieves a tenant by ID.
func (s *Service) GetTenant(ctx context.Context, id uuid.UUID) (*Tenant, error) {
	return s.repo.GetTenantByID(ctx, id)
}

// GetTenantBySlug retrieves a tenant by slug.
func (s *Service) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	return s.repo.GetTenantBySlug(ctx, slug)
}

// ListTenants retrieves tenants with pagination.
func (s *Service) ListTenants(ctx context.Context, limit, offset int) ([]*Tenant, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListTenants(ctx, limit, offset)
}

// UpdateTenant updates a tenant's name and/or status.
func (s *Service) UpdateTenant(ctx context.Context, id uuid.UUID, name, status string) (*Tenant, error) {
	t, err := s.repo.GetTenantByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if name != "" {
		t.Name = name
		t.Slug = slugify(name)
	}
	if status != "" {
		switch status {
		case StatusActive, StatusSuspended, StatusArchived:
			t.Status = status
		default:
			return nil, fmt.Errorf("invalid tenant status: %s", status)
		}
	}

	if err := s.repo.UpdateTenant(ctx, t); err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}
	return t, nil
}

// DeleteTenant deletes a tenant by ID. This cascades to all related data.
func (s *Service) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	// Prevent deleting the default tenant
	if id == auth.DefaultTenantID {
		return fmt.Errorf("cannot delete the default tenant")
	}
	return s.repo.DeleteTenant(ctx, id)
}

// ---- API Key operations ----

// CreateAPIKey generates a new API key for the given tenant.
// Returns the key record and the plaintext key (shown only once).
func (s *Service) CreateAPIKey(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResult, error) {
	// Validate role
	if !auth.IsValidRole(string(req.Role)) {
		return nil, fmt.Errorf("invalid role: %s", req.Role)
	}

	// Validate namespace scope
	if auth.IsNamespaceRole(req.Role) && (req.NamespaceScope == nil || *req.NamespaceScope == "") {
		return nil, fmt.Errorf("namespace-scoped role %s requires a namespace_scope", req.Role)
	}
	if auth.IsTenantRole(req.Role) && req.NamespaceScope != nil {
		return nil, fmt.Errorf("tenant-scoped role %s must not have a namespace_scope", req.Role)
	}

	if req.Name == "" {
		return nil, fmt.Errorf("API key name is required")
	}

	// Look up the tenant to get its slug for the key prefix
	t, err := s.repo.GetTenantByID(ctx, req.TenantID)
	if err != nil {
		return nil, fmt.Errorf("tenant lookup: %w", err)
	}

	// Generate the raw key: sk_<slug>_<32 random hex chars>
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return nil, fmt.Errorf("generate random key: %w", err)
	}
	prefix := auth.GenerateAPIKeyPrefix(t.Slug)
	rawKey := prefix + hex.EncodeToString(randomBytes)
	keyHash := auth.HashAPIKey(rawKey)

	key := &APIKey{
		TenantID:        req.TenantID,
		Name:            req.Name,
		KeyPrefix:       prefix,
		KeyHash:         keyHash,
		Role:            req.Role,
		NamespaceScope:  req.NamespaceScope,
		IsPlatformAdmin: req.IsPlatformAdmin,
		ExpiresAt:       req.ExpiresAt,
	}

	if err := s.repo.CreateAPIKey(ctx, key); err != nil {
		return nil, fmt.Errorf("create API key: %w", err)
	}

	return &CreateAPIKeyResult{
		Key:    key,
		RawKey: rawKey,
	}, nil
}

// GetAPIKey retrieves an API key by ID.
func (s *Service) GetAPIKey(ctx context.Context, id uuid.UUID) (*APIKey, error) {
	return s.repo.GetAPIKeyByID(ctx, id)
}

// ListAPIKeys retrieves API keys for a tenant with pagination.
func (s *Service) ListAPIKeys(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*APIKey, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListAPIKeys(ctx, tenantID, limit, offset)
}

// RevokeAPIKey revokes an API key by ID, preventing further use.
func (s *Service) RevokeAPIKey(ctx context.Context, id uuid.UUID) error {
	return s.repo.RevokeAPIKey(ctx, id)
}

// ---- Helpers ----

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// slugify converts a string to a URL-safe slug.
func slugify(s string) string {
	slug := strings.ToLower(strings.TrimSpace(s))
	slug = nonAlphanumeric.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	return slug
}
