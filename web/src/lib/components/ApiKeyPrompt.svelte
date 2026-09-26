<script lang="ts">
  import { auth } from "$lib/auth.svelte";

  // Shown when the Sparrow API answers 401: the server has SPARROW_API_KEY set
  // and this UI either has no key (standalone deployment) or a stale one.
  let key = $state("");
  let input = $state<HTMLInputElement | null>(null);
  const rejected = auth.key !== "";

  $effect(() => {
    if (auth.required) input?.focus();
  });

  function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!key.trim()) return;
    auth.save(key);
    // Reload so every page re-fetches with the new key.
    location.reload();
  }
</script>

{#if auth.required}
  <div class="fixed inset-0 z-[60] flex items-center justify-center p-4" role="dialog" aria-modal="true" aria-labelledby="apikey-title">
    <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" role="presentation"></div>
    <form class="panel relative max-w-md w-full p-6" onsubmit={submit}>
      <span class="eyebrow" style="color:var(--color-beacon)">Authentication</span>
      <h3 id="apikey-title" class="text-lg font-semibold text-text mt-2 mb-2">API key required</h3>
      <p class="text-sm text-muted mb-4 leading-relaxed">
        {#if rejected}
          The Sparrow server rejected the current API key. Enter the server's <span class="mono">SPARROW_API_KEY</span> to continue.
        {:else}
          This Sparrow server requires an API key. Enter its <span class="mono">SPARROW_API_KEY</span> to continue.
        {/if}
        It is kept in this browser only.
      </p>
      <label class="block mb-6">
        <span class="field-label">API key</span>
        <input bind:this={input} bind:value={key} type="password" autocomplete="current-password" class="input mono" aria-label="API key" />
      </label>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-ghost" onclick={() => auth.dismiss()}>Cancel</button>
        <button type="submit" class="btn btn-beacon" disabled={!key.trim()}>Save key</button>
      </div>
    </form>
  </div>
{/if}
