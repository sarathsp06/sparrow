<script lang="ts">
  // Event type picker shared by the admin register page, the consumer
  // portal and the subscription form. Each row is the name plus a one-line
  // description; clicking the name opens what the event delivers
  // (EventTypeDetails) under the row. Built for long lists: search covers
  // names and descriptions, and past GROUP_AT types the list is grouped by
  // name prefix (order.*, invoice.*) with per-group select-all.
  import type { components } from "#lib/api-types.js";
  import { groupByPrefix } from "#lib/schema-fields.js";
  import EventTypeDetails from "./EventTypeDetails.svelte";

  type EventTypeItem = components["schemas"]["EventTypeItem"];

  let {
    events,
    selected = $bindable([]),
    multiple = true,
    editable = false,
    searchId = "event-search",
  }: {
    events: EventTypeItem[];
    selected?: string[];
    // false: picking one replaces the previous pick (subscription form).
    multiple?: boolean;
    // Admin: links empty descriptions to the event's edit page.
    editable?: boolean;
    searchId?: string;
  } = $props();

  const GROUP_AT = 12;

  let search = $state("");
  let openName = $state<string | null>(null);
  // Groups the user expanded; while searching every matching group is open.
  let expanded = $state<Record<string, boolean>>({});

  const q = $derived(search.trim().toLowerCase());
  const matches = $derived(
    q ? events.filter((e) => e.name.toLowerCase().includes(q) || (e.description ?? "").toLowerCase().includes(q)) : events,
  );
  const groups = $derived(groupByPrefix(matches));
  const grouped = $derived(events.length > GROUP_AT && groupByPrefix(events).length > 1);
  const chosen = $derived(new Set(selected));

  function isOpen(prefix: string, items: EventTypeItem[]) {
    if (q || items.some((e) => e.name === openName)) return true;
    if (prefix in expanded) return expanded[prefix];
    // A single pick is always in view; multi picks show as chips instead.
    return !multiple && items.some((e) => chosen.has(e.name));
  }

  function toggle(name: string) {
    if (!multiple) {
      selected = chosen.has(name) ? [] : [name];
      return;
    }
    selected = chosen.has(name) ? selected.filter((n) => n !== name) : [...selected, name];
  }

  function toggleGroup(items: EventTypeItem[]) {
    const names = items.map((e) => e.name);
    const all = names.every((n) => chosen.has(n));
    selected = all ? selected.filter((n) => !names.includes(n)) : [...selected, ...names.filter((n) => !chosen.has(n))];
  }

  function showDetails(name: string) {
    openName = openName === name ? null : name;
  }

  // Keeps an opened row's details in view inside the scrolling list.
  function reveal(node: HTMLElement) {
    node.scrollIntoView({ block: "nearest" });
  }
</script>

{#snippet row(ev: EventTypeItem)}
  {@const open = openName === ev.name}
  <li class="rounded {open ? 'panel-2' : ''}" data-testid="event-row">
    <div class="flex items-start gap-2 px-2 py-1.5 {open ? '' : 'row-hover'} rounded">
      <input
        type={multiple ? "checkbox" : "radio"}
        name={multiple ? undefined : `${searchId}-pick`}
        checked={chosen.has(ev.name)}
        onchange={() => toggle(ev.name)}
        aria-label="Subscribe to {ev.name}"
        class="mt-1 shrink-0 accent-[color:var(--color-beacon)]"
      />
      <button
        type="button"
        class="min-w-0 flex-1 text-left"
        aria-expanded={open}
        onclick={() => showDetails(ev.name)}
      >
        <span class="flex items-center gap-1.5">
          <span class="mono text-sm text-text truncate">{ev.name}</span>
          <svg class="w-3 h-3 shrink-0 text-faint transition-transform {open ? 'rotate-90' : ''}" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><path d="M4.5 2.5 8 6l-3.5 3.5" /></svg>
        </span>
        {#if ev.description && !open}<span class="block text-xs text-faint truncate">{ev.description}</span>{/if}
      </button>
    </div>
    {#if open}
      <div class="px-2 pb-3 ml-6" use:reveal>
        <EventTypeDetails event={ev} {editable} showName={false} />
      </div>
    {/if}
  </li>
{/snippet}

<!-- inline-size containment: long one-line descriptions must not widen the host grid -->
<div class="space-y-2 min-w-0 [contain:inline-size]" data-testid="event-picker">
  {#if selected.length > 0}
    <div class="flex flex-wrap gap-1.5 max-h-20 overflow-y-auto" data-testid="event-picker-selected">
      {#each selected as name (name)}
        <span class="chip !border-beacon !text-text gap-1">
          <span class="mono text-xs">{name}</span>
          <button type="button" class="text-faint hover:text-text" aria-label="Remove {name}" onclick={() => toggle(name)}>&times;</button>
        </span>
      {/each}
    </div>
  {/if}

  {#if events.length > 6}
    <input id={searchId} type="search" placeholder="Search {events.length} events by name or description…" bind:value={search} class="input w-full" />
  {/if}

  {#if events.length === 0}
    <p class="text-xs text-faint py-2">No event types are registered yet.</p>
  {:else if matches.length === 0}
    <p class="text-xs text-faint py-2">No events match “{search}”.</p>
  {:else}
    <div class="max-h-[26rem] overflow-y-auto -mx-2">
      {#if grouped}
        {#each groups as g (g.prefix)}
          {@const open = isOpen(g.prefix, g.items)}
          {@const n = g.items.filter((e) => chosen.has(e.name)).length}
          <div class="mb-0.5">
            <div class="flex items-center gap-2 px-2 py-1.5 row-hover rounded">
              {#if multiple}
                <input
                  type="checkbox"
                  checked={n === g.items.length}
                  indeterminate={n > 0 && n < g.items.length}
                  onchange={() => toggleGroup(g.items)}
                  aria-label="Subscribe to every {g.prefix || 'other'} event"
                  class="shrink-0 accent-[color:var(--color-beacon)]"
                />
              {/if}
              <button
                type="button"
                class="flex-1 flex items-center gap-1.5 text-left min-w-0"
                aria-expanded={open}
                disabled={!!q}
                onclick={() => (expanded[g.prefix] = !open)}
              >
                <svg class="w-3 h-3 shrink-0 text-faint transition-transform {open ? 'rotate-90' : ''}" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true"><path d="M4.5 2.5 8 6l-3.5 3.5" /></svg>
                <span class="mono text-sm text-text truncate">{g.prefix ? `${g.prefix}.*` : "Other"}</span>
                <span class="text-xs text-faint ml-auto tnum shrink-0">{n ? `${n}/` : ""}{g.items.length}</span>
              </button>
            </div>
            {#if open}
              <ul class="ml-5 space-y-0.5">
                {#each g.items as ev (ev.name)}{@render row(ev)}{/each}
              </ul>
            {/if}
          </div>
        {/each}
      {:else}
        <ul class="space-y-0.5">
          {#each matches as ev (ev.name)}{@render row(ev)}{/each}
        </ul>
      {/if}
    </div>
  {/if}
</div>
