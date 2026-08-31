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

<main class="mx-auto max-w-[1600px] px-4 sm:px-8 py-8">
  <div class="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 mb-6">
    <div>
      <p class="eyebrow mb-1.5">System / Health</p>
      <h1 class="text-2xl">Health Dashboard</h1>
      <p class="text-sm text-muted mt-1">Fleet health and namespace statistics</p>
    </div>
    <button onclick={() => fetchData()} disabled={loading} class="btn btn-ghost" aria-label="Refresh health data">
      <svg class="w-4 h-4 {loading ? 'animate-spin' : ''}" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
      </svg>
      {loading ? 'Refreshing…' : 'Refresh'}
    </button>
  </div>

  {#if loading}
    <div class="space-y-6">
      <div class="panel readout animate-pulse">
        {#each Array(4) as _}
          <div class="cell"><div class="h-7 w-10 bg-white/5 rounded mb-2"></div><div class="h-3 w-16 bg-white/[0.03] rounded"></div></div>
        {/each}
      </div>
      <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
        {#each Array(6) as _}
          <div class="panel px-4 py-4 animate-pulse"><div class="h-3 w-20 bg-white/[0.03] rounded mb-2"></div><div class="h-6 w-14 bg-white/5 rounded"></div></div>
        {/each}
      </div>
    </div>
  {:else if error}
    <div class="panel p-4" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
      <div class="flex items-start gap-3">
        <svg class="w-5 h-5 mt-0.5 shrink-0" style="color:var(--color-bad)" fill="currentColor" viewBox="0 0 20 20" aria-hidden="true">
          <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
        </svg>
        <div>
          <p class="text-sm font-medium" style="color:var(--color-bad)">{error}</p>
          <button onclick={() => { error = ''; fetchData(); }} class="text-sm link-beacon underline mt-1">Retry</button>
        </div>
      </div>
    </div>
  {:else}
    <div class="space-y-8" aria-live="polite">
      {#if healthSummary}
        <div>
          <h2 class="eyebrow mb-3">Overall Health</h2>
          <div class="panel readout">
            <div class="cell"><div class="val" style="color:var(--color-ok)">{healthSummary.healthy_count}</div><div class="key">Healthy</div></div>
            <div class="cell"><div class="val" style="color:var(--color-warn)">{healthSummary.degraded_count}</div><div class="key">Degraded</div></div>
            <div class="cell"><div class="val" style="color:var(--color-bad)">{healthSummary.unhealthy_count}</div><div class="key">Unhealthy</div></div>
            <div class="cell" title="Webhooks with no deliveries yet"><div class="val" style="color:var(--color-idle)">{healthSummary.unknown_count}</div><div class="key">No Data</div></div>
          </div>
        </div>
      {/if}

      {#if namespaceStats}
        <div>
          <h2 class="eyebrow mb-3">Statistics · All Namespaces</h2>
          <div class="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-6 gap-3">
            <div class="panel px-4 py-4">
              <p class="key">Total Webhooks</p>
              <p class="text-2xl font-semibold tnum mt-1 text-text">{namespaceStats.total_webhooks}</p>
            </div>
            <div class="panel px-4 py-4">
              <p class="key">Active Webhooks</p>
              <p class="text-2xl font-semibold tnum mt-1" style="color:var(--color-ok)">{namespaceStats.active_webhooks}</p>
            </div>
            <div class="panel px-4 py-4">
              <p class="key">Success Rate</p>
              <p class="text-2xl font-semibold tnum mt-1" style="color:var(--color-{namespaceStats.success_rate >= 0.95 ? 'ok' : namespaceStats.success_rate >= 0.8 ? 'warn' : 'bad'})">
                {(namespaceStats.success_rate * 100).toFixed(1)}%
              </p>
            </div>
            <div class="panel px-4 py-4">
              <p class="key">Successful</p>
              <p class="text-2xl font-semibold tnum mt-1" style="color:var(--color-ok)">{namespaceStats.successful_deliveries}</p>
            </div>
            <div class="panel px-4 py-4">
              <p class="key">Failed</p>
              <p class="text-2xl font-semibold tnum mt-1" style="color:var(--color-bad)">{namespaceStats.failed_deliveries}</p>
            </div>
            <div class="panel px-4 py-4">
              <p class="key">Pending</p>
              <p class="text-2xl font-semibold tnum mt-1" style="color:var(--color-warn)">{namespaceStats.pending_deliveries}</p>
            </div>
          </div>
        </div>
      {/if}

      {#snippet webhookCard(wh: WebhookOut)}
        {@const metrics = webhookMetrics.get(wh.webhook_id)}
        <a href="/webhooks/{wh.webhook_id}" class="block panel-2 px-4 py-4 hover:border-line-strong transition-colors">
          <div class="flex items-center justify-between mb-2 gap-2">
            <div class="flex items-center gap-2 min-w-0">
              <span class="text-sm font-medium text-text truncate">{wh.description || 'Webhook'}</span>
              <HealthBadge health={wh.health} size="sm" />
              <span class="chip">{wh.namespace}</span>
            </div>
            <CopyableId id={wh.webhook_id} />
          </div>

          {#if metrics}
            <div class="flex items-center gap-4 text-xs text-muted mb-2">
              <span>Success rate: <span class="mono tnum font-medium" style="color:var(--color-{metrics.success_rate >= 0.8 ? 'warn' : 'bad'})">{(metrics.success_rate * 100).toFixed(1)}%</span></span>
              <span>Failed: <span class="mono tnum font-medium" style="color:var(--color-bad)">{metrics.failed_deliveries}</span></span>
              <span>Consecutive: <span class="mono tnum font-medium" style="color:var(--color-bad)">{metrics.consecutive_failures}</span></span>
            </div>

            {@const totalErrors = (metrics.client_errors || 0) + (metrics.server_errors || 0) + (metrics.timeout_errors || 0) + (metrics.network_errors || 0) + (metrics.unexpected_status_errors || 0)}
            {#if totalErrors > 0}
              {@const segs = [
                { n: metrics.client_errors, tone: 'warn', label: '4xx' },
                { n: metrics.server_errors, tone: 'bad', label: '5xx' },
                { n: metrics.timeout_errors, tone: 'beacon', label: 'timeout' },
                { n: metrics.network_errors, tone: 'idle', label: 'network' },
                { n: metrics.unexpected_status_errors, tone: 'faint', label: 'unexpected' },
              ]}
              <div class="flex items-center gap-3">
                <div class="flex-1 h-2 rounded-full overflow-hidden flex bg-ink">
                  {#each segs as s}
                    {#if s.n > 0}<div class="h-full" style="width:{(s.n / totalErrors) * 100}%;background:var(--color-{s.tone})"></div>{/if}
                  {/each}
                </div>
                <div class="flex items-center gap-2 text-[10px] text-muted mono shrink-0">
                  {#each segs as s}
                    {#if s.n > 0}<span class="flex items-center gap-0.5"><span class="w-1.5 h-1.5 rounded-full" style="background:var(--color-{s.tone})"></span>{s.n} {s.label}</span>{/if}
                  {/each}
                </div>
              </div>
            {/if}
          {/if}
        </a>
      {/snippet}

      {#if unhealthyWebhooks.length > 0}
        <div>
          <h2 class="eyebrow mb-1" style="color:var(--color-bad)">Unhealthy Webhooks</h2>
          <p class="text-sm text-muted mb-4">Critical: &lt;50% success rate or 10+ consecutive failures</p>
          <div class="space-y-3">
            {#each unhealthyWebhooks as wh}{@render webhookCard(wh)}{/each}
          </div>
        </div>
      {/if}

      {#if degradedWebhooks.length > 0}
        <div>
          <h2 class="eyebrow mb-1" style="color:var(--color-warn)">Degraded Webhooks</h2>
          <p class="text-sm text-muted mb-4">Warning: 50–90% success rate or 3–9 consecutive failures</p>
          <div class="space-y-3">
            {#each degradedWebhooks as wh}{@render webhookCard(wh)}{/each}
          </div>
        </div>
      {/if}

      {#if unhealthyWebhooks.length === 0 && degradedWebhooks.length === 0}
        <div class="panel px-6 py-10 text-center" style="border-color:color-mix(in srgb,var(--color-ok) 30%,transparent)">
          <svg class="w-10 h-10 mx-auto mb-3" style="color:var(--color-ok)" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <p class="text-sm font-medium text-text">All webhooks are healthy</p>
          <p class="text-xs text-muted mt-1">No webhooks require attention right now</p>
        </div>
      {/if}
    </div>
  {/if}
</main>
