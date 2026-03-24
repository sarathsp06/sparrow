<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { namespaceClient } from '$lib/services';
  import type { NamespaceResource } from '../../../../proto/webhook_pb.js';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

  let namespacesList = $state<NamespaceResource[]>([]);
  let loading = $state(true);
  let error = $state('');

  // Create namespace form
  let showCreateForm = $state(false);
  let newName = $state('');
  let newDescription = $state('');
  let creating = $state(false);

  // Delete confirmation
  let confirmDelete = $state(false);
  let namespaceToDelete = $state<NamespaceResource | null>(null);

  async function fetchNamespaces() {
    loading = true;
    error = '';
    try {
      const res = await namespaceClient.listNamespaces({});
      namespacesList = res.namespaces ?? [];
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to load namespaces';
    } finally {
      loading = false;
    }
  }

  async function handleCreate() {
    if (!newName.trim()) return;
    creating = true;
    error = '';
    try {
      const res = await namespaceClient.createNamespace({
        name: newName.trim().toLowerCase(),
        description: newDescription.trim(),
      });
      if (res.namespace) {
        namespacesList = [...namespacesList, res.namespace];
        newName = '';
        newDescription = '';
        showCreateForm = false;
      }
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to create namespace';
    } finally {
      creating = false;
    }
  }

  function promptDelete(ns: NamespaceResource) {
    namespaceToDelete = ns;
    confirmDelete = true;
  }

  async function handleDelete() {
    if (!namespaceToDelete) return;
    try {
      await namespaceClient.deleteNamespace({ id: namespaceToDelete.id });
      namespacesList = namespacesList.filter(ns => ns.id !== namespaceToDelete!.id);
      confirmDelete = false;
      namespaceToDelete = null;
    } catch (e: unknown) {
      error = e instanceof Error ? e.message : 'Failed to delete namespace';
      confirmDelete = false;
    }
  }

  function formatDate(ts: { seconds?: bigint } | undefined): string {
    if (!ts?.seconds) return '—';
    return new Date(Number(ts.seconds) * 1000).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }

  onMount(fetchNamespaces);
</script>

<svelte:head>
  <title>Namespaces | Sparrow</title>
</svelte:head>

<div class="max-w-6xl mx-auto px-4 py-8 font-inter">
  <!-- Header -->
  <div class="flex items-center justify-between mb-8">
    <div>
      <h1 class="text-3xl font-bold text-gray-900">Namespaces</h1>
      <p class="text-gray-500 mt-1">Manage namespace scopes for your webhooks and events.</p>
    </div>
      <button
        onclick={() => (showCreateForm = !showCreateForm)}
        class="px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition text-sm font-medium"
      >
        {showCreateForm ? 'Cancel' : '+ Create Namespace'}
      </button>
  </div>

  <!-- Create form -->
  {#if showCreateForm}
    <div class="bg-white border border-gray-200 rounded-xl p-6 mb-6 shadow-sm">
      <h2 class="text-lg font-semibold text-gray-900 mb-4">Create Namespace</h2>
      <form onsubmit={(e) => { e.preventDefault(); handleCreate(); }} class="space-y-4">
        <div>
          <label for="ns-name" class="block text-sm font-medium text-gray-700 mb-1">Name</label>
          <input
            id="ns-name"
            type="text"
            bind:value={newName}
            placeholder="e.g., production, staging, dev"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900/20"
            required
          />
          <p class="text-xs text-gray-400 mt-1">Lowercase letters, numbers, hyphens, and underscores only.</p>
        </div>
        <div>
          <label for="ns-desc" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
          <input
            id="ns-desc"
            type="text"
            bind:value={newDescription}
            placeholder="Optional description"
            class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900/20"
          />
        </div>
        <div class="flex gap-3">
          <button
            type="submit"
            disabled={creating || !newName.trim()}
            class="px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition text-sm font-medium disabled:opacity-50"
          >
            {creating ? 'Creating...' : 'Create'}
          </button>
          <button
            type="button"
            onclick={() => (showCreateForm = false)}
            class="px-4 py-2 bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition text-sm"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  {/if}

  <!-- Error -->
  {#if error}
    <div class="bg-red-50 text-red-700 px-4 py-3 rounded-lg mb-6 text-sm">
      {error}
    </div>
  {/if}

  <!-- Loading -->
  {#if loading}
    <div class="text-center py-12 text-gray-400">Loading namespaces...</div>
  {:else if namespacesList.length === 0}
    <EmptyState
      title="No namespaces yet"
      description="Create your first namespace to start organizing your webhooks and events."
    >
      {#snippet action()}
        <button
          onclick={() => (showCreateForm = true)}
          class="px-4 py-2 bg-gray-900 text-white rounded-lg hover:bg-gray-800 transition text-sm font-medium"
        >
          Create Namespace
        </button>
      {/snippet}
    </EmptyState>
  {:else}
    <!-- Namespace list -->
    <div class="grid gap-4">
      {#each namespacesList as ns}
        <div class="bg-white border border-gray-200 rounded-xl p-5 shadow-sm hover:shadow-md transition">
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0">
              <button
                onclick={() => goto(`/namespaces/${ns.name}`)}
                class="text-lg font-semibold text-gray-900 hover:text-primary transition cursor-pointer"
              >
                {ns.name}
              </button>
              {#if ns.description}
                <p class="text-sm text-gray-500 mt-1">{ns.description}</p>
              {/if}
              <div class="flex gap-4 mt-2 text-xs text-gray-400">
                <span>Created {formatDate(ns.createdAt)}</span>
                <span>ID: <code class="font-fira">{ns.id.slice(0, 8)}...</code></span>
              </div>
            </div>
            <div class="flex gap-2 ml-4 flex-shrink-0">
              <button
                onclick={() => goto(`/namespaces/${ns.name}`)}
                class="px-3 py-1.5 text-sm bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 transition"
              >
                Manage
              </button>
              <button
                  onclick={() => promptDelete(ns)}
                  class="px-3 py-1.5 text-sm bg-red-50 text-red-600 rounded-lg hover:bg-red-100 transition"
                >
                  Delete
                </button>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<!-- Delete confirmation -->
<ConfirmDialog
  open={confirmDelete}
  title="Delete Namespace"
  message={`Are you sure you want to delete namespace "${namespaceToDelete?.name}"? This will cascade-delete all webhooks, subscriptions, and deliveries within it.`}
  confirmLabel="Delete"
  variant="danger"
  onconfirm={handleDelete}
  oncancel={() => { confirmDelete = false; namespaceToDelete = null; }}
/>
