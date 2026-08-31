<script lang="ts">
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import { onMount } from "svelte";
  import EmptyState from "./EmptyState.svelte";
  import CopyableId from "./CopyableId.svelte";
  import ConfirmDialog from "./ConfirmDialog.svelte";
  import type { components } from "$lib/api-types";

  type SubscriptionItem = components["schemas"]["SubscriptionItem"];
  type EventTypeItem = components["schemas"]["EventTypeItem"];

  let {
    webhookId,
    namespace,
    subscriptions = $bindable([]),
    onRefresh,
  }: {
    webhookId: string;
    namespace: string;
    subscriptions: SubscriptionItem[];
    onRefresh?: () => void;
  } = $props();

  let error: string = $state("");
  let loading = $state(false);

  // Modal state
  let modalOpen = $state(false);
  let modalMode = $state<"create" | "edit">("create");

  // Delete confirmation
  let confirmDeleteOpen = $state(false);
  let subscriptionToDelete = $state<string | null>(null);

  // Template docs
  let showTemplateDocs = $state(false);
  let templateFunctions: { name: string; description: string }[] = $state([]);
  let loadingTemplateFunctions = $state(false);
  let selectedFunction: { name: string; description: string } | null = $state(null);

  // Event data
  let availableEvents: EventTypeItem[] = $state([]);
  let selectedEventDetails: EventTypeItem | null = $state(null);
  let previewLoading = $state(false);
  let dryRunResult = $state("");
  let dryRunError = $state("");

  const CATCH_ALL_EVENT = "*";

  let form = $state({
    subscriptionId: "",
    eventName: "",
    namespace: "",
    transformEnabled: false,
    transformTemplate: "",
    method: "POST",
    timeout: 30,
    headers: {} as Record<string, string>,
    labelFilters: {} as Record<string, string>,
  });

  let catchAllEnabled = $state(false);

  let newHeaderKey = $state("");
  let newHeaderValue = $state("");
  let newLabelFilterKey = $state("");
  let newLabelFilterValue = $state("");

  function resetForm() {
    form = {
      subscriptionId: "",
      eventName: "",
      namespace: namespace || "",
      transformEnabled: false,
      transformTemplate: "",
      method: "POST",
      timeout: 30,
      headers: {},
      labelFilters: {},
    };
    catchAllEnabled = false;
    newHeaderKey = "";
    newHeaderValue = "";
    newLabelFilterKey = "";
    newLabelFilterValue = "";
    selectedEventDetails = null;
    dryRunResult = "";
    dryRunError = "";
    showTemplateDocs = false;
    selectedFunction = null;
  }

  function addHeader() {
    if (newHeaderKey.trim() && newHeaderValue.trim()) {
      form.headers = { ...form.headers, [newHeaderKey.trim()]: newHeaderValue.trim() };
      newHeaderKey = "";
      newHeaderValue = "";
    }
  }

  function removeHeader(key: string) {
    const { [key]: _, ...rest } = form.headers;
    form.headers = rest;
  }

  function addLabelFilter() {
    if (newLabelFilterKey.trim() && newLabelFilterValue.trim()) {
      form.labelFilters = { ...form.labelFilters, [newLabelFilterKey.trim()]: newLabelFilterValue.trim() };
      newLabelFilterKey = "";
      newLabelFilterValue = "";
    }
  }

  function removeLabelFilter(key: string) {
    const { [key]: _, ...rest } = form.labelFilters;
    form.labelFilters = rest;
  }

  async function fetchAvailableEvents() {
    try {
      const res = unwrap(await api.GET('/v1/event-types', { params: { query: { active_only: true } } }));
      availableEvents = res.items || [];
    } catch (e: any) {
      console.error("Failed to fetch events:", e);
    }
  }

  async function handleEventChange(eventName: string) {
    if (!eventName) {
      selectedEventDetails = null;
      return;
    }
    const event = availableEvents.find((e) => e.name === eventName);
    if (event) {
      selectedEventDetails = event;
    } else {
      try {
        selectedEventDetails = unwrap(await api.GET('/v1/event-types/{name}', { params: { path: { name: eventName } } }));
      } catch {
        selectedEventDetails = null;
      }
    }
    dryRunResult = "";
    dryRunError = "";
  }

  async function testTemplate() {
    if (!form.eventName || !form.transformTemplate) return;
    try {
      previewLoading = true;
      dryRunError = "";
      dryRunResult = "";
      const response = unwrap(await api.POST('/v1/subscriptions:testTemplate', {
        body: { event_name: form.eventName, template: form.transformTemplate },
      }));
      dryRunResult = response.rendered;
    } catch (e: any) {
      dryRunError = e.message;
    } finally {
      previewLoading = false;
    }
  }

  async function fetchSubscriptions() {
    if (!webhookId) return;
    try {
      loading = true;
      const response = unwrap(await api.GET('/v1/namespaces/{namespace}/subscriptions', {
        params: { path: { namespace: namespace || "default" }, query: { webhook_id: webhookId } },
      }));
      subscriptions = response.items || [];
      onRefresh?.();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to fetch subscriptions');
    } finally {
      loading = false;
    }
  }

  async function saveSubscription() {
    error = "";
    try {
      const eventName = catchAllEnabled ? CATCH_ALL_EVENT : form.eventName;
      const ns = form.namespace || namespace || "default";
      if (modalMode === "create") {
        unwrap(await api.POST('/v1/namespaces/{namespace}/subscriptions', {
          params: { path: { namespace: ns } },
          body: {
            webhook_id: webhookId,
            event_name: eventName,
            transform_enabled: form.transformEnabled,
            transform_template: form.transformTemplate,
            method: form.method,
            timeout: form.timeout,
            headers: form.headers,
            label_filters: form.labelFilters,
          },
        }));
      } else {
        unwrap(await api.PATCH('/v1/namespaces/{namespace}/subscriptions/{subscription_id}', {
          params: { path: { namespace: ns, subscription_id: form.subscriptionId } },
          body: {
            headers: form.headers,
            method: form.method,
            timeout: form.timeout,
            transform_enabled: form.transformEnabled,
            transform_template: form.transformTemplate,
            label_filters: form.labelFilters,
          },
        }));
      }
      modalOpen = false;
      resetForm();
      await fetchSubscriptions();
    } catch (e: any) {
      error = formatAPIError(e, `Failed to ${modalMode === "create" ? "create" : "update"} subscription`);
    }
  }

  function openCreateModal() {
    resetForm();
    modalMode = "create";
    modalOpen = true;
  }

  function openEditModal(subscription: SubscriptionItem) {
    catchAllEnabled = subscription.event_name === CATCH_ALL_EVENT;
    form = {
      subscriptionId: subscription.subscription_id,
      eventName: subscription.event_name,
      namespace: subscription.namespace,
      transformEnabled: subscription.transform_enabled,
      transformTemplate: subscription.transform_template || "",
      method: subscription.method || "POST",
      timeout: subscription.timeout || 30,
      headers: { ...(subscription.headers || {}) },
      labelFilters: { ...(subscription.label_filters || {}) },
    };
    handleEventChange(subscription.event_name);
    modalMode = "edit";
    modalOpen = true;
  }

  function promptDelete(subscriptionId: string) {
    subscriptionToDelete = subscriptionId;
    confirmDeleteOpen = true;
  }

  async function executeDelete() {
    if (!subscriptionToDelete) return;
    try {
      unwrap(await api.DELETE('/v1/namespaces/{namespace}/subscriptions/{subscription_id}', {
        params: { path: { namespace: namespace || "default", subscription_id: subscriptionToDelete } },
      }));
      confirmDeleteOpen = false;
      subscriptionToDelete = null;
      await fetchSubscriptions();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to delete subscription');
      confirmDeleteOpen = false;
    }
  }

  function formatTemplate(template: string) {
    try {
      return JSON.stringify(JSON.parse(template), null, 2);
    } catch {
      return template;
    }
  }

  function formatCreatedAt(createdAt: string | undefined): string {
    if (!createdAt) return "N/A";
    const d = new Date(createdAt);
    return isNaN(d.getTime()) ? "N/A" : d.toLocaleDateString();
  }

  async function fetchTemplateFunctions() {
    if (templateFunctions.length > 0) {
      showTemplateDocs = !showTemplateDocs;
      return;
    }
    try {
      loadingTemplateFunctions = true;
      const response = unwrap(await api.GET('/v1/template-functions'));
      templateFunctions = response.items || [];
      showTemplateDocs = true;
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to fetch template functions');
    } finally {
      loadingTemplateFunctions = false;
    }
  }

  function parseMarkdown(description: string): {
    title: string;
    summary: string;
    usage: string;
    example: string;
  } {
    const lines = description.split("\n");
    let title = "";
    let summary = "";
    let usage = "";
    let example = "";
    let currentSection = "";

    for (const line of lines) {
      if (line.trim().startsWith("# ")) {
        title = line.replace(/^#\s*/, "").trim();
      } else if (line.trim().startsWith("## Usage")) {
        currentSection = "usage";
      } else if (line.trim().startsWith("## Example")) {
        currentSection = "example";
      } else if (line.trim() === "```") {
        if (currentSection === "usage") usage += "\n";
        if (currentSection === "example") example += "\n";
      } else if (line.trim().startsWith("##")) {
        currentSection = "other";
      } else {
        if (!currentSection && !line.trim().startsWith("#") && line.trim()) {
          summary += line + "\n";
        } else if (currentSection === "usage") {
          usage += line + "\n";
        } else if (currentSection === "example") {
          example += line + "\n";
        }
      }
    }

    return {
      title: title || "Function",
      summary: summary.trim(),
      usage: usage.trim(),
      example: example.trim(),
    };
  }

  onMount(() => {
    fetchAvailableEvents();
  });
</script>

<div>
  <div class="flex items-center justify-between mb-4">
    <h3 class="eyebrow">Event Subscriptions</h3>
    <button
      onclick={openCreateModal}
      class="btn btn-beacon !px-3 !py-1.5"
    >
      <span class="text-lg leading-none">+</span>
      Add Subscription
    </button>
  </div>

  {#if error}
    <div class="panel p-3 mb-4 flex items-start justify-between" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
      <p class="text-sm" style="color:var(--color-bad)">{error}</p>
      <button onclick={() => { error = ""; }} class="ml-3 shrink-0 text-faint hover:text-bad transition-colors" aria-label="Dismiss error">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  {#if loading}
    <div class="space-y-3">
      {#each Array(3) as _}
        <div class="panel p-4 animate-pulse">
          <div class="flex items-center gap-2 mb-2">
            <div class="h-4 bg-white/5 rounded w-32"></div>
            <div class="h-4 bg-white/[0.03] rounded w-16"></div>
            <div class="h-4 bg-white/[0.03] rounded w-12"></div>
          </div>
          <div class="flex gap-4">
            <div class="h-3 bg-white/[0.03] rounded w-24"></div>
            <div class="h-3 bg-white/[0.03] rounded w-32"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else if subscriptions.length === 0}
    <div class="panel">
      <EmptyState icon="link" title="No subscriptions yet" description="Create subscriptions to define which events this webhook receives and how payloads are transformed.">
        {#snippet action()}
          <button onclick={openCreateModal} class="btn btn-beacon">
            Create First Subscription
          </button>
        {/snippet}
      </EmptyState>
    </div>
  {:else}
    <div class="space-y-3">
      {#each subscriptions as subscription}
        <div class="panel-2 hover:border-line-strong transition">
          <div class="p-4">
            <div class="flex items-start justify-between gap-4">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-2 flex-wrap">
                  <h3 class="text-sm font-semibold text-text">
                    {#if subscription.event_name === CATCH_ALL_EVENT}
                      All Events
                    {:else}
                      {subscription.event_name}
                    {/if}
                  </h3>
                  {#if subscription.event_name === CATCH_ALL_EVENT}
                    <span class="chip" style="color:var(--color-beacon);border-color:color-mix(in srgb,var(--color-beacon) 35%,transparent);background:color-mix(in srgb,var(--color-beacon) 12%,var(--color-panel))">
                      Catch-All
                    </span>
                  {/if}
                  <span class="chip">
                    {subscription.namespace}
                  </span>
                  <span class="chip">
                    {subscription.method || "POST"}
                  </span>
                  {#if subscription.transform_enabled}
                    <span class="chip" style="color:var(--color-ok);border-color:color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 12%,var(--color-panel))">
                      Template
                    </span>
                  {/if}
                </div>

                <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted mono tnum">
                  <span>Timeout: {subscription.timeout || 30}s</span>
                  <span>Created: {formatCreatedAt(subscription.created_at)}</span>
                  <CopyableId id={subscription.subscription_id} truncate={12} />
                </div>

                {#if subscription.transform_enabled && subscription.transform_template}
                  <details class="mt-3">
                    <summary class="text-xs font-medium text-muted cursor-pointer hover:text-text select-none">
                      View Template
                    </summary>
                    <pre class="mt-1.5 panel-2 p-3 text-xs overflow-x-auto max-h-32 mono text-text">{formatTemplate(subscription.transform_template)}</pre>
                  </details>
                {/if}

                {#if Object.keys(subscription.headers || {}).length > 0}
                  <div class="mt-2 flex flex-wrap gap-1">
                    {#each Object.entries(subscription.headers || {}) as [key, value]}
                      <span class="chip">
                        {key}: {value}
                      </span>
                    {/each}
                  </div>
                {/if}

                {#if Object.keys(subscription.label_filters || {}).length > 0}
                  <div class="mt-2 flex flex-wrap items-center gap-1">
                    <span class="text-xs text-muted font-medium mr-1">Filters:</span>
                    {#each Object.entries(subscription.label_filters || {}) as [key, value]}
                      <span class="chip" style="color:var(--color-warn);border-color:color-mix(in srgb,var(--color-warn) 35%,transparent);background:color-mix(in srgb,var(--color-warn) 12%,var(--color-panel))">
                        {key}={value}
                      </span>
                    {/each}
                  </div>
                {:else}
                  <div class="mt-2">
                    <span class="text-xs text-faint italic">No label filters — matches all events</span>
                  </div>
                {/if}
              </div>

              <div class="flex items-center gap-1.5 shrink-0">
                <button onclick={() => openEditModal(subscription)} class="btn btn-ghost !px-3 !py-1.5">
                  Edit
                </button>
                <button onclick={() => promptDelete(subscription.subscription_id)} class="btn btn-danger !px-3 !py-1.5">
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

{#if modalOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div class="fixed inset-0 bg-ink/70 backdrop-blur-sm" role="presentation" onclick={() => { modalOpen = false; resetForm(); }}></div>
    <div class="panel relative w-full max-w-2xl mx-4 my-12">
      <div class="flex items-center justify-between px-6 py-4 border-b border-line">
        <h3 class="text-lg font-semibold text-text">
          {modalMode === "create" ? "Create Subscription" : "Edit Subscription"}
        </h3>
        <button onclick={() => { modalOpen = false; resetForm(); }} class="p-1 text-faint hover:text-text rounded transition" aria-label="Close modal">
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      <div class="px-6 py-4 space-y-4 max-h-[calc(100vh-12rem)] overflow-y-auto">
        <div>
          <label for="modal-event-name" class="field-label">Event Name</label>
          {#if modalMode === "create"}
            <div class="flex items-center gap-3 mb-3">
              <button
                type="button"
                onclick={() => {
                  catchAllEnabled = !catchAllEnabled;
                  if (catchAllEnabled) {
                    form.eventName = CATCH_ALL_EVENT;
                    selectedEventDetails = null;
                  } else {
                    form.eventName = "";
                  }
                }}
                aria-label="Toggle catch-all subscription"
                class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out {catchAllEnabled ? 'bg-beacon' : 'bg-line-strong'}"
              >
                <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {catchAllEnabled ? 'translate-x-4' : 'translate-x-0'}"></span>
              </button>
              <div>
                <span class="text-sm font-medium text-text">Catch-All Subscription</span>
                <p class="text-xs text-muted">Receive every event in this namespace</p>
              </div>
            </div>

            {#if !catchAllEnabled}
              <div class="flex gap-2">
                <select id="modal-event-name" bind:value={form.eventName} onchange={() => handleEventChange(form.eventName)} class="select flex-1">
                  <option value="">Select an event…</option>
                  {#each availableEvents as event}
                    <option value={event.name}>{event.name}</option>
                  {/each}
                </select>
                <span class="text-xs text-faint self-center">or</span>
                <input type="text" bind:value={form.eventName} oninput={() => handleEventChange(form.eventName)} placeholder="Type event name" class="input flex-1" />
              </div>
            {:else}
              <div class="panel-2 px-3 py-2" style="border-color:color-mix(in srgb,var(--color-beacon) 35%,transparent)">
                <p class="text-sm font-medium" style="color:var(--color-beacon)">This subscription will receive all events in the namespace.</p>
                <p class="text-xs text-muted mt-1">
                  Use template variables <code class="chip !px-1 !py-0">{"{{.EventName}}"}</code>,
                  <code class="chip !px-1 !py-0">{"{{.EventID}}"}</code>,
                  <code class="chip !px-1 !py-0">{"{{.Payload}}"}</code> for dynamic payloads.
                </p>
              </div>
            {/if}
          {:else}
            <input type="text" value={form.eventName === CATCH_ALL_EVENT ? "All Events (Catch-All)" : form.eventName} disabled class="input opacity-60" />
          {/if}
        </div>

        <div>
          <label for="modal-namespace" class="field-label">Namespace</label>
          <input
            id="modal-namespace"
            type="text"
            bind:value={form.namespace}
            disabled={modalMode === "edit"}
            class="input {modalMode === 'edit' ? 'opacity-60' : ''}"
          />
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label for="modal-method" class="field-label">Method</label>
            <select id="modal-method" bind:value={form.method} class="select">
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="PATCH">PATCH</option>
            </select>
          </div>
          <div>
            <label for="modal-timeout" class="field-label">Timeout (seconds)</label>
            <input id="modal-timeout" type="number" bind:value={form.timeout} min="1" max="300" class="input" />
          </div>
        </div>

        <div class="flex items-center gap-3">
          <button
            type="button"
            onclick={() => { form.transformEnabled = !form.transformEnabled; }}
            aria-label="Toggle payload transformation"
            class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out {form.transformEnabled ? 'bg-ok' : 'bg-line-strong'}"
          >
            <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {form.transformEnabled ? 'translate-x-4' : 'translate-x-0'}"></span>
          </button>
          <span class="text-sm font-medium text-text">Enable Payload Transformation</span>
        </div>

        {#if form.transformEnabled}
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <label for="modal-template" class="field-label !mb-0">Transform Template</label>
              <button type="button" onclick={fetchTemplateFunctions} class="text-xs font-medium text-muted hover:text-text transition">
                {showTemplateDocs ? "Hide" : "Show"} Functions Reference
              </button>
            </div>

            {#if selectedEventDetails}
              <div class="panel-2 p-3">
                <div class="flex items-center justify-between mb-2">
                  <span class="eyebrow">Preview</span>
                  <button type="button" onclick={testTemplate} disabled={previewLoading || !form.transformTemplate} class="btn btn-beacon !px-3 !py-1">
                    {previewLoading ? "Running…" : "Run Preview"}
                  </button>
                </div>
                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <p class="text-[10px] text-muted mb-1 font-medium">Input (Sample)</p>
                    <pre class="panel p-2 rounded text-[10px] overflow-auto max-h-32 mono text-text">{JSON.stringify(selectedEventDetails.sample_payload, null, 2)}</pre>
                  </div>
                  <div>
                    <p class="text-[10px] text-muted mb-1 font-medium">Output (Transformed)</p>
                    <div class="panel p-2 rounded text-[10px] overflow-auto max-h-32 min-h-[40px] mono text-text">
                      {#if dryRunResult}
                        <pre>{dryRunResult}</pre>
                      {:else if dryRunError}
                        <pre style="color:var(--color-bad)">{dryRunError}</pre>
                      {:else}
                        <span class="text-faint italic">Click "Run Preview" to see results</span>
                      {/if}
                    </div>
                  </div>
                </div>
              </div>
            {/if}

            <div class="grid gap-3" style="grid-template-columns: {showTemplateDocs ? '1fr 1fr' : '1fr'}">
              <div>
                <textarea
                  id="modal-template"
                  bind:value={form.transformTemplate}
                  rows="8"
                  placeholder={'{\n  "user_id": "{{ .Payload.id }}",\n  "email": "{{ .Payload.email | urlencode }}"\n}'}
                  class="input"
                ></textarea>
                <p class="text-xs text-faint mt-1">Go template syntax for payload transformation</p>
              </div>

              {#if showTemplateDocs}
                <div class="panel-2 overflow-hidden">
                  <div class="px-3 py-2 border-b border-line">
                    <h4 class="eyebrow">Template Functions</h4>
                  </div>
                  <div class="overflow-y-auto max-h-72">
                    {#if loadingTemplateFunctions}
                      <div class="p-4 text-center text-sm text-muted">Loading…</div>
                    {:else}
                      <div>
                        {#each templateFunctions as func}
                          {@const parsed = parseMarkdown(func.description)}
                          <button type="button" onclick={() => (selectedFunction = selectedFunction?.name === func.name ? null : func)} class="w-full text-left px-3 py-2 row-line row-hover transition">
                            <code class="text-xs font-semibold" style="color:var(--color-beacon)">{func.name}</code>
                            {#if selectedFunction?.name === func.name}
                              <div class="mt-1.5 space-y-1.5">
                                <p class="text-xs text-muted">{parsed.summary}</p>
                                {#if parsed.usage}
                                  <pre class="text-[10px] panel p-2 rounded overflow-x-auto text-text">{parsed.usage}</pre>
                                {/if}
                                {#if parsed.example}
                                  <pre class="text-[10px] panel p-2 rounded overflow-x-auto text-text">{parsed.example}</pre>
                                {/if}
                              </div>
                            {:else}
                              <p class="text-[10px] text-faint mt-0.5 truncate">{parsed.summary}</p>
                            {/if}
                          </button>
                        {/each}
                      </div>
                    {/if}
                  </div>
                </div>
              {/if}
            </div>
          </div>
        {/if}

        <div>
          <div class="flex items-center justify-between mb-2">
            <span class="field-label !mb-0">Custom Headers</span>
          </div>

          {#if Object.keys(form.headers).length > 0}
            <div class="space-y-1.5 mb-2">
              {#each Object.entries(form.headers) as [key, value]}
                <div class="flex items-center gap-2">
                  <span class="flex-1 text-xs mono panel-2 px-2 py-1.5 rounded truncate text-text">{key}</span>
                  <span class="flex-1 text-xs mono panel-2 px-2 py-1.5 rounded truncate text-text">{value}</span>
                  <button onclick={() => removeHeader(key)} class="shrink-0 p-1 text-faint hover:text-bad rounded transition" aria-label="Remove header {key}">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              {/each}
            </div>
          {/if}

          <div class="flex items-center gap-2">
            <input type="text" bind:value={newHeaderKey} placeholder="Header name" class="input flex-1" />
            <input type="text" bind:value={newHeaderValue} placeholder="Header value" class="input flex-1" />
            <button onclick={addHeader} disabled={!newHeaderKey.trim() || !newHeaderValue.trim()} class="btn btn-ghost !px-3 !py-1.5 shrink-0">
              Add
            </button>
           </div>
        </div>

        <div>
          <div class="flex items-center justify-between mb-2">
            <span class="field-label !mb-0">Label Filters</span>
          </div>

          <div class="panel-2 p-3 mb-3" style="border-color:color-mix(in srgb,var(--color-warn) 35%,transparent)">
            <p class="text-xs font-medium mb-1.5" style="color:var(--color-warn)">How label filtering works</p>
            <p class="text-xs text-muted leading-relaxed mb-2">
              Only events whose labels contain <strong class="text-text">all</strong> of these key-value pairs will be delivered (AND logic).
              If no filters are set, this subscription matches <strong class="text-text">every</strong> event of the selected type.
            </p>
            <details class="group">
              <summary class="text-xs cursor-pointer font-medium select-none" style="color:var(--color-warn)">Show example</summary>
              <div class="mt-2 panel rounded p-2.5 space-y-1.5">
                <p class="text-[11px] text-muted">
                  <span class="font-medium text-text">Subscription filters:</span>
                  <code class="chip !px-1 !py-0" style="color:var(--color-warn)">region=us-east</code>
                  <code class="chip !px-1 !py-0" style="color:var(--color-warn)">env=prod</code>
                </p>
                <div class="flex items-center gap-2 text-[11px]">
                  <span class="font-medium" style="color:var(--color-ok)">Matches:</span>
                  <span class="text-muted">event with labels <code class="chip !px-1 !py-0">region=us-east, env=prod, team=payments</code></span>
                </div>
                <div class="flex items-center gap-2 text-[11px]">
                  <span class="font-medium" style="color:var(--color-bad)">Skipped:</span>
                  <span class="text-muted">event with labels <code class="chip !px-1 !py-0">region=us-east, env=staging</code></span>
                </div>
              </div>
            </details>
          </div>

          <p class="text-xs text-faint mb-2">
            Max 20 filters. Keys: 1-64 chars, alphanumeric with <code class="chip !px-1 !py-0">._-/</code>. Values: up to 256 chars.
          </p>

          {#if Object.keys(form.labelFilters).length > 0}
            <div class="space-y-1.5 mb-2">
              {#each Object.entries(form.labelFilters) as [key, value]}
                <div class="flex items-center gap-2">
                  <span class="flex-1 text-xs mono px-2 py-1.5 rounded truncate" style="color:var(--color-warn);border:1px solid color-mix(in srgb,var(--color-warn) 35%,transparent);background:color-mix(in srgb,var(--color-warn) 10%,var(--color-panel-2))">{key}</span>
                  <span class="flex-1 text-xs mono px-2 py-1.5 rounded truncate" style="color:var(--color-warn);border:1px solid color-mix(in srgb,var(--color-warn) 35%,transparent);background:color-mix(in srgb,var(--color-warn) 10%,var(--color-panel-2))">{value}</span>
                  <button onclick={() => removeLabelFilter(key)} class="shrink-0 p-1 text-faint hover:text-bad rounded transition" aria-label="Remove label filter {key}">
                    <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                    </svg>
                  </button>
                </div>
              {/each}
            </div>
          {/if}

          <div class="flex items-center gap-2">
            <input type="text" bind:value={newLabelFilterKey} placeholder="Label key" class="input flex-1" />
            <input type="text" bind:value={newLabelFilterValue} placeholder="Label value" class="input flex-1" />
            <button onclick={addLabelFilter} disabled={!newLabelFilterKey.trim() || !newLabelFilterValue.trim()} class="btn btn-ghost !px-3 !py-1.5 shrink-0">
              Add
            </button>
          </div>
        </div>
      </div>

      <div class="flex items-center justify-end gap-3 px-6 py-4 border-t border-line">
        <button onclick={() => { modalOpen = false; resetForm(); }} class="btn btn-ghost">
          Cancel
        </button>
        <button onclick={saveSubscription} disabled={!form.eventName} class="btn btn-beacon">
          {modalMode === "create" ? "Create Subscription" : "Update Subscription"}
        </button>
      </div>
    </div>
  </div>
{/if}

<ConfirmDialog
  open={confirmDeleteOpen}
  title="Delete Subscription"
  message="This will permanently remove this event subscription. The webhook will no longer receive events for this subscription."
  confirmLabel="Delete"
  variant="danger"
  onconfirm={executeDelete}
  oncancel={() => {
    confirmDeleteOpen = false;
    subscriptionToDelete = null;
  }}
/>
