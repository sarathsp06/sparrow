<script lang="ts">
    import { page } from '$app/state';
    import { EventReportsTable, Pagination } from '$lib';
    import { api, unwrap } from '$lib/services';
    import { onDestroy } from 'svelte';
    import { namespaceStore } from '$lib/namespace.svelte';
    import type { components } from '$lib/api-types';
    import { formatAPIError } from '$lib/utils';
    import BatchProgress from '$lib/components/BatchProgress.svelte';
    import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

    type EventOccurrenceItem = components["schemas"]["EventOccurrenceItem"];
    type EventTypeItem = components["schemas"]["EventTypeItem"];

    let eventReports: EventOccurrenceItem[] = $state([]);
    let currentEvent: EventTypeItem | undefined = $state();
    let loading = $state(true);
    let error = $state('');
    let currentPage = $state(1);
    let totalCount = $state(0);
    let pageSize = $state(20);
    let totalPages = $derived(Math.max(1, Math.ceil(totalCount / pageSize)));

    // Filters
    let schemaValidFilter = $state<'all' | 'valid' | 'invalid'>('all');
    let labelsFilter = $state('');
    let createdAfterFilter = $state('');
    let createdBeforeFilter = $state('');

    // Batch re-push state
    let repushId = $state('');
    let batchStatus = $state<{ status: string; total: number; processed: number; failed: number } | undefined>();
    let preparingRepush = $state(false);
    let confirmRepush = $state(false);
    let repushTotal = $state(0);
    let pollingTimer: ReturnType<typeof setInterval> | undefined;

    const eventName = decodeURIComponent(page.params.eventName || '');

    onDestroy(() => {
        if (pollingTimer) clearInterval(pollingTimer);
    });

    async function fetchEventDetails() {
        try {
            currentEvent = unwrap(await api.GET('/v1/event-types/{name}', { params: { path: { name: eventName } } }));
        } catch (e: any) {
            console.error('Failed to fetch event details:', e);
            error = formatAPIError(e, 'Failed to load event details');
        }
    }

    async function fetchEventReports(pageNum: number = 1, prepareRepush: boolean = false) {
        loading = !prepareRepush;
        if (!prepareRepush) error = '';

        if (!currentEvent) {
            await fetchEventDetails();
            if (!currentEvent) { loading = false; return; }
        }

        const offset = (pageNum - 1) * pageSize;
        const ns = namespaceStore.value;

        try {
            const res = unwrap(await api.GET('/v1/namespaces/{namespace}/events', {
                params: {
                    path: { namespace: ns },
                    query: { event: eventName, prepare_repush: prepareRepush, limit: pageSize, offset },
                },
            }));
            if (prepareRepush) {
                if (res.repush_id) {
                    repushId = res.repush_id;
                    repushTotal = res.pagination?.total_count || 0;
                    confirmRepush = true;
                } else {
                    error = 'No matching events to re-push.';
                }
                return;
            }
            eventReports = res.items || [];
            totalCount = res.pagination?.total_count || 0;
            currentPage = pageNum;
        } catch (e: any) {
            console.error('Failed to fetch event reports:', e);
            error = formatAPIError(e, 'Failed to load event reports');
        } finally {
            loading = false;
        }
    }

    function handlePageChange(pageNum: number) {
        if (pageNum >= 1 && pageNum <= totalPages) {
            fetchEventReports(pageNum);
        }
    }

    function applyFilters() {
        currentPage = 1;
        fetchEventReports(1);
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
            await fetchEventReports(1, true);
        } finally {
            preparingRepush = false;
        }
    }

    async function executeRepush() {
        confirmRepush = false;
        if (!repushId) return;
        const ns = namespaceStore.value;
        try {
            const res = unwrap(await api.POST('/v1/namespaces/{namespace}/events:rePush', {
                params: { path: { namespace: ns } },
                body: { repush_id: repushId },
            }));
            batchStatus = { status: res.status, total: res.total, processed: res.processed, failed: res.failed };
            startPolling();
        } catch (e: any) {
            error = formatAPIError(e, 'Failed to start re-push');
        }
    }

    function startPolling() {
        if (pollingTimer) clearInterval(pollingTimer);
        const ns = namespaceStore.value;
        pollingTimer = setInterval(async () => {
            if (!repushId) { stopPolling(); return; }
            try {
                const res = unwrap(await api.GET('/v1/namespaces/{namespace}/repush-jobs/{job_id}', {
                    params: { path: { namespace: ns, job_id: repushId } },
                }));
                batchStatus = { status: res.status, total: res.total, processed: res.processed, failed: res.failed };
                if (res.status === 'completed' || res.status === 'failed' || res.status === 'cancelled') {
                    stopPolling();
                }
            } catch {
                stopPolling();
            }
        }, 2000);
    }

    function stopPolling() {
        if (pollingTimer) { clearInterval(pollingTimer); pollingTimer = undefined; }
    }

    async function cancelRepush() {
        if (!repushId) return;
        const ns = namespaceStore.value;
        try {
            await api.POST('/v1/namespaces/{namespace}/repush-jobs/{job_id}:cancel', {
                params: { path: { namespace: ns, job_id: repushId } },
            });
        } catch (e: any) {
            error = formatAPIError(e, 'Failed to cancel re-push');
        }
    }

    function onBatchDone() {
        fetchEventReports(currentPage);
    }

    $effect(() => {
        namespaceStore.value; // refetch when the active namespace changes
        fetchEventReports(1);
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
                    Instances of "{currentEvent?.name || 'Loading…'}" in namespace <span class="chip">{namespaceStore.value}</span>
                </p>
            </div>
            {#if !loading}
                <div class="flex items-center gap-3">
                    <span class="text-sm text-muted mono tnum">
                        {totalCount} report{totalCount !== 1 ? 's' : ''}
                    </span>
                    {#if totalCount > 0}
                        <button
                            onclick={prepareRepush}
                            disabled={preparingRepush}
                            class="btn btn-ghost !px-3 !py-1.5 text-xs"
                        >
                            {preparingRepush ? 'Preparing…' : 'Re-push All Matching'}
                        </button>
                    {/if}
                </div>
            {/if}
        </div>
    </div>

    <div class="panel p-4 mb-4">
        <div class="flex flex-col sm:flex-row gap-3">
            <input type="text" placeholder="Labels (key1=val1, key2=val2)" bind:value={labelsFilter} class="input flex-1" />
            <select bind:value={schemaValidFilter} class="select sm:w-48">
                <option value="all">All schema validity</option>
                <option value="valid">Valid only</option>
                <option value="invalid">Invalid only</option>
            </select>
            <button onclick={applyFilters} class="btn btn-beacon !px-4 !py-1.5 text-sm">Apply</button>
            {#if hasActiveFilters}
                <button onclick={clearFilters} class="btn btn-ghost !px-4 !py-1.5 text-sm">Clear</button>
            {/if}
        </div>
    </div>

    {#if batchStatus}
        <div class="mb-4">
            <BatchProgress
                batch={batchStatus}
                oncancel={cancelRepush}
                ondone={onBatchDone}
            />
        </div>
    {/if}

    {#if loading}
        <div class="panel overflow-hidden">
            <div class="animate-pulse">
                {#each Array(5) as _}
                    <div class="row-line h-14 bg-white/[0.015]"></div>
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

        <Pagination
            {currentPage}
            {totalPages}
            {totalCount}
            {pageSize}
            onPageChange={handlePageChange}
        />
    {/if}
</main>

<ConfirmDialog
    open={confirmRepush}
    title="Re-push Events"
    message="This will re-push {repushTotal} matching event{repushTotal !== 1 ? 's' : ''} as new events. Each event will be schema-validated against the current schema and delivered to all matching subscriptions. Continue?"
    confirmLabel="Re-push"
    variant="warning"
    onconfirm={executeRepush}
    oncancel={() => { confirmRepush = false; repushId = ''; }}
/>
