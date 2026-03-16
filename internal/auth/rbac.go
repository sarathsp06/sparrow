// Package auth provides authentication, authorization, and RBAC for Sparrow.
//
// Roles and permissions are scoped to two levels:
//   - Tenant-level roles (tenant:admin, tenant:member) grant access across ALL namespaces within the tenant.
//   - Namespace-level roles (namespace:admin, namespace:member, namespace:viewer) grant access to a SPECIFIC namespace only.
//
// The platform_admin flag is orthogonal to roles — it grants cross-tenant access for SaaS operations.
package auth

import "fmt"

// Permission represents a specific action that can be authorized.
type Permission string

// Role represents a named set of permissions.
type Role string

// --- Roles ---

const (
	RoleTenantAdmin     Role = "tenant:admin"
	RoleTenantMember    Role = "tenant:member"
	RoleNamespaceAdmin  Role = "namespace:admin"
	RoleNamespaceMember Role = "namespace:member"
	RoleNamespaceViewer Role = "namespace:viewer"
)

// AllRoles is the list of valid roles.
var AllRoles = []Role{
	RoleTenantAdmin,
	RoleTenantMember,
	RoleNamespaceAdmin,
	RoleNamespaceMember,
	RoleNamespaceViewer,
}

// IsValidRole checks if a string is a valid role.
func IsValidRole(r string) bool {
	for _, role := range AllRoles {
		if Role(r) == role {
			return true
		}
	}
	return false
}

// IsTenantRole returns true if the role is tenant-scoped (applies to all namespaces).
func IsTenantRole(r Role) bool {
	return r == RoleTenantAdmin || r == RoleTenantMember
}

// IsNamespaceRole returns true if the role is namespace-scoped.
func IsNamespaceRole(r Role) bool {
	return r == RoleNamespaceAdmin || r == RoleNamespaceMember || r == RoleNamespaceViewer
}

// --- Permissions ---

const (
	// Tenant management
	PermTenantCreate        Permission = "tenant:create"
	PermTenantRead          Permission = "tenant:read"
	PermTenantUpdate        Permission = "tenant:update"
	PermTenantDelete        Permission = "tenant:delete"
	PermTenantManageMembers Permission = "tenant:manage_members"
	PermTenantManageAPIKeys Permission = "tenant:manage_api_keys"

	// Event type management (tenant-scoped: shared across namespaces)
	PermEventTypeCreate Permission = "event_type:create"
	PermEventTypeRead   Permission = "event_type:read"
	PermEventTypeUpdate Permission = "event_type:update"
	PermEventTypeDelete Permission = "event_type:delete"

	// Namespace management
	PermNamespaceCreate Permission = "namespace:create"
	PermNamespaceRead   Permission = "namespace:read"
	PermNamespaceUpdate Permission = "namespace:update"
	PermNamespaceDelete Permission = "namespace:delete"

	// Namespace membership management (assigning users to namespaces)
	PermNamespaceMembershipManage Permission = "namespace:manage_members"

	// Webhook management (namespace-scoped)
	PermWebhookCreate Permission = "webhook:create"
	PermWebhookRead   Permission = "webhook:read"
	PermWebhookUpdate Permission = "webhook:update"
	PermWebhookDelete Permission = "webhook:delete"
	PermWebhookPause  Permission = "webhook:pause"

	// Subscription management (namespace-scoped)
	PermSubscriptionCreate Permission = "subscription:create"
	PermSubscriptionRead   Permission = "subscription:read"
	PermSubscriptionUpdate Permission = "subscription:update"
	PermSubscriptionDelete Permission = "subscription:delete"

	// Event push and read (namespace-scoped)
	PermEventPush Permission = "event:push"
	PermEventRead Permission = "event:read"

	// Delivery management (namespace-scoped)
	PermDeliveryRead  Permission = "delivery:read"
	PermDeliveryRetry Permission = "delivery:retry"

	// Health (namespace-scoped)
	PermHealthRead Permission = "health:read"
)

// --- Role → Permission mapping (single source of truth) ---

