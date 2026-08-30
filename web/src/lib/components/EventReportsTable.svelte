<script lang="ts">
    import type { components } from '$lib/api-types';
    import { api, unwrap } from '$lib/services';
    import { getCategoryBadge } from '$lib/utils';
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
    <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
                <thead>
                    <tr class="border-b border-gray-200 bg-gray-50/50">
                        <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Event ID</th>
                        <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Schema</th>
                        <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Namespace</th>
                        <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Created At</th>
                        <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden md:table-cell">Deliveries</th>
                        <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Payload</th>
                        <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider"></th>
                    </tr>
                </thead>
                <tbody class="divide-y divide-gray-100">
                    {#each eventReports as report}
                        <tr class="hover:bg-gray-50 transition">
                            <td class="px-4 py-3">
                                <CopyableId id={report.event_id} href="/events/instances/{report.event_id}" truncate={12} />
                                <span class="block sm:hidden mt-0.5">
                                    <span class="px-1.5 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 rounded">{report.namespace || 'N/A'}</span>
                                </span>
                            </td>
                            <td class="px-4 py-3 hidden sm:table-cell">
                                {#if report.schema_valid}
                                    <span class="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-green-50 text-green-700">Valid</span>
                                {:else}
                                    <span class="inline-flex items-center px-1.5 py-0.5 rounded-full text-[10px] font-medium bg-red-50 text-red-700">Invalid</span>
                                {/if}
                            </td>
                            <td class="px-4 py-3 hidden sm:table-cell">
                                <span class="px-1.5 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 rounded">{report.namespace || 'N/A'}</span>
                            </td>
                            <td class="px-4 py-3 text-xs text-gray-700 hidden sm:table-cell">{formatTimestamp(report.created_at)}</td>
                            <td class="px-4 py-3 hidden md:table-cell">
                                <div class="flex items-center gap-2 text-xs">
                                    {#if report.successful_deliveries > 0}
                                        <span class="text-green-700 font-medium">{report.successful_deliveries} ok</span>
                                    {/if}
                                    {#if report.failed_deliveries > 0}
                                        <span class="text-red-700 font-medium">{report.failed_deliveries} failed</span>
                                    {/if}
                                    {#if report.pending_deliveries > 0}
                                        <span class="text-yellow-700 font-medium">{report.pending_deliveries} pending</span>
                                    {/if}
                                    {#if report.webhook_count === 0 && report.successful_deliveries === 0 && report.failed_deliveries === 0 && report.pending_deliveries === 0}
                                        <span class="text-gray-400">None</span>
                                    {/if}
                                </div>
                            </td>
                            <td class="px-4 py-3">
                                <details class="cursor-pointer">
                                    <summary class="text-xs font-medium text-blue-600 hover:text-blue-800 list-none select-none">View</summary>
                                    <pre class="mt-1.5 bg-gray-50 p-2 rounded border border-gray-200 text-[10px] overflow-auto max-h-32 max-w-xs font-mono">{formatPayload(report.payload)}</pre>
                                </details>
                            </td>
                            <td class="px-4 py-3">
                                <button
                                    onclick={() => toggleRow(report.event_id, report.namespace)}
                                    class="text-xs font-medium text-gray-600 hover:text-gray-900 transition"
                                >
                                    {expandedRows.has(report.event_id) ? 'Hide' : 'Deliveries'}
                                </button>
                            </td>
                        </tr>

                        {#if expandedRows.has(report.event_id)}
                            <tr class="bg-gray-50/50">
                                <td colspan="7" class="px-4 py-4">
                                    {#if loadingDeliveries.has(report.event_id)}
                                        <div class="flex items-center justify-center py-4">
                                            <img src={favicon} alt="Loading" class="w-4 h-4 animate-spin mr-2" />
                                            <span class="text-sm text-gray-500">Loading deliveries...</span>
                                        </div>
                                    {:else if deliveriesByEvent.has(report.event_id) && (deliveriesByEvent.get(report.event_id)?.length ?? 0) > 0}
                                        <div class="space-y-2">
                                            <p class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2">Webhook Deliveries</p>
                                            <div class="overflow-x-auto">
                                                <table class="w-full text-xs">
                                                    <thead>
                                                        <tr class="border-b border-gray-200">
                                                            <th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase">Webhook</th>
                                                            <th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase">Status</th>
                                                            <th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase hidden sm:table-cell">Response</th>
                                                            <th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase hidden md:table-cell">Attempts</th>
                                                            <th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase hidden lg:table-cell">Last Attempt</th>
                                                            <th class="px-3 py-2 text-left text-xs font-semibold text-gray-500 uppercase">Actions</th>
                                                        </tr>
                                                    </thead>
                                                    <tbody class="divide-y divide-gray-100">
                                                        {#each deliveriesByEvent.get(report.event_id) ?? [] as delivery}
                                                            <tr class="hover:bg-white transition {expandedDeliveries.has(delivery.delivery_id) ? 'bg-blue-50/30' : ''}">
                                                                <td class="px-3 py-2">
                                                                    <CopyableId id={delivery.webhook_id} href="/webhooks/{delivery.webhook_id}" truncate={12} />
                                                                </td>
                                                                <td class="px-3 py-2">
                                                                    <div class="flex items-center gap-1.5">
                                                                        <StatusBadge status={delivery.status} />
                                                                        {#if delivery.error_category && delivery.error_category !== 'success'}
                                                                            {@const badge = getCategoryBadge(delivery.error_category)}
                                                                            <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                                                                        {/if}
                                                                    </div>
                                                                </td>
                                                                <td class="px-3 py-2 hidden sm:table-cell">
                                                                    <span class="font-mono {(delivery.response_code ?? 0) >= 200 && (delivery.response_code ?? 0) < 300 ? 'text-green-600' : (delivery.response_code ?? 0) >= 400 ? 'text-red-600' : 'text-gray-500'}">
                                                                        {delivery.response_code || '—'}
                                                                    </span>
                                                                </td>
                                                                <td class="px-3 py-2 hidden md:table-cell">
                                                                    <button
                                                                        onclick={() => toggleDeliveryAttempts(delivery.delivery_id, report.namespace)}
                                                                        class="text-blue-600 hover:text-blue-800 font-medium hover:underline transition"
                                                                    >
                                                                        {delivery.attempt_count}/{delivery.max_attempts}
                                                                        <span class="text-[10px] ml-0.5">{expandedDeliveries.has(delivery.delivery_id) ? '▲' : '▼'}</span>
                                                                    </button>
                                                                </td>
                                                                <td class="px-3 py-2 text-gray-500 hidden lg:table-cell">{formatTimestamp(delivery.last_attempted_at)}</td>
                                                                <td class="px-3 py-2">
                                                                    <div class="flex items-center gap-2">
                                                                        {#if delivery.status === 'failed' || delivery.status === 'expired' || (delivery.error_category && delivery.error_category !== 'success')}
                                                                            <button
                                                                                onclick={() => retryDelivery(delivery.delivery_id, report.namespace, report.event_id)}
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
                                                                    <td colspan="6" class="px-3 py-3">
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
                                    {:else}
                                        <p class="text-sm text-gray-500 text-center py-3">No deliveries found for this event.</p>
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
