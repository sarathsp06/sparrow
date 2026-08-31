<script lang="ts">
  interface Props {
    currentPage: number;
    totalPages: number;
    totalCount: number;
    pageSize: number;
    onPageChange: (pageNum: number) => void;
    itemLabel?: string;
  }

  let { currentPage, totalPages, totalCount, pageSize, onPageChange, itemLabel = "items" }: Props = $props();

  const from = $derived((currentPage - 1) * pageSize + 1);
  const to = $derived(Math.min(currentPage * pageSize, totalCount));

  function go(n: number) {
    if (n >= 1 && n <= totalPages && n !== currentPage) onPageChange(n);
  }

  const pages = $derived(
    Array.from({ length: totalPages }, (_, i) => i + 1).filter(
      (n) => n === 1 || n === totalPages || (n >= currentPage - 2 && n <= currentPage + 2),
    ),
  );
</script>

{#if totalPages > 1}
  <div class="mt-6 flex flex-col sm:flex-row justify-between items-center gap-3">
    <div class="eyebrow tnum" style="letter-spacing:0.1em">
      {from}–{to} <span class="text-faint">/</span> {totalCount} {itemLabel}
    </div>

    <div class="flex items-center gap-1.5">
      <button class="btn btn-ghost !px-3 !py-1.5" onclick={() => go(currentPage - 1)} disabled={currentPage === 1}>
        Prev
      </button>

      <div class="flex items-center gap-1">
        {#each pages as n, i}
          {#if i > 0 && n - pages[i - 1] > 1}
            <span class="px-1 text-faint mono">·</span>
          {/if}
          <button
            onclick={() => go(n)}
            aria-current={n === currentPage ? "page" : undefined}
            class="min-w-8 px-2.5 py-1.5 rounded-md text-sm mono tnum transition-colors {n === currentPage
              ? 'bg-beacon text-[#1a1204] font-semibold'
              : 'text-muted hover:text-text hover:bg-white/5 border border-line'}"
          >
            {n}
          </button>
        {/each}
      </div>

      <button class="btn btn-ghost !px-3 !py-1.5" onclick={() => go(currentPage + 1)} disabled={currentPage === totalPages}>
        Next
      </button>
    </div>
  </div>
{/if}
