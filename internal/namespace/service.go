package namespace

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Service provides business logic for namespace management.
type Service struct {
	repo   Repository
	logger *slog.Logger
}

// ServiceOption configures the namespace service.
type ServiceOption func(*Service)


// WithServiceLogger sets the logger for the namespace service.
func WithServiceLogger(logger *slog.Logger) ServiceOption {
	return func(s *Service) {
		s.logger = logger
	}
}

// NewService creates a new namespace service.
func NewService(repo Repository, opts ...ServiceOption) *Service {
	s := &Service{
		repo:   repo,
		logger: slog.Default(),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ---- Namespace operations ----

// CreateNamespace creates a new namespace within a tenant.
func (s *Service) CreateNamespace(ctx context.Context, req CreateNamespaceRequest) (*Namespace, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("namespace name is required")
	}
	if !isValidNamespaceName(req.Name) {
		return nil, fmt.Errorf("namespace name must be lowercase alphanumeric with hyphens or underscores")
	}

	ns := &Namespace{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Description: req.Description,
	}

	if err := s.repo.CreateNamespace(ctx, ns); err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}
	return ns, nil
}

// GetNamespace retrieves a namespace by ID.
func (s *Service) GetNamespace(ctx context.Context, id uuid.UUID) (*Namespace, error) {
	return s.repo.GetNamespaceByID(ctx, id)
}

// GetNamespaceByName retrieves a namespace by tenant and name.
func (s *Service) GetNamespaceByName(ctx context.Context, tenantID uuid.UUID, name string) (*Namespace, error) {
	return s.repo.GetNamespaceByName(ctx, tenantID, name)
}

// ListNamespaces retrieves namespaces for a tenant with pagination.
func (s *Service) ListNamespaces(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]*Namespace, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListNamespaces(ctx, tenantID, limit, offset)
}

// UpdateNamespace updates a namespace's name and/or description.
func (s *Service) UpdateNamespace(ctx context.Context, req UpdateNamespaceRequest) (*Namespace, error) {
	ns, err := s.repo.GetNamespaceByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}

	// Verify tenant ownership
	if ns.TenantID != req.TenantID {
		return nil, fmt.Errorf("namespace not found")
	}

	if req.Name != "" {
		if !isValidNamespaceName(req.Name) {
			return nil, fmt.Errorf("namespace name must be lowercase alphanumeric with hyphens or underscores")
		}
		ns.Name = req.Name
	}
	if req.Description != "" {
		ns.Description = req.Description
	}

	if err := s.repo.UpdateNamespace(ctx, ns); err != nil {
		return nil, fmt.Errorf("update namespace: %w", err)
	}
	return ns, nil
}

// DeleteNamespace deletes a namespace by ID.
func (s *Service) DeleteNamespace(ctx context.Context, tenantID, id uuid.UUID) error {
	ns, err := s.repo.GetNamespaceByID(ctx, id)
	if err != nil {
		return err
	}
	if ns.TenantID != tenantID {
		return fmt.Errorf("namespace not found")
	}
	return s.repo.DeleteNamespace(ctx, id)
}

// ---- Helpers ----

var validNamespaceName = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// isValidNamespaceName checks if a namespace name is valid.
// Must be lowercase, start with a letter, and contain only alphanumeric, hyphens, or underscores.
func isValidNamespaceName(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	return validNamespaceName.MatchString(name)
}
