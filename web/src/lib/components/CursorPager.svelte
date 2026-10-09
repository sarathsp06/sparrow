<script lang="ts">
  // Newer/Older paging for a CursorPages list. Hidden while everything fits
  // on one page.
  import type { CursorPages } from "$lib/cursor-pages.svelte";

  interface Props {
    pages: CursorPages;
    /** Items on the page shown. */
    shown: number;
    onchange: () => void;
    itemLabel?: string;
  }

  let { pages, shown, onchange, itemLabel = "items" }: Props = $props();
</script>

{#if pages.hasNewer || pages.hasOlder}
  <div class="mt-6 flex flex-col sm:flex-row justify-between items-center gap-3" data-testid="cursor-pager">
    <div class="eyebrow tnum" style="letter-spacing:0.1em">
      Page {pages.page} <span class="text-faint">·</span> {shown} {itemLabel}
    </div>
    <div class="flex items-center gap-1.5">
      <button class="btn btn-ghost !px-3 !py-1.5" onclick={() => { pages.newer(); onchange(); }} disabled={!pages.hasNewer}>
        Newer
      </button>
      <button class="btn btn-ghost !px-3 !py-1.5" onclick={() => { pages.older(); onchange(); }} disabled={!pages.hasOlder}>
        Older
      </button>
    </div>
  </div>
{/if}
