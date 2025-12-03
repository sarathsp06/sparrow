<script lang="ts">
  import { page } from "$app/state";
  import favicon from "$lib/assets/favicon.svg";
  import "../app.css";

  let { children } = $props();

  const titles: Record<string, string> = {
    "/": "Home",
    "/webhooks": "Webhooks",
    "/webhooks/[webhookId]": "Webhook",
    "/events/[eventId]/update": "Update Event",
    "/events": "Events",
    "/health": "Health",
    "/deliveries": "Deliveries",
    "/events/push": "Push Event",
    "/webhooks/register": "Register Webhook",
    "/events/[eventId]/reports": "Event Reports",
    "/webhooks/[webhookId]/subscriptions": "Subscriptions"
  };
  function getTitle(): string {
    const path: string = page.route.id?.toString() || "/";
    return titles[path] || path;
  }
</script>

<svelte:head>
  <link rel="icon" href={favicon} />
</svelte:head>
<header
  class="sticky flex w-full left-0 top-0 items-center justify-between px-8 py-2 z-999 bg-white/20 backdrop-blur-md border-b border-gray-100 shadow-xs"
>
  <div class="flex items-center gap-2">
    <a href="/" class="text-2xl font-bold text-gray-500 hover:text-blue-700/90">
      <img src={favicon} alt="favicon" class="inline-block w-12 h-12" />
    </a>
    <h2 class="text-gray-500 font-bold text-2xl hover:text-blue-700/90">
      {getTitle()}
    </h2>
  </div>
  <nav
    class="px-2 flex items-center flex-wrap gap-2 md:gap-8 text-lg font-medium"
  >
    <a href="/webhooks" class="hover:text-primary transition">Webhooks</a>
    <a href="/events" class="hover:text-primary transition">Events</a>
    <a href="/health" class="hover:text-primary transition">Health</a>
    <a href="https://github.com/sarathsp06/sparrow" class="hover:text-primary transition">
      <img src="https://logo.svgcdn.com/devicon/github-original.png" alt="github" class="inline-block hover:text-primary w-8 h-8">
    </a>
  </nav>
</header>
{@render children?.()}
