<script lang="ts">
  import {
    type Content,
    createAjvValidator,
    JSONEditor,
    Mode,
    type Validator
  } from "svelte-jsoneditor";

  import { eventClient as client } from "$lib/services";
  import { onMount } from "svelte";
  import type { RegisteredEvent } from "../../../../../proto/webhook_pb.js";
  import { activeNamespace, namespaces as allNamespaces } from "$lib/stores/namespace.svelte";

  let namespace = $state(activeNamespace() ?? '');
  let event = $state("");
  let payload = $state({ json: {} } as Content);
  let loading = $state(false);
  let loadingEvents = $state(true);
  let error = $state("");
  let validationDetails = $state<string[]>([]);
  let successMessage = $state("");
  let availableEvents: RegisteredEvent[] = $state([]);

  // Sync with store when activeNamespace changes
  $effect(() => {
    if (activeNamespace() !== null) {
      namespace = activeNamespace()!;
    }
  });

  // Watch for event changes and update payload with sample_payload
  $effect(() => {
    const selectedEvent = availableEvents.find((e) => e.name === event);
    if (selectedEvent && selectedEvent.samplePayload) {
      payload = { json: selectedEvent.samplePayload };
    }
  });

  function validator(): Validator {
    const selectedEvent = availableEvents.find((e) => e.name === event);
    if (selectedEvent && selectedEvent.schema && Object.keys(selectedEvent.schema).length > 0) {
      return createAjvValidator({ schema: selectedEvent.schema as any });
    }
    return createAjvValidator({ schema: {} });
  }

  function hasSchema(): boolean {
    const selectedEvent = availableEvents.find((e) => e.name === event);
    return !!(selectedEvent && selectedEvent.schema && Object.keys(selectedEvent.schema).length > 0);
  }

  async function fetchEvents() {
    loadingEvents = true;
    try {
      const req = { activeOnly: true };
      const res = await client.listEvents(req);
      availableEvents = res.events || [];
      if (availableEvents.length > 0) {
        event = availableEvents[0].name;
      }
    } catch (e: any) {
      error = `Failed to load available events: ${e.message}`;
    } finally {
      loadingEvents = false;
    }
  }

  onMount(fetchEvents);

  /**
   * Parse structured validation details from server error messages.
   * The backend returns errors like: "payload validation failed: /field/path: error message; /other: msg"
   */
  function parseValidationError(message: string): { summary: string; details: string[] } {
    const details: string[] = [];

    // Match patterns like "field_path: error description" separated by "; "
    const schemaMatch = message.match(/payload does not match event schema: payload validation failed: (.+)$/);
    if (schemaMatch) {
      const detailStr = schemaMatch[1];
      // Split on "; " to get individual field errors
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

      const req = {
        namespace,
        event,
        payload: payloadObj,
      };
      const res = await client.pushEvent(req);
      successMessage = `Event pushed successfully! Event ID: ${res.eventId}`;
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
    <!-- Breadcrumb -->
    <nav class="flex items-center text-sm text-gray-500 mb-6">
      <a href="/events" class="hover:text-gray-900 transition">Events</a>
      <span class="mx-2">/</span>
      <span class="text-gray-900 font-medium">Push Event</span>
    </nav>

    <div class="max-w-2xl">
      <h1 class="text-2xl font-bold text-gray-900 mb-6">Push a Test Event</h1>

      {#if loadingEvents}
        <!-- Loading skeleton for form -->
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
            <label for="namespace" class="block text-sm font-medium text-gray-700 mb-1">
              Namespace
            </label>
            {#if allNamespaces().length > 0}
              <select
                id="namespace"
                bind:value={namespace}
                class="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-gray-900 focus:border-gray-900 bg-white"
                required
              >
                <option value="">Select a namespace...</option>
                {#each allNamespaces() as ns}
                  <option value={ns.name}>{ns.name}</option>
                {/each}
              </select>
            {:else}
              <input
                type="text"
                id="namespace"
                bind:value={namespace}
                placeholder="Enter namespace..."
                class="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
                required
              />
            {/if}
          </div>

          <div>
            <label for="event" class="block text-sm font-medium text-gray-700 mb-1">
              Event Type
            </label>
            <div class="flex gap-2">
              {#if availableEvents.length > 0}
                <select
                  id="event-select"
                  bind:value={event}
                  class="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-gray-900 focus:border-gray-900 bg-white"
                >
                  <option value="">Select an event...</option>
                  {#each availableEvents as e}
                    <option value={e.name}>{e.name}</option>
                  {/each}
                </select>
                <span class="text-xs text-gray-400 self-center">or</span>
              {/if}
              <input
                id="event"
                type="text"
                bind:value={event}
                placeholder="Type event name"
                class="flex-1 px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
                required
              />
            </div>
            <p class="text-xs text-gray-500 mt-1">New event names will be auto-registered when pushed.</p>
          </div>

          <div>
            <label for="payload" class="block text-sm font-medium text-gray-700 mb-1">
              Payload (JSON)
            </label>
            {#if !hasSchema()}
              <p class="text-xs text-amber-600 mb-2">
                This event has no schema defined — any JSON payload will be accepted.
              </p>
            {/if}
            <div class="border border-gray-300 rounded-lg overflow-hidden">
              <JSONEditor
                validator={validator()}
                bind:content={payload}
                mode={Mode.text}
                mainMenuBar={false}
              />
            </div>
          </div>

          <div class="flex items-center gap-3 pt-2">
            <button
              type="submit"
              class="inline-flex items-center px-4 py-2 bg-gray-900 text-white text-sm font-medium rounded-lg hover:bg-gray-800 transition shadow-sm disabled:opacity-50 disabled:cursor-not-allowed"
              disabled={loading || !event.trim()}
            >
              {loading ? "Pushing..." : "Push Event"}
            </button>
            <a
              href="/events"
              class="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition"
            >
              Cancel
            </a>
          </div>
        </form>
      {/if}

      <!-- Error banner -->
      {#if error}
        <div class="mt-4 bg-red-50 border border-red-200 rounded-lg p-4">
          <div class="flex items-start gap-3">
            <svg class="w-5 h-5 text-red-500 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
            </svg>
            <div class="flex-1">
              <p class="text-sm font-medium text-red-800">{error}</p>
              {#if validationDetails.length > 0}
                <ul class="mt-2 space-y-1">
                  {#each validationDetails as detail}
                    <li class="text-xs text-red-700 font-mono bg-red-100 rounded px-2 py-1">
                      {detail}
                    </li>
                  {/each}
                </ul>
              {/if}
            </div>
            <button onclick={() => { error = ''; validationDetails = []; }} class="text-red-400 hover:text-red-600 transition" aria-label="Dismiss error">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      {/if}

      <!-- Success banner -->
      {#if successMessage}
        <div class="mt-4 bg-green-50 border border-green-200 rounded-lg p-4">
          <div class="flex items-start gap-3">
            <svg class="w-5 h-5 text-green-500 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
              <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clip-rule="evenodd" />
            </svg>
            <div class="flex-1">
              <p class="text-sm font-medium text-green-800">{successMessage}</p>
            </div>
            <button onclick={() => successMessage = ''} class="text-green-400 hover:text-green-600 transition" aria-label="Dismiss success message">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      {/if}
    </div>
  </main>
</div>
