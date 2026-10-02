<script lang="ts">
  // A collapsed form section: label on the left, a one-line summary of the
  // current values on the right, fields underneath when opened.
  import type { Snippet } from "svelte";

  let {
    label,
    summary = "",
    open = $bindable(false),
    children,
  }: { label: string; summary?: string; open?: boolean; children: Snippet } = $props();
</script>

<details class="group border-t border-line pt-3" bind:open>
  <summary class="flex items-center justify-between gap-3 cursor-pointer select-none list-none [&::-webkit-details-marker]:hidden">
    <span class="field-label !mb-0 shrink-0">{label}</span>
    <span class="flex items-center gap-1 min-w-0 text-xs text-muted">
      <span class="truncate">{summary}</span>
      <span class="inline-block shrink-0 transition group-open:rotate-90">›</span>
    </span>
  </summary>
  <div class="pt-3 space-y-3">{@render children()}</div>
</details>
