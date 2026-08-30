<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, unwrap } from '$lib/services';
  import { onMount } from 'svelte';
  import type { components } from '$lib/api-types';
  import HealthBadge from '$lib/components/HealthBadge.svelte';
  import CopyableId from '$lib/components/CopyableId.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import FloatingAction from '$lib/components/FloatingAction.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import { formatAPIError } from '$lib/utils';

  type WebhookOut = components["schemas"]["WebhookOut"];

  let webhooks: WebhookOut[] = $state([]);
  let loading = $state(true);
  let error = $state('');

  let healthFilter = $state<string | null>(null);
  let urlSearch = $state('');
  let namespace = $state('default');

  let limit = $state(25);
  let offset = $state(0);
  let totalCount = $state(0);

  let confirmUnregister = $state(false);
  let webhookToUnregister = $state<WebhookOut | null>(null);

  let stats = $state<{ total: number; active: number; healthy: number; unhealthy: number }>({
    total: 0, active: 0, healthy: 0, unhealthy: 0,
  });

  const healthFilters: { value: string | null; label: string }[] = [
    { value: null, label: 'All' },
    { value: 'healthy', label: 'Healthy' },
    { value: 'degraded', label: 'Degraded' },
    { value: 'unhealthy', label: 'Unhealthy' },
  ];

  let filteredWebhooks = $derived.by(() => {
    let result = webhooks;
    if (urlSearch.trim()) {
      const q = urlSearch.toLowerCase();
      result = result.filter(w =>
        w.url.toLowerCase().includes(q) ||
        (w.description ?? '').toLowerCase().includes(q) ||
        w.webhook_id.toLowerCase().includes(q) ||
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
      if (healthFilter !== null) {
        // Global endpoint — health is computed per-webhook, independent of namespace.
        const res = unwrap(await api.GET('/v1/webhooks', {
          params: { query: { health: healthFilter as any, limit, offset } },
        }));
        webhooks = res.items || [];
        totalCount = res.pagination?.total_count || 0;
      } else {
        const res = unwrap(await api.GET('/v1/namespaces/{namespace}/webhooks', {
          params: { path: { namespace }, query: { limit, offset } },
        }));
        webhooks = res.items || [];
        totalCount = res.pagination?.total_count || 0;
      }

      stats = {
        total: totalCount,
        active: webhooks.filter(w => w.active).length,
        healthy: webhooks.filter(w => w.health === 'healthy').length,
        unhealthy: webhooks.filter(w => w.health === 'unhealthy').length,
      };
    } catch (e: any) {
      console.error(e);
      error = formatAPIError(e, 'Failed to load webhooks');
    } finally {
      loading = false;
    }
  }

  onMount(fetchWebhooks);

  function handlePageChange(pageNum: number) {
    offset = (pageNum - 1) * limit;
    fetchWebhooks();
  }

  function handleHealthFilterChange(health: string | null) {
    healthFilter = health;
    offset = 0;
    fetchWebhooks();
  }

  function promptUnregister(wh: WebhookOut, e: Event) {
    e.stopPropagation();
    webhookToUnregister = wh;
    confirmUnregister = true;
  }

  async function executeUnregister() {
    if (!webhookToUnregister) return;
    try {
      unwrap(await api.DELETE('/v1/namespaces/{namespace}/webhooks/{webhook_id}', {
        params: { path: { namespace: webhookToUnregister.namespace, webhook_id: webhookToUnregister.webhook_id } },
      }));
      confirmUnregister = false;
      webhookToUnregister = null;
      await fetchWebhooks();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to unregister webhook');
      confirmUnregister = false;
    }
  }

  async function toggleActive(wh: WebhookOut, e: Event) {
    e.stopPropagation();
    try {
      if (wh.active) {
        unwrap(await api.POST('/v1/namespaces/{namespace}/webhooks/{webhook_id}:pause', {
          params: { path: { namespace: wh.namespace, webhook_id: wh.webhook_id } },
        }));
      } else {
        unwrap(await api.POST('/v1/namespaces/{namespace}/webhooks/{webhook_id}:resume', {
          params: { path: { namespace: wh.namespace, webhook_id: wh.webhook_id } },
        }));
      }
      await fetchWebhooks();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to update webhook');
    }
  }
</script>

<svelte:head>
  <title>Webhooks | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
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

    {#if !loading && !error}
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 mb-6">
        <div class="bg-white rounded-lg border border-gray-200 px-4 py-3 text-center">
          <p class="text-2xl font-bold text-gray-900">{stats.total}</p>
          <p class="text-xs text-gray-500 mt-0.5">Total</p>
        </div>
        <div class="bg-white rounded-lg border border-gray-200 px-4 py-3 text-center">
          <p class="text-2xl font-bold text-green-600">{stats.active}</p>
          <p class="text-xs text-gray-500 mt-0.5">Active</p>
        </div>
        <div class="bg-white rounded-lg border border-gray-200 px-4 py-3 text-center">
          <p class="text-2xl font-bold text-green-600">{stats.healthy}</p>
          <p class="text-xs text-gray-500 mt-0.5">Healthy</p>
        </div>
        <div class="bg-white rounded-lg border border-gray-200 px-4 py-3 text-center">
          <p class="text-2xl font-bold text-red-600">{stats.unhealthy}</p>
          <p class="text-xs text-gray-500 mt-0.5">Unhealthy</p>
        </div>
      </div>
    {/if}

    <div class="flex flex-col sm:flex-row gap-3 mb-4">
      <input type="text" placeholder="Namespace" bind:value={namespace} onchange={() => { offset = 0; fetchWebhooks(); }} class="w-40 px-3 py-2 border border-gray-300 rounded-lg text-sm" />
      <input type="text" placeholder="Search by URL, description, or ID..." bind:value={urlSearch} class="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm" />
      <div class="flex gap-1">
        {#each healthFilters as f}
          <button
            onclick={() => handleHealthFilterChange(f.value)}
            class="px-3 py-1.5 text-xs font-medium rounded-lg transition {healthFilter === f.value ? 'bg-gray-900 text-white' : 'bg-white text-gray-600 border border-gray-300 hover:bg-gray-50'}"
          >
            {f.label}
          </button>
        {/each}
      </div>
    </div>

    {#if loading}
      <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div class="animate-pulse divide-y divide-gray-100">
          {#each Array(5) as _}
            <div class="p-4 h-14 bg-gray-50"></div>
          {/each}
        </div>
      </div>
    {:else if error}
      <div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
        <p class="text-sm text-red-700">{error}</p>
      </div>
    {:else if webhooks.length === 0}
      <div class="bg-white border border-gray-200 rounded-lg p-8">
        <!-- Header -->
        <div class="text-center mb-6">
          <h3 class="text-xl font-bold text-gray-900 mb-2">How Sparrow works</h3>
          <p class="text-sm text-gray-500 max-w-lg mx-auto">Register events and webhooks, push events, and Sparrow fans out deliveries with retries and health tracking.</p>
        </div>

        <!-- Compact flow diagram -->
        <div class="max-w-3xl mx-auto mb-8">
          <div class="bg-gray-50 border border-gray-200 rounded-lg p-5">
            <!-- Flow: horizontal on desktop, vertical on mobile -->
            <div class="hidden sm:flex items-center justify-center gap-0">
              <!-- Push Event -->
              <div class="flex flex-col items-center text-center min-w-0">
                <div class="w-10 h-10 rounded-lg bg-gray-900 text-white flex items-center justify-center mb-1.5">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7l5 5m0 0l-5 5m5-5H6" /></svg>
                </div>
                <span class="text-xs font-semibold text-gray-900">Push Event</span>
              </div>
              <div class="w-8 border-t-2 border-dashed border-gray-300 shrink-0"></div>
              <!-- Event Worker -->
              <div class="flex flex-col items-center text-center min-w-0">
                <div class="w-10 h-10 rounded-lg bg-gray-900 text-white flex items-center justify-center mb-1.5">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" /></svg>
                </div>
                <span class="text-xs font-semibold text-gray-900">Event Worker</span>
                <span class="text-[10px] text-gray-400">find subscriptions</span>
              </div>
              <div class="w-8 border-t-2 border-dashed border-gray-300 shrink-0"></div>
              <!-- Fan Out -->
              <div class="flex flex-col items-center text-center min-w-0">
                <div class="w-10 h-10 rounded-lg bg-gray-900 text-white flex items-center justify-center mb-1.5">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" /></svg>
                </div>
                <span class="text-xs font-semibold text-gray-900">Fan Out</span>
                <span class="text-[10px] text-gray-400">create deliveries</span>
              </div>
              <div class="w-8 border-t-2 border-dashed border-gray-300 shrink-0"></div>
              <!-- Webhook Worker -->
              <div class="flex flex-col items-center text-center min-w-0">
                <div class="w-10 h-10 rounded-lg bg-gray-900 text-white flex items-center justify-center mb-1.5">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" /></svg>
                </div>
                <span class="text-xs font-semibold text-gray-900">Deliver</span>
                <span class="text-[10px] text-gray-400">HTTP POST + HMAC</span>
              </div>
              <div class="w-8 border-t-2 border-dashed border-gray-300 shrink-0"></div>
              <!-- Result -->
              <div class="flex flex-col items-center text-center min-w-0">
                <div class="flex gap-1.5">
                  <div class="flex flex-col items-center">
                    <div class="w-10 h-10 rounded-lg bg-green-600 text-white flex items-center justify-center mb-1.5">
                      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                    </div>
                    <span class="text-xs font-semibold text-green-700">Done</span>
                  </div>
                  <div class="flex flex-col items-center">
                    <div class="w-10 h-10 rounded-lg bg-amber-500 text-white flex items-center justify-center mb-1.5">
                      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                    </div>
                    <span class="text-xs font-semibold text-amber-600">Retry</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- Mobile: vertical flow -->
            <div class="sm:hidden flex flex-col items-center gap-0">
              <div class="flex items-center gap-3 w-full max-w-xs">
                <div class="w-9 h-9 rounded-lg bg-gray-900 text-white flex items-center justify-center shrink-0">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7l5 5m0 0l-5 5m5-5H6" /></svg>
                </div>
                <div><span class="text-xs font-semibold text-gray-900">Push Event</span></div>
              </div>
              <div class="h-4 w-px border-l-2 border-dashed border-gray-300"></div>
              <div class="flex items-center gap-3 w-full max-w-xs">
                <div class="w-9 h-9 rounded-lg bg-gray-900 text-white flex items-center justify-center shrink-0">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" /></svg>
                </div>
                <div><span class="text-xs font-semibold text-gray-900">Event Worker</span> <span class="text-[10px] text-gray-400">-- find subscriptions</span></div>
              </div>
              <div class="h-4 w-px border-l-2 border-dashed border-gray-300"></div>
              <div class="flex items-center gap-3 w-full max-w-xs">
                <div class="w-9 h-9 rounded-lg bg-gray-900 text-white flex items-center justify-center shrink-0">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" /></svg>
                </div>
                <div><span class="text-xs font-semibold text-gray-900">Fan Out</span> <span class="text-[10px] text-gray-400">-- create deliveries</span></div>
              </div>
              <div class="h-4 w-px border-l-2 border-dashed border-gray-300"></div>
              <div class="flex items-center gap-3 w-full max-w-xs">
                <div class="w-9 h-9 rounded-lg bg-gray-900 text-white flex items-center justify-center shrink-0">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" /></svg>
                </div>
                <div><span class="text-xs font-semibold text-gray-900">Deliver</span> <span class="text-[10px] text-gray-400">-- HTTP POST + HMAC</span></div>
              </div>
              <div class="h-4 w-px border-l-2 border-dashed border-gray-300"></div>
              <div class="flex items-center gap-3 w-full max-w-xs">
                <div class="flex gap-1.5 shrink-0">
                  <div class="w-9 h-9 rounded-lg bg-green-600 text-white flex items-center justify-center">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                  </div>
                  <div class="w-9 h-9 rounded-lg bg-amber-500 text-white flex items-center justify-center">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                  </div>
                </div>
                <div><span class="text-xs font-semibold text-green-700">Done</span> <span class="text-gray-400">/</span> <span class="text-xs font-semibold text-amber-600">Retry with backoff</span></div>
              </div>
            </div>
          </div>
        </div>

        <!-- 3-step getting started -->
        <div class="max-w-2xl mx-auto mb-8">
          <p class="text-xs text-gray-400 uppercase tracking-wide font-semibold mb-4">Get started in 3 steps</p>

          <!-- Step 1 -->
          <div class="flex gap-4 mb-1">
            <div class="flex flex-col items-center">
              <span class="inline-flex items-center justify-center w-8 h-8 rounded-full bg-gray-900 text-white text-sm font-bold shrink-0">1</span>
              <div class="w-px h-full bg-gray-200 mt-1"></div>
            </div>
            <div class="pb-6">
              <h4 class="text-sm font-semibold text-gray-900 mb-1">Create an event</h4>
              <p class="text-sm text-gray-500 mb-2">Define what happened in your system -- like <code class="text-xs bg-gray-100 px-1.5 py-0.5 rounded">order.created</code> or <code class="text-xs bg-gray-100 px-1.5 py-0.5 rounded">user.signed_up</code>.</p>
              <a href="/events" class="inline-flex items-center gap-1 text-xs text-gray-600 hover:text-gray-900 transition">
                Go to Events <span class="text-[10px]">&rarr;</span>
              </a>
            </div>
          </div>

          <!-- Step 2 -->
          <div class="flex gap-4 mb-1">
            <div class="flex flex-col items-center">
              <span class="inline-flex items-center justify-center w-8 h-8 rounded-full bg-gray-900 text-white text-sm font-bold shrink-0">2</span>
              <div class="w-px h-full bg-gray-200 mt-1"></div>
            </div>
            <div class="pb-6">
              <h4 class="text-sm font-semibold text-gray-900 mb-1">Register a webhook</h4>
              <p class="text-sm text-gray-500 mb-2">Add your endpoint URL and pick the events it should receive. Sparrow auto-creates the subscriptions for you.</p>
              <a href="/webhooks/register" class="inline-flex items-center gap-1 text-xs text-gray-600 hover:text-gray-900 transition">
                Register a webhook <span class="text-[10px]">&rarr;</span>
              </a>
            </div>
          </div>

          <!-- Step 3 -->
          <div class="flex gap-4 mb-1">
            <div class="flex flex-col items-center">
              <span class="inline-flex items-center justify-center w-8 h-8 rounded-full bg-gray-900 text-white text-sm font-bold shrink-0">3</span>
            </div>
            <div class="pb-2">
              <h4 class="text-sm font-semibold text-gray-900 mb-1">Push an event</h4>
              <p class="text-sm text-gray-500 mb-2">Send an event payload and watch Sparrow deliver it to your webhook. Check the result in <a href="/deliveries" class="underline hover:text-gray-900">Deliveries</a>.</p>
              <a href="/events/push" class="inline-flex items-center gap-1 text-xs text-gray-600 hover:text-gray-900 transition">
                Push an event <span class="text-[10px]">&rarr;</span>
              </a>
            </div>
          </div>
        </div>

        <!-- After you're set up -->
        <div class="max-w-2xl mx-auto border-t border-gray-100 pt-6 mb-6">
          <p class="text-xs text-gray-400 uppercase tracking-wide font-semibold mb-3">Then customize</p>
          <div class="flex flex-wrap gap-x-6 gap-y-2 text-sm text-gray-500">
            <span>Payload transforms on subscriptions</span>
            <span>Retry policies and timeouts</span>
            <span>HMAC signature verification</span>
            <span>Health monitoring</span>
          </div>
        </div>

        <!-- Actions -->
        <div class="flex flex-col sm:flex-row items-center justify-center gap-3">
          <a href="/events" class="inline-flex items-center gap-2 bg-gray-900 text-white px-5 py-2.5 rounded-lg text-sm font-medium hover:bg-gray-800 transition shadow-sm">
            Start with Step 1
          </a>
          <a
            href="/docs"
            class="inline-flex items-center gap-2 border border-gray-300 bg-white text-gray-700 px-5 py-2.5 rounded-lg text-sm font-medium hover:bg-gray-50 hover:border-gray-400 transition shadow-sm"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
            Read the API docs
          </a>
        </div>
      </div>
    {:else if filteredWebhooks.length === 0}
      <div class="bg-white border border-gray-200 rounded-lg">
        <EmptyState icon="search" title="No matching webhooks" description="Try a different search term." />
      </div>
    {:else}
      <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead>
              <tr class="border-b border-gray-200 bg-gray-50/50">
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Webhook</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Namespace</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Health</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider"></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100">
              {#each filteredWebhooks as wh}
                <tr class="hover:bg-gray-50 transition cursor-pointer" onclick={() => goto(`/webhooks/${wh.webhook_id}`)}>
                  <td class="px-4 py-3">
                    <p class="font-medium text-gray-900 truncate max-w-xs">{wh.description || wh.url}</p>
                    <div class="mt-0.5"><CopyableId id={wh.webhook_id} truncate={12} /></div>
                  </td>
                  <td class="px-4 py-3 hidden sm:table-cell">
                    <span class="px-1.5 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 rounded">{wh.namespace}</span>
                  </td>
                  <td class="px-4 py-3"><HealthBadge health={wh.health} /></td>
                  <td class="px-4 py-3">
                    <button
                      onclick={(e) => toggleActive(wh, e)}
                      class="px-2 py-0.5 text-xs font-medium rounded-full transition {wh.active ? 'bg-green-100 text-green-700 hover:bg-green-200' : 'bg-gray-100 text-gray-500 hover:bg-gray-200'}"
                    >
                      {wh.active ? 'Active' : 'Paused'}
                    </button>
                  </td>
                  <td class="px-4 py-3 text-right">
                    <button onclick={(e) => promptUnregister(wh, e)} class="text-xs text-red-500 hover:text-red-700 underline">Unregister</button>
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>

      <Pagination {currentPage} {totalPages} {totalCount} pageSize={limit} onPageChange={handlePageChange} />
    {/if}
  </main>
</div>

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
