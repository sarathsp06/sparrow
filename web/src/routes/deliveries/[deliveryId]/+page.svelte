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
  <title>Delivery {deliveryId.substring(0, 8)}... | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <main class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    <nav class="flex items-center text-sm text-gray-500 mb-6">
      <a href="/deliveries" class="hover:text-gray-900 transition">Deliveries</a>
      <span class="mx-2">/</span>
      <span class="text-gray-900 font-medium">{deliveryId.substring(0, 8)}...</span>
    </nav>

    {#if loading}
      <div class="animate-pulse space-y-4">
        <div class="h-8 bg-gray-200 rounded w-64"></div>
        <div class="h-40 bg-gray-100 rounded"></div>
      </div>
    {:else if error}
      <div class="bg-red-50 border border-red-200 rounded-lg p-4">
        <p class="text-sm text-red-700">{error}</p>
      </div>
    {:else if delivery}
      <div class="mb-6 flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900">Delivery Details</h1>
          <div class="mt-1"><CopyableId id={delivery.delivery_id} truncate={0} /></div>
        </div>
        <StatusBadge status={delivery.status} />
      </div>

      <div class="bg-white rounded-lg border border-gray-200 divide-y divide-gray-100 mb-6">
        <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-gray-100">
          <div class="px-6 py-4">
            <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Webhook</p>
            <CopyableId id={delivery.webhook_id} href="/webhooks/{delivery.webhook_id}" truncate={0} />
          </div>
          <div class="px-6 py-4">
            <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Event</p>
            <CopyableId id={delivery.event_id} truncate={0} />
          </div>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-gray-100">
          <div class="px-6 py-4">
            <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Response Code</p>
            <span class="text-sm font-mono {(delivery.response_code ?? 0) >= 200 && (delivery.response_code ?? 0) < 300 ? 'text-green-600' : (delivery.response_code ?? 0) >= 400 ? 'text-red-600' : 'text-gray-500'}">
              {formatResponseCode(delivery.response_code ?? 0)}
            </span>
          </div>
          <div class="px-6 py-4">
            <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Attempts</p>
            <span class="text-sm text-gray-900">{delivery.attempt_count} / {delivery.max_attempts}</span>
          </div>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-gray-100">
          <div class="px-6 py-4">
            <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Created At</p>
            <span class="text-sm text-gray-900">{formatTimestamp(delivery.created_at)}</span>
          </div>
          <div class="px-6 py-4">
            <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Last Attempted</p>
            <span class="text-sm text-gray-900">{formatTimestamp(delivery.last_attempted_at)}</span>
          </div>
        </div>
        {#if delivery.error_category && delivery.error_category !== 'success'}
          <div class="px-6 py-4">
            <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Error</p>
            <p class="text-sm text-red-700">{getCategoryDisplay(delivery.error_category)}: {delivery.error_message || 'No details'}</p>
          </div>
        {/if}
        <div class="px-6 py-4">
          <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Response Body</p>
          <pre class="text-xs bg-gray-50 border border-gray-200 rounded-lg p-4 overflow-auto max-h-64 text-gray-800 font-mono">{formatResponseBody(delivery.response_body)}</pre>
        </div>
      </div>
    {/if}
  </main>
</div>
