<script lang="ts">
  import { goto } from '$app/navigation';
  import { webhookClient as client, healthClient } from '$lib/services';
  import { onMount } from 'svelte';
  import type { RegisteredWebhook } from '../../../../proto/webhook_pb.js';
  import { WebhookHealth } from '../../../../proto/webhook_pb.js';
  import HealthBadge from '$lib/components/HealthBadge.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import FloatingAction from '$lib/components/FloatingAction.svelte';
  import Pagination from '$lib/components/Pagination.svelte';

  let webhooks: RegisteredWebhook[] = $state([]);
  let loading = $state(true);
  let error = $state('');

  // Filtering
  let healthFilter = $state<WebhookHealth | null>(null);
  let urlSearch = $state('');

  // Pagination
  let limit = $state(25);
  let offset = $state(0);
  let totalCount = $state(0);

  // Unregister confirmation
  let confirmUnregister = $state(false);
  let webhookToUnregister = $state<RegisteredWebhook | null>(null);

  // Namespace stats
  let stats = $state<{ total: number; active: number; healthy: number; unhealthy: number }>({
    total: 0, active: 0, healthy: 0, unhealthy: 0,
  });

  const healthFilters: { value: WebhookHealth | null; label: string }[] = [
    { value: null, label: 'All' },
    { value: WebhookHealth.HEALTH_HEALTHY, label: 'Healthy' },
    { value: WebhookHealth.HEALTH_DEGRADED, label: 'Degraded' },
    { value: WebhookHealth.HEALTH_UNHEALTHY, label: 'Unhealthy' },
  ];

  // Client-side search filter (URL search is always client-side)
  let filteredWebhooks = $derived.by(() => {
    let result = webhooks;
    if (urlSearch.trim()) {
      const q = urlSearch.toLowerCase();
      result = result.filter(w =>
        w.url.toLowerCase().includes(q) ||
        w.description.toLowerCase().includes(q) ||
        w.webhookId.toLowerCase().includes(q) ||
        w.namespace.toLowerCase().includes(q)
      );
    }
    return result;
  });

  let currentPage = $derived(Math.floor(offset / limit) + 1);
  let totalPages = $derived(Math.max(1, Math.ceil(totalCount / limit)));

  async function fetchWebhooks() {
    loading = true;
    error = '';
    try {
      // Use server-side health filtering when a health filter is active
      if (healthFilter !== null) {
        const res = await healthClient.listWebhooksByHealth({
          health: healthFilter,
          pagination: { limit, offset },
        });
        webhooks = res.webhooks || [];
        totalCount = res.pagination?.totalCount || 0;
      } else {
        const res = await client.listWebhooks({
          namespace: '',
          pagination: { limit, offset },
        });
        webhooks = res.webhooks || [];
        totalCount = res.pagination?.totalCount || 0;
      }

      // Compute quick stats from current page
      stats = {
        total: totalCount,
        active: webhooks.filter(w => w.active).length,
        healthy: webhooks.filter(w => w.health === WebhookHealth.HEALTH_HEALTHY).length,
        unhealthy: webhooks.filter(w => w.health === WebhookHealth.HEALTH_UNHEALTHY).length,
      };
    } catch (e: any) {
      console.error(e);
      error = e.message || 'Failed to load webhooks';
    } finally {
      loading = false;
    }
  }

  onMount(fetchWebhooks);

  function handlePageChange(pageNum: number) {
    offset = (pageNum - 1) * limit;
    fetchWebhooks();
  }

  function handleHealthFilterChange(health: WebhookHealth | null) {
    healthFilter = health;
    offset = 0; // Reset to first page when filter changes
    fetchWebhooks();
  }

  function promptUnregister(wh: RegisteredWebhook, e: Event) {
    e.stopPropagation();
    webhookToUnregister = wh;
    confirmUnregister = true;
  }

  async function executeUnregister() {
    if (!webhookToUnregister) return;
    try {
      await client.unregisterWebhook({
        webhookId: webhookToUnregister.webhookId,
        namespace: webhookToUnregister.namespace,
      });
      confirmUnregister = false;
      webhookToUnregister = null;
      await fetchWebhooks();
    } catch (e: any) {
      error = `Failed to unregister webhook: ${e.message}`;
      confirmUnregister = false;
    }
  }

  async function toggleActive(wh: RegisteredWebhook, e: Event) {
    e.stopPropagation();
    try {
      if (wh.active) {
        await client.pauseWebhook({ webhookId: wh.webhookId, namespace: wh.namespace });
      } else {
        await client.resumeWebhook({ webhookId: wh.webhookId, namespace: wh.namespace });
      }
      await fetchWebhooks();
    } catch (e: any) {
      error = `Failed to update webhook: ${(e as Error).message}`;
    }
  }
