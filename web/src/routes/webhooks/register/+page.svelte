<script lang="ts">
  import { goto } from '$app/navigation';
  import { eventClient, webhookClient as client } from '$lib/services';
  import { onMount } from 'svelte';
  import type { RegisteredEvent } from '../../../../../proto/webhook_pb.js';

  let namespace = $state('');
  let events: string[] = $state([]);
  let url = $state('');
  let description = $state('');
  let active = $state(true);
  let allEvents: RegisteredEvent[] = $state([]);
  let error = $state('');
  let submitting = $state(false);
  let eventSearch = $state('');

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
  let headers: { key: string; value: string }[] = $state([]);

  // Validation
  let urlError = $state('');
  let namespaceError = $state('');

  let filteredEvents = $derived(
    eventSearch.trim()
      ? allEvents.filter(e => e.name.toLowerCase().includes(eventSearch.toLowerCase()))
      : allEvents
  );

  onMount(async () => {
    try {
      const res = await eventClient.listEvents({ activeOnly: true });
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

  function validateUrl(val: string): boolean {
    if (!val.trim()) { urlError = 'URL is required'; return false; }
    try { new URL(val); urlError = ''; return true; }
    catch { urlError = 'Enter a valid URL (e.g. https://api.example.com/webhook)'; return false; }
  }

  function validateNamespace(val: string): boolean {
    if (!val.trim()) { namespaceError = 'Namespace is required'; return false; }
    namespaceError = '';
    return true;
  }

  async function registerWebhook(e: Event) {
    e.preventDefault();
    error = '';

    const urlValid = validateUrl(url);
    const nsValid = validateNamespace(namespace);
    if (!urlValid || !nsValid) return;

    if (events.length === 0) {
      error = 'Select at least one event to subscribe to.';
      return;
    }

    submitting = true;
    try {
      const headersMap: Record<string, string> = {};
      headers.forEach(h => {
        if (h.key.trim() && h.value.trim()) {
          headersMap[h.key.trim()] = h.value.trim();
        }
      });

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
          userAgent: userAgent.trim() || 'Sparrow-Webhook/1.0',
          contentType: contentType.trim() || 'application/json',
        } : undefined,
      };

      await client.registerWebhook(req);
      goto('/webhooks');
    } catch (e: any) {
      error = `Failed to register webhook: ${e.message}`;
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>Register Webhook | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <div class="max-w-2xl mx-auto px-4 sm:px-6 py-6">
    <!-- Header -->
    <div class="mb-6">
      <nav class="mb-3">
        <a href="/webhooks" class="text-sm text-gray-500 hover:text-gray-700 transition">
          &larr; Back to Webhooks
        </a>
      </nav>
      <h1 class="text-2xl font-bold text-gray-900">Register New Webhook</h1>
      <p class="text-sm text-gray-500 mt-1">Configure a new webhook endpoint to receive event notifications.</p>
    </div>

    <form onsubmit={registerWebhook} class="space-y-6">
      <!-- Basic Configuration -->
      <section class="bg-white rounded-lg border border-gray-200 p-5">
        <h2 class="text-sm font-semibold text-gray-900 uppercase tracking-wide mb-4">Basic Configuration</h2>

        <div class="space-y-4">
          <div>
            <label for="namespace" class="block text-sm font-medium text-gray-700 mb-1">Namespace</label>
            <input
              type="text"
              id="namespace"
              bind:value={namespace}
              oninput={() => validateNamespace(namespace)}
              placeholder="Enter namespace..."
              class="block w-full rounded-lg border-gray-300 text-sm shadow-sm focus:border-gray-900 focus:ring-gray-900 {namespaceError ? 'border-red-300' : ''}"
            />
            {#if namespaceError}
              <p class="mt-1 text-xs text-red-600">{namespaceError}</p>
            {/if}
          </div>

          <div>
            <label for="url" class="block text-sm font-medium text-gray-700 mb-1">Endpoint URL</label>
            <input
              type="url"
              id="url"
              bind:value={url}
              oninput={() => { if (urlError) validateUrl(url); }}
              onblur={() => validateUrl(url)}
              placeholder="https://api.example.com/webhook"
              class="block w-full rounded-lg border-gray-300 text-sm shadow-sm focus:border-gray-900 focus:ring-gray-900 {urlError ? 'border-red-300' : ''}"
            />
            {#if urlError}
              <p class="mt-1 text-xs text-red-600">{urlError}</p>
            {/if}
          </div>

          <div>
            <label for="description" class="block text-sm font-medium text-gray-700 mb-1">
              Description <span class="text-gray-400 font-normal">(optional)</span>
            </label>
            <input
              type="text"
              id="description"
              bind:value={description}
              placeholder="e.g., Order notification handler"
              class="block w-full rounded-lg border-gray-300 text-sm shadow-sm focus:border-gray-900 focus:ring-gray-900"
            />
          </div>

          <div class="flex items-center gap-3">
            <button
              type="button"
              onclick={() => { active = !active; }}
              aria-label={active ? 'Deactivate webhook' : 'Activate webhook'}
              class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none focus:ring-2 focus:ring-gray-900 focus:ring-offset-2 {active ? 'bg-green-500' : 'bg-gray-300'}"
            >
              <span
                class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {active ? 'translate-x-4' : 'translate-x-0'}"
              ></span>
            </button>
            <span class="text-sm text-gray-700">{active ? 'Active' : 'Inactive'} - webhook will {active ? '' : 'not '}receive events immediately</span>
          </div>
        </div>
      </section>

      <!-- Event Selection -->
      <section class="bg-white rounded-lg border border-gray-200 p-5">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-sm font-semibold text-gray-900 uppercase tracking-wide">
            Events
            {#if events.length > 0}
              <span class="ml-1 text-xs font-normal text-gray-500">({events.length} selected)</span>
            {/if}
          </h2>
        </div>

        {#if allEvents.length > 5}
          <div class="mb-3">
            <input
              type="text"
              placeholder="Filter events..."
              bind:value={eventSearch}
              class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
            />
          </div>
        {/if}

        <div class="grid grid-cols-1 sm:grid-cols-2 gap-1.5 max-h-56 overflow-y-auto border border-gray-200 rounded-lg p-2">
          {#each filteredEvents as event}
            <label class="flex items-center gap-2 px-2 py-1.5 rounded hover:bg-gray-50 cursor-pointer transition text-sm {events.includes(event.name) ? 'bg-blue-50' : ''}">
              <input
                type="checkbox"
                value={event.name}
                bind:group={events}
                class="rounded border-gray-300 text-gray-900 shadow-sm focus:ring-gray-900"
              />
              <span class="text-gray-700 truncate">{event.name}</span>
            </label>
          {/each}
          {#if filteredEvents.length === 0}
            <p class="col-span-2 text-sm text-gray-400 py-4 text-center">No events found</p>
          {/if}
        </div>
      </section>

      <!-- HTTP Headers -->
      <section class="bg-white rounded-lg border border-gray-200 p-5">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-sm font-semibold text-gray-900 uppercase tracking-wide">Custom HTTP Headers</h2>
          <button type="button" onclick={addHeader} class="text-xs font-medium text-gray-600 hover:text-gray-900 transition">
            + Add Header
          </button>
        </div>

        {#if headers.length === 0}
          <p class="text-sm text-gray-400">No custom headers configured.</p>
        {:else}
          <div class="space-y-2">
            {#each headers as header, index}
              <div class="flex items-center gap-2">
                <input
                  type="text"
                  bind:value={header.key}
                  placeholder="Header Name"
                  class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
                />
                <input
                  type="text"
                  bind:value={header.value}
                  placeholder="Header Value"
                  class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
                />
                <button
                  type="button"
                  onclick={() => removeHeader(index)}
                  class="shrink-0 p-1.5 text-gray-400 hover:text-red-600 rounded transition"
                  title="Remove header"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            {/each}
          </div>
        {/if}
      </section>

      <!-- Advanced HTTP Config -->
      <section class="bg-white rounded-lg border border-gray-200">
        <button
          type="button"
          onclick={() => showAdvanced = !showAdvanced}
          class="flex items-center justify-between w-full px-5 py-4 text-left"
        >
          <h2 class="text-sm font-semibold text-gray-900 uppercase tracking-wide">Advanced HTTP Configuration</h2>
          <svg
            class="w-4 h-4 text-gray-400 transition-transform {showAdvanced ? 'rotate-180' : ''}"
            fill="none" stroke="currentColor" viewBox="0 0 24 24"
          >
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {#if showAdvanced}
          <div class="border-t border-gray-200 px-5 py-4 space-y-4">
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="maxRetries" class="block text-sm font-medium text-gray-700 mb-1">Max Retries</label>
                <input type="number" id="maxRetries" bind:value={maxRetries} min="0" max="10"
                  class="block w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                <p class="mt-0.5 text-xs text-gray-400">0-10 attempts</p>
              </div>
              <div>
                <label for="retryBackoff" class="block text-sm font-medium text-gray-700 mb-1">Retry Backoff</label>
                <input type="number" id="retryBackoff" bind:value={retryBackoffSeconds} min="1" max="3600"
                  class="block w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                <p class="mt-0.5 text-xs text-gray-400">seconds between retries</p>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="timeout" class="block text-sm font-medium text-gray-700 mb-1">Request Timeout</label>
                <input type="number" id="timeout" bind:value={requestTimeoutSeconds} min="1" max="300"
                  class="block w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                <p class="mt-0.5 text-xs text-gray-400">seconds</p>
              </div>
              <div>
                <label for="statusCodes" class="block text-sm font-medium text-gray-700 mb-1">Expected Status Codes</label>
                <input type="text" id="statusCodes" bind:value={expectedStatusCodes} placeholder="200,201,202,204"
                  class="block w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                <p class="mt-0.5 text-xs text-gray-400">comma-separated</p>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="userAgent" class="block text-sm font-medium text-gray-700 mb-1">User Agent</label>
                <input type="text" id="userAgent" bind:value={userAgent}
                  class="block w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
              </div>
              <div>
                <label for="contentType" class="block text-sm font-medium text-gray-700 mb-1">Content Type</label>
                <input type="text" id="contentType" bind:value={contentType}
                  class="block w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
              </div>
            </div>

            <div>
              <label for="webhookSecret" class="block text-sm font-medium text-gray-700 mb-1">Webhook Secret</label>
              <input type="password" id="webhookSecret" bind:value={webhookSecret} placeholder="Optional secret for signature verification"
                class="block w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
            </div>

            <div class="space-y-2 pt-2">
              {#each [
                { label: 'Capture Response Body', bind: () => captureResponseBody, toggle: () => captureResponseBody = !captureResponseBody, hint: 'Store response bodies for debugging' },
                { label: 'Follow Redirects', bind: () => followRedirects, toggle: () => followRedirects = !followRedirects, hint: 'Allow HTTP redirects' },
                { label: 'Verify SSL Certificates', bind: () => verifySSL, toggle: () => verifySSL = !verifySSL, hint: 'Disable for development only' },
              ] as opt}
                <label class="flex items-center justify-between py-1 cursor-pointer">
                  <div>
                    <span class="text-sm text-gray-700">{opt.label}</span>
                    <span class="block text-xs text-gray-400">{opt.hint}</span>
                  </div>
                  <button
                    type="button"
                    onclick={opt.toggle}
                    aria-label="Toggle {opt.label}"
                    class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out {opt.bind() ? 'bg-green-500' : 'bg-gray-300'}"
                  >
                    <span
                      class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {opt.bind() ? 'translate-x-4' : 'translate-x-0'}"
                    ></span>
                  </button>
                </label>
              {/each}
            </div>
          </div>
        {/if}
      </section>

      <!-- Error -->
      {#if error}
        <div class="bg-red-50 border border-red-200 rounded-lg p-4">
          <p class="text-sm text-red-700">{error}</p>
        </div>
      {/if}

      <!-- Info banner -->
      <div class="bg-gray-50 border border-gray-200 rounded-lg p-4">
        <p class="text-sm text-gray-600">
          <span class="font-medium text-gray-700">Tip:</span> After registering, you can customize how event payloads are formatted by adding templates to individual event subscriptions from the webhook detail page.
        </p>
      </div>

      <!-- Actions -->
      <div class="flex items-center justify-end gap-3 pt-2">
        <button
          type="button"
          onclick={() => goto('/webhooks')}
          class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={submitting}
          class="px-6 py-2 text-sm font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition disabled:opacity-50 disabled:cursor-not-allowed shadow-sm"
        >
          {submitting ? 'Registering...' : 'Register Webhook'}
        </button>
      </div>
    </form>
  </div>
</div>
