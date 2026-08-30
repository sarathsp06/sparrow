<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, unwrap } from '$lib/services';
  import { formatAPIError } from '$lib/utils';
  import { onMount } from 'svelte';
  import type { components } from '$lib/api-types';

  type EventTypeItem = components["schemas"]["EventTypeItem"];

  let namespace = $state('default');
  let events: string[] = $state([]);
  let url = $state('');
  let description = $state('');
  let active = $state(true);
  let allEvents: EventTypeItem[] = $state([]);
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

  // Secret Headers (encrypted server-side)
  let secretHeaders: { key: string; value: string }[] = $state([]);

  // Validation
  let urlError = $state('');
  let namespaceError = $state('');
  let eventsError = $state('');

  let filteredEvents = $derived(
    eventSearch.trim()
      ? allEvents.filter(e => e.name.toLowerCase().includes(eventSearch.toLowerCase()))
      : allEvents
  );

  onMount(async () => {
    try {
      const res = unwrap(await api.GET('/v1/event-types', { params: { query: { active_only: true } } }));
      allEvents = res.items || [];
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to load events');
    }
  });

  function addHeader() {
    headers = [...headers, { key: '', value: '' }];
  }

  function removeHeader(index: number) {
    headers = headers.filter((_, i) => i !== index);
  }

  function addSecretHeader() {
    secretHeaders = [...secretHeaders, { key: '', value: '' }];
  }

  function removeSecretHeader(index: number) {
    secretHeaders = secretHeaders.filter((_, i) => i !== index);
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
    eventsError = events.length === 0 ? 'Select at least one event' : '';
    if (!urlValid || !nsValid || eventsError) return;

    submitting = true;
    try {
      const headersMap: Record<string, string> = {};
      headers.forEach(h => {
        if (h.key.trim() && h.value.trim()) {
          headersMap[h.key.trim()] = h.value.trim();
        }
      });

      const secretHeadersMap: Record<string, string> = {};
      secretHeaders.forEach(h => {
        if (h.key.trim() && h.value.trim()) {
          secretHeadersMap[h.key.trim()] = h.value.trim();
        }
      });

      const statusCodes = expectedStatusCodes
        .split(',')
        .map(code => parseInt(code.trim()))
        .filter(code => !isNaN(code) && code >= 100 && code < 600);

      unwrap(await api.POST('/v1/namespaces/{namespace}/webhooks', {
        params: { path: { namespace } },
        body: {
          events,
          url,
          description,
          active,
          headers: headersMap,
          secret_headers: Object.keys(secretHeadersMap).length > 0 ? secretHeadersMap : undefined,
          http_config: showAdvanced ? {
            max_retries: maxRetries,
            retry_backoff_seconds: retryBackoffSeconds,
            capture_response_body: captureResponseBody,
            follow_redirects: followRedirects,
            verify_ssl: verifySSL,
            request_timeout_seconds: requestTimeoutSeconds,
            expected_status_codes: statusCodes,
            user_agent: userAgent.trim() || 'Sparrow-Webhook/1.0',
            content_type: contentType.trim() || 'application/json',
          } : undefined,
        },
      }));
      goto('/webhooks');
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to register webhook');
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
      <section class="bg-white rounded-lg border border-gray-200 p-5 space-y-4">
        <div>
          <label for="namespace" class="block text-sm font-medium text-gray-700 mb-1">Namespace</label>
          <input id="namespace" type="text" bind:value={namespace} class="w-full px-3 py-2 border {namespaceError ? 'border-red-300' : 'border-gray-300'} rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900" />
          {#if namespaceError}<p class="text-xs text-red-600 mt-1">{namespaceError}</p>{/if}
        </div>
        <div>
          <label for="url" class="block text-sm font-medium text-gray-700 mb-1">Target URL</label>
          <input id="url" type="text" bind:value={url} placeholder="https://example.com/webhook" class="w-full px-3 py-2 border {urlError ? 'border-red-300' : 'border-gray-300'} rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900" />
          {#if urlError}<p class="text-xs text-red-600 mt-1">{urlError}</p>{/if}
        </div>
        <div>
          <label for="description" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
          <input id="description" type="text" bind:value={description} class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900" />
        </div>
        <div class="flex items-center gap-2">
          <input id="active" type="checkbox" bind:checked={active} class="rounded border-gray-300" />
          <label for="active" class="text-sm text-gray-700">Active</label>
        </div>
      </section>

      <section class="bg-white rounded-lg border border-gray-200 p-5">
        <label for="event-search" class="block text-sm font-medium text-gray-700 mb-2">Subscribe to Events</label>
        {#if eventsError}<p class="text-xs text-red-600 mb-2">{eventsError}</p>{/if}
        <input id="event-search" type="text" placeholder="Search events..." bind:value={eventSearch} class="w-full px-3 py-2 mb-3 border border-gray-300 rounded-lg text-sm" />
        <div class="space-y-1 max-h-56 overflow-y-auto">
          {#each filteredEvents as ev}
            <label class="flex items-center gap-2 px-2 py-1.5 hover:bg-gray-50 rounded cursor-pointer">
              <input
                type="checkbox"
                checked={events.includes(ev.name)}
                onchange={() => {
                  events = events.includes(ev.name) ? events.filter(e => e !== ev.name) : [...events, ev.name];
                }}
                class="rounded border-gray-300"
              />
              <span class="text-sm text-gray-800">{ev.name}</span>
            </label>
          {/each}
        </div>
      </section>

      <section class="bg-white rounded-lg border border-gray-200 p-5">
        <span class="block text-sm font-medium text-gray-700 mb-2">HTTP Headers</span>
        {#each headers as h, i}
          <div class="flex gap-2 mb-2">
            <input type="text" placeholder="key" bind:value={h.key} class="flex-1 px-2 py-1.5 border border-gray-300 rounded text-sm" />
            <input type="text" placeholder="value" bind:value={h.value} class="flex-1 px-2 py-1.5 border border-gray-300 rounded text-sm" />
            <button type="button" onclick={() => removeHeader(i)} class="px-2 text-gray-400 hover:text-red-600">&times;</button>
          </div>
        {/each}
        <button type="button" onclick={addHeader} class="text-xs px-3 py-1.5 bg-gray-100 rounded-lg hover:bg-gray-200">+ Add Header</button>
      </section>

      <section class="bg-white rounded-lg border border-gray-200 p-5">
        <span class="block text-sm font-medium text-gray-700 mb-2">Secret Headers (encrypted at rest)</span>
        {#each secretHeaders as h, i}
          <div class="flex gap-2 mb-2">
            <input type="text" placeholder="key" bind:value={h.key} class="flex-1 px-2 py-1.5 border border-gray-300 rounded text-sm" />
            <input type="password" placeholder="value" bind:value={h.value} class="flex-1 px-2 py-1.5 border border-gray-300 rounded text-sm" />
            <button type="button" onclick={() => removeSecretHeader(i)} class="px-2 text-gray-400 hover:text-red-600">&times;</button>
          </div>
        {/each}
        <button type="button" onclick={addSecretHeader} class="text-xs px-3 py-1.5 bg-gray-100 rounded-lg hover:bg-gray-200">+ Add Secret Header</button>
      </section>

      <section class="bg-white rounded-lg border border-gray-200">
        <button type="button" onclick={() => (showAdvanced = !showAdvanced)} class="w-full flex items-center justify-between p-5 text-left">
          <span class="text-sm font-medium text-gray-700">Advanced HTTP Configuration</span>
          <span class="text-gray-400">{showAdvanced ? '−' : '+'}</span>
        </button>
        {#if showAdvanced}
          <div class="p-5 pt-0 space-y-4 border-t border-gray-100">
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="maxRetries" class="block text-xs font-medium text-gray-700 mb-1">Max Retries</label>
                <input id="maxRetries" type="number" bind:value={maxRetries} class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm" />
              </div>
              <div>
                <label for="retryBackoff" class="block text-xs font-medium text-gray-700 mb-1">Retry Backoff (s)</label>
                <input id="retryBackoff" type="number" bind:value={retryBackoffSeconds} class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm" />
              </div>
              <div>
                <label for="timeout" class="block text-xs font-medium text-gray-700 mb-1">Request Timeout (s)</label>
                <input id="timeout" type="number" bind:value={requestTimeoutSeconds} class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm" />
              </div>
              <div>
                <label for="statusCodes" class="block text-xs font-medium text-gray-700 mb-1">Expected Status Codes</label>
                <input id="statusCodes" type="text" bind:value={expectedStatusCodes} class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm" />
              </div>
            </div>
            <div>
              <label for="secret" class="block text-xs font-medium text-gray-700 mb-1">Webhook Secret (HMAC signing key)</label>
              <input id="secret" type="password" bind:value={webhookSecret} placeholder="Leave blank to auto-generate" class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm" />
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="userAgent" class="block text-xs font-medium text-gray-700 mb-1">User-Agent</label>
                <input id="userAgent" type="text" bind:value={userAgent} class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm" />
              </div>
              <div>
                <label for="contentType" class="block text-xs font-medium text-gray-700 mb-1">Content-Type</label>
                <input id="contentType" type="text" bind:value={contentType} class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm" />
              </div>
            </div>
            <div class="flex items-center gap-6">
              <label class="flex items-center gap-2 text-sm text-gray-700"><input type="checkbox" bind:checked={captureResponseBody} class="rounded border-gray-300" /> Capture response body</label>
              <label class="flex items-center gap-2 text-sm text-gray-700"><input type="checkbox" bind:checked={followRedirects} class="rounded border-gray-300" /> Follow redirects</label>
              <label class="flex items-center gap-2 text-sm text-gray-700"><input type="checkbox" bind:checked={verifySSL} class="rounded border-gray-300" /> Verify SSL</label>
            </div>
          </div>
        {/if}
      </section>

      {#if error}
        <div class="bg-red-50 border border-red-200 rounded-lg p-4">
          <p class="text-sm text-red-700">{error}</p>
        </div>
      {/if}

      <div class="bg-gray-50 border border-gray-200 rounded-lg p-4">
        <p class="text-xs text-gray-500">A signing secret is generated automatically unless you provide one above. It's returned once on creation — copy it before leaving this page.</p>
      </div>

      <div class="flex items-center justify-end gap-3 pt-2">
        <a href="/webhooks" class="px-4 py-2 text-sm font-medium text-gray-700 hover:text-gray-900 transition">Cancel</a>
        <button type="submit" disabled={submitting} class="px-4 py-2 bg-gray-900 text-white rounded-lg text-sm font-medium hover:bg-gray-800 disabled:opacity-50 transition">
          {submitting ? 'Registering...' : 'Register Webhook'}
        </button>
      </div>
    </form>
  </div>
</div>
