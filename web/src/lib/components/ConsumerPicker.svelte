<script lang="ts">
  // Consumer input with server-side search: a tenant can have hundreds of
  // consumers, so suggestions come from GET /v1/consumers?q= as you type
  // rather than from a preloaded list. Free text is allowed, because a new
  // consumer is created by registering a webhook or pushing an event under it.
  import { api, unwrap } from "$lib/services";
  import { SYSTEM_CONSUMER, SYSTEM_CONSUMER_LABEL } from "$lib/system";

  let {
    value = $bindable(""),
    id,
    placeholder = "All consumers",
    required = false,
    live = false,
    label = "Consumer",
    class: className = "",
    onchange,
  }: {
    value?: string;
    id: string;
    placeholder?: string;
    required?: boolean;
    /** Update value on every keystroke (forms) instead of on change (filters). */
    live?: boolean;
    /** Accessible name when no visible <label for={id}> is present. */
    label?: string;
    class?: string;
    onchange?: (consumer: string) => void;
  } = $props();

  let draft = $state("");
  let options: string[] = $state([]);
  let timer: ReturnType<typeof setTimeout> | undefined;
  let seq = 0;

  // Follow value when it changes from outside (URL, reset, recipe).
  $effect(() => {
    draft = value;
  });

  function search(q: string) {
    clearTimeout(timer);
    timer = setTimeout(async () => {
      const mine = ++seq;
      try {
        const res = unwrap(await api.GET("/v1/consumers", { params: { query: { q: q.trim() || undefined, limit: 20 } } }));
        if (mine === seq) options = (res.items || []).map((i) => i.name);
      } catch {
        // Suggestions are a convenience; typing a name still works.
      }
    }, 200);
  }

  function commit() {
    const next = draft.trim();
    if (next === value) return;
    value = next;
    onchange?.(next);
  }

  function clear() {
    draft = "";
    commit();
  }
</script>

<span class="relative flex items-center {className}">
  <input
    {id}
    type="text"
    list="{id}-options"
    autocomplete="off"
    spellcheck="false"
    aria-label={label}
    class="input w-full {value && !required ? '!pr-8' : ''}"
    {placeholder}
    {required}
    bind:value={draft}
    oninput={() => {
      search(draft);
      if (live) commit();
    }}
    onfocus={() => search(draft)}
    onchange={commit}
    onkeydown={(e) => {
      if (e.key === "Enter") commit();
    }}
  />
  {#if value && !required}
    <button type="button" class="absolute right-2 text-faint hover:text-text" aria-label="Show all consumers" title="Show all consumers" onclick={clear}>
      <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M6 6l12 12M18 6L6 18" /></svg>
    </button>
  {/if}
  <datalist id="{id}-options">
    {#each options as n (n)}<option value={n} label={n === SYSTEM_CONSUMER ? SYSTEM_CONSUMER_LABEL : undefined}></option>{/each}
  </datalist>
</span>