// rolePermissions defines which permissions each role grants.
// Tenant-level roles apply to ALL namespaces within the tenant.
// Namespace-level roles apply to a SPECIFIC namespace only.
var rolePermissions = map[Role]map[Permission]bool{
	RoleTenantAdmin: {
		PermTenantRead:          true,
		PermTenantUpdate:        true,
		PermTenantDelete:        true,
		PermTenantManageMembers: true,
		PermTenantManageAPIKeys: true,

		PermEventTypeCreate: true,
		PermEventTypeRead:   true,
		PermEventTypeUpdate: true,
		PermEventTypeDelete: true,

		PermNamespaceCreate:           true,
		PermNamespaceRead:             true,
		PermNamespaceUpdate:           true,
		PermNamespaceDelete:           true,
		PermNamespaceMembershipManage: true,

		PermWebhookCreate: true,
		PermWebhookRead:   true,
		PermWebhookUpdate: true,
		PermWebhookDelete: true,
		PermWebhookPause:  true,

		PermSubscriptionCreate: true,
		PermSubscriptionRead:   true,
		PermSubscriptionUpdate: true,
		PermSubscriptionDelete: true,

		PermEventPush: true,
		PermEventRead: true,

		PermDeliveryRead:  true,
		PermDeliveryRetry: true,

		PermHealthRead: true,
	},

	RoleTenantMember: {
		PermTenantRead: true,

		PermEventTypeRead: true,

		PermNamespaceRead: true,

		PermWebhookCreate: true,
		PermWebhookRead:   true,
		PermWebhookUpdate: true,
		PermWebhookDelete: true,
		PermWebhookPause:  true,

		PermSubscriptionCreate: true,
		PermSubscriptionRead:   true,
		PermSubscriptionUpdate: true,
		PermSubscriptionDelete: true,

		PermEventPush: true,
		PermEventRead: true,

		PermDeliveryRead:  true,
		PermDeliveryRetry: true,

		PermHealthRead: true,
	},

	RoleNamespaceAdmin: {
		PermEventTypeRead: true,

		PermNamespaceRead:             true,
		PermNamespaceUpdate:           true,
		PermNamespaceMembershipManage: true,

		PermWebhookCreate: true,
		PermWebhookRead:   true,
		PermWebhookUpdate: true,
		PermWebhookDelete: true,
		PermWebhookPause:  true,

		PermSubscriptionCreate: true,
		PermSubscriptionRead:   true,
		PermSubscriptionUpdate: true,
		PermSubscriptionDelete: true,

		PermEventPush: true,
		PermEventRead: true,

		PermDeliveryRead:  true,
		PermDeliveryRetry: true,

		PermHealthRead: true,
	},

	RoleNamespaceMember: {
		PermEventTypeRead: true,

		PermNamespaceRead: true,

		PermWebhookCreate: true,
		PermWebhookRead:   true,
		PermWebhookUpdate: true,
		PermWebhookDelete: true,
		PermWebhookPause:  true,

		PermSubscriptionCreate: true,
		PermSubscriptionRead:   true,
		PermSubscriptionUpdate: true,
		PermSubscriptionDelete: true,

		PermEventPush: true,
		PermEventRead: true,

		PermDeliveryRead:  true,
		PermDeliveryRetry: true,

		PermHealthRead: true,
	},

	RoleNamespaceViewer: {
		PermEventTypeRead: true,

		PermNamespaceRead: true,

		PermWebhookRead: true,

		PermSubscriptionRead: true,

		PermEventRead: true,

		PermDeliveryRead: true,

		PermHealthRead: true,
	},
}

// RoleHasPermission checks if a role grants a specific permission.
func RoleHasPermission(r Role, p Permission) bool {
	perms, ok := rolePermissions[r]
	if !ok {
		return false
	}
	return perms[p]
}

// --- Authorization ---

// Authorize checks whether the given AuthInfo has the required permission,
// optionally scoped to a specific namespace.
//
// Authorization logic:
//  1. Platform admins are always authorized.
//  2. If a namespace is specified and the identity has namespace memberships:
//     only the membership role on that specific namespace is checked (tenant role
//     is NOT used as a fallback for namespace-scoped operations).
//  3. If a namespace is specified but the identity has NO namespace memberships:
//     the tenant role is used (backward-compatible: tenant-wide access).
//  4. For tenant-level operations (namespace == ""), the tenant role is checked.
//  5. Otherwise, deny.
func Authorize(info *AuthInfo, perm Permission, namespace string) error {
	if info == nil {
		return fmt.Errorf("unauthorized: no auth context")
	}

	// Platform admins bypass all checks
	if info.IsPlatformAdmin {
		return nil
	}

	// Namespace-scoped operation
	if namespace != "" {
		// If the identity has namespace memberships, ONLY check the membership
		// role for this namespace — tenant role does not grant namespace access
		// when memberships exist.
		if len(info.NamespaceRoles) > 0 {
			if role, ok := info.NamespaceRoles[namespace]; ok {
				if RoleHasPermission(role, perm) {
					return nil
				}
			}
			return &PermissionDeniedError{
				Permission: perm,
				Namespace:  namespace,
			}
		}

		// No namespace memberships — fall back to tenant role (tenant-wide access)
		if info.TenantRole != "" && RoleHasPermission(info.TenantRole, perm) {
			return nil
		}

		return &PermissionDeniedError{
			Permission: perm,
			Namespace:  namespace,
		}
	}

	// Tenant-level operation (namespace == "")
	if info.TenantRole != "" && RoleHasPermission(info.TenantRole, perm) {
		return nil
	}

	return &PermissionDeniedError{
		Permission: perm,
		Namespace:  namespace,
	}
}

// PermissionDeniedError is returned when authorization fails.
type PermissionDeniedError struct {
	Permission Permission
	Namespace  string
}

func (e *PermissionDeniedError) Error() string {
	if e.Namespace != "" {
		return fmt.Sprintf("permission denied: requires %s on namespace %q", e.Permission, e.Namespace)
	}
	return fmt.Sprintf("permission denied: requires %s", e.Permission)
}
