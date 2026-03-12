package tenant

import (
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/auth"
)

// Tenant represents an organization/team using Sparrow.
type Tenant struct {
	ID         uuid.UUID `json:"id" db:"id"`
	Name       string    `json:"name" db:"name"`
	Slug       string    `json:"slug" db:"slug"`
	Status     string    `json:"status" db:"status"`
	Settings   string    `json:"settings" db:"settings"`                 // JSONB stored as string
	ExternalID *string   `json:"external_id,omitempty" db:"external_id"` // OIDC provider org ID (e.g., Clerk org_id)
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// TenantStatus constants.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusArchived  = "archived"
)

// APIKey represents a stored API key.
type APIKey struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	TenantID        uuid.UUID  `json:"tenant_id" db:"tenant_id"`
	Name            string     `json:"name" db:"name"`
	KeyPrefix       string     `json:"key_prefix" db:"key_prefix"`
	KeyHash         string     `json:"-" db:"key_hash"` // never serialized
	Role            auth.Role  `json:"role" db:"role"`
	NamespaceScope  *string    `json:"namespace_scope,omitempty" db:"namespace_scope"`
	IsPlatformAdmin bool       `json:"is_platform_admin" db:"is_platform_admin"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	RevokedAt       *time.Time `json:"revoked_at,omitempty" db:"revoked_at"`
}

// IsRevoked returns true if the key has been revoked.
func (k *APIKey) IsRevoked() bool {
	return k.RevokedAt != nil
}

// IsExpired returns true if the key has expired.
func (k *APIKey) IsExpired() bool {
	return k.ExpiresAt != nil && k.ExpiresAt.Before(time.Now())
}

// IsActive returns true if the key is neither revoked nor expired.
func (k *APIKey) IsActive() bool {
	return !k.IsRevoked() && !k.IsExpired()
}

// CreateAPIKeyRequest contains the parameters for creating a new API key.
type CreateAPIKeyRequest struct {
	TenantID        uuid.UUID
	Name            string
	Role            auth.Role
	NamespaceScope  *string // required for namespace-scoped roles
	IsPlatformAdmin bool
	ExpiresAt       *time.Time
}

// CreateAPIKeyResult is returned after creating an API key.
// The RawKey is the plaintext key shown only once at creation.
type CreateAPIKeyResult struct {
	Key    *APIKey
	RawKey string
}
