<script lang="ts">
    import { page } from '$app/state';
    import { api, unwrap } from '$lib/services';
    import { ERROR_CATEGORIES, formatAPIError } from '$lib/utils';
    import { consumerFilter, withConsumer } from '$lib/consumer.svelte';
    import { SYSTEM_CONSUMER, isSystemConsumer } from '$lib/system';
    import { pulseStore } from '$lib/pulse.svelte';
    import { onDestroy, untrack } from 'svelte';
    import type { components } from '$lib/api-types';
    type DeliveryStatus = components['schemas']['DeliveryItem']['status'];
    import DeliveryTable from '$lib/components/DeliveryTable.svelte';
    import CursorPager from '$lib/components/CursorPager.svelte';
    import { CursorPages } from '$lib/cursor-pages.svelte';
    import ConsumerPicker from '$lib/components/ConsumerPicker.svelte';
    import EmptyState from '$lib/components/EmptyState.svelte';
    import BatchProgress from '$lib/components/BatchProgress.svelte';
    import { BatchJob } from '$lib/batch-job.svelte';
    import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

    type DeliveryItem = components["schemas"]["DeliveryItem"];

    let deliveries: DeliveryItem[] = $state([]);
    let loading = $state(true);
    let error = $state('');
    let pageSize = $state(25);
    const pages = new CursorPages();

    // Filters. Initial values may come from the URL, e.g. the event type
    // import links here with ?status=failed&error_category=template_error.
    const initial = page.url.searchParams;
    let webhookIdFilter = $state(initial.get('webhook_id') ?? '');
    let eventIdFilter = $state(initial.get('event_id') ?? '');
    let statusFilter = $state(initial.get('status') ?? '');
    let errorCategoryFilter = $state(initial.get('error_category') ?? '');
    let createdAfterFilter = $state(initial.get('created_after') ?? '');
    let createdBeforeFilter = $state(initial.get('created_before') ?? '');
    let allConsumers = $derived(!consumerFilter.value);
    // A batch re-delivery that can reach _sparrow re-sends alert emails.
    let alertsInScope = $derived(
        isSystemConsumer(consumerFilter.value) ||
        (allConsumers && deliveries.some((d) => isSystemConsumer(d.consumer)))
    );

    // Batch retry state
    const retry = new BatchJob('retry', () => consumerFilter.value, (m) => (error = m));
    let preparingRetry = $state(false);

    let live = $state(false);
    let liveTimer: ReturnType<typeof setInterval> | undefined;

    onDestroy(() => {
        retry.stop();
        if (liveTimer) clearInterval(liveTimer);
    });

    function startLive() {
        stopLive();
        live = true;
        liveTimer = setInterval(() => {
            // Don't yank the user off an older page, and skip hidden tabs.
            if (document.hidden || pages.page !== 1) return;
            pages.reset(); fetchDeliveries(false, true); pulseStore.ping();
        }, 5000);
    }
    function stopLive() {
        live = false;
        if (liveTimer) { clearInterval(liveTimer); liveTimer = undefined; }
    }
    function toggleLive() {
        if (live) stopLive();
        else startLive();
    }

    let fetchSeq = 0; // drops out-of-order responses (live tick vs. filter change)
    async function fetchDeliveries(prepareRetry: boolean = false, silent: boolean = false) {
        const seq = ++fetchSeq;
        loading = !prepareRetry && !silent;
        if (!prepareRetry) error = '';

        const ns = consumerFilter.value;

        try {
            const res = unwrap(await api.GET('/v1/deliveries', {
                params: {
                    query: {
                        consumer: ns || undefined,
                        webhook_id: webhookIdFilter.trim() || undefined,
                        event_id: eventIdFilter.trim() || undefined,
                        status: (statusFilter || undefined) as DeliveryStatus | undefined,
                        error_category: errorCategoryFilter || undefined,
                        created_after: createdAfterFilter || undefined,
                        created_before: createdBeforeFilter || undefined,
                        prepare_retry: prepareRetry,
                        limit: pageSize,
                        cursor: prepareRetry ? undefined : pages.cursor || undefined,
                    },
                },
            }));
            if (seq !== fetchSeq) return; // a newer fetch superseded this one
            if (prepareRetry) {
                if (!retry.prepared(res.retry_id, res.retry_total)) {
                    error = 'No matching deliveries to retry.';
                }
                return;
            }
            deliveries = res.items || [];
            pages.update(res.pagination);
        } catch (e: any) {
            if (seq !== fetchSeq) return;
            console.error('Failed to fetch deliveries:', e);
            error = formatAPIError(e, 'Failed to load deliveries');
        } finally {
            if (seq === fetchSeq) loading = false;
        }
    }

    function applyFilters() {
        pages.reset();
        fetchDeliveries();
    }

    function clearFilters() {
        webhookIdFilter = '';
        eventIdFilter = '';
        statusFilter = '';
        errorCategoryFilter = '';
        createdAfterFilter = '';
        createdBeforeFilter = '';
        applyFilters();
    }

    let hasActiveFilters = $derived(
        webhookIdFilter.trim() !== '' ||
        eventIdFilter.trim() !== '' ||
        statusFilter !== '' ||
        errorCategoryFilter !== '' ||
        createdAfterFilter !== '' ||
        createdBeforeFilter !== ''
    );

    async function prepareRetryBatch() {
        preparingRetry = true;
        error = '';
        try {
            await fetchDeliveries(true);
        } finally {
            preparingRetry = false;
        }
    }

    function onBatchDone() {
        fetchDeliveries();
    }

    $effect(() => {
        consumerFilter.value; // refetch when the active consumer changes
        untrack(() => {
            pages.reset();
            fetchDeliveries();
        });
    });
