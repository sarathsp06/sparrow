<!--
  ClerkAuthShell — Clerk-specific auth wrapper.

  Provides authentication gating via Clerk's <Show> component.

  When isPublicRoute is true, signed-out users can still see the page
  content (e.g., the landing page). Protected routes automatically
  redirect signed-out users to Clerk's hosted sign-in page.

  When a signed-in user has no active Organization, OrgGate prompts them
  to create or select one before accessing the app. Each Organization
  represents a tenant — org-scoped JWTs ensure data isolation.

  This component also mounts ClerkAuthBridge to inject the Clerk session JWT
  into API requests via the auth.ts token provider abstraction.
-->
<script lang="ts">
  import type { Snippet } from "svelte";
  import {
    ClerkProvider,
    Show,
    RedirectToSignIn,
  } from "svelte-clerk/client";
  import ClerkAuthBridge from "$lib/ClerkAuthBridge.svelte";
  import OrgGate from "./OrgGate.svelte";

  let {
    publishableKey,
    header,
    children,
    isPublicRoute = false,
  }: {
    publishableKey: string;
    header: Snippet;
    children: Snippet;
    isPublicRoute?: boolean;
  } = $props();
</script>

<ClerkProvider {publishableKey}>
  <ClerkAuthBridge />

  <Show when="signed-in">
    <OrgGate {header} {isPublicRoute}>
      {@render children()}
    </OrgGate>
  </Show>

  <Show when="signed-out">
    {#if isPublicRoute}
      <!-- Public route: show page content without authentication -->
      {@render children()}
    {:else}
      <!-- Protected route: redirect to Clerk sign-in page -->
      <RedirectToSignIn signInForceRedirectUrl="/webhooks" />
    {/if}
  </Show>
</ClerkProvider>
