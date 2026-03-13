package auth

import (
	"context"
	"fmt"
	"log/slog"
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

// TenantProvisioner creates a new tenant for an external ID that has no
// existing mapping. This enables auto-provisioning: when a user logs in
// via an identity provider with an org/tenant that Sparrow doesn't know
// about yet, the resolver can create it on the fly.
type TenantProvisioner interface {
	// ProvisionTenant creates a new tenant for the given external ID.
	// createdBy is the identity provider user ID (JWT "sub") of the user
	// who triggered the provisioning. Implementations should enforce
	// per-user tenant limits using this value.
	// Returns the new internal tenant UUID.
	ProvisionTenant(ctx context.Context, externalID string, createdBy string) (uuid.UUID, error)
}

// CachingTenantResolver resolves external tenant IDs to internal UUIDs
// with an in-memory cache to avoid hitting the database on every request.
type CachingTenantResolver struct {
	lookup      TenantLookup
	provisioner TenantProvisioner
	logger      *slog.Logger
	cache       map[string]*tenantCacheEntry
	cacheMu     sync.RWMutex
	cacheTTL    time.Duration
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

// WithTenantProvisioner enables auto-provisioning of tenants.
// When a lookup fails for an unknown external ID, the provisioner creates
// a new tenant automatically. This is the recommended approach for
// identity-provider-managed organizations (e.g., Clerk).
func WithTenantProvisioner(p TenantProvisioner) CachingTenantResolverOption {
	return func(r *CachingTenantResolver) {
		r.provisioner = p
	}
}

// WithTenantResolverLogger sets the logger for the tenant resolver.
func WithTenantResolverLogger(logger *slog.Logger) CachingTenantResolverOption {
	return func(r *CachingTenantResolver) {
		r.logger = logger
	}
}

// NewCachingTenantResolver creates a tenant resolver with caching.
func NewCachingTenantResolver(lookup TenantLookup, opts ...CachingTenantResolverOption) *CachingTenantResolver {
	r := &CachingTenantResolver{
		lookup:   lookup,
		cache:    make(map[string]*tenantCacheEntry),
		cacheTTL: 5 * time.Minute,
		logger:   slog.Default(),
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

// ResolveTenant maps an external tenant ID to an internal UUID.
// If the external ID is not found and a TenantProvisioner is configured,
// a new tenant is automatically created.
func (r *CachingTenantResolver) ResolveTenant(ctx context.Context, externalID string, subjectID string) (uuid.UUID, error) {
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
	if err == nil {
		r.cacheResult(externalID, id)
		return id, nil
	}

	// If lookup failed and we have a provisioner, auto-create the tenant
	if r.provisioner != nil {
		r.logger.Info("auto-provisioning tenant for external ID",
			"external_id", externalID,
			"subject_id", subjectID,
		)
		newID, provErr := r.provisioner.ProvisionTenant(ctx, externalID, subjectID)
		if provErr != nil {
			return uuid.Nil, fmt.Errorf("auto-provision tenant for %q failed: %w", externalID, provErr)
		}
		r.cacheResult(externalID, newID)
		r.logger.Info("tenant auto-provisioned successfully",
			"external_id", externalID,
			"tenant_id", newID.String(),
		)
		return newID, nil
	}

	return uuid.Nil, fmt.Errorf("unknown tenant %q: %w", externalID, err)
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
