<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    icon?: string;
    title: string;
    description?: string;
    action?: Snippet;
  }

  let { icon = "inbox", title, description, action }: Props = $props();

  // ponytail: 5-icon set covers every EmptyState call site; add a path when a new icon is needed.
  const paths: Record<string, string> = {
    inbox: "M3 12h4l2 3h6l2-3h4M4.5 6h15L21 12v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6L4.5 6z",
    calendar: "M6.75 3v3m10.5-3v3M3.75 8.25h16.5M4.5 5.25h15a.75.75 0 01.75.75v13.5a.75.75 0 01-.75.75h-15a.75.75 0 01-.75-.75V6a.75.75 0 01.75-.75z",
    search: "M10.5 17.5a7 7 0 100-14 7 7 0 000 14zM21 21l-4.35-4.35",
    send: "M6 12L3.269 3.126A59.77 59.77 0 0121.485 12 59.77 59.77 0 013.27 20.876L6 12zm0 0h7.5",
    link: "M13.19 8.688a4.5 4.5 0 011.242 7.244l-4.5 4.5a4.5 4.5 0 01-6.364-6.364l1.757-1.757m3.35 8.9l1.757-1.757a4.5 4.5 0 00-6.364-6.364l-4.5 4.5a4.5 4.5 0 001.242 7.244",
    filter_alt: "M3 4.774c0-.54.384-1.005.917-1.096A48.32 48.32 0 0112 3c2.755 0 5.455.232 8.083.678.533.09.917.556.917 1.096v1.044a2.25 2.25 0 01-.659 1.591l-5.432 5.432a2.25 2.25 0 00-.659 1.591v2.927a2.25 2.25 0 01-1.244 2.013L9.75 21v-6.568a2.25 2.25 0 00-.659-1.591L3.659 7.409A2.25 2.25 0 013 5.818v-1.044z",
  };
</script>

<div class="flex flex-col items-center justify-center py-16 px-4 text-center">
  <div
    class="grid place-items-center w-14 h-14 rounded-xl border border-line bg-panel-2 mb-4 shadow-[inset_0_1px_0_rgba(255,255,255,0.04)]"
  >
    <svg class="w-6 h-6 text-faint" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d={paths[icon] ?? paths.inbox} />
    </svg>
  </div>
  <h3 class="text-base font-semibold text-text mb-1">{title}</h3>
  {#if description}
    <p class="text-sm text-muted max-w-sm mb-4">{description}</p>
  {/if}
  {#if action}
    {@render action()}
  {/if}
</div>
