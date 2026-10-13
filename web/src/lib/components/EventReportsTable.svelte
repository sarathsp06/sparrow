<script lang="ts">
    import type { components } from '$lib/api-types';
    import { api, unwrap } from '$lib/services';
    import { timeAgo, formatTimestamp } from '$lib/utils';
    import CopyableId from './CopyableId.svelte';
    import EmptyState from './EmptyState.svelte';
    import DeliveryTable from './DeliveryTable.svelte';
    import favicon from '$lib/assets/favicon.svg';

    type EventOccurrenceItem = components["schemas"]["EventOccurrenceItem"];
    type DeliveryItem = components["schemas"]["DeliveryItem"];

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

    function formatPayload(payload: any): string {
        if (!payload) return 'N/A';
        try {
            return JSON.stringify(payload, null, 2);
        } catch {
            return String(payload);
        }
    }

    async function toggleRow(eventId: string, consumer: string) {
        if (expandedRows.has(eventId)) {
            expandedRows.delete(eventId);
            expandedRows = new Set(expandedRows);
        } else {
            expandedRows.add(eventId);
            expandedRows = new Set(expandedRows);

            if (!deliveriesByEvent.has(eventId)) {
                await fetchDeliveries(eventId, consumer);
            }
        }
    }

    async function fetchDeliveries(eventId: string, consumer: string) {
        loadingDeliveries.add(eventId);
        loadingDeliveries = new Set(loadingDeliveries);
        try {
            const res = unwrap(await api.GET('/v1/consumers/{consumer}/deliveries', {
                params: { path: { consumer }, query: { event_id: eventId, limit: 100 } },
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
                        <th class="th hidden sm:table-cell">Consumer</th>
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
                                    <span class="chip">{report.consumer || 'N/A'}</span>
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
                                <span class="chip">{report.consumer || 'N/A'}</span>
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
                                    onclick={() => toggleRow(report.event_id, report.consumer)}
                                    aria-expanded={expandedRows.has(report.event_id)}
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
                                            <DeliveryTable deliveries={deliveriesByEvent.get(report.event_id) ?? []} hide={['event', 'consumer']} onchange={() => fetchDeliveries(report.event_id, report.consumer)} />
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
