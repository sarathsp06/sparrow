<script lang="ts">
	import { goto } from '$app/navigation';
	import { eventClient, webhookClient as client } from '$lib/services';
	import { onMount } from 'svelte';
	import type {
	  RegisteredEvent
	} from '../../../../../proto/webhook_pb.js';

	let namespace = $state('default');
	let events: string[] = $state([]);
	let url = $state('');
	let description = $state('');
	let active = $state(true);
	let allEvents: RegisteredEvent[] = $state([]);
	let error = $state('');

	// HTTP Configuration
	let showAdvanced = $state(false);
	let maxRetries = $state(3);
	let retryBackoffSeconds = $state(60);
	let captureResponseBody = $state(false);
	let followRedirects = $state(true);
	let verifySSL = $state(true);
	let requestTimeoutSeconds = $state(30);
	let expectedStatusCodes = $state('200,201,202,204');
	let webhookSecret = $state('');
	let userAgent = $state('Sparrow-Webhook/1.0');
	let contentType = $state('application/json');
	
	// HTTP Headers
	let headers: { key: string; value: string }[] = $state([{ key: '', value: '' }]);

	onMount(async () => {
		try {
			const req = { activeOnly: true };
			const res = await eventClient.listEvents(req);
			allEvents = res.events || [];
		} catch (e: any) {
			error = `Failed to load events: ${e.message}`;
		}
	});

	function addHeader() {
		headers = [...headers, { key: '', value: '' }];
	}

	function removeHeader(index: number) {
		headers = headers.filter((_, i) => i !== index);
	}

	async function registerWebhook(e: Event) {
    e.preventDefault();
		error = '';
		try {
			// Convert headers array to map
			const headersMap: Record<string, string> = {};
			headers.forEach(header => {
				if (header.key.trim() && header.value.trim()) {
					headersMap[header.key.trim()] = header.value.trim();
				}
			});

			// Parse expected status codes
			const statusCodes = expectedStatusCodes
				.split(',')
				.map(code => parseInt(code.trim()))
				.filter(code => !isNaN(code) && code >= 100 && code < 600);

			const req = {
				namespace,
				events,
				url,
				description,
				active,
				headers: headersMap,
				httpConfig: showAdvanced ? {
					maxRetries,
					retryBackoffSeconds,
					captureResponseBody,
					followRedirects,
					verifySsl: verifySSL,
					requestTimeoutSeconds,
					expectedStatusCodes: statusCodes,
					webhookSecret: webhookSecret.trim() || undefined,
					userAgent: userAgent.trim() || 'Sparrow-Webhook/0.1.0',
					contentType: contentType.trim() || 'application/json'
				} : undefined
			};
			
			await client.registerWebhook(req);
			goto('/webhooks');
		} catch (e: any) {
			error = `Failed to register webhook: ${e.message}`;
		}
	}
</script>

