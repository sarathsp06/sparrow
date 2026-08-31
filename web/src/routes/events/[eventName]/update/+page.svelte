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
  import 'svelte-jsoneditor/themes/jse-theme-dark.css';

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

<main class="mx-auto max-w-2xl px-4 sm:px-6 py-8">
  <nav class="flex items-center gap-2 text-sm text-muted mb-6">
    <a class="link" href="/events">Events</a>
    <span class="text-faint">/</span>
    <span class="text-text">Update</span>
  </nav>

  <div class="mb-6">
    <p class="eyebrow mb-1.5">Catalog / Update Event</p>
    <h1 class="text-2xl">Update Event Type</h1>
  </div>

  {#if loading}
    <div class="panel p-5">
      <div class="animate-pulse space-y-4">
        <div class="h-10 bg-white/5 rounded"></div>
        <div class="h-32 bg-white/[0.03] rounded"></div>
      </div>
    </div>
  {:else}
    <form onsubmit={updateEvent} class="space-y-6">
      <section class="panel p-5 space-y-4">
        <div>
          <label for="name" class="field-label">Event Name</label>
          <input id="name" type="text" value={name} disabled class="input opacity-60" />
        </div>
        <div>
          <label for="description" class="field-label">Description</label>
          <input
            id="description"
            type="text"
            bind:value={description}
            class="input"
          />
        </div>
        <label for="active" class="flex items-center gap-2 rounded px-1 py-1 hover:bg-white/5 cursor-pointer w-fit">
          <input id="active" type="checkbox" bind:checked={active} class="accent-[color:var(--color-beacon)]" />
          <span class="text-sm text-text">Active</span>
        </label>
      </section>

      <section class="panel p-5">
        <div class="flex items-center justify-between mb-3">
          <span class="field-label mb-0">JSON Schema</span>
          <button type="button" onclick={() => (showSchemaHelper = !showSchemaHelper)} class="text-xs link-beacon">
            {showSchemaHelper ? 'Hide' : 'Generate from sample'}
          </button>
        </div>
        {#if showSchemaHelper}
          <div class="panel-2 p-4 mb-4">
            <p class="text-muted text-xs mb-2">Paste a sample payload to generate a schema:</p>
            <div class="jse-theme-dark h-32 mb-2">
              <JSONEditor bind:content={sampleJson} mode={Mode.text} />
            </div>
            {#if schemaHelperError}
              <p class="text-xs mb-2" style="color:var(--color-bad)">{schemaHelperError}</p>
            {/if}
            <button type="button" onclick={generateSchemaFromSample} class="btn btn-beacon !px-3 !py-1.5">
              Generate Schema
            </button>
          </div>
        {/if}
        <div class="jse-theme-dark h-64">
          <JSONEditor bind:content={schema} {validator} />
        </div>
      </section>

      {#if error}
        <div class="panel p-4" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
          <p class="text-sm" style="color:var(--color-bad)">{error}</p>
        </div>
      {/if}

      <div class="flex items-center justify-end gap-3 pt-2">
        <a href="/events" class="btn btn-ghost">Cancel</a>
        <button type="submit" disabled={submitting} class="btn btn-beacon">
          {submitting ? 'Saving…' : 'Save Changes'}
        </button>
      </div>
    </form>
  {/if}
</main>
