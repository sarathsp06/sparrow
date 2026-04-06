<script lang="ts">
    import { page } from '$app/state';
    import { EventReportsTable, Pagination } from '$lib';
    import { eventClient as client } from '$lib/services';
    import { onMount, onDestroy } from 'svelte';
    import type { EventReport, RegisteredEvent } from '../../../../../../proto/webhook_pb.js';
    import { timestampFromDate } from '@bufbuild/protobuf/wkt';
    import BatchProgress from '$lib/components/BatchProgress.svelte';
    import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

    let eventReports: EventReport[] = $state([]);
    let currentEvent: RegisteredEvent | undefined = $state();
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
            const res = await client.getEvent({ name: eventName });
            currentEvent = res.event;
            if (!currentEvent) {
                error = 'Event not found';
            }
        } catch (e: any) {
            console.error('Failed to fetch event details:', e);
            error = `Failed to load event details: ${e.message}`;
        }
    }

    function buildRequest(pageNum: number, prepareRepush: boolean = false) {
        const offset = (pageNum - 1) * pageSize;
        const req: Record<string, any> = {
            namespace: namespaceFilter.trim(),
            eventName: currentEvent!.name,
            pagination: { limit: pageSize, offset },
            prepareRepush,
        };

        if (schemaValidFilter === 'valid') req.schemaValid = true;
        else if (schemaValidFilter === 'invalid') req.schemaValid = false;

        // Parse labels: "key1=val1, key2=val2"
        if (labelsFilter.trim()) {
            const labels: Record<string, string> = {};
            for (const pair of labelsFilter.split(',')) {
                const [k, v] = pair.split('=').map(s => s.trim());
                if (k && v) labels[k] = v;
            }
            if (Object.keys(labels).length > 0) req.labels = labels;
        }

        if (createdAfterFilter) {
            req.createdAfter = timestampFromDate(new Date(createdAfterFilter));
        }
        if (createdBeforeFilter) {
            req.createdBefore = timestampFromDate(new Date(createdBeforeFilter));
        }

        return req;
    }

    async function fetchEventReports(pageNum: number = 1) {
        loading = true;
        error = '';

        if (!currentEvent) {
            await fetchEventDetails();
            if (!currentEvent) { loading = false; return; }
        }

        try {
            const req = buildRequest(pageNum);
            const res = await client.listEventReports(req);
            eventReports = res.events || [];
            totalCount = res.pagination?.totalCount || 0;
            currentPage = pageNum;
        } catch (e: any) {
            console.error('Failed to fetch event reports:', e);
            error = `Failed to load event reports: ${e.message}`;
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

    // -- Batch Re-push --

    async function prepareRepush() {
        if (!currentEvent) return;
        preparingRepush = true;
        error = '';
        try {
            const req = buildRequest(1, true);
            const res = await client.listEventReports(req);
            if (res.repushId) {
                repushId = res.repushId;
                repushTotal = res.pagination?.totalCount || 0;
                confirmRepush = true;
            } else {
                error = 'No matching events to re-push.';
            }
        } catch (e: any) {
            error = `Failed to prepare re-push: ${e.message}`;
        } finally {
            preparingRepush = false;
        }
    }

    async function executeRepush() {
        confirmRepush = false;
        if (!repushId) return;
        try {
            const res = await client.rePushEvents({ repushId });
            batchStatus = { status: res.status, total: res.total, processed: 0, failed: 0 };
            startPolling();
        } catch (e: any) {
            error = `Failed to start re-push: ${e.message}`;
        }
    }

    function startPolling() {
        if (pollingTimer) clearInterval(pollingTimer);
        pollingTimer = setInterval(async () => {
            if (!repushId) { stopPolling(); return; }
            try {
                const res = await client.getRepushStatus({ repushId });
                if (res.batch) {
                    batchStatus = {
                        status: res.batch.status,
                        total: res.batch.total,
                        processed: res.batch.processed,
                        failed: res.batch.failed,
                    };
                    if (res.batch.status === 'completed' || res.batch.status === 'failed' || res.batch.status === 'cancelled') {
                        stopPolling();
                    }
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
        try {
            await client.cancelRepush({ repushId });
        } catch (e: any) {
            error = `Failed to cancel re-push: ${e.message}`;
        }
    }

    function onBatchDone() {
        // Refresh list after batch completes
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
        <!-- Header -->
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
                            across all namespaces
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

        <!-- Filters -->
        <div class="bg-white rounded-lg border border-gray-200 p-4 mb-4">
            <div class="flex flex-wrap items-end gap-3">
                <div class="w-full sm:w-40">
                    <label for="ns-filter" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Namespace</label>
                    <input
                        id="ns-filter"
                        type="text"
                        placeholder="All namespaces"
                        bind:value={namespaceFilter}
                        onkeydown={(e) => e.key === 'Enter' && applyFilters()}
                        class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
                    />
                </div>
                <div class="w-full sm:w-32">
                    <label for="schema-filter" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Schema Match</label>
                    <select
                        id="schema-filter"
                        bind:value={schemaValidFilter}
                        onchange={applyFilters}
                        class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
                    >
                        <option value="all">All</option>
                        <option value="valid">Valid</option>
                        <option value="invalid">Invalid</option>
                    </select>
                </div>
                <div class="w-full sm:w-48">
                    <label for="labels-filter" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Labels</label>
                    <input
                        id="labels-filter"
                        type="text"
                        placeholder="key=val, key2=val2"
                        bind:value={labelsFilter}
                        onkeydown={(e) => e.key === 'Enter' && applyFilters()}
                        class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900 font-mono"
                    />
                </div>
                <div class="w-full sm:w-44">
                    <label for="after-filter" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Created After</label>
                    <input
                        id="after-filter"
                        type="datetime-local"
                        bind:value={createdAfterFilter}
                        onchange={applyFilters}
                        class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
                    />
                </div>
                <div class="w-full sm:w-44">
                    <label for="before-filter" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Created Before</label>
                    <input
                        id="before-filter"
                        type="datetime-local"
                        bind:value={createdBeforeFilter}
                        onchange={applyFilters}
                        class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
                    />
                </div>
                <div class="flex items-center gap-2">
                    <button
                        onclick={applyFilters}
                        class="px-3 py-1.5 text-xs font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition"
                    >
                        Apply
                    </button>
                    {#if hasActiveFilters}
                        <button
                            onclick={clearFilters}
                            class="px-3 py-1.5 text-xs font-medium text-gray-600 bg-gray-100 rounded-lg hover:bg-gray-200 transition"
                        >
                            Clear
                        </button>
                    {/if}
                </div>
            </div>
        </div>

        <!-- Batch progress -->
        {#if batchStatus}
            <div class="mb-4">
                <BatchProgress
                    batch={batchStatus}
                    label="Re-push Events"
                    oncancel={cancelRepush}
                    ondone={onBatchDone}
                />
            </div>
        {/if}

        {#if loading}
            <!-- Loading skeleton -->
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
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden lg:table-cell">TTL</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Payload</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-gray-100">
                            {#each Array(5) as _}
                                <tr class="animate-pulse">
                                    <td class="px-4 py-3"><div class="h-4 bg-gray-200 rounded w-32"></div></td>
                                    <td class="px-4 py-3 hidden sm:table-cell"><div class="h-4 bg-gray-100 rounded w-12"></div></td>
                                    <td class="px-4 py-3 hidden sm:table-cell"><div class="h-4 bg-gray-100 rounded w-20"></div></td>
                                    <td class="px-4 py-3 hidden sm:table-cell"><div class="h-4 bg-gray-100 rounded w-36"></div></td>
                                    <td class="px-4 py-3 hidden md:table-cell"><div class="h-4 bg-gray-100 rounded w-20"></div></td>
                                    <td class="px-4 py-3 hidden lg:table-cell"><div class="h-4 bg-gray-100 rounded w-12"></div></td>
                                    <td class="px-4 py-3"><div class="h-4 bg-gray-200 rounded w-16"></div></td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        {:else if error}
            <div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
                <div class="flex items-start gap-3">
                    <svg class="w-5 h-5 text-red-500 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                        <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
                    </svg>
                    <div>
                        <p class="text-sm font-medium text-red-800">{error}</p>
                        <button onclick={() => { error = ''; fetchEventReports(currentPage); }} class="text-sm text-red-600 hover:text-red-800 underline mt-1">Retry</button>
                    </div>
                </div>
            </div>
        {:else}
            <EventReportsTable
                {eventReports}
                {loading}
                {error}
                currentEventName={currentEvent?.name || 'this event'}
            />

            <Pagination
                {currentPage}
                {totalPages}
                {totalCount}
                {pageSize}
                onPageChange={handlePageChange}
                itemLabel="reports"
            />
        {/if}
    </main>
</div>

<!-- Confirm Re-push Dialog -->
<ConfirmDialog
    open={confirmRepush}
    title="Re-push Events"
    message="This will re-push {repushTotal} matching event{repushTotal !== 1 ? 's' : ''} as new events. Each event will be schema-validated against the current schema and delivered to all matching subscriptions. Continue?"
    confirmLabel="Re-push"
    variant="warning"
    onconfirm={executeRepush}
    oncancel={() => { confirmRepush = false; repushId = ''; }}
/>
