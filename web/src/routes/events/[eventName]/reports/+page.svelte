<script lang="ts">
    import { page } from '$app/state';
    import { EventReportsTable, Pagination } from '$lib';
    import { api, unwrap } from '$lib/services';
    import { onMount, onDestroy } from 'svelte';
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
    let namespaceFilter = $state('');
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
        const ns = namespaceFilter.trim() || 'default';

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
        namespaceFilter = '';
        schemaValidFilter = 'all';
        labelsFilter = '';
        createdAfterFilter = '';
        createdBeforeFilter = '';
        applyFilters();
    }

    let hasActiveFilters = $derived(
        namespaceFilter.trim() !== '' ||
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
        const ns = namespaceFilter.trim() || 'default';
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
        const ns = namespaceFilter.trim() || 'default';
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
        const ns = namespaceFilter.trim() || 'default';
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

    onMount(() => {
        fetchEventReports();
    });
</script>

<svelte:head>
    <title>{currentEvent?.name || 'Event'} Reports | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div class="mb-6">
            <nav class="flex items-center text-sm text-gray-500 mb-4">
                <a href="/events" class="hover:text-gray-900 transition">Events</a>
                <span class="mx-2">/</span>
                <span class="text-gray-900 font-medium">{currentEvent?.name || 'Loading...'}</span>
            </nav>

            <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-2">
                <div>
                    <h1 class="text-2xl font-bold text-gray-900">Event Reports</h1>
                    <p class="text-sm text-gray-500 mt-0.5">
                        Instances of "{currentEvent?.name || 'Loading...'}"
                        {#if namespaceFilter.trim()}
                            in namespace <span class="font-semibold text-gray-700">{namespaceFilter.trim()}</span>
                        {:else}
                            in namespace <span class="font-semibold text-gray-700">default</span>
                        {/if}
                    </p>
                </div>
                {#if !loading}
                    <div class="flex items-center gap-3">
                        <span class="text-sm text-gray-500">
                            {totalCount} report{totalCount !== 1 ? 's' : ''}
                        </span>
                        {#if totalCount > 0}
                            <button
                                onclick={prepareRepush}
                                disabled={preparingRepush}
                                class="px-3 py-1.5 text-xs font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 disabled:opacity-50 transition"
                            >
                                {preparingRepush ? 'Preparing...' : 'Re-push All Matching'}
                            </button>
                        {/if}
                    </div>
                {/if}
            </div>
        </div>

        <div class="bg-white rounded-lg border border-gray-200 p-4 mb-4">
            <div class="flex flex-col sm:flex-row gap-3">
                <input type="text" placeholder="Namespace (default: default)" bind:value={namespaceFilter} class="flex-1 px-3 py-1.5 border border-gray-300 rounded-lg text-sm" />
                <input type="text" placeholder="Labels (key1=val1, key2=val2)" bind:value={labelsFilter} class="flex-1 px-3 py-1.5 border border-gray-300 rounded-lg text-sm" />
                <select bind:value={schemaValidFilter} class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm">
                    <option value="all">All schema validity</option>
                    <option value="valid">Valid only</option>
                    <option value="invalid">Invalid only</option>
                </select>
                <button onclick={applyFilters} class="px-4 py-1.5 bg-gray-900 text-white rounded-lg text-sm font-medium hover:bg-gray-800">Apply</button>
                {#if hasActiveFilters}
                    <button onclick={clearFilters} class="px-4 py-1.5 text-gray-500 hover:text-gray-700 text-sm">Clear</button>
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
            <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
                <div class="animate-pulse divide-y divide-gray-100">
                    {#each Array(5) as _}
                        <div class="p-4 h-14 bg-gray-50"></div>
                    {/each}
                </div>
            </div>
        {:else if error}
            <div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
                <p class="text-sm text-red-700">{error}</p>
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
</div>

<ConfirmDialog
    open={confirmRepush}
    title="Re-push Events"
    message="This will re-push {repushTotal} matching event{repushTotal !== 1 ? 's' : ''} as new events. Each event will be schema-validated against the current schema and delivered to all matching subscriptions. Continue?"
    confirmLabel="Re-push"
    variant="warning"
    onconfirm={executeRepush}
    oncancel={() => { confirmRepush = false; repushId = ''; }}
/>