<div class="min-h-screen bg-linear-to-br from-white via-gray-50 to-gray-100 font-display">
  <div class="p-6 max-w-2xl mx-auto">
    <h1 class="text-2xl font-bold mb-4">Register New Webhook</h1>
    <form onsubmit={registerWebhook} class="bg-white rounded-lg shadow p-6 space-y-6">
      
      <!-- Basic Configuration -->
      <div class="space-y-4">
        <h2 class="text-lg font-semibold text-gray-900 border-b pb-2">Basic Configuration</h2>
        
        <div>
          <label for="namespace" class="block text-sm font-medium text-gray-700">Namespace</label>
          <input type="text" id="namespace" bind:value={namespace} 
                 class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
        </div>
        
        <div>
          <label for="url" class="block text-sm font-medium text-gray-700">URL</label>
          <input type="url" id="url" bind:value={url} 
                 class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
                 placeholder="https://api.example.com/webhooks">
        </div>
        
        <div>
          <label for="description" class="block text-sm font-medium text-gray-700">Description</label>
          <input type="text" id="description" bind:value={description} 
                 class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50"
                 placeholder="Optional description for this webhook">
        </div>
        
        <div>
          <label for="events" class="block text-sm font-medium text-gray-700 mb-2">Events</label>
          <div class="grid grid-cols-2 gap-2 max-h-48 overflow-y-auto border border-gray-200 rounded-md p-3">
            {#each allEvents as event}
              <label class="flex items-center">
                <input  type="checkbox" value={event.name} bind:group={events} 
                       class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-offset-0 focus:ring-indigo-200 focus:ring-opacity-50">
                <span class="ml-2 text-sm text-gray-600">{event.name}</span>
              </label>
            {/each}
          </div>
        </div>
        
        <div>
          <label class="flex items-center">
            <input type="checkbox" bind:checked={active} 
                   class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-offset-0 focus:ring-indigo-200 focus:ring-opacity-50">
            <span class="ml-2 text-sm text-gray-600">Active</span>
          </label>
        </div>
      </div>

      <!-- HTTP Headers -->
      <div class="space-y-4">
        <h2 class="text-lg font-semibold text-gray-900 border-b pb-2">HTTP Headers</h2>
        {#each headers as header, index}
          <div class="flex space-x-2">
            <input type="text" bind:value={header.key} placeholder="Header Name"
                   class="flex-1 rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
            <input type="text" bind:value={header.value} placeholder="Header Value"
                   class="flex-1 rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
            {#if headers.length > 1}
              <button type="button" onclick={() => removeHeader(index)}
                      class="px-3 py-2 bg-red-100 text-red-700 rounded-md hover:bg-red-200">Remove</button>
            {/if}
          </div>
        {/each}
        <button type="button" onclick={addHeader}
                class="text-sm text-indigo-600 hover:text-indigo-500">+ Add Header</button>
      </div>

      <!-- Advanced HTTP Configuration -->
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold text-gray-900">HTTP Configuration</h2>
          <button type="button" onclick={() => showAdvanced = !showAdvanced}
                  class="text-sm text-indigo-600 hover:text-indigo-500">
            {showAdvanced ? 'Hide Advanced' : 'Show Advanced'}
          </button>
        </div>
        
        {#if showAdvanced}
          <div class="bg-gray-100/90 p-4 rounded-lg space-y-4">  
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="maxRetries" class="block text-sm font-medium text-gray-700">Max Retries (0-10)</label>
                <input type="number" id="maxRetries" bind:value={maxRetries} min="0" max="10"
                       class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
              </div>
              <div>
                <label for="retryBackoffSeconds" class="block text-sm font-medium text-gray-700">Retry Backoff (seconds)</label>
                <input type="number" id="retryBackoffSeconds" bind:value={retryBackoffSeconds} min="1" max="3600"
                       class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
              </div>
            </div>
            
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="requestTimeoutSeconds" class="block text-sm font-medium text-gray-700">Request Timeout (seconds)</label>
                <input type="number" id="requestTimeoutSeconds" bind:value={requestTimeoutSeconds} min="1" max="300"
                       class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
              </div>
              <div>
                <label for="expectedStatusCodes" class="block text-sm font-medium text-gray-700">Expected Status Codes</label>
                <input type="text" id="expectedStatusCodes" bind:value={expectedStatusCodes} placeholder="200,201,202,204"
                       class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
              </div>
            </div>
            
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="userAgent" class="block text-sm font-medium text-gray-700">User Agent</label>
                <input type="text" id="userAgent" bind:value={userAgent}
                       class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
              </div>
              <div>
                <label for="contentType" class="block text-sm font-medium text-gray-700">Content Type</label>
                <input type="text" id="contentType" bind:value={contentType}
                       class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
              </div>
            </div>
            
            <div>
              <label for="webhookSecret" class="block text-sm font-medium text-gray-700">Webhook Secret (for signature verification)</label>
              <input type="password" id="webhookSecret" bind:value={webhookSecret} placeholder="Optional secret key"
                     class="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-indigo-300 focus:ring focus:ring-indigo-200 focus:ring-opacity-50">
            </div>
            
            <div class="space-y-2">
              <label class="flex items-center">
                <input type="checkbox" bind:checked={captureResponseBody}
                       class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-offset-0 focus:ring-indigo-200 focus:ring-opacity-50">
                <span class="ml-2 text-sm text-gray-700">Capture Response Body</span>
                <span class="ml-2 text-xs text-gray-500">(for debugging/compliance)</span>
              </label>
              
              <label class="flex items-center">
                <input type="checkbox" bind:checked={followRedirects}
                       class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-offset-0 focus:ring-indigo-200 focus:ring-opacity-50">
                <span class="ml-2 text-sm text-gray-700">Follow Redirects</span>
                <span class="ml-2 text-xs text-gray-500">(allow HTTP redirects)</span>
              </label>
              
              <label class="flex items-center">
                <input type="checkbox" bind:checked={verifySSL}
                       class="rounded border-gray-300 text-indigo-600 shadow-sm focus:border-indigo-300 focus:ring focus:ring-offset-0 focus:ring-indigo-200 focus:ring-opacity-50">
                <span class="ml-2 text-sm text-gray-700">Verify SSL Certificates</span>
                <span class="ml-2 text-xs text-gray-500">(disable for development only)</span>
              </label>
            </div>
          </div>
        {/if}
      </div>

      {#if error}
        <div class="p-4 bg-red-50 border border-red-200 rounded-md">
          <p class="text-red-600 text-sm">{error}</p>
        </div>
      {/if}
      
      <!-- Template Info -->
      <div class="p-4 bg-blue-50 border border-blue-200 rounded-md">
        <h3 class="text-sm font-medium text-blue-800 mb-2">💡 Custom Payload Templates</h3>
        <p class="text-blue-700 text-sm">
          After registering your webhook, you can customize how event payloads are formatted by adding templates to individual event subscriptions. 
          Visit the webhook detail page to configure templates for Slack, Discord, or custom API formats.
        </p>
      </div>
      
      <div class="flex space-x-4">
        <button type="submit" 
                class="flex-1 bg-primary text-white py-2 px-4 rounded-md hover:bg-primary-dark focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-primary">
          Register Webhook
        </button>
        <button type="button" onclick={() => goto('/webhooks')}
                class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500">
          Cancel
        </button>
      </div>
    </form>
  </div>
</div>
<style>
  .bg-primary {
    background-color: #13348f;
  }
</style>
