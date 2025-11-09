<script lang="ts">
    import { page } from '$app/state';
    import { EventReportsTable, Pagination } from '$lib';
    import favicon from '$lib/assets/favicon.svg';
    import { client } from '$lib/services';
    import { onMount } from 'svelte';
    import type { EventReport, RegisteredEvent } from '../../../../../../proto/webhook_pb.js';

    let eventReports: EventReport[] = $state([]);
    let currentEvent: RegisteredEvent | undefined = $state();
    let loading = $state(true);
    let error = $state('');
    let currentPage = $state(1);
    let totalCount = $state(0);
    let pageSize = $state(20);
    let totalPages = $state(0);

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
        
        // First ensure we have event details
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
                namespace: 'default', // TODO: Get from context or make configurable
                eventName: currentEvent.name, // Use the event name for filtering
                limit: pageSize,
                offset: offset
            };
            
            const res = await client.listEventReports(req);
            eventReports = res.events || [];
            totalCount = res.totalCount || 0;
            totalPages = Math.ceil(totalCount / pageSize);
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
</script>

<div class="min-h-screen bg-gray-50 font-display">
    <main class="p-6">
        <!-- Header -->
        <div class="mb-6">
            <nav class="flex items-center text-sm text-gray-500 mb-4">
                <a href="/events" class="hover:text-blue-600">Events</a>
                <span class="mx-2">/</span>
                <span class="text-gray-900 font-medium">{currentEvent?.name || 'Loading...'} Reports</span>
            </nav>
            
            <div class="flex justify-between items-center pb-1">
                <div>
                    <p class="text-gray-600 mt-1">Event instances for "{currentEvent?.name || 'Loading...'}"</p>
                </div>
                <div class="text-sm text-gray-500">
                    Total: {totalCount} events
            </div>
        </div>

        {#if loading}
            <div class="flex justify-center items-center h-40">
                <span class="material-symbols-outlined animate-spin text-4xl text-primary">
                    <img src={favicon} alt="favicon" class="inline-block w-8 h-8" />
                </span>
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
            />
        {/if}
        </div>
    </main>
</div>