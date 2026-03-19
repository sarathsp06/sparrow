package auth

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Test helpers ----

// generateTestKeyPair creates an RSA key pair for testing.
func generateTestKeyPair(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	return key
}

// signJWT creates a signed JWT for testing.
func signJWT(t *testing.T, key *rsa.PrivateKey, kid string, claims map[string]any) string {
	t.Helper()

	header := map[string]string{
		"alg": "RS256",
		"typ": "JWT",
		"kid": kid,
	}

	headerJSON, err := json.Marshal(header)
	require.NoError(t, err)

	claimsJSON, err := json.Marshal(claims)
	require.NoError(t, err)

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	signingInput := headerB64 + "." + claimsB64
	hash := sha256.Sum256([]byte(signingInput))

	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
	require.NoError(t, err)

	signatureB64 := base64.RawURLEncoding.EncodeToString(signature)
	return signingInput + "." + signatureB64
}

// serveJWKS starts an HTTP test server that serves JWKS containing the given key.
func serveJWKS(t *testing.T, key *rsa.PublicKey, kid string) *httptest.Server {
	t.Helper()

	nB64 := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	eB64 := base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.E)).Bytes())

	jwks := map[string]any{
		"keys": []map[string]string{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": kid,
				"alg": "RS256",
				"n":   nB64,
				"e":   eB64,
			},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks) //nolint:errcheck
	}))
	t.Cleanup(srv.Close)
	return srv
}

// mockTenantResolver is a simple in-memory resolver for testing.
type mockTenantResolver struct {
	tenants map[string]uuid.UUID
}

func (r *mockTenantResolver) ResolveTenant(_ context.Context, externalID string, _ string) (uuid.UUID, error) {
	id, ok := r.tenants[externalID]
	if !ok {
		return uuid.Nil, fmt.Errorf("unknown tenant %q", externalID)
	}
	return id, nil
}

// ---- JWKS Provider Tests ----

func TestJWKSProvider_Keyfunc(t *testing.T) {
	key := generateTestKeyPair(t)
	kid := "test-key-1"
	srv := serveJWKS(t, &key.PublicKey, kid)

	provider := NewJWKSProvider(srv.URL)

	t.Run("validates a signed token", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub": "user_1",
			"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		parsed, err := jwt.Parse(token, provider.Keyfunc(), jwt.WithValidMethods([]string{"RS256"}))
		require.NoError(t, err)
		assert.True(t, parsed.Valid)
	})

	t.Run("rejects token signed with wrong key", func(t *testing.T) {
		wrongKey := generateTestKeyPair(t)
		token := signJWT(t, wrongKey, kid, map[string]any{
			"sub": "user_1",
			"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		_, err := jwt.Parse(token, provider.Keyfunc(), jwt.WithValidMethods([]string{"RS256"}))
		require.Error(t, err)
	})
}

// ---- JWT Authenticator Tests ----

func TestJWTAuthenticator_Authenticate(t *testing.T) {
	key := generateTestKeyPair(t)
	kid := "test-key-1"
	srv := serveJWKS(t, &key.PublicKey, kid)
	jwksProvider := NewJWKSProvider(srv.URL)

	tenantID := uuid.New()
	resolver := &mockTenantResolver{
		tenants: map[string]uuid.UUID{
			"org_clerk123": tenantID,
		},
	}

	authn := NewJWTAuthenticator(
		jwksProvider,
		WithTenantResolver(resolver),
	)

	t.Run("scheme is Bearer", func(t *testing.T) {
		assert.Equal(t, "Bearer", authn.Scheme())
	})

	t.Run("valid token with org_id and org_role", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":      "user_abc123",
			"org_id":   "org_clerk123",
			"org_role": "org:admin",
			"exp":      float64(time.Now().Add(1 * time.Hour).Unix()),
			"iat":      float64(time.Now().Unix()),
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.Equal(t, tenantID, info.TenantID)
		assert.Equal(t, RoleTenantAdmin, info.TenantRole)
		assert.NotNil(t, info.ExpiresAt)
		assert.Nil(t, info.KeyID, "JWT auth should not set KeyID")
	})

	t.Run("valid token with member role", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":      "user_def456",
			"org_id":   "org_clerk123",
			"org_role": "org:member",
			"exp":      float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.Equal(t, tenantID, info.TenantID)
		assert.Equal(t, RoleTenantMember, info.TenantRole)
	})

	t.Run("unknown role falls back to default", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":      "user_ghi789",
			"org_id":   "org_clerk123",
			"org_role": "org:custom_role",
			"exp":      float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.Equal(t, RoleTenantMember, info.TenantRole, "should fall back to default role")
	})

	t.Run("missing role claim uses default", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_no_role",
			"org_id": "org_clerk123",
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.Equal(t, RoleTenantMember, info.TenantRole)
	})

	t.Run("expired token is rejected", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_expired",
			"org_id": "org_clerk123",
			"exp":    float64(time.Now().Add(-1 * time.Hour).Unix()),
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("missing exp claim is rejected", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_no_exp",
			"org_id": "org_clerk123",
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "exp")
	})

	t.Run("missing org_id is rejected", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub": "user_no_org",
			"exp": float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "org_id")
	})

	t.Run("unknown tenant is rejected", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_unknown_tenant",
			"org_id": "org_unknown",
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "tenant resolution failed")
	})

	t.Run("API key credential is rejected", func(t *testing.T) {
		_, err := authn.Authenticate(context.Background(), "sk_default_abc123")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a JWT")
	})

	t.Run("malformed JWT is rejected", func(t *testing.T) {
		_, err := authn.Authenticate(context.Background(), "not.a.valid.jwt.at.all")
		require.Error(t, err)
	})

	t.Run("wrong signature is rejected", func(t *testing.T) {
		// Sign with a different key
		wrongKey := generateTestKeyPair(t)
		token := signJWT(t, wrongKey, kid, map[string]any{
			"sub":    "user_wrong_sig",
			"org_id": "org_clerk123",
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "jwt verification failed")
	})
}

