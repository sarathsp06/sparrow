<script lang="ts">
  import { fade, fly } from "svelte/transition";
  import { apiConsole, toCurl, type ApiLogEntry } from "$lib/apiConsole.svelte";

  let { open = $bindable(false) }: { open?: boolean } = $props();
  let copiedId = $state<string | null>(null);
  const reduce = typeof window !== "undefined" && window.matchMedia("(prefers-reduced-motion: reduce)").matches;

  const METHOD_TONE: Record<string, string> = {
    GET: "var(--color-ok)",
    POST: "var(--color-beacon)",
    PUT: "var(--color-warn)",
    PATCH: "var(--color-warn)",
    DELETE: "var(--color-bad)",
  };

  async function copy(entry: ApiLogEntry) {
    try {
      await navigator.clipboard.writeText(toCurl(entry));
      copiedId = entry.id;
      setTimeout(() => { if (copiedId === entry.id) copiedId = null; }, 1500);
    } catch {
      // Fallback: noop
    }
  }

  function statusTone(status?: number): string {
    if (status === undefined) return "var(--color-idle)";
    return status < 400 ? "var(--color-ok)" : "var(--color-bad)";
  }

  function clock(time: number): string {
    return new Date(time).toLocaleTimeString([], { hour12: false });
  }
</script>

<svelte:window
  onkeydown={(e) => {
    if (e.key === "Escape" && open) open = false;
    else if ((e.code === "Backquote" || e.key === "`" || e.key === "~") && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      open = !open;
    }
  }}
/>

