package auth

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// IdentityProvider is the interface for syncing namespace role assignments
// to an external identity provider (e.g., Clerk, Auth0). This enables
// namespace roles to be embedded directly in JWTs, eliminating the need
// for a per-request database lookup.
//
// The database remains the source of truth for namespace memberships.
// The identity provider sync is best-effort: if it fails, the system
// falls back to database-based membership resolution on the next request.
//
// Flow:
//  1. Namespace membership is written to the database (source of truth)
//  2. All of the user's namespace roles are re-read from the database
//  3. SyncNamespaceRoles pushes the complete role set to the identity provider
//  4. On the next JWT refresh, the roles appear in the token claims
//  5. JWTAuthenticator reads roles from claims, skipping the DB lookup
type IdentityProvider interface {
	// SyncNamespaceRoles pushes a user's complete set of namespace roles
	// to the identity provider so they appear in future JWT tokens.
	//
	// Parameters:
	//   - externalTenantID: the identity provider's organization/tenant ID
	//     (e.g., Clerk org_id). This is NOT the internal Sparrow tenant UUID.
	//   - subjectID: the identity provider's user ID (JWT "sub" claim)
	//   - roles: the complete namespace→role map for this user within this tenant.
	//     An empty map means the user has no namespace-specific roles (tenant-wide access).
	//     A nil map is treated the same as empty.
	//
	// The method is idempotent: calling it with the same roles has no effect.
	// Implementations should handle this gracefully (e.g., skip the API call
	// if the roles haven't changed).
	//
	// Errors are non-fatal. Callers should log the error but not fail the
	// operation — the database write has already succeeded.
	SyncNamespaceRoles(ctx context.Context, externalTenantID string, subjectID string, roles map[string]Role) error

	// TeamManagement returns the team management interface if the provider
	// supports org-level member and invitation management.
	// Returns nil if not supported (e.g., NoopIdentityProvider).
	TeamManagement() TeamManager
}

// TeamManager provides organization-level team management operations:
// listing members, inviting/removing members, changing roles, and
// managing invitations. Backed by the identity provider (e.g., Clerk).
type TeamManager interface {
	// ListMembers lists all members of an organization.
	ListMembers(ctx context.Context, externalOrgID string, limit, offset int) ([]TeamMember, int, error)

	// InviteMember invites a new member to the organization by email.
	InviteMember(ctx context.Context, externalOrgID, email, role string) (*TeamInvitation, error)

	// RemoveMember removes a member from the organization.
	RemoveMember(ctx context.Context, externalOrgID, userID string) error

	// UpdateMemberRole changes a member's organization-level role.
	UpdateMemberRole(ctx context.Context, externalOrgID, userID, role string) (*TeamMember, error)

	// ListInvitations lists invitations for an organization, optionally filtered by status.
	ListInvitations(ctx context.Context, externalOrgID string, status string, limit, offset int) ([]TeamInvitation, int, error)

	// RevokeInvitation revokes a pending invitation.
	RevokeInvitation(ctx context.Context, externalOrgID, invitationID string) error
}

// TeamMember represents a member of an organization with profile info.
type TeamMember struct {
	UserID    string
	FirstName string
	LastName  string
	Email     string
	ImageURL  string
	Role      string
	JoinedAt  time.Time
}

// TeamInvitation represents a pending organization invitation.
type TeamInvitation struct {
	ID        string
	Email     string
	Role      string
	Status    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// ExternalTenantLookup is a narrow interface for looking up the external
// identity provider ID for a Sparrow tenant. This is needed when syncing
// roles to the identity provider — we need to map the internal tenant UUID
// back to the external org_id.
type ExternalTenantLookup interface {
	// LookupExternalIDByTenantID returns the external identity provider ID
	// (e.g., Clerk org_id) for the given internal tenant UUID.
	LookupExternalIDByTenantID(ctx context.Context, tenantID uuid.UUID) (string, error)
}

// NoopIdentityProvider is a no-op implementation that does nothing.
// Used when no external identity provider is configured (e.g., API-key-only
// deployments, or identity providers that don't support metadata sync).
type NoopIdentityProvider struct {
	logger *slog.Logger
}

// NewNoopIdentityProvider creates a no-op identity provider.
func NewNoopIdentityProvider(logger *slog.Logger) *NoopIdentityProvider {
	if logger == nil {
		logger = slog.Default()
	}
	return &NoopIdentityProvider{logger: logger}
}

// SyncNamespaceRoles is a no-op — roles are resolved from the database only.
func (p *NoopIdentityProvider) SyncNamespaceRoles(_ context.Context, _ string, _ string, _ map[string]Role) error {
	return nil
}

// TeamManagement returns nil — no team management without an identity provider.
func (p *NoopIdentityProvider) TeamManagement() TeamManager {
	return nil
}

// Compile-time checks
var _ IdentityProvider = (*NoopIdentityProvider)(nil)
