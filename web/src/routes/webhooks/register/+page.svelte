<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, unwrap } from '$lib/services';
  import { formatAPIError } from '$lib/utils';
  import { namespaceStore } from '$lib/namespace.svelte';
  import { onMount } from 'svelte';
  import type { components } from '$lib/api-types';

  type EventTypeItem = components["schemas"]["EventTypeItem"];

  let namespace = $state(namespaceStore.value);
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

<main class="mx-auto max-w-2xl px-4 sm:px-6 py-8">
  <nav class="flex items-center gap-2 text-sm text-muted mb-6">
    <a class="link" href="/webhooks">Webhooks</a>
    <span class="text-faint">/</span>
    <span class="text-text">Register</span>
  </nav>
  <div class="mb-6">
    <p class="eyebrow mb-1.5">Fleet / Webhooks</p>
    <h1 class="text-2xl">Register New Webhook</h1>
    <p class="text-sm text-muted mt-1">Configure a new webhook endpoint to receive event notifications.</p>
  </div>

    <form onsubmit={registerWebhook} class="space-y-6">
      <section class="panel p-5 space-y-4">
        <div>
          <label for="namespace" class="field-label">Namespace</label>
          <input id="namespace" type="text" bind:value={namespace} class="input" style={namespaceError ? 'border-color:color-mix(in srgb,var(--color-bad) 55%,transparent)' : ''} />
          {#if namespaceError}<p class="text-xs mt-1" style="color:var(--color-bad)">{namespaceError}</p>{/if}
        </div>
        <div>
          <label for="url" class="field-label">Target URL</label>
          <input id="url" type="text" bind:value={url} placeholder="https://example.com/webhook" class="input" style={urlError ? 'border-color:color-mix(in srgb,var(--color-bad) 55%,transparent)' : ''} />
          {#if urlError}<p class="text-xs mt-1" style="color:var(--color-bad)">{urlError}</p>{/if}
        </div>
        <div>
          <label for="description" class="field-label">Description</label>
          <input id="description" type="text" bind:value={description} class="input" />
        </div>
        <label for="active" class="flex items-center gap-2 hover:bg-white/5 rounded px-2 py-1.5 -mx-2 cursor-pointer">
          <input id="active" type="checkbox" bind:checked={active} class="accent-[color:var(--color-beacon)]" />
          <span class="text-sm text-text">Active</span>
        </label>
      </section>

      <section class="panel p-5">
        <label for="event-search" class="field-label">Subscribe to Events</label>
        {#if eventsError}<p class="text-xs mb-2" style="color:var(--color-bad)">{eventsError}</p>{/if}
        <input id="event-search" type="text" placeholder="Search events…" bind:value={eventSearch} class="input mb-3" />
        <div class="space-y-1 max-h-56 overflow-y-auto">
          {#each filteredEvents as ev}
            <label class="flex items-center gap-2 px-2 py-1.5 hover:bg-white/5 rounded cursor-pointer">
              <input
                type="checkbox"
                checked={events.includes(ev.name)}
                onchange={() => {
                  events = events.includes(ev.name) ? events.filter(e => e !== ev.name) : [...events, ev.name];
                }}
                class="accent-[color:var(--color-beacon)]"
              />
              <span class="text-sm text-text">{ev.name}</span>
            </label>
          {/each}
        </div>
      </section>

      <section class="panel p-5">
        <span class="field-label">HTTP Headers</span>
        {#each headers as h, i}
          <div class="flex gap-2 mb-2">
            <input type="text" placeholder="key" bind:value={h.key} class="input flex-1" />
            <input type="text" placeholder="value" bind:value={h.value} class="input flex-1" />
            <button type="button" onclick={() => removeHeader(i)} class="px-2 text-faint hover:text-bad transition-colors" aria-label="Remove header">&times;</button>
          </div>
        {/each}
        <button type="button" onclick={addHeader} class="btn btn-ghost !px-3 !py-1.5">+ Add Header</button>
      </section>

      <section class="panel p-5">
        <span class="field-label">Secret Headers (encrypted at rest)</span>
        {#each secretHeaders as h, i}
          <div class="flex gap-2 mb-2">
            <input type="text" placeholder="key" bind:value={h.key} class="input flex-1" />
            <input type="password" placeholder="value" bind:value={h.value} class="input flex-1" />
            <button type="button" onclick={() => removeSecretHeader(i)} class="px-2 text-faint hover:text-bad transition-colors" aria-label="Remove secret header">&times;</button>
          </div>
        {/each}
        <button type="button" onclick={addSecretHeader} class="btn btn-ghost !px-3 !py-1.5">+ Add Secret Header</button>
      </section>

      <section class="panel">
        <button type="button" onclick={() => (showAdvanced = !showAdvanced)} class="w-full flex items-center justify-between p-5 text-left">
          <span class="text-sm font-medium text-text">Advanced HTTP Configuration</span>
          <span class="text-muted mono text-lg leading-none">{showAdvanced ? '−' : '+'}</span>
        </button>
        {#if showAdvanced}
          <div class="p-5 pt-0 space-y-4 border-t border-line">
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="maxRetries" class="field-label">Max Retries</label>
                <input id="maxRetries" type="number" bind:value={maxRetries} class="input" />
              </div>
              <div>
                <label for="retryBackoff" class="field-label">Retry Backoff (s)</label>
                <input id="retryBackoff" type="number" bind:value={retryBackoffSeconds} class="input" />
              </div>
              <div>
                <label for="timeout" class="field-label">Request Timeout (s)</label>
                <input id="timeout" type="number" bind:value={requestTimeoutSeconds} class="input" />
              </div>
              <div>
                <label for="statusCodes" class="field-label">Expected Status Codes</label>
                <input id="statusCodes" type="text" bind:value={expectedStatusCodes} class="input" />
              </div>
            </div>
            <div>
              <label for="secret" class="field-label">Webhook Secret (HMAC signing key)</label>
              <input id="secret" type="password" bind:value={webhookSecret} placeholder="Leave blank to auto-generate…" class="input" />
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <label for="userAgent" class="field-label">User-Agent</label>
                <input id="userAgent" type="text" bind:value={userAgent} class="input" />
              </div>
              <div>
                <label for="contentType" class="field-label">Content-Type</label>
                <input id="contentType" type="text" bind:value={contentType} class="input" />
              </div>
            </div>
            <div class="flex flex-wrap items-center gap-4">
              <label class="flex items-center gap-2 text-sm text-text hover:bg-white/5 rounded px-2 py-1.5 cursor-pointer"><input type="checkbox" bind:checked={captureResponseBody} class="accent-[color:var(--color-beacon)]" /> Capture response body</label>
              <label class="flex items-center gap-2 text-sm text-text hover:bg-white/5 rounded px-2 py-1.5 cursor-pointer"><input type="checkbox" bind:checked={followRedirects} class="accent-[color:var(--color-beacon)]" /> Follow redirects</label>
              <label class="flex items-center gap-2 text-sm text-text hover:bg-white/5 rounded px-2 py-1.5 cursor-pointer"><input type="checkbox" bind:checked={verifySSL} class="accent-[color:var(--color-beacon)]" /> Verify SSL</label>
            </div>
          </div>
        {/if}
      </section>

      {#if error}
        <div class="panel p-4" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
          <p class="text-sm" style="color:var(--color-bad)">{error}</p>
        </div>
      {/if}

      <div class="panel-2 p-4">
        <p class="text-muted text-xs">A signing secret is generated automatically unless you provide one above. It's returned once on creation — copy it before leaving this page.</p>
      </div>

      <div class="flex items-center justify-end gap-3 pt-2">
        <a href="/webhooks" class="btn btn-ghost">Cancel</a>
        <button type="submit" disabled={submitting} class="btn btn-beacon">
          {submitting ? 'Registering…' : 'Register Webhook'}
        </button>
      </div>
    </form>
</main>
