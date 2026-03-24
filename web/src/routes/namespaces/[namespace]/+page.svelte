<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import { namespaceClient } from '$lib/services';
  import type { NamespaceResource } from '../../../../../proto/webhook_pb.js';

  const namespaceName = $derived(page.params.namespace ?? '');

  let ns = $state<NamespaceResource | null>(null);
  let loading = $state(true);
  let error = $state('');

  // Edit mode (namespace description)
  let editing = $state(false);
  let editDescription = $state('');
  let saving = $state(false);

  async function fetchNamespace() {
    loading = true;
    error = '';
    try {
      const resp = await namespaceClient.getNamespace({ name: namespaceName });
      ns = resp.namespace ?? null;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load namespace';
    } finally {
      loading = false;
    }
  }

  function startEdit() {
    editDescription = ns?.description ?? '';
    editing = true;
  }

  async function saveEdit() {
    if (!ns) return;
    saving = true;
    error = '';
    try {
      const resp = await namespaceClient.updateNamespace({
        id: ns.id,
        description: editDescription.trim(),
      });
      if (resp.namespace) {
        ns = resp.namespace;
      }
      editing = false;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to update namespace';
    } finally {
      saving = false;
    }
  }

  function formatDate(ts: { seconds?: bigint } | undefined): string {
    if (!ts?.seconds) return '—';
    return new Date(Number(ts.seconds) * 1000).toLocaleDateString('en-US', {
      year: 'numeric', month: 'short', day: 'numeric',
    });
  }

  onMount(fetchNamespace);
</script>

<svelte:head>
  <title>{namespaceName} | Namespaces | Sparrow</title>
</svelte:head>

<div class="max-w-6xl mx-auto px-4 py-8 font-inter">
  <!-- Breadcrumb -->
  <nav class="text-sm text-gray-400 mb-6">
    <button onclick={() => goto('/namespaces')} class="hover:text-gray-600 transition">Namespaces</button>
    <span class="mx-2">/</span>
    <span class="text-gray-700 font-medium">{namespaceName}</span>
  </nav>

  {#if error}
    <div class="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg mb-6 text-sm flex items-start gap-2">
      <span class="material-symbols-outlined text-red-500 text-lg mt-0.5">error</span>
      <div>
        <p>{error}</p>
        <button onclick={() => { error = ''; }} class="text-xs text-red-500 hover:text-red-700 underline mt-1">Dismiss</button>
      </div>
    </div>
  {/if}

  {#if loading}
    <div class="text-center py-12 text-gray-400">Loading namespace...</div>
  {:else if !ns}
    <div class="text-center py-12 text-gray-500">Namespace not found.</div>
  {:else}
    <!-- Namespace details -->
    <div class="bg-white border border-gray-200 rounded-xl p-6 shadow-sm">
      <div class="flex items-start justify-between mb-4">
        <div>
          <h1 class="text-2xl font-bold text-gray-900">{ns.name}</h1>
          <p class="text-xs text-gray-400 mt-1 font-fira">ID: {ns.id}</p>
        </div>
        {#if !editing}
          <button
            onclick={startEdit}
            class="px-3 py-1.5 text-sm bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition"
          >
            Edit
          </button>
        {/if}
      </div>

      {#if editing}
        <form onsubmit={(e) => { e.preventDefault(); saveEdit(); }} class="space-y-3">
          <div>
            <label for="edit-desc" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <input
              id="edit-desc"
              type="text"
              bind:value={editDescription}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900/20"
            />
          </div>
          <div class="flex gap-2">
            <button
              type="submit"
              disabled={saving}
              class="px-3 py-1.5 text-sm bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition disabled:opacity-50"
            >
              {saving ? 'Saving...' : 'Save'}
            </button>
            <button
              type="button"
              onclick={() => (editing = false)}
              class="px-3 py-1.5 text-sm bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition"
            >
              Cancel
            </button>
          </div>
        </form>
      {:else}
        <p class="text-gray-600 text-sm">{ns.description || 'No description.'}</p>
      {/if}

      <div class="flex gap-6 mt-4 text-xs text-gray-400">
        <span>Created {formatDate(ns.createdAt)}</span>
        <span>Updated {formatDate(ns.updatedAt)}</span>
      </div>
    </div>

    <!-- Info: namespace scope -->
    <div class="mt-6 bg-blue-50 border border-blue-100 rounded-xl p-4 text-sm text-blue-700 flex items-start gap-2">
      <span class="material-symbols-outlined text-blue-500 text-lg mt-0.5">info</span>
      <p>Namespaces isolate webhooks and events into separate scopes. Webhooks registered in this namespace will only receive events pushed to it.</p>
    </div>
  {/if}
</div>
