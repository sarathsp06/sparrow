<script lang="ts">
  import { createClient } from "@connectrpc/connect";
  import { createConnectTransport } from "@connectrpc/connect-web";
  import { WebhookService, PushEventRequest, ListEventsRequest } from '../../../../../proto/webhook_pb.js';
  import { onMount } from 'svelte';
  import type { RegisteredEvent } from '../../../../../proto/webhook_pb.js';

  let namespace = 'default';
  let event = '';
  let payload = '';
  let allEvents: RegisteredEvent[] = [];
  let error = '';
  let success = '';

  const transport = createConnectTransport({
    baseUrl: "http://localhost:8080",
  });
  const client = createClient(WebhookService, transport);

  onMount(async () => {
    try {
      const req = new ListEventsRequest({ activeOnly: true });
      const res = await client.listEvents(req);
      allEvents = res.events || [];
    } catch (e) {
      error = 'Failed to load events';
    }
  });

  async function pushEvent() {
    error = '';
    success = '';
    try {
      const req = new PushEventRequest({
        namespace,
        event,
        payload,
      });
      const res = await client.pushEvent(req);
      success = `Event pushed successfully! ${res.webhooksTriggered} webhooks triggered.`;
    } catch (e) {
      error = 'Failed to push event';
    }
  }
</script>

<div class="min-h-screen bg-gradient-to-br from-white via-gray-50 to-gray-100 font-display">
  <div class="p-6 max-w-lg mx-auto">
    <h1 class="text-2xl font-bold mb-4">Push Event</h1>
    <form on:submit|preventDefault={pushEvent} class="bg-white rounded-lg shadow p-6">
      <div class="mb-4">
        <label for="namespace" class="block text-sm font-medium text-gray-700">Namespace</label>
        <input type="text" id="namespace" bind:value={namespace} class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
      </div>
      <div class="mb-4">
        <label for="event" class="block text-sm font-medium text-gray-700">Event</label>
        <select id="event" bind:value={event} class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
          <option value="">Select an event</option>
          {#each allEvents as e}
            <option value={e.name}>{e.name}</option>
          {/each}
        </select>
      </div>
      <div class="mb-4">
        <label for="payload" class="block text-sm font-medium text-gray-700">Payload (JSON)</label>
        <textarea id="payload" bind:value={payload} rows="5" class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"></textarea>
      </div>
      {#if error}
        <p class="text-red-500 text-sm mb-4">{error}</p>
      {/if}
      {#if success}
        <p class="text-green-500 text-sm mb-4">{success}</p>
      {/if}
      <button type="submit" class="w-full bg-primary text-white py-2 px-4 rounded-md hover:bg-primary-dark focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary">Push</button>
    </form>
  </div>
</div>
