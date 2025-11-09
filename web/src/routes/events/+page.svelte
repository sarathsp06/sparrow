<script lang="ts">
	import favicon from '$lib/assets/favicon.svg';
	import { client } from '$lib/services';
	import { onMount } from 'svelte';
	import { JSONEditor, Mode, type Content } from 'svelte-jsoneditor';
	import type { RegisteredEvent } from '../../../../proto/webhook_pb.js';

	let events: RegisteredEvent[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let content: Content = $state({json: {} });
	let isModalOpen = $state(false);
	$inspect(events,content)

	async function fetchEvents() {
		loading = true;
		error = '';
		try {
			const req = { activeOnly: false };
			const res = await client.listEvents(req);
			events = res.events || [];
		} catch (e: any) {
			error = `Failed to load events: ${e.message}`;
		} finally {
			loading = false;
		}
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && isModalOpen) {
			closeModal();
		}
	};

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
		fetchEvents();
	});

	async function deleteEvent(name: string) {
		try {
			const req = { name };
			await client.deleteEvent(req);
			await fetchEvents(); // Refresh the list
		} catch (e: any) {
			error = `Failed to delete event: ${e.message}`;
		}
	}

	function viewSchema(schema: string) {
		content = { json: JSON.parse(schema) };
		isModalOpen = true;
	}

	function closeModal() {
		isModalOpen = false;
		content = { json: {} };
	}
</script>

<div class="min-h-screen bg-gray-50 font-display">
	
	<main class="p-6">
		{#if loading}
			<div class="flex justify-center items-center h-40">
				<span class="material-symbols-outlined animate-spin text-4xl text-primary">
					<img src={favicon} alt="favicon" class="inline-block w-8 h-8" />
				</span>
			</div>
		{:else if error}
			<div
				class="bg-red-100 border border-red-300 text-red-700 rounded-lg p-4 mb-6 flex items-center gap-3 shadow-sm"
			>
				<span class="material-symbols-outlined">error</span>
				<p>{error}</p>
			</div>
		{:else if events.length === 0}
			<div
				class="bg-white border rounded-lg p-8 text-center text-gray-500 shadow-sm flex flex-col items-center gap-4"
			>
				<span class="material-symbols-outlined text-5xl text-gray-300">event_busy</span>
				<h3 class="text-xl font-semibold">No events found</h3>
				<p>Get started by registering a new event.</p>
			</div>
		{:else}
			<div class="w-full grid md:grid-cols-1 gap-4">
				{#each events as event}
					<div
						class="w-full bg-white rounded-lg shadow-sm border p-5 transition-all hover:shadow-lg hover:border-primary cursor-pointer"
						onclick={() => window.location.href = `/events/${encodeURIComponent(event.eventId)}/reports`}
						onkeydown={(e) => {
							if (e.key === 'Enter' || e.key === ' ') {
								window.location.href = `/events/${encodeURIComponent(event.eventId)}/reports`;
							}
						}}
						role="button"
						tabindex="0"
						aria-label={`View event instances for ${event.name}`}
					>
						<div class="flex flex-col sm:flex-row justify-between items-start sm:items-center">
							<div class="flex-1 mb-4 sm:mb-0">
								<div class="flex items-center gap-3 mb-2">
									<span class="font-bold text-lg text-gray-800">{event.name}</span>
									<span
										class={`px-2 py-0.5 text-xs font-semibold rounded-full ${
											event.active
												? 'bg-blue-100 text-blue-700'
												: 'bg-gray-200 text-gray-600'
										}`}
									>
										{event.active ? 'Active' : 'Inactive'}
									</span>
								</div>
								<p class="text-gray-600">{event.description}</p>
							</div>
							<div class="flex gap-2 items-center">
								{#if event.schema}
									<button
										onclick={(e) => {
											e.stopPropagation();
											viewSchema(event.schema);
										}}
										class="text-gray-600 font-semibold px-4 py-2 rounded-lg hover:bg-red-500/10 transition"
										>View Schema</button
									>
								{/if}
								<a
									href={`/events/${event.eventId}/update`}
									onclick={(e) => e.stopPropagation()}
									class="text-primary font-semibold px-4 py-2 rounded-lg hover:bg-red-500/10 transition"
									>Update</a
								>
								<button
									onclick={(e) => {
										e.stopPropagation();
										deleteEvent(event.name);
									}}
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

	<footer class="fixed bottom-0  w-full bg-white/90 backdrop-blur-md border-t border-gray-200 shadow-sm p-4 gap-4 flex justify-end">
			<a
				href="/events/push"
				class="button"
				>Push Test Event</a
			>
			<a
				href="/events/register"
				class="button"
				>+ Register Event</a
			>
	</footer>
</div>

{#if isModalOpen}
	<div
		class="fixed inset-0 bg-black/30 flex items-center justify-center z-50"
	>
		<div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-2xl">
			<h3 class="text-lg font-bold mb-4">Event Schema</h3>
			<JSONEditor bind:content={content} mode={Mode.text} mainMenuBar={false} readOnly={true}/>
			<button
				type="button"
				onclick={closeModal}
				class="block mt-4 bg-primary text-white px-4 p-2 rounded font-semibold hover:bg-primary/90 transition"
				>Close</button
			>
		</div>
	</div>
{/if}

<style>
	.event-card:hover {
		transform: translateY(-2px);
	}
</style>