func TestJWTAuthenticator_IssuerValidation(t *testing.T) {
	key := generateTestKeyPair(t)
	kid := "test-key-1"
	srv := serveJWKS(t, &key.PublicKey, kid)
	jwksProvider := NewJWKSProvider(srv.URL)

	tenantID := uuid.New()
	resolver := &mockTenantResolver{
		tenants: map[string]uuid.UUID{"org_test": tenantID},
	}

	authn := NewJWTAuthenticator(
		jwksProvider,
		WithTenantResolver(resolver),
		WithClaimsConfig(JWTClaimsConfig{
			TenantClaim:  "org_id",
			RoleClaim:    "org_role",
			SubjectClaim: "sub",
			Issuer:       "https://clerk.example.com",
			DefaultRole:  RoleTenantMember,
			ClockSkew:    30 * time.Second,
			RoleMapping:  map[string]Role{"org:admin": RoleTenantAdmin, "org:member": RoleTenantMember},
		}),
	)

	t.Run("correct issuer passes", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_1",
			"org_id": "org_test",
			"iss":    "https://clerk.example.com",
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.Equal(t, tenantID, info.TenantID)
	})

	t.Run("wrong issuer is rejected", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_2",
			"org_id": "org_test",
			"iss":    "https://evil.example.com",
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "issuer")
	})
}

func TestJWTAuthenticator_AudienceValidation(t *testing.T) {
	key := generateTestKeyPair(t)
	kid := "test-key-1"
	srv := serveJWKS(t, &key.PublicKey, kid)
	jwksProvider := NewJWKSProvider(srv.URL)

	tenantID := uuid.New()
	resolver := &mockTenantResolver{
		tenants: map[string]uuid.UUID{"org_test": tenantID},
	}

	authn := NewJWTAuthenticator(
		jwksProvider,
		WithTenantResolver(resolver),
		WithClaimsConfig(JWTClaimsConfig{
			TenantClaim:  "org_id",
			RoleClaim:    "org_role",
			SubjectClaim: "sub",
			Audiences:    []string{"https://api.sparrow.dev"},
			DefaultRole:  RoleTenantMember,
			ClockSkew:    30 * time.Second,
			RoleMapping:  map[string]Role{},
		}),
	)

	t.Run("matching audience passes", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_1",
			"org_id": "org_test",
			"aud":    "https://api.sparrow.dev",
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
	})

	t.Run("audience array with match passes", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_2",
			"org_id": "org_test",
			"aud":    []string{"https://other.dev", "https://api.sparrow.dev"},
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
	})

	t.Run("wrong audience is rejected", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_3",
			"org_id": "org_test",
			"aud":    "https://wrong.dev",
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "aud")
	})
}

func TestJWTAuthenticator_DirectUUIDTenant(t *testing.T) {
	key := generateTestKeyPair(t)
	kid := "test-key-1"
	srv := serveJWKS(t, &key.PublicKey, kid)
	jwksProvider := NewJWKSProvider(srv.URL)

	// No tenant resolver — should parse org_id as UUID directly
	authn := NewJWTAuthenticator(jwksProvider)

	tenantID := uuid.New()
	token := signJWT(t, key, kid, map[string]any{
		"sub":    "user_1",
		"org_id": tenantID.String(),
		"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
	})

	info, err := authn.Authenticate(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, tenantID, info.TenantID)
}

