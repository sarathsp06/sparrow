<script lang="ts">
    import type { components } from '$lib/api-types';
    import { api, unwrap } from '$lib/services';
    import { getCategoryBadge, timeAgo } from '$lib/utils';
    import StatusBadge from './StatusBadge.svelte';
    import CopyableId from './CopyableId.svelte';
    import EmptyState from './EmptyState.svelte';
    import favicon from '$lib/assets/favicon.svg';

    type EventOccurrenceItem = components["schemas"]["EventOccurrenceItem"];
    type DeliveryItem = components["schemas"]["DeliveryItem"];
    type AttemptItem = components["schemas"]["AttemptItem"];

    interface Props {
        eventReports: EventOccurrenceItem[];
        loading?: boolean;
        error?: string;
        currentEventName?: string;
    }

    let { eventReports, loading = false, error = '', currentEventName = 'this event' }: Props = $props();

    // Expanded event rows state
    let expandedRows: Set<string> = $state(new Set());
    let deliveriesByEvent: Map<string, DeliveryItem[]> = $state(new Map());
    let loadingDeliveries: Set<string> = $state(new Set());
    let retryingDeliveries: Set<string> = $state(new Set());

    // Expanded delivery attempt rows state
    let expandedDeliveries: Set<string> = $state(new Set());
    let attemptsByDelivery: Map<string, AttemptItem[]> = $state(new Map());
    let loadingAttempts: Set<string> = $state(new Set());

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

    async function toggleRow(eventId: string, namespace: string) {
        if (expandedRows.has(eventId)) {
            expandedRows.delete(eventId);
            expandedRows = new Set(expandedRows);
        } else {
            expandedRows.add(eventId);
            expandedRows = new Set(expandedRows);

            if (!deliveriesByEvent.has(eventId)) {
                await fetchDeliveries(eventId, namespace);
            }
        }
    }

    async function fetchDeliveries(eventId: string, namespace: string) {
        loadingDeliveries.add(eventId);
        loadingDeliveries = new Set(loadingDeliveries);
        try {
            const res = unwrap(await api.GET('/v1/namespaces/{namespace}/deliveries', {
                params: { path: { namespace }, query: { event_id: eventId, limit: 100, offset: 0 } },
            }));
            deliveriesByEvent.set(eventId, res.items || []);
            deliveriesByEvent = new Map(deliveriesByEvent);
        } catch (e: any) {
            console.error('Failed to fetch deliveries for event:', e);
        } finally {
            loadingDeliveries.delete(eventId);
            loadingDeliveries = new Set(loadingDeliveries);
        }
    }

    async function retryDelivery(deliveryId: string, namespace: string, eventId: string) {
        retryingDeliveries.add(deliveryId);
        retryingDeliveries = new Set(retryingDeliveries);
        try {
            unwrap(await api.POST('/v1/namespaces/{namespace}/deliveries/{delivery_id}:retry', {
                params: { path: { namespace, delivery_id: deliveryId } },
            }));
            await fetchDeliveries(eventId, namespace);
        } catch (e: any) {
            console.error('Failed to retry delivery:', e);
        } finally {
            retryingDeliveries.delete(deliveryId);
            retryingDeliveries = new Set(retryingDeliveries);
        }
    }

    async function toggleDeliveryAttempts(deliveryId: string, namespace: string) {
        if (expandedDeliveries.has(deliveryId)) {
            expandedDeliveries.delete(deliveryId);
            expandedDeliveries = new Set(expandedDeliveries);
        } else {
            expandedDeliveries.add(deliveryId);
            expandedDeliveries = new Set(expandedDeliveries);

            if (!attemptsByDelivery.has(deliveryId)) {
                await fetchAttempts(deliveryId, namespace);
            }
        }
    }

    async function fetchAttempts(deliveryId: string, namespace: string) {
        loadingAttempts.add(deliveryId);
        loadingAttempts = new Set(loadingAttempts);
        try {
            const res = unwrap(await api.GET('/v1/namespaces/{namespace}/deliveries/{delivery_id}/attempts', {
                params: { path: { namespace, delivery_id: deliveryId } },
            }));
            attemptsByDelivery.set(deliveryId, res.items || []);
            attemptsByDelivery = new Map(attemptsByDelivery);
        } catch (e: any) {
            console.error('Failed to fetch attempts for delivery:', e);
        } finally {
            loadingAttempts.delete(deliveryId);
            loadingAttempts = new Set(loadingAttempts);
        }
    }
</script>

