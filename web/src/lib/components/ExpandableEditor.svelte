<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    expanded = $bindable(false),
    label = "Editor",
    children,
  }: {
    expanded?: boolean;
    label?: string;
    children: Snippet;
  } = $props();
</script>

{#if !expanded}
  <div class="relative group">
    {@render children()}
    <button
      type="button"
      onclick={() => (expanded = true)}
      class="absolute top-2 right-2 p-1 rounded opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity text-faint hover:text-text bg-panel/80 backdrop-blur-sm border border-line"
      aria-label="Expand {label} to full screen"
      title="Expand to full screen"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 8V4m0 0h4M4 4l5 5m11-1V4m0 0h-4m4 0l-5 5M4 16v4m0 0h4m-4 0l5-5m11 5v-4m0 4h-4m4 0l-5-5" />
      </svg>
    </button>
  </div>
{/if}

{#if expanded}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex flex-col bg-panel"
    role="dialog"
    aria-modal="true"
    aria-label="{label} (expanded)"
    tabindex="-1"
    onkeydown={(e) => { if (e.key === "Escape") expanded = false; }}
  >
    <div class="flex items-center justify-between px-4 py-3 border-b border-line shrink-0">
      <span class="eyebrow">{label}</span>
      <button
        type="button"
        onclick={() => (expanded = false)}
        class="btn btn-ghost !px-3 !py-1.5 text-xs"
      >
        <span class="mr-1">Esc</span>
        <svg class="w-4 h-4 inline" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
    <div class="flex-1 overflow-auto p-4">
      {@render children()}
    </div>
  </div>
{/if}
