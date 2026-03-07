<script lang="ts">
    import { page } from '$app/state';
    import { EventReportsTable, Pagination } from '$lib';
    import { eventClient as client } from '$lib/services';
    import { onMount, untrack } from 'svelte';
    import { namespaceState } from '$lib/namespace.svelte';
    import type { EventReport, RegisteredEvent } from '../../../../../../proto/webhook_pb.js';

    let eventReports: EventReport[] = $state([]);
    let currentEvent: RegisteredEvent | undefined = $state();
    let loading = $state(true);
    let error = $state('');
    let currentPage = $state(1);
    let totalCount = $state(0);
    let pageSize = $state(20);
    let totalPages = $derived(Math.max(1, Math.ceil(totalCount / pageSize)));

    const eventId = page.params.eventId || '';

    async function fetchEventDetails() {
        try {
            const req = { activeOnly: false };
            const res = await client.listEvents(req);
            currentEvent = res.events?.find(event => event.eventId === eventId);
            
            if (!currentEvent) {
                error = 'Event not found';
                return;
            }
        } catch (e: any) {
            console.error('Failed to fetch event details:', e);
            error = `Failed to load event details: ${e.message}`;
        }
    }

    async function fetchEventReports(pageNum: number = 1) {
        loading = true;
        error = '';
        
        if (!currentEvent) {
            await fetchEventDetails();
            if (!currentEvent) {
                loading = false;
                return;
            }
        }

        try {
            const offset = (pageNum - 1) * pageSize;
            const req = {
                namespace: namespaceState.current,
                eventName: currentEvent.name,
                pagination: {
                    limit: pageSize,
                    offset: offset
                }
            };
            
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

    onMount(() => {
        fetchEventReports();
    });

    // Re-fetch when namespace changes
    $effect(() => {
        const ns = namespaceState.current;
        untrack(() => {
            currentPage = 1;
            fetchEventReports(1);
        });
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
                        Instances of "{currentEvent?.name || 'Loading...'}" in namespace
                        <span class="font-semibold text-gray-700">{namespaceState.current}</span>
                    </p>
                </div>
                {#if !loading}
                    <div class="text-sm text-gray-500">
                        {totalCount} report{totalCount !== 1 ? 's' : ''}
                    </div>
                {/if}
            </div>
        </div>

        {#if loading}
            <!-- Loading skeleton -->
            <div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
                <div class="overflow-x-auto">
                    <table class="w-full text-sm text-left">
                        <thead>
                            <tr class="border-b border-gray-200 bg-gray-50/50">
                                <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Event ID</th>
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
