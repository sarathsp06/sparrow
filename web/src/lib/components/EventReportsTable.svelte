<script lang="ts">
    import type { EventReport } from '../../../../proto/webhook_pb.js';
    import Table from './table.svelte';

    interface Props {
        eventReports: EventReport[];
        loading?: boolean;
        error?: string;
        currentEventName?: string;
    }

    let { eventReports, loading = false, error = '', currentEventName = 'this event' }: Props = $props();

    const headers = ['eventId', 'createdAt',  'deliveries', 'ttlSeconds', 'payload'];

    function formatTimestamp(timestamp: number): string {
        return new Date(timestamp * 1000).toLocaleString();
    }

    function formatDeliveries(successful: number, failed: number): string {
        const total = successful + failed;
        if (total === 0) return 'N/A';
        let result = "";
        if (successful > 0) {
            result += `✅ ${successful} `;
        }
        if (failed > 0) {
            result += `❌ ${failed} `;
        }
        return result;
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

{#snippet eventIdSnippet({ value }: { value: any; row: Record<string, any>; rowIndex: number })}
    {value}
{/snippet}

{#snippet createdAtSnippet({ value }: { value: any; row: Record<string, any>; rowIndex: number })}
    {formatTimestamp(Number(value))}
{/snippet}


{#snippet deliveriesSnippet({ row }: { value: any; row: Record<string, any>; rowIndex: number })}
    {formatDeliveries(row.successfulDeliveries, row.failedDeliveries)}
{/snippet}

{#snippet ttlSnippet({ value }: { value: any; row: Record<string, any>; rowIndex: number })}
    {value}s
{/snippet}

{#snippet payloadSnippet({ value }: { value: any; row: Record<string, any>; rowIndex: number })}
    <details class="cursor-pointer">
        <summary class="text-blue-600 hover:text-blue-800 list-none">View</summary>
        <pre class="mt-2 bg-gray-100 p-2 rounded text-xs overflow-auto max-h-32 max-w-xs">
            {formatPayload(value)}
        </pre>
    </details>
{/snippet}

{#snippet emptyStateSnippet({ itemName }: { itemName: string })}
    <div class="bg-white border rounded-lg p-8 text-center text-gray-500 shadow-sm flex flex-col items-center gap-4">
        <span class="material-symbols-outlined text-5xl text-gray-300">event_note</span>
        <h3 class="text-xl font-semibold">No event instances found</h3>
        <p>No events of type "{currentEventName}" have been pushed yet.</p>
    </div>
{/snippet}

<Table 
    {headers} 
    data={eventReports} 
    {loading} 
    {error}
    itemName="event instances"
    columnFormatters={{
        eventId: { header: 'Event ID', snippet: eventIdSnippet },
        createdAt: { header: 'Created At', snippet: createdAtSnippet },
        deliveries: { header: 'Deliveries', snippet: deliveriesSnippet },
        ttlSeconds: { header: 'TTL', snippet: ttlSnippet },
        payload: { header: 'Payload', snippet: payloadSnippet }
    }}
    {emptyStateSnippet}
/>

<style>
    details summary::-webkit-details-marker {
        display: none;
    }
</style>