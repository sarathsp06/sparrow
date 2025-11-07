<script lang="ts">
    import type { EventReport } from '../../../../proto/webhook_pb.js';

    interface Props {
        eventReports: EventReport[];
        loading?: boolean;
        error?: string;
        currentEventName?: string;
    }

    let { eventReports, loading = false, error = '', currentEventName = 'this event' }: Props = $props();

    function formatTimestamp(timestamp: number): string {
        return new Date(timestamp * 1000).toLocaleString();
    }

    function formatPayload(payload: any): string {
        if (!payload) return 'N/A';
        try {
            return JSON.stringify(payload, null, 2);
        } catch {
            return String(payload);
        }
    }
</script>

{#if error}
    <div class="bg-red-100 border border-red-300 text-red-700 rounded-lg p-4 mb-6 flex items-center gap-3 shadow-sm">
        <span class="material-symbols-outlined">error</span>
        <p>{error}</p>
    </div>
{:else if eventReports.length === 0}
    <div class="bg-white border rounded-lg p-8 text-center text-gray-500 shadow-sm flex flex-col items-center gap-4">
        <span class="material-symbols-outlined text-5xl text-gray-300">event_note</span>
        <h3 class="text-xl font-semibold">No event instances found</h3>
        <p>No events of type "{currentEventName}" have been pushed yet.</p>
    </div>
{:else}
    <!-- Event Reports Table -->
    <div class="bg-white rounded-lg shadow-sm border overflow-hidden">
        <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
                <thead class="text-xs text-gray-700 uppercase bg-gray-50">
                    <tr>
                        <th class="px-4 py-3">Event ID</th>
                        <th class="px-4 py-3">Created At</th>
                        <th class="px-4 py-3">Webhooks</th>
                        <th class="px-4 py-3">Deliveries</th>
                        <th class="px-4 py-3">TTL</th>
                        <th class="px-4 py-3">Payload</th>
                    </tr>
                </thead>
                <tbody>
                    {#each eventReports as report}
                        <tr class="border-b hover:bg-gray-50">
                            <td class="px-4 py-3 font-mono text-xs">
                                {report.eventId}
                            </td>
                            <td class="px-4 py-3">
                                {formatTimestamp(Number(report.createdAt))}
                            </td>
                            <td class="px-4 py-3">
                                <span class="bg-blue-100 text-blue-800 px-2 py-1 rounded text-xs">
                                    {report.webhookCount} webhooks
                                </span>
                            </td>
                            <td class="px-4 py-3">
                                <div class="flex gap-1">
                                    {#if report.successfulDeliveries > 0}
                                        <span class="bg-green-100 text-green-800 px-2 py-1 rounded text-xs">
                                            ✓ {report.successfulDeliveries}
                                        </span>
                                    {/if}
                                    {#if report.failedDeliveries > 0}
                                        <span class="bg-red-100 text-red-800 px-2 py-1 rounded text-xs">
                                            ✗ {report.failedDeliveries}
                                        </span>
                                    {/if}
                                    {#if report.pendingDeliveries > 0}
                                        <span class="bg-yellow-100 text-yellow-800 px-2 py-1 rounded text-xs">
                                            ⏳ {report.pendingDeliveries}
                                        </span>
                                    {/if}
                                </div>
                            </td>
                            <td class="px-4 py-3">
                                {report.ttlSeconds}s
                            </td>
                            <td class="px-4 py-3">
                                <details class="cursor-pointer">
                                    <summary class="text-blue-600 hover:text-blue-800">View</summary>
                                    <pre class="mt-2 bg-gray-100 p-2 rounded text-xs overflow-auto max-h-32 max-w-xs">
                                        {formatPayload(report.payload)}
                                    </pre>
                                </details>
                            </td>
                        </tr>
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