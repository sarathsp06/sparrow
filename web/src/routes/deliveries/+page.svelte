<script lang="ts">
    import { api, unwrap } from '$lib/services';
    import { getCategoryBadge, ERROR_CATEGORIES, formatAPIError } from '$lib/utils';
    import { onMount, onDestroy } from 'svelte';
    import type { components } from '$lib/api-types';
    import StatusBadge from '$lib/components/StatusBadge.svelte';
    import CopyableId from '$lib/components/CopyableId.svelte';
    import Pagination from '$lib/components/Pagination.svelte';
    import EmptyState from '$lib/components/EmptyState.svelte';
    import BatchProgress from '$lib/components/BatchProgress.svelte';
    import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

    type DeliveryItem = components["schemas"]["DeliveryItem"];

    let deliveries: DeliveryItem[] = $state([]);
    let loading = $state(true);
    let error = $state('');
    let currentPage = $state(1);
    let totalCount = $state(0);
    let pageSize = $state(25);
    let totalPages = $derived(Math.max(1, Math.ceil(totalCount / pageSize)));

    // Filters
    let namespaceFilter = $state('default');
    let webhookIdFilter = $state('');
    let eventIdFilter = $state('');
    let statusFilter = $state('');
    let errorCategoryFilter = $state('');

    // Batch retry state
    let retryId = $state('');
    let batchStatus = $state<{ status: string; total: number; processed: number; failed: number } | undefined>();
    let preparingRetry = $state(false);
    let confirmRetry = $state(false);
    let retryTotal = $state(0);
    let pollingTimer: ReturnType<typeof setInterval> | undefined;

    let retryingDeliveries: Set<string> = $state(new Set());

    onDestroy(() => {
        if (pollingTimer) clearInterval(pollingTimer);
    });

    async function fetchDeliveries(pageNum: number = 1, prepareRetry: boolean = false) {
        loading = !prepareRetry;
        if (!prepareRetry) error = '';

        const ns = namespaceFilter.trim() || 'default';
        const offset = (pageNum - 1) * pageSize;

        try {
            const res = unwrap(await api.GET('/v1/namespaces/{namespace}/deliveries', {
                params: {
                    path: { namespace: ns },
                    query: {
                        webhook_id: webhookIdFilter.trim() || undefined,
                        event_id: eventIdFilter.trim() || undefined,
                        status: statusFilter || undefined,
                        prepare_retry: prepareRetry,
                        limit: pageSize,
                        offset,
                    },
                },
            }));
            if (prepareRetry) {
                if (res.retry_id) {
                    retryId = res.retry_id;
                    retryTotal = res.pagination?.total_count || 0;
                    confirmRetry = true;
                } else {
                    error = 'No matching deliveries to retry.';
                }
                return;
            }
            deliveries = res.items || [];
            totalCount = res.pagination?.total_count || 0;
            currentPage = pageNum;
        } catch (e: any) {
            console.error('Failed to fetch deliveries:', e);
            error = formatAPIError(e, 'Failed to load deliveries');
        } finally {
            loading = false;
        }
    }

    function handlePageChange(pageNum: number) {
        if (pageNum >= 1 && pageNum <= totalPages) {
            fetchDeliveries(pageNum);
        }
    }

    function applyFilters() {
        currentPage = 1;
        fetchDeliveries(1);
    }

    function clearFilters() {
        namespaceFilter = 'default';
        webhookIdFilter = '';
        eventIdFilter = '';
        statusFilter = '';
        errorCategoryFilter = '';
        applyFilters();
    }

    let hasActiveFilters = $derived(
        webhookIdFilter.trim() !== '' ||
        eventIdFilter.trim() !== '' ||
        statusFilter !== '' ||
        errorCategoryFilter !== ''
    );

    async function retrySingleDelivery(deliveryId: string) {
        retryingDeliveries.add(deliveryId);
        retryingDeliveries = new Set(retryingDeliveries);
        try {
            const ns = namespaceFilter.trim() || 'default';
            unwrap(await api.POST('/v1/namespaces/{namespace}/deliveries/{delivery_id}:retry', {
                params: { path: { namespace: ns, delivery_id: deliveryId } },
            }));
            await fetchDeliveries(currentPage);
        } catch (e: any) {
            error = formatAPIError(e, 'Failed to retry delivery');
        } finally {
            retryingDeliveries.delete(deliveryId);
            retryingDeliveries = new Set(retryingDeliveries);
        }
    }

    async function prepareRetryBatch() {
        preparingRetry = true;
        error = '';
        try {
            await fetchDeliveries(1, true);
        } finally {
            preparingRetry = false;
        }
    }

    async function executeRetry() {
        confirmRetry = false;
        if (!retryId) return;
        const ns = namespaceFilter.trim() || 'default';
        try {
            const res = unwrap(await api.POST('/v1/namespaces/{namespace}/deliveries:retryBatch', {
                params: { path: { namespace: ns } },
                body: { repush_id: retryId },
            }));
            batchStatus = { status: res.status, total: res.total, processed: res.processed, failed: res.failed };
            startPolling();
        } catch (e: any) {
            error = formatAPIError(e, 'Failed to start retry');
        }
    }

    function startPolling() {
        if (pollingTimer) clearInterval(pollingTimer);
        const ns = namespaceFilter.trim() || 'default';
        pollingTimer = setInterval(async () => {
            if (!retryId) { stopPolling(); return; }
            try {
                const res = unwrap(await api.GET('/v1/namespaces/{namespace}/retry-jobs/{job_id}', {
                    params: { path: { namespace: ns, job_id: retryId } },
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

    async function cancelRetryBatch() {
        if (!retryId) return;
        const ns = namespaceFilter.trim() || 'default';
        try {
            await api.POST('/v1/namespaces/{namespace}/retry-jobs/{job_id}:cancel', {
                params: { path: { namespace: ns, job_id: retryId } },
            });
        } catch (e: any) {
            error = formatAPIError(e, 'Failed to cancel retry');
        }
    }

    function onBatchDone() {
        fetchDeliveries(currentPage);
    }

    function formatTimestamp(timestamp: string | null | undefined): string {
        if (!timestamp) return 'N/A';
        const d = new Date(timestamp);
        return isNaN(d.getTime()) ? 'N/A' : d.toLocaleString();
    }

    onMount(() => {
        fetchDeliveries();
    });
</script>

<svelte:head>
    <title>Deliveries | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
    <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
        <div class="mb-6">
            <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-2">
                <div>
                    <h1 class="text-2xl font-bold text-gray-900">Deliveries</h1>
                    <p class="text-sm text-gray-500 mt-0.5">All webhook deliveries in namespace <span class="font-medium text-gray-700">{namespaceFilter.trim() || 'default'}</span></p>
                </div>
                {#if !loading}
                    <div class="flex items-center gap-3">
                        <span class="text-sm text-gray-500">
                            {totalCount} deliver{totalCount !== 1 ? 'ies' : 'y'}
                        </span>
                        {#if totalCount > 0}
                            <button
                                onclick={prepareRetryBatch}
                                disabled={preparingRetry}
                                class="px-3 py-1.5 text-xs font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 disabled:opacity-50 transition"
                            >
                                {preparingRetry ? 'Preparing...' : 'Retry All Matching'}
                            </button>
                        {/if}
                    </div>
                {/if}
            </div>
        </div>

        <div class="bg-white rounded-lg border border-gray-200 p-4 mb-4">
            <div class="flex flex-col sm:flex-row gap-3 flex-wrap">
                <input type="text" placeholder="Namespace" bind:value={namespaceFilter} class="w-32 px-3 py-1.5 border border-gray-300 rounded-lg text-sm" />
                <input type="text" placeholder="Webhook ID" bind:value={webhookIdFilter} class="flex-1 px-3 py-1.5 border border-gray-300 rounded-lg text-sm" />
                <input type="text" placeholder="Event ID" bind:value={eventIdFilter} class="flex-1 px-3 py-1.5 border border-gray-300 rounded-lg text-sm" />
                <select bind:value={statusFilter} class="px-3 py-1.5 border border-gray-300 rounded-lg text-sm">
                    <option value="">All statuses</option>
                    <option value="pending">Pending</option>
                    <option value="sending">Sending</option>
                    <option value="success">Success</option>
                    <option value="failed">Failed</option>
                    <option value="retrying">Retrying</option>
                    <option value="expired">Expired</option>
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
                    oncancel={cancelRetryBatch}
                    ondone={onBatchDone}
                />
            </div>
        {/if}

        {#if error}
            <div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-4">
                <p class="text-sm text-red-700">{error}</p>
            </div>
        {/if}

        {#if loading}
            <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
                <div class="animate-pulse divide-y divide-gray-100">
                    {#each Array(8) as _}
                        <div class="p-4 h-14 bg-gray-50"></div>
                    {/each}
                </div>
            </div>
        {:else if deliveries.length === 0}
            <EmptyState icon="send" title="No deliveries found" description="No deliveries match the current filters." />
        {:else}
            <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
                <div class="overflow-x-auto">
                    <table class="w-full text-sm text-left">
                        <thead>
                            <tr class="border-b border-gray-200 bg-gray-50/50">
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Delivery</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Webhook</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Response</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden md:table-cell">Attempts</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden lg:table-cell">Created</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Actions</th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-gray-100">
                            {#each deliveries as d}
                                <tr class="hover:bg-gray-50 transition">
                                    <td class="px-4 py-3"><CopyableId id={d.delivery_id} href="/deliveries/{d.delivery_id}" truncate={12} /></td>
                                    <td class="px-4 py-3 hidden sm:table-cell"><CopyableId id={d.webhook_id} href="/webhooks/{d.webhook_id}" truncate={12} /></td>
                                    <td class="px-4 py-3">
                                        <div class="flex items-center gap-1.5">
                                            <StatusBadge status={d.status} />
                                        {#if d.error_category && d.error_category !== 'success'}
                                                {@const badge = getCategoryBadge(d.error_category)}
                                                <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                                            {/if}
                                        </div>
                                    </td>
                                    <td class="px-4 py-3 hidden sm:table-cell">
                                        <span class="font-mono text-xs {(d.response_code ?? 0) >= 200 && (d.response_code ?? 0) < 300 ? 'text-green-600' : (d.response_code ?? 0) >= 400 ? 'text-red-600' : 'text-gray-500'}">
                                            {d.response_code || '—'}
                                        </span>
                                    </td>
                                    <td class="px-4 py-3 hidden md:table-cell text-xs text-gray-700">{d.attempt_count}/{d.max_attempts}</td>
                                    <td class="px-4 py-3 hidden lg:table-cell text-xs text-gray-500">{formatTimestamp(d.created_at)}</td>
                                    <td class="px-4 py-3">
                                        <div class="flex items-center gap-2">
                                            {#if d.status === 'failed' || d.status === 'expired'}
                                                <button
                                                    onclick={() => retrySingleDelivery(d.delivery_id)}
                                                    disabled={retryingDeliveries.has(d.delivery_id)}
                                                    class="inline-flex items-center px-2 py-1 text-xs font-medium text-white bg-gray-900 rounded-md hover:bg-gray-800 disabled:opacity-50 transition"
                                                >
                                                    {retryingDeliveries.has(d.delivery_id) ? 'Retrying...' : 'Retry'}
                                                </button>
                                            {/if}
                                            <a href="/deliveries/{d.delivery_id}" class="inline-flex items-center px-2 py-1 text-xs font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 transition">Details</a>
                                        </div>
                                    </td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>

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
    open={confirmRetry}
    title="Retry Deliveries"
    message="This will retry {retryTotal} matching deliver{retryTotal !== 1 ? 'ies' : 'y'}. Each delivery will be re-attempted with its original payload. Continue?"
    confirmLabel="Retry"
    variant="warning"
    onconfirm={executeRetry}
    oncancel={() => { confirmRetry = false; retryId = ''; }}
/>
