<script lang="ts">
  import {
    type Content,
    createAjvValidator,
    JSONEditor,
    Mode,
    type Validator
  } from "svelte-jsoneditor";

  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from '$lib/utils';
  import { onMount } from "svelte";
  import type { components } from "$lib/api-types";

  type EventTypeItem = components["schemas"]["EventTypeItem"];

  let namespace = $state('default');
  let event = $state("");
  let payload = $state({ json: {} } as Content);
  let labels = $state<Record<string, string>>({});
  let newLabelKey = $state("");
  let newLabelValue = $state("");
  let loading = $state(false);
  let loadingEvents = $state(true);
  let error = $state("");
  let validationDetails = $state<string[]>([]);
  let successMessage = $state("");
  let availableEvents: EventTypeItem[] = $state([]);

  // Watch for event changes and update payload with sample_payload
  $effect(() => {
    const selectedEvent = availableEvents.find((e) => e.name === event);
    if (selectedEvent && selectedEvent.sample_payload) {
      payload = { json: selectedEvent.sample_payload };
    }
  });

  const validator: Validator = $derived.by(() => {
    const selectedEvent = availableEvents.find((e) => e.name === event);
    if (selectedEvent && selectedEvent.event_schema && Object.keys(selectedEvent.event_schema).length > 0) {
      return createAjvValidator({ schema: selectedEvent.event_schema as any });
    }
    return createAjvValidator({ schema: {} });
  });

  function hasSchema(): boolean {
    const selectedEvent = availableEvents.find((e) => e.name === event);
    return !!(selectedEvent && selectedEvent.event_schema && Object.keys(selectedEvent.event_schema).length > 0);
  }

  async function fetchEvents() {
    loadingEvents = true;
    try {
      const res = unwrap(await api.GET('/v1/event-types', { params: { query: { active_only: true } } }));
      availableEvents = res.items || [];
      if (availableEvents.length > 0) {
        event = availableEvents[0].name;
      }
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to load available events');
    } finally {
      loadingEvents = false;
    }
  }

  function addLabel() {
    if (newLabelKey.trim() && newLabelValue.trim()) {
      labels = { ...labels, [newLabelKey.trim()]: newLabelValue.trim() };
      newLabelKey = "";
      newLabelValue = "";
    }
  }

  function removeLabel(key: string) {
    const { [key]: _, ...rest } = labels;
    labels = rest;
  }

  onMount(fetchEvents);

  /**
   * Parse structured validation details from server error messages.
   * The backend returns errors like: "payload validation failed: /field/path: error message; /other: msg"
   */
  function parseValidationError(message: string): { summary: string; details: string[] } {
    const details: string[] = [];

    const schemaMatch = message.match(/payload does not match event schema: payload validation failed: (.+)$/);
    if (schemaMatch) {
      const detailStr = schemaMatch[1];
      const parts = detailStr.split("; ");
      for (const part of parts) {
        details.push(part.trim());
      }
      return {
        summary: "Payload does not match the event schema",
        details
      };
    }

    return { summary: message, details: [] };
  }

  async function pushEvent(e: Event) {
    e.preventDefault();
    loading = true;
    error = "";
    validationDetails = [];
    successMessage = "";
    try {
      let payloadObj: any = {};
      if ("text" in payload && payload.text) {
        payloadObj = JSON.parse(payload.text);
      } else if ("json" in payload && payload.json !== undefined) {
        payloadObj = payload.json;
      }

      const res = unwrap(await api.POST('/v1/namespaces/{namespace}/events', {
        params: { path: { namespace }, query: { event } },
        body: { payload: payloadObj, labels },
      }));
      successMessage = `Event pushed successfully! Event ID: ${res.event_id}`;
      labels = {};
      newLabelKey = "";
      newLabelValue = "";
    } catch (e: any) {
      const parsed = parseValidationError(e.message);
      error = parsed.summary;
      validationDetails = parsed.details;
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>Push Event | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    <nav class="flex items-center text-sm text-gray-500 mb-6">
      <a href="/events" class="hover:text-gray-900 transition">Events</a>
      <span class="mx-2">/</span>
      <span class="text-gray-900 font-medium">Push Event</span>
    </nav>

    <div class="max-w-2xl">
      <h1 class="text-2xl font-bold text-gray-900 mb-6">Push a Test Event</h1>

      {#if loadingEvents}
        <div class="bg-white rounded-lg border border-gray-200 p-6 space-y-6">
          <div class="animate-pulse space-y-6">
            <div>
              <div class="h-4 bg-gray-200 rounded w-24 mb-2"></div>
              <div class="h-10 bg-gray-100 rounded w-full"></div>
            </div>
            <div>
              <div class="h-4 bg-gray-200 rounded w-16 mb-2"></div>
              <div class="h-10 bg-gray-100 rounded w-full"></div>
            </div>
            <div>
              <div class="h-4 bg-gray-200 rounded w-28 mb-2"></div>
              <div class="h-40 bg-gray-100 rounded w-full"></div>
            </div>
            <div class="h-10 bg-gray-200 rounded w-32"></div>
          </div>
        </div>
      {:else}
        <form onsubmit={pushEvent} class="bg-white rounded-lg border border-gray-200 p-6 space-y-5">
          <div>
            <label for="namespace" class="block text-sm font-medium text-gray-700 mb-1">Namespace</label>
            <input id="namespace" type="text" bind:value={namespace} required class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900" />
          </div>

          <div>
            <label for="event" class="block text-sm font-medium text-gray-700 mb-1">Event Type</label>
            <select id="event" bind:value={event} required class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900">
              {#each availableEvents as e}
                <option value={e.name}>{e.name}</option>
              {/each}
            </select>
          </div>

          <div>
            <label for="payload" class="block text-sm font-medium text-gray-700 mb-1">
              Payload {hasSchema() ? '(validated against schema)' : ''}
            </label>
            <div class="h-40">
              <JSONEditor bind:content={payload} {validator} />
            </div>
          </div>

          <div>
            <span class="block text-sm font-medium text-gray-700 mb-1">Labels</span>
            <div class="flex flex-wrap gap-2 mb-2">
              {#each Object.entries(labels) as [k, v]}
                <span class="inline-flex items-center gap-1 px-2 py-1 bg-gray-100 rounded text-xs">
                  {k}={v}
                  <button type="button" onclick={() => removeLabel(k)} class="text-gray-400 hover:text-gray-700">&times;</button>
                </span>
              {/each}
            </div>
            <div class="flex gap-2">
              <input type="text" placeholder="key" bind:value={newLabelKey} class="flex-1 px-2 py-1.5 border border-gray-300 rounded text-sm" />
              <input type="text" placeholder="value" bind:value={newLabelValue} class="flex-1 px-2 py-1.5 border border-gray-300 rounded text-sm" />
              <button type="button" onclick={addLabel} class="px-3 py-1.5 bg-gray-100 rounded text-sm hover:bg-gray-200">Add</button>
            </div>
          </div>

          <div class="flex items-center gap-3 pt-2">
            <button type="submit" disabled={loading} class="px-4 py-2 bg-gray-900 text-white rounded-lg text-sm font-medium hover:bg-gray-800 disabled:opacity-50 transition">
              {loading ? 'Pushing...' : 'Push Event'}
            </button>
          </div>
        </form>
      {/if}

      {#if error}
        <div class="mt-4 bg-red-50 border border-red-200 rounded-lg p-4">
          <p class="text-sm font-medium text-red-800">{error}</p>
          {#if validationDetails.length > 0}
            <ul class="mt-2 text-xs text-red-700 list-disc list-inside">
              {#each validationDetails as d}
                <li>{d}</li>
              {/each}
            </ul>
          {/if}
        </div>
      {/if}

      {#if successMessage}
        <div class="mt-4 bg-green-50 border border-green-200 rounded-lg p-4">
          <p class="text-sm text-green-800">{successMessage}</p>
        </div>
      {/if}
    </div>
  </main>
</div>
