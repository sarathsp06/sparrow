<script lang="ts">
  import CopyableId from './CopyableId.svelte';

  // A webhook shown by what the webhooks list shows (description, else URL),
  // linking to its page by id. Falls back to the id when the response carries
  // neither (only delivery listings fill them in).
  interface Props {
    id: string;
    description?: string;
    url?: string;
  }

  let { id, description, url }: Props = $props();
  const label = $derived(description || url || '');
</script>

{#if label}
  <a
    href="/webhooks/{id}"
    onclick={(e) => e.stopPropagation()}
    class="link-beacon hover:underline text-xs block truncate max-w-[16rem] {description ? '' : 'mono'}"
    title={[description, url, id].filter(Boolean).join('\n')}
  >{label}</a>
{:else}
  <CopyableId {id} href="/webhooks/{id}" truncate={12} />
{/if}
