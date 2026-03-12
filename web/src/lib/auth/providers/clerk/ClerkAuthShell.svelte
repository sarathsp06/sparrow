<!--
  ClerkAuthShell — Clerk-specific auth wrapper.

  Provides sign-in/sign-out UI (SignInButton, UserButton, OrganizationSwitcher)
  and gates content behind authentication via Clerk's <Show> component.

  This component also mounts ClerkAuthBridge to inject the Clerk session JWT
  into API requests via the auth.ts token provider abstraction.
-->
<script lang="ts">
  import type { Snippet } from "svelte";
  import { ClerkProvider, Show, SignInButton, UserButton, OrganizationSwitcher } from "svelte-clerk/client";
  import ClerkAuthBridge from "$lib/ClerkAuthBridge.svelte";

  let {
    publishableKey,
    header,
    children,
  }: {
    publishableKey: string;
    header: Snippet;
    children: Snippet;
  } = $props();
</script>

<ClerkProvider {publishableKey}>
  <ClerkAuthBridge />

  <header
    class="sticky flex w-full left-0 top-0 items-center justify-between px-8 py-2 z-999 bg-white/20 backdrop-blur-md border-b border-gray-100 shadow-xs"
  >
    {@render header()}
    <nav class="px-2 flex items-center flex-wrap gap-2 md:gap-8 text-lg font-medium">
      <a href="/webhooks" class="hover:text-primary transition">Webhooks</a>
      <a href="/events" class="hover:text-primary transition">Events</a>
      <a href="/health" class="hover:text-primary transition">Health</a>
      <Show when="signed-in">
        <OrganizationSwitcher />
        <UserButton />
      </Show>
      <Show when="signed-out">
        <SignInButton />
      </Show>
    </nav>
  </header>

  <Show when="signed-in">
    {@render children()}
  </Show>
  <Show when="signed-out">
    <div class="flex items-center justify-center min-h-[60vh]">
      <div class="text-center">
        <h1 class="text-3xl font-bold text-gray-700 mb-4">Welcome to Sparrow</h1>
        <p class="text-gray-500 mb-6">Sign in to manage your webhooks and events.</p>
        <SignInButton />
      </div>
    </div>
  </Show>
</ClerkProvider>
