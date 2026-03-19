package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	// ErrJWKSNotAvailable is returned when the JWKS provider failed to initialize.
	ErrJWKSNotAvailable = errors.New("JWKS provider not available")
)

// TenantResolver resolves a tenant ID string from a JWT claim into a
// validated tenant UUID. This decouples the JWT authenticator from the
// database layer — the resolver can look up tenants by external ID, slug,
// or UUID string.
type TenantResolver interface {
	// ResolveTenant maps an external tenant identifier (e.g., Clerk org_id,
	// Auth0 org_id, or a UUID string) to an internal tenant UUID.
	// subjectID is the authenticated user's identity (JWT "sub" claim),
	// used for auto-provisioning with per-user limits.
	// Returns an error if the tenant is unknown or inactive.
	ResolveTenant(ctx context.Context, externalID string, subjectID string) (uuid.UUID, error)
}

// JWTClaimsConfig defines which JWT claims to read for tenant and role info.
// This is what makes the authenticator provider-agnostic — different identity
// providers put org/role info in different claims.
type JWTClaimsConfig struct {
	// TenantClaim is the JWT claim containing the tenant/org identifier.
	// Clerk: "org_id", Auth0: "org_id", custom: whatever you configure.
	// Default: "org_id"
	TenantClaim string

	// RoleClaim is the JWT claim containing the user's role within the org.
	// Clerk: "org_role" (e.g., "org:admin", "org:member")
	// Default: "org_role"
	RoleClaim string

	// SubjectClaim is the JWT claim for the user's unique identifier.
	// Almost always "sub" per the JWT spec. Default: "sub"
	SubjectClaim string

	// NamespaceRolesClaim is the JWT claim containing namespace role assignments.
	// When present and non-empty, namespace roles are extracted from the JWT
	// directly, skipping the DB-based MembershipResolver lookup.
	//
	// Expected format: a JSON array of strings like:
	//   ["namespace:admin:customer-a", "namespace:viewer:customer-b"]
	//
	// This claim is populated by the IdentityProvider sync (e.g., Clerk
	// publicMetadata → session token customization).
	//
	// Default: "namespace_roles" (empty string disables claim-based resolution)
	NamespaceRolesClaim string

	// Issuer is the expected JWT issuer (iss claim). If set, tokens from
	// other issuers are rejected. Leave empty to skip issuer validation.
	Issuer string

	// Audiences is the list of acceptable audience values (aud claim).
	// If empty, audience validation is skipped.
	Audiences []string

	// RoleMapping maps provider-specific role strings to Sparrow roles.
	// Example: {"org:admin": "tenant:admin", "org:member": "tenant:member"}
	// If a role is not in the map, it falls back to the default role.
	RoleMapping map[string]Role

	// DefaultRole is assigned when the JWT contains no role claim or the
	// role is not in the RoleMapping. Default: RoleTenantMember.
	DefaultRole Role

	// ClockSkew is the maximum acceptable clock skew for exp/nbf validation.
	// Default: 30 seconds.
	ClockSkew time.Duration
}

// DefaultJWTClaimsConfig returns a claims config with Clerk-compatible defaults.
func DefaultJWTClaimsConfig() JWTClaimsConfig {
	return JWTClaimsConfig{
		TenantClaim:         "org_id",
		RoleClaim:           "org_role",
		SubjectClaim:        "sub",
		NamespaceRolesClaim: "namespace_roles",
		DefaultRole:         RoleTenantMember,
		ClockSkew:           30 * time.Second,
		RoleMapping: map[string]Role{
			"org:admin":  RoleTenantAdmin,
			"org:member": RoleTenantMember,
		},
	}
}

// JWTAuthenticator validates JWTs using JWKS and maps claims to AuthInfo.
// It is completely provider-agnostic — it works with any identity provider
// that issues RS256 JWTs and publishes a JWKS endpoint.
//
// Uses github.com/golang-jwt/jwt/v5 for parsing and signature verification,
// and github.com/MicahParks/keyfunc/v3 for JWKS key management.
type JWTAuthenticator struct {
	jwks               *JWKSProvider
	claimsConfig       JWTClaimsConfig
	tenantResolver     TenantResolver
	membershipResolver MembershipResolver
	logger             *slog.Logger
}

