<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/stores';
	import { client } from '$lib/services';
	import {
		RegisteredWebhook,
		WebhookDelivery,
		WebhookHealthMetrics,
		GetRegisteredWebhooksRequest,
		GetWebhookDeliveryHistoryRequest,
		GetWebhookHealthRequest,
		WebhookHealth,
		WebhookDeliveryStatus
	} from '../../../../../proto/webhook_pb.js';

	let webhook: RegisteredWebhook | undefined;
	let deliveries: WebhookDelivery[] = [];
	let healthMetrics: WebhookHealthMetrics | undefined;
	let loading = true;
	let error = '';

	const webhookId = $page.params.webhookId;

	const healthColor: Record<WebhookHealth, string> = {
		[WebhookHealth.HEALTH_UNKNOWN]: 'bg-gray-400',
		[WebhookHealth.HEALTH_HEALTHY]: 'bg-green-500',
		[WebhookHealth.HEALTH_DEGRADED]: 'bg-yellow-500',
		[WebhookHealth.HEALTH_UNHEALTHY]: 'bg-red-500'
	};

	const statusColor: Record<WebhookDeliveryStatus, string> = {
		[WebhookDeliveryStatus.DELIVERY_UNKNOWN]: 'text-gray-500',
		[WebhookDeliveryStatus.DELIVERY_PENDING]: 'text-yellow-500',
		[WebhookDeliveryStatus.DELIVERY_SENDING]: 'text-blue-500',
		[WebhookDeliveryStatus.DELIVERY_SUCCESS]: 'text-green-500',
		[WebhookDeliveryStatus.DELIVERY_FAILED]: 'text-red-500',
		[WebhookDeliveryStatus.DELIVERY_RETRYING]: 'text-yellow-600',
		[WebhookDeliveryStatus.DELIVERY_EXPIRED]: 'text-gray-600'
	};

	async function fetchData() {
		loading = true;
		error = '';
		try {
			const webhookReq = {
				webhookId,
				namespace: 'default'
			};
			const webhookRes = await client.getRegisteredWebhooks(webhookReq);
			webhook = webhookRes.webhooks[0];

			const historyReq = {
				webhookId,
				namespace: 'default',
				limit: 20
			};
			const historyRes = await client.getWebhookDeliveryHistory(historyReq);
			deliveries = historyRes.deliveries || [];

			const healthReq = {
				webhookId,
				namespace: 'default'
			};
			const healthRes = await client.getWebhookHealth(healthReq);
			healthMetrics = healthRes.metrics;
		} catch (e: any) {
			console.error(e);
			error = `Failed to load data: ${e.message}`;
		} finally {
			loading = false;
		}
	}

	onMount(fetchData);

	async function toggleWebhookStatus() {
		if (!webhook) return;
		try {
			if (webhook.active) {
				await client.pauseWebhook({ webhookId, namespace: 'default' });
			} else {
				await client.resumeWebhook({ webhookId, namespace: 'default' });
			}
			await fetchData(); // Refresh data
		} catch (e: any) {
			error = `Failed to update status: ${e.message}`;
		}
	}

	async function resendAllFailed() {
		if (!webhookId) return;
		try {
			await client.resubmitWebhook({ identifier: { case: 'webhookId', value: webhookId }, namespace: 'default' });
			await fetchData(); // Refresh data
		} catch (e: any) {
			error = `Failed to resend webhooks: ${e.message}`;
		}
	}

	function formatTimestamp(timestamp: bigint): string {
		return new Date(Number(timestamp) * 1000).toLocaleString();
	}
</script>

