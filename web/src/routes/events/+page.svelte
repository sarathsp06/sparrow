<script lang="ts">
  import { onMount } from 'svelte';
  import { ListEventsRequestSchema } from '../../../../proto/webhook_pb.js';

  let events = [];
  let loading = true;
  let error = '';

  async function fetchEvents() {
    loading = true;
    error = '';
    try {
      const req = ListEventsRequestSchema.create({
        activeOnly: false
      });
      const res = await fetch('http://localhost:8080/events/list', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(req)
      });
      const data = await res.json();
      events = data.events || [];
    } catch (e) {
      error = 'Failed to load events';
    }
    loading = false;
  }

  onMount(fetchEvents);
</script>

<!-- Paper-inspired Light Theme Events Page -->
<div class="min-h-screen bg-gradient-to-br from-white via-gray-50 to-gray-100 font-display relative overflow-hidden">
  <div class="flex items-center bg-white/90 backdrop-blur-md p-6 pb-2 justify-between sticky top-0 z-10 border-b border-gray-200 shadow-sm">
    <h1 class="text-gray-900 text-2xl font-extrabold leading-tight tracking-tight flex items-center gap-2">
      <span class="material-symbols-outlined text-primary text-3xl">event</span>
      Event List
    </h1>
    <a href="/events/push" class="bg-primary/90 text-white px-5 py-2 rounded-lg font-semibold shadow hover:bg-primary transition">+ Push New Event</a>
  </div>
  <div class="p-6 max-w-3xl mx-auto">
    {#if loading}
      <div class="flex justify-center items-center h-32">
        <span class="material-symbols-outlined animate-spin text-4xl text-primary">autorenew</span>
        <span class="ml-3 text-lg text-gray-700">Loading...</span>
      </div>
    {:else if error}
      <div class="bg-red-50 rounded-lg p-4 text-red-600 shadow mb-4 flex items-center gap-2">
        <span class="material-symbols-outlined">error</span>{error}
      </div>
    {:else if events.length === 0}
      <div class="bg-white rounded-lg p-4 text-gray-600 shadow mb-4 flex items-center gap-2">
        <span class="material-symbols-outlined text-gray-400">info</span>No events found.
      </div>
    {:else}
      <div class="grid gap-6">
        {#each events as evt}
          <div class="paper-card flex flex-col md:flex-row justify-between items-center bg-white rounded-xl shadow-md p-6 hover:shadow-lg transition">
            <div class="flex flex-col gap-1">
              <div class="flex items-center gap-2">
                <span class="material-symbols-outlined text-primary text-2xl">event</span>
                <span class="font-bold text-gray-900">{evt.name ?? evt.eventId}</span>
              </div>
              <span class="text-gray-500 text-sm">{evt.eventId}</span>
            </div>
            <a href={`/events/${evt.eventId}`} class="bg-primary/10 text-primary font-semibold px-4 py-2 rounded-lg shadow hover:bg-primary/20 transition ml-4">Details</a>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <style>
    .paper-card {
      background: #fff;
      border: 1px solid #f3f3f3;
      box-shadow: 0 2px 8px 0 rgba(60, 60, 100, 0.06);
      transition: box-shadow 0.2s, transform 0.2s;
    }
    .paper-card:hover {
      box-shadow: 0 8px 24px 0 rgba(60, 60, 100, 0.10);
      transform: translateY(-2px) scale(1.02);
    }
    .animate-spin {
      animation: spin 1s linear infinite;
    }
    @keyframes spin {
      100% { transform: rotate(360deg); }
    }
    .bg-primary { background-color: #2563eb; }
    .text-primary { color: #2563eb; }
  </style>
</div>
