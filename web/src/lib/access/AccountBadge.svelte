<script lang="ts">
  import { onMount } from "svelte";
  import { api, signOut } from "$lib/services";
  import { auth } from "./auth.svelte";

  // "Signed in as …" in the sidebar, from GET /v1/whoami.
  let who = $state<{ auth_enabled: boolean; name: string; master_key: boolean } | null>(null);
  let signingOut = $state(false);

  onMount(async () => {
    const res = await api.GET("/v1/whoami").catch(() => null);
    if (res?.data) who = res.data;
  });

  async function onSignOut() {
    signingOut = true;
    await signOut();
  }
</script>

{#if who}
  <div class="flex items-center gap-2 px-1 text-xs" data-testid="account-badge">
    {#if !who.auth_enabled}
      <span class="w-2 h-2 rounded-full shrink-0" style="background:var(--color-warn)"></span>
      <span class="text-muted truncate" title="SPARROW_API_KEY is not set: anyone who can reach Sparrow has full access">Authentication off</span>
    {:else}
      <span class="w-2 h-2 rounded-full shrink-0" style="background:var(--color-ok)"></span>
      <span class="text-muted truncate" title={who.name}>Signed in as <span class="text-text font-medium">{who.name}</span></span>
    {/if}
    {#if auth.hasStoredKey}
      <button onclick={onSignOut} disabled={signingOut} class="ml-auto shrink-0 link text-xs" title="Remove this browser's credential{auth.ownedTokenId ? ' and revoke its token' : ''}">
        {signingOut ? "…" : "Sign out"}
      </button>
    {/if}
  </div>
{/if}
