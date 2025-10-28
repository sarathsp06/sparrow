<script lang="ts">
  import { createClient } from "@connectrpc/connect";
  import { createConnectTransport } from "@connectrpc/connect-web";
  import { onMount } from 'svelte';
  import type { RegisteredEvent } from '../../../../proto/webhook_pb.js';
  import { WebhookService } from '../../../../proto/webhook_pb.js';

  let events: RegisteredEvent[] = [];
  let loading = true;
  let error = '';
  const transport = createConnectTransport({
    baseUrl: "http://localhost:8080",
  });
  const client = createClient(WebhookService, transport);

  async function fetchEvents() {
    loading = true;
    error = '';
    try {
      const req = new ListEventsRequest({ activeOnly: false });
      const res = await client.listEvents(req);
      events = res.events || [];
    } catch (e) {
      error = 'Failed to load events';
    }
    loading = false;
  }

  onMount(fetchEvents);

  async function deleteEvent(eventId: string) {
    try {
      const req =({ eventId });
      await client.deleteEvent(req);
      await fetchEvents(); // Refresh the list
    } catch (e) {
      error = `Failed to delete event.`;
    }
  }
</script>

<div class="min-h-screen bg-white font-display">
  <div class="flex items-center bg-white p-6 pb-2 justify-between sticky top-0 z-10 border-b border-gray-200 shadow-sm">
    <h1 class="text-gray-900 text-xl font-bold leading-tight tracking-tight flex items-center gap-2">
      <span class="material-symbols-outlined text-gray-500 text-2xl">event</span>
      Event Management
    </h1>
    <a href="/events/register" class="bg-gray-800 text-white px-4 py-2 rounded-md font-medium shadow hover:bg-gray-700 transition">+ Register Event</a>
  </div>

  <div class="p-6 max-w-2xl mx-auto">
    {#if loading}
      <div class="flex justify-center items-center h-24">
        <span class="material-symbols-outlined animate-spin text-3xl text-gray-400">autorenew</span>
        <span class="ml-3 text-base text-gray-600">Loading...</span>
      </div>
    {:else if error}
      <div class="bg-red-50 rounded-md p-3 text-red-600 shadow mb-4 flex items-center gap-2">
        <span class="material-symbols-outlined">error</span>{error}
      </div>
    {:else if events.length === 0}
      <div class="bg-gray-50 rounded-md p-3 text-gray-500 shadow mb-4 flex items-center gap-2">
        <span class="material-symbols-outlined text-gray-400">info</span>No events found.
      </div>
    {:else}
      <div class="grid gap-4">
        {#each events as event}
          <div class="event-card flex flex-col md:flex-row justify-between items-center bg-white rounded-md shadow-sm p-4 hover:shadow-md transition">
            <div class="flex flex-col gap-1">
              <div class="flex items-center gap-2">
                <span class="material-symbols-outlined text-gray-500 text-xl">label</span>
                <span class="font-semibold text-gray-900">{event.name}</span>
              </div>
              <span class="text-gray-500 text-sm">{event.description}</span>
            </div>
            <div class="flex gap-2">
              <a href={`/events/${event.eventId}/update`} class="text-gray-700 font-medium px-3 py-1 rounded hover:underline">Update</a>
              <button on:click={() => deleteEvent(event.eventId)} class="text-red-500 font-medium px-3 py-1 rounded hover:underline">Delete</button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</div>
<style>
    .event-card {
      background: #fff;
      border: 1px solid #ececec;
      box-shadow: 0 1px 4px 0 rgba(60, 60, 100, 0.04);
      transition: box-shadow 0.2s, transform 0.2s;
    }
    .event-card:hover {
      box-shadow: 0 4px 12px 0 rgba(60, 60, 100, 0.08);
      transform: translateY(-1px) scale(1.01);
    }
    .animate-spin {
      animation: spin 1s linear infinite;
    }
    @keyframes spin {
      100% { transform: rotate(360deg); }
    }
</style>
