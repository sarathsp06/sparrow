package auth

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TenantLookup is the narrow interface the tenant resolver needs from the
// tenant store. It can be implemented by the tenant repository or service.
type TenantLookup interface {
	// LookupTenantIDByExternalID maps an external identity provider ID
	// (e.g., Clerk org_id) to an internal tenant UUID.
	// Returns the tenant ID and nil error if found.
	// Returns an error if the tenant is unknown.
	LookupTenantIDByExternalID(ctx context.Context, externalID string) (uuid.UUID, error)
}

// CachingTenantResolver resolves external tenant IDs to internal UUIDs
// with an in-memory cache to avoid hitting the database on every request.
type CachingTenantResolver struct {
	lookup   TenantLookup
	cache    map[string]*tenantCacheEntry
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

type tenantCacheEntry struct {
	tenantID  uuid.UUID
	expiresAt time.Time
}

// CachingTenantResolverOption configures the caching tenant resolver.
type CachingTenantResolverOption func(*CachingTenantResolver)

// WithTenantCacheTTL sets the cache TTL. Default is 5 minutes.
func WithTenantCacheTTL(ttl time.Duration) CachingTenantResolverOption {
	return func(r *CachingTenantResolver) {
		r.cacheTTL = ttl
	}
}

// NewCachingTenantResolver creates a tenant resolver with caching.
func NewCachingTenantResolver(lookup TenantLookup, opts ...CachingTenantResolverOption) *CachingTenantResolver {
	r := &CachingTenantResolver{
		lookup:   lookup,
		cache:    make(map[string]*tenantCacheEntry),
		cacheTTL: 5 * time.Minute,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ResolveTenant maps an external tenant ID to an internal UUID.
func (r *CachingTenantResolver) ResolveTenant(ctx context.Context, externalID string) (uuid.UUID, error) {
	// Try cache first
	r.cacheMu.RLock()
	entry, ok := r.cache[externalID]
	r.cacheMu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.tenantID, nil
	}

	// Try parsing as UUID first (direct tenant ID)
	if id, err := uuid.Parse(externalID); err == nil {
		r.cacheResult(externalID, id)
		return id, nil
	}

	// Look up via the store
	id, err := r.lookup.LookupTenantIDByExternalID(ctx, externalID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("unknown tenant %q: %w", externalID, err)
	}

	r.cacheResult(externalID, id)
	return id, nil
}

func (r *CachingTenantResolver) cacheResult(externalID string, tenantID uuid.UUID) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.cache[externalID] = &tenantCacheEntry{
		tenantID:  tenantID,
		expiresAt: time.Now().Add(r.cacheTTL),
	}
	// Simple eviction
	if len(r.cache) > 10000 {
		r.cache = make(map[string]*tenantCacheEntry)
	}
}

// InvalidateAll clears the resolver cache.
func (r *CachingTenantResolver) InvalidateAll() {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.cache = make(map[string]*tenantCacheEntry)
}

// Compile-time check
var _ TenantResolver = (*CachingTenantResolver)(nil)
