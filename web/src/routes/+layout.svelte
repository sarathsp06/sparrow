<script lang="ts">
  import { page } from '$app/state';
  import favicon from '$lib/assets/favicon.svg';
  import '../app.css';

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
	}
	function getTitle(): string {
		const path: string = page.route.id?.toString() || "/";
		return titles[path] || path;
	}
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>
<header
  class="sticky top-0 flex items-center justify-between px-8 py-6 z-999 bg-white/90 backdrop-blur-md border-b border-gray-200 shadow-sm"
>
  <div class="flex items-center gap-2">
    <a href="/" class="text-2xl font-bold text-gray-500 hover:text-blue-700/90">{getTitle()}</a>
  </div>
  <nav class="flex gap-8 text-lg font-medium">
    <a href="/webhooks" class="hover:text-primary transition">Webhooks</a>
    <a href="/events" class="hover:text-primary transition">Events</a>
    <a href="/health" class="hover:text-primary transition">Health</a>
    <a href="#features" class="hover:text-primary transition">Features</a>
  </nav>
</header>
{@render children?.()}
