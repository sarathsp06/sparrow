<!--
  AuthShell — dynamically renders the correct auth provider shell
  based on the PUBLIC_AUTH_PROVIDER env var.

  Adding a new auth provider:
  1. Create a new directory under lib/auth/providers/<name>/
  2. Create a <Name>AuthShell.svelte with the same snippet contract
     (header: Snippet, children: Snippet)
  3. Add the provider type to types.ts
  4. Add detection logic to provider.ts
  5. Add a case here
-->
<script lang="ts">
  import type { Snippet } from "svelte";
  import { authConfig } from "./provider.js";
  import ClerkAuthShell from "./providers/clerk/ClerkAuthShell.svelte";
  import NoAuthShell from "./providers/none/NoAuthShell.svelte";

  let {
    header,
    children,
    isPublicRoute = false,
  }: {
    header: Snippet;
    children: Snippet;
    isPublicRoute?: boolean;
  } = $props();
</script>

{#if authConfig.type === "clerk"}
  <ClerkAuthShell publishableKey={authConfig.options.publishableKey} {header} {isPublicRoute}>
    {@render children()}
  </ClerkAuthShell>
{:else}
  <NoAuthShell {header} {isPublicRoute}>
    {@render children()}
  </NoAuthShell>
{/if}
