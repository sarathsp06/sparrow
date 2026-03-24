<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { eventClient as client } from "$lib/services";
  import { JSONSchemaMetaSchema, jsonToJsonSchema, toJSONObject } from "$lib/utils";
  import { onMount } from "svelte";
  import {
    createAjvValidator,
    type JSONContent,
    JSONEditor,
    Mode,
    type Validator
  } from "svelte-jsoneditor";

  let name = $state("");
  let description = $state("");
  let schema: JSONContent = $state({ json: {} });
  let active = $state(true);
  let error = $state("");
  let loading = $state(true);
  let submitting = $state(false);
  let showSchemaHelper = $state(false);
  let sampleJson: JSONContent = $state({ json: {} });
  let schemaHelperError = $state('');

  function validator(): Validator {
    return createAjvValidator({ schema: JSONSchemaMetaSchema });
  }

  onMount(async () => {
    const eventName = decodeURIComponent(page.params.eventName ?? '');
    try {
      const res = await client.getEvent({ name: eventName });
      const event = res.event;
      if (event) {
        name = event.name;
        description = event.description;
        schema = { json: event.schema || {} };
        active = event.active;
      } else {
        error = "Event not found";
      }
    } catch (e: any) {
      error = `Failed to load event details: ${e.message}`;
    } finally {
      loading = false;
    }
  });

  function generateSchemaFromSample() {
    schemaHelperError = '';
    try {
      let sampleObj: any;
      if ("text" in sampleJson && typeof sampleJson.text === "string") {
        sampleObj = JSON.parse(sampleJson.text);
      } else if ("json" in sampleJson && sampleJson.json !== undefined) {
        sampleObj = sampleJson.json;
      } else {
        schemaHelperError = 'Please enter a valid JSON sample.';
        return;
      }

      const generatedSchema = jsonToJsonSchema(sampleObj);
      schema = { json: generatedSchema };
      showSchemaHelper = false;
      sampleJson = { json: {} };
    } catch (e: any) {
      schemaHelperError = `Invalid JSON: ${e.message}`;
    }
  }

  async function updateEvent(e: Event) {
    e.preventDefault();
    error = "";
    submitting = true;
    try {
      const req = {
        name,
        description,
        schema: toJSONObject(schema),
        active,
      };
      await client.updateEvent(req);
      goto("/events");
    } catch (e: any) {
      error = `Failed to update event: ${e.message}`;
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>Update {name || 'Event'} | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    <!-- Breadcrumb -->
    <nav class="flex items-center text-sm text-gray-500 mb-6">
      <a href="/events" class="hover:text-gray-900 transition">Events</a>
      <span class="mx-2">/</span>
      <span class="text-gray-900 font-medium">{name || 'Loading...'}</span>
      <span class="mx-2">/</span>
      <span class="text-gray-900 font-medium">Update</span>
    </nav>

    <div class="max-w-2xl">
      <h1 class="text-2xl font-bold text-gray-900 mb-6">Update Event</h1>

      {#if loading}
        <!-- Loading skeleton -->
        <div class="bg-white rounded-lg border border-gray-200 p-6 space-y-6">
          <div class="animate-pulse space-y-6">
            <div>
              <div class="h-4 bg-gray-200 rounded w-16 mb-2"></div>
              <div class="h-10 bg-gray-100 rounded w-full"></div>
            </div>
            <div>
              <div class="h-4 bg-gray-200 rounded w-24 mb-2"></div>
              <div class="h-20 bg-gray-100 rounded w-full"></div>
            </div>
            <div>
              <div class="h-4 bg-gray-200 rounded w-28 mb-2"></div>
              <div class="h-40 bg-gray-100 rounded w-full"></div>
            </div>
            <div class="h-10 bg-gray-200 rounded w-32"></div>
          </div>
        </div>
      {:else}
        <form onsubmit={updateEvent} class="bg-white rounded-lg border border-gray-200 p-6 space-y-5">
          <div>
            <label for="name" class="block text-sm font-medium text-gray-700 mb-1">Name</label>
            <input
              type="text"
              id="name"
              bind:value={name}
              class="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg bg-gray-50 text-gray-500 cursor-not-allowed"
              disabled
            />
            <p class="text-xs text-gray-500 mt-1">Event name cannot be changed after creation.</p>
          </div>

          <div>
            <label for="description" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <textarea
              id="description"
              bind:value={description}
              rows="3"
              class="w-full px-3 py-2 text-sm border border-gray-300 rounded-lg focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
            ></textarea>
          </div>

          <div>
            <div class="flex items-center justify-between mb-1">
              <label for="schema" class="block text-sm font-medium text-gray-700">Schema (JSON Schema)</label>
              <button
                type="button"
                onclick={() => { showSchemaHelper = !showSchemaHelper; schemaHelperError = ''; }}
                class="text-xs font-medium text-blue-600 hover:text-blue-800 transition"
              >
                {showSchemaHelper ? 'Close helper' : 'Generate from sample JSON'}
              </button>
            </div>
            <p class="text-xs text-gray-500 mb-2">
              Define the expected payload structure. Any valid JSON Schema is accepted — leave as <code class="bg-gray-100 px-1 rounded">{'{}'}</code> to allow any payload.
            </p>

            {#if showSchemaHelper}
              <div class="mb-3 bg-blue-50 border border-blue-200 rounded-lg p-4 space-y-3">
                <div>
                  <p class="text-sm font-medium text-blue-900 mb-1">Paste a sample JSON payload</p>
                  <p class="text-xs text-blue-700 mb-2">
                    A schema will be inferred from the structure. You can then refine it — add <code class="bg-blue-100 px-1 rounded">enum</code>, <code class="bg-blue-100 px-1 rounded">minLength</code>, <code class="bg-blue-100 px-1 rounded">pattern</code>, remove <code class="bg-blue-100 px-1 rounded">required</code> fields, etc.
                  </p>
                  <div class="border border-blue-300 rounded-lg overflow-hidden">
                    <JSONEditor bind:content={sampleJson} mode={Mode.text} mainMenuBar={false} />
                  </div>
                </div>
                {#if schemaHelperError}
                  <p class="text-xs text-red-600">{schemaHelperError}</p>
                {/if}
                <button
                  type="button"
                  onclick={generateSchemaFromSample}
                  class="inline-flex items-center px-3 py-1.5 bg-blue-600 text-white text-xs font-medium rounded-lg hover:bg-blue-700 transition"
                >
                  Generate Schema
                </button>
              </div>
            {/if}

            <div class="border border-gray-300 rounded-lg overflow-hidden">
              <JSONEditor validator={validator()} bind:content={schema} mode={Mode.text} mainMenuBar={false} />
            </div>
          </div>

          <div>
            <label class="flex items-center gap-2">
              <input
                type="checkbox"
                bind:checked={active}
                class="rounded border-gray-300 text-gray-900 shadow-sm focus:ring-gray-900 focus:ring-offset-0"
              />
              <span class="text-sm text-gray-700">Active</span>
            </label>
          </div>

          {#if error}
            <div class="bg-red-50 border border-red-200 rounded-lg p-4">
              <div class="flex items-start gap-3">
                <svg class="w-5 h-5 text-red-500 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
                  <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
                </svg>
                <div class="flex-1">
                  <p class="text-sm font-medium text-red-800">{error}</p>
                </div>
                <button onclick={() => error = ''} class="text-red-400 hover:text-red-600 transition" aria-label="Dismiss error">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>
            </div>
          {/if}

          <div class="flex items-center gap-3 pt-2">
            <button
              type="submit"
              class="inline-flex items-center px-4 py-2 bg-gray-900 text-white text-sm font-medium rounded-lg hover:bg-gray-800 transition shadow-sm disabled:opacity-50 disabled:cursor-not-allowed"
              disabled={submitting}
            >
              {submitting ? 'Updating...' : 'Update Event'}
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
    </div>
  </main>
</div>
