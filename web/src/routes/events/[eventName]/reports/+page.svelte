<script lang="ts">
    import { page } from '$app/state';
    import { EventReportsTable } from '$lib';
    import CursorPager from '$lib/components/CursorPager.svelte';
    import { CursorPages } from '$lib/cursor-pages.svelte';
    import { api, unwrap } from '$lib/services';
    import { onDestroy, untrack } from 'svelte';
    import { consumerFilter } from '$lib/consumer.svelte';
    import ConsumerPicker from '$lib/components/ConsumerPicker.svelte';
    import type { components } from '$lib/api-types';
    import { formatAPIError } from '$lib/utils';
    import BatchProgress from '$lib/components/BatchProgress.svelte';
    import { BatchJob } from '$lib/batch-job.svelte';
    import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

    type EventOccurrenceItem = components["schemas"]["EventOccurrenceItem"];
    type EventTypeItem = components["schemas"]["EventTypeItem"];

    let eventReports: EventOccurrenceItem[] = $state([]);
    let currentEvent: EventTypeItem | undefined = $state();
    let loading = $state(true);
    let error = $state('');
    let pageSize = $state(20);
    const pages = new CursorPages();

    // Filters
    let schemaValidFilter = $state<'all' | 'valid' | 'invalid'>('all');
    let labelsFilter = $state('');
    let createdAfterFilter = $state('');
    let createdBeforeFilter = $state('');

    // Batch re-push state
    const repush = new BatchJob('repush', () => consumerFilter.value, (m) => (error = m));
    let preparingRepush = $state(false);

    const eventName = decodeURIComponent(page.params.eventName || '');

    onDestroy(() => {
        repush.stop();
    });

    async function fetchEventDetails() {
        try {
            currentEvent = unwrap(await api.GET('/v1/event-types/{name}', { params: { path: { name: eventName } } }));
        } catch (e: any) {
            console.error('Failed to fetch event details:', e);
            error = formatAPIError(e, 'Failed to load event details');
        }
    }

    async function fetchEventReports(prepareRepush: boolean = false) {
        loading = !prepareRepush;
        if (!prepareRepush) error = '';

        if (!currentEvent) {
            await fetchEventDetails();
            if (!currentEvent) { loading = false; return; }
        }

        const ns = consumerFilter.value;

        try {
            const res = unwrap(await api.GET('/v1/events', {
                params: {
                    query: {
                        consumer: ns || undefined,
                        event: eventName,
                        schema_valid: schemaValidFilter === 'all' ? undefined : schemaValidFilter,
                        labels: labelsFilter.trim() || undefined,
                        created_after: createdAfterFilter || undefined,
                        created_before: createdBeforeFilter || undefined,
                        prepare_repush: prepareRepush,
                        limit: pageSize,
                        cursor: prepareRepush ? undefined : pages.cursor || undefined,
                    },
                },
            }));
            if (prepareRepush) {
                if (!repush.prepared(res.repush_id, res.repush_total)) {
                    error = 'No matching events to re-push.';
                }
                return;
            }
            eventReports = res.items || [];
            pages.update(res.pagination);
        } catch (e: any) {
            console.error('Failed to fetch event reports:', e);
            error = formatAPIError(e, 'Failed to load event reports');
        } finally {
            loading = false;
        }
    }

    function applyFilters() {
        pages.reset();
        fetchEventReports();
    }

    function clearFilters() {
        schemaValidFilter = 'all';
        labelsFilter = '';
        createdAfterFilter = '';
        createdBeforeFilter = '';
        applyFilters();
    }

    let hasActiveFilters = $derived(
        schemaValidFilter !== 'all' ||
        labelsFilter.trim() !== '' ||
        createdAfterFilter !== '' ||
        createdBeforeFilter !== ''
    );

    async function prepareRepush() {
        if (!currentEvent) return;
        preparingRepush = true;
        try {
            await fetchEventReports(true);
        } finally {
            preparingRepush = false;
        }
    }

    function onBatchDone() {
        fetchEventReports();
    }

    $effect(() => {
        consumerFilter.value; // refetch when the active consumer changes
        untrack(() => {
            pages.reset();
            fetchEventReports();
        });
    });
</script>

