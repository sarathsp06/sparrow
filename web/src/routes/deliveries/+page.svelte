<script lang="ts">
    import { api, unwrap } from '$lib/services';
    import { getCategoryBadge, ERROR_CATEGORIES, formatAPIError, timeAgo } from '$lib/utils';
    import { namespaceStore } from '$lib/namespace.svelte';
    import { pulseStore } from '$lib/pulse.svelte';
    import { onDestroy } from 'svelte';
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

    let live = $state(false);
    let liveTimer: ReturnType<typeof setInterval> | undefined;

    onDestroy(() => {
        if (pollingTimer) clearInterval(pollingTimer);
        if (liveTimer) clearInterval(liveTimer);
    });

    function startLive() {
        stopLive();
        live = true;
        liveTimer = setInterval(() => { fetchDeliveries(1, false, true); pulseStore.ping(); }, 5000);
    }
    function stopLive() {
        live = false;
        if (liveTimer) { clearInterval(liveTimer); liveTimer = undefined; }
    }
    function toggleLive() {
        if (live) stopLive();
        else startLive();
    }

    async function fetchDeliveries(pageNum: number = 1, prepareRetry: boolean = false, silent: boolean = false) {
        loading = !prepareRetry && !silent;
        if (!prepareRetry) error = '';

        const ns = namespaceStore.value;
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
            const ns = namespaceStore.value;
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
        const ns = namespaceStore.value;
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
        const ns = namespaceStore.value;
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
        const ns = namespaceStore.value;
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

    $effect(() => {
        namespaceStore.value; // refetch when the active namespace changes
        fetchDeliveries(1);
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
            <p class="text-sm text-muted mt-1">All webhook deliveries in namespace <span class="mono text-text">{namespaceStore.value}</span></p>
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
                <span class="text-sm text-muted mono tnum">
                    {totalCount} deliver{totalCount !== 1 ? 'ies' : 'y'}
                </span>
                {#if totalCount > 0}
                    <button onclick={prepareRetryBatch} disabled={preparingRetry} class="btn btn-ghost !px-3 !py-1.5 !text-xs">
                        {preparingRetry ? 'Preparing…' : 'Retry all matching'}
                    </button>
                {/if}
            {/if}
        </div>
    </div>

    <div class="panel p-4 mb-4">
        <div class="flex flex-col sm:flex-row gap-3 flex-wrap">
            <input type="text" placeholder="Webhook ID" bind:value={webhookIdFilter} class="input flex-1" />
            <input type="text" placeholder="Event ID" bind:value={eventIdFilter} class="input flex-1" />
            <select bind:value={statusFilter} class="select sm:w-44">
                <option value="">All statuses</option>
                <option value="pending">Pending</option>
                <option value="sending">Sending</option>
                <option value="success">Success</option>
                <option value="failed">Failed</option>
                <option value="retrying">Retrying</option>
                <option value="expired">Expired</option>
            </select>
            <button onclick={applyFilters} class="btn btn-beacon">Apply</button>
            {#if hasActiveFilters}
                <button onclick={clearFilters} class="btn btn-ghost">Clear</button>
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
        <div class="panel p-4 mb-4" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
            <p class="text-sm" style="color:var(--color-bad)">{error}</p>
        </div>
    {/if}

    {#if loading}
        <div class="panel overflow-hidden">
            <div class="animate-pulse">
                {#each Array(8) as _}
                    <div class="row-line h-14 bg-white/[0.015]"></div>
                {/each}
            </div>
        </div>
    {:else if deliveries.length === 0}
        <div class="panel">
            <EmptyState icon="send" title="No deliveries found" description="No deliveries match the current filters." />
        </div>
    {:else}
        <div class="panel overflow-hidden">
            <div class="overflow-x-auto">
                <table class="w-full text-left">
                    <thead>
                        <tr class="border-b border-line">
                            <th class="th">Delivery</th>
                            <th class="th hidden sm:table-cell">Webhook</th>
                            <th class="th">Status</th>
                            <th class="th hidden sm:table-cell">Response</th>
                            <th class="th hidden md:table-cell">Attempts</th>
                            <th class="th hidden lg:table-cell">Created</th>
                            <th class="th"></th>
                        </tr>
                    </thead>
                    <tbody>
                        {#each deliveries as d}
                            <tr class="row-line row-hover transition">
                                <td class="td"><CopyableId id={d.delivery_id} href="/deliveries/{d.delivery_id}" truncate={12} /></td>
                                <td class="td hidden sm:table-cell"><CopyableId id={d.webhook_id} href="/webhooks/{d.webhook_id}" truncate={12} /></td>
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
                                    <span class="mono tnum text-xs" style="color:{(d.response_code ?? 0) >= 200 && (d.response_code ?? 0) < 300 ? 'var(--color-ok)' : (d.response_code ?? 0) >= 400 ? 'var(--color-bad)' : 'var(--color-muted)'}">
                                        {d.response_code || '—'}
                                    </span>
                                </td>
                                <td class="td hidden md:table-cell mono tnum text-xs text-muted">{d.attempt_count}/{d.max_attempts}</td>
                                <td class="td hidden lg:table-cell"><span class="mono tnum text-muted text-xs" title={formatTimestamp(d.created_at)}>{timeAgo(d.created_at)}</span></td>
                                <td class="td">
                                    <div class="flex items-center gap-2">
                                        {#if d.status === 'failed' || d.status === 'expired'}
                                            <button
                                                onclick={() => retrySingleDelivery(d.delivery_id)}
                                                disabled={retryingDeliveries.has(d.delivery_id)}
                                                class="btn btn-ghost !px-2 !py-1 !text-xs"
                                            >
                                                {retryingDeliveries.has(d.delivery_id) ? 'Retrying…' : 'Retry'}
                                            </button>
                                        {/if}
                                        <a href="/deliveries/{d.delivery_id}" class="btn btn-ghost !px-2 !py-1 !text-xs">Details</a>
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

<ConfirmDialog
    open={confirmRetry}
    title="Retry Deliveries"
    message="This will retry {retryTotal} matching deliver{retryTotal !== 1 ? 'ies' : 'y'}. Each delivery will be re-attempted with its original payload. Continue?"
    confirmLabel="Retry"
    variant="warning"
    onconfirm={executeRetry}
    oncancel={() => { confirmRetry = false; retryId = ''; }}
/>
