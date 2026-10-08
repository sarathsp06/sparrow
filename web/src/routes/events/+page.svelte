<script lang="ts">
	import { goto } from '$app/navigation';
	import { api, unwrap } from '$lib/services';
	import { untrack } from 'svelte';
	import { JSONEditor, Mode, type Content } from 'svelte-jsoneditor';
	import type { components } from '$lib/api-types';
	import EmptyState from '$lib/components/EmptyState.svelte';
	import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
	import FloatingAction from '$lib/components/FloatingAction.svelte';
	import EventTypeImport from '$lib/components/EventTypeImport.svelte';
	import Pagination from '$lib/components/Pagination.svelte';
	import { formatAPIError } from '$lib/utils';
	import { consumerFilter, withConsumer } from '$lib/consumer.svelte';
	import { ALERTS_GUIDE_URL, SYSTEM_CONSUMER, isSystemConsumer } from '$lib/system';

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

	// Event types are never deleted; retiring one sets active=false.
	let confirmToggle = $state(false);
	let eventToToggle = $state<EventTypeItem | null>(null);

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
				// Sparrow's own sparrow.* events are listed only when the switcher is on _sparrow.
				params: { query: { active_only: false, consumer: consumerFilter.value || undefined, limit: pageSize, offset } },
			}));
			events = res.items || [];
			totalCount = res.pagination?.total_count || 0;
		} catch (e: any) {
			error = formatAPIError(e, 'Failed to load events');
		} finally {
			loading = false;
		}
	}

	$effect(() => {
		consumerFilter.value; // refetch when the active consumer changes
		untrack(() => {
			currentPage = 1;
			fetchEvents();
		});
	});

	function handlePageChange(pageNum: number) {
		currentPage = pageNum;
		fetchEvents();
	}

	function promptToggle(event: EventTypeItem, e: Event) {
		e.stopPropagation();
		eventToToggle = event;
		confirmToggle = true;
	}

	async function executeToggle() {
		if (!eventToToggle) return;
		const ev = eventToToggle;
		try {
			unwrap(await api.PATCH('/v1/event-types/{name}', {
				params: { path: { name: ev.name } },
				body: { active: !ev.active },
			}));
			confirmToggle = false;
			eventToToggle = null;
			await fetchEvents();
		} catch (e: any) {
			error = formatAPIError(e, ev.active ? 'Failed to deactivate event' : 'Failed to reactivate event');
			confirmToggle = false;
		}
	}

	const isSystemEvent = (name: string) => name.toLowerCase().startsWith('sparrow.');

	// Export and import. System event types are never exported or imported.
	let selected = $state<Set<string>>(new Set());
	let importOpen = $state(false);
	let exporting = $state(false);
	let exportable = $derived(filteredEvents.filter((e) => !isSystemEvent(e.name)));
	let allVisibleSelected = $derived(exportable.length > 0 && exportable.every((e) => selected.has(e.name)));

	function toggleSelected(name: string) {
		const next = new Set(selected);
		if (next.has(name)) next.delete(name);
		else next.add(name);
		selected = next;
	}

	function toggleAllVisible() {
		selected = allVisibleSelected ? new Set() : new Set(exportable.map((e) => e.name));
	}

	async function exportTypes(all: boolean) {
		exporting = true;
		error = '';
		try {
			const bundle = unwrap(await api.POST('/v1/event-types:export', {
				body: all ? { all: true } : { names: [...selected].sort() },
			}));
			const { $schema: _, ...clean } = bundle as typeof bundle & { $schema?: string };
			const blob = new Blob([JSON.stringify(clean, null, 2) + '\n'], { type: 'application/json' });
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = 'event-types.json';
			a.click();
			URL.revokeObjectURL(url);
		} catch (e: any) {
			error = formatAPIError(e, 'Failed to export event types');
		} finally {
			exporting = false;
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
		<div class="flex flex-wrap items-center gap-2">
			<button class="btn btn-ghost" onclick={() => (importOpen = true)}>Import</button>
			<button class="btn btn-ghost" disabled={exporting || events.length === 0} onclick={() => exportTypes(true)}>Export all</button>
			<a href="/events/push" class="btn btn-ghost">Push Test Event</a>
			<a id="header-register-btn" href="/events/register" class="btn btn-beacon">
				<span class="text-lg leading-none">+</span>
				Register Event
			</a>
		</div>
	</div>

	<!-- Event types are tenant-wide; the only split is yours versus Sparrow's own alert events. -->
	<div class="flex gap-1 mb-4" role="group" aria-label="Which event types">
		{#each [{ value: '', label: 'Your event types' }, { value: SYSTEM_CONSUMER, label: `Sparrow alert events (${SYSTEM_CONSUMER})` }] as opt (opt.value)}
			<button
				type="button"
				onclick={() => (consumerFilter.value = opt.value)}
				aria-pressed={consumerFilter.value === opt.value || (!opt.value && !isSystemConsumer(consumerFilter.value))}
				class="px-3 py-1.5 text-xs rounded-md mono transition-colors {consumerFilter.value === opt.value || (!opt.value && !isSystemConsumer(consumerFilter.value)) ? 'bg-beacon text-[#2a1a02] font-semibold' : 'text-muted border border-line hover:text-text hover:bg-black/5'}"
			>
				{opt.label}
			</button>
		{/each}
	</div>

	{#if isSystemConsumer(consumerFilter.value)}
		<p class="panel-2 px-4 py-3 mb-4 text-sm text-muted" data-testid="system-events-note">
			These are Sparrow's own alert events. Sparrow emits them itself, under this consumer only; subscribe an alert-delivery webhook to them to send alert emails.
			<a href={ALERTS_GUIDE_URL} target="_blank" rel="noreferrer" class="link-beacon">Guide</a>
		</p>
	{/if}

	{#if !loading && !error && events.length > 0}
		<div class="flex flex-col sm:flex-row sm:items-center gap-3 mb-4">
			<input
				type="text"
				placeholder="Search by name or description…"
				aria-label="Search events by name or description"
				bind:value={searchQuery}
				class="input flex-1"
			/>
			{#if selected.size > 0}
				<button class="btn btn-beacon whitespace-nowrap" disabled={exporting} onclick={() => exportTypes(false)}>
					Export {selected.size} selected
				</button>
			{/if}
		</div>
	{/if}

	{#if loading}
		<div class="panel overflow-hidden">
			<div class="animate-pulse">
				{#each Array(5) as _}
					<div class="row-line h-14 bg-black/[0.015]"></div>
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
							<th class="th w-10">
								<input type="checkbox" aria-label="Select all event types for export" checked={allVisibleSelected} onchange={toggleAllVisible} class="accent-[color:var(--color-beacon)]" />
							</th>
							<th class="th">Name</th>
							<th class="th hidden sm:table-cell">Description</th>
							<th class="th">Version</th>
							<th class="th">Status</th>
							<th class="th"></th>
						</tr>
					</thead>
					<tbody>
						{#each filteredEvents as ev}
							<tr class="row-line row-hover transition cursor-pointer" onclick={() => goto(withConsumer(`/events/${encodeURIComponent(ev.name)}/reports`, consumerFilter.value))}>
								<td class="td" onclick={(e) => e.stopPropagation()}>
									{#if !isSystemEvent(ev.name)}
										<input type="checkbox" aria-label="Select {ev.name} for export" checked={selected.has(ev.name)} onchange={() => toggleSelected(ev.name)} class="accent-[color:var(--color-beacon)]" />
									{/if}
								</td>
								<td class="td font-medium text-text">{ev.name}</td>
								<td class="td text-muted hidden sm:table-cell">{ev.description || '—'}</td>
								<td class="td mono text-xs">
									{#if isSystemEvent(ev.name)}
										<span class="text-muted">v{ev.version}</span>
									{:else}
										<a class="link" href="/events/{encodeURIComponent(ev.name)}/update#history" onclick={(e) => e.stopPropagation()} title="Version history">v{ev.version}</a>
									{/if}
								</td>
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
									{:else if !isSystemEvent(ev.name)}
										<a href="/events/{encodeURIComponent(ev.name)}/update?infer" onclick={(e) => e.stopPropagation()} class="link-beacon text-xs mono mr-4" title="No schema yet: generate one from pushed events">Infer schema</a>
									{/if}
									{#if !isSystemEvent(ev.name)}
										<a href="/events/{encodeURIComponent(ev.name)}/update" onclick={(e) => e.stopPropagation()} class="link text-xs mono mr-4">Edit</a>
										<button onclick={(e) => promptToggle(ev, e)} class="btn {ev.active ? 'btn-danger' : 'btn-ghost'} !px-3 !py-1.5 text-xs" aria-label="{ev.active ? 'Deactivate' : 'Reactivate'} event {ev.name}">{ev.active ? 'Deactivate' : 'Reactivate'}</button>
									{/if}
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
		<div class="fixed inset-0 bg-black/40 backdrop-blur-sm" role="presentation" onclick={closeModal}></div>
		<div class="panel relative w-full max-w-2xl p-6">
			<div class="flex items-center justify-between mb-4">
				<div>
					<p class="eyebrow mb-1.5">Catalog / Schema</p>
					<h3 class="text-lg font-semibold text-text">Event Schema</h3>
				</div>
				<button onclick={closeModal} class="link text-2xl leading-none" aria-label="Close">&times;</button>
			</div>
			<div class="h-96">
				<JSONEditor bind:content mode={Mode.text} readOnly />
			</div>
		</div>
	</div>
{/if}

<ConfirmDialog
	open={confirmToggle}
	title={eventToToggle?.active ? 'Deactivate event type' : 'Reactivate event type'}
	message={eventToToggle?.active
		? `Pushes of "${eventToToggle?.name}" will be rejected until it is reactivated. Its definition, versions and past events are kept.`
		: `Pushes of "${eventToToggle?.name}" will be accepted again.`}
	confirmLabel={eventToToggle?.active ? 'Deactivate' : 'Reactivate'}
	variant={eventToToggle?.active ? 'warning' : 'info'}
	onconfirm={executeToggle}
	oncancel={() => { confirmToggle = false; eventToToggle = null; }}
/>

<EventTypeImport open={importOpen} onclose={() => (importOpen = false)} onimported={fetchEvents} />

<FloatingAction href="/events/register" label="Register Event" targetSelector="#header-register-btn" />
