<script lang="ts">
  import { page } from '$app/state';
  import { api, unwrap } from '#lib/services.js';
  import { formatAPIError, formatTimestamp } from '#lib/utils.js';
  import CopyableId from '#lib/components/CopyableId.svelte';
  import DeliveryTable from '#lib/components/DeliveryTable.svelte';
  import CursorPager from '#lib/components/CursorPager.svelte';
  import { CursorPages } from '#lib/cursor-pages.svelte.js';
  import EmptyState from '#lib/components/EmptyState.svelte';
  import favicon from '#lib/assets/favicon.svg';
  import { onMount } from 'svelte';
  import type { components } from '#lib/api-types.js';

  type EventOccurrenceItem = components["schemas"]["EventOccurrenceItem"];
  type DeliveryItem = components["schemas"]["DeliveryItem"];

  let event: EventOccurrenceItem | undefined = $state();
  let labels: Record<string, string> = $state({});
  let loading = $state(true);
  let error = $state('');

  // Deliveries
  let deliveries: DeliveryItem[] = $state([]);
  let deliveriesLoading = $state(true);
  let deliveriesError = $state('');
  let pageSize = $state(25);
  const pages = new CursorPages();

  // Re-push
  let repushing = $state(false);
  let repushError = $state('');
  // Sparrow's own sparrow.* events: the server refuses to push them.
  let systemEvent = $derived(!!event?.event.toLowerCase().startsWith('sparrow.'));

  const eventId = page.params.eventId ?? '';
  let consumer = $state('');

  onMount(async () => {
    // Deliveries are listed under the event's own consumer, so load it first.
    await fetchEvent();
    await fetchDeliveries();
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
      const ns = event?.consumer || consumer || 'default';
      const res = unwrap(await api.GET('/v1/consumers/{consumer}/deliveries', {
        params: { path: { consumer: ns }, query: { event_id: eventId, limit: pageSize, cursor: pages.cursor || undefined } },
      }));
      deliveries = res.items || [];
      pages.update(res.pagination);
    } catch (e: any) {
      deliveriesError = formatAPIError(e, 'Failed to load deliveries');
    } finally {
      deliveriesLoading = false;
    }
  }

  function formatPayload(payload: any): string {
    if (!payload) return 'N/A';
    try {
      return JSON.stringify(payload, null, 2);
    } catch {
      return String(payload);
    }
  }

  async function rePushEvent() {
    repushing = true;
    repushError = '';
    try {
      const res = unwrap(await api.POST('/v1/events/{event_id}:repush', {
        params: { path: { event_id: eventId } },
      }));
      if (res.event_id) {
        window.location.href = `/events/instances/${res.event_id}`;
      }
    } catch (e: any) {
      repushError = formatAPIError(e, 'Failed to re-push event');
    } finally {
      repushing = false;
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
        <div class="h-4 bg-black/5 rounded w-48 mb-2"></div>
        <div class="h-4 bg-black/[0.03] rounded w-64"></div>
    </div>
    <div class="panel p-6">
      <div class="animate-pulse space-y-3">
        <div class="h-4 bg-black/[0.03] rounded w-full"></div>
        <div class="h-4 bg-black/[0.03] rounded w-3/4"></div>
        <div class="h-4 bg-black/[0.03] rounded w-1/2"></div>
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
      <div class="flex flex-wrap items-center gap-2">
      <!-- Sparrow's own sparrow.* events can't be pushed, so the same actions
           show disabled with the reason instead of disappearing. -->
      {#if systemEvent}
        <button type="button" disabled class="btn btn-ghost !px-3 !py-1.5" title="Sparrow defines the schema of its own events">
          Use as schema sample
        </button>
      {:else}
        <a
          href={`/events/${encodeURIComponent(event.event)}/update?sample=${event.event_id}`}
          class="btn btn-ghost !px-3 !py-1.5"
          title="Generate the event type's schema from this payload"
        >
          Use as schema sample
        </a>
      {/if}
      <button
        onclick={rePushEvent}
        disabled={repushing || systemEvent}
        class="btn btn-ghost !px-3 !py-1.5"
        title={systemEvent ? "Only Sparrow emits its own events. To resend an alert, retry its delivery below." : undefined}
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
    </div>
    {#if repushError}<p class="text-xs mb-4" style="color:var(--color-bad)">{repushError}</p>{/if}

    <div class="panel divide-y divide-line mb-8">
      <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-line">
        <div class="px-6 py-4">
          <p class="eyebrow mb-1">Event Type</p>
          <a href={`/events/${encodeURIComponent(event.event)}/reports`} class="link-beacon text-sm mono">{event.event}</a>
        </div>
        <div class="px-6 py-4">
          <p class="eyebrow mb-1">Consumer</p>
          <span class="chip">{event.consumer}</span>
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
      {#if event.scheduled_at}
        <div class="px-6 py-4">
          <p class="eyebrow mb-1">Scheduled For</p>
          <span class="text-sm mono tnum text-text">{formatTimestamp(event.scheduled_at)}</span>
        </div>
      {/if}
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
          <DeliveryTable {deliveries} hide={['event', 'consumer']} onchange={fetchDeliveries} onerror={(m) => (deliveriesError = m)} />
        </div>

        <CursorPager {pages} shown={deliveries.length} itemLabel="deliveries" onchange={fetchDeliveries} />
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