</script>

<svelte:head>
    <title>Deliveries | Sparrow</title>
</svelte:head>

<main class="mx-auto max-w-[1600px] px-4 sm:px-8 py-8">
    <div class="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 mb-6">
        <div>
            <p class="eyebrow mb-1.5">Traffic / Deliveries</p>
            <h1 class="text-2xl">Deliveries</h1>
            <p class="text-sm text-muted mt-1">All webhook deliveries in <span class="mono text-text">{consumerFilter.label}</span></p>
        </div>
        <div class="flex items-center gap-3">
            <button onclick={toggleLive} aria-pressed={live} class="btn btn-ghost !px-3 !py-1.5 !text-xs" title="Auto-refresh every 5s">
                <span class="relative flex w-2 h-2">
                    {#if live}<span class="absolute inline-flex w-full h-full rounded-full opacity-70 animate-ping" style="background:var(--color-ok)"></span>{/if}
                    <span class="relative inline-flex w-2 h-2 rounded-full" style="background:var(--color-{live ? 'ok' : 'idle'})"></span>
                </span>
                {live ? 'Live' : 'Live off'}
            </button>
            {#if !loading}
                {#if deliveries.length > 0}
                    <button onclick={prepareRetryBatch} disabled={preparingRetry} class="btn btn-ghost !px-3 !py-1.5 !text-xs">
                        {preparingRetry ? 'Preparing…' : 'Re-deliver all matching'}
                    </button>
                {/if}
            {/if}
        </div>
    </div>

    <div class="panel p-4 mb-4">
        <div class="flex flex-col gap-3">
            <div class="flex flex-col sm:flex-row gap-3 flex-wrap">
                <ConsumerPicker id="deliveries-consumer" label="Filter by consumer" value={consumerFilter.value} onchange={(c) => (consumerFilter.value = c)} class="sm:w-60" />
                <input type="text" placeholder="Webhook ID" aria-label="Filter by webhook ID" bind:value={webhookIdFilter} class="input flex-1" />
                <input type="text" placeholder="Event ID" aria-label="Filter by event ID" bind:value={eventIdFilter} class="input flex-1" />
                <select bind:value={statusFilter} aria-label="Filter by status" class="select sm:w-44">
                    <option value="">All statuses</option>
                    <option value="pending">Pending</option>
                    <option value="sending">Sending</option>
                    <option value="success">Success</option>
                    <option value="failed">Failed</option>
                    <option value="retrying">Retrying</option>
                    <option value="expired">Expired</option>
                    <option value="paused">Paused</option>
                </select>
                <select bind:value={errorCategoryFilter} aria-label="Filter by error category" class="select sm:w-52">
                    <option value="">All error categories</option>
                    {#each ERROR_CATEGORIES as cat}
                        <option value={cat.value}>{cat.label}</option>
                    {/each}
                </select>
            </div>
            <div class="flex flex-col sm:flex-row gap-3 flex-wrap sm:items-end">
                <label class="block">
                    <span class="field-label !mb-1.5">Created after</span>
                    <input type="date" bind:value={createdAfterFilter} aria-label="Filter deliveries created after" class="input sm:w-44" />
                </label>
                <label class="block">
                    <span class="field-label !mb-1.5">Created before</span>
                    <input type="date" bind:value={createdBeforeFilter} aria-label="Filter deliveries created before" class="input sm:w-44" />
                </label>
                <div class="flex items-center gap-3 sm:ml-auto">
                    <button onclick={applyFilters} class="btn btn-beacon">Apply</button>
                    {#if hasActiveFilters}
                        <button onclick={clearFilters} class="btn btn-ghost">Clear</button>
                    {/if}
                </div>
            </div>
        </div>
    </div>

    {#if retry.progress}
        <div class="mb-4">
            <BatchProgress
                batch={retry.progress}
                oncancel={() => retry.cancel()}
                ondone={onBatchDone}
            />
        </div>
    {/if}

    {#if error}
        <div class="panel p-4 mb-4" role="alert" aria-live="assertive" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
            <p class="text-sm" style="color:var(--color-bad)">{error}</p>
        </div>
    {/if}

    {#if loading}
        <div class="panel overflow-hidden">
            <div class="animate-pulse">
                {#each Array(8) as _}
                    <div class="row-line h-14 bg-black/[0.015]"></div>
                {/each}
            </div>
        </div>
    {:else if deliveries.length === 0}
        <div class="panel">
            <EmptyState icon="send" title="No deliveries found" description="No deliveries match the current filters." />
        </div>
    {:else}
        <div class="panel overflow-hidden">
            <DeliveryTable {deliveries} hide={allConsumers ? [] : ['consumer']} onchange={() => fetchDeliveries()} onerror={(m) => (error = m)} />
        </div>

        <CursorPager {pages} shown={deliveries.length} itemLabel="deliveries" onchange={() => fetchDeliveries()} />
    {/if}
</main>

<ConfirmDialog
    open={retry.confirming}
    title="Re-deliver Matching Deliveries"
    message="This will retry {retry.total} matching deliver{retry.total !== 1 ? 'ies' : 'y'}. Sparrow re-sends each stored delivery to the same webhook; it does not create new events.{alertsInScope ? ` This scope includes ${SYSTEM_CONSUMER}'s alert deliveries: their alert emails are sent again.` : ''} Continue?"
    confirmLabel="Re-deliver"
    variant="warning"
    onconfirm={() => retry.start()}
    oncancel={() => retry.dismiss()}
/>