{#if open}
  <div class="fixed inset-0 z-50 flex justify-end" role="dialog" aria-modal="true" aria-label="API console">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div
      class="absolute inset-0 bg-black/40 backdrop-blur-[2px]"
      role="presentation"
      onclick={() => (open = false)}
      transition:fade={{ duration: reduce ? 0 : 150 }}
    ></div>

    <aside
      class="console relative w-full max-w-2xl h-full flex flex-col"
      transition:fly={{ x: 480, duration: reduce ? 0 : 220, opacity: 1 }}
    >
      <!-- terminal title bar -->
      <div class="flex items-center gap-3 px-4 h-14 border-b border-white/10 shrink-0">
        <span class="flex items-center gap-1.5" aria-hidden="true">
          <span class="w-2.5 h-2.5 rounded-full bg-[#ff5f57]"></span>
          <span class="w-2.5 h-2.5 rounded-full bg-[#febc2e]"></span>
          <span class="w-2.5 h-2.5 rounded-full bg-[#28c840]"></span>
        </span>
        <span class="eyebrow !text-[#8f887a]">api console</span>
        {#if apiConsole.entries.length > 0}
          <span class="mono text-[10.5px] px-1.5 py-0.5 rounded bg-white/10 text-[#c9c2b2] tnum">{apiConsole.entries.length}</span>
        {/if}
        <span class="flex-1"></span>
        {#if apiConsole.entries.length > 0}
          <button class="console-btn" onclick={() => apiConsole.clear()}>clear</button>
        {/if}
        <button class="console-btn" onclick={() => (open = false)} aria-label="Close console (Esc)">esc ✕</button>
      </div>

      <p class="flex items-baseline gap-2 px-4 py-2 text-[11.5px] text-[#8f887a] border-b border-white/10 shrink-0">
        <span>Every request this UI makes, as copy-ready curl. <span class="mono text-[10.5px]">X-API-Key</span> is redacted.</span>
        <a href="/docs" target="_blank" rel="noreferrer" class="ml-auto shrink-0 mono text-[10.5px] !text-beacon hover:underline">API reference ↗</a>
      </p>

      <div class="flex-1 overflow-y-auto">
        {#if apiConsole.entries.length === 0}
          <div class="h-full grid place-items-center px-8 text-center">
            <div>
              <svg class="w-8 h-8 mx-auto mb-3 text-[#5d5749]" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.6" d="m8 9 3 3-3 3m5 0h4" /><rect x="3" y="4" width="18" height="16" rx="2" stroke-width="1.6" /></svg>
              <p class="text-sm text-[#8f887a]">Nothing captured yet.</p>
              <p class="text-xs text-[#5d5749] mt-1">Do something in the UI — create a webhook, push an event — and the exact API call lands here.</p>
            </div>
          </div>
        {/if}

        {#each apiConsole.entries as entry (entry.id)}
          <article class="px-4 py-3 border-b border-white/[0.06] group">
            <header class="flex items-center gap-2 mb-2 min-w-0">
              <span
                class="mono text-[10.5px] font-bold px-1.5 py-0.5 rounded shrink-0"
                style="color:{METHOD_TONE[entry.method] ?? 'var(--color-idle)'}; background:color-mix(in srgb, {METHOD_TONE[entry.method] ?? 'var(--color-idle)'} 14%, transparent)"
              >{entry.method}</span>
              <span class="mono text-xs text-[#e8e2d4] truncate" title={entry.url}>{new URL(entry.url).pathname}{new URL(entry.url).search}</span>
              {#if (entry.count ?? 1) > 1}
                <span class="mono text-[10px] px-1.5 py-px rounded-full bg-white/10 text-[#c9c2b2] tnum shrink-0" title="{entry.count} consecutive identical requests, coalesced">×{entry.count}</span>
              {/if}
              <span class="flex-1"></span>
              {#if entry.status !== undefined}
                <span class="mono text-[10.5px] font-bold tnum shrink-0" style="color:{statusTone(entry.status)}">{entry.status}</span>
              {:else}
                <span class="w-1.5 h-1.5 rounded-full animate-pulse shrink-0" style="background:{statusTone(entry.status)}" title="In flight"></span>
              {/if}
              {#if entry.durationMs !== undefined}
                <span class="mono text-[10.5px] text-[#8f887a] tnum shrink-0">{entry.durationMs}ms</span>
              {/if}
              <span class="mono text-[10.5px] text-[#5d5749] tnum shrink-0">{clock(entry.time)}</span>
            </header>

            <div class="relative">
              <pre class="curl-block">{#each toCurl(entry).split("\n") as line, i}{#if i > 0}{"\n"}{/if}<span
                >{#if i === 0}<span class="text-[#5d5749] select-none">$ </span><span class="text-beacon">curl</span><span class="text-[#e8e2d4]">{line.slice(4)}</span
                >{:else if line.trimStart().startsWith("--")}<span class="text-[#8f887a]">{line}</span
                >{:else}<span class="text-[#c9e4b4]">{line}</span>{/if}</span
              >{/each}</pre>
              <button
                class="console-btn absolute top-2 right-2 opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity"
                onclick={() => copy(entry)}
              >
                {copiedId === entry.id ? "copied ✓" : "copy"}
              </button>
            </div>
          </article>
        {/each}
      </div>
    </aside>
  </div>
{/if}

<style>
  /* The one deliberately dark surface in the app: a warm-black terminal. */
  .console {
    background: #171512;
    box-shadow: -12px 0 40px rgba(0, 0, 0, 0.35);
  }
  .console-btn {
    font-family: var(--font-mono);
    font-size: 10.5px;
    letter-spacing: 0.08em;
    padding: 4px 8px;
    border-radius: 6px;
    color: #8f887a;
    background: rgba(255, 255, 255, 0.06);
    border: 1px solid rgba(255, 255, 255, 0.08);
    transition: color 120ms, background 120ms;
  }
  .console-btn:hover {
    color: #e8e2d4;
    background: rgba(255, 255, 255, 0.12);
  }
  .curl-block {
    font-family: var(--font-mono);
    font-size: 11px;
    line-height: 1.55;
    background: #100e0c;
    border: 1px solid rgba(255, 255, 255, 0.07);
    border-radius: 8px;
    padding: 10px 12px;
    overflow-x: auto;
    white-space: pre;
  }
</style>
