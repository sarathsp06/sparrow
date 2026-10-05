<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import { api, unwrap } from "$lib/services";
  import { JSONSchemaMetaSchema, toJSONObject, formatAPIError, safeAjvValidator } from "$lib/utils";
  import { onMount } from "svelte";
  import ExpandableEditor from "$lib/components/ExpandableEditor.svelte";
  import SchemaFromSamples from "$lib/components/SchemaFromSamples.svelte";
  import {
    type JSONContent,
    JSONEditor,
    type Validator
  } from "svelte-jsoneditor";
  import type { components } from "$lib/api-types";

  type VersionItem = components["schemas"]["EventTypeVersionItem"];
  type ImportItem = components["schemas"]["ImportItemResult"];

  let name = $state("");
  let description = $state("");
  let schema: JSONContent = $state({ json: {} });
  let active = $state(true);
  let version = $state(1);
  let metadata = $state<Record<string, string>>({});
  let versions = $state<VersionItem[]>([]);
  // Set when the save would be breaking or break templates: shown for
  // confirmation before anything is written.
  let pending = $state<ImportItem | null>(null);
  let typedName = $state("");
  let error = $state("");
  let loading = $state(true);
  let submitting = $state(false);
  // ?sample=<event id> (from an event record) or ?infer opens the helper.
  const sampleId = page.url.searchParams.get("sample") ?? "";
  let showSchemaHelper = $state(!!sampleId || page.url.searchParams.has("infer"));
  let schemaExpanded = $state(false);

  const validator: Validator | undefined = safeAjvValidator({ schema: JSONSchemaMetaSchema });
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
      version = event.version;
      metadata = event.metadata ?? {};
      await loadVersions();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to load event details');
    } finally {
      loading = false;
    }
  });

  function hasSchema() {
    const current = toJSONObject(schema);
    return !!current && Object.keys(current).length > 0;
  }

  async function loadVersions() {
    try {
      const res = unwrap(await api.GET('/v1/event-types/{name}/versions', { params: { path: { name } } }));
      versions = res.items ?? [];
    } catch {
      versions = [];
    }
  }

  // Before saving, run the change as a one-item dry-run import. It reports
  // whether the schema change is breaking for subscriptions and whose
  // templates would stop rendering, without writing anything.
  async function preflight(): Promise<ImportItem | null> {
    const res = unwrap(await api.POST('/v1/event-types:import', {
      body: {
        items: [{ name, description, event_schema: toJSONObject(schema), metadata, active }],
        dry_run: true,
      },
    }));
    return res.items?.[0] ?? null;
  }

  async function save(allowBreaking: boolean) {
    unwrap(await api.PATCH('/v1/event-types/{name}', {
      params: { path: { name }, query: { allow_breaking: allowBreaking } },
      body: { description, event_schema: toJSONObject(schema), active },
    }));
    goto("/events");
  }

  async function updateEvent(e: Event) {
    e.preventDefault();
    error = "";
    submitting = true;
    try {
      const item = await preflight();
      if (item && (item.blocked || (item.subscriptions?.failures?.length ?? 0) > 0)) {
        pending = item;
        typedName = "";
        return;
      }
      await save(false);
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to update event');
    } finally {
      submitting = false;
    }
  }

  async function confirmPending() {
    if (!pending) return;
    error = "";
    submitting = true;
    try {
      await save(pending.blocked ?? false);
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to update event');
    } finally {
      submitting = false;
    }
  }

  function formatDate(ts?: string | null) {
    return ts ? new Date(ts).toLocaleString() : "";
  }
</script>

<svelte:head>
  <title>Update {name || 'Event'} | Sparrow</title>
</svelte:head>

