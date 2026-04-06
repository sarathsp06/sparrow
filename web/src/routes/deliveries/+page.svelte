<script lang="ts">
    import { deliveryClient } from '$lib/services';
    import { onMount, onDestroy } from 'svelte';
    import type { WebhookDelivery } from '../../../../proto/webhook_pb.js';
    import { WebhookDeliveryStatus } from '../../../../proto/webhook_pb.js';
    import { timestampFromDate } from '@bufbuild/protobuf/wkt';
    import StatusBadge from '$lib/components/StatusBadge.svelte';
    import Pagination from '$lib/components/Pagination.svelte';
    import EmptyState from '$lib/components/EmptyState.svelte';
    import BatchProgress from '$lib/components/BatchProgress.svelte';
    import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

    let deliveries: WebhookDelivery[] = $state([]);
    let loading = $state(true);
    let error = $state('');
    let currentPage = $state(1);
    let totalCount = $state(0);
    let pageSize = $state(25);
    let totalPages = $derived(Math.max(1, Math.ceil(totalCount / pageSize)));

    // Filters
    let namespaceFilter = $state('');
    let webhookIdFilter = $state('');
    let eventIdFilter = $state('');
    let statusFilter = $state('');
    let errorCategoryFilter = $state('');
    let subscriptionIdFilter = $state('');
    let createdAfterFilter = $state('');
    let createdBeforeFilter = $state('');

    // Batch retry state
    let retryId = $state('');
    let batchStatus = $state<{ status: string; total: number; processed: number; failed: number } | undefined>();
    let preparingRetry = $state(false);
    let confirmRetry = $state(false);
    let retryTotal = $state(0);
    let pollingTimer: ReturnType<typeof setInterval> | undefined;

    // Retry single delivery
    let retryingDeliveries: Set<string> = $state(new Set());

    onDestroy(() => {
        if (pollingTimer) clearInterval(pollingTimer);
    });

    function buildRequest(pageNum: number, prepareRetry: boolean = false) {
        const offset = (pageNum - 1) * pageSize;
        const req: Record<string, any> = {
            namespace: namespaceFilter.trim(),
            pagination: { limit: pageSize, offset },
            prepareRetry,
        };

        if (webhookIdFilter.trim()) req.webhookId = webhookIdFilter.trim();
        if (eventIdFilter.trim()) req.eventId = eventIdFilter.trim();
        if (statusFilter) req.status = statusFilter;
        if (errorCategoryFilter) req.errorCategory = errorCategoryFilter;
        if (subscriptionIdFilter.trim()) req.subscriptionId = subscriptionIdFilter.trim();

        if (createdAfterFilter) {
            req.createdAfter = timestampFromDate(new Date(createdAfterFilter));
        }
        if (createdBeforeFilter) {
            req.createdBefore = timestampFromDate(new Date(createdBeforeFilter));
        }

        return req;
    }

    async function fetchDeliveries(pageNum: number = 1) {
        loading = true;
        error = '';

        try {
            const req = buildRequest(pageNum);
            const res = await deliveryClient.listDeliveries(req);
            deliveries = res.deliveries || [];
            totalCount = res.pagination?.totalCount || 0;
            currentPage = pageNum;
        } catch (e: any) {
            console.error('Failed to fetch deliveries:', e);
            error = `Failed to load deliveries: ${e.message}`;
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
        namespaceFilter = '';
        webhookIdFilter = '';
        eventIdFilter = '';
        statusFilter = '';
        errorCategoryFilter = '';
        subscriptionIdFilter = '';
        createdAfterFilter = '';
        createdBeforeFilter = '';
        applyFilters();
    }

    let hasActiveFilters = $derived(
        namespaceFilter.trim() !== '' ||
        webhookIdFilter.trim() !== '' ||
        eventIdFilter.trim() !== '' ||
        statusFilter !== '' ||
        errorCategoryFilter !== '' ||
        subscriptionIdFilter.trim() !== '' ||
        createdAfterFilter !== '' ||
        createdBeforeFilter !== ''
    );

    // Single delivery retry
    async function retrySingleDelivery(deliveryId: string) {
        retryingDeliveries.add(deliveryId);
        retryingDeliveries = new Set(retryingDeliveries);
        try {
            await deliveryClient.retryDelivery({ deliveryId, namespace: namespaceFilter.trim() });
            await fetchDeliveries(currentPage);
        } catch (e: any) {
            error = `Failed to retry delivery: ${e.message}`;
        } finally {
            retryingDeliveries.delete(deliveryId);
            retryingDeliveries = new Set(retryingDeliveries);
        }
    }

    // -- Batch Retry --

    async function prepareRetryBatch() {
        preparingRetry = true;
        error = '';
        try {
            const req = buildRequest(1, true);
            const res = await deliveryClient.listDeliveries(req);
            if (res.retryId) {
                retryId = res.retryId;
                retryTotal = res.pagination?.totalCount || 0;
                confirmRetry = true;
            } else {
                error = 'No matching deliveries to retry.';
            }
        } catch (e: any) {
            error = `Failed to prepare retry: ${e.message}`;
        } finally {
            preparingRetry = false;
        }
    }

    async function executeRetry() {
        confirmRetry = false;
        if (!retryId) return;
        try {
            const res = await deliveryClient.retryDeliveries({ retryId });
            batchStatus = { status: res.status, total: res.total, processed: 0, failed: 0 };
            startPolling();
        } catch (e: any) {
            error = `Failed to start retry: ${e.message}`;
        }
    }

    function startPolling() {
        if (pollingTimer) clearInterval(pollingTimer);
        pollingTimer = setInterval(async () => {
            if (!retryId) { stopPolling(); return; }
            try {
                const res = await deliveryClient.getRetryStatus({ retryId });
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

    async function cancelRetryBatch() {
        if (!retryId) return;
        try {
            await deliveryClient.cancelRetry({ retryId });
        } catch (e: any) {
            error = `Failed to cancel retry: ${e.message}`;
        }
    }

    function onBatchDone() {
        fetchDeliveries(currentPage);
    }

    function formatTimestamp(timestamp: any): string {
        if (!timestamp) return 'N/A';
        const seconds = timestamp.seconds ? Number(timestamp.seconds) : Number(timestamp);
        if (isNaN(seconds)) return 'N/A';
        return new Date(seconds * 1000).toLocaleString();
    }

    function getCategoryBadge(category: string): { label: string; classes: string } {
        switch (category) {
            case 'client_error': return { label: '4xx', classes: 'bg-orange-50 text-orange-700 border-orange-200' };
            case 'server_error': return { label: '5xx', classes: 'bg-red-50 text-red-700 border-red-200' };
            case 'timeout': return { label: 'Timeout', classes: 'bg-yellow-50 text-yellow-700 border-yellow-200' };
            case 'dns_error': return { label: 'DNS', classes: 'bg-purple-50 text-purple-700 border-purple-200' };
            case 'tls_error': return { label: 'TLS', classes: 'bg-purple-50 text-purple-700 border-purple-200' };
            case 'connection_refused': return { label: 'Conn Refused', classes: 'bg-purple-50 text-purple-700 border-purple-200' };
            case 'network_error': return { label: 'Network', classes: 'bg-purple-50 text-purple-700 border-purple-200' };
            default: return { label: category || 'Unknown', classes: 'bg-gray-50 text-gray-700 border-gray-200' };
        }
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
        <!-- Header -->
        <div class="mb-6">
            <div class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-2">
                <div>
                    <h1 class="text-2xl font-bold text-gray-900">Deliveries</h1>
                    <p class="text-sm text-gray-500 mt-0.5">All webhook deliveries across your system</p>
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

        <!-- Filters -->
        <div class="bg-white rounded-lg border border-gray-200 p-4 mb-4">
            <div class="flex flex-wrap items-end gap-3">
                <div class="w-full sm:w-36">
                    <label for="ns-filter" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Namespace</label>
                    <input
                        id="ns-filter"
                        type="text"
                        placeholder="All"
                        bind:value={namespaceFilter}
                        onkeydown={(e) => e.key === 'Enter' && applyFilters()}
                        class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
                    />
                </div>
                <div class="w-full sm:w-32">
                    <label for="status-filter" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Status</label>
                    <select
                        id="status-filter"
                        bind:value={statusFilter}
                        onchange={applyFilters}
                        class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
                    >
                        <option value="">All</option>
                        <option value="pending">Pending</option>
                        <option value="sending">Sending</option>
                        <option value="success">Success</option>
                        <option value="failed">Failed</option>
                        <option value="retrying">Retrying</option>
                        <option value="expired">Expired</option>
                    </select>
                </div>
                <div class="w-full sm:w-36">
                    <label for="error-filter" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Error Category</label>
                    <select
                        id="error-filter"
                        bind:value={errorCategoryFilter}
                        onchange={applyFilters}
                        class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
                    >
                        <option value="">All</option>
                        <option value="client_error">Client (4xx)</option>
                        <option value="server_error">Server (5xx)</option>
                        <option value="timeout">Timeout</option>
                        <option value="dns_error">DNS</option>
                        <option value="tls_error">TLS</option>
                        <option value="connection_refused">Conn Refused</option>
                        <option value="network_error">Network</option>
                    </select>
                </div>
                <div class="w-full sm:w-44">
                    <label for="webhook-filter" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Webhook ID</label>
                    <input
                        id="webhook-filter"
                        type="text"
                        placeholder="Filter by webhook..."
                        bind:value={webhookIdFilter}
                        onkeydown={(e) => e.key === 'Enter' && applyFilters()}
                        class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900 font-mono"
                    />
                </div>
                <div class="w-full sm:w-44">
                    <label for="event-filter" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Event ID</label>
                    <input
                        id="event-filter"
                        type="text"
                        placeholder="Filter by event..."
                        bind:value={eventIdFilter}
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
                    label="Retry Deliveries"
                    oncancel={cancelRetryBatch}
                    ondone={onBatchDone}
                />
            </div>
        {/if}

        {#if error}
            <div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-4">
                <div class="flex items-start justify-between">
                    <div class="flex items-start gap-3">
                        <svg class="w-5 h-5 text-red-500 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
                        </svg>
                        <p class="text-sm font-medium text-red-800">{error}</p>
                    </div>
                    <button onclick={() => { error = ''; }} class="text-red-400 hover:text-red-600 ml-3 shrink-0" aria-label="Dismiss error">
                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                        </svg>
                    </button>
                </div>
            </div>
        {/if}

        {#if loading}
            <!-- Loading skeleton -->
            <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
                <div class="overflow-x-auto">
                    <table class="w-full text-sm text-left">
                        <thead>
                            <tr class="border-b border-gray-200 bg-gray-50/50">
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Delivery ID</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Webhook</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Event</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden md:table-cell">Attempts</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden lg:table-cell">Last Attempt</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider"></th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-gray-100">
                            {#each Array(5) as _}
                                <tr class="animate-pulse">
                                    <td class="px-4 py-3"><div class="h-4 bg-gray-200 rounded w-28"></div></td>
                                    <td class="px-4 py-3 hidden sm:table-cell"><div class="h-4 bg-gray-100 rounded w-28"></div></td>
                                    <td class="px-4 py-3 hidden sm:table-cell"><div class="h-4 bg-gray-100 rounded w-28"></div></td>
                                    <td class="px-4 py-3"><div class="h-4 bg-gray-200 rounded w-16"></div></td>
                                    <td class="px-4 py-3 hidden md:table-cell"><div class="h-4 bg-gray-100 rounded w-8"></div></td>
                                    <td class="px-4 py-3 hidden lg:table-cell"><div class="h-4 bg-gray-100 rounded w-32"></div></td>
                                    <td class="px-4 py-3"><div class="h-4 bg-gray-100 rounded w-12"></div></td>
                                </tr>
                            {/each}
                        </tbody>
                    </table>
                </div>
            </div>
        {:else if deliveries.length === 0}
            <EmptyState
                icon="send"
                title="No deliveries found"
                description={hasActiveFilters ? "No deliveries match your current filters. Try adjusting or clearing them." : "No deliveries have been made yet. Push an event to create deliveries."}
            />
        {:else}
            <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
                <div class="overflow-x-auto">
                    <table class="w-full text-sm text-left">
                        <thead>
                            <tr class="border-b border-gray-200 bg-gray-50/50">
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Delivery ID</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Webhook</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Event</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden md:table-cell">Attempts</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden lg:table-cell">Last Attempt</th>
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider"></th>
                            </tr>
                        </thead>
                        <tbody class="divide-y divide-gray-100">
                            {#each deliveries as delivery}
                                <tr class="hover:bg-gray-50 transition">
                                    <td class="px-4 py-3">
                                        <a href="/deliveries/{delivery.deliveryId}" class="font-mono text-xs text-blue-600 hover:text-blue-800 hover:underline transition">
                                            {delivery.deliveryId.substring(0, 12)}...
                                        </a>
                                    </td>
                                    <td class="px-4 py-3 hidden sm:table-cell">
                                        <a href="/webhooks/{delivery.webhookId}" class="font-mono text-xs text-blue-600 hover:text-blue-800 hover:underline transition">
                                            {delivery.webhookId.substring(0, 12)}...
                                        </a>
                                    </td>
                                    <td class="px-4 py-3 font-mono text-xs text-gray-700 hidden sm:table-cell">
                                        {delivery.eventId.substring(0, 12)}...
                                    </td>
                                    <td class="px-4 py-3">
                                        <div class="flex items-center gap-1.5">
                                            <StatusBadge status={delivery.status} />
                                            {#if delivery.errorCategory && delivery.errorCategory !== '' && delivery.errorCategory !== 'success'}
                                                {@const badge = getCategoryBadge(delivery.errorCategory)}
                                                <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                                            {/if}
                                        </div>
                                    </td>
                                    <td class="px-4 py-3 text-xs text-gray-700 hidden md:table-cell">
                                        {delivery.attemptCount}/{delivery.maxAttempts}
                                    </td>
                                    <td class="px-4 py-3 text-xs text-gray-500 hidden lg:table-cell">
                                        {formatTimestamp(delivery.lastAttemptedAt)}
                                    </td>
                                    <td class="px-4 py-3">
                                        <div class="flex items-center gap-2">
                                            {#if delivery.status === WebhookDeliveryStatus.DELIVERY_FAILED || delivery.status === WebhookDeliveryStatus.DELIVERY_EXPIRED}
                                                <button
                                                    onclick={() => retrySingleDelivery(delivery.deliveryId)}
                                                    disabled={retryingDeliveries.has(delivery.deliveryId)}
                                                    class="inline-flex items-center px-2 py-1 text-xs font-medium text-white bg-gray-900 rounded-md hover:bg-gray-800 disabled:opacity-50 transition"
                                                >
                                                    {retryingDeliveries.has(delivery.deliveryId) ? 'Retrying...' : 'Retry'}
                                                </button>
                                            {/if}
                                            <a
                                                href="/deliveries/{delivery.deliveryId}"
                                                class="inline-flex items-center px-2 py-1 text-xs font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 transition"
                                            >
                                                Details
                                            </a>
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
                itemLabel="deliveries"
            />
        {/if}
    </main>
</div>

<!-- Confirm Retry Dialog -->
<ConfirmDialog
    open={confirmRetry}
    title="Retry Deliveries"
    message="This will retry {retryTotal} matching deliver{retryTotal !== 1 ? 'ies' : 'y'}. Each delivery will be re-attempted with its original payload. Continue?"
    confirmLabel="Retry"
    variant="warning"
    onconfirm={executeRetry}
    oncancel={() => { confirmRetry = false; retryId = ''; }}
/>
