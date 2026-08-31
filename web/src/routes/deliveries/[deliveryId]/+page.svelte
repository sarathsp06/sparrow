<script lang="ts">
  import { page } from '$app/state';
  import { api, unwrap } from '$lib/services';
  import { getCategoryDisplay, formatAPIError } from '$lib/utils';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import CopyableId from '$lib/components/CopyableId.svelte';
  import { onMount } from 'svelte';
  import type { components } from '$lib/api-types';

  type DeliveryItem = components["schemas"]["DeliveryItem"];

  let delivery: DeliveryItem | undefined = $state();
  let loading = $state(true);
  let error = $state('');

  const deliveryId = page.params.deliveryId ?? '';

  onMount(async () => {
    try {
      delivery = unwrap(await api.GET('/v1/deliveries/{delivery_id}', {
        params: { path: { delivery_id: deliveryId } },
      }));
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to load delivery details');
    } finally {
      loading = false;
    }
  });

  function formatResponseCode(code: number): string {
    if (!code) return 'N/A';
    return String(code);
  }

  function formatResponseBody(body: string | undefined): string {
    if (!body) return 'No response body';
    try {
      return JSON.stringify(JSON.parse(body), null, 2);
    } catch {
      return body;
    }
  }

  function formatTimestamp(timestamp: string | null | undefined): string {
    if (!timestamp) return 'N/A';
    const d = new Date(timestamp);
    return isNaN(d.getTime()) ? 'N/A' : d.toLocaleString();
  }
</script>

<svelte:head>
  <title>Delivery {deliveryId.substring(0, 8)}… | Sparrow</title>
</svelte:head>

<main class="mx-auto max-w-4xl px-4 sm:px-6 py-8">
    <nav class="flex items-center gap-2 text-sm text-muted mb-6">
        <a class="link" href="/deliveries">Deliveries</a>
        <span class="text-faint">/</span>
        <span class="text-text mono">{deliveryId.substring(0, 8)}…</span>
    </nav>

    {#if loading}
        <div class="animate-pulse space-y-4">
            <div class="h-8 bg-white/5 rounded w-64"></div>
            <div class="panel h-40 bg-white/[0.03]"></div>
        </div>
    {:else if error}
        <div class="panel p-4" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
            <p class="text-sm" style="color:var(--color-bad)">{error}</p>
        </div>
    {:else if delivery}
        <div class="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 mb-6">
            <div>
                <p class="eyebrow mb-1.5">Traffic / Delivery</p>
                <h1 class="text-2xl">Delivery details</h1>
                <div class="mt-1.5"><CopyableId id={delivery.delivery_id} truncate={0} /></div>
            </div>
            <StatusBadge status={delivery.status} />
        </div>

        <div class="panel mb-6">
            <div class="grid grid-cols-1 sm:grid-cols-2">
                <div class="px-6 py-4 row-line sm:border-t-0 sm:border-r border-line">
                    <p class="eyebrow mb-1.5">Webhook</p>
                    <CopyableId id={delivery.webhook_id} href="/webhooks/{delivery.webhook_id}" truncate={0} />
                </div>
                <div class="px-6 py-4 row-line sm:border-t-0 border-line">
                    <p class="eyebrow mb-1.5">Event</p>
                    <CopyableId id={delivery.event_id} truncate={0} />
                </div>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 border-t border-line">
                <div class="px-6 py-4 row-line sm:border-t-0 sm:border-r border-line">
                    <p class="eyebrow mb-1.5">Response code</p>
                    <span class="text-xl font-semibold mono tnum" style="color:{(delivery.response_code ?? 0) >= 200 && (delivery.response_code ?? 0) < 300 ? 'var(--color-ok)' : (delivery.response_code ?? 0) >= 400 ? 'var(--color-bad)' : 'var(--color-muted)'}">
                        {formatResponseCode(delivery.response_code ?? 0)}
                    </span>
                </div>
                <div class="px-6 py-4 row-line sm:border-t-0 border-line">
                    <p class="eyebrow mb-1.5">Attempts</p>
                    <span class="text-xl font-semibold tnum text-text">{delivery.attempt_count} / {delivery.max_attempts}</span>
                </div>
            </div>
            <div class="grid grid-cols-1 sm:grid-cols-2 border-t border-line">
                <div class="px-6 py-4 row-line sm:border-t-0 sm:border-r border-line">
                    <p class="eyebrow mb-1.5">Created at</p>
                    <span class="mono tnum text-sm text-muted">{formatTimestamp(delivery.created_at)}</span>
                </div>
                <div class="px-6 py-4 row-line sm:border-t-0 border-line">
                    <p class="eyebrow mb-1.5">Last attempted</p>
                    <span class="mono tnum text-sm text-muted">{formatTimestamp(delivery.last_attempted_at)}</span>
                </div>
            </div>
            {#if delivery.error_category && delivery.error_category !== 'success'}
                <div class="px-6 py-4 border-t border-line">
                    <p class="eyebrow mb-1.5">Error</p>
                    <p class="text-sm" style="color:var(--color-bad)">{getCategoryDisplay(delivery.error_category)}: {delivery.error_message || 'No details'}</p>
                </div>
            {/if}
            <div class="px-6 py-4 border-t border-line">
                <p class="eyebrow mb-2">Response body</p>
                <pre class="panel-2 mono text-xs p-4 overflow-auto max-h-64 text-text">{formatResponseBody(delivery.response_body)}</pre>
            </div>
        </div>
    {/if}
</main>
