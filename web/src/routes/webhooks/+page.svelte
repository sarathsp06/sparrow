<script lang="ts">
  import { goto } from '$app/navigation';
  import { webhookClient as client, healthClient } from '$lib/services';
  import { onMount } from 'svelte';
  import type { RegisteredWebhook } from '../../../../proto/webhook_pb.js';
  import { WebhookHealth } from '../../../../proto/webhook_pb.js';
  import HealthBadge from '$lib/components/HealthBadge.svelte';
  import CopyableId from '$lib/components/CopyableId.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import FloatingAction from '$lib/components/FloatingAction.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import { formatAPIError } from '$lib/utils';

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
      error = formatAPIError(e, 'Failed to unregister webhook');
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
      error = formatAPIError(e, 'Failed to update webhook');
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
            href="https://sarathsp06.github.io/sparrow/getting-started/how-it-works/"
            target="_blank"
            rel="noopener noreferrer"
            class="inline-flex items-center gap-2 border border-gray-300 bg-white text-gray-700 px-5 py-2.5 rounded-lg text-sm font-medium hover:bg-gray-50 hover:border-gray-400 transition shadow-sm"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
            Read the docs
          </a>
        </div>
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
                      <CopyableId id={wh.webhookId} href="/webhooks/{wh.webhookId}" />
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
