<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  let {
    mergeTags = {},
    onHtml,
  }: {
    mergeTags?: Record<string, { name: string; value: string; sample: string }>;
    onHtml: (html: string) => void;
  } = $props();

  interface UnlayerInstance {
    setMergeTags(tags: Record<string, { name: string; value: string; sample: string }>): void;
    exportHtml(cb: (data: { html: string }) => void): void;
    addEventListener(event: string, cb: () => void): void;
  }

  let container: HTMLDivElement;
  let root: { unmount(): void } | null = null;
  let unlayerInstance: UnlayerInstance | null = $state(null);
  let failed = $state(false);
  let loading = $state(true);

  // Merge tags go through setMergeTags, not createEditor options: the options
  // path is unreliable in react-email-editor (tags render as "[object Object]")
  // and never updates when the sample payload changes.
  $effect(() => {
    unlayerInstance?.setMergeTags({ ...mergeTags });
  });

  onMount(async () => {
    try {
      const [{ default: React }, { createRoot }, { default: EmailEditor }] = await Promise.all([
        import('react'),
        import('react-dom/client'),
        import('react-email-editor'),
      ]);
      root = createRoot(container);
      root.render(
        React.createElement(EmailEditor, {
          minHeight: 520,
          options: { displayMode: 'email' },
          onReady: (unlayer: UnlayerInstance) => {
            loading = false;
            unlayerInstance = unlayer;
            const push = () => unlayer.exportHtml(({ html }) => onHtml(html));
            unlayer.addEventListener('design:updated', push);
            push();
          },
        }),
      );
    } catch {
      failed = true;
      loading = false;
    }
  });

  onDestroy(() => root?.unmount());
</script>

{#if loading}
  <div class="ee-loading">Loading drag-and-drop editor…</div>
{:else if failed}
  <div class="ee-fallback">
    The drag-and-drop editor could not load (it requires network access to editor.unlayer.com).
    Switch to the Plain text tab, or paste HTML into the generated template directly.
  </div>
{/if}
<div bind:this={container} class="ee-host"></div>

<style>
  .ee-host { min-height: 520px; border: 1px solid #e5e7eb; border-radius: 8px; overflow: hidden; }
  .ee-host :global(iframe) { min-width: 0 !important; }
  .ee-fallback { padding: 0.75rem; background: #fef3c7; border: 1px solid #f59e0b; border-radius: 8px; font-size: 13px; margin-bottom: 0.5rem; }
  .ee-loading { padding: 0.75rem; color: #6b7280; font-size: 13px; }
</style>
