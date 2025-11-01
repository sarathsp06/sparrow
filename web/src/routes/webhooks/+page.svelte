<script lang="ts">
	import { onMount } from 'svelte';
	import { client } from '$lib/services';
	import {
		RegisteredWebhook,
		ListWebhooksRequest,
		UnregisterWebhookRequest,
		WebhookHealth
	} from '../../../../proto/webhook_pb.js';

	let webhooks: RegisteredWebhook[] = [];
	let loading = true;
	let error = '';

	const healthColor: Record<WebhookHealth, string> = {
		[WebhookHealth.HEALTH_UNKNOWN]: 'bg-gray-400',
		[WebhookHealth.HEALTH_HEALTHY]: 'bg-green-500',
		[WebhookHealth.HEALTH_DEGRADED]: 'bg-yellow-500',
		[WebhookHealth.HEALTH_UNHEALTHY]: 'bg-red-500'
	};

	async function fetchWebhooks() {
		loading = true;
		error = '';
		try {
			const req = {
				namespace: 'default'
			};
			const res = await client.listWebhooks(req);
			webhooks = res.webhooks || [];
		} catch (e: any) {
			console.error(e);
			error = `Failed to load webhooks: ${e.message}`;
		} finally {
			loading = false;
		}
	}

	onMount(fetchWebhooks);

	async function unregisterWebhook(webhookId: string) {
		try {
			const req = { webhookId };
			await client.unregisterWebhook(req);
			await fetchWebhooks(); // Refresh the list
		} catch (e: any) {
			error = `Failed to unregister webhook: ${e.message}`;
		}
	}
</script>

<!-- Enhanced Webhook Dashboard -->
<div class="min-h-screen bg-gray-50 font-display">
	<header
		class="flex items-center justify-between p-4 bg-white/80 backdrop-blur-md sticky top-0 z-10 border-b"
	>
		<h1 class="text-2xl font-bold text-gray-800 flex items-center gap-2">
			<span class="material-symbols-outlined text-primary text-3xl">hub</span>
			Webhooks
		</h1>
		<a
			href="/webhooks/register"
			class="bg-primary text-white px-4 py-2 rounded-lg font-semibold shadow-md hover:bg-primary/90 transition"
			>+ Register Webhook</a
		>
	</header>

	<main class="p-6">
		{#if loading}
			<div class="flex justify-center items-center h-40">
				<span class="material-symbols-outlined animate-spin text-4xl text-primary">autorenew</span>
			</div>
		{:else if error}
			<div
				class="bg-red-100 border border-red-300 text-red-700 rounded-lg p-4 mb-6 flex items-center gap-3 shadow-sm"
			>
				<span class="material-symbols-outlined">error</span>
				<p>{error}</p>
			</div>
		{:else if webhooks.length === 0}
			<div
				class="bg-white border rounded-lg p-8 text-center text-gray-500 shadow-sm flex flex-col items-center gap-4"
			>
				<span class="material-symbols-outlined text-5xl text-gray-300">webhook</span>
				<h3 class="text-xl font-semibold">No webhooks found</h3>
				<p>Get started by registering a new webhook.</p>
			</div>
		{:else}
			<div class="grid gap-5">
				{#each webhooks as wh}
					<div
						class="webhook-card bg-white rounded-lg shadow-sm border p-5 transition-all hover:shadow-lg hover:border-primary"
					>
						<div class="flex flex-col sm:flex-row justify-between items-start sm:items-center">
							<div class="flex-1 mb-4 sm:mb-0">
								<div class="flex items-center gap-3 mb-2">
									<div class="flex items-center gap-2">
										<span class={`w-3 h-3 rounded-full ${healthColor[wh.health]}`} />
										<span class="font-bold text-lg text-gray-800">{wh.description}</span>
									</div>
									<span
										class={`px-2 py-0.5 text-xs font-semibold rounded-full ${
											wh.active
												? 'bg-blue-100 text-blue-700'
												: 'bg-gray-200 text-gray-600'
										}`}
									>
										{wh.active ? 'Active' : 'Inactive'}
									</span>
								</div>
								<p class="text-gray-500 font-mono text-sm break-all">{wh.url}</p>
								<div class="flex flex-wrap gap-2 mt-3">
									{#each wh.events as event}
										<span class="bg-gray-100 text-gray-600 px-2 py-1 text-xs rounded-md"
											>{event}</span
										>
									{/each}
								</div>
							</div>
							<div class="flex gap-2 items-center">
								<a
									href={`/webhooks/${wh.webhookId}`}
									class="text-primary font-semibold px-4 py-2 rounded-lg hover:bg-primary/10 transition"
									>Details</a
								>
								<button
									on:click={() => unregisterWebhook(wh.webhookId)}
									class="text-red-600 font-semibold px-4 py-2 rounded-lg hover:bg-red-500/10 transition"
									>Delete</button
								>
							</div>
						</div>
					</div>
				{/each}
			</div>
		{/if}
	</main>
</div>

<style>
	.bg-primary {
		background-color: #1d4ed8;
	}
	.text-primary {
		color: #1d4ed8;
	}
	.border-primary {
		border-color: #1d4ed8;
	}
	.webhook-card:hover {
		transform: translateY(-2px);
	}
	.animate-spin {
		animation: spin 1.5s linear infinite;
	}
	@keyframes spin {
		to {
			transform: rotate(360deg);
		}
	}
</style>
