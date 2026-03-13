package auth

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
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
}

// DefaultJWTClaimsConfig returns a claims config with Clerk-compatible defaults.
func DefaultJWTClaimsConfig() JWTClaimsConfig {
	return JWTClaimsConfig{
		TenantClaim:  "org_id",
		RoleClaim:    "org_role",
		SubjectClaim: "sub",
		DefaultRole:  RoleTenantMember,
		RoleMapping: map[string]Role{
			"org:admin":  RoleTenantAdmin,
			"org:member": RoleTenantMember,
		},
	}
}

// JWTAuthenticator validates JWTs using JWKS and maps claims to AuthInfo.
// It is completely provider-agnostic — it works with any identity provider
// that issues RS256 JWTs and publishes a JWKS endpoint.
type JWTAuthenticator struct {
	jwks           *JWKSProvider
	claimsConfig   JWTClaimsConfig
	tenantResolver TenantResolver
	logger         *slog.Logger
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
//  1. Parse the JWT header to get the key ID (kid) and algorithm
//  2. Fetch the corresponding public key from the JWKS provider
//  3. Verify the JWT signature
//  4. Validate standard claims (exp, nbf, iat, iss, aud)
//  5. Extract tenant and role from configured claims
//  6. Resolve the tenant ID (via TenantResolver or direct UUID parse)
//  7. Map the role string to a Sparrow Role
//  8. Build and return AuthInfo
func (a *JWTAuthenticator) Authenticate(ctx context.Context, credential string) (*AuthInfo, error) {
	// Don't try to authenticate API keys
	if strings.HasPrefix(credential, "sk_") {
		return nil, fmt.Errorf("not a JWT")
	}

	// Parse and verify the JWT
	claims, err := a.verifyJWT(ctx, credential)
	if err != nil {
		return nil, fmt.Errorf("jwt verification failed: %w", err)
	}

	// Validate standard claims
	if err := a.validateStandardClaims(claims); err != nil {
		return nil, err
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

	return info, nil
}

// verifyJWT parses a JWT, verifies its signature using the JWKS, and returns
// the payload claims. Only RS256 is supported.
func (a *JWTAuthenticator) verifyJWT(ctx context.Context, tokenString string) (map[string]any, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("malformed JWT: expected 3 parts, got %d", len(parts))
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("decode JWT header: %w", err)
	}

	var header jwtHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("parse JWT header: %w", err)
	}

	if header.Alg != "RS256" {
		return nil, fmt.Errorf("unsupported JWT algorithm %q (only RS256 is supported)", header.Alg)
	}

	if header.Kid == "" {
		return nil, fmt.Errorf("JWT header missing kid")
	}

	// Get the public key
	pubKey, err := a.jwks.GetKey(ctx, header.Kid)
	if err != nil {
		return nil, fmt.Errorf("get signing key: %w", err)
	}

	// Verify signature: RS256 = RSASSA-PKCS1-v1_5 with SHA-256
	signingInput := parts[0] + "." + parts[1]
	signatureBytes, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("decode JWT signature: %w", err)
	}

	hash := sha256.Sum256([]byte(signingInput))
	if err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, hash[:], signatureBytes); err != nil {
		return nil, fmt.Errorf("invalid JWT signature")
	}

	// Decode and return payload claims
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("decode JWT payload: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, fmt.Errorf("parse JWT payload: %w", err)
	}

	return claims, nil
}

// validateStandardClaims checks exp, nbf, iss, and aud.
func (a *JWTAuthenticator) validateStandardClaims(claims map[string]any) error {
	now := time.Now()

	// Check expiration (required)
	exp, ok := claims["exp"].(float64)
	if !ok {
		return fmt.Errorf("jwt missing required exp claim")
	}
	if time.Unix(int64(exp), 0).Before(now) {
		return fmt.Errorf("jwt has expired")
	}

	// Check not-before (optional)
	if nbf, ok := claims["nbf"].(float64); ok {
		if time.Unix(int64(nbf), 0).After(now) {
			return fmt.Errorf("jwt is not yet valid (nbf)")
		}
	}

	// Check issuer (if configured)
	if a.claimsConfig.Issuer != "" {
		iss, _ := claims["iss"].(string)
		if iss != a.claimsConfig.Issuer {
			return fmt.Errorf("jwt issuer %q does not match expected %q", iss, a.claimsConfig.Issuer)
		}
	}

	// Check audience (if configured)
	if len(a.claimsConfig.Audiences) > 0 {
		if !a.audienceMatches(claims) {
			return fmt.Errorf("jwt audience does not match any expected audience")
		}
	}

	return nil
}

// audienceMatches checks if any of the token's audiences match the configured ones.
func (a *JWTAuthenticator) audienceMatches(claims map[string]any) bool {
	audClaim, ok := claims["aud"]
	if !ok {
		return false
	}

	// aud can be a string or an array of strings
	var tokenAudiences []string
	switch v := audClaim.(type) {
	case string:
		tokenAudiences = []string{v}
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				tokenAudiences = append(tokenAudiences, s)
			}
		}
	default:
		return false
	}

	for _, expected := range a.claimsConfig.Audiences {
		for _, actual := range tokenAudiences {
			if expected == actual {
				return true
			}
		}
	}
	return false
}

// ---- JWT structures ----

type jwtHeader struct {
	Alg string `json:"alg"`
	Typ string `json:"typ"`
	Kid string `json:"kid"`
}

// Compile-time check
var _ Authenticator = (*JWTAuthenticator)(nil)
