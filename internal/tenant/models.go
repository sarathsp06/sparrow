package tenant

import (
	"time"

	"github.com/google/uuid"
)

// Tenant represents an organization/team using Sparrow.
type Tenant struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	Slug      string    `json:"slug" db:"slug"`
	Status    string    `json:"status" db:"status"`
	Settings  string    `json:"settings" db:"settings"` // JSONB stored as string
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TenantStatus constants.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusArchived  = "archived"
)
