<script lang="ts">
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import { onMount } from "svelte";
  import type { components } from "$lib/api-types";
  import HealthBadge from "$lib/components/HealthBadge.svelte";
  import CopyableId from "$lib/components/CopyableId.svelte";

  type HealthSummary = components["schemas"]["HealthSummaryOutputBody"];
  type NamespaceStats = components["schemas"]["NamespaceStatsOutputBody"];
  type WebhookOut = components["schemas"]["WebhookOut"];
  type WebhookHealthOutput = components["schemas"]["WebhookHealthOutputBody"];

  let healthSummary: HealthSummary | undefined = $state();
  let namespaceStats: NamespaceStats | undefined = $state();
  let unhealthyWebhooks: WebhookOut[] = $state([]);
  let degradedWebhooks: WebhookOut[] = $state([]);
  let webhookMetrics: Map<string, WebhookHealthOutput> = $state(new Map());
  let loading = $state(true);
  let error = $state("");

  async function fetchData() {
    loading = true;
    error = "";
    try {
      const [summary, stats, unhealthyRes, degradedRes] = await Promise.all([
        api.GET('/v1/health-summary'),
        api.GET('/v1/stats'),
        api.GET('/v1/webhooks', { params: { query: { health: 'unhealthy', limit: 20, offset: 0 } } }),
        api.GET('/v1/webhooks', { params: { query: { health: 'degraded', limit: 20, offset: 0 } } }),
      ]);
      healthSummary = unwrap(summary);
      namespaceStats = unwrap(stats);
      unhealthyWebhooks = unwrap(unhealthyRes).items || [];
      degradedWebhooks = unwrap(degradedRes).items || [];

      const allWebhooks = [...unhealthyWebhooks, ...degradedWebhooks];
      const metricsMap = new Map<string, WebhookHealthOutput>();
      await Promise.all(
        allWebhooks.map(async (wh) => {
          try {
            const healthRes = unwrap(await api.GET('/v1/namespaces/{namespace}/webhooks/{webhook_id}/health', {
              params: { path: { namespace: wh.namespace, webhook_id: wh.webhook_id } },
            }));
            metricsMap.set(wh.webhook_id, healthRes);
          } catch {
            // Ignore individual fetch failures
          }
        })
      );
      webhookMetrics = metricsMap;
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to load health data');
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
      <div class="space-y-8">
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
        {#if healthSummary}
          <div>
            <h2 class="text-lg font-semibold text-gray-900 mb-4">Overall Health</h2>
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4 text-center">
                <p class="text-3xl font-bold text-green-600">{healthSummary.healthy_count}</p>
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mt-1">Healthy</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4 text-center">
                <p class="text-3xl font-bold text-yellow-500">{healthSummary.degraded_count}</p>
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mt-1">Degraded</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4 text-center">
                <p class="text-3xl font-bold text-red-600">{healthSummary.unhealthy_count}</p>
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mt-1">Unhealthy</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4 text-center" title="Webhooks with no deliveries yet">
                <p class="text-3xl font-bold text-gray-400">{healthSummary.unknown_count}</p>
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mt-1">No Data</p>
              </div>
            </div>
          </div>
        {/if}

        {#if namespaceStats}
          <div>
            <h2 class="text-lg font-semibold text-gray-900 mb-1">Statistics</h2>
            <p class="text-sm text-gray-500 mb-4">All Namespaces</p>
            <div class="grid grid-cols-2 sm:grid-cols-3 gap-3">
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Total Webhooks</p>
                <p class="text-2xl font-bold text-gray-900 mt-1">{namespaceStats.total_webhooks}</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Active Webhooks</p>
                <p class="text-2xl font-bold text-green-600 mt-1">{namespaceStats.active_webhooks}</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Success Rate</p>
                <p class="text-2xl font-bold mt-1 {namespaceStats.success_rate >= 0.95 ? 'text-green-600' : namespaceStats.success_rate >= 0.8 ? 'text-yellow-600' : 'text-red-600'}">
                  {(namespaceStats.success_rate * 100).toFixed(1)}%
                </p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Successful Deliveries</p>
                <p class="text-2xl font-bold text-green-600 mt-1">{namespaceStats.successful_deliveries}</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Failed Deliveries</p>
                <p class="text-2xl font-bold text-red-600 mt-1">{namespaceStats.failed_deliveries}</p>
              </div>
              <div class="bg-white rounded-lg border border-gray-200 px-4 py-4">
                <p class="text-xs font-medium text-gray-500 uppercase tracking-wider">Pending Deliveries</p>
                <p class="text-2xl font-bold text-yellow-600 mt-1">{namespaceStats.pending_deliveries}</p>
              </div>
            </div>
          </div>
        {/if}

        {#snippet webhookCard(wh: WebhookOut)}
          {@const metrics = webhookMetrics.get(wh.webhook_id)}
          <a
            href="/webhooks/{wh.webhook_id}"
            class="block bg-white rounded-lg border border-gray-200 px-4 py-4 hover:border-gray-300 hover:shadow-sm transition"
          >
            <div class="flex items-center justify-between mb-2">
              <div class="flex items-center gap-2 min-w-0">
                <span class="text-sm font-medium text-gray-900 truncate">{wh.description || 'Webhook'}</span>
                <HealthBadge health={wh.health} size="sm" />
                <span class="px-1.5 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 rounded">{wh.namespace}</span>
              </div>
              <CopyableId id={wh.webhook_id} />
            </div>

            {#if metrics}
              <div class="flex items-center gap-4 text-xs text-gray-500 mb-2">
                <span>Success rate: <span class="font-medium {metrics.success_rate >= 0.8 ? 'text-yellow-600' : 'text-red-600'}">{(metrics.success_rate * 100).toFixed(1)}%</span></span>
                <span>Failed: <span class="font-medium text-red-600">{metrics.failed_deliveries}</span></span>
                <span>Consecutive: <span class="font-medium text-red-600">{metrics.consecutive_failures}</span></span>
              </div>

              {@const totalErrors = (metrics.client_errors || 0) + (metrics.server_errors || 0) + (metrics.timeout_errors || 0) + (metrics.network_errors || 0) + (metrics.unexpected_status_errors || 0)}
              {#if totalErrors > 0}
                <div class="flex items-center gap-3">
                  <div class="flex-1 h-2 rounded-full overflow-hidden flex bg-gray-100">
                    {#if metrics.client_errors > 0}
                      <div class="bg-orange-400 h-full" style="width: {(metrics.client_errors / totalErrors) * 100}%"></div>
                    {/if}
                    {#if metrics.server_errors > 0}
                      <div class="bg-red-400 h-full" style="width: {(metrics.server_errors / totalErrors) * 100}%"></div>
                    {/if}
                    {#if metrics.timeout_errors > 0}
                      <div class="bg-yellow-400 h-full" style="width: {(metrics.timeout_errors / totalErrors) * 100}%"></div>
                    {/if}
                    {#if metrics.network_errors > 0}
                      <div class="bg-purple-400 h-full" style="width: {(metrics.network_errors / totalErrors) * 100}%"></div>
                    {/if}
                    {#if metrics.unexpected_status_errors > 0}
                      <div class="bg-amber-400 h-full" style="width: {(metrics.unexpected_status_errors / totalErrors) * 100}%"></div>
                    {/if}
                  </div>
                  <div class="flex items-center gap-2 text-[10px] text-gray-500 shrink-0">
                    {#if metrics.client_errors > 0}<span class="flex items-center gap-0.5"><span class="w-1.5 h-1.5 rounded-full bg-orange-400"></span>{metrics.client_errors} 4xx</span>{/if}
                    {#if metrics.server_errors > 0}<span class="flex items-center gap-0.5"><span class="w-1.5 h-1.5 rounded-full bg-red-400"></span>{metrics.server_errors} 5xx</span>{/if}
                    {#if metrics.timeout_errors > 0}<span class="flex items-center gap-0.5"><span class="w-1.5 h-1.5 rounded-full bg-yellow-400"></span>{metrics.timeout_errors} timeout</span>{/if}
                    {#if metrics.network_errors > 0}<span class="flex items-center gap-0.5"><span class="w-1.5 h-1.5 rounded-full bg-purple-400"></span>{metrics.network_errors} network</span>{/if}
                    {#if metrics.unexpected_status_errors > 0}<span class="flex items-center gap-0.5"><span class="w-1.5 h-1.5 rounded-full bg-amber-400"></span>{metrics.unexpected_status_errors} unexpected</span>{/if}
                  </div>
                </div>
              {/if}
            {/if}
          </a>
        {/snippet}

        {#if unhealthyWebhooks.length > 0}
          <div>
            <h2 class="text-lg font-semibold text-gray-900 mb-1">Unhealthy Webhooks</h2>
            <p class="text-sm text-gray-500 mb-4">Critical: &lt;50% success rate or 10+ consecutive failures</p>
            <div class="space-y-3">
              {#each unhealthyWebhooks as wh}
                {@render webhookCard(wh)}
              {/each}
            </div>
          </div>
        {/if}

        {#if degradedWebhooks.length > 0}
          <div>
            <h2 class="text-lg font-semibold text-gray-900 mb-1">Degraded Webhooks</h2>
            <p class="text-sm text-gray-500 mb-4">Warning: 50-90% success rate or 3-9 consecutive failures</p>
            <div class="space-y-3">
              {#each degradedWebhooks as wh}
                {@render webhookCard(wh)}
              {/each}
            </div>
          </div>
        {/if}

        {#if unhealthyWebhooks.length === 0 && degradedWebhooks.length === 0}
          <div class="bg-white rounded-lg border border-gray-200 px-6 py-8 text-center">
            <svg class="w-10 h-10 text-green-400 mx-auto mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <p class="text-sm font-medium text-gray-900">All webhooks are healthy</p>
            <p class="text-xs text-gray-500 mt-1">No webhooks require attention right now</p>
          </div>
        {/if}
      </div>
    {/if}
  </main>
</div>
