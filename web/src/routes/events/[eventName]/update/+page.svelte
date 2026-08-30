<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { api, unwrap } from "$lib/services";
  import { JSONSchemaMetaSchema, jsonToJsonSchema, toJSONObject, formatAPIError } from "$lib/utils";
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

  const validator: Validator = createAjvValidator({ schema: JSONSchemaMetaSchema });

  onMount(async () => {
    const eventName = decodeURIComponent(page.params.eventName ?? '');
    try {
      const event = unwrap(await api.GET('/v1/event-types/{name}', {
        params: { path: { name: eventName } },
      }));
      name = event.name;
      description = event.description ?? '';
      schema = { json: event.event_schema || {} };
      active = event.active;
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to load event details');
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
      unwrap(await api.PATCH('/v1/event-types/{name}', {
        params: { path: { name } },
        body: {
          description,
          event_schema: toJSONObject(schema),
          active,
        },
      }));
      goto("/events");
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to update event');
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>Update {name || 'Event'} | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <div class="max-w-2xl mx-auto px-4 sm:px-6 py-6">
    <div class="mb-6">
      <nav class="mb-3">
        <a href="/events" class="text-sm text-gray-500 hover:text-gray-700 transition">&larr; Back to Events</a>
      </nav>
      <h1 class="text-2xl font-bold text-gray-900">Update Event Type</h1>
    </div>

    {#if loading}
      <div class="animate-pulse space-y-4">
        <div class="h-10 bg-gray-200 rounded"></div>
        <div class="h-32 bg-gray-100 rounded"></div>
      </div>
    {:else}
      <form onsubmit={updateEvent} class="space-y-6">
        <section class="bg-white rounded-lg border border-gray-200 p-5 space-y-4">
          <div>
            <label for="name" class="block text-sm font-medium text-gray-700 mb-1">Event Name</label>
            <input id="name" type="text" value={name} disabled class="w-full px-3 py-2 border border-gray-200 bg-gray-50 rounded-lg text-sm text-gray-500" />
          </div>
          <div>
            <label for="description" class="block text-sm font-medium text-gray-700 mb-1">Description</label>
            <input
              id="description"
              type="text"
              bind:value={description}
              class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900 focus:border-transparent"
            />
          </div>
          <div class="flex items-center gap-2">
            <input id="active" type="checkbox" bind:checked={active} class="rounded border-gray-300" />
            <label for="active" class="text-sm text-gray-700">Active</label>
          </div>
        </section>

        <section class="bg-white rounded-lg border border-gray-200 p-5">
          <div class="flex items-center justify-between mb-3">
            <label class="block text-sm font-medium text-gray-700">JSON Schema</label>
            <button type="button" onclick={() => (showSchemaHelper = !showSchemaHelper)} class="text-xs text-gray-500 hover:text-gray-700 underline">
              {showSchemaHelper ? 'Hide' : 'Generate from sample'}
            </button>
          </div>
          {#if showSchemaHelper}
            <div class="mb-4 p-3 bg-gray-50 rounded-lg border border-gray-200">
              <p class="text-xs text-gray-500 mb-2">Paste a sample payload to generate a schema:</p>
              <div class="h-32 mb-2">
                <JSONEditor bind:content={sampleJson} mode={Mode.text} />
              </div>
              {#if schemaHelperError}
                <p class="text-xs text-red-600 mb-2">{schemaHelperError}</p>
              {/if}
              <button type="button" onclick={generateSchemaFromSample} class="text-xs px-3 py-1.5 bg-gray-900 text-white rounded-lg hover:bg-gray-800">
                Generate Schema
              </button>
            </div>
          {/if}
          <div class="h-64">
            <JSONEditor bind:content={schema} {validator} />
          </div>
        </section>

        {#if error}
          <div class="bg-red-50 border border-red-200 rounded-lg p-4">
            <p class="text-sm text-red-700">{error}</p>
          </div>
        {/if}

        <div class="flex items-center justify-end gap-3 pt-2">
          <a href="/events" class="px-4 py-2 text-sm font-medium text-gray-700 hover:text-gray-900 transition">Cancel</a>
          <button
            type="submit"
            disabled={submitting}
            class="px-4 py-2 bg-gray-900 text-white rounded-lg text-sm font-medium hover:bg-gray-800 disabled:opacity-50 transition"
          >
            {submitting ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </form>
    {/if}
  </div>
</div>
