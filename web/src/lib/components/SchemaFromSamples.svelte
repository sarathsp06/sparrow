<script lang="ts">
  import { onMount } from 'svelte';
  import { JSONEditor, Mode, type JSONContent } from 'svelte-jsoneditor';
  import { api, unwrap } from '$lib/services';
  import { analyzeSamples, formatAPIError, inferSchema, timeAgo } from '$lib/utils';
  import type { components } from '$lib/api-types';

  type EventOccurrenceItem = components['schemas']['EventOccurrenceItem'];

  interface Props {
    /** Event type whose stored events can be used as samples. */
    eventName: string;
    /** Preselect this event occurrence (e.g. from "Use as schema sample"). */
    sampleId?: string;
    /** Whether the editor already holds a schema; generating then asks before replacing it. */
    hasSchema: boolean;
    ongenerate: (schema: Record<string, any>) => void;
  }

  let { eventName, sampleId = '', hasSchema, ongenerate }: Props = $props();

  const RECENT_LIMIT = 20;

  let source = $state<'events' | 'paste'>('events');
  let occurrences = $state<EventOccurrenceItem[]>([]);
  let loadingEvents = $state(true);
  let eventsError = $state('');
  let selected = $state<Set<string>>(new Set());
  // The sample in the editor: the one ticked event (edits are used as-is) or a pasted payload.
  let sample: JSONContent = $state({ json: {} });
  let error = $state('');
  let notice = $state('');
  let pendingSchema = $state<Record<string, any> | null>(null);

  let picked = $derived(occurrences.filter((o) => selected.has(o.event_id)));
  // With several events ticked, show how each field showed up instead of one payload.
  let coverage = $derived(picked.length > 1 ? analyzeSamples(picked.map((o) => o.payload)).fields : null);

  onMount(loadEvents);

  async function loadEvents() {
    loadingEvents = true;
    eventsError = '';
    try {
      const res = unwrap(await api.GET('/v1/events', {
        params: { query: { event: eventName, limit: RECENT_LIMIT } },
      }));
      let items = res.items ?? [];
      // A preselected event goes first so it is visibly the one ticked.
      if (sampleId) {
        let one = items.find((o) => o.event_id === sampleId);
        one ??= unwrap(await api.GET('/v1/events/{event_id}', {
          params: { path: { event_id: sampleId } },
        }));
        if (one.event === eventName) items = [one, ...items.filter((o) => o.event_id !== sampleId)];
      }
      occurrences = items;
      const initial = items[0];
      if (initial) setSelection([initial.event_id]);
      else source = 'paste';
    } catch (e: any) {
      eventsError = formatAPIError(e, 'Failed to load recent events');
      source = 'paste';
    } finally {
      loadingEvents = false;
    }
  }

  function setSelection(ids: string[]) {
    selected = new Set(ids);
    error = '';
    notice = '';
    pendingSchema = null;
    if (ids.length === 1) {
      const o = occurrences.find((x) => x.event_id === ids[0]);
      if (o) sample = { json: o.payload };
    }
  }

  function toggle(id: string) {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelection(occurrences.filter((o) => next.has(o.event_id)).map((o) => o.event_id));
  }

  function readSample(): unknown {
    if ('text' in sample && typeof sample.text === 'string') return JSON.parse(sample.text);
    if ('json' in sample) return sample.json;
    throw new Error('enter a JSON sample');
  }

  function generate() {
    error = '';
    notice = '';
    let samples: unknown[];
    try {
      samples = source === 'events' && picked.length > 1 ? picked.map((o) => o.payload) : [readSample()];
    } catch (e: any) {
      error = `Invalid JSON: ${e.message}`;
      return;
    }
    const schema = inferSchema(samples);
    notice = samples.length === 1
      ? 'Generated from 1 sample. Review it before saving.'
      : `Generated from ${samples.length} events. Review it before saving.`;
    if (hasSchema) pendingSchema = schema;
    else ongenerate(schema);
  }

  function confirmReplace() {
    if (pendingSchema) ongenerate(pendingSchema);
    pendingSchema = null;
  }

  function cancelReplace() {
    pendingSchema = null;
    notice = '';
  }

  function preview(payload: unknown) {
    const s = JSON.stringify(payload);
    return s.length > 80 ? s.slice(0, 80) + '…' : s;
  }

  const tabClass = (on: boolean) =>
    `px-3 py-1.5 text-xs rounded-md mono transition-colors ${on ? 'bg-beacon text-[#2a1a02] font-semibold' : 'text-muted border border-line hover:text-text hover:bg-black/5'}`;
</script>

