<script lang="ts">
    import type { components } from '#lib/api-types.js';
    import { api, unwrap } from '#lib/services.js';
    import { getCategoryBadge, formatAPIError, timeAgo, formatTimestamp } from '#lib/utils.js';
    import { consumerLabel } from '#lib/consumer.svelte.js';
    import { isSystemConsumer } from '#lib/system.js';
    import StatusBadge from './StatusBadge.svelte';
    import CopyableId from './CopyableId.svelte';
    import WebhookLabel from './WebhookLabel.svelte';
    import AttemptHistory from './AttemptHistory.svelte';

    type DeliveryItem = components["schemas"]["DeliveryItem"];
    type Column = 'event' | 'webhook' | 'consumer';

    // The one delivery list of the admin UI: every page that lists
    // deliveries renders this, so the columns, retry rule and attempt
    // history are the same wherever a delivery is viewed. A page only hides
    // the columns its own context already fixes (the webhook on a webhook's
    // page, the event on an event's page).
    interface Props {
        deliveries: DeliveryItem[];
        hide?: Column[];
        // Called after a retry was accepted, to reload the list.
        onchange?: () => void | Promise<void>;
        onerror?: (message: string) => void;
    }

    let { deliveries, hide = [], onchange, onerror }: Props = $props();

    const show = (c: Column) => !hide.includes(c);
    let colspan = $derived(6 + (['event', 'webhook', 'consumer'] as Column[]).filter(show).length);

    let expanded: Set<string> = $state(new Set());
    let retrying: Set<string> = $state(new Set());

    function toggle(id: string) {
        if (expanded.has(id)) expanded.delete(id);
        else expanded.add(id);
        expanded = new Set(expanded);
    }

    // Failed and expired deliveries are retried by hand; paused ones are held
    // until retried.
    const retriable = (d: DeliveryItem) => d.status === 'failed' || d.status === 'expired' || d.status === 'paused';

    async function retry(id: string) {
        retrying.add(id);
        retrying = new Set(retrying);
        try {
            unwrap(await api.POST('/v1/deliveries/{delivery_id}:retry', {
                params: { path: { delivery_id: id } },
            }));
            await onchange?.();
        } catch (e: any) {
            onerror?.(formatAPIError(e, 'Failed to retry delivery'));
        } finally {
            retrying.delete(id);
            retrying = new Set(retrying);
        }
    }

    function codeColor(code: number | undefined): string {
        const c = code ?? 0;
        return c >= 200 && c < 300 ? 'var(--color-ok)' : c >= 400 ? 'var(--color-bad)' : 'var(--color-muted)';
    }
</script>

<div class="overflow-x-auto">
    <table class="w-full text-left">
        <thead>
            <tr class="border-b border-line">
                <th class="th">Delivery</th>
                {#if show('event')}<th class="th hidden md:table-cell">Event</th>{/if}
                {#if show('webhook')}<th class="th hidden sm:table-cell">Webhook</th>{/if}
                {#if show('consumer')}<th class="th hidden md:table-cell">Consumer</th>{/if}
                <th class="th">Status</th>
                <th class="th hidden sm:table-cell">Response</th>
                <th class="th hidden sm:table-cell">Attempts</th>
                <th class="th hidden lg:table-cell">Created</th>
                <th class="th"></th>
            </tr>
        </thead>
        <tbody>
            {#each deliveries as d (d.delivery_id)}
                <tr class="row-line row-hover transition {expanded.has(d.delivery_id) ? 'bg-black/[0.03]' : ''}">
                    <td class="td"><CopyableId id={d.delivery_id} href="/deliveries/{d.delivery_id}" truncate={12} /></td>
                    {#if show('event')}
                        <td class="td hidden md:table-cell">
                            {#if d.event_name}
                                <a href="/events/{encodeURIComponent(d.event_name)}/reports" class="block text-xs mono text-text hover:text-beacon truncate max-w-[14rem]" title={d.event_name}>{d.event_name}</a>
                            {/if}
                            <CopyableId id={d.event_id} href="/events/instances/{d.event_id}" truncate={12} />
                        </td>
                    {/if}
                    {#if show('webhook')}
                        <td class="td hidden sm:table-cell"><WebhookLabel id={d.webhook_id} description={d.webhook_description} url={d.webhook_url} /></td>
                    {/if}
                    {#if show('consumer')}
                        <td class="td hidden md:table-cell">
                            {#if d.consumer}<span class="chip" title={isSystemConsumer(d.consumer) ? "Sparrow's own alert delivery" : undefined}>{consumerLabel(d.consumer)}</span>{/if}
                        </td>
                    {/if}
                    <td class="td">
                        <div class="flex items-center gap-1.5">
                            <StatusBadge status={d.status} />
                            {#if d.error_category && d.error_category !== 'success'}
                                {@const badge = getCategoryBadge(d.error_category)}
                                <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                            {/if}
                        </div>
                    </td>
                    <td class="td hidden sm:table-cell">
                        <span class="mono tnum text-xs" style="color:{codeColor(d.response_code)}">{d.response_code || '—'}</span>
                    </td>
                    <td class="td hidden sm:table-cell">
                        <button
                            onclick={() => toggle(d.delivery_id)}
                            aria-expanded={expanded.has(d.delivery_id)}
                            class="link-beacon mono tnum text-xs font-medium transition"
                        >
                            {d.attempt_count}/{d.max_attempts}
                            <span class="text-[10px] ml-0.5" aria-hidden="true">{expanded.has(d.delivery_id) ? '▲' : '▼'}</span>
                        </button>
                    </td>
                    <td class="td hidden lg:table-cell">
                        <span class="mono tnum text-muted text-xs" title="Created {formatTimestamp(d.created_at)}{d.last_attempted_at ? `\nLast attempt ${formatTimestamp(d.last_attempted_at)}` : ''}">{timeAgo(d.created_at)}</span>
                    </td>
                    <td class="td">
                        <div class="flex items-center justify-end gap-2">
                            {#if retriable(d)}
                                <button
                                    onclick={() => retry(d.delivery_id)}
                                    disabled={retrying.has(d.delivery_id)}
                                    class="btn btn-ghost !px-2 !py-1 !text-xs"
                                >
                                    {retrying.has(d.delivery_id) ? 'Retrying…' : 'Retry'}
                                </button>
                            {/if}
                            <a href="/deliveries/{d.delivery_id}" class="btn btn-ghost !px-2 !py-1 !text-xs">Details</a>
                        </div>
                    </td>
                </tr>
                {#if expanded.has(d.delivery_id)}
                    <tr class="row-line" style="background:var(--color-panel-2)">
                        <td {colspan} class="px-3 py-3">
                            <AttemptHistory deliveryId={d.delivery_id} />
                        </td>
                    </tr>
                {/if}
            {/each}
        </tbody>
    </table>
</div>
