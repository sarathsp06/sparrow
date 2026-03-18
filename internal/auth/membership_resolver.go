package auth

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
)

// membershipCacheKey identifies a user's memberships within a tenant.
type membershipCacheKey struct {
	TenantID  uuid.UUID
	SubjectID string
}

type membershipCacheEntry struct {
	roles     map[string]Role
	expiresAt time.Time
}

// CachingMembershipResolver wraps a MembershipResolver with an in-memory
// TTL cache so that namespace memberships are not re-fetched from the database
// on every request. Follows the same pattern as CachingTenantResolver.
type CachingMembershipResolver struct {
	inner    MembershipResolver
	logger   *slog.Logger
	cache    map[membershipCacheKey]*membershipCacheEntry
	cacheMu  sync.RWMutex
	cacheTTL time.Duration
}

// CachingMembershipResolverOption configures the caching membership resolver.
type CachingMembershipResolverOption func(*CachingMembershipResolver)

// WithMembershipCacheTTL sets the cache TTL. Default is 30 seconds.
func WithMembershipCacheTTL(ttl time.Duration) CachingMembershipResolverOption {
	return func(r *CachingMembershipResolver) {
		r.cacheTTL = ttl
	}
}

// WithMembershipResolverLogger sets the logger.
func WithMembershipResolverLogger(logger *slog.Logger) CachingMembershipResolverOption {
	return func(r *CachingMembershipResolver) {
		r.logger = logger
	}
}

// NewCachingMembershipResolver creates a membership resolver with caching.
func NewCachingMembershipResolver(inner MembershipResolver, opts ...CachingMembershipResolverOption) *CachingMembershipResolver {
	r := &CachingMembershipResolver{
		inner:    inner,
		cache:    make(map[membershipCacheKey]*membershipCacheEntry),
		cacheTTL: 30 * time.Second,
		logger:   slog.Default(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ResolveNamespaceMemberships returns cached namespace roles for the user,
// falling back to the underlying resolver on cache miss or expiry.
func (r *CachingMembershipResolver) ResolveNamespaceMemberships(ctx context.Context, tenantID uuid.UUID, subjectID string) (map[string]Role, error) {
	key := membershipCacheKey{TenantID: tenantID, SubjectID: subjectID}

	// Try cache first
	r.cacheMu.RLock()
	entry, ok := r.cache[key]
	r.cacheMu.RUnlock()

	if ok && time.Now().Before(entry.expiresAt) {
		return entry.roles, nil
	}

	// Cache miss — resolve from database
	roles, err := r.inner.ResolveNamespaceMemberships(ctx, tenantID, subjectID)
	if err != nil {
		return nil, err
	}

	r.cacheResult(key, roles)
	return roles, nil
}

func (r *CachingMembershipResolver) cacheResult(key membershipCacheKey, roles map[string]Role) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()
	r.cache[key] = &membershipCacheEntry{
		roles:     roles,
		expiresAt: time.Now().Add(r.cacheTTL),
	}
	// Evict expired entries when cache grows large
	if len(r.cache) > 10000 {
		now := time.Now()
		for k, e := range r.cache {
			if now.After(e.expiresAt) {
				delete(r.cache, k)
			}
		}
		if len(r.cache) > 10000 {
			count := 0
			for k := range r.cache {
				if count >= len(r.cache)/2 {
					break
				}
				delete(r.cache, k)
				count++
			}
		}
	}
}

// Compile-time check
var _ MembershipResolver = (*CachingMembershipResolver)(nil)
