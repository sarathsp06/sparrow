<script lang="ts">
  // What one event type delivers: its description, payload fields (from the
  // registered JSON Schema), an example payload and the raw schema. Shown
  // next to every event picker so a subscriber knows what they sign up for.
  import type { components } from "#lib/api-types.js";
  import { hasSchema, schemaFields } from "#lib/schema-fields.js";
  import { timeAgo } from "#lib/utils.js";

  type EventTypeItem = components["schemas"]["EventTypeItem"];

  let {
    event,
    // Admin only: links an empty description to the event's edit page.
    editable = false,
    // false inside a picker row, which already shows the name.
    showName = true,
  }: { event: EventTypeItem; editable?: boolean; showName?: boolean } = $props();

  type Tab = "fields" | "example" | "schema";
  let tab = $state<Tab>("fields");
  let copied = $state(false);

  const withSchema = $derived(hasSchema(event.event_schema));
  const fields = $derived(schemaFields(event.event_schema));
  const sample = $derived(
    event.sample_payload && Object.keys(event.sample_payload).length ? JSON.stringify(event.sample_payload, null, 2) : "",
  );
  // Without a schema there is nothing to tabulate: open on the example.
  const shown = $derived<Tab>(!withSchema && tab !== "example" ? "example" : tab);

  async function copySample() {
    try {
      await navigator.clipboard.writeText(sample);
      copied = true;
      setTimeout(() => (copied = false), 1500);
    } catch {
      // Clipboard blocked: the text is still selectable.
    }
  }
</script>

<div class="space-y-3 min-w-0" data-testid="event-type-details">
  <div>
    <div class="flex items-baseline gap-2 flex-wrap">
      {#if showName}<h4 class="mono text-sm text-text break-all">{event.name}</h4>{/if}
      <span class="text-xs text-faint">v{event.version}{event.updated_at ? ` · updated ${timeAgo(event.updated_at)}` : ""}</span>
    </div>
    {#if event.description}
      <p class="text-sm text-muted mt-1">{event.description}</p>
    {:else if editable}
      <p class="text-xs text-faint mt-1">
        No description yet. <a class="link-beacon" href="/events/{encodeURIComponent(event.name)}/update">Add one</a> so subscribers know when this event fires.
      </p>
    {/if}
  </div>

  {#if !withSchema && !sample}
    <p class="text-xs text-faint" data-testid="event-undocumented">This event's payload is not documented yet: no schema or example is registered.</p>
  {:else}
  {#if withSchema}
  <div class="flex gap-1 text-xs border-b border-line" role="tablist">
    {#each [["fields", "Payload fields"], ["example", "Example"], ["schema", "JSON Schema"]] as [key, label]}
      <button
        type="button"
        role="tab"
        aria-selected={shown === key}
        onclick={() => (tab = key as Tab)}
        class="px-2 py-1.5 -mb-px border-b-2 whitespace-nowrap transition-colors {shown === key ? 'border-beacon text-text' : 'border-transparent text-muted hover:text-text'}"
      >{label}</button>
    {/each}
  </div>
  {:else}
    <p class="text-xs text-faint">No schema is registered for this event, so its payloads are not validated. A typical payload:</p>
  {/if}

  {#if shown === "fields"}
    {#if fields.length === 0}
      <p class="text-xs text-faint">The schema declares no fields.</p>
    {:else}
      <div class="max-h-60 overflow-auto -mx-1">
        <table class="w-full text-xs">
          <thead>
            <tr class="text-left text-faint">
              <th class="font-normal px-1 pb-1">Field</th>
              <th class="font-normal px-1 pb-1">Type</th>
            </tr>
          </thead>
          <tbody>
            {#each fields as f (f.path)}
              <tr class="border-t border-line align-top">
                <td class="px-1 py-1" style="padding-left:{0.25 + f.depth * 0.9}rem">
                  <span class="mono text-text break-all">{f.path.split(".").pop()}</span>
                  {#if f.required}<span class="text-[10px] uppercase tracking-wide ml-1" style="color:var(--color-warn)">required</span>{/if}
                  {#if f.description}<span class="block text-muted">{f.description}</span>{/if}
                </td>
                <td class="px-1 py-1 mono text-muted whitespace-nowrap">
                  {f.type}
                  {#if f.enumValues}<span class="block text-faint whitespace-normal">{f.enumValues.map((v) => JSON.stringify(v)).join(", ")}</span>{/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  {:else if shown === "example"}
    {#if sample}
      <div class="relative">
        <button type="button" onclick={copySample} class="absolute top-1.5 right-1.5 text-xs link">{copied ? "Copied" : "Copy"}</button>
        <pre class="panel-2 mono text-xs p-3 max-h-60 overflow-auto whitespace-pre" data-testid="event-sample">{sample}</pre>
      </div>
    {:else}
      <p class="text-xs text-faint">No example payload is registered for this event.</p>
    {/if}
  {:else}
    <pre class="panel-2 mono text-xs p-3 max-h-60 overflow-auto whitespace-pre">{JSON.stringify(event.event_schema, null, 2)}</pre>
  {/if}
  {/if}
</div>
