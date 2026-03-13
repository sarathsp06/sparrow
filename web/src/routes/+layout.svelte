<script lang="ts">
  import { page } from "$app/state";
  import favicon from "$lib/assets/favicon.svg";
  import AuthShell from "$lib/auth/AuthShell.svelte";
  import "../app.css";

  let { children } = $props();

  const titles: Record<string, string> = {
    "/": "Home",
    "/webhooks": "Webhooks",
    "/webhooks/[webhookId]": "Webhook",
    "/events/[eventName]/update": "Update Event",
    "/events": "Events",
    "/health": "Health",
    "/deliveries": "Deliveries",
    "/events/push": "Push Event",
    "/webhooks/register": "Register Webhook",
    "/events/[eventName]/reports": "Event Reports"
  };

  function getTitle(): string {
    const path: string = page.route.id?.toString() || "/";
    return titles[path] || path;
  }
</script>

<svelte:head>
  <link rel="icon" href={favicon} />
</svelte:head>

<AuthShell>
  {#snippet header()}
    <div class="flex items-center gap-2">
      <a href="/" class="text-2xl font-bold text-gray-500 hover:text-blue-700/90">
        <img src={favicon} alt="favicon" class="inline-block w-12 h-12" />
      </a>
      <h2 class="text-gray-500 font-bold text-2xl hover:text-blue-700/90">
        {getTitle()}
      </h2>
    </div>
  {/snippet}

  {@render children?.()}
</AuthShell>
