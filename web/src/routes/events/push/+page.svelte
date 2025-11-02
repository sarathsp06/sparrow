<script lang="ts">
	import { preventDefault } from 'svelte/legacy';

	import { client } from '$lib/services';
	import { onMount } from 'svelte';
	import type { PushEventRequest, RegisteredEvent } from '../../../../../proto/webhook_pb.js';

	let namespace = $state('default');
	let event = $state('');
	let payload = $state('{}');
	let loading = $state(false);
	let error = $state('');
	let successMessage = $state('');
	let availableEvents: RegisteredEvent[] = $state([]);

	async function fetchEvents() {
		try {
			const req = { activeOnly: true };
			const res = await client.listEvents(req);
			availableEvents = res.events || [];
			if (availableEvents.length > 0) {
				event = availableEvents[0].name;
			}
		} catch (e: any) {
			error = `Failed to load available events: ${e.message}`;
		}
	}

	onMount(fetchEvents);

	async function pushEvent() {
		loading = true;
		error = '';
		successMessage = '';
		try {
			const req:PushEventRequest = {
				namespace,
				event,
				payload
			};
			const res = await client.pushEvent(req);
			successMessage = `Event pushed successfully! ${res.webhooksTriggered} webhooks triggered.`;
		} catch (e: any) {
			error = `Failed to push event: ${e.message}`;
		} finally {
			loading = false;
		}
	}
</script>

<div class="min-h-screen bg-gray-50 font-display">
	<main class="p-6">
		<div class="max-w-xl mx-auto bg-white rounded-lg shadow-sm border p-6">
			<h1 class="text-2xl font-bold text-gray-800 mb-4">Push a Test Event</h1>
			<form onsubmit={preventDefault(pushEvent)} class="flex flex-col gap-4">
				<div>
					<label for="namespace" class="font-semibold text-gray-600">Namespace</label>
					<input
						type="text"
						id="namespace"
						bind:value={namespace}
						class="w-full mt-1 p-2 border rounded-md"
						required
					/>
				</div>
				<div>
					<label for="event" class="font-semibold text-gray-600">Event</label>
					<select id="event" bind:value={event} class="w-full mt-1 p-2 border rounded-md" required>
						{#each availableEvents as e}
							<option value={e.name}>{e.name}</option>
						{/each}
					</select>
				</div>
				<div>
					<label for="payload" class="font-semibold text-gray-600">Payload (JSON)</label>
					<textarea
						id="payload"
						bind:value={payload}
						rows="10"
						class="w-full mt-1 p-2 border rounded-md font-mono"
						required
					></textarea>
				</div>
				<button
					type="submit"
					class="bg-primary text-white px-4 py-2 rounded-lg font-semibold hover:bg-primary/90 transition"
					disabled={loading}
				>
					{loading ? 'Pushing...' : 'Push Event'}
				</button>
			</form>
			{#if error}
				<div class="mt-4 bg-red-100 text-red-700 p-3 rounded-md">{error}</div>
			{/if}
			{#if successMessage}
				<div class="mt-4 bg-green-100 text-green-700 p-3 rounded-md">{successMessage}</div>
			{/if}
		</div>
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
