<script lang="ts">
  import { apiBase, browserSignInName } from "$lib/services";
  import { auth } from "./auth.svelte";
  import { exchangeKey, rejectMessage } from "./client";

  // Shown when the Sparrow API answers 401: the server requires a key and
  // this browser has none, a wrong one, or a revoked/expired token.
  // A pasted master key is swapped for a named browser token, so the master
  // key itself is never stored; a pasted token is stored as is.
  let key = $state("");
  let error = $state("");
  let busy = $state(false);
  let input = $state<HTMLInputElement | null>(null);

  $effect(() => {
    if (auth.required) input?.focus();
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    const value = key.trim();
    if (!value || busy) return;
    busy = true;
    error = "";
    try {
      const res = await exchangeKey(apiBase, value, browserSignInName());
      if (res.kind === "rejected") {
        error = rejectMessage(res.reason, true);
        return;
      }
      if (res.kind === "token") auth.save(res.secret, res.tokenId);
      else auth.save(value);
      // Reload so every page re-fetches with the new credential.
      location.reload();
    } catch {
      error = "Could not reach the Sparrow server. Check the connection and try again.";
    } finally {
      busy = false;
    }
  }
</script>

{#if auth.required}
  <div class="fixed inset-0 z-[60] flex items-center justify-center p-4" role="dialog" aria-modal="true" aria-labelledby="signin-title">
    <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" role="presentation"></div>
    <form class="panel relative max-w-md w-full p-6" onsubmit={submit}>
      <span class="eyebrow" style="color:var(--color-beacon)">Authentication</span>
      <h3 id="signin-title" class="text-lg font-semibold text-text mt-2 mb-2">Sign in to Sparrow</h3>
      <p class="text-sm text-muted mb-4 leading-relaxed">
        {auth.message || rejectMessage(undefined, auth.key !== "")}
      </p>
      <label class="block mb-2">
        <span class="field-label">API key or access token</span>
        <input bind:this={input} bind:value={key} type="password" autocomplete="current-password" class="input mono" aria-label="API key or access token" />
      </label>
      <p class="text-xs text-faint mb-5 leading-relaxed">
        A master key is exchanged for a token for this browser, so the key itself is never stored. You can also open an invite link instead.
      </p>
      {#if error}
        <p class="text-sm mb-4" role="alert" style="color:var(--color-bad)">{error}</p>
      {/if}
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-ghost" onclick={() => auth.dismiss()}>Cancel</button>
        <button type="submit" class="btn btn-beacon" disabled={!key.trim() || busy}>{busy ? "Signing in…" : "Sign in"}</button>
      </div>
    </form>
  </div>
{/if}
