<script lang="ts">
  import { healthClient, webhookClient } from "$lib/services";
  import { namespaceState } from "$lib/namespace.svelte";
  import { onMount, untrack } from "svelte";
  import type {
    HealthSummary,
    NamespaceStats,
  } from "../../../../proto/webhook_pb.js";

  let healthSummary: HealthSummary | undefined = $state();
  let namespaceStats: NamespaceStats | undefined = $state();
  let loading = $state(true);
  let error = $state("");

  async function fetchData() {
    loading = true;
    error = "";
    try {
      const summaryRes = await healthClient.getHealthSummary({});
      healthSummary = summaryRes.summary;

      const statsRes = await webhookClient.getNamespaceStats({ namespace: namespaceState.current });
      namespaceStats = statsRes.stats;
    } catch (e: any) {
      error = `Failed to load health data: ${e.message}`;
    } finally {
      loading = false;
    }
  }

  onMount(fetchData);

  // Re-fetch when namespace changes
  $effect(() => {
    const ns = namespaceState.current;
    untrack(() => {
      fetchData();
    });
  });
</script>

<svelte:head>
  <title>Health | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    <!-- Page header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">Health Dashboard</h1>
        <p class="text-sm text-gray-500 mt-0.5">System health and namespace statistics</p>
      </div>
      <button
        onclick={() => fetchData()}
        class="inline-flex items-center gap-2 bg-white text-gray-700 border border-gray-300 px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-50 transition shadow-sm disabled:opacity-50"
        disabled={loading}
      >
        <svg class="w-4 h-4 {loading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
        </svg>
        {loading ? 'Refreshing...' : 'Refresh'}
      </button>
    </div>

    {#if loading}
      <!-- Loading skeleton -->
      <div class="space-y-8">
        <!-- Overall health skeleton -->
        <div>
          <div class="h-6 bg-gray-200 rounded w-36 mb-4 animate-pulse"></div>
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
            {#each Array(4) as _}
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4 animate-pulse">
                <div class="h-8 bg-gray-200 rounded w-12 mx-auto mb-2"></div>
                <div class="h-4 bg-gray-100 rounded w-20 mx-auto"></div>
              </div>
            {/each}
          </div>
        </div>
        <!-- Namespace stats skeleton -->
        <div>
          <div class="h-6 bg-gray-200 rounded w-48 mb-4 animate-pulse"></div>
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
            {#each Array(6) as _}
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4 animate-pulse">
                <div class="h-4 bg-gray-100 rounded w-28 mb-2"></div>
                <div class="h-8 bg-gray-200 rounded w-16"></div>
              </div>
            {/each}
          </div>
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
            <button onclick={() => { error = ''; fetchData(); }} class="text-sm text-red-600 hover:text-red-800 underline mt-1">Retry</button>
          </div>
        </div>
      </div>
    {:else}
      <div class="space-y-8">
        <!-- Overall Health -->
        {#if healthSummary}
          <div>
            <h2 class="text-lg font-semibold text-gray-900 mb-4">Overall Health</h2>
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4 text-center">
                <p class="text-3xl font-bold text-green-600">{healthSummary.healthyCount}</p>
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mt-1">Healthy</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4 text-center">
                <p class="text-3xl font-bold text-yellow-500">{healthSummary.degradedCount}</p>
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mt-1">Degraded</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4 text-center">
                <p class="text-3xl font-bold text-red-600">{healthSummary.unhealthyCount}</p>
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mt-1">Unhealthy</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4 text-center">
                <p class="text-3xl font-bold text-gray-400">{healthSummary.unknownCount}</p>
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mt-1">Unknown</p>
              </div>
            </div>
          </div>
        {/if}

        <!-- Namespace Stats -->
        {#if namespaceStats}
          <div>
            <h2 class="text-lg font-semibold text-gray-900 mb-1">
              Namespace Statistics
            </h2>
            <p class="text-sm text-gray-500 mb-4">
              Namespace: <span class="font-semibold text-gray-700">{namespaceState.current}</span>
            </p>
            <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Total Webhooks</p>
                <p class="text-2xl font-bold text-gray-900 mt-1">{namespaceStats.totalWebhooks}</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Active Webhooks</p>
                <p class="text-2xl font-bold text-green-600 mt-1">{namespaceStats.activeWebhooks}</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Success Rate</p>
                <p class="text-2xl font-bold mt-1 {namespaceStats.successRate >= 0.95 ? 'text-green-600' : namespaceStats.successRate >= 0.8 ? 'text-yellow-600' : 'text-red-600'}">
                  {(namespaceStats.successRate * 100).toFixed(1)}%
                </p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Successful Deliveries</p>
                <p class="text-2xl font-bold text-green-600 mt-1">{namespaceStats.successfulDeliveries}</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Failed Deliveries</p>
                <p class="text-2xl font-bold text-red-600 mt-1">{namespaceStats.failedDeliveries}</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Pending Deliveries</p>
                <p class="text-2xl font-bold text-yellow-600 mt-1">{namespaceStats.pendingDeliveries}</p>
              </div>
            </div>
          </div>
        {/if}
      </div>
    {/if}
  </main>
</div>