// JWTAuthenticatorOption configures the JWT authenticator.
type JWTAuthenticatorOption func(*JWTAuthenticator)

// WithClaimsConfig overrides the default claims configuration.
func WithClaimsConfig(cfg JWTClaimsConfig) JWTAuthenticatorOption {
	return func(a *JWTAuthenticator) {
		a.claimsConfig = cfg
	}
}

// WithTenantResolver sets a custom tenant resolver.
// If not set, the authenticator parses the tenant claim as a UUID directly.
func WithTenantResolver(resolver TenantResolver) JWTAuthenticatorOption {
	return func(a *JWTAuthenticator) {
		a.tenantResolver = resolver
	}
}

// WithJWTLogger sets the logger for the JWT authenticator.
func WithJWTLogger(logger *slog.Logger) JWTAuthenticatorOption {
	return func(a *JWTAuthenticator) {
		a.logger = logger
	}
}

// WithMembershipResolver sets the membership resolver for populating
// namespace roles from the database after JWT authentication.
func WithMembershipResolver(resolver MembershipResolver) JWTAuthenticatorOption {
	return func(a *JWTAuthenticator) {
		a.membershipResolver = resolver
	}
}

// NewJWTAuthenticator creates a JWT authenticator backed by the given JWKS provider.
func NewJWTAuthenticator(jwks *JWKSProvider, opts ...JWTAuthenticatorOption) *JWTAuthenticator {
	a := &JWTAuthenticator{
		jwks:         jwks,
		claimsConfig: DefaultJWTClaimsConfig(),
		logger:       slog.Default(),
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Scheme returns "Bearer" — JWTs are sent as Bearer tokens.
func (a *JWTAuthenticator) Scheme() string {
	return "Bearer"
}

// Authenticate validates a JWT and returns the corresponding AuthInfo.
//
// The authentication flow:
//  1. Parse and verify the JWT using golang-jwt/jwt with JWKS keyfunc
//  2. Validate standard claims (exp, nbf, iss, aud) with clock skew tolerance
//  3. Extract tenant and role from configured claims
//  4. Resolve the tenant ID (via TenantResolver or direct UUID parse)
//  5. Map the role string to a Sparrow Role
//  6. Build and return AuthInfo
func (a *JWTAuthenticator) Authenticate(ctx context.Context, credential string) (*AuthInfo, error) {
	// Don't try to authenticate API keys
	if strings.HasPrefix(credential, "sk_") {
		return nil, fmt.Errorf("not a JWT")
	}

	// Build parser options for standard claim validation
	parserOpts := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256"}),
		jwt.WithLeeway(a.claimsConfig.ClockSkew),
		jwt.WithExpirationRequired(),
	}
	if a.claimsConfig.Issuer != "" {
		parserOpts = append(parserOpts, jwt.WithIssuer(a.claimsConfig.Issuer))
	}
	if len(a.claimsConfig.Audiences) > 0 {
		// jwt.WithAudience checks if ANY of the token's audiences match
		for _, aud := range a.claimsConfig.Audiences {
			parserOpts = append(parserOpts, jwt.WithAudience(aud))
			break // jwt library checks "at least one matches"
		}
	}

	// Parse and verify the JWT
	token, err := jwt.Parse(credential, a.jwks.Keyfunc(), parserOpts...)
	if err != nil {
		return nil, fmt.Errorf("jwt verification failed: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid JWT claims")
	}

	// Extract tenant identifier from claims
	tenantExtID, _ := claims[a.claimsConfig.TenantClaim].(string)
	if tenantExtID == "" {
		return nil, fmt.Errorf("jwt missing required claim %q", a.claimsConfig.TenantClaim)
	}

	// Extract subject (user ID) from claims
	subjectID, _ := claims[a.claimsConfig.SubjectClaim].(string)

	// Resolve tenant ID
	var tenantID uuid.UUID
	if a.tenantResolver != nil {
		tenantID, err = a.tenantResolver.ResolveTenant(ctx, tenantExtID, subjectID)
		if err != nil {
			return nil, fmt.Errorf("tenant resolution failed: %w", err)
		}
	} else {
		// Default: try parsing as UUID directly
		tenantID, err = uuid.Parse(tenantExtID)
		if err != nil {
			return nil, fmt.Errorf("tenant claim %q is not a valid UUID: %w", tenantExtID, err)
		}
	}

	// Map role
	role := a.claimsConfig.DefaultRole
	if roleStr, ok := claims[a.claimsConfig.RoleClaim].(string); ok && roleStr != "" {
		if mapped, exists := a.claimsConfig.RoleMapping[roleStr]; exists {
			role = mapped
		}
	}

	// Build AuthInfo
	info := &AuthInfo{
		TenantID:  tenantID,
		SubjectID: subjectID,
	}

	if IsTenantRole(role) {
		info.TenantRole = role
	}

	// Extract expiry from token claims for the AuthInfo
	if exp, ok := claims["exp"].(float64); ok {
		t := time.Unix(int64(exp), 0)
		info.ExpiresAt = &t
	}

	// Log extracted JWT claims for debugging tenant isolation
	a.logger.InfoContext(ctx, "jwt claims extracted",
		slog.String("external_tenant_id", tenantExtID),
		slog.String("resolved_tenant_id", tenantID.String()),
		slog.String("subject_id", subjectID),
		slog.String("role", string(role)),
	)

	// Resolve namespace memberships: prefer JWT claims, fall back to database.
	//
	// When the IdentityProvider sync is active (e.g., Clerk), namespace roles
	// are embedded in the JWT as the namespace_roles claim. This avoids a
	// per-request DB lookup and is the preferred path.
	//
	// When the claim is absent (non-Clerk providers, or roles not yet synced),
	// we fall back to the DB-based MembershipResolver.
	nsRolesResolved := false

	if a.claimsConfig.NamespaceRolesClaim != "" {
		if rolesRaw, ok := claims[a.claimsConfig.NamespaceRolesClaim]; ok {
			if encoded := extractStringSlice(rolesRaw); len(encoded) > 0 {
				nsRoles := DecodeNamespaceRoles(encoded)
				if len(nsRoles) > 0 {
					info.NamespaceRoles = nsRoles
					nsRolesResolved = true
					a.logger.InfoContext(ctx, "namespace roles resolved from JWT claims",
						slog.String("subject_id", subjectID),
						slog.Int("namespace_count", len(nsRoles)),
					)
				}
			}
		}
	}

	// Fall back to DB-based membership resolution if JWT claims didn't provide roles
	if !nsRolesResolved && a.membershipResolver != nil && subjectID != "" {
		nsRoles, err := a.membershipResolver.ResolveNamespaceMemberships(ctx, tenantID, subjectID)
		if err != nil {
			a.logger.WarnContext(ctx, "failed to resolve namespace memberships",
				slog.String("subject_id", subjectID),
				slog.String("tenant_id", tenantID.String()),
				slog.String("error", err.Error()),
			)
			// Non-fatal: fall back to tenant-level access
		} else if len(nsRoles) > 0 {
			info.NamespaceRoles = nsRoles
			a.logger.InfoContext(ctx, "namespace memberships resolved from database",
				slog.String("subject_id", subjectID),
				slog.Int("namespace_count", len(nsRoles)),
			)
		}
	}

	return info, nil
}

// extractStringSlice extracts a []string from a JWT claim value.
// JWT claims are typically deserialized as []interface{} by the JSON parser,
// so we handle both []string and []interface{} (with string elements).
func extractStringSlice(v interface{}) []string {
	switch val := v.(type) {
	case []string:
		return val
	case []interface{}:
		result := make([]string, 0, len(val))
		for _, item := range val {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// Compile-time check
var _ Authenticator = (*JWTAuthenticator)(nil)