func TestJWTAuthenticator_NbfValidation(t *testing.T) {
	key := generateTestKeyPair(t)
	kid := "test-key-1"
	srv := serveJWKS(t, &key.PublicKey, kid)
	jwksProvider := NewJWKSProvider(srv.URL)

	tenantID := uuid.New()
	authn := NewJWTAuthenticator(jwksProvider)

	t.Run("future nbf is rejected", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_1",
			"org_id": tenantID.String(),
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
			"nbf":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not valid yet")
	})

	t.Run("past nbf passes", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_2",
			"org_id": tenantID.String(),
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
			"nbf":    float64(time.Now().Add(-1 * time.Minute).Unix()),
		})

		_, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
	})
}

// ---- Interceptor authenticate() fallback test ----

func TestAuthenticate_FallbackBetweenAuthenticators(t *testing.T) {
	key := generateTestKeyPair(t)
	kid := "test-key-1"
	srv := serveJWKS(t, &key.PublicKey, kid)
	jwksProvider := NewJWKSProvider(srv.URL)

	tenantID := uuid.New()
	resolver := &mockTenantResolver{
		tenants: map[string]uuid.UUID{"org_test": tenantID},
	}

	jwtAuthn := NewJWTAuthenticator(jwksProvider, WithTenantResolver(resolver))

	// Create a mock API key authenticator that succeeds for "sk_" prefixed tokens
	apiKeyAuthn := &mockAPIKeyAuthenticator{
		tenantID: tenantID,
	}

	authenticators := []Authenticator{jwtAuthn, apiKeyAuthn}

	t.Run("JWT token uses JWT authenticator", func(t *testing.T) {
		token := signJWT(t, key, kid, map[string]any{
			"sub":      "user_1",
			"org_id":   "org_test",
			"org_role": "org:admin",
			"exp":      float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		info, err := authenticate(context.Background(), authenticators, "Bearer", token)
		require.NoError(t, err)
		assert.Equal(t, tenantID, info.TenantID)
		assert.Equal(t, RoleTenantAdmin, info.TenantRole)
	})

	t.Run("API key falls through to API key authenticator", func(t *testing.T) {
		info, err := authenticate(context.Background(), authenticators, "Bearer", "sk_test_abc123def456")
		require.NoError(t, err)
		assert.Equal(t, tenantID, info.TenantID)
	})
}

// mockAPIKeyAuthenticator is a simple mock for testing authenticator fallback.
type mockAPIKeyAuthenticator struct {
	tenantID uuid.UUID
}

func (m *mockAPIKeyAuthenticator) Scheme() string { return "Bearer" }

func (m *mockAPIKeyAuthenticator) Authenticate(_ context.Context, credential string) (*AuthInfo, error) {
	if len(credential) > 3 && credential[:3] == "sk_" {
		return &AuthInfo{
			TenantID:   m.tenantID,
			TenantRole: RoleTenantAdmin,
		}, nil
	}
	return nil, fmt.Errorf("not an API key")
}

// ---- CachingTenantResolver Tests ----

func TestCachingTenantResolver(t *testing.T) {
	tenantID := uuid.New()
	lookup := &mockTenantLookup{
		tenants: map[string]uuid.UUID{"org_ext_123": tenantID},
	}

	resolver := NewCachingTenantResolver(lookup, WithTenantCacheTTL(100*time.Millisecond))

	t.Run("resolves external ID via lookup", func(t *testing.T) {
		id, err := resolver.ResolveTenant(context.Background(), "org_ext_123", "user_1")
		require.NoError(t, err)
		assert.Equal(t, tenantID, id)
		assert.Equal(t, 1, lookup.callCount)
	})

	t.Run("caches result", func(t *testing.T) {
		id, err := resolver.ResolveTenant(context.Background(), "org_ext_123", "user_1")
		require.NoError(t, err)
		assert.Equal(t, tenantID, id)
		assert.Equal(t, 1, lookup.callCount, "should use cache")
	})

	t.Run("cache expires", func(t *testing.T) {
		time.Sleep(110 * time.Millisecond)
		// After expiry, it should try UUID parse first (fails), then call lookup
		id, err := resolver.ResolveTenant(context.Background(), "org_ext_123", "user_1")
		require.NoError(t, err)
		assert.Equal(t, tenantID, id)
		assert.Equal(t, 2, lookup.callCount)
	})

	t.Run("UUID external ID requires database validation", func(t *testing.T) {
		// Previously, passing a valid UUID would bypass DB lookup entirely.
		// After the security fix, ALL external IDs must be validated via the database.
		directID := uuid.New()
		_, err := resolver.ResolveTenant(context.Background(), directID.String(), "user_1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown tenant")
		assert.Equal(t, 3, lookup.callCount, "should call lookup even for valid UUID")
	})

	t.Run("unknown tenant returns error", func(t *testing.T) {
		_, err := resolver.ResolveTenant(context.Background(), "org_unknown", "user_1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown tenant")
	})
}

type mockTenantLookup struct {
	tenants   map[string]uuid.UUID
	callCount int
}

func (m *mockTenantLookup) LookupTenantIDByExternalID(_ context.Context, externalID string) (uuid.UUID, error) {
	m.callCount++
	id, ok := m.tenants[externalID]
	if !ok {
		return uuid.Nil, fmt.Errorf("not found")
	}
	return id, nil
}

// ---- Mock MembershipResolver for namespace role tests ----

type mockMembershipResolver struct {
	roles  map[string]Role
	err    error
	called bool
}

func (m *mockMembershipResolver) ResolveNamespaceMemberships(_ context.Context, _ uuid.UUID, _ string) (map[string]Role, error) {
	m.called = true
	return m.roles, m.err
}

// ---- JWT namespace_roles claim extraction tests ----

func TestJWTAuthenticator_NamespaceRolesFromClaims(t *testing.T) {
	key := generateTestKeyPair(t)
	kid := "test-key-1"
	srv := serveJWKS(t, &key.PublicKey, kid)
	jwksProvider := NewJWKSProvider(srv.URL)

	tenantID := uuid.New()
	resolver := &mockTenantResolver{
		tenants: map[string]uuid.UUID{"org_test": tenantID},
	}

	t.Run("JWT with namespace_roles uses claim and skips DB", func(t *testing.T) {
		memberResolver := &mockMembershipResolver{
			roles: map[string]Role{"should-not-use": RoleNamespaceViewer},
		}

		authn := NewJWTAuthenticator(
			jwksProvider,
			WithTenantResolver(resolver),
			WithMembershipResolver(memberResolver),
		)

		token := signJWT(t, key, kid, map[string]any{
			"sub":      "user_1",
			"org_id":   "org_test",
			"org_role": "org:admin",
			"exp":      float64(time.Now().Add(1 * time.Hour).Unix()),
			"namespace_roles": []string{
				"namespace:admin:customer-a",
				"namespace:viewer:customer-b",
			},
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.False(t, memberResolver.called, "DB resolver should NOT be called when JWT has namespace_roles")
		require.Len(t, info.NamespaceRoles, 2)
		assert.Equal(t, RoleNamespaceAdmin, info.NamespaceRoles["customer-a"])
		assert.Equal(t, RoleNamespaceViewer, info.NamespaceRoles["customer-b"])
	})

	t.Run("JWT without namespace_roles falls back to DB", func(t *testing.T) {
		memberResolver := &mockMembershipResolver{
			roles: map[string]Role{"ns-from-db": RoleNamespaceMember},
		}

		authn := NewJWTAuthenticator(
			jwksProvider,
			WithTenantResolver(resolver),
			WithMembershipResolver(memberResolver),
		)

		token := signJWT(t, key, kid, map[string]any{
			"sub":      "user_2",
			"org_id":   "org_test",
			"org_role": "org:member",
			"exp":      float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.True(t, memberResolver.called, "DB resolver SHOULD be called when JWT lacks namespace_roles")
		require.Len(t, info.NamespaceRoles, 1)
		assert.Equal(t, RoleNamespaceMember, info.NamespaceRoles["ns-from-db"])
	})

	t.Run("JWT with empty namespace_roles array falls back to DB", func(t *testing.T) {
		memberResolver := &mockMembershipResolver{
			roles: map[string]Role{"ns-from-db": RoleNamespaceAdmin},
		}

		authn := NewJWTAuthenticator(
			jwksProvider,
			WithTenantResolver(resolver),
			WithMembershipResolver(memberResolver),
		)

		token := signJWT(t, key, kid, map[string]any{
			"sub":             "user_3",
			"org_id":          "org_test",
			"exp":             float64(time.Now().Add(1 * time.Hour).Unix()),
			"namespace_roles": []string{},
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.True(t, memberResolver.called, "DB resolver should be called when namespace_roles is empty")
		require.Len(t, info.NamespaceRoles, 1)
		assert.Equal(t, RoleNamespaceAdmin, info.NamespaceRoles["ns-from-db"])
	})

	t.Run("JWT with invalid entries in namespace_roles", func(t *testing.T) {
		memberResolver := &mockMembershipResolver{}

		authn := NewJWTAuthenticator(
			jwksProvider,
			WithTenantResolver(resolver),
			WithMembershipResolver(memberResolver),
		)

		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_4",
			"org_id": "org_test",
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
			"namespace_roles": []string{
				"bad-entry",
				"namespace:admin:valid-ns",
				"namespace:superadmin:invalid-role",
			},
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.False(t, memberResolver.called, "DB resolver should not be called when valid roles exist")
		require.Len(t, info.NamespaceRoles, 1)
		assert.Equal(t, RoleNamespaceAdmin, info.NamespaceRoles["valid-ns"])
	})

	t.Run("DB resolver failure is non-fatal", func(t *testing.T) {
		memberResolver := &mockMembershipResolver{
			err: fmt.Errorf("database connection lost"),
		}

		authn := NewJWTAuthenticator(
			jwksProvider,
			WithTenantResolver(resolver),
			WithMembershipResolver(memberResolver),
		)

		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_5",
			"org_id": "org_test",
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err, "auth should succeed even if membership resolver fails")
		assert.True(t, memberResolver.called)
		assert.Nil(t, info.NamespaceRoles, "NamespaceRoles should be nil on resolver error")
	})

	t.Run("no resolver and no claims", func(t *testing.T) {
		authn := NewJWTAuthenticator(
			jwksProvider,
			WithTenantResolver(resolver),
			// No MembershipResolver
		)

		token := signJWT(t, key, kid, map[string]any{
			"sub":    "user_6",
			"org_id": "org_test",
			"exp":    float64(time.Now().Add(1 * time.Hour).Unix()),
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.Nil(t, info.NamespaceRoles)
	})

	t.Run("NamespaceRolesClaim disabled skips claim and uses DB", func(t *testing.T) {
		memberResolver := &mockMembershipResolver{
			roles: map[string]Role{"ns-from-db": RoleNamespaceViewer},
		}

		authn := NewJWTAuthenticator(
			jwksProvider,
			WithTenantResolver(resolver),
			WithMembershipResolver(memberResolver),
			WithClaimsConfig(JWTClaimsConfig{
				TenantClaim:         "org_id",
				RoleClaim:           "org_role",
				SubjectClaim:        "sub",
				NamespaceRolesClaim: "", // disabled
				DefaultRole:         RoleTenantMember,
				ClockSkew:           30 * time.Second,
				RoleMapping:         map[string]Role{"org:admin": RoleTenantAdmin, "org:member": RoleTenantMember},
			}),
		)

		token := signJWT(t, key, kid, map[string]any{
			"sub":             "user_7",
			"org_id":          "org_test",
			"exp":             float64(time.Now().Add(1 * time.Hour).Unix()),
			"namespace_roles": []string{"namespace:admin:from-jwt"},
		})

		info, err := authn.Authenticate(context.Background(), token)
		require.NoError(t, err)
		assert.True(t, memberResolver.called, "DB resolver should be used when claim is disabled")
		require.Len(t, info.NamespaceRoles, 1)
		assert.Equal(t, RoleNamespaceViewer, info.NamespaceRoles["ns-from-db"])
	})
}

// ---- extractStringSlice Tests ----

func TestExtractStringSlice(t *testing.T) {
	t.Run("[]string input", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := extractStringSlice(input)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("[]interface{} with strings", func(t *testing.T) {
		input := []interface{}{"a", "b", "c"}
		result := extractStringSlice(input)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("[]interface{} with mixed types", func(t *testing.T) {
		input := []interface{}{"a", 42, "b", true, "c"}
		result := extractStringSlice(input)
		assert.Equal(t, []string{"a", "b", "c"}, result)
	})

	t.Run("nil input", func(t *testing.T) {
		result := extractStringSlice(nil)
		assert.Nil(t, result)
	})

	t.Run("wrong type returns nil", func(t *testing.T) {
		result := extractStringSlice("not a slice")
		assert.Nil(t, result)
	})

	t.Run("empty []interface{}", func(t *testing.T) {
		input := []interface{}{}
		result := extractStringSlice(input)
		require.NotNil(t, result)
		assert.Len(t, result, 0)
	})
}