<div class="min-h-screen bg-gray-50 font-display">
	<header
		class="flex items-center justify-between p-4 bg-white/80 backdrop-blur-md sticky top-0 z-10 border-b"
	>
		<a href="/webhooks" class="text-primary font-semibold hover:underline flex items-center gap-2">
			<span class="material-symbols-outlined">arrow_back</span>
			Back to Webhooks
		</a>
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
		{:else if webhook}
			<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
				<div class="lg:col-span-2">
					<!-- Webhook Details -->
					<div class="bg-white rounded-lg shadow-sm border p-6 mb-6">
						<div class="flex items-center gap-3 mb-4">
							<span class={`w-4 h-4 rounded-full ${healthColor[webhook.health]}`} />
							<h1 class="text-2xl font-bold text-gray-800">{webhook.description}</h1>
						</div>
						<p class="text-gray-500 font-mono text-sm break-all mb-4">{webhook.url}</p>
						<div class="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
							<div>
								<p class="font-semibold text-gray-600">Status</p>
								<p>{webhook.active ? 'Active' : 'Inactive'}</p>
							</div>
							<div>
								<p class="font-semibold text-gray-600">Namespace</p>
								<p>{webhook.namespace}</p>
							</div>
							<div>
								<p class="font-semibold text-gray-600">Created</p>
								<p>{formatTimestamp(webhook.createdAt)}</p>
							</div>
							<div>
								<p class="font-semibold text-gray-600">Timeout</p>
								<p>{webhook.timeout}s</p>
							</div>
						</div>
						<div class="mt-4">
							<p class="font-semibold text-gray-600 mb-2">Subscribed Events</p>
							<div class="flex flex-wrap gap-2">
								{#each webhook.events as event}
									<span class="bg-gray-100 text-gray-600 px-2 py-1 text-xs rounded-md"
										>{event}</span
									>
								{/each}
							</div>
						</div>
					</div>
				</div>
				<div>
					<!-- Actions -->
					<div class="bg-white rounded-lg shadow-sm border p-6">
						<h2 class="text-xl font-bold text-gray-800 mb-4">Actions</h2>
						<div class="flex flex-col gap-3">
							<button
								on:click={toggleWebhookStatus}
								class="w-full text-left font-semibold px-4 py-2 rounded-lg transition {webhook.active
									? 'bg-yellow-100 text-yellow-800 hover:bg-yellow-200'
									: 'bg-green-100 text-green-800 hover:bg-green-200'}"
							>
								{webhook.active ? 'Pause Webhook' : 'Resume Webhook'}
							</button>
							<button
								on:click={resendAllFailed}
								class="w-full text-left font-semibold px-4 py-2 rounded-lg transition bg-blue-100 text-blue-800 hover:bg-blue-200"
							>
								Resend All Failed
							</button>
						</div>
					</div>
				</div>
			</div>

			<!-- Health Metrics -->
			{#if healthMetrics}
				<div class="bg-white rounded-lg shadow-sm border p-6 mb-6">
					<h2 class="text-xl font-bold text-gray-800 mb-4">Health Metrics</h2>
					<div class="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
						<div>
							<p class="font-semibold text-gray-600">Success Rate</p>
							<p>{(healthMetrics.successRate * 100).toFixed(2)}%</p>
						</div>
						<div>
							<p class="font-semibold text-gray-600">Total Deliveries</p>
							<p>{healthMetrics.totalDeliveries}</p>
						</div>
						<div>
							<p class="font-semibold text-gray-600">Successful</p>
							<p>{healthMetrics.successfulDeliveries}</p>
						</div>
						<div>
							<p class="font-semibold text-gray-600">Failed</p>
							<p>{healthMetrics.failedDeliveries}</p>
						</div>
						<div>
							<p class="font-semibold text-gray-600">Avg. Response</p>
							<p>{healthMetrics.avgResponseTime}ms</p>
						</div>
						<div>
							<p class="font-semibold text-gray-600">Last Success</p>
							<p>
								{healthMetrics.lastSuccessAt
									? formatTimestamp(healthMetrics.lastSuccessAt)
									: 'N/A'}
							</p>
						</div>
						<div>
							<p class="font-semibold text-gray-600">Last Failure</p>
							<p>
								{healthMetrics.lastFailureAt
									? formatTimestamp(healthMetrics.lastFailureAt)
									: 'N/A'}
							</p>
						</div>
					</div>
				</div>
			{/if}

			<!-- Delivery History -->
			<div class="bg-white rounded-lg shadow-sm border p-6">
				<h2 class="text-xl font-bold text-gray-800 mb-4">Delivery History</h2>
				{#if deliveries.length === 0}
					<p class="text-gray-500">No deliveries found for this webhook.</p>
				{:else}
					<div class="overflow-x-auto">
						<table class="w-full text-sm text-left">
							<thead class="text-xs text-gray-700 uppercase bg-gray-50">
								<tr>
									<th class="px-4 py-3">Delivery ID</th>
									<th class="px-4 py-3">Event ID</th>
									<th class="px-4 py-3">Status</th>
									<th class="px-4 py-3">Attempts</th>
									<th class="px-4 py-3">Last Attempt</th>
								</tr>
							</thead>
							<tbody>
								{#each deliveries as delivery}
									<tr class="border-b hover:bg-gray-50">
										<td class="px-4 py-3 font-mono text-xs">{delivery.deliveryId}</td>
										<td class="px-4 py-3 font-mono text-xs">{delivery.eventId}</td>
										<td class="px-4 py-3">
											<span class={`font-semibold ${statusColor[delivery.status]}`}>
												{WebhookDeliveryStatus[delivery.status].replace('DELIVERY_', '')}
											</span>
										</td>
										<td class="px-4 py-3">{delivery.attemptCount}</td>
										<td class="px-4 py-3">{formatTimestamp(delivery.lastAttemptedAt)}</td>
									</tr>
								{/each}
							</tbody>
						</table>
					</div>
				{/if}
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
</style>