<div class="panel-2 p-4 space-y-3">
  <div class="flex gap-1" role="group" aria-label="Sample source">
    <button type="button" class={tabClass(source === 'events')} aria-pressed={source === 'events'} onclick={() => (source = 'events')}>
      Recent events
    </button>
    <button type="button" class={tabClass(source === 'paste')} aria-pressed={source === 'paste'} onclick={() => (source = 'paste')}>
      Paste
    </button>
  </div>

  {#if source === 'events'}
    {#if loadingEvents}
      <div class="animate-pulse space-y-2">
        <div class="h-8 bg-black/5 rounded"></div>
        <div class="h-8 bg-black/[0.03] rounded"></div>
      </div>
    {:else if eventsError}
      <p class="text-xs" style="color:var(--color-bad)">{eventsError}</p>
    {:else if occurrences.length === 0}
      <p class="text-xs text-muted">
        No <span class="mono text-text">{eventName}</span> events yet.
        <a class="link-beacon" href="/events/push">Push one</a> or paste a sample.
      </p>
    {:else}
      <div class="flex flex-wrap items-center justify-between gap-2 text-xs">
        <span class="text-muted">
          Tick one event to use its payload, or several to merge them.
        </span>
        <span class="flex gap-3">
          <button type="button" class="link" onclick={() => setSelection([occurrences[0].event_id])}>Latest only</button>
          <button type="button" class="link" onclick={() => setSelection(occurrences.map((o) => o.event_id))}>All {occurrences.length}</button>
        </span>
      </div>
      <ul class="max-h-44 overflow-auto divide-y divide-line border border-line rounded-md" aria-label="Recent events">
        {#each occurrences as o (o.event_id)}
          <li>
            <label class="flex items-start gap-2 px-3 py-2 cursor-pointer hover:bg-black/[0.03]">
              <input
                type="checkbox"
                class="mt-0.5 accent-[color:var(--color-beacon)]"
                checked={selected.has(o.event_id)}
                onchange={() => toggle(o.event_id)}
              />
              <span class="min-w-0 flex-1">
                <span class="flex flex-wrap items-center gap-2 text-xs">
                  <span class="text-text" title={new Date(o.created_at).toLocaleString()}>{timeAgo(o.created_at)}</span>
                  <span class="chip !text-[10px]">{o.consumer}</span>
                  {#if !o.schema_valid}<span class="chip !text-[10px]" style="color:var(--color-bad)">invalid</span>{/if}
                </span>
                <span class="block mono text-[11px] text-muted truncate">{preview(o.payload)}</span>
              </span>
            </label>
          </li>
        {/each}
      </ul>

      {#if coverage}
        <div>
          <p class="text-xs text-muted mb-1.5">
            Field coverage across <span class="text-text">{picked.length}</span> events. A field missing from some becomes optional.
          </p>
          <div class="max-h-52 overflow-auto border border-line rounded-md">
            <table class="w-full text-left text-xs">
              <thead class="sticky top-0 bg-[color:var(--color-panel-2)]">
                <tr class="border-b border-line">
                  <th class="px-3 py-1.5 font-medium text-muted">Field</th>
                  <th class="px-3 py-1.5 font-medium text-muted">Type</th>
                  <th class="px-3 py-1.5 font-medium text-muted text-right">Seen</th>
                  <th class="px-3 py-1.5 font-medium text-muted"></th>
                </tr>
              </thead>
              <tbody>
                {#each coverage as f (f.path)}
                  <tr class="border-b border-line last:border-0">
                    <td class="px-3 py-1.5 mono text-text">{f.path}</td>
                    <td class="px-3 py-1.5 mono text-muted">
                      {f.types.join(' | ')}{#if f.format}<span class="text-text">&nbsp;·&nbsp;{f.format}</span>{/if}
                    </td>
                    <td
                      class="px-3 py-1.5 mono tnum text-right {f.required ? 'text-muted' : 'text-text'}"
                      title={f.path.includes('[]') ? `In ${f.seen} of ${f.of} array elements` : `In ${f.seen} of ${f.of} objects`}
                    >{f.seen}/{f.of}</td>
                    <td class="px-3 py-1.5">
                      {#if f.required}
                        <span class="text-muted">required</span>
                      {:else}
                        <span class="chip !text-[10px]">optional</span>
                      {/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {:else if picked.length === 1}
        <div class="h-52">
          <JSONEditor bind:content={sample} mode={Mode.text} />
        </div>
      {:else}
        <p class="text-xs text-muted">Tick at least one event.</p>
      {/if}
    {/if}
  {:else}
    <p class="text-muted text-xs">Paste a sample payload to generate a schema:</p>
    <div class="h-52">
      <JSONEditor bind:content={sample} mode={Mode.text} />
    </div>
  {/if}

  {#if error}
    <p class="text-xs" style="color:var(--color-bad)">{error}</p>
  {/if}

  {#if pendingSchema}
    <div class="flex flex-wrap items-center gap-2 text-xs">
      <span class="text-text">Replace the current schema with the generated one?</span>
      <button type="button" class="btn btn-danger !px-3 !py-1.5" onclick={confirmReplace}>Replace</button>
      <button type="button" class="btn btn-ghost !px-3 !py-1.5" onclick={cancelReplace}>Cancel</button>
    </div>
  {:else}
    <div class="flex flex-wrap items-center gap-3">
      <button
        type="button"
        class="btn btn-beacon !px-3 !py-1.5"
        disabled={source === 'events' && picked.length === 0}
        onclick={generate}
      >
        Generate Schema
      </button>
      {#if notice}<span class="text-xs text-muted">{notice}</span>{/if}
    </div>
  {/if}
</div>
