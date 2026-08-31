<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    icon?: string;
    title: string;
    description?: string;
    action?: Snippet;
  }

  let { icon = "inbox", title, description, action }: Props = $props();
</script>

<div class="flex flex-col items-center justify-center py-16 px-4 text-center">
  <div
    class="grid place-items-center w-14 h-14 rounded-xl border border-line bg-panel-2 mb-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]"
  >
    <span class="material-symbols-outlined text-3xl text-faint">{icon}</span>
  </div>
  <h3 class="text-base font-semibold text-text mb-1">{title}</h3>
  {#if description}
    <p class="text-sm text-muted max-w-sm mb-4">{description}</p>
  {/if}
  {#if action}
    {@render action()}
  {/if}
</div>
