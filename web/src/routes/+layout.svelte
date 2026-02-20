<script lang="ts">
  import { page } from "$app/state";
  import { goto } from "$app/navigation";
  import favicon from "$lib/assets/favicon.svg";
  import { namespaceState } from "$lib/namespace.svelte";
  import "../app.css";

  let { children } = $props();
  let editingNamespace = $state(false);
  let namespaceInput = $state(namespaceState.current);

  function handleNamespaceSubmit() {
    namespaceState.setNamespace(namespaceInput);
    editingNamespace = false;
    // Redirect to webhooks page when namespace changes to refresh context
    goto("/webhooks");
  }

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
  <div class="flex items-center gap-4 bg-gray-100 px-3 py-1 rounded-full border border-gray-200 shadow-inner">
    <span class="text-xs font-bold text-gray-400 uppercase tracking-tighter">Namespace:</span>
    {#if editingNamespace}
      <input
        type="text"
        bind:value={namespaceInput}
        onkeydown={(e) => e.key === "Enter" && handleNamespaceSubmit()}
        onblur={() => (editingNamespace = false)}
        class="bg-transparent border-none text-sm font-bold text-primary focus:ring-0 p-0 w-24"
        autofocus
      />
    {:else}
      <button
        onclick={() => {
          namespaceInput = namespaceState.current;
          editingNamespace = true;
        }}
        class="text-sm font-bold text-primary hover:text-blue-700 transition cursor-pointer"
      >
        {namespaceState.current}
      </button>
    {/if}
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
