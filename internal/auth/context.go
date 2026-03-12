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

	// IsPlatformAdmin grants cross-tenant access for SaaS operations.
	IsPlatformAdmin bool

	// TenantRole is the tenant-level role (e.g., tenant:admin, tenant:member).
	// Empty if the identity only has namespace-level roles.
	TenantRole Role

	// NamespaceRoles maps namespace names to the role granted on each.
	// Only populated for namespace-scoped API keys.
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