{#if eventReports.length === 0}
    <EmptyState
        icon="send"
        title="No event instances found"
        description='No events of type "{currentEventName}" have been pushed yet.'
    />
{:else}
    <div class="panel overflow-hidden">
        <div class="overflow-x-auto">
            <table class="w-full text-left">
                <thead>
                    <tr class="border-b border-line">
                        <th class="th">Event ID</th>
                        <th class="th hidden sm:table-cell">Schema</th>
                        <th class="th hidden sm:table-cell">Namespace</th>
                        <th class="th hidden sm:table-cell">Created At</th>
                        <th class="th hidden md:table-cell">Deliveries</th>
                        <th class="th">Payload</th>
                        <th class="th"></th>
                    </tr>
                </thead>
                <tbody>
                    {#each eventReports as report}
                        <tr class="row-line row-hover transition">
                            <td class="td">
                                <CopyableId id={report.event_id} href="/events/instances/{report.event_id}" truncate={12} />
                                <span class="block sm:hidden mt-0.5">
                                    <span class="chip">{report.namespace || 'N/A'}</span>
                                </span>
                            </td>
                            <td class="td hidden sm:table-cell">
                                {#if report.schema_valid}
                                    <span class="chip" style="color:var(--color-ok);border-color:color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 12%,var(--color-panel-2))">Valid</span>
                                {:else}
                                    <span class="chip" style="color:var(--color-bad);border-color:color-mix(in srgb,var(--color-bad) 35%,transparent);background:color-mix(in srgb,var(--color-bad) 12%,var(--color-panel-2))">Invalid</span>
                                {/if}
                            </td>
                            <td class="td hidden sm:table-cell">
                                <span class="chip">{report.namespace || 'N/A'}</span>
                            </td>
                            <td class="td hidden sm:table-cell"><span class="mono tnum text-muted text-xs" title={formatTimestamp(report.created_at)}>{timeAgo(report.created_at)}</span></td>
                            <td class="td hidden md:table-cell">
                                <div class="flex items-center gap-2 text-xs mono tnum">
                                    {#if report.successful_deliveries > 0}
                                        <span class="font-medium" style="color:var(--color-ok)">{report.successful_deliveries} ok</span>
                                    {/if}
                                    {#if report.failed_deliveries > 0}
                                        <span class="font-medium" style="color:var(--color-bad)">{report.failed_deliveries} failed</span>
                                    {/if}
                                    {#if report.pending_deliveries > 0}
                                        <span class="font-medium" style="color:var(--color-warn)">{report.pending_deliveries} pending</span>
                                    {/if}
                                    {#if report.webhook_count === 0 && report.successful_deliveries === 0 && report.failed_deliveries === 0 && report.pending_deliveries === 0}
                                        <span class="text-faint">None</span>
                                    {/if}
                                </div>
                            </td>
                            <td class="td">
                                <details class="cursor-pointer">
                                    <summary class="text-xs mono link-beacon list-none select-none">View</summary>
                                    <pre class="panel-2 mono mt-1.5 p-2 text-[10px] text-muted overflow-auto max-h-32 max-w-xs">{formatPayload(report.payload)}</pre>
                                </details>
                            </td>
                            <td class="td">
                                <button
                                    onclick={() => toggleRow(report.event_id, report.namespace)}
                                    class="link text-xs mono transition"
                                >
                                    {expandedRows.has(report.event_id) ? 'Hide' : 'Deliveries'}
                                </button>
                            </td>
                        </tr>

                        {#if expandedRows.has(report.event_id)}
                            <tr class="row-line" style="background:var(--color-panel-2)">
                                <td colspan="7" class="px-4 py-4">
                                    {#if loadingDeliveries.has(report.event_id)}
                                        <div class="flex items-center justify-center py-4">
                                            <img src={favicon} alt="" aria-hidden="true" class="w-4 h-4 animate-spin mr-2" />
                                            <span class="text-sm text-muted">Loading deliveries…</span>
                                        </div>
                                    {:else if deliveriesByEvent.has(report.event_id) && (deliveriesByEvent.get(report.event_id)?.length ?? 0) > 0}
                                        <div class="space-y-2">
                                            <p class="eyebrow mb-2">Webhook Deliveries</p>
                                            <div class="overflow-x-auto">
                                                <table class="w-full text-xs">
                                                    <thead>
                                                        <tr class="border-b border-line">
                                                            <th class="th !py-2 !px-3">Webhook</th>
                                                            <th class="th !py-2 !px-3">Status</th>
                                                            <th class="th !py-2 !px-3 hidden sm:table-cell">Response</th>
                                                            <th class="th !py-2 !px-3 hidden md:table-cell">Attempts</th>
                                                            <th class="th !py-2 !px-3 hidden lg:table-cell">Last Attempt</th>
                                                            <th class="th !py-2 !px-3">Actions</th>
                                                        </tr>
                                                    </thead>
                                                    <tbody>
                                                        {#each deliveriesByEvent.get(report.event_id) ?? [] as delivery}
                                                            <tr class="row-line row-hover transition {expandedDeliveries.has(delivery.delivery_id) ? 'bg-white/[0.03]' : ''}">
                                                                <td class="td !py-2 !px-3">
                                                                    <CopyableId id={delivery.webhook_id} href="/webhooks/{delivery.webhook_id}" truncate={12} />
                                                                </td>
                                                                <td class="td !py-2 !px-3">
                                                                    <div class="flex items-center gap-1.5">
                                                                        <StatusBadge status={delivery.status} />
                                                                        {#if delivery.error_category && delivery.error_category !== 'success'}
                                                                            {@const badge = getCategoryBadge(delivery.error_category)}
                                                                            <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                                                                        {/if}
                                                                    </div>
                                                                </td>
                                                                <td class="td !py-2 !px-3 hidden sm:table-cell">
                                                                    <span class="mono tnum" style="color:{(delivery.response_code ?? 0) >= 200 && (delivery.response_code ?? 0) < 300 ? 'var(--color-ok)' : (delivery.response_code ?? 0) >= 400 ? 'var(--color-bad)' : 'var(--color-muted)'}">
                                                                        {delivery.response_code || '—'}
                                                                    </span>
                                                                </td>
                                                                <td class="td !py-2 !px-3 hidden md:table-cell">
                                                                    <button
                                                                        onclick={() => toggleDeliveryAttempts(delivery.delivery_id, report.namespace)}
                                                                        class="link-beacon mono tnum font-medium transition"
                                                                    >
                                                                        {delivery.attempt_count}/{delivery.max_attempts}
                                                                        <span class="text-[10px] ml-0.5" aria-hidden="true">{expandedDeliveries.has(delivery.delivery_id) ? '▲' : '▼'}</span>
                                                                    </button>
                                                                </td>
                                                                <td class="td !py-2 !px-3 hidden lg:table-cell"><span class="mono tnum text-muted" title={formatTimestamp(delivery.last_attempted_at)}>{timeAgo(delivery.last_attempted_at)}</span></td>
                                                                <td class="td !py-2 !px-3">
                                                                    <div class="flex items-center gap-2">
                                                                        {#if delivery.status === 'failed' || delivery.status === 'expired' || (delivery.error_category && delivery.error_category !== 'success')}
                                                                            <button
                                                                                onclick={() => retryDelivery(delivery.delivery_id, report.namespace, report.event_id)}
                                                                                disabled={retryingDeliveries.has(delivery.delivery_id)}
                                                                                class="btn btn-ghost !px-2 !py-1 text-xs"
                                                                            >
                                                                                {retryingDeliveries.has(delivery.delivery_id) ? 'Retrying…' : 'Retry'}
                                                                            </button>
                                                                        {/if}
                                                                        <a
                                                                            href="/deliveries/{delivery.delivery_id}"
                                                                            class="btn btn-ghost !px-2 !py-1 text-xs"
                                                                        >
                                                                            Details
                                                                        </a>
                                                                    </div>
                                                                </td>
                                                            </tr>

                                                            {#if expandedDeliveries.has(delivery.delivery_id)}
                                                                <tr class="row-line" style="background:var(--color-panel-2)">
                                                                    <td colspan="6" class="px-3 py-3">
                                                                        {#if loadingAttempts.has(delivery.delivery_id)}
                                                                            <div class="flex items-center justify-center py-3">
                                                                                <img src={favicon} alt="" aria-hidden="true" class="w-3 h-3 animate-spin mr-2" />
                                                                                <span class="text-xs text-muted">Loading attempt history…</span>
                                                                            </div>
                                                                        {:else if attemptsByDelivery.has(delivery.delivery_id) && (attemptsByDelivery.get(delivery.delivery_id)?.length ?? 0) > 0}
                                                                            <div class="ml-4 border-l-2 border-line pl-3">
                                                                                <p class="eyebrow mb-1.5">Attempt History</p>
                                                                                <div class="space-y-1.5">
                                                                                    {#each attemptsByDelivery.get(delivery.delivery_id) ?? [] as attempt, i}
                                                                                        <div class="flex items-start gap-3 py-1.5 px-2 rounded" style="background:color-mix(in srgb,var(--color-{attempt.success ? 'ok' : 'bad'}) 8%,var(--color-panel-2))">
                                                                                            <div class="flex-shrink-0 w-5 h-5 flex items-center justify-center rounded-full text-[10px] font-bold mono tnum" style="color:var(--color-{attempt.success ? 'ok' : 'bad'});background:color-mix(in srgb,var(--color-{attempt.success ? 'ok' : 'bad'}) 15%,transparent)">
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
                                    {:else}
                                        <p class="text-sm text-muted text-center py-3">No deliveries found for this event.</p>
                                    {/if}
                                </td>
                            </tr>
                        {/if}
                    {/each}
                </tbody>
            </table>
        </div>
    </div>
{/if}

<style>
    details summary::-webkit-details-marker {
        display: none;
    }
</style>
