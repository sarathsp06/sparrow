<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, unwrap } from '$lib/services';
  import { JSONSchemaMetaSchema, jsonToJsonSchema } from '$lib';
  import { formatAPIError } from '$lib/utils';
  import {
    createAjvValidator,
    JSONEditor,
    Mode,
    type JSONContent,
    type Validator
  } from "svelte-jsoneditor";
  import 'svelte-jsoneditor/themes/jse-theme-dark.css';

  let name = $state('');
  let description = $state('');
  let schema = $state({ json: {} } as JSONContent);
  let active = $state(true);
  let error = $state('');
  let submitting = $state(false);
  let showSchemaHelper = $state(false);
  let sampleJson = $state({ json: {} } as JSONContent);
  let schemaHelperError = $state('');

  const validator: Validator = createAjvValidator({ schema: JSONSchemaMetaSchema });

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

  async function registerEvent(e: Event) {
    e.preventDefault();
    error = '';
    submitting = true;
    try {
      let schemaString: string = "{}";
      if ("text" in schema && typeof schema.text === "string") {
        schemaString = schema.text;
      } else if ("json" in schema && schema.json !== undefined) {
        schemaString = JSON.stringify(schema.json);
      }

      const event_schema = JSON.parse(schemaString);

      unwrap(await api.POST('/v1/event-types', {
        body: { name, description, event_schema, active },
      }));
      goto('/events');
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to register event');
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>Register Event | Sparrow</title>
</svelte:head>

<main class="mx-auto max-w-2xl px-4 sm:px-6 py-8">
  <nav class="flex items-center gap-2 text-sm text-muted mb-6">
    <a class="link" href="/events">Events</a>
    <span class="text-faint">/</span>
    <span class="text-text">Register</span>
  </nav>

  <div class="mb-6">
    <p class="eyebrow mb-1.5">Catalog / New Event</p>
    <h1 class="text-2xl">Register Event Type</h1>
    <p class="text-sm text-muted mt-1">Define a new event type that webhooks can subscribe to.</p>
  </div>

  <form onsubmit={registerEvent} class="space-y-6">
    <section class="panel p-5 space-y-4">
      <div>
        <label for="name" class="field-label">Event Name</label>
        <input
          id="name"
          type="text"
          bind:value={name}
          required
          placeholder="order.created"
          class="input"
        />
      </div>
      <div>
        <label for="description" class="field-label">Description</label>
        <input
          id="description"
          type="text"
          bind:value={description}
          placeholder="Fired when a new order is created"
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
        <span class="field-label mb-0">JSON Schema (optional)</span>
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
        {submitting ? 'Registering…' : 'Register Event'}
      </button>
    </div>
  </form>
</main>
