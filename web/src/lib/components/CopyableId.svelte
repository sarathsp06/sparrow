<script lang="ts">
  interface Props {
    id: string;
    href?: string;
    truncate?: number;
  }

  let { id, href, truncate = 8 }: Props = $props();
  let copied = $state(false);

  const display = $derived(
    truncate > 0 && id.length > truncate ? id.substring(0, truncate) + '…' : id
  );

  async function copyId(e: Event) {
    e.stopPropagation();
    e.preventDefault();
    try {
      await navigator.clipboard.writeText(id);
      copied = true;
      setTimeout(() => { copied = false; }, 1500);
    } catch {
      // Fallback: noop
    }
  }
</script>

<span class="inline-flex items-center gap-1 group/copy">
  {#if href}
    <a
      {href}
      onclick={(e) => e.stopPropagation()}
      class="mono text-xs link-beacon hover:underline"
      title={id}
    >
      {display}
    </a>
  {:else}
    <span class="mono text-xs text-muted" title={id}>{display}</span>
  {/if}
  <button
    onclick={copyId}
    class="inline-flex items-center justify-center w-4 h-4 rounded opacity-0 group-hover/copy:opacity-100 focus-visible:opacity-100 hover:bg-white/10 transition text-faint hover:text-text shrink-0"
    title="Copy full ID"
    aria-label="Copy ID to clipboard"
  >
    {#if copied}
      <svg class="w-3 h-3 text-ok" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
      </svg>
    {:else}
      <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
      </svg>
    {/if}
  </button>
</span>
