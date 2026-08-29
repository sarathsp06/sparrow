<script lang="ts">
  import { page } from '$app/state';
  import { api, unwrap } from '$lib/services';
  import { getCategoryBadge, formatAPIError } from '$lib/utils';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import CopyableId from '$lib/components/CopyableId.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import favicon from '$lib/assets/favicon.svg';
  import { onMount } from 'svelte';
  import type { components } from '$lib/api-types';

  type EventOccurrenceItem = components["schemas"]["EventOccurrenceItem"];
  type DeliveryItem = components["schemas"]["DeliveryItem"];
  type AttemptItem = components["schemas"]["AttemptItem"];

  let event: EventOccurrenceItem | undefined = $state();
  let labels: Record<string, string> = $state({});
  let loading = $state(true);
  let error = $state('');

  // Deliveries
  let deliveries: DeliveryItem[] = $state([]);
  let deliveriesLoading = $state(true);
  let deliveriesError = $state('');
  let currentPage = $state(1);
  let totalCount = $state(0);
  let pageSize = $state(25);
  let totalPages = $derived(Math.max(1, Math.ceil(totalCount / pageSize)));

  // Expanded delivery attempt rows
  let expandedDeliveries: Set<string> = $state(new Set());
  let attemptsByDelivery: Map<string, AttemptItem[]> = $state(new Map());
  let loadingAttempts: Set<string> = $state(new Set());

  // Retry
  let retryingDeliveries: Set<string> = $state(new Set());
  let repushing = $state(false);

  const eventId = page.params.eventId ?? '';
  let namespace = $state('');

  onMount(async () => {
    await Promise.all([fetchEvent(), fetchDeliveries()]);
  });

  async function fetchEvent() {
    loading = true;
    try {
      event = unwrap(await api.GET('/v1/events/{event_id}', {
        params: { path: { event_id: eventId } },
      }));
      labels = event.labels ?? {};
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to load event record');
    } finally {
      loading = false;
    }
  }

  async function fetchDeliveries() {
    deliveriesLoading = true;
    deliveriesError = '';
    try {
      const ns = event?.namespace || namespace || 'default';
      const offset = (currentPage - 1) * pageSize;
      const res = unwrap(await api.GET('/v1/namespaces/{namespace}/deliveries', {
        params: { path: { namespace: ns }, query: { event_id: eventId, limit: pageSize, offset } },
      }));
      deliveries = res.items || [];
      totalCount = res.pagination?.total_count ?? 0;
    } catch (e: any) {
      deliveriesError = formatAPIError(e, 'Failed to load deliveries');
    } finally {
      deliveriesLoading = false;
    }
  }

  function handlePageChange(newPage: number) {
    currentPage = newPage;
    fetchDeliveries();
  }

  function formatTimestamp(timestamp: string | null | undefined): string {
    if (!timestamp) return 'N/A';
    const d = new Date(timestamp);
    return isNaN(d.getTime()) ? 'N/A' : d.toLocaleString();
  }

  function formatPayload(payload: any): string {
    if (!payload) return 'N/A';
    try {
      return JSON.stringify(payload, null, 2);
    } catch {
      return String(payload);
    }
  }

  async function retryDelivery(deliveryId: string) {
    retryingDeliveries.add(deliveryId);
    retryingDeliveries = new Set(retryingDeliveries);
    try {
      const ns = event?.namespace || namespace || 'default';
      unwrap(await api.POST('/v1/namespaces/{namespace}/deliveries/{delivery_id}:retry', {
        params: { path: { namespace: ns, delivery_id: deliveryId } },
      }));
      await fetchDeliveries();
    } catch (e: any) {
      console.error('Failed to retry delivery:', e);
    } finally {
      retryingDeliveries.delete(deliveryId);
      retryingDeliveries = new Set(retryingDeliveries);
    }
  }

  async function rePushEvent() {
    repushing = true;
    try {
      const res = unwrap(await api.POST('/v1/events/{event_id}:repush', {
        params: { path: { event_id: eventId } },
      }));
      if (res.event_id) {
        window.location.href = `/events/instances/${res.event_id}`;
      }
    } catch (e: any) {
      console.error('Failed to re-push event:', e);
    } finally {
      repushing = false;
    }
  }

  async function toggleDeliveryAttempts(deliveryId: string) {
    if (expandedDeliveries.has(deliveryId)) {
      expandedDeliveries.delete(deliveryId);
      expandedDeliveries = new Set(expandedDeliveries);
    } else {
      expandedDeliveries.add(deliveryId);
      expandedDeliveries = new Set(expandedDeliveries);
      if (!attemptsByDelivery.has(deliveryId)) {
        await fetchAttempts(deliveryId);
      }
    }
  }

  async function fetchAttempts(deliveryId: string) {
    loadingAttempts.add(deliveryId);
    loadingAttempts = new Set(loadingAttempts);
    try {
      const ns = event?.namespace || namespace || 'default';
      const res = unwrap(await api.GET('/v1/namespaces/{namespace}/deliveries/{delivery_id}/attempts', {
        params: { path: { namespace: ns, delivery_id: deliveryId } },
      }));
      attemptsByDelivery.set(deliveryId, res.items || []);
      attemptsByDelivery = new Map(attemptsByDelivery);
    } catch (e: any) {
      console.error('Failed to fetch attempts:', e);
    } finally {
      loadingAttempts.delete(deliveryId);
      loadingAttempts = new Set(loadingAttempts);
    }
  }
