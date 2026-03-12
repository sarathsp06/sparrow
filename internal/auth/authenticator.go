package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Authenticator resolves a raw credential (e.g., API key, JWT) into an AuthInfo.
type Authenticator interface {
	// Authenticate extracts and validates credentials from the request,
	// returning the identity information on success.
	// Returns an error if the credential is missing, invalid, or expired.
	Authenticate(ctx context.Context, credential string) (*AuthInfo, error)

	// Scheme returns the authentication scheme this authenticator handles
	// (e.g., "Bearer", "ApiKey").
	Scheme() string
}

// ---- API Key types ----

// APIKeyRecord represents a stored API key as read from the database.
// This is the shape the authenticator expects from its key store.
type APIKeyRecord struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	KeyHash         string
	Role            Role
	NamespaceScope  *string
	IsPlatformAdmin bool
	ExpiresAt       *time.Time
	RevokedAt       *time.Time
}

// APIKeyStore is the interface the API key authenticator uses to look up keys.
// This is a narrow interface so the authenticator doesn't depend on the full
// repository — it can be implemented by the tenant repository or a dedicated
// key store.
type APIKeyStore interface {
	// GetAPIKeyByHash looks up an API key by its SHA-256 hash.
	// Returns sql.ErrNoRows if not found.
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKeyRecord, error)

	// UpdateAPIKeyLastUsed updates the last_used_at timestamp for a key.
	// This is best-effort and should not block authentication.
	UpdateAPIKeyLastUsed(ctx context.Context, keyID uuid.UUID, usedAt time.Time) error
}

// ---- API Key Authenticator ----

// APIKeyAuthenticator validates API keys by hashing and looking them up.
// It includes an optional in-memory cache to avoid hitting the database on
// every request for recently validated keys.
type APIKeyAuthenticator struct {
	store APIKeyStore

	// cache maps key_hash → cached auth result
	cache    map[string]*cacheEntry
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

type cacheEntry struct {
	info      *AuthInfo
	expiresAt time.Time
}

// APIKeyAuthenticatorOption configures the API key authenticator.
type APIKeyAuthenticatorOption func(*APIKeyAuthenticator)

// WithCacheTTL sets the TTL for cached API key lookups. Default is 5 minutes.
// Set to 0 to disable caching.
func WithCacheTTL(ttl time.Duration) APIKeyAuthenticatorOption {
	return func(a *APIKeyAuthenticator) {
		a.cacheTTL = ttl
	}
}

// NewAPIKeyAuthenticator creates an API key authenticator backed by the given store.
func NewAPIKeyAuthenticator(store APIKeyStore, opts ...APIKeyAuthenticatorOption) *APIKeyAuthenticator {
	a := &APIKeyAuthenticator{
		store:    store,
		cache:    make(map[string]*cacheEntry),
		cacheTTL: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

// Scheme returns "Bearer" — API keys are sent as Bearer tokens.
func (a *APIKeyAuthenticator) Scheme() string {
	return "Bearer"
}

// Authenticate validates an API key string.
//
// The key format is: sk_<tenant_slug>_<random>
// The full key is SHA-256 hashed and looked up in the database.
func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, credential string) (*AuthInfo, error) {
	// Basic format validation
	if !strings.HasPrefix(credential, "sk_") {
		return nil, fmt.Errorf("invalid API key format")
	}

	// Hash the key
	keyHash := HashAPIKey(credential)

	// Check cache first
	if a.cacheTTL > 0 {
		if info := a.fromCache(keyHash); info != nil {
			// Update last_used_at asynchronously (best-effort)
			if info.KeyID != nil {
				go func() {
					bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					_ = a.store.UpdateAPIKeyLastUsed(bgCtx, *info.KeyID, time.Now())
				}()
			}
			return info, nil
		}
	}

	// Look up by hash
	record, err := a.store.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, fmt.Errorf("authentication error: %w", err)
	}

	// Check if key is revoked
	if record.RevokedAt != nil {
		return nil, fmt.Errorf("API key has been revoked")
	}

	// Check if key is expired
	if record.ExpiresAt != nil && record.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("API key has expired")
	}

	// Build AuthInfo from the record
	info := &AuthInfo{
		TenantID:        record.TenantID,
		IsPlatformAdmin: record.IsPlatformAdmin,
		KeyID:           &record.ID,
		ExpiresAt:       record.ExpiresAt,
	}

	if IsTenantRole(record.Role) {
		info.TenantRole = record.Role
	} else if IsNamespaceRole(record.Role) && record.NamespaceScope != nil {
		info.NamespaceRoles = map[string]Role{
			*record.NamespaceScope: record.Role,
		}
	}

	// Cache the result
	if a.cacheTTL > 0 {
		a.toCache(keyHash, info)
	}

	// Update last_used_at asynchronously (best-effort)
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = a.store.UpdateAPIKeyLastUsed(bgCtx, record.ID, time.Now())
	}()

	return info, nil
}

// InvalidateKey removes a key from the cache by its hash.
// Call this when a key is revoked or modified.
func (a *APIKeyAuthenticator) InvalidateKey(keyHash string) {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	delete(a.cache, keyHash)
}

// InvalidateAll clears the entire cache.
func (a *APIKeyAuthenticator) InvalidateAll() {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	a.cache = make(map[string]*cacheEntry)
}

func (a *APIKeyAuthenticator) fromCache(keyHash string) *AuthInfo {
	a.cacheMu.RLock()
	defer a.cacheMu.RUnlock()
	entry, ok := a.cache[keyHash]
	if !ok {
		return nil
	}
	if time.Now().After(entry.expiresAt) {
		// Expired — will be cleaned up on next write
		return nil
	}
	return entry.info
}

func (a *APIKeyAuthenticator) toCache(keyHash string, info *AuthInfo) {
	a.cacheMu.Lock()
	defer a.cacheMu.Unlock()
	a.cache[keyHash] = &cacheEntry{
		info:      info,
		expiresAt: time.Now().Add(a.cacheTTL),
	}

	// Simple cache eviction: if cache is too large, drop everything.
	// A proper LRU would be better but this is sufficient for now.
	if len(a.cache) > 10000 {
		a.cache = make(map[string]*cacheEntry)
	}
}

// ---- Key generation helpers ----

// HashAPIKey returns the SHA-256 hex digest of a raw API key.
func HashAPIKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

// ConstantTimeCompareHash compares a raw key against a stored hash
// in constant time to prevent timing attacks.
func ConstantTimeCompareHash(rawKey, storedHash string) bool {
	computed := HashAPIKey(rawKey)
	return subtle.ConstantTimeCompare([]byte(computed), []byte(storedHash)) == 1
}

// GenerateAPIKeyPrefix creates the prefix portion of an API key.
// Format: sk_<tenantSlug>_
func GenerateAPIKeyPrefix(tenantSlug string) string {
	return fmt.Sprintf("sk_%s_", tenantSlug)
}

// Compile-time check
var _ Authenticator = (*APIKeyAuthenticator)(nil)
