<script>
	/** @type {{ href: string, label: string, targetSelector: string }} */
	let { href, label, targetSelector } = $props();

	let visible = $state(false);

	$effect(() => {
		const target = document.querySelector(targetSelector);
		if (!target) return;

		const observer = new IntersectionObserver(
			([entry]) => {
				// Show FAB when the header button scrolls out of view
				visible = !entry.isIntersecting;
			},
			{ threshold: 0 }
		);

		observer.observe(target);

		return () => observer.disconnect();
	});
</script>

{#if visible}
	<a
		{href}
		class="fixed bottom-6 right-6 z-50 inline-flex items-center gap-2 bg-gray-900 text-white pl-4 pr-5 py-3 rounded-full text-sm font-medium shadow-lg hover:bg-gray-800 transition-all duration-150 ease-out animate-fab-in"
	>
		<span class="text-lg leading-none">+</span>
		{label}
	</a>
{/if}

<style>
	@keyframes fab-in {
		from {
			opacity: 0;
			transform: translateY(16px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.animate-fab-in {
		animation: fab-in 150ms ease-out;
	}
</style>
