package namespace

import (
	"time"

	"github.com/google/uuid"

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