<main class="mx-auto max-w-5xl px-4 sm:px-6 py-8">
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
        <div class="h-10 bg-black/5 rounded"></div>
        <div class="h-32 bg-black/[0.03] rounded"></div>
      </div>
    </div>
  {:else}
    <form onsubmit={updateEvent} class="space-y-6">
      <section class="panel p-5 space-y-4 max-w-2xl">
        <div>
          <label for="name" class="field-label">Event Name</label>
          <input id="name" type="text" value={name} disabled class="input opacity-60" />
          <p class="text-xs text-muted mt-1.5">
            Version <span class="mono text-text">v{version}</span>. Changing the schema creates a new version and keeps this one; other fields change in place.
          </p>
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
          <label for="active" class="flex items-center gap-2 rounded px-1 py-1 hover:bg-black/5 cursor-pointer w-fit">
          <input id="active" type="checkbox" bind:checked={active} class="accent-[color:var(--color-beacon)]" />
          <span class="text-sm text-text">Active</span>
        </label>
      </section>

      <section class="panel p-5">
        <div class="flex items-center justify-between mb-3">
          <span class="field-label mb-0">JSON Schema</span>
          <button type="button" onclick={() => (showSchemaHelper = !showSchemaHelper)} class="text-xs link-beacon">
            {showSchemaHelper ? 'Hide' : 'Generate from events or a sample'}
          </button>
        </div>
        <div class="grid grid-cols-1 {showSchemaHelper ? 'lg:grid-cols-2' : ''} gap-4">
          <div>
            <ExpandableEditor bind:expanded={schemaExpanded} label="JSON Schema">
              <div class="{schemaExpanded ? 'h-full' : 'h-72'}">
                <JSONEditor bind:content={schema} {validator} />
              </div>
            </ExpandableEditor>
          </div>
          {#if showSchemaHelper}
            <SchemaFromSamples
              eventName={name}
              {sampleId}
              hasSchema={hasSchema()}
              ongenerate={(generated) => (schema = { json: generated })}
            />
          {/if}
        </div>
      </section>

      {#if error}
        <div class="panel p-4" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
          <p class="text-sm" style="color:var(--color-bad)">{error}</p>
        </div>
      {/if}

      {#if pending}
        <section class="panel p-5 space-y-3" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent)">
          <p class="eyebrow">Check before saving</p>
          {#if pending.blocked}
            <p class="text-sm text-text">
              This schema change could break {pending.subscriptions ? "subscriptions that receive" : "consumers of"} <span class="mono">{name}</span>:
            </p>
            <ul class="list-disc pl-5 text-sm" style="color:var(--color-bad)">
              {#each pending.compatibility?.reasons ?? [] as reason}
                <li class="mono text-xs">{reason}</li>
              {/each}
            </ul>
          {/if}
          {#each pending.subscriptions?.failures ?? [] as f}
            <p class="text-xs" style="color:var(--color-bad)">
              Template fails ({f.payload === "required_only" ? "when optional fields are absent" : "against the new schema"}) for subscription
              <span class="mono">{f.subscription_id.slice(0, 8)}</span> in {f.consumer}: <span class="mono">{f.error}</span>
            </p>
          {/each}
          {#if pending.subscriptions?.without_transform}
            <p class="text-xs text-muted">
              {pending.subscriptions.without_transform} subscription(s) without a transform will receive the new payload as is.
            </p>
          {/if}
          {#if pending.blocked}
            <label class="block">
              <span class="text-xs text-muted">Type <span class="mono text-text">{name}</span> to save it anyway</span>
              <input class="input mono mt-1" autocomplete="off" spellcheck="false" bind:value={typedName} />
            </label>
          {/if}
          <div class="flex items-center justify-end gap-3">
            <button type="button" class="btn btn-ghost" onclick={() => (pending = null)}>Keep editing</button>
            <button type="button" class="btn btn-danger" disabled={submitting || (pending.blocked && typedName.trim() !== name)} onclick={confirmPending}>
              Save anyway
            </button>
          </div>
        </section>
      {/if}

      <div class="flex items-center justify-end gap-3 pt-2">
        <a href="/events" class="btn btn-ghost">Cancel</a>
        <button type="submit" disabled={submitting} class="btn btn-beacon">
          {submitting ? 'Saving...' : 'Save Changes'}
        </button>
      </div>
    </form>

    <section id="history" class="panel p-5 mt-8">
      <p class="eyebrow mb-1.5">History</p>
      <h2 class="text-lg font-semibold text-text mb-3">Versions</h2>
      {#if versions.length === 0}
        <p class="text-sm text-muted">No versions recorded.</p>
      {:else}
        <ul class="space-y-2">
          {#each versions as v}
            <li class="panel-2 p-3">
              <details>
                <summary class="cursor-pointer flex flex-wrap items-center gap-3 text-sm">
                  <span class="mono text-text">v{v.version}</span>
                  {#if v.version === version}<span class="chip text-xs">current</span>{/if}
                  <span class="text-muted">{formatDate(v.created_at)}</span>
                  {#if !v.event_schema || Object.keys(v.event_schema).length === 0}
                    <span class="text-muted">no schema</span>
                  {/if}
                  {#if v.schema_defined_at}
                    <span class="text-muted">schema added {formatDate(v.schema_defined_at)}</span>
                  {/if}
                </summary>
                {#if v.event_schema && Object.keys(v.event_schema).length > 0}
                  <pre class="mono text-xs mt-3 p-3 overflow-auto max-h-72 text-text">{JSON.stringify(v.event_schema, null, 2)}</pre>
                {/if}
              </details>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {/if}
</main>
