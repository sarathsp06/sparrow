package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// JWKSProvider fetches and caches JSON Web Key Sets from a remote JWKS URL.
// It is provider-agnostic — works with Clerk, Auth0, Keycloak, or any
// OIDC-compliant identity provider that publishes a JWKS endpoint.
type JWKSProvider struct {
	jwksURL    string
	httpClient *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey // kid → public key
	fetchedAt time.Time
	cacheTTL  time.Duration
}

// JWKSOption configures the JWKS provider.
type JWKSOption func(*JWKSProvider)

// WithJWKSCacheTTL sets how long fetched keys are cached before being
// re-fetched. Default is 1 hour.
func WithJWKSCacheTTL(ttl time.Duration) JWKSOption {
	return func(p *JWKSProvider) {
		p.cacheTTL = ttl
	}
}

// WithJWKSHTTPClient sets a custom HTTP client for fetching the JWKS.
func WithJWKSHTTPClient(client *http.Client) JWKSOption {
	return func(p *JWKSProvider) {
		p.httpClient = client
	}
}

// NewJWKSProvider creates a JWKS provider that fetches keys from the given URL.
//
// Example URLs:
//   - Clerk:    https://<your-domain>.clerk.accounts.dev/.well-known/jwks.json
//   - Auth0:    https://<tenant>.auth0.com/.well-known/jwks.json
//   - Keycloak: https://<host>/realms/<realm>/protocol/openid-connect/certs
func NewJWKSProvider(jwksURL string, opts ...JWKSOption) *JWKSProvider {
	p := &JWKSProvider{
		jwksURL:    jwksURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		keys:       make(map[string]*rsa.PublicKey),
		cacheTTL:   1 * time.Hour,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

// GetKey returns the RSA public key for the given key ID (kid).
// It fetches/refreshes the JWKS if the cache is stale or the kid is unknown.
func (p *JWKSProvider) GetKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	// Try cache first
	p.mu.RLock()
	key, ok := p.keys[kid]
	stale := time.Since(p.fetchedAt) > p.cacheTTL
	p.mu.RUnlock()

	if ok && !stale {
		return key, nil
	}

	// Fetch fresh keys (either cache is stale or kid is unknown)
	if err := p.refresh(ctx); err != nil {
		// If we had a cached key and refresh failed, return the cached one
		if ok {
			return key, nil
		}
		return nil, fmt.Errorf("jwks: fetch failed: %w", err)
	}

	// Try again after refresh
	p.mu.RLock()
	key, ok = p.keys[kid]
	p.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("jwks: unknown key ID %q", kid)
	}
	return key, nil
}

// refresh fetches the JWKS from the remote URL and updates the cache.
func (p *JWKSProvider) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch JWKS: HTTP %d", resp.StatusCode)
	}

	var jwks jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}

	keys := make(map[string]*rsa.PublicKey, len(jwks.Keys))
	for _, jwk := range jwks.Keys {
		if jwk.Kty != "RSA" || jwk.Use != "sig" {
			continue
		}
		pubKey, err := jwk.toRSAPublicKey()
		if err != nil {
			continue // skip malformed keys
		}
		keys[jwk.Kid] = pubKey
	}

	if len(keys) == 0 {
		return fmt.Errorf("JWKS contains no usable RSA signing keys")
	}

	p.mu.Lock()
	p.keys = keys
	p.fetchedAt = time.Now()
	p.mu.Unlock()

	return nil
}

// ---- JWKS JSON structures ----

type jwksResponse struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"` // Key type: "RSA"
	Use string `json:"use"` // Usage: "sig"
	Kid string `json:"kid"` // Key ID
	Alg string `json:"alg"` // Algorithm: "RS256"
	N   string `json:"n"`   // RSA modulus (base64url)
	E   string `json:"e"`   // RSA exponent (base64url)
}

func (k *jwkKey) toRSAPublicKey() (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(k.N)
	if err != nil {
		return nil, fmt.Errorf("decode modulus: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(k.E)
	if err != nil {
		return nil, fmt.Errorf("decode exponent: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: n,
		E: int(e.Int64()),
	}, nil
}
