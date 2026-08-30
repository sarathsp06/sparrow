<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, unwrap } from '$lib/services';
	import { onMount } from 'svelte';
	import { JSONEditor, Mode, type Content } from 'svelte-jsoneditor';
	import type { components } from '$lib/api-types';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import FloatingAction from '$lib/components/FloatingAction.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { formatAPIError } from '$lib/utils';

	type EventTypeItem = components["schemas"]["EventTypeItem"];

	let events: EventTypeItem[] = $state([]);
	let loading = $state(true);
	let error = $state('');
	let content: Content = $state({ json: {} });
	let isModalOpen = $state(false);

	let searchQuery = $state('');

	let pageSize = $state(25);
	let currentPage = $state(1);
	let totalCount = $state(0);
	let totalPages = $derived(Math.max(1, Math.ceil(totalCount / pageSize)));

	let confirmDelete = $state(false);
	let eventToDelete = $state<EventTypeItem | null>(null);

	let filteredEvents = $derived.by(() => {
		if (!searchQuery.trim()) return events;
		const q = searchQuery.toLowerCase();
		return events.filter(
			(e) =>
				e.name.toLowerCase().includes(q) ||
				(e.description ?? '').toLowerCase().includes(q)
		);
	});

	async function fetchEvents() {
		loading = true;
		error = '';
		try {
			const offset = (currentPage - 1) * pageSize;
			const res = unwrap(await api.GET('/v1/event-types', {
				params: { query: { active_only: false, limit: pageSize, offset } },
			}));
			events = res.items || [];
			totalCount = res.pagination?.total_count || 0;
		} catch (e: any) {
			error = formatAPIError(e, 'Failed to load events');
		} finally {
			loading = false;
		}
	}

	onMount(fetchEvents);

	function handlePageChange(pageNum: number) {
		currentPage = pageNum;
		fetchEvents();
	}

	function promptDelete(event: EventTypeItem, e: Event) {
		e.stopPropagation();
		eventToDelete = event;
		confirmDelete = true;
	}

	async function executeDelete() {
		if (!eventToDelete) return;
		try {
			unwrap(await api.DELETE('/v1/event-types/{name}', { params: { path: { name: eventToDelete.name } } }));
			confirmDelete = false;
			eventToDelete = null;
			await fetchEvents();
		} catch (e: any) {
			error = formatAPIError(e, 'Failed to delete event');
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
		<div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
			<div>
				<h1 class="text-2xl font-bold text-gray-900">Events</h1>
				<p class="text-sm text-gray-500 mt-0.5">Manage registered event types</p>
			</div>
			<div class="flex items-center gap-2">
				<a
					href="/events/push"
					class="inline-flex items-center gap-2 bg-white text-gray-700 border border-gray-300 px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-50 transition shadow-sm"
				>
					Push Test Event
				</a>
			<a
				id="header-register-btn"
				href="/events/register"
				class="inline-flex items-center gap-2 bg-gray-900 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-800 transition shadow-sm"
			>
				<span class="text-lg leading-none">+</span>
				Register Event
			</a>
			</div>
		</div>

		{#if !loading && !error && events.length > 0}
			<div class="flex flex-col sm:flex-row gap-3 mb-4">
				<input
					type="text"
					placeholder="Search by name or description..."
					bind:value={searchQuery}
					class="flex-1 px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-gray-900"
				/>
			</div>
		{/if}

		{#if loading}
			<div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
				<div class="animate-pulse divide-y divide-gray-100">
					{#each Array(5) as _}
						<div class="p-4 flex items-center gap-4">
							<div class="h-4 bg-gray-200 rounded w-32"></div>
							<div class="h-4 bg-gray-100 rounded flex-1"></div>
						</div>
					{/each}
				</div>
			</div>
		{:else if error}
			<div class="bg-red-50 border border-red-200 rounded-lg p-4 mb-6">
				<p class="text-sm text-red-700">{error}</p>
			</div>
		{:else if events.length === 0}
			<div class="bg-white border border-gray-200 rounded-lg">
				<EmptyState icon="calendar" title="No events registered" description="Register an event type to start pushing events." />
			</div>
		{:else if filteredEvents.length === 0}
			<div class="bg-white border border-gray-200 rounded-lg">
				<EmptyState icon="search" title="No matching events" description="Try a different search term." />
			</div>
		{:else}
			<div class="bg-white rounded-lg border border-gray-200 overflow-hidden">
				<div class="overflow-x-auto">
					<table class="w-full text-sm text-left">
						<thead>
							<tr class="border-b border-gray-200 bg-gray-50/50">
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Name</th>
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Description</th>
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
								<th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider"></th>
							</tr>
						</thead>
						<tbody class="divide-y divide-gray-100">
							{#each filteredEvents as ev}
								<tr class="hover:bg-gray-50 transition cursor-pointer" onclick={() => goto(`/events/${encodeURIComponent(ev.name)}/reports`)}>
									<td class="px-4 py-3 font-medium text-gray-900">{ev.name}</td>
									<td class="px-4 py-3 text-gray-600 hidden sm:table-cell">{ev.description || '—'}</td>
									<td class="px-4 py-3">
										<span class="px-2 py-0.5 text-xs font-medium rounded-full {ev.active ? 'bg-green-100 text-green-700' : 'bg-gray-100 text-gray-500'}">
											{ev.active ? 'Active' : 'Inactive'}
										</span>
									</td>
									<td class="px-4 py-3 text-right whitespace-nowrap">
										{#if ev.event_schema && Object.keys(ev.event_schema).length > 0}
											<button onclick={(e) => viewSchema(ev.event_schema, e)} class="text-xs text-gray-500 hover:text-gray-800 underline mr-3">Schema</button>
										{/if}
										<a href="/events/{encodeURIComponent(ev.name)}/update" onclick={(e) => e.stopPropagation()} class="text-xs text-gray-500 hover:text-gray-800 underline mr-3">Edit</a>
										<button onclick={(e) => promptDelete(ev, e)} class="text-xs text-red-500 hover:text-red-700 underline">Delete</button>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>

		<Pagination
			{currentPage}
			{totalPages}
			{totalCount}
			{pageSize}
			onPageChange={handlePageChange}
		/>
		{/if}
	</main>
</div>

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
				<button onclick={closeModal} class="text-gray-400 hover:text-gray-600">&times;</button>
			</div>
			<div class="h-96">
				<JSONEditor bind:content mode={Mode.text} readOnly />
			</div>
		</div>
	</div>
{/if}

<ConfirmDialog
	open={confirmDelete}
	title="Delete Event"
	message={`This will permanently remove the event type "${eventToDelete?.name}". Existing event instances will not be affected, but no new events of this type can be pushed.`}
	confirmLabel="Delete"
	variant="danger"
	onconfirm={executeDelete}
	oncancel={() => { confirmDelete = false; eventToDelete = null; }}
/>

<FloatingAction href="/events/register" label="Register Event" targetSelector="#header-register-btn" />
