package namespace

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/auth"
)

// Service provides business logic for namespace and membership management.
type Service struct {
	repo             Repository
	identityProvider auth.IdentityProvider
	tenantLookup     auth.ExternalTenantLookup
	logger           *slog.Logger
}

// ServiceOption configures the namespace service.
type ServiceOption func(*Service)

// WithIdentityProvider sets the identity provider for syncing namespace roles
// to external identity providers (e.g., Clerk).
func WithIdentityProvider(provider auth.IdentityProvider) ServiceOption {
	return func(s *Service) {
		s.identityProvider = provider
	}
}

// WithExternalTenantLookup sets the tenant lookup for mapping internal tenant
// UUIDs to external identity provider IDs.
func WithExternalTenantLookup(lookup auth.ExternalTenantLookup) ServiceOption {
	return func(s *Service) {
		s.tenantLookup = lookup
	}
}

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

// ---- Membership operations ----

// AssignNamespaceRole assigns a user a role on a specific namespace.
// If the user already has a role on that namespace, it is updated (upsert).
//
// After the DB write succeeds, if an IdentityProvider is configured, the
// user's complete namespace role set is synced to the external provider
// (best-effort, non-fatal).
func (s *Service) AssignNamespaceRole(ctx context.Context, req AssignMembershipRequest) (*Membership, error) {
	if req.SubjectID == "" {
		return nil, fmt.Errorf("subject_id is required")
	}
	if req.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if !auth.IsNamespaceRole(req.Role) {
		return nil, fmt.Errorf("invalid namespace role: %s (must be namespace:admin, namespace:member, or namespace:viewer)", req.Role)
	}

	// Verify the namespace exists
	if _, err := s.repo.GetNamespaceByName(ctx, req.TenantID, req.Namespace); err != nil {
		return nil, fmt.Errorf("namespace %q not found", req.Namespace)
	}

	m := &Membership{
		TenantID:  req.TenantID,
		SubjectID: req.SubjectID,
		Namespace: req.Namespace,
		Role:      req.Role,
	}

	if err := s.repo.UpsertMembership(ctx, m); err != nil {
		return nil, fmt.Errorf("assign namespace role: %w", err)
	}

	// Sync to identity provider (best-effort)
	s.syncRolesToProvider(ctx, req.TenantID, req.SubjectID)

	return m, nil
}

// RemoveNamespaceRole removes a user's role from a namespace.
//
// After the DB delete succeeds, if an IdentityProvider is configured, the
// user's updated namespace role set is synced to the external provider
// (best-effort, non-fatal).
func (s *Service) RemoveNamespaceRole(ctx context.Context, tenantID uuid.UUID, subjectID, namespace string) error {
	if subjectID == "" {
		return fmt.Errorf("subject_id is required")
	}
	if namespace == "" {
		return fmt.Errorf("namespace is required")
	}
	if err := s.repo.DeleteMembership(ctx, tenantID, subjectID, namespace); err != nil {
		return err
	}

	// Sync to identity provider (best-effort)
	s.syncRolesToProvider(ctx, tenantID, subjectID)

	return nil
}

// ListNamespaceMembers lists all members of a namespace.
func (s *Service) ListNamespaceMembers(ctx context.Context, tenantID uuid.UUID, namespace string, limit, offset int) ([]*Membership, int, error) {
	if namespace == "" {
		return nil, 0, fmt.Errorf("namespace is required")
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListMembersByNamespace(ctx, tenantID, namespace, limit, offset)
}

// GetUserNamespaces returns all namespace memberships for a user.
func (s *Service) GetUserNamespaces(ctx context.Context, tenantID uuid.UUID, subjectID string) ([]*Membership, error) {
	if subjectID == "" {
		return nil, fmt.Errorf("subject_id is required")
	}
	return s.repo.ListNamespacesBySubject(ctx, tenantID, subjectID)
}

// ---- Identity provider sync ----

// syncRolesToProvider reads the user's complete namespace roles from the DB
// and pushes them to the identity provider. This runs async (fire-and-forget)
// so it doesn't block the RPC response.
//
// If no identity provider or tenant lookup is configured, this is a no-op.
func (s *Service) syncRolesToProvider(ctx context.Context, tenantID uuid.UUID, subjectID string) {
	if s.identityProvider == nil || s.tenantLookup == nil {
		return
	}

	// Run async so the RPC doesn't wait for the Clerk API call.
	// Use context.WithoutCancel so the sync completes even if the
	// request context is cancelled.
	go func() {
		syncCtx := context.WithoutCancel(ctx)

		// Look up the external org ID for this tenant
		externalID, err := s.tenantLookup.LookupExternalIDByTenantID(syncCtx, tenantID)
		if err != nil {
			s.logger.WarnContext(syncCtx, "failed to lookup external tenant ID for role sync",
				slog.String("tenant_id", tenantID.String()),
				slog.String("subject_id", subjectID),
				slog.String("error", err.Error()),
			)
			return
		}
		if externalID == "" {
			s.logger.DebugContext(syncCtx, "tenant has no external ID, skipping role sync",
				slog.String("tenant_id", tenantID.String()),
			)
			return
		}

		// Re-read all of this user's namespace roles from DB (source of truth)
		memberships, err := s.repo.ListNamespacesBySubject(syncCtx, tenantID, subjectID)
		if err != nil {
			s.logger.WarnContext(syncCtx, "failed to read namespace roles for sync",
				slog.String("tenant_id", tenantID.String()),
				slog.String("subject_id", subjectID),
				slog.String("error", err.Error()),
			)
			return
		}

		// Convert memberships to a role map
		roles := make(map[string]auth.Role, len(memberships))
		for _, m := range memberships {
			roles[m.Namespace] = m.Role
		}

		// Push to identity provider
		if err := s.identityProvider.SyncNamespaceRoles(syncCtx, externalID, subjectID, roles); err != nil {
			s.logger.WarnContext(syncCtx, "failed to sync namespace roles to identity provider",
				slog.String("tenant_id", tenantID.String()),
				slog.String("subject_id", subjectID),
				slog.String("external_id", externalID),
				slog.String("error", err.Error()),
			)
		}
	}()
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
