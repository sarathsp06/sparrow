<script lang="ts">
    import type { components } from '$lib/api-types';
    import { api, unwrap } from '$lib/services';
    import { getCategoryBadge, formatTimestamp } from '$lib/utils';
    import favicon from '$lib/assets/favicon.svg';

    type AttemptItem = components["schemas"]["AttemptItem"];

    // The attempt history of one delivery, fetched when mounted (i.e. when
    // its row is expanded).
    interface Props {
        deliveryId: string;
    }

    let { deliveryId }: Props = $props();

    let attempts: AttemptItem[] = $state([]);
    let loading = $state(true);
    let error = $state('');

    $effect(() => {
        const id = deliveryId;
        loading = true;
        error = '';
        api.GET('/v1/deliveries/{delivery_id}/attempts', {
            params: { path: { delivery_id: id } },
        }).then((r) => {
            attempts = unwrap(r).items || [];
        }).catch((e: any) => {
            console.error('Failed to fetch attempts for delivery:', e);
            error = 'Failed to load attempt history.';
        }).finally(() => {
            loading = false;
        });
    });
</script>

{#if loading}
    <div class="flex items-center justify-center py-3">
        <img src={favicon} alt="" aria-hidden="true" class="w-3 h-3 animate-spin mr-2" />
        <span class="text-xs text-muted">Loading attempt history…</span>
    </div>
{:else if error}
    <p class="text-[11px] ml-4" style="color:var(--color-bad)">{error}</p>
{:else if attempts.length > 0}
    <div class="ml-4 border-l-2 border-line pl-3">
        <p class="eyebrow mb-1.5">Attempt History</p>
        <div class="space-y-1.5">
            {#each attempts as attempt, i}
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
