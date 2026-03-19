<script lang="ts">
	import { goto } from '$app/navigation';
	import { eventClient as client } from '$lib/services';
	import { onMount } from 'svelte';
	import { JSONEditor, Mode, type Content } from 'svelte-jsoneditor';
	import type { RegisteredEvent } from '../../../../proto/webhook_pb.js';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { activeNamespace } from '$lib/stores/namespace.svelte';

	let events: RegisteredEvent[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let content: Content = $state({ json: {} });
	let isModalOpen = $state(false);

	// Search
	let searchQuery = $state('');

	// Pagination state
	let pageSize = $state(25);
	let currentPage = $state(1);
	let totalCount = $state(0);
	let totalPages = $derived(Math.max(1, Math.ceil(totalCount / pageSize)));

	// Delete confirmation
	let confirmDelete = $state(false);
	let eventToDelete = $state<RegisteredEvent | null>(null);

	let filteredEvents = $derived.by(() => {
		if (!searchQuery.trim()) return events;
		const q = searchQuery.toLowerCase();
		return events.filter(
			(e) =>
				e.name.toLowerCase().includes(q) ||
				e.description.toLowerCase().includes(q)
		);
	});

	async function fetchEvents() {
		loading = true;
		error = '';
		try {
			const offset = (currentPage - 1) * pageSize;
			const req = { activeOnly: false, pagination: { limit: pageSize, offset } };
			const res = await client.listEvents(req);
			events = res.events || [];
			totalCount = res.pagination?.totalCount || 0;
		} catch (e: any) {
			error = `Failed to load events: ${e.message}`;
		} finally {
			loading = false;
		}
	}

	onMount(fetchEvents);

	function handlePageChange(pageNum: number) {
		currentPage = pageNum;
		fetchEvents();
	}

	function promptDelete(event: RegisteredEvent, e: Event) {
		e.stopPropagation();
		eventToDelete = event;
		confirmDelete = true;
	}

	async function executeDelete() {
		if (!eventToDelete) return;
		try {
			await client.deleteEvent({ name: eventToDelete.name });
			confirmDelete = false;
			eventToDelete = null;
			await fetchEvents();
		} catch (e: any) {
			error = `Failed to delete event: ${e.message}`;
			confirmDelete = false;
		}
	}

	function viewSchema(schema: any, e: Event) {
		e.stopPropagation();
		content = { json: schema };
		isModalOpen = true;
	}

	function closeModal() {
		isModalOpen = false;
		content = { json: {} };
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape' && isModalOpen) {
			closeModal();
		}
	}
</script>

<svelte:head>
	<title>Events | Sparrow</title>
</svelte:head>

<svelte:window onkeydown={handleKeydown} />

