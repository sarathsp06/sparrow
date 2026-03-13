<!--
  Team / Organisation Management page.

  When Clerk auth is active, renders the OrganizationProfile component which
  lets admins manage members, roles, and org settings.
  When auth is "none", shows a not-available message.
-->
<script lang="ts">
  import { authConfig } from "$lib/auth/provider.js";

  // Dynamically import Clerk component only when needed
  let OrganizationProfile: any = $state(null);

  if (authConfig.type === "clerk") {
    import("svelte-clerk/client").then((mod) => {
      OrganizationProfile = mod.OrganizationProfile;
    });
  }
</script>

<div class="min-h-screen bg-gray-50">
  <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    {#if authConfig.type === "clerk"}
      {#if OrganizationProfile}
        <div class="flex justify-center">
          <OrganizationProfile />
        </div>
      {:else}
        <div class="flex justify-center py-12">
          <div class="animate-pulse text-gray-400">Loading team management...</div>
        </div>
      {/if}
    {:else}
      <div class="flex items-center justify-center min-h-[60vh]">
        <div class="text-center">
          <h1 class="text-2xl font-bold text-gray-700 mb-2">Team Management Not Available</h1>
          <p class="text-gray-500">
            Team management requires an authentication provider. Configure Clerk to enable this feature.
          </p>
        </div>
      </div>
    {/if}
  </div>
</div>