</script>

<svelte:head>
  <title>Event {eventId.substring(0, 8)}... | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    <nav class="flex items-center text-sm text-gray-500 mb-6">
      <a href="/events" class="hover:text-gray-900 transition">Events</a>
      <span class="mx-2">/</span>
      {#if event}
        <a href={`/events/${encodeURIComponent(event.event)}/reports`} class="hover:text-gray-900 transition">
          {event.event}
        </a>
        <span class="mx-2">/</span>
      {/if}
      <span class="text-gray-900 font-medium">Event {eventId.substring(0, 8)}...</span>
    </nav>

    {#if loading}
      <div class="max-w-3xl">
        <div class="animate-pulse mb-6">
          <div class="h-7 bg-gray-200 rounded w-48 mb-2"></div>
          <div class="h-4 bg-gray-100 rounded w-64"></div>
        </div>
        <div class="bg-white rounded-lg border border-gray-200 p-6">
          <div class="animate-pulse space-y-3">
            <div class="h-4 bg-gray-100 rounded w-full"></div>
            <div class="h-4 bg-gray-100 rounded w-3/4"></div>
            <div class="h-4 bg-gray-100 rounded w-1/2"></div>
          </div>
        </div>
      </div>
    {:else if error}
      <div class="bg-red-50 border border-red-200 rounded-lg p-4 max-w-3xl">
        <div class="flex items-start gap-3">
          <p class="text-sm text-red-700">{error}</p>
        </div>
      </div>
    {:else if event}
      <div class="max-w-3xl">
        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
          <div>
            <h1 class="text-2xl font-bold text-gray-900">Event Record</h1>
            <div class="mt-0.5"><CopyableId id={event.event_id} truncate={0} /></div>
          </div>
          <button
            onclick={rePushEvent}
            disabled={repushing}
            class="inline-flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 disabled:opacity-50 transition"
          >
            {#if repushing}
              <img src={favicon} alt="" class="w-3.5 h-3.5 animate-spin" />
              Re-pushing...
            {:else}
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              Re-push
            {/if}
          </button>
        </div>

        <div class="bg-white rounded-lg border border-gray-200 divide-y divide-gray-100 mb-8">
          <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-gray-100">
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Event Type</p>
              <a href={`/events/${encodeURIComponent(event.event)}/reports`} class="text-sm font-medium text-blue-600 hover:text-blue-800 hover:underline transition">
                {event.event}
              </a>
            </div>
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Namespace</p>
              <span class="inline-flex items-center px-2 py-0.5 text-xs font-medium bg-gray-100 text-gray-700 rounded">{event.namespace}</span>
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-gray-100">
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Schema Validation</p>
              {#if event.schema_valid}
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-50 text-green-700 border border-green-200">Valid</span>
              {:else}
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-red-50 text-red-700 border border-red-200">Invalid</span>
              {/if}
            </div>
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Created At</p>
              <span class="text-sm text-gray-900">{formatTimestamp(event.created_at)}</span>
            </div>
          </div>

          <div class="px-6 py-4">
            <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Delivery Statistics</p>
            <div class="flex flex-wrap items-center gap-4 text-sm">
              <div class="flex items-center gap-1.5">
                <span class="text-gray-500">Webhooks:</span>
                <span class="font-medium text-gray-900">{event.webhook_count}</span>
              </div>
              <div class="flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full bg-green-500"></span>
                <span class="text-green-700 font-medium">{event.successful_deliveries} success</span>
              </div>
              <div class="flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full bg-red-500"></span>
                <span class="text-red-700 font-medium">{event.failed_deliveries} failed</span>
              </div>
              <div class="flex items-center gap-1.5">
                <span class="w-2 h-2 rounded-full bg-yellow-500"></span>
                <span class="text-yellow-700 font-medium">{event.pending_deliveries} pending</span>
              </div>
            </div>
          </div>

          {#if event.metadata && Object.keys(event.metadata).length > 0}
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Metadata</p>
              <div class="flex flex-wrap gap-2">
                {#each Object.entries(event.metadata) as [key, value]}
                  <span class="inline-flex items-center px-2 py-1 text-xs font-mono bg-gray-50 text-gray-700 rounded border border-gray-200">
                    <span class="text-gray-500">{key}:</span>
                    <span class="ml-1 font-medium">{value}</span>
                  </span>
                {/each}
              </div>
            </div>
          {/if}

          {#if Object.keys(labels).length > 0}
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Labels</p>
              <div class="flex flex-wrap gap-2">
                {#each Object.entries(labels) as [key, value]}
                  <span class="inline-flex items-center px-2 py-1 text-xs font-mono bg-blue-50 text-blue-700 rounded border border-blue-200">
                    <span class="text-blue-500">{key}:</span>
                    <span class="ml-1 font-medium">{value}</span>
                  </span>
                {/each}
              </div>
            </div>
          {/if}

          <div class="px-6 py-4">
            <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Payload</p>
            <pre class="text-xs bg-gray-50 border border-gray-200 rounded-lg p-4 overflow-auto max-h-64 text-gray-800 font-mono">{formatPayload(event.payload)}</pre>
          </div>
        </div>

        <div>
          <div class="flex items-center justify-between mb-4">
            <h2 class="text-lg font-bold text-gray-900">Deliveries</h2>
            {#if totalCount > 0}
              <span class="text-sm text-gray-500">{totalCount} total</span>
            {/if}
          </div>

          {#if deliveriesLoading}
            <div class="bg-white rounded-lg border border-gray-200 p-8">
              <div class="flex items-center justify-center">
                <img src={favicon} alt="Loading" class="w-5 h-5 animate-spin mr-2" />
                <span class="text-sm text-gray-500">Loading deliveries...</span>
              </div>
            </div>
          {:else if deliveriesError}
            <div class="bg-red-50 border border-red-200 rounded-lg p-4">
              <p class="text-sm text-red-800">{deliveriesError}</p>
            </div>
          {:else if deliveries.length === 0}
            <EmptyState icon="send" title="No deliveries" description="No deliveries have been created for this event yet." />
          {:else}
            <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
              <div class="overflow-x-auto">
                <table class="w-full text-sm text-left">
                  <thead>
                    <tr class="border-b border-gray-200 bg-gray-50/50">
                      <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Delivery ID</th>
                      <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Webhook</th>
                      <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
                      <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Response</th>
                      <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden md:table-cell">Attempts</th>
                      <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden lg:table-cell">Last Attempt</th>
                      <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Actions</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-100">
                    {#each deliveries as delivery}
                      <tr class="hover:bg-gray-50 transition {expandedDeliveries.has(delivery.delivery_id) ? 'bg-blue-50/30' : ''}">
                        <td class="px-4 py-3">
                          <CopyableId id={delivery.delivery_id} href="/deliveries/{delivery.delivery_id}" truncate={12} />
                          <span class="block sm:hidden mt-0.5"><CopyableId id={delivery.webhook_id} href="/webhooks/{delivery.webhook_id}" truncate={12} /></span>
                        </td>
                        <td class="px-4 py-3 hidden sm:table-cell">
                          <CopyableId id={delivery.webhook_id} href="/webhooks/{delivery.webhook_id}" truncate={12} />
                        </td>
                        <td class="px-4 py-3">
                          <div class="flex items-center gap-1.5">
                            <StatusBadge status={delivery.status} />
                            {#if delivery.error_category && delivery.error_category !== 'success'}
                              {@const badge = getCategoryBadge(delivery.error_category)}
                              <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                            {/if}
                          </div>
                        </td>
                        <td class="px-4 py-3 hidden sm:table-cell">
                          <span class="font-mono text-xs {(delivery.response_code ?? 0) >= 200 && (delivery.response_code ?? 0) < 300 ? 'text-green-600' : (delivery.response_code ?? 0) >= 400 ? 'text-red-600' : 'text-gray-500'}">
                            {delivery.response_code || '—'}
                          </span>
                        </td>
                        <td class="px-4 py-3 hidden md:table-cell">
                          <button
                            onclick={() => toggleDeliveryAttempts(delivery.delivery_id)}
                            class="text-xs text-blue-600 hover:text-blue-800 font-medium hover:underline transition"
                          >
                            {delivery.attempt_count}/{delivery.max_attempts}
                            <span class="text-[10px] ml-0.5">{expandedDeliveries.has(delivery.delivery_id) ? '▲' : '▼'}</span>
                          </button>
                        </td>
                        <td class="px-4 py-3 text-xs text-gray-500 hidden lg:table-cell">{formatTimestamp(delivery.last_attempted_at)}</td>
                        <td class="px-4 py-3">
                          <div class="flex items-center gap-2">
                            {#if delivery.status === 'failed' || delivery.status === 'expired' || (delivery.error_category && delivery.error_category !== 'success')}
                              <button
                                onclick={() => retryDelivery(delivery.delivery_id)}
                                disabled={retryingDeliveries.has(delivery.delivery_id)}
                                class="inline-flex items-center px-2 py-1 text-xs font-medium text-white bg-gray-900 rounded-md hover:bg-gray-800 disabled:opacity-50 transition"
                              >
                                {retryingDeliveries.has(delivery.delivery_id) ? 'Retrying...' : 'Retry'}
                              </button>
                            {/if}
                            <a
                              href="/deliveries/{delivery.delivery_id}"
                              class="inline-flex items-center px-2 py-1 text-xs font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 transition"
                            >
                              Details
                            </a>
                          </div>
                        </td>
                      </tr>

                      {#if expandedDeliveries.has(delivery.delivery_id)}
                        <tr class="bg-blue-50/20">
                          <td colspan="7" class="px-4 py-3">
                            {#if loadingAttempts.has(delivery.delivery_id)}
                              <div class="flex items-center justify-center py-3">
                                <img src={favicon} alt="Loading" class="w-3 h-3 animate-spin mr-2" />
                                <span class="text-xs text-gray-500">Loading attempt history...</span>
                              </div>
                            {:else if attemptsByDelivery.has(delivery.delivery_id) && (attemptsByDelivery.get(delivery.delivery_id)?.length ?? 0) > 0}
                              <div class="ml-4 border-l-2 border-blue-200 pl-3">
                                <p class="text-[10px] font-semibold text-gray-400 uppercase tracking-wide mb-1.5">Attempt History</p>
                                <div class="space-y-1.5">
                                  {#each attemptsByDelivery.get(delivery.delivery_id) ?? [] as attempt, i}
                                    <div class="flex items-start gap-3 py-1.5 px-2 rounded {attempt.success ? 'bg-green-50/50' : 'bg-red-50/50'}">
                                      <div class="flex-shrink-0 w-5 h-5 flex items-center justify-center rounded-full text-[10px] font-bold {attempt.success ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}">
                                        {i + 1}
                                      </div>
                                      <div class="flex-1 min-w-0">
                                        <div class="flex flex-wrap items-center gap-2 text-[11px]">
                                          <span class="font-medium {attempt.success ? 'text-green-700' : 'text-red-700'}">
                                            {attempt.success ? 'Success' : 'Failed'}
                                          </span>
                                          {#if attempt.response_code > 0}
                                            <span class="font-mono px-1 py-0.5 rounded text-[10px] {attempt.response_code >= 200 && attempt.response_code < 300 ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}">
                                              {attempt.response_code}
                                            </span>
                                          {/if}
                                          {#if attempt.error_category && attempt.error_category !== 'success'}
                                            {@const badge = getCategoryBadge(attempt.error_category)}
                                            <span class="inline-flex items-center px-1 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                                          {/if}
                                          <span class="text-gray-400 font-mono text-[10px]">{attempt.response_time}ms</span>
                                          <span class="text-gray-400 text-[10px]">{formatTimestamp(attempt.timestamp)}</span>
                                        </div>
                                        {#if attempt.error_message}
                                          <p class="text-[10px] text-red-600 mt-0.5 truncate" title={attempt.error_message}>{attempt.error_message}</p>
                                        {/if}
                                      </div>
                                    </div>
                                  {/each}
                                </div>
                              </div>
                            {:else}
                              <p class="text-[11px] text-gray-400 ml-4">No attempt history recorded yet.</p>
                            {/if}
                          </td>
                        </tr>
                      {/if}
                    {/each}
                  </tbody>
                </table>
              </div>
            </div>

            {#if totalPages > 1}
              <div class="mt-4">
                <Pagination {currentPage} {totalPages} {totalCount} {pageSize} onPageChange={handlePageChange} />
              </div>
            {/if}
          {/if}
        </div>

        <div class="mt-6">
          <a
            href={`/events/${encodeURIComponent(event.event)}/reports`}
            class="inline-flex items-center gap-1 text-sm text-gray-600 hover:text-gray-900 transition"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
            Back to {event.event} reports
          </a>
        </div>
      </div>
    {/if}
  </main>
</div>