<div class="min-h-screen bg-gray-50">
	<main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 pb-24">
		<!-- Page header -->
		<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
			<div>
				<h1 class="text-2xl font-bold text-gray-900">Events</h1>
				<p class="text-sm text-gray-500 mt-0.5">
					{activeNamespace() ? `Namespace: ${activeNamespace()}` : 'Manage registered event types'}
				</p>
			</div>
			<div class="flex items-center gap-2">
				<a
					href="/events/push"
					class="inline-flex items-center gap-2 bg-white text-gray-700 border border-gray-300 px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-50 transition shadow-sm"
				>
					Push Test Event
				</a>
				<a
					href="/events/register"
					class="inline-flex items-center gap-2 bg-gray-900 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-800 transition shadow-sm"
				>
					<span class="text-lg leading-none">+</span>
					Register Event
				</a>
			</div>
		</div>

		<!-- Search toolbar -->
		{#if !loading && !error && events.length > 0}
			<div class="flex flex-col sm:flex-row gap-3 mb-4">
				<div class="relative flex-1 max-w-md">
					<input
						type="text"
						placeholder="Search by name or description..."
						bind:value={searchQuery}
						class="w-full pl-9 pr-4 py-2 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
					/>
					<svg class="absolute left-3 top-2.5 w-4 h-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
					</svg>
				</div>
				<div class="text-sm text-gray-500 flex items-center">
					{totalCount} event{totalCount !== 1 ? 's' : ''} registered
				</div>
			</div>
		{/if}

		<!-- Content -->
		{#if loading}
			<!-- Loading skeleton -->
			<div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
				<div class="overflow-x-auto">
					<table class="w-full text-sm text-left">
						<thead>
							<tr class="border-b border-gray-200 bg-gray-50/50">
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Event</th>
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Description</th>
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider text-right">Actions</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							{#each Array(5) as _}
								<tr class="animate-pulse">
									<td class="px-4 py-3">
										<div class="space-y-2">
											<div class="h-4 bg-gray-200 rounded w-40"></div>
											<div class="h-3 bg-gray-100 rounded w-24 sm:hidden"></div>
										</div>
									</td>
									<td class="px-4 py-3 hidden sm:table-cell"><div class="h-4 bg-gray-100 rounded w-56"></div></td>
									<td class="px-4 py-3"><div class="h-5 bg-gray-200 rounded-full w-16"></div></td>
									<td class="px-4 py-3 text-right">
										<div class="flex gap-2 justify-end">
											<div class="h-7 bg-gray-200 rounded w-16"></div>
											<div class="h-7 bg-gray-200 rounded w-16"></div>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{:else if error}
			<div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
				<div class="flex items-start gap-3">
					<svg class="w-5 h-5 text-red-500 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
						<path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
					</svg>
					<div>
						<p class="text-sm font-medium text-red-800">{error}</p>
						<button onclick={() => { error = ''; fetchEvents(); }} class="text-sm text-red-600 hover:text-red-800 underline mt-1">Retry</button>
					</div>
				</div>
			</div>
		{:else if events.length === 0}
			<div class="bg-white border border-gray-200 rounded-lg">
				<EmptyState
					icon="event_busy"
					title="No events registered"
					description="Get started by registering a new event type to define your webhook payloads."
				>
					{#snippet action()}
						<a href="/events/register" class="inline-flex items-center gap-2 bg-gray-900 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-800 transition">
							Register Event
						</a>
					{/snippet}
				</EmptyState>
			</div>
		{:else if filteredEvents.length === 0}
			<div class="bg-white border border-gray-200 rounded-lg">
				<EmptyState
					icon="filter_alt"
					title="No matching events"
					description="Try adjusting your search criteria."
				>
					{#snippet action()}
						<button
							onclick={() => { searchQuery = ''; }}
							class="text-sm text-gray-600 hover:text-gray-900 underline"
						>
							Clear search
						</button>
					{/snippet}
				</EmptyState>
			</div>
		{:else}
			<!-- Events table -->
			<div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
				<div class="overflow-x-auto">
					<table class="w-full text-sm text-left">
						<thead>
							<tr class="border-b border-gray-200 bg-gray-50/50">
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Event</th>
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Description</th>
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider text-right">Actions</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							{#each filteredEvents as event}
								<tr
									class="hover:bg-gray-50 cursor-pointer transition"
									onclick={() => goto(`/events/${encodeURIComponent(event.name)}/reports`)}
								>
									<td class="px-4 py-3">
										<div class="flex flex-col gap-0.5">
											<span class="font-medium text-gray-900">{event.name}</span>
											<!-- Show description inline on mobile -->
											{#if event.description}
												<span class="text-xs text-gray-500 sm:hidden mt-0.5">{event.description}</span>
											{/if}
										</div>
									</td>
									<td class="px-4 py-3 hidden sm:table-cell">
										<span class="text-gray-600 text-sm">{event.description || '—'}</span>
									</td>
									<td class="px-4 py-3">
										<span
											class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium {event.active ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-600'}"
										>
											{event.active ? 'Active' : 'Inactive'}
										</span>
									</td>
									<td class="px-4 py-3 text-right">
										<div class="flex items-center gap-1 justify-end">
											{#if event.schema}
												<button
													onclick={(e) => viewSchema(event.schema, e)}
													class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium text-gray-700 bg-gray-50 hover:bg-gray-100 transition"
												>
													Schema
												</button>
											{/if}
											<a
												href={`/events/${encodeURIComponent(event.name)}/update`}
												onclick={(e) => e.stopPropagation()}
												class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium text-blue-700 bg-blue-50 hover:bg-blue-100 transition"
											>
												Update
											</a>
											<button
												onclick={(e) => promptDelete(event, e)}
												class="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium text-red-700 bg-red-50 hover:bg-red-100 transition"
											>
												Delete
											</button>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<!-- Pagination -->
				<div class="border-t border-gray-200 px-4">
					<Pagination
						{currentPage}
						{totalPages}
						{totalCount}
						{pageSize}
						onPageChange={handlePageChange}
						itemLabel="events"
					/>
				</div>
			</div>
		{/if}
	</main>
</div>

<!-- Schema Modal -->
{#if isModalOpen}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center"
		role="dialog"
		aria-modal="true"
		aria-label="Event Schema"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && closeModal()}
	>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="fixed inset-0 bg-black/40 backdrop-blur-sm" role="presentation" onclick={closeModal}></div>
		<div class="relative bg-white rounded-xl shadow-2xl w-full max-w-2xl mx-4 p-6">
			<div class="flex items-center justify-between mb-4">
				<h3 class="text-lg font-semibold text-gray-900">Event Schema</h3>
				<button
					onclick={closeModal}
					class="text-gray-400 hover:text-gray-600 transition"
					aria-label="Close schema modal"
				>
					<svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
					</svg>
				</button>
			</div>
			<JSONEditor bind:content={content} mode={Mode.text} mainMenuBar={false} readOnly={true} />
			<div class="flex justify-end mt-4">
				<button
					type="button"
					onclick={closeModal}
					class="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition"
				>
					Close
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- Confirm Delete Dialog -->
<ConfirmDialog
	open={confirmDelete}
	title="Delete Event"
	message={`This will permanently remove the event type "${eventToDelete?.name}". Existing event instances will not be affected, but no new events of this type can be pushed.`}
	confirmLabel="Delete"
	variant="danger"
	onconfirm={executeDelete}
	oncancel={() => { confirmDelete = false; eventToDelete = null; }}
/>