</script>

<svelte:head>
  <title>Webhooks | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    <!-- Page header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Webhooks</h1>
        <p class="text-sm text-gray-500 mt-0.5">Manage registered webhooks</p>
      </div>
      <a
        id="header-register-btn"
        href="/webhooks/register"
        class="inline-flex items-center gap-2 bg-gray-900 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-800 transition shadow-sm"
      >
        <span class="text-lg leading-none">+</span>
        Register Webhook
      </a>
    </div>

    <!-- Stats bar -->
    {#if !loading && !error}
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
        <div class="bg-white rounded-lg border border-gray-200 px-4 py-3">
          <p class="text-xs font-medium text-gray-500 uppercase tracking-wide">Total</p>
          <p class="text-2xl font-bold text-gray-900">{stats.total}</p>
        </div>
        <div class="bg-white rounded-lg border border-gray-200 px-4 py-3">
          <p class="text-xs font-medium text-gray-500 uppercase tracking-wide">Active</p>
          <p class="text-2xl font-bold text-green-600">{stats.active}</p>
        </div>
        <div class="bg-white rounded-lg border border-gray-200 px-4 py-3">
          <p class="text-xs font-medium text-gray-500 uppercase tracking-wide">Healthy</p>
          <p class="text-2xl font-bold text-green-600">{stats.healthy}</p>
        </div>
        <div class="bg-white rounded-lg border border-gray-200 px-4 py-3">
          <p class="text-xs font-medium text-gray-500 uppercase tracking-wide">Unhealthy</p>
          <p class="text-2xl font-bold text-red-600">{stats.unhealthy}</p>
        </div>
      </div>
    {/if}

    <!-- Toolbar: Search + Health filter pills -->
    <div class="flex flex-col sm:flex-row gap-3 mb-4">
      <div class="relative flex-1 max-w-md">
        <input
          type="text"
          placeholder="Search by URL, description, or ID..."
          bind:value={urlSearch}
          class="w-full pl-9 pr-4 py-2 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
        />
        <svg class="absolute left-3 top-2.5 w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
        </svg>
      </div>
      <div class="flex items-center gap-1 bg-gray-100 rounded-lg p-1">
        {#each healthFilters as filter}
          <button
            class="px-3 py-1.5 text-xs font-medium rounded-md transition {healthFilter === filter.value ? 'bg-white text-gray-900 shadow-sm' : 'text-gray-500 hover:text-gray-700'}"
            onclick={() => handleHealthFilterChange(filter.value)}
          >
            {filter.label}
          </button>
        {/each}
      </div>
    </div>

    <!-- Content -->
    {#if loading}
      <!-- Loading skeleton -->
      <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead>
              <tr class="border-b border-gray-200 bg-gray-50/50">
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Webhook</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden md:table-cell">Namespace</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Events</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden lg:table-cell">Config</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Health</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100">
              {#each Array(5) as _}
                <tr class="animate-pulse">
                  <td class="px-4 py-3"><div class="space-y-2"><div class="h-4 bg-gray-200 rounded w-48"></div><div class="h-3 bg-gray-100 rounded w-32"></div></div></td>
                  <td class="px-4 py-3 hidden md:table-cell"><div class="h-4 bg-gray-200 rounded w-20"></div></td>
                  <td class="px-4 py-3 hidden sm:table-cell"><div class="flex gap-1"><div class="h-5 bg-gray-200 rounded w-16"></div><div class="h-5 bg-gray-200 rounded w-16"></div></div></td>
                  <td class="px-4 py-3 hidden lg:table-cell"><div class="h-4 bg-gray-100 rounded w-20"></div></td>
                  <td class="px-4 py-3"><div class="h-5 bg-gray-200 rounded-full w-20"></div></td>
                  <td class="px-4 py-3"><div class="h-5 bg-gray-200 rounded-full w-9"></div></td>
                  <td class="px-4 py-3 text-right"><div class="h-6 bg-gray-200 rounded w-20 ml-auto"></div></td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {:else if error}
      <div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
        <div class="flex items-start gap-3">
          <svg class="w-5 h-5 text-red-500 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
          </svg>
          <div>
            <p class="text-sm font-medium text-red-800">{error}</p>
            <button onclick={fetchWebhooks} class="text-sm text-red-600 hover:text-red-800 underline mt-1">Retry</button>
          </div>
        </div>
      </div>
    {:else if webhooks.length === 0}
      <div class="bg-white border border-gray-200 rounded-lg">
        <EmptyState
          icon="webhook"
          title="No webhooks registered"
          description="Get started by registering your first webhook to begin receiving event notifications."
        >
          {#snippet action()}
            <a href="/webhooks/register" class="inline-flex items-center gap-2 bg-gray-900 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-800 transition">
              Register Webhook
            </a>
          {/snippet}
        </EmptyState>
      </div>
    {:else if filteredWebhooks.length === 0}
      <div class="bg-white border border-gray-200 rounded-lg">
        <EmptyState
          icon="filter_alt"
          title="No matching webhooks"
          description="Try adjusting your search or filter criteria."
        >
          {#snippet action()}
            <button
              onclick={() => { urlSearch = ''; handleHealthFilterChange(null); }}
              class="text-sm text-gray-600 hover:text-gray-900 underline"
            >
              Clear filters
            </button>
          {/snippet}
        </EmptyState>
      </div>
    {:else}
      <!-- Webhooks table -->
      <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead>
              <tr class="border-b border-gray-200 bg-gray-50/50">
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Webhook</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden md:table-cell">Namespace</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Events</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden lg:table-cell">Config</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Health</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider text-right">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100">
              {#each filteredWebhooks as wh}
                <tr
                  class="hover:bg-gray-50 cursor-pointer transition"
                  onclick={() => goto(`/webhooks/${wh.webhookId}`)}
                >
                  <td class="px-4 py-3">
                    <div class="flex flex-col gap-0.5">
                      <span class="font-medium text-gray-900 truncate max-w-xs" title={wh.url}>
                        {wh.url}
                      </span>
                      {#if wh.description}
                        <span class="text-xs text-gray-500 truncate max-w-xs">{wh.description}</span>
                      {/if}
                      <span class="text-xs text-gray-400 font-mono">{wh.webhookId.substring(0, 8)}...</span>
                      <!-- Show namespace + events inline on mobile -->
                      <span class="text-xs text-gray-400 md:hidden">ns: {wh.namespace}</span>
                      <div class="flex flex-wrap gap-1 mt-1 sm:hidden">
                        {#each wh.events.slice(0, 2) as event}
                          <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700">
                            {event}
                          </span>
                        {/each}
                        {#if wh.events.length > 2}
                          <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
                            +{wh.events.length - 2}
                          </span>
                        {/if}
                      </div>
                    </div>
                  </td>
                  <td class="px-4 py-3 hidden md:table-cell">
                    <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">
                      {wh.namespace}
                    </span>
                  </td>
                  <td class="px-4 py-3 hidden sm:table-cell">
                    <div class="flex flex-wrap gap-1">
                      {#each wh.events.slice(0, 3) as event}
                        <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700">
                          {event}
                        </span>
                      {/each}
                      {#if wh.events.length > 3}
                        <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-600">
                          +{wh.events.length - 3} more
                        </span>
                      {/if}
                    </div>
                  </td>
                  <td class="px-4 py-3 hidden lg:table-cell">
                    {#if wh.httpConfig}
                      <div class="flex flex-col gap-0.5 text-xs text-gray-600">
                        <span>{wh.httpConfig.maxRetries} retries</span>
                        <span>{wh.httpConfig.requestTimeoutSeconds}s timeout</span>
                      </div>
                    {:else}
                      <span class="text-xs text-gray-400">Default</span>
                    {/if}
                  </td>
                  <td class="px-4 py-3">
                    <HealthBadge health={wh.health} />
                  </td>
                  <td class="px-4 py-3">
                    <button
                      onclick={(e) => toggleActive(wh, e)}
                      class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 {wh.active ? 'bg-green-500' : 'bg-gray-300'}"
                      title={wh.active ? 'Click to pause' : 'Click to resume'}
                    >
                      <span
                        class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {wh.active ? 'translate-x-4' : 'translate-x-0'}"
                      ></span>
                    </button>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <button
                      onclick={(e) => promptUnregister(wh, e)}
                      class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium text-red-700 bg-red-50 hover:bg-red-100 transition"
                    >
                      Unregister
                    </button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>

        <!-- Pagination -->
        <div class="border-t border-gray-200 px-4">
          <Pagination
            {currentPage}
            {totalPages}
            {totalCount}
            pageSize={limit}
            onPageChange={handlePageChange}
            itemLabel="webhooks"
          />
        </div>
      </div>
    {/if}
  </main>
</div>

<!-- Confirm Unregister Dialog -->
<ConfirmDialog
  open={confirmUnregister}
  title="Unregister Webhook"
  message="This will permanently remove the webhook and stop all future deliveries. This action cannot be undone."
  confirmLabel="Unregister"
  variant="danger"
  onconfirm={executeUnregister}
  oncancel={() => { confirmUnregister = false; webhookToUnregister = null; }}
/>

<FloatingAction href="/webhooks/register" label="Register Webhook" targetSelector="#header-register-btn" />
