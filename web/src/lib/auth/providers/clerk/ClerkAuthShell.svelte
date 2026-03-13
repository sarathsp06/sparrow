<!--
  ClerkAuthShell — Clerk-specific auth wrapper.

  Provides sign-in/sign-out UI and gates content behind authentication
  via Clerk's <Show> component.

  When isPublicRoute is true, signed-out users can still see the page
  content (e.g., the landing page) with a sign-in button in the header.
  Protected routes always require authentication.

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
    SignInButton,
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
    <OrgGate {header}>
      {@render children()}
    </OrgGate>
  </Show>

  <Show when="signed-out">
    <header
      class="sticky flex w-full left-0 top-0 items-center justify-between px-8 py-2 z-999 bg-white/20 backdrop-blur-md border-b border-gray-100 shadow-xs"
    >
      {@render header()}
      <nav class="px-2 flex items-center flex-wrap gap-2 md:gap-8 text-lg font-medium">
        <SignInButton />
      </nav>
    </header>

    {#if isPublicRoute}
      <!-- Public route: show page content without authentication -->
      {@render children()}
    {:else}
      <!-- Protected route: prompt sign-in -->
      <div class="flex items-center justify-center min-h-[60vh]">
        <div class="text-center">
          <h1 class="text-3xl font-bold text-gray-700 mb-4">Welcome to Sparrow</h1>
          <p class="text-gray-500 mb-6">Sign in to manage your webhooks and events.</p>
          <SignInButton />
        </div>
      </div>
    {/if}
  </Show>
</ClerkProvider>