<svelte:head>
    <title>{currentEvent?.name || 'Event'} Reports | Sparrow</title>
</svelte:head>

<main class="mx-auto max-w-7xl px-4 sm:px-8 py-8">
    <div class="mb-6">
        <nav class="flex items-center gap-2 text-sm text-muted mb-4">
            <a href="/events" class="link">Events</a>
            <span class="text-faint">/</span>
            <span class="text-text">{currentEvent?.name || 'Loading…'}</span>
        </nav>

        <div class="flex flex-col sm:flex-row sm:justify-between sm:items-end gap-2">
            <div>
                <p class="eyebrow mb-1.5">Catalog / Reports</p>
                <h1 class="text-2xl">Event Reports</h1>
                <p class="text-sm text-muted mt-1">
                    Instances of "{currentEvent?.name || 'Loading…'}" in <span class="chip">{consumerFilter.label}</span>
                </p>
            </div>
            {#if !loading}
                <div class="flex items-center gap-3">
                    {#if eventReports.length > 0}
                        <button
                            onclick={prepareRepush}
                            disabled={preparingRepush}
                            class="btn btn-ghost !px-3 !py-1.5 text-xs"
                        >
                            {preparingRepush ? 'Preparing…' : 'Re-push all matching as new events'}
                        </button>
                    {/if}
                </div>
            {/if}
        </div>
    </div>

    <div class="panel p-4 mb-4">
        <div class="flex flex-col gap-3">
            <div class="flex flex-col sm:flex-row gap-3">
                <ConsumerPicker id="reports-consumer" label="Filter by consumer" value={consumerFilter.value} onchange={(c) => (consumerFilter.value = c)} class="sm:w-60" />
                <input type="text" placeholder="Labels (key=value, key2=value2)" bind:value={labelsFilter} class="input flex-1" />
                <select bind:value={schemaValidFilter} class="select sm:w-48">
                    <option value="all">All schema validity</option>
                    <option value="valid">Valid only</option>
                    <option value="invalid">Invalid only</option>
                </select>
            </div>
            <div class="flex flex-col sm:flex-row gap-3 flex-wrap sm:items-end">
                <label class="block">
                    <span class="field-label !mb-1.5">Created after</span>
                    <input type="date" bind:value={createdAfterFilter} class="input sm:w-44" />
                </label>
                <label class="block">
                    <span class="field-label !mb-1.5">Created before</span>
                    <input type="date" bind:value={createdBeforeFilter} class="input sm:w-44" />
                </label>
                <div class="flex items-center gap-3 sm:ml-auto">
                    <button onclick={applyFilters} class="btn btn-beacon !px-4 !py-1.5 text-sm">Apply</button>
                    {#if hasActiveFilters}
                        <button onclick={clearFilters} class="btn btn-ghost !px-4 !py-1.5 text-sm">Clear</button>
                    {/if}
                </div>
            </div>
        </div>
    </div>

    {#if repush.progress}
        <div class="mb-4">
            <BatchProgress
                batch={repush.progress}
                oncancel={() => repush.cancel()}
                ondone={onBatchDone}
            />
        </div>
    {/if}

    {#if loading}
        <div class="panel overflow-hidden">
            <div class="animate-pulse">
                {#each Array(5) as _}
                    <div class="row-line h-14 bg-black/[0.015]"></div>
                {/each}
            </div>
        </div>
    {:else if error}
        <div class="panel p-4 mb-6" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
            <p class="text-sm" style="color:var(--color-bad)">{error}</p>
        </div>
    {:else}
        <EventReportsTable
            {eventReports}
            {loading}
            {error}
            currentEventName={currentEvent?.name}
        />

        <CursorPager {pages} shown={eventReports.length} itemLabel="events" onchange={() => fetchEventReports()} />
    {/if}
</main>

<ConfirmDialog
    open={repush.confirming}
    title="Re-push Matching Events"
    message="This will re-push {repush.total} matching event{repush.total !== 1 ? 's' : ''} as new event occurrences. Sparrow re-runs current schema validation, subscription matching, and delivery fan-out. Continue?"
    confirmLabel="Re-push as new events"
    variant="warning"
    onconfirm={() => repush.start()}
    oncancel={() => repush.dismiss()}
/>
