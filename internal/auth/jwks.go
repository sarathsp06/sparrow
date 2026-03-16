package auth

import (
	"context"
	"net/http"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// JWKSProvider fetches and caches JSON Web Key Sets from a remote JWKS URL.
// It is provider-agnostic — works with Clerk, Auth0, Keycloak, or any
// OIDC-compliant identity provider that publishes a JWKS endpoint.
//
// This is a thin wrapper around github.com/MicahParks/keyfunc which handles
// key rotation, caching, and background refresh automatically.
type JWKSProvider struct {
	kf keyfunc.Keyfunc
}

// JWKSOption configures the JWKS provider.
type JWKSOption func(*jwksConfig)

type jwksConfig struct {
	httpClient *http.Client
	cacheTTL   time.Duration
}

// WithJWKSCacheTTL sets how long fetched keys are cached before being
// re-fetched. Default is 1 hour.
func WithJWKSCacheTTL(ttl time.Duration) JWKSOption {
	return func(c *jwksConfig) {
		c.cacheTTL = ttl
	}
}

// WithJWKSHTTPClient sets a custom HTTP client for fetching the JWKS.
func WithJWKSHTTPClient(client *http.Client) JWKSOption {
	return func(c *jwksConfig) {
		c.httpClient = client
	}
}

// NewJWKSProvider creates a JWKS provider that fetches keys from the given URL.
//
// Example URLs:
//   - Clerk:    https://<your-domain>.clerk.accounts.dev/.well-known/jwks.json
//   - Auth0:    https://<tenant>.auth0.com/.well-known/jwks.json
//   - Keycloak: https://<host>/realms/<realm>/protocol/openid-connect/certs
func NewJWKSProvider(jwksURL string, opts ...JWKSOption) *JWKSProvider {
	cfg := &jwksConfig{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		cacheTTL:   1 * time.Hour,
	}
	for _, opt := range opts {
		opt(cfg)
	}

	override := keyfunc.Override{
		Client:          cfg.httpClient,
		RefreshInterval: cfg.cacheTTL,
	}

	k, err := keyfunc.NewDefaultOverrideCtx(context.Background(), []string{jwksURL}, override)
	if err != nil {
		// Return a provider whose Keyfunc will always error,
		// matching the previous lazy-fetch behavior.
		return &JWKSProvider{
			kf: nil,
		}
	}

	return &JWKSProvider{
		kf: k,
	}
}

// Keyfunc returns the jwt.Keyfunc for use with jwt.Parse.
func (p *JWKSProvider) Keyfunc() jwt.Keyfunc {
	if p.kf == nil {
		return func(*jwt.Token) (any, error) {
			return nil, ErrJWKSNotAvailable
		}
	}
	return p.kf.Keyfunc
}
