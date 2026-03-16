package namespace

import (
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/auth"
)

// Namespace represents a first-class namespace entity within a tenant.
// Namespaces are sub-tenant scopes used to isolate webhooks, subscriptions,
// deliveries, and other resources.
type Namespace struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TenantID    uuid.UUID `json:"tenant_id" db:"tenant_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Membership represents a user's role assignment on a specific namespace.
// When a user has memberships, they can only access those namespaces.
type Membership struct {
	ID        uuid.UUID `json:"id" db:"id"`
	TenantID  uuid.UUID `json:"tenant_id" db:"tenant_id"`
	SubjectID string    `json:"subject_id" db:"subject_id"` // JWT sub claim
	Namespace string    `json:"namespace" db:"namespace"`   // Namespace name
	Role      auth.Role `json:"role" db:"role"`             // namespace:admin, namespace:member, namespace:viewer
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// CreateNamespaceRequest contains the parameters for creating a namespace.
type CreateNamespaceRequest struct {
	TenantID    uuid.UUID
	Name        string
	Description string
}

// UpdateNamespaceRequest contains the parameters for updating a namespace.
type UpdateNamespaceRequest struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	Name        string // empty = no change
	Description string // empty = no change
}

// AssignMembershipRequest contains the parameters for assigning a user to a namespace.
type AssignMembershipRequest struct {
	TenantID  uuid.UUID
	SubjectID string
	Namespace string
	Role      auth.Role
}
