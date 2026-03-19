<!--
  NamespaceSwitcher — smart namespace selector for the nav bar.

  Visibility rules:
  - Single-namespace users: show namespace name as static text (no dropdown)
  - Multi-namespace or tenant-wide users: show dropdown
  - Tenant-wide users (no memberships) see "All namespaces" option + all namespaces
  - Tenant-wide users see data from ALL namespaces by default

  This component initializes the namespace store on mount.
-->
<script lang="ts">
  import {
    namespaces,
    activeNamespace,
    loading,
    isMultiNamespace,
    isTenantWide,
    selectNamespace,
    initializeNamespaces,
  } from "$lib/stores/namespace.svelte.js";

  // Initialize on mount
  $effect(() => {
    initializeNamespaces();
  });

  function handleChange(event: Event) {
    const target = event.target as HTMLSelectElement;
    const value = target.value;
    selectNamespace(value === "" ? null : value);
  }
</script>

{#if loading()}
  <span class="text-sm text-gray-400 px-2">...</span>
{:else if namespaces().length === 0}
  <!-- No namespaces — show nothing (user needs to create one first) -->
{:else if namespaces().length === 1 && !isTenantWide()}
  <!-- Single namespace, non-tenant-wide: show static text -->
  <span class="text-sm font-medium text-gray-600 px-2 py-1 bg-gray-100 rounded">
    {namespaces()[0].name}
  </span>
{:else}
  <!-- Multi-namespace or tenant-wide: show dropdown -->
  <select
    value={activeNamespace() ?? ""}
    onchange={handleChange}
    class="text-sm font-medium text-gray-700 bg-gray-100 border border-gray-200 rounded px-2 py-1 focus:outline-none focus:ring-2 focus:ring-primary/50 cursor-pointer"
  >
    {#if isTenantWide()}
      <option value="">All namespaces</option>
    {/if}
    {#each namespaces() as ns}
      <option value={ns.name}>{ns.name}</option>
    {/each}
  </select>
{/if}
