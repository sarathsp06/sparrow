<script lang="ts">
  import { page } from "$app/state";
  import { onMount, onDestroy } from "svelte";
  import favicon from "$lib/assets/favicon.svg";
  import Pulse from "$lib/components/Pulse.svelte";
  import { namespaceStore } from "$lib/namespace.svelte";
  import { api, unwrap } from "$lib/services";
  import { pulseStore } from "$lib/pulse.svelte";
  import "../app.css";

  let { children } = $props();

  let sidebarOpen = $state(false);
  let fleet = $state<{ tone: string; label: string } | null>(null);

  const nav = [
    { href: "/webhooks", label: "Webhooks", d: "M9 7a4 4 0 1 1 4 4l-2 3.5M15 17a4 4 0 1 1-4-4M7.5 13.5 5 17a4 4 0 1 0 4 2" },
    { href: "/events", label: "Events", d: "M13 2 3 14h7v8l10-12h-7z" },
    { href: "/deliveries", label: "Deliveries", d: "M22 2 11 13M22 2l-7 20-4-9-9-4 20-7z" },
    { href: "/health", label: "Health", d: "M3 12h4l2 6 4-14 3 10 2-2h3" },
  ];

  function isActive(href: string): boolean {
    const id = page.route.id ?? "";
    return id === href || id.startsWith(href + "/");
  }

  async function refreshTelemetry() {
    try {
      const [sum, stats] = await Promise.all([
        api.GET("/v1/health-summary"),
        api.GET("/v1/stats"),
      ]);
      const s = unwrap(sum);
      const bad = s.unhealthy_count ?? 0;
      const warn = s.degraded_count ?? 0;
      if (bad > 0) fleet = { tone: "bad", label: `${bad} unhealthy` };
      else if (warn > 0) fleet = { tone: "warn", label: `${warn} degraded` };
      else fleet = { tone: "ok", label: "All systems go" };

      const st = unwrap(stats);
      pulseStore.set({
        ok: st.successful_deliveries ?? 0,
        warn: 0,
        bad: st.failed_deliveries ?? 0,
        pending: st.pending_deliveries ?? 0,
      });
    } catch {
      fleet = { tone: "idle", label: "Status unknown" };
    }
  }

  let telemetryTimer: ReturnType<typeof setInterval>;
  onMount(() => {
    refreshTelemetry();
    telemetryTimer = setInterval(refreshTelemetry, 10000);
  });
  onDestroy(() => clearInterval(telemetryTimer));
</script>

