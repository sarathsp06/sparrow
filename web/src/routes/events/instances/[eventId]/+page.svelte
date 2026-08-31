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
  <title>Event {eventId.substring(0, 8)}… | Sparrow</title>
</svelte:head>

<main class="mx-auto max-w-4xl px-4 sm:px-6 py-8">
  <nav class="flex items-center gap-2 text-sm text-muted mb-6">
    <a href="/events" class="link">Events</a>
    {#if event}
      <span class="text-faint">/</span>
      <a href={`/events/${encodeURIComponent(event.event)}/reports`} class="link">{event.event}</a>
    {/if}
    <span class="text-faint">/</span>
    <span class="text-text mono tnum">Event {eventId.substring(0, 8)}…</span>
  </nav>

  {#if loading}
    <div class="animate-pulse mb-6">
      <div class="h-7 bg-white/5 rounded w-48 mb-2"></div>
      <div class="h-4 bg-white/[0.03] rounded w-64"></div>
    </div>
    <div class="panel p-6">
      <div class="animate-pulse space-y-3">
        <div class="h-4 bg-white/[0.03] rounded w-full"></div>
        <div class="h-4 bg-white/[0.03] rounded w-3/4"></div>
        <div class="h-4 bg-white/[0.03] rounded w-1/2"></div>
      </div>
    </div>
  {:else if error}
    <div class="panel p-4" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
      <p class="text-sm" style="color:var(--color-bad)">{error}</p>
    </div>
  {:else if event}
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
      <div>
        <p class="eyebrow mb-1.5">Events / Record</p>
        <h1 class="text-2xl">Event Record</h1>
        <div class="mt-1"><CopyableId id={event.event_id} truncate={0} /></div>
      </div>
      <button
        onclick={rePushEvent}
        disabled={repushing}
        class="btn btn-ghost !px-3 !py-1.5"
      >
        {#if repushing}
          <img src={favicon} alt="" aria-hidden="true" class="w-3.5 h-3.5 animate-spin" />
          Re-pushing…
        {:else}
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          Re-push
        {/if}
      </button>
    </div>

    <div class="panel divide-y divide-line mb-8">
      <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-line">
        <div class="px-6 py-4">
          <p class="eyebrow mb-1">Event Type</p>
          <a href={`/events/${encodeURIComponent(event.event)}/reports`} class="link-beacon text-sm mono">{event.event}</a>
        </div>
        <div class="px-6 py-4">
          <p class="eyebrow mb-1">Namespace</p>
          <span class="chip">{event.namespace}</span>
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-line">
        <div class="px-6 py-4">
          <p class="eyebrow mb-1.5">Schema Validation</p>
          {#if event.schema_valid}
            <span class="chip" style="color:var(--color-ok);border-color:color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 12%,var(--color-panel-2))">Valid</span>
          {:else}
            <span class="chip" style="color:var(--color-bad);border-color:color-mix(in srgb,var(--color-bad) 35%,transparent);background:color-mix(in srgb,var(--color-bad) 12%,var(--color-panel-2))">Invalid</span>
          {/if}
        </div>
        <div class="px-6 py-4">
          <p class="eyebrow mb-1">Created At</p>
          <span class="text-sm mono tnum text-text">{formatTimestamp(event.created_at)}</span>
        </div>
      </div>

      <div class="px-6 py-4">
        <p class="eyebrow mb-2">Delivery Statistics</p>
        <div class="flex flex-wrap items-center gap-4 text-sm">
          <div class="flex items-center gap-1.5">
            <span class="text-muted">Webhooks:</span>
            <span class="font-medium tnum text-text">{event.webhook_count}</span>
          </div>
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full" style="background:var(--color-ok)"></span>
            <span class="font-medium tnum" style="color:var(--color-ok)">{event.successful_deliveries} success</span>
          </div>
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full" style="background:var(--color-bad)"></span>
            <span class="font-medium tnum" style="color:var(--color-bad)">{event.failed_deliveries} failed</span>
          </div>
          <div class="flex items-center gap-1.5">
            <span class="w-2 h-2 rounded-full" style="background:var(--color-warn)"></span>
            <span class="font-medium tnum" style="color:var(--color-warn)">{event.pending_deliveries} pending</span>
          </div>
        </div>
      </div>

      {#if event.metadata && Object.keys(event.metadata).length > 0}
        <div class="px-6 py-4">
          <p class="eyebrow mb-2">Metadata</p>
          <div class="flex flex-wrap gap-2">
            {#each Object.entries(event.metadata) as [key, value]}
              <span class="chip">
                <span class="text-faint">{key}:</span>
                <span class="text-text">{value}</span>
              </span>
            {/each}
          </div>
        </div>
      {/if}

      {#if Object.keys(labels).length > 0}
        <div class="px-6 py-4">
          <p class="eyebrow mb-2">Labels</p>
          <div class="flex flex-wrap gap-2">
            {#each Object.entries(labels) as [key, value]}
              <span class="chip">
                <span class="text-faint">{key}:</span>
                <span class="text-text">{value}</span>
              </span>
            {/each}
          </div>
        </div>
      {/if}

      <div class="px-6 py-4">
        <p class="eyebrow mb-2">Payload</p>
        <pre class="panel-2 mono text-xs p-4 overflow-auto max-h-64 text-text">{formatPayload(event.payload)}</pre>
      </div>
    </div>

    <div>
      <div class="flex items-center justify-between mb-4">
        <h2 class="text-lg">Deliveries</h2>
        {#if totalCount > 0}
          <span class="text-sm text-muted tnum">{totalCount} total</span>
        {/if}
      </div>

      {#if deliveriesLoading}
        <div class="panel p-8">
          <div class="flex items-center justify-center">
            <img src={favicon} alt="" aria-hidden="true" class="w-5 h-5 animate-spin mr-2" />
            <span class="text-sm text-muted">Loading deliveries…</span>
          </div>
        </div>
      {:else if deliveriesError}
        <div class="panel p-4" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
          <p class="text-sm" style="color:var(--color-bad)">{deliveriesError}</p>
        </div>
      {:else if deliveries.length === 0}
        <div class="panel">
          <EmptyState icon="send" title="No deliveries" description="No deliveries have been created for this event yet." />
        </div>
      {:else}
        <div class="panel overflow-hidden">
          <div class="overflow-x-auto">
            <table class="w-full text-left">
              <thead>
                <tr class="border-b border-line">
                  <th class="th">Delivery ID</th>
                  <th class="th hidden sm:table-cell">Webhook</th>
                  <th class="th">Status</th>
                  <th class="th hidden sm:table-cell">Response</th>
                  <th class="th hidden md:table-cell">Attempts</th>
                  <th class="th hidden lg:table-cell">Last Attempt</th>
                  <th class="th">Actions</th>
                </tr>
              </thead>
              <tbody>
                {#each deliveries as delivery}
                  <tr class="row-line row-hover transition {expandedDeliveries.has(delivery.delivery_id) ? 'bg-white/[0.03]' : ''}">
                    <td class="td">
                      <CopyableId id={delivery.delivery_id} href="/deliveries/{delivery.delivery_id}" truncate={12} />
                      <span class="block sm:hidden mt-0.5"><CopyableId id={delivery.webhook_id} href="/webhooks/{delivery.webhook_id}" truncate={12} /></span>
                    </td>
                    <td class="td hidden sm:table-cell">
                      <CopyableId id={delivery.webhook_id} href="/webhooks/{delivery.webhook_id}" truncate={12} />
                    </td>
                    <td class="td">
                      <div class="flex items-center gap-1.5">
                        <StatusBadge status={delivery.status} />
                        {#if delivery.error_category && delivery.error_category !== 'success'}
                          {@const badge = getCategoryBadge(delivery.error_category)}
                          <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                        {/if}
                      </div>
                    </td>
                    <td class="td hidden sm:table-cell">
                      <span class="mono tnum text-xs {(delivery.response_code ?? 0) >= 200 && (delivery.response_code ?? 0) < 300 ? 'text-ok' : (delivery.response_code ?? 0) >= 400 ? 'text-bad' : 'text-muted'}">
                        {delivery.response_code || '—'}
                      </span>
                    </td>
                    <td class="td hidden md:table-cell">
                      <button
                        onclick={() => toggleDeliveryAttempts(delivery.delivery_id)}
                        aria-expanded={expandedDeliveries.has(delivery.delivery_id)}
                        class="text-xs mono tnum link-beacon font-medium transition"
                      >
                        {delivery.attempt_count}/{delivery.max_attempts}
                        <span class="text-[10px] ml-0.5" aria-hidden="true">{expandedDeliveries.has(delivery.delivery_id) ? '▲' : '▼'}</span>
                      </button>
                    </td>
                    <td class="td text-xs mono tnum text-muted hidden lg:table-cell">{formatTimestamp(delivery.last_attempted_at)}</td>
                    <td class="td">
                      <div class="flex items-center gap-2">
                        {#if delivery.status === 'failed' || delivery.status === 'expired' || (delivery.error_category && delivery.error_category !== 'success')}
                          <button
                            onclick={() => retryDelivery(delivery.delivery_id)}
                            disabled={retryingDeliveries.has(delivery.delivery_id)}
                            class="btn btn-ghost !px-2 !py-1 text-xs"
                          >
                            {retryingDeliveries.has(delivery.delivery_id) ? 'Retrying…' : 'Retry'}
                          </button>
                        {/if}
                        <a href="/deliveries/{delivery.delivery_id}" class="btn btn-ghost !px-2 !py-1 text-xs">Details</a>
                      </div>
                    </td>
                  </tr>

                  {#if expandedDeliveries.has(delivery.delivery_id)}
                    <tr class="row-line bg-white/[0.015]">
                      <td colspan="7" class="px-4 py-3">
                        {#if loadingAttempts.has(delivery.delivery_id)}
                          <div class="flex items-center justify-center py-3">
                            <img src={favicon} alt="" aria-hidden="true" class="w-3 h-3 animate-spin mr-2" />
                            <span class="text-xs text-muted">Loading attempt history…</span>
                          </div>
                        {:else if attemptsByDelivery.has(delivery.delivery_id) && (attemptsByDelivery.get(delivery.delivery_id)?.length ?? 0) > 0}
                          <div class="ml-4 border-l-2 border-line-strong pl-3">
                            <p class="eyebrow mb-1.5">Attempt History</p>
                            <div class="space-y-1.5">
                              {#each attemptsByDelivery.get(delivery.delivery_id) ?? [] as attempt, i}
                                <div class="flex items-start gap-3 py-1.5 px-2 rounded" style="background:color-mix(in srgb,var(--color-{attempt.success ? 'ok' : 'bad'}) 8%,transparent)">
                                  <div class="flex-shrink-0 w-5 h-5 flex items-center justify-center rounded-full text-[10px] font-bold tnum" style="color:var(--color-{attempt.success ? 'ok' : 'bad'});background:color-mix(in srgb,var(--color-{attempt.success ? 'ok' : 'bad'}) 15%,transparent)">
                                    {i + 1}
                                  </div>
                                  <div class="flex-1 min-w-0">
                                    <div class="flex flex-wrap items-center gap-2 text-[11px]">
                                      <span class="font-medium" style="color:var(--color-{attempt.success ? 'ok' : 'bad'})">
                                        {attempt.success ? 'Success' : 'Failed'}
                                      </span>
                                      {#if attempt.response_code > 0}
                                        <span class="mono tnum px-1 py-0.5 rounded text-[10px]" style="color:var(--color-{attempt.response_code >= 200 && attempt.response_code < 300 ? 'ok' : 'bad'});background:color-mix(in srgb,var(--color-{attempt.response_code >= 200 && attempt.response_code < 300 ? 'ok' : 'bad'}) 15%,transparent)">
                                          {attempt.response_code}
                                        </span>
                                      {/if}
                                      {#if attempt.error_category && attempt.error_category !== 'success'}
                                        {@const badge = getCategoryBadge(attempt.error_category)}
                                        <span class="inline-flex items-center px-1 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                                      {/if}
                                      <span class="text-faint mono tnum text-[10px]">{attempt.response_time}ms</span>
                                      <span class="text-faint mono tnum text-[10px]">{formatTimestamp(attempt.timestamp)}</span>
                                    </div>
                                    {#if attempt.error_message}
                                      <p class="text-[10px] mt-0.5 truncate" style="color:var(--color-bad)" title={attempt.error_message}>{attempt.error_message}</p>
                                    {/if}
                                  </div>
                                </div>
                              {/each}
                            </div>
                          </div>
                        {:else}
                          <p class="text-[11px] text-faint ml-4">No attempt history recorded yet.</p>
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
      <a href={`/events/${encodeURIComponent(event.event)}/reports`} class="inline-flex items-center gap-1 text-sm link">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
        </svg>
        Back to {event.event} reports
      </a>
    </div>
  {/if}
</main>
