<script lang="ts">
	import { client } from '$lib/services';
	import { onMount } from 'svelte';
	import { JSONEditor, type Content } from 'svelte-jsoneditor';


	import type {
	  RegisteredEvent
	} from '../../../../proto/webhook_pb.js';

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
	<header
		class="flex items-center  justify-between p-4 bg-white/80 backdrop-blur-md sticky top-0 z-10 border-b border-gray-200"
	>
		<h1 class="text-2xl font-bold text-gray-800 flex items-center gap-2">
			<a title="home" class="text-primary text-3xl" href="/"><span class="inline-block bg-primary rounded-full p-2 shadow-lg"><svg class="w-8 h-8 text-white" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path></svg></span></a>
			<span class="text-primary">Events</span>
		</h1>
		<div>
			<a
				href="/events/push"
				class="bg-blue-500 text-white px-4 py-2 rounded-lg font-semibold shadow-md hover:bg-blue-600 transition mr-4"
				>Push Test Event</a
			>
			<a
				href="/events/register"
				class="bg-primary text-white px-4 py-2 rounded-lg font-semibold shadow-md hover:bg-primary/90 transition"
				>+ Register Event</a
			>
		</div>
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
		{:else if events.length === 0}
			<div
				class="bg-white border rounded-lg p-8 text-center text-gray-500 shadow-sm flex flex-col items-center gap-4"
			>
				<span class="material-symbols-outlined text-5xl text-gray-300">event_busy</span>
				<h3 class="text-xl font-semibold">No events found</h3>
				<p>Get started by registering a new event.</p>
			</div>
		{:else}
			<div class="grid gap-5">
				{#each events as event}
					<div
						class="event-card bg-white rounded-lg shadow-sm border p-5 transition-all hover:shadow-lg hover:border-primary"
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
										onclick={() => viewSchema(event.schema)}
										class="text-gray-600 font-semibold px-4 py-2 rounded-lg hover:bg-gray-100 transition"
										>View Schema</button
									>
								{/if}
								<a
									href={`/events/${event.eventId}/update`}
									class="text-primary font-semibold px-4 py-2 rounded-lg hover:bg-primary/10 transition"
									>Update</a
								>
								<button
									onclick={() => deleteEvent(event.name)}
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

{#if isModalOpen}
	<div
		class="fixed inset-0 bg-black/30 flex items-center justify-center z-50"
	>
		<div class="bg-white rounded-lg shadow-xl p-6 w-full max-w-2xl">
			<h3 class="text-lg font-bold mb-4">Event Schema</h3>
			<JSONEditor bind:content={content} />
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
	.bg-primary {
		background-color: #1d4ed8;
	}
	.text-primary {
		color: #1d4ed8;
	}
	.border-primary {
		border-color: #1d4ed8;
	}
	.event-card:hover {
		transform: translateY(-2px);
	}
</style>
