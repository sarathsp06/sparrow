<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, unwrap } from '$lib/services';
  import type { components } from '$lib/api-types';
  import HealthBadge from '$lib/components/HealthBadge.svelte';
  import CopyableId from '$lib/components/CopyableId.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import FloatingAction from '$lib/components/FloatingAction.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import { formatAPIError } from '$lib/utils';
  import { namespaceStore } from '$lib/namespace.svelte';

  type WebhookOut = components["schemas"]["WebhookOut"];

  let webhooks: WebhookOut[] = $state([]);
  let loading = $state(true);
  let error = $state('');

  let healthFilter = $state<string | null>(null);
  let urlSearch = $state('');


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
          params: { path: { namespace: namespaceStore.value }, query: { limit, offset } },
        }));
        webhooks = res.items || [];
        totalCount = res.pagination?.total_count || 0;
      }
      namespaceStore.remember(...webhooks.map((w) => w.namespace));

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

  $effect(() => {
    namespaceStore.value; // refetch when the active namespace changes
    offset = 0;
    fetchWebhooks();
  });

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

<main class="mx-auto max-w-[1600px] px-4 sm:px-8 py-8">
  <div class="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 mb-6">
    <div>
      <p class="eyebrow mb-1.5">Fleet / Webhooks</p>
      <h1 class="text-2xl">Webhooks</h1>
      <p class="text-sm text-muted mt-1">Endpoints receiving your dispatched events</p>
    </div>
    <a id="header-register-btn" href="/webhooks/register" class="btn btn-beacon">
      <span class="text-lg leading-none">+</span>
      Register Webhook
    </a>
  </div>

  {#if !loading && !error}
    <div class="panel readout mb-6">
      <div class="cell">
        <div class="val text-text">{stats.total}</div>
        <div class="key">Total</div>
      </div>
      <div class="cell">
        <div class="val" style="color:var(--color-ok)">{stats.active}</div>
        <div class="key">Active</div>
      </div>
      <button type="button" onclick={() => handleHealthFilterChange('healthy')} aria-pressed={healthFilter === 'healthy'} class="cell text-left transition-colors hover:bg-white/[0.03] {healthFilter === 'healthy' ? 'bg-white/5' : ''}">
        <div class="val" style="color:var(--color-ok)">{stats.healthy}</div>
        <div class="key">Healthy ▸</div>
      </button>
      <button type="button" onclick={() => handleHealthFilterChange('unhealthy')} aria-pressed={healthFilter === 'unhealthy'} class="cell text-left transition-colors hover:bg-white/[0.03] {healthFilter === 'unhealthy' ? 'bg-white/5' : ''}">
        <div class="val" style="color:var(--color-bad)">{stats.unhealthy}</div>
        <div class="key">Unhealthy ▸</div>
      </button>
    </div>
  {/if}

  <div class="flex flex-col sm:flex-row gap-3 mb-4">
    <input type="text" placeholder="Search URL, description, or ID…" bind:value={urlSearch} class="input flex-1" />
    <div class="flex gap-1">
      {#each healthFilters as f}
        <button
          onclick={() => handleHealthFilterChange(f.value)}
          aria-pressed={healthFilter === f.value}
          class="px-3 py-1.5 text-xs rounded-md mono transition-colors {healthFilter === f.value ? 'bg-beacon text-[#1a1204] font-semibold' : 'text-muted border border-line hover:text-text hover:bg-white/5'}"
        >
          {f.label}
        </button>
      {/each}
    </div>
  </div>

  {#if loading}
    <div class="panel overflow-hidden">
      <div class="animate-pulse">
        {#each Array(5) as _}
          <div class="row-line h-14 bg-white/[0.015]"></div>
        {/each}
      </div>
    </div>
  {:else if error}
    <div class="panel p-4 mb-6" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
      <p class="text-sm" style="color:var(--color-bad)">{error}</p>
    </div>
  {:else if webhooks.length === 0 && healthFilter === null && !urlSearch.trim()}
    <div class="panel ticked p-8">
      <div class="text-center mb-8">
        <p class="eyebrow mb-2">Signal path</p>
        <h3 class="text-xl mb-2">How Sparrow works</h3>
        <p class="text-sm text-muted max-w-lg mx-auto">Register events and webhooks, push events, and Sparrow fans out deliveries with retries and health tracking.</p>
      </div>

      <div class="max-w-3xl mx-auto mb-8">
        <div class="panel-2 p-5">
          <div class="hidden sm:flex items-center justify-center gap-0">
            {#each [
              { label: 'Push Event', sub: '', d: 'M13 7l5 5m0 0l-5 5m5-5H6' },
              { label: 'Event Worker', sub: 'find subscriptions', d: 'M4 6h16M4 12h16M4 18h16' },
              { label: 'Fan Out', sub: 'create deliveries', d: 'M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4' },
              { label: 'Deliver', sub: 'HTTP POST + HMAC', d: 'M12 19l9 2-9-18-9 18 9-2zm0 0v-8' },
            ] as step, i}
              {#if i > 0}<div class="w-8 border-t border-dashed border-line shrink-0"></div>{/if}
              <div class="flex flex-col items-center text-center min-w-0">
                <div class="w-10 h-10 rounded-lg bg-panel border border-line text-beacon flex items-center justify-center mb-1.5">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d={step.d} /></svg>
                </div>
                <span class="text-xs font-semibold text-text">{step.label}</span>
                {#if step.sub}<span class="text-[10px] text-faint mono">{step.sub}</span>{/if}
              </div>
            {/each}
            <div class="w-8 border-t border-dashed border-line shrink-0"></div>
            <div class="flex gap-1.5">
              <div class="flex flex-col items-center">
                <div class="w-10 h-10 rounded-lg flex items-center justify-center mb-1.5" style="color:var(--color-ok);border:1px solid color-mix(in srgb,var(--color-ok) 40%,transparent);background:color-mix(in srgb,var(--color-ok) 12%,transparent)">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
                </div>
                <span class="text-xs font-semibold" style="color:var(--color-ok)">Done</span>
              </div>
              <div class="flex flex-col items-center">
                <div class="w-10 h-10 rounded-lg flex items-center justify-center mb-1.5" style="color:var(--color-warn);border:1px solid color-mix(in srgb,var(--color-warn) 40%,transparent);background:color-mix(in srgb,var(--color-warn) 12%,transparent)">
                  <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" /></svg>
                </div>
                <span class="text-xs font-semibold" style="color:var(--color-warn)">Retry</span>
              </div>
            </div>
          </div>

          <div class="sm:hidden flex flex-col gap-2">
            {#each ['Push Event','Event Worker · find subscriptions','Fan Out · create deliveries','Deliver · HTTP POST + HMAC','Done / Retry with backoff'] as label}
              <div class="flex items-center gap-3">
                <span class="w-1.5 h-1.5 rounded-full bg-beacon shrink-0"></span>
                <span class="text-xs text-text">{label}</span>
              </div>
            {/each}
          </div>
        </div>
      </div>

      <div class="max-w-2xl mx-auto mb-8">
        <p class="eyebrow mb-4">Get started in 3 steps</p>
        {#each [
          { n: 1, title: 'Create an event', body: 'Define what happened in your system.', href: '/events', cta: 'Go to Events', last: false },
          { n: 2, title: 'Register a webhook', body: 'Add your endpoint URL and pick the events it should receive. Sparrow auto-creates the subscriptions.', href: '/webhooks/register', cta: 'Register a webhook', last: false },
          { n: 3, title: 'Push an event', body: 'Send a payload and watch Sparrow deliver it. Check the result in Deliveries.', href: '/events/push', cta: 'Push an event', last: true },
        ] as step}
          <div class="flex gap-4">
            <div class="flex flex-col items-center">
              <span class="grid place-items-center w-8 h-8 rounded-full border border-beacon/60 text-beacon mono text-sm font-semibold shrink-0">{step.n}</span>
              {#if !step.last}<div class="w-px flex-1 bg-line mt-1"></div>{/if}
            </div>
            <div class="{step.last ? 'pb-2' : 'pb-6'}">
              <h4 class="text-sm font-semibold text-text mb-1">{step.title}</h4>
              <p class="text-sm text-muted mb-2">{step.body}</p>
              <a href={step.href} class="inline-flex items-center gap-1 text-xs link-beacon">{step.cta} <span class="text-[10px]">&rarr;</span></a>
            </div>
          </div>
        {/each}
      </div>

      <div class="max-w-2xl mx-auto border-t border-line pt-6 mb-6">
        <p class="eyebrow mb-3">Then customize</p>
        <div class="flex flex-wrap gap-x-6 gap-y-2 text-sm text-muted">
          <span>Payload transforms on subscriptions</span>
          <span>Retry policies and timeouts</span>
          <span>HMAC signature verification</span>
          <span>Health monitoring</span>
        </div>
      </div>

      <div class="flex flex-col sm:flex-row items-center justify-center gap-3">
        <a href="/events" class="btn btn-beacon">Start with Step 1</a>
        <a href="/docs" class="btn btn-ghost">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253" /></svg>
          Read the API docs
        </a>
      </div>
    </div>
  {:else if filteredWebhooks.length === 0}
    <div class="panel">
      <EmptyState icon="filter_alt" title="No webhooks match" description={healthFilter ? `No ${healthFilter} webhooks in this view.` : 'Try a different search or namespace.'}>
        {#snippet action()}
          {#if healthFilter !== null}
            <button class="btn btn-ghost" onclick={() => handleHealthFilterChange(null)}>Clear filter</button>
          {/if}
        {/snippet}
      </EmptyState>
    </div>
  {:else}
    <div class="panel overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-left">
          <thead>
            <tr class="border-b border-line">
              <th class="th">Endpoint</th>
              <th class="th hidden sm:table-cell">Namespace</th>
              <th class="th">Health</th>
              <th class="th">Status</th>
              <th class="th"></th>
            </tr>
          </thead>
          <tbody>
            {#each filteredWebhooks as wh}
              <tr class="row-line row-hover transition cursor-pointer" onclick={() => goto(`/webhooks/${wh.webhook_id}`)}>
                <td class="td">
                  <p class="font-medium text-text truncate max-w-xs">{wh.description || wh.url}</p>
                  <div class="mt-0.5"><CopyableId id={wh.webhook_id} truncate={12} /></div>
                </td>
                <td class="td hidden sm:table-cell">
                  <span class="chip">{wh.namespace}</span>
                </td>
                <td class="td"><HealthBadge health={wh.health} /></td>
                <td class="td">
                  <button
                    onclick={(e) => toggleActive(wh, e)}
                    class="chip transition-colors {wh.active ? '!text-ok' : ''}"
                    style={wh.active ? 'border-color:color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 12%,var(--color-panel-2))' : ''}
                  >
                    <span class="w-1.5 h-1.5 rounded-full" style="background:var(--color-{wh.active ? 'ok' : 'idle'})"></span>
                    {wh.active ? 'Active' : 'Paused'}
                  </button>
                </td>
                <td class="td text-right">
                  <button onclick={(e) => promptUnregister(wh, e)} class="text-xs mono transition-colors" style="color:var(--color-faint)" onmouseenter={(e)=>e.currentTarget.style.color='var(--color-bad)'} onmouseleave={(e)=>e.currentTarget.style.color='var(--color-faint)'}>Unregister</button>
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
