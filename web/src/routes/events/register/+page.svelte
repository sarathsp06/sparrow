<script lang="ts">
  import { createClient } from "@connectrpc/connect";
  import { createConnectTransport } from "@connectrpc/connect-web";
  import  { WebhookService } from '../../../../../proto/webhook_pb.js';
  import { goto } from '$app/navigation';

  let name = '';
  let description = '';
  let schema = '';
  let active = true;
  let error = '';

  const transport = createConnectTransport({
    baseUrl: "http://localhost:8080",
  });
  const client = createClient(WebhookService, transport);

  async function registerEvent() {
    error = '';
    try {
      const req = {
        name,
        description,
        schema,
        active,
      };
      await client.registerEvent(req);
      goto('/events');
    } catch (e) {
      error = 'Failed to register event';
    }
  }
</script>

<div class="min-h-screen bg-gradient-to-br from-white via-gray-50 to-gray-100 font-display">
  <div class="p-6 max-w-lg mx-auto">
    <h1 class="text-2xl font-bold mb-4">Register New Event</h1>
    <form on:submit|preventDefault={registerEvent} class="bg-white rounded-lg shadow p-6">
      <div class="mb-4">
        <label for="name" class="block text-sm font-medium text-gray-700">Name</label>
        <input type="text" id="name" bind:value={name} class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
      </div>
      <div class="mb-4">
        <label for="description" class="block text-sm font-medium text-gray-700">Description</label>
        <textarea id="description" bind:value={description} rows="3" class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"></textarea>
      </div>
      <div class="mb-4">
        <label for="schema" class="block text-sm font-medium text-gray-700">Schema (JSON)</label>
        <textarea id="schema" bind:value={schema} rows="5" class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"></textarea>
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
      <button type="submit" class="bg-blue-200 text-black py-2 px-4 rounded-md hover:bg-blue-300">Register</button>
    </form>
  </div>
</div>