{#snippet rail()}
  <div class="flex flex-col h-full">
    <a href="/webhooks" onclick={() => (sidebarOpen = false)} class="flex items-center gap-3 px-4 h-16 border-b border-line shrink-0 group">
      <span class="grid place-items-center w-9 h-9 rounded-lg border border-line bg-panel-2 shadow-[inset_0_1px_0_rgba(255,255,255,0.05)] group-hover:border-beacon/60 transition-colors">
        <img src={favicon} alt="" class="w-5 h-5" />
      </span>
      <span class="flex flex-col leading-none">
        <span class="font-display font-bold tracking-[0.18em] text-text text-sm">SPARROW</span>
        <span class="eyebrow mt-1" style="font-size:9.5px">dispatch console</span>
      </span>
    </a>

    <nav class="flex flex-col gap-1 p-3">
      {#each nav as item}
        <a
          href={item.href}
          onclick={() => (sidebarOpen = false)}
          aria-current={isActive(item.href) ? "page" : undefined}
          class="relative flex items-center gap-3 px-3 py-2 rounded-lg transition-colors {isActive(item.href) ? 'bg-white/5 text-text' : 'text-muted hover:text-text hover:bg-white/[0.03]'}"
        >
          {#if isActive(item.href)}<span class="absolute left-0 top-1.5 bottom-1.5 w-0.5 rounded-full bg-beacon shadow-[0_0_8px_rgba(242,169,59,0.7)]"></span>{/if}
          <svg class="w-4.5 h-4.5 shrink-0" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d={item.d} /></svg>
          <span class="text-sm">{item.label}</span>
        </a>
      {/each}
    </nav>

    <div class="px-3">
      <a href="/events/push" onclick={() => (sidebarOpen = false)} class="btn btn-beacon w-full justify-center">
        <span class="text-base leading-none">+</span> Push Event
      </a>
    </div>

    <div class="flex-1"></div>

    <div class="p-3 border-t border-line space-y-3">
      <label class="block">
        <span class="field-label !mb-1.5">Namespace</span>
        <span class="relative flex items-center">
          <span class="absolute left-2.5 text-faint" aria-hidden="true">
            <svg class="w-3.5 h-3.5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8"><path d="M4 7h16M4 12h16M4 17h10" stroke-linecap="round"/></svg>
          </span>
          <input
            list="ns-options"
            class="input !pl-8"
            aria-label="Active namespace"
            value={namespaceStore.value}
            onchange={(e) => (namespaceStore.value = e.currentTarget.value)}
          />
          <datalist id="ns-options">
            {#each namespaceStore.options as n}<option value={n}></option>{/each}
          </datalist>
        </span>
      </label>

      <a href="/health" onclick={() => (sidebarOpen = false)} class="flex items-center gap-2 px-1 py-1 rounded-md text-xs text-muted hover:text-text transition-colors" title="Fleet health">
        <span class="relative flex w-2 h-2">
          {#if fleet?.tone === "ok" || fleet?.tone === "warn"}
            <span class="absolute inline-flex w-full h-full rounded-full opacity-60 animate-ping" style="background:var(--color-{fleet?.tone})"></span>
          {/if}
          <span class="relative inline-flex w-2 h-2 rounded-full" style="background:var(--color-{fleet?.tone ?? 'idle'})"></span>
        </span>
        <span class="mono">{fleet?.label ?? "…"}</span>
      </a>

      <div class="flex items-center gap-1 text-muted">
        <a href="/docs" target="_blank" rel="noreferrer" class="flex-1 text-center px-2 py-1.5 rounded-md text-xs hover:text-text hover:bg-white/5 transition-colors">Docs</a>
        <a href="https://github.com/sarathsp06/sparrow" target="_blank" rel="noreferrer" aria-label="Sparrow on GitHub" class="grid place-items-center w-8 h-8 rounded-md hover:text-text hover:bg-white/5 transition-colors">
          <svg viewBox="0 0 16 16" class="w-4 h-4" fill="currentColor" aria-hidden="true"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0016 8c0-4.42-3.58-8-8-8z"/></svg>
        </a>
      </div>
    </div>
  </div>
{/snippet}

<!-- Pulse: sticky heartbeat strip across the whole top -->
<div class="sticky top-0 z-40 bg-ink border-b border-line">
  <Pulse height={26} />
</div>

<div class="flex">
  <!-- desktop rail -->
  <aside class="hidden lg:block w-64 shrink-0 self-start sticky top-[27px] h-[calc(100vh-27px)] border-r border-line bg-panel/40 backdrop-blur-sm overflow-y-auto">
    {@render rail()}
  </aside>

  <!-- mobile drawer -->
  {#if sidebarOpen}
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="fixed inset-0 z-50 bg-ink/70 backdrop-blur-sm lg:hidden" role="presentation" onclick={() => (sidebarOpen = false)}></div>
    <aside class="fixed left-0 top-0 bottom-0 z-50 w-64 border-r border-line bg-panel lg:hidden overflow-y-auto">
      {@render rail()}
    </aside>
  {/if}

  <div class="flex-1 min-w-0">
    <!-- mobile top bar -->
    <div class="lg:hidden flex items-center gap-3 h-14 px-4 border-b border-line">
      <button onclick={() => (sidebarOpen = true)} aria-label="Open navigation" class="grid place-items-center w-9 h-9 rounded-md border border-line text-muted hover:text-text">
        <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><path d="M4 6h16M4 12h16M4 18h16"/></svg>
      </button>
      <span class="font-display font-bold tracking-[0.18em] text-text text-sm">SPARROW</span>
      <span class="ml-auto chip">{namespaceStore.value}</span>
    </div>

    {@render children?.()}
  </div>
</div>
