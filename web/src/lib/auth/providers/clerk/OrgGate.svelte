<!--
  OrgGate — gates app content behind an active Organization.

  Must be rendered inside <ClerkProvider> so it can read Clerk context.

  • If the user has an active Organization → render the full app (header + nav + children).
  • If the user has NO active Organization → show a prompt to create or select one.

  The OrganizationList component (with hidePersonal) ensures users cannot
  bypass organization selection. Each Organization is a tenant.
-->
<script lang="ts">
  import type { Snippet } from "svelte";
  import {
    useClerkContext,
    OrganizationList,
    OrganizationSwitcher,
    UserButton,
  } from "svelte-clerk/client";

  let {
    header,
    children,
  }: {
    header: Snippet;
    children: Snippet;
  } = $props();

  const ctx = useClerkContext();
  const hasOrg = $derived(!!ctx.auth.orgId);

  // Track the active org so we can detect switches.
  // When the user picks a different Organisation via the switcher,
  // force a full page reload so every page re-fetches with the new tenant.
  let prevOrgId: string | null | undefined = $state(undefined);

  $effect(() => {
    const currentOrgId = ctx.auth.orgId;
    if (prevOrgId === undefined) {
      // First run — just record the initial value.
      prevOrgId = currentOrgId ?? null;
      return;
    }
    if (currentOrgId && currentOrgId !== prevOrgId) {
      prevOrgId = currentOrgId;
      // Hard reload clears all component state and re-fetches data
      // for the newly-active tenant.
      window.location.href = "/";
    }
  });
</script>

{#if hasOrg}
  <!-- User has an active Organization — show full app -->
  <header
    class="sticky flex w-full left-0 top-0 items-center justify-between px-8 py-2 z-999 bg-white/20 backdrop-blur-md border-b border-gray-100 shadow-xs"
  >
    {@render header()}
    <nav class="px-2 flex items-center flex-wrap gap-2 md:gap-8 text-lg font-medium">
      <a href="/webhooks" class="hover:text-primary transition">Webhooks</a>
      <a href="/events" class="hover:text-primary transition">Events</a>
      <a href="/health" class="hover:text-primary transition">Health</a>
      <a href="/team" class="hover:text-primary transition">Team</a>
      <OrganizationSwitcher hidePersonal={true} />
      <UserButton />
    </nav>
  </header>

  {@render children()}
{:else}
  <!-- No active Organization — prompt user to create or join one -->
  <header
    class="sticky flex w-full left-0 top-0 items-center justify-between px-8 py-2 z-999 bg-white/20 backdrop-blur-md border-b border-gray-100 shadow-xs"
  >
    {@render header()}
    <nav class="px-2 flex items-center flex-wrap gap-2 md:gap-8 text-lg font-medium">
      <UserButton />
    </nav>
  </header>

  <div class="flex items-center justify-center min-h-[60vh]">
    <div class="text-center">
      <h1 class="text-3xl font-bold text-gray-700 mb-4">Select an Organisation</h1>
      <p class="text-gray-500 mb-6">
        Create a new organisation or select an existing one to continue.
      </p>
      <div class="flex justify-center">
        <OrganizationList
          hidePersonal={true}
          afterSelectOrganizationUrl="/"
          afterCreateOrganizationUrl="/"
        />
      </div>
    </div>
  </div>
{/if}
