<script lang="ts">
  import { createClient } from "@connectrpc/connect";
  import { createConnectTransport } from "@connectrpc/connect-web";
  import { WebhookService, RegisterWebhookRequest, ListEventsRequest } from '../../../../../proto/webhook_pb.js';
  import { goto } from '$app/navigation';
  import { onMount } from 'svelte';
  import type { RegisteredEvent } from '../../../../../proto/webhook_pb.js';

  let namespace = 'default';
  let events: string[] = [];
  let url = '';
  let description = '';
  let active = true;
  let allEvents: RegisteredEvent[] = [];
  let error = '';

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

  async function registerWebhook() {
    error = '';
    try {
      const req = new RegisterWebhookRequest({
        namespace,
        events,
        url,
        description,
        active,
      });
      await client.registerWebhook(req);
      goto('/webhooks');
    } catch (e) {
      error = 'Failed to register webhook';
    }
  }
</script>

<div class="min-h-screen bg-gradient-to-br from-white via-gray-50 to-gray-100 font-display">
  <div class="p-6 max-w-lg mx-auto">
    <h1 class="text-2xl font-bold mb-4">Register New Webhook</h1>
    <form on:submit|preventDefault={registerWebhook} class="bg-white rounded-lg shadow p-6">
      <div class="mb-4">
        <label for="namespace" class="block text-sm font-medium text-gray-700">Namespace</label>
        <input type="text" id="namespace" bind:value={namespace} class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
      </div>
      <div class="mb-4">
        <label for="url" class="block text-sm font-medium text-gray-700">URL</label>
        <input type="text" id="url" bind:value={url} class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
      </div>
      <div class="mb-4">
        <label for="description" class="block text-sm font-medium text-gray-700">Description</label>
        <input type="text" id="description" bind:value={description} class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
      </div>
      <div class="mb-4">
        <label class="block text-sm font-medium text-gray-700">Events</label>
        <div class="mt-2 grid grid-cols-2 gap-2">
          {#each allEvents as event}
            <label class="flex items-center">
              <input type="checkbox" value={event.name} bind:group={events} class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-offset-0 focus:ring-indigo-200 focus:ring-opacity-50">
              <span class="ml-2 text-sm text-gray-600">{event.name}</span>
            </label>
          {/each}
        </div>
      </div>
      <div class="mb-4">
        <label class="flex items-center">
          <input type="checkbox" bind:checked={active} class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-offset-0 focus:ring-indigo-200 focus:ring-opacity-50">
          <span class="ml-2 text-sm text-gray-600">Active</span>
        </label>
      </div>
      {#if error}
        <p class="text-red-500 text-sm mb-4">{error}</p>
      {/if}
      <button type="submit" class="w-full bg-primary text-white py-2 px-4 rounded-md hover:bg-primary-dark focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary">Register</button>
    </form>
  </div>
</div>
