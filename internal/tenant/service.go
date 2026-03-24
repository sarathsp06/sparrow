package tenant

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// DefaultTenantID is the well-known UUID for the default tenant,
// created by the initial database migration.
var DefaultTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// Service provides business logic for tenant management.
type Service struct {
	repo Repository
}

// NewService creates a new tenant service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ---- Tenant operations ----

// CreateTenant creates a new tenant with the given name.
// The slug is derived from the name (lowercased, non-alphanumeric replaced with hyphens).
func (s *Service) CreateTenant(ctx context.Context, name string) (*Tenant, error) {
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
	if id == DefaultTenantID {
		return fmt.Errorf("cannot delete the default tenant")
	}
	return s.repo.DeleteTenant(ctx, id)
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
