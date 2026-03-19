/**
 * Namespace store — manages the list of namespaces and the currently
 * active namespace selection.
 *
 * Uses Svelte 5 module-level runes so all components share the same
 * reactive state without prop drilling.
 *
 * Namespace selection semantics:
 * - null means "all namespaces" (tenant-wide view)
 * - a string means a specific namespace is selected
 *
 * The store auto-fetches namespaces on first access. Components that
 * need namespace data should call `loadNamespaces()` on mount.
 *
 * NOTE: Svelte 5 does not allow exporting reassigned $state or $derived
 * values from modules. All reactive values are exposed via getter functions.
 */

import { namespaceClient, namespaceMembershipClient } from "$lib/services.js";
import type { NamespaceResource, NamespaceMembership } from "../../../../proto/webhook_pb.js";

// ── Reactive state ──────────────────────────────────────────────────────

let namespaces = $state<NamespaceResource[]>([]);
let memberships = $state<NamespaceMembership[]>([]);
let activeNamespace = $state<string | null>(null);
let loading = $state(false);
let error = $state<string | null>(null);

// ── Derived state (internal) ────────────────────────────────────────────

/** Whether the user has access to multiple namespaces (show dropdown). */
const _isMultiNamespace = $derived(namespaces.length > 1);

/** Whether the user has no namespace memberships (tenant-wide access). */
const _isTenantWide = $derived(memberships.length === 0);

/** The active namespace resource object, or null for "all namespaces". */
const _activeNamespaceResource = $derived(
  activeNamespace
    ? namespaces.find((ns) => ns.name === activeNamespace) ?? null
    : null
);

// ── State getters (Svelte 5 cannot export reassigned $state from modules) ──

function getNamespaces(): NamespaceResource[] {
  return namespaces;
}

function getMemberships(): NamespaceMembership[] {
  return memberships;
}

function getActiveNamespace(): string | null {
  return activeNamespace;
}

function getLoading(): boolean {
  return loading;
}

function getError(): string | null {
  return error;
}

// ── Derived getters ─────────────────────────────────────────────────────

function getIsMultiNamespace(): boolean {
  return _isMultiNamespace;
}

function getIsTenantWide(): boolean {
  return _isTenantWide;
}

function getActiveNamespaceResource(): NamespaceResource | null {
  return _activeNamespaceResource;
}

// ── Actions ─────────────────────────────────────────────────────────────

/**
 * Fetch all namespaces for the current tenant.
 * Safe to call multiple times — deduplicates concurrent calls.
 */
async function loadNamespaces(): Promise<void> {
  if (loading) return;
  loading = true;
  error = null;

  try {
    const resp = await namespaceClient.listNamespaces({});
    namespaces = resp.namespaces ?? [];

    // If only one namespace and none is selected, auto-select it.
    if (namespaces.length === 1 && activeNamespace === null) {
      activeNamespace = namespaces[0].name;
    }
  } catch (e) {
    error = e instanceof Error ? e.message : "Failed to load namespaces";
    console.error("[namespace store] loadNamespaces error:", e);
  } finally {
    loading = false;
  }
}

/**
 * Fetch the current user's namespace memberships.
 * Pass subjectId="" to fetch for the caller (server resolves from auth context).
 */
async function loadMemberships(subjectId: string = ""): Promise<void> {
  try {
    const resp = await namespaceMembershipClient.getUserNamespaces({ subjectId });
    memberships = resp.memberships ?? [];
  } catch (e) {
    console.error("[namespace store] loadMemberships error:", e);
  }
}

/**
 * Load both namespaces and memberships. Call this on app init.
 */
async function initialize(): Promise<void> {
  await Promise.all([loadNamespaces(), loadMemberships()]);
}

/**
 * Set the active namespace. Pass null for "all namespaces" (tenant-wide).
 */
function selectNamespace(name: string | null): void {
  activeNamespace = name;
}

/**
 * Create a new namespace.
 */
async function createNamespace(
  name: string,
  description: string
): Promise<NamespaceResource | null> {
  try {
    const resp = await namespaceClient.createNamespace({ name, description });
    const ns = resp.namespace;
    if (ns) {
      namespaces = [...namespaces, ns];
    }
    return ns ?? null;
  } catch (e) {
    error = e instanceof Error ? e.message : "Failed to create namespace";
    console.error("[namespace store] createNamespace error:", e);
    return null;
  }
}

/**
 * Delete a namespace by ID.
 */
async function deleteNamespace(id: string): Promise<boolean> {
  try {
    await namespaceClient.deleteNamespace({ id });
    namespaces = namespaces.filter((ns) => ns.id !== id);
    // If the deleted namespace was active, reset to "all".
    const deleted = namespaces.find((ns) => ns.id === id);
    if (deleted && activeNamespace === deleted.name) {
      activeNamespace = null;
    }
    return true;
  } catch (e) {
    error = e instanceof Error ? e.message : "Failed to delete namespace";
    console.error("[namespace store] deleteNamespace error:", e);
    return false;
  }
}

// ── Exports ─────────────────────────────────────────────────────────────

export {
  // State (via getter functions — Svelte 5 cannot export reassigned $state)
  getNamespaces as namespaces,
  getMemberships as memberships,
  getActiveNamespace as activeNamespace,
  getLoading as loading,
  getError as error,

  // Derived (via getter functions — Svelte 5 cannot export $derived directly)
  getIsMultiNamespace as isMultiNamespace,
  getIsTenantWide as isTenantWide,
  getActiveNamespaceResource as activeNamespaceResource,

  // Actions
  initialize as initializeNamespaces,
  loadNamespaces,
  loadMemberships,
  selectNamespace,
  createNamespace,
  deleteNamespace,
};
