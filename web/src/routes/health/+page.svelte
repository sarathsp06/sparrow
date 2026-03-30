<script lang="ts">
  import { healthClient, webhookClient } from "$lib/services";
  import { onMount } from "svelte";
  import type {
    HealthSummary,
    NamespaceStats,
    RegisteredWebhook,
    WebhookHealthMetrics,
  } from "../../../../proto/webhook_pb.js";
  import { WebhookHealth } from "../../../../proto/webhook_pb.js";
  import HealthBadge from "$lib/components/HealthBadge.svelte";

  let healthSummary: HealthSummary | undefined = $state();
  let namespaceStats: NamespaceStats | undefined = $state();
  let unhealthyWebhooks: RegisteredWebhook[] = $state([]);
  let unhealthyMetrics: Map<string, WebhookHealthMetrics> = $state(new Map());
  let loading = $state(true);
  let error = $state("");

  async function fetchData() {
    loading = true;
    error = "";
    try {
      const [summaryRes, statsRes] = await Promise.all([
        healthClient.getHealthSummary({}),
        webhookClient.getNamespaceStats({ namespace: '' }),
      ]);
      healthSummary = summaryRes.summary;
      namespaceStats = statsRes.stats;

      // Fetch unhealthy webhooks with their health metrics
      const unhealthyRes = await healthClient.listWebhooksByHealth({
        health: WebhookHealth.HEALTH_UNHEALTHY,
        pagination: { limit: 10, offset: 0 },
      });
      unhealthyWebhooks = unhealthyRes.webhooks || [];

      // Fetch health metrics for each unhealthy webhook
      const metricsMap = new Map<string, WebhookHealthMetrics>();
      await Promise.all(
        unhealthyWebhooks.map(async (wh) => {
          try {
            const healthRes = await healthClient.getWebhookHealth({
              webhookId: wh.webhookId,
              namespace: wh.namespace,
            });
            if (healthRes.metrics) {
              metricsMap.set(wh.webhookId, healthRes.metrics);
            }
          } catch {
            // Ignore individual fetch failures
          }
        })
      );
      unhealthyMetrics = metricsMap;
    } catch (e: any) {
      error = `Failed to load health data: ${e.message}`;
    } finally {
      loading = false;
    }
  }

  onMount(fetchData);
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
              Statistics
            </h2>
            <p class="text-sm text-gray-500 mb-4">
              All Namespaces
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

        <!-- Unhealthy Webhooks with Error Breakdown -->
        {#if unhealthyWebhooks.length > 0}
          <div>
            <h2 class="text-lg font-semibold text-gray-900 mb-1">Unhealthy Webhooks</h2>
            <p class="text-sm text-gray-500 mb-4">Webhooks with delivery failures (last 24 hours)</p>
            <div class="space-y-3">
              {#each unhealthyWebhooks as wh}
                {@const metrics = unhealthyMetrics.get(wh.webhookId)}
                <a
                  href="/webhooks/{wh.webhookId}"
                  class="block bg-white rounded-lg border border-gray-200 px-4 py-4 hover:border-gray-300 hover:shadow-sm transition"
                >
                  <div class="flex items-center justify-between mb-2">
                    <div class="flex items-center gap-2 min-w-0">
                      <span class="text-sm font-medium text-gray-900 truncate">{wh.description || 'Webhook'}</span>
                      <HealthBadge health={wh.health} size="sm" />
                      <span class="px-1.5 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 rounded">{wh.namespace}</span>
                    </div>
                    <span class="text-xs font-mono text-gray-400 shrink-0 ml-2">{wh.webhookId.substring(0, 8)}...</span>
                  </div>

                  {#if metrics}
                    <div class="flex items-center gap-4 text-xs text-gray-500 mb-2">
                      <span>Success rate: <span class="font-medium {metrics.successRate >= 0.8 ? 'text-yellow-600' : 'text-red-600'}">{(metrics.successRate * 100).toFixed(1)}%</span></span>
                      <span>Failed: <span class="font-medium text-red-600">{metrics.failedDeliveries}</span></span>
                      <span>Consecutive: <span class="font-medium text-red-600">{metrics.consecutiveFailures}</span></span>
                    </div>

                    <!-- Error category breakdown bar -->
                    {@const totalErrors = (metrics.clientErrors || 0) + (metrics.serverErrors || 0) + (metrics.timeoutErrors || 0) + (metrics.networkErrors || 0)}
                    {#if totalErrors > 0}
                      <div class="flex items-center gap-3">
                        <div class="flex-1 h-2 rounded-full overflow-hidden flex bg-gray-100">
                          {#if metrics.clientErrors > 0}
                            <div class="bg-orange-400 h-full" style="width: {(metrics.clientErrors / totalErrors) * 100}%"></div>
                          {/if}
                          {#if metrics.serverErrors > 0}
                            <div class="bg-red-400 h-full" style="width: {(metrics.serverErrors / totalErrors) * 100}%"></div>
                          {/if}
                          {#if metrics.timeoutErrors > 0}
                            <div class="bg-yellow-400 h-full" style="width: {(metrics.timeoutErrors / totalErrors) * 100}%"></div>
                          {/if}
                          {#if metrics.networkErrors > 0}
                            <div class="bg-purple-400 h-full" style="width: {(metrics.networkErrors / totalErrors) * 100}%"></div>
                          {/if}
                        </div>
                        <div class="flex items-center gap-2 text-[10px] text-gray-500 shrink-0">
                          {#if metrics.clientErrors > 0}<span class="flex items-center gap-0.5"><span class="w-1.5 h-1.5 rounded-full bg-orange-400"></span>{metrics.clientErrors} 4xx</span>{/if}
                          {#if metrics.serverErrors > 0}<span class="flex items-center gap-0.5"><span class="w-1.5 h-1.5 rounded-full bg-red-400"></span>{metrics.serverErrors} 5xx</span>{/if}
                          {#if metrics.timeoutErrors > 0}<span class="flex items-center gap-0.5"><span class="w-1.5 h-1.5 rounded-full bg-yellow-400"></span>{metrics.timeoutErrors} timeout</span>{/if}
                          {#if metrics.networkErrors > 0}<span class="flex items-center gap-0.5"><span class="w-1.5 h-1.5 rounded-full bg-purple-400"></span>{metrics.networkErrors} network</span>{/if}
                        </div>
                      </div>
                    {/if}
                  {/if}
                </a>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}
  </main>
</div>
