<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, unwrap } from '$lib/services';
	import { onMount } from 'svelte';
	import { JSONEditor, Mode, type Content } from 'svelte-jsoneditor';
	import 'svelte-jsoneditor/themes/jse-theme-dark.css';
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

<main class="mx-auto max-w-7xl px-4 sm:px-8 py-8 pb-24">
	<div class="flex flex-col sm:flex-row sm:items-end sm:justify-between gap-4 mb-6">
		<div>
			<p class="eyebrow mb-1.5">Catalog / Events</p>
			<h1 class="text-2xl">Events</h1>
			<p class="text-sm text-muted mt-1">Manage registered event types</p>
		</div>
		<div class="flex items-center gap-2">
			<a href="/events/push" class="btn btn-ghost">Push Test Event</a>
			<a id="header-register-btn" href="/events/register" class="btn btn-beacon">
				<span class="text-lg leading-none">+</span>
				Register Event
			</a>
		</div>
	</div>

	{#if !loading && !error && events.length > 0}
		<div class="flex flex-col sm:flex-row gap-3 mb-4">
			<input
				type="text"
				placeholder="Search by name or description…"
				bind:value={searchQuery}
				class="input flex-1"
			/>
		</div>
	{/if}

	{#if loading}
		<div class="panel overflow-hidden">
			<div class="animate-pulse">
				{#each Array(5) as _}
					<div class="row-line h-14 bg-white/[0.015]"></div>
				{/each}
			</div>
		</div>
	{:else if error}
		<div class="panel p-4 mb-6" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
			<p class="text-sm" style="color:var(--color-bad)">{error}</p>
		</div>
	{:else if events.length === 0}
		<div class="panel">
			<EmptyState icon="calendar" title="No events registered" description="Register an event type to start pushing events." />
		</div>
	{:else if filteredEvents.length === 0}
		<div class="panel">
			<EmptyState icon="search" title="No matching events" description="Try a different search term." />
		</div>
	{:else}
		<div class="panel overflow-hidden">
			<div class="overflow-x-auto">
				<table class="w-full text-left">
					<thead>
						<tr class="border-b border-line">
							<th class="th">Name</th>
							<th class="th hidden sm:table-cell">Description</th>
							<th class="th">Status</th>
							<th class="th"></th>
						</tr>
					</thead>
					<tbody>
						{#each filteredEvents as ev}
							<tr class="row-line row-hover transition cursor-pointer" onclick={() => goto(`/events/${encodeURIComponent(ev.name)}/reports`)}>
								<td class="td font-medium text-text">{ev.name}</td>
								<td class="td text-muted hidden sm:table-cell">{ev.description || '—'}</td>
								<td class="td">
									<span
										class="chip"
										style={ev.active
											? 'color:var(--color-ok);border-color:color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 12%,var(--color-panel-2))'
											: 'color:var(--color-idle);border-color:color-mix(in srgb,var(--color-idle) 35%,transparent);background:color-mix(in srgb,var(--color-idle) 12%,var(--color-panel-2))'}
									>
										<span class="w-1.5 h-1.5 rounded-full" style="background:var(--color-{ev.active ? 'ok' : 'idle'})"></span>
										{ev.active ? 'Active' : 'Inactive'}
									</span>
								</td>
								<td class="td text-right whitespace-nowrap">
									{#if ev.event_schema && Object.keys(ev.event_schema).length > 0}
										<button onclick={(e) => viewSchema(ev.event_schema, e)} class="link text-xs mono mr-4">Schema</button>
									{/if}
									<a href="/events/{encodeURIComponent(ev.name)}/update" onclick={(e) => e.stopPropagation()} class="link text-xs mono mr-4">Edit</a>
									<button onclick={(e) => promptDelete(ev, e)} class="btn btn-danger !px-3 !py-1.5 text-xs" aria-label="Delete event {ev.name}">Delete</button>
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

{#if isModalOpen}
	<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
	<div
		class="fixed inset-0 z-50 flex items-center justify-center p-4"
		role="dialog"
		aria-modal="true"
		aria-label="Event Schema"
		tabindex="-1"
		onkeydown={(e) => e.key === 'Escape' && closeModal()}
	>
		<!-- svelte-ignore a11y_click_events_have_key_events -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div class="fixed inset-0 bg-ink/70 backdrop-blur-sm" role="presentation" onclick={closeModal}></div>
		<div class="panel ticked relative w-full max-w-2xl p-6">
			<div class="flex items-center justify-between mb-4">
				<div>
					<p class="eyebrow mb-1.5">Catalog / Schema</p>
					<h3 class="text-lg font-semibold text-text">Event Schema</h3>
				</div>
				<button onclick={closeModal} class="link text-2xl leading-none" aria-label="Close">&times;</button>
			</div>
			<div class="jse-theme-dark h-96">
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
