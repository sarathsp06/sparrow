/**
 * Auth store — exposes the current user's role information as reactive
 * state so components can make RBAC-aware UI decisions.
 *
 * Role hierarchy (from most to least privileged):
 *   tenant:admin > tenant:member > namespace:admin > namespace:member > namespace:viewer
 *
 * This store does NOT own the auth session — that remains in auth.ts.
 * It only tracks the role metadata derived from the JWT claims or
 * API key, which the backend returns via namespace memberships.
 *
 * NOTE: Svelte 5 does not allow directly exporting $derived state from
 * modules. We expose getter functions that read the derived values instead.
 */

import { memberships, isTenantWide } from "./namespace.svelte.js";

// ── Derived role state ──────────────────────────────────────────────────

/**
 * Whether the current user has tenant:admin role.
 * In no-auth mode this is always true (DefaultAuthInfo is tenant:admin).
 */
const _isTenantAdmin = $derived.by(() => isTenantWide());

/**
 * Map of namespace name -> role for the current user's namespace memberships.
 */
const _namespaceRoles = $derived.by(() => {
  const roles = new Map<string, string>();
  for (const m of memberships()) {
    roles.set(m.namespace, m.role);
  }
  return roles;
});

// ── Exported getters ────────────────────────────────────────────────────

/** Returns whether the current user has tenant-wide (admin) access. */
function getIsTenantAdmin(): boolean {
  return _isTenantAdmin;
}

/** Returns a Map of namespace name -> role for the current user. */
function getNamespaceRoles(): Map<string, string> {
  return _namespaceRoles;
}

/**
 * Check if the user has at least the given role for a namespace.
 * Role hierarchy: namespace:admin > namespace:member > namespace:viewer
 * Tenant-wide users (no memberships) always have access.
 */
function hasNamespaceAccess(namespace: string, minRole: string = "namespace:viewer"): boolean {
  // Tenant-wide users have full access
  if (_isTenantAdmin) return true;

  const role = _namespaceRoles.get(namespace);
  if (!role) return false;

  const hierarchy = ["namespace:viewer", "namespace:member", "namespace:admin"];
  const userLevel = hierarchy.indexOf(role);
  const requiredLevel = hierarchy.indexOf(minRole);

  if (userLevel === -1 || requiredLevel === -1) return false;
  return userLevel >= requiredLevel;
}

/**
 * Check if the user can manage (admin) a specific namespace.
 */
function canManageNamespace(namespace: string): boolean {
  return hasNamespaceAccess(namespace, "namespace:admin");
}

/**
 * Check if the user can write to a specific namespace.
 */
function canWriteNamespace(namespace: string): boolean {
  return hasNamespaceAccess(namespace, "namespace:member");
}

// ── Exports ─────────────────────────────────────────────────────────────

export {
  getIsTenantAdmin,
  getNamespaceRoles,
  hasNamespaceAccess,
  canManageNamespace,
  canWriteNamespace,
};
