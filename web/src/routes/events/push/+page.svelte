<script lang="ts">
  import {
    type Content,
    createAjvValidator,
    JSONEditor,
    Mode,
    type Validator
  } from "svelte-jsoneditor";
  import 'svelte-jsoneditor/themes/jse-theme-dark.css';

  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from '$lib/utils';
  import { namespaceStore } from '$lib/namespace.svelte';
  import { onMount } from "svelte";
  import type { components } from "$lib/api-types";

  type EventTypeItem = components["schemas"]["EventTypeItem"];

  let namespace = $state(namespaceStore.value);
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

<main class="mx-auto max-w-2xl px-4 sm:px-6 py-8">
  <nav class="flex items-center gap-2 text-sm text-muted mb-6">
    <a class="link" href="/events">Events</a>
    <span class="text-faint">/</span>
    <span class="text-text">Push Event</span>
  </nav>

  <div class="mb-6">
    <p class="eyebrow mb-1.5">Catalog / Push Event</p>
    <h1 class="text-2xl">Push a Test Event</h1>
  </div>

  {#if loadingEvents}
    <div class="panel p-6 space-y-6">
      <div class="animate-pulse space-y-6">
        <div>
          <div class="h-4 bg-white/5 rounded w-24 mb-2"></div>
          <div class="h-10 bg-white/[0.03] rounded w-full"></div>
        </div>
        <div>
          <div class="h-4 bg-white/5 rounded w-16 mb-2"></div>
          <div class="h-10 bg-white/[0.03] rounded w-full"></div>
        </div>
        <div>
          <div class="h-4 bg-white/5 rounded w-28 mb-2"></div>
          <div class="h-40 bg-white/[0.03] rounded w-full"></div>
        </div>
        <div class="h-10 bg-white/5 rounded w-32"></div>
      </div>
    </div>
  {:else}
    <form onsubmit={pushEvent} class="panel p-6 space-y-5">
      <div>
        <label for="namespace" class="field-label">Namespace</label>
        <input id="namespace" type="text" bind:value={namespace} required class="input" />
      </div>

      <div>
        <label for="event" class="field-label">Event Type</label>
        <select id="event" bind:value={event} required class="select">
          {#each availableEvents as e}
            <option value={e.name}>{e.name}</option>
          {/each}
        </select>
      </div>

      <div>
        <label for="payload" class="field-label">
          Payload {hasSchema() ? '(validated against schema)' : ''}
        </label>
        <div class="jse-theme-dark h-40">
          <JSONEditor bind:content={payload} {validator} />
        </div>
      </div>

      <div>
        <span class="field-label">Labels</span>
        <div class="flex flex-wrap gap-2 mb-2">
          {#each Object.entries(labels) as [k, v]}
            <span class="chip">
              {k}={v}
              <button type="button" onclick={() => removeLabel(k)} aria-label="Remove label {k}" class="text-faint hover:text-text transition-colors">&times;</button>
            </span>
          {/each}
        </div>
        <div class="flex gap-2">
          <input type="text" placeholder="key" bind:value={newLabelKey} class="input flex-1" />
          <input type="text" placeholder="value" bind:value={newLabelValue} class="input flex-1" />
          <button type="button" onclick={addLabel} class="btn btn-ghost !px-3 !py-1.5">Add</button>
        </div>
      </div>

      <div class="flex items-center gap-3 pt-2">
        <button type="submit" disabled={loading} class="btn btn-beacon">
          {loading ? 'Pushing…' : 'Push Event'}
        </button>
      </div>
    </form>
  {/if}

  {#if error}
    <div class="mt-4 panel p-4" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
      <p class="text-sm font-medium" style="color:var(--color-bad)">{error}</p>
      {#if validationDetails.length > 0}
        <ul class="mt-2 text-xs list-disc list-inside" style="color:var(--color-bad)">
          {#each validationDetails as d}
            <li>{d}</li>
          {/each}
        </ul>
      {/if}
    </div>
  {/if}

  {#if successMessage}
    <div class="mt-4 panel p-4" style="border-color:color-mix(in srgb,var(--color-ok) 40%,transparent);background:color-mix(in srgb,var(--color-ok) 8%,var(--color-panel))">
      <p class="text-sm" style="color:var(--color-ok)">{successMessage}</p>
    </div>
  {/if}
</main>
