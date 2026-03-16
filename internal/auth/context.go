package auth

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// contextKey is an unexported type used for context keys in this package.
type contextKey struct{}

// authInfoKey is the context key for AuthInfo.
var authInfoKey = contextKey{}

// AuthInfo carries the authenticated identity through request context.
// It is injected by the auth interceptor and consumed by service methods.
type AuthInfo struct {
	// TenantID is the tenant this identity belongs to.
	TenantID uuid.UUID

	// SubjectID is the unique user identifier from the identity provider
	// (JWT "sub" claim). Empty for API key auth.
	SubjectID string

	// IsPlatformAdmin grants cross-tenant access for SaaS operations.
	IsPlatformAdmin bool

	// TenantRole is the tenant-level role (e.g., tenant:admin, tenant:member).
	// Empty if the identity only has namespace-level roles.
	TenantRole Role

	// NamespaceRoles maps namespace names to the role granted on each.
	// Populated from namespace-scoped API keys or from namespace memberships
	// resolved during JWT authentication.
	NamespaceRoles map[string]Role

	// KeyID is the API key ID that was used to authenticate (nil for OIDC sessions).
	KeyID *uuid.UUID

	// ExpiresAt is when this auth session expires (if applicable).
	ExpiresAt *time.Time
}

// NewContext returns a new context with the given AuthInfo.
func NewContext(ctx context.Context, info *AuthInfo) context.Context {
	return context.WithValue(ctx, authInfoKey, info)
}

// FromContext extracts AuthInfo from the context.
// Returns nil if no auth info is present.
func FromContext(ctx context.Context) *AuthInfo {
	info, _ := ctx.Value(authInfoKey).(*AuthInfo)
	return info
}

// MustFromContext extracts AuthInfo from the context or panics.
// Use only when the auth interceptor guarantees presence.
func MustFromContext(ctx context.Context) *AuthInfo {
	info := FromContext(ctx)
	if info == nil {
		panic("auth: no AuthInfo in context — was the auth interceptor applied?")
	}
	return info
}

// Require is a convenience method that checks if this identity has the
// required permission, optionally scoped to a namespace.
func (a *AuthInfo) Require(perm Permission, namespace string) error {
	return Authorize(a, perm, namespace)
}

// HasTenantWideAccess returns true if the identity can access ALL namespaces.
// Platform admins always have tenant-wide access.
// Users with a tenant role BUT with namespace memberships are restricted to
// those namespaces and do NOT have tenant-wide access.
// Users with a tenant role and NO namespace memberships have tenant-wide access
// (backward compatible).
func (a *AuthInfo) hasTenantWideAccess() bool {
	if a.IsPlatformAdmin {
		return true
	}
	// If user has namespace memberships, they are scoped — no tenant-wide access
	if len(a.NamespaceRoles) > 0 {
		return false
	}
	return a.TenantRole != ""
}

// CanAccessNamespace checks if the identity has access to the given namespace
// with at least the specified permission. Platform admins can access any namespace.
// For users with namespace memberships, only the membership role is checked.
// For users without memberships, tenant role provides access.
// When namespace is empty (cross-namespace query), users with memberships are
// allowed but the service layer must filter results to their accessible namespaces.
func (a *AuthInfo) CanAccessNamespace(namespace string, perm Permission) error {
	if namespace == "" {
		// Cross-namespace query. Platform admins and users with tenant roles
		// (and no memberships) can query freely. Users WITH memberships can
		// also perform cross-namespace queries, but service layer must apply
		// AccessibleNamespaces() filtering.
		if a.IsPlatformAdmin {
			return nil
		}
		if len(a.NamespaceRoles) > 0 {
			// Allow cross-namespace queries for users with memberships,
			// but only if any of their namespace roles grant the permission.
			for _, role := range a.NamespaceRoles {
				if RoleHasPermission(role, perm) {
					return nil
				}
			}
			return &PermissionDeniedError{Permission: perm, Namespace: ""}
		}
		// No memberships — use tenant role
		if a.TenantRole != "" && RoleHasPermission(a.TenantRole, perm) {
			return nil
		}
		return &PermissionDeniedError{Permission: perm, Namespace: ""}
	}
	return Authorize(a, perm, namespace)
}

// AccessibleNamespaces returns the list of namespaces the identity has access to.
// Returns nil if the identity has tenant-wide access (all namespaces) — this
// applies to platform admins and users with a tenant role but NO namespace memberships.
// Returns the explicit list of namespaces for namespace-scoped identities
// (API keys with namespace scope, or users with namespace memberships).
func (a *AuthInfo) AccessibleNamespaces() []string {
	if a.hasTenantWideAccess() {
		return nil // nil means "all namespaces"
	}
	namespaces := make([]string, 0, len(a.NamespaceRoles))
	for ns := range a.NamespaceRoles {
		namespaces = append(namespaces, ns)
	}
	return namespaces
}

// DefaultTenantID is the well-known UUID for the auto-bootstrapped default tenant.
var DefaultTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")

// DefaultAuthInfo returns an AuthInfo for the default tenant with tenant:admin role.
// Used when auth is disabled (self-hosted, backwards-compatible mode).
func DefaultAuthInfo() *AuthInfo {
	return &AuthInfo{
		TenantID:   DefaultTenantID,
		TenantRole: RoleTenantAdmin,
	}
}
