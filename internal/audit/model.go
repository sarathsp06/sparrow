package audit

import (
	"time"

	"github.com/google/uuid"
)

// ActorType identifies how the caller authenticated.
type ActorType string

const (
	ActorAPIKey ActorType = "api_key"
	ActorUser   ActorType = "user"
	ActorSystem ActorType = "system"
)

// Action identifies what mutation was performed.
// Convention: "<resource_type>.<verb>"
type Action string

// Webhook actions
const (
	ActionWebhookRegister   Action = "webhook.register"
	ActionWebhookUnregister Action = "webhook.unregister"
	ActionWebhookUpdate     Action = "webhook.update"
	ActionWebhookPause      Action = "webhook.pause"
	ActionWebhookResume     Action = "webhook.resume"
)

// Event actions
const (
	ActionEventRegister Action = "event.register"
	ActionEventUpdate   Action = "event.update"
	ActionEventDelete   Action = "event.delete"
)

// Subscription actions
const (
	ActionSubscriptionCreate Action = "subscription.create"
	ActionSubscriptionUpdate Action = "subscription.update"
	ActionSubscriptionDelete Action = "subscription.delete"
)

// Delivery actions
const (
	ActionDeliveryRetry Action = "delivery.retry"
)

// Tenant actions
const (
	ActionTenantCreate Action = "tenant.create"
	ActionTenantUpdate Action = "tenant.update"
	ActionTenantDelete Action = "tenant.delete"
)

// API key actions
const (
	ActionAPIKeyCreate Action = "api_key.create"
	ActionAPIKeyRevoke Action = "api_key.revoke"
)

// Namespace actions
const (
	ActionNamespaceCreate Action = "namespace.create"
	ActionNamespaceUpdate Action = "namespace.update"
	ActionNamespaceDelete Action = "namespace.delete"
)

// Membership actions
const (
	ActionMembershipAssign Action = "membership.assign"
	ActionMembershipRemove Action = "membership.remove"
)

// ResourceType identifies the type of resource affected.
type ResourceType string

const (
	ResourceWebhook      ResourceType = "webhook"
	ResourceEvent        ResourceType = "event"
	ResourceSubscription ResourceType = "subscription"
	ResourceDelivery     ResourceType = "delivery"
	ResourceTenant       ResourceType = "tenant"
	ResourceAPIKey       ResourceType = "api_key"
	ResourceNamespace    ResourceType = "namespace"
	ResourceMembership   ResourceType = "membership"
)

// Entry represents a single audit log record.
type Entry struct {
	ID           uuid.UUID    `json:"id" db:"id"`
	TenantID     uuid.UUID    `json:"tenant_id" db:"tenant_id"`
	ActorID      string       `json:"actor_id" db:"actor_id"`
	ActorType    ActorType    `json:"actor_type" db:"actor_type"`
	Action       Action       `json:"action" db:"action"`
	ResourceType ResourceType `json:"resource_type" db:"resource_type"`
	ResourceID   string       `json:"resource_id" db:"resource_id"`
	Namespace    string       `json:"namespace" db:"namespace"`
	Metadata     string       `json:"metadata" db:"metadata"` // JSONB stored as string
	IPAddress    string       `json:"ip_address" db:"ip_address"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
}

// ListFilter defines filters for querying audit logs.
type ListFilter struct {
	TenantID     uuid.UUID
	Namespace    string
	ResourceType ResourceType
	ResourceID   string
	Action       Action
	ActorID      string
	Limit        int
	Offset       int
}
