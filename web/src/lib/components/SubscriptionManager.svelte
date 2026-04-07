<script lang="ts">
  import {
    subscriptionClient as client,
    webhookClient,
    eventClient,
  } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import { onMount } from "svelte";

  import type {
    EventSubscription,
    TemplateFunction,
    RegisteredEvent,
  } from "../../../../proto/webhook_pb";
  import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
  import CopyableId from "$lib/components/CopyableId.svelte";
  import EmptyState from "$lib/components/EmptyState.svelte";

  let {
    webhookId,
    namespace,
    subscriptions = $bindable([]),
    onRefresh,
  }: {
    webhookId: string;
    namespace: string;
    subscriptions: EventSubscription[];
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
  let templateFunctions: TemplateFunction[] = $state([]);
  let loadingTemplateFunctions = $state(false);
  let selectedFunction: TemplateFunction | null = $state(null);

  // Event data
  let availableEvents: RegisteredEvent[] = $state([]);
  let selectedEventDetails: RegisteredEvent | null = $state(null);
  let previewLoading = $state(false);
  let dryRunResult = $state("");
  let dryRunError = $state("");

  // Catch-all constant
  const CATCH_ALL_EVENT = "*";

  // Form state (shared for create/edit)
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

  // Whether the catch-all toggle is on (controls eventName = '*')
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
      form.headers = {
        ...form.headers,
        [newHeaderKey.trim()]: newHeaderValue.trim(),
      };
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
      form.labelFilters = {
        ...form.labelFilters,
        [newLabelFilterKey.trim()]: newLabelFilterValue.trim(),
      };
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
      const response = await eventClient.listEvents({ activeOnly: true });
      availableEvents = response.events;
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
        const res = await eventClient.getEvent({ name: eventName });
        selectedEventDetails = res.event || null;
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
      const response = await client.testSubscriptionTemplate({
        eventName: form.eventName,
        transformTemplate: form.transformTemplate,
        namespace: form.namespace,
      });
      dryRunResult = response.transformedPayload;
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
      const response = await client.listSubscriptions({
        webhookId,
        namespace: namespace || "",
      });
      subscriptions = response.subscriptions;
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
      if (modalMode === "create") {
        await client.createSubscription({
          webhookId,
          eventName,
          namespace: form.namespace,
          transformEnabled: form.transformEnabled,
          transformTemplate: form.transformTemplate,
          method: form.method,
          timeout: form.timeout,
          headers: form.headers,
          labelFilters: form.labelFilters,
        });
      } else {
        await client.updateSubscription({
          subscriptionId: form.subscriptionId,
          namespace: form.namespace,
          headers: form.headers,
          method: form.method,
          timeout: form.timeout,
          transformEnabled: form.transformEnabled,
          transformTemplate: form.transformTemplate,
          labelFilters: form.labelFilters,
        });
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

  function openEditModal(subscription: EventSubscription) {
    catchAllEnabled = subscription.eventName === CATCH_ALL_EVENT;
    form = {
      subscriptionId: subscription.subscriptionId,
      eventName: subscription.eventName,
      namespace: subscription.namespace,
      transformEnabled: subscription.transformEnabled,
      transformTemplate: subscription.transformTemplate || "",
      method: subscription.method || "POST",
      timeout: subscription.timeout || 30,
      headers: { ...(subscription.headers || {}) },
      labelFilters: { ...(subscription.labelFilters || {}) },
    };
    handleEventChange(subscription.eventName);
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
      await client.deleteSubscription({
        subscriptionId: subscriptionToDelete,
        namespace: namespace || "",
      });
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

  async function fetchTemplateFunctions() {
    if (templateFunctions.length > 0) {
      showTemplateDocs = !showTemplateDocs;
      return;
    }
    try {
      loadingTemplateFunctions = true;
      const response = await webhookClient.getTemplateFunctions({});
      templateFunctions = response.functions;
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
  <!-- Header with Add button -->
  <div class="flex items-center justify-between mb-4">
    <h3 class="text-sm font-semibold text-gray-900 uppercase tracking-wide">
      Event Subscriptions
    </h3>
    <button
      onclick={openCreateModal}
      class="inline-flex items-center gap-1.5 bg-gray-900 text-white px-3 py-1.5 rounded-lg text-sm font-medium hover:bg-gray-800 transition shadow-sm"
    >
      <span class="text-lg leading-none">+</span>
      Add Subscription
    </button>
  </div>

  {#if error}
    <div
      class="bg-red-50 border border-red-200 rounded-lg p-3 mb-4 flex items-start justify-between"
    >
      <p class="text-sm text-red-700">{error}</p>
      <button
        onclick={() => {
          error = "";
        }}
        class="text-red-400 hover:text-red-600 ml-3 shrink-0"
        aria-label="Dismiss error"
      >
        <svg
          class="w-4 h-4"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>
  {/if}

  {#if loading}
    <!-- Loading skeleton -->
    <div class="space-y-3">
      {#each Array(3) as _}
        <div
          class="bg-white rounded-lg border border-gray-200 p-4 animate-pulse"
        >
          <div class="flex items-center gap-2 mb-2">
            <div class="h-4 bg-gray-200 rounded w-32"></div>
            <div class="h-4 bg-gray-100 rounded w-16"></div>
            <div class="h-4 bg-gray-100 rounded w-12"></div>
          </div>
          <div class="flex gap-4">
            <div class="h-3 bg-gray-100 rounded w-24"></div>
            <div class="h-3 bg-gray-100 rounded w-32"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else if subscriptions.length === 0}
    <div class="bg-white rounded-lg border border-gray-200">
      <EmptyState
        icon="link"
        title="No subscriptions yet"
        description="Create subscriptions to define which events this webhook receives and how payloads are transformed."
      >
        {#snippet action()}
          <button
            onclick={openCreateModal}
            class="inline-flex items-center gap-2 bg-gray-900 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-gray-800 transition"
          >
            Create First Subscription
          </button>
        {/snippet}
      </EmptyState>
    </div>
  {:else}
    <!-- Subscription cards -->
    <div class="space-y-3">
      {#each subscriptions as subscription}
        <div
          class="bg-white rounded-lg border border-gray-200 hover:border-gray-300 transition"
        >
          <div class="p-4">
            <div class="flex items-start justify-between gap-4">
              <div class="flex-1 min-w-0">
                <!-- Title row -->
                <div class="flex items-center gap-2 mb-2">
                  <h3 class="text-sm font-semibold text-gray-900">
                    {#if subscription.eventName === CATCH_ALL_EVENT}
                      All Events
                    {:else}
                      {subscription.eventName}
                    {/if}
                  </h3>
                  {#if subscription.eventName === CATCH_ALL_EVENT}
                    <span
                      class="px-1.5 py-0.5 text-xs font-medium bg-purple-50 text-purple-700 rounded border border-purple-200"
                    >
                      Catch-All
                    </span>
                  {/if}
                  <span
                    class="px-1.5 py-0.5 text-xs font-medium bg-gray-100 text-gray-600 rounded"
                  >
                    {subscription.namespace}
                  </span>
                  <span
                    class="px-1.5 py-0.5 text-xs font-medium bg-blue-50 text-blue-700 rounded"
                  >
                    {subscription.method || "POST"}
                  </span>
                  {#if subscription.transformEnabled}
                    <span
                      class="px-1.5 py-0.5 text-xs font-medium bg-green-50 text-green-700 rounded"
                    >
                      Template
                    </span>
                  {/if}
                </div>

                <!-- Details row -->
                <div
                  class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-gray-500"
                >
                  <span>Timeout: {subscription.timeout || 30}s</span>
                  <span
                    >Created: {subscription.createdAt
                      ? new Date(
                          Number(subscription.createdAt.seconds) * 1000,
                        ).toLocaleDateString()
                      : "N/A"}</span
                  >
                  <CopyableId id={subscription.subscriptionId} truncate={12} />
                </div>

                <!-- Template preview -->
                {#if subscription.transformEnabled && subscription.transformTemplate}
                  <details class="mt-3">
                    <summary
                      class="text-xs font-medium text-gray-600 cursor-pointer hover:text-gray-800 select-none"
                    >
                      View Template
                    </summary>
                    <pre
                      class="mt-1.5 bg-gray-50 p-3 rounded-lg text-xs overflow-x-auto max-h-32 border border-gray-200 font-mono">{formatTemplate(
                        subscription.transformTemplate,
                      )}</pre>
                  </details>
                {/if}

                <!-- Custom headers -->
                {#if Object.keys(subscription.headers || {}).length > 0}
                  <div class="mt-2 flex flex-wrap gap-1">
                    {#each Object.entries(subscription.headers) as [key, value]}
                      <span
                        class="text-xs bg-gray-50 text-gray-600 px-2 py-0.5 rounded border border-gray-200 font-mono"
                      >
                        {key}: {value}
                      </span>
                    {/each}
                  </div>
                {/if}

                <!-- Label filters -->
                {#if Object.keys(subscription.labelFilters || {}).length > 0}
                  <div class="mt-2 flex flex-wrap items-center gap-1">
                    <span class="text-xs text-gray-500 font-medium mr-1">Filters:</span>
                    {#each Object.entries(subscription.labelFilters) as [key, value]}
                      <span
                        class="text-xs bg-amber-50 text-amber-700 px-2 py-0.5 rounded border border-amber-200 font-mono"
                      >
                        {key}={value}
                      </span>
                    {/each}
                  </div>
                {:else}
                  <div class="mt-2">
                    <span class="text-xs text-gray-400 italic">No label filters — matches all events</span>
                  </div>
                {/if}
              </div>

              <!-- Actions -->
              <div class="flex items-center gap-1.5 shrink-0">
                <button
                  onclick={() => openEditModal(subscription)}
                  class="px-2.5 py-1 text-xs font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 transition"
                >
                  Edit
                </button>
                <button
                  onclick={() => promptDelete(subscription.subscriptionId)}
                  class="px-2.5 py-1 text-xs font-medium text-red-700 bg-red-50 rounded-md hover:bg-red-100 transition"
                >
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

<!-- Create/Edit Subscription Modal -->
{#if modalOpen}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div
    class="fixed inset-0 z-50 flex items-start justify-center overflow-y-auto"
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div
      class="fixed inset-0 bg-black/40 backdrop-blur-sm"
      role="presentation"
      onclick={() => {
        modalOpen = false;
        resetForm();
      }}
    ></div>
    <div
      class="relative w-full max-w-2xl mx-4 my-12 bg-white rounded-xl shadow-2xl"
    >
      <!-- Modal header -->
      <div
        class="flex items-center justify-between px-6 py-4 border-b border-gray-200"
      >
        <h3 class="text-lg font-semibold text-gray-900">
          {modalMode === "create" ? "Create Subscription" : "Edit Subscription"}
        </h3>
        <button
          onclick={() => {
            modalOpen = false;
            resetForm();
          }}
          class="p-1 text-gray-400 hover:text-gray-600 rounded transition"
          aria-label="Close modal"
        >
          <svg
            class="w-5 h-5"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>

      <!-- Modal body -->
      <div
        class="px-6 py-4 space-y-4 max-h-[calc(100vh-12rem)] overflow-y-auto"
      >
        <!-- Event Name -->
        <div>
          <label
            for="modal-event-name"
            class="block text-sm font-medium text-gray-700 mb-1"
            >Event Name</label
          >
          {#if modalMode === "create"}
            <!-- Catch-all toggle -->
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
                class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out {catchAllEnabled
                  ? 'bg-purple-500'
                  : 'bg-gray-300'}"
              >
                <span
                  class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {catchAllEnabled
                    ? 'translate-x-4'
                    : 'translate-x-0'}"
                ></span>
              </button>
              <div>
                <span class="text-sm font-medium text-gray-700"
                  >Catch-All Subscription</span
                >
                <p class="text-xs text-gray-500">
                  Receive every event in this namespace
                </p>
              </div>
            </div>

            {#if !catchAllEnabled}
              <div class="flex gap-2">
                <select
                  id="modal-event-name"
                  bind:value={form.eventName}
                  onchange={() => handleEventChange(form.eventName)}
                  class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
                >
                  <option value="">Select an event...</option>
                  {#each availableEvents as event}
                    <option value={event.name}>{event.name}</option>
                  {/each}
                </select>
                <span class="text-xs text-gray-400 self-center">or</span>
                <input
                  type="text"
                  bind:value={form.eventName}
                  oninput={() => handleEventChange(form.eventName)}
                  placeholder="Type event name"
                  class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
                />
              </div>
            {:else}
              <div
                class="bg-purple-50 border border-purple-200 rounded-lg px-3 py-2"
              >
                <p class="text-sm text-purple-700 font-medium">
                  This subscription will receive all events in the namespace.
                </p>
                <p class="text-xs text-purple-600 mt-1">
                  Use template variables <code
                    class="bg-purple-100 px-1 rounded">{"{{.EventName}}"}</code
                  >,
                  <code class="bg-purple-100 px-1 rounded"
                    >{"{{.EventID}}"}</code
                  >,
                  <code class="bg-purple-100 px-1 rounded"
                    >{"{{.Payload}}"}</code
                  > for dynamic payloads.
                </p>
              </div>
            {/if}
          {:else}
            <input
              type="text"
              value={form.eventName === CATCH_ALL_EVENT
                ? "All Events (Catch-All)"
                : form.eventName}
              disabled
              class="w-full text-sm rounded-lg border-gray-300 bg-gray-50 text-gray-500"
            />
          {/if}
        </div>

        <!-- Namespace -->
        <div>
          <label
            for="modal-namespace"
            class="block text-sm font-medium text-gray-700 mb-1"
            >Namespace</label
          >
          <input
            id="modal-namespace"
            type="text"
            bind:value={form.namespace}
            disabled={modalMode === "edit"}
            class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900 {modalMode ===
            'edit'
              ? 'bg-gray-50 text-gray-500'
              : ''}"
          />
        </div>

        <!-- Method + Timeout -->
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label
              for="modal-method"
              class="block text-sm font-medium text-gray-700 mb-1">Method</label
            >
            <select
              id="modal-method"
              bind:value={form.method}
              class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
            >
              <option value="GET">GET</option>
              <option value="POST">POST</option>
              <option value="PUT">PUT</option>
              <option value="PATCH">PATCH</option>
            </select>
          </div>
          <div>
            <label
              for="modal-timeout"
              class="block text-sm font-medium text-gray-700 mb-1"
              >Timeout (seconds)</label
            >
            <input
              id="modal-timeout"
              type="number"
              bind:value={form.timeout}
              min="1"
              max="300"
              class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
            />
          </div>
        </div>

        <!-- Transform toggle -->
        <div class="flex items-center gap-3">
          <button
            type="button"
            onclick={() => {
              form.transformEnabled = !form.transformEnabled;
            }}
            aria-label="Toggle payload transformation"
            class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out {form.transformEnabled
              ? 'bg-green-500'
              : 'bg-gray-300'}"
          >
            <span
              class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {form.transformEnabled
                ? 'translate-x-4'
                : 'translate-x-0'}"
            ></span>
          </button>
          <span class="text-sm font-medium text-gray-700"
            >Enable Payload Transformation</span
          >
        </div>

        <!-- Transform template -->
        {#if form.transformEnabled}
          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <label
                for="modal-template"
                class="block text-sm font-medium text-gray-700"
                >Transform Template</label
              >
              <button
                type="button"
                onclick={fetchTemplateFunctions}
                class="text-xs font-medium text-gray-600 hover:text-gray-900 transition"
              >
                {showTemplateDocs ? "Hide" : "Show"} Functions Reference
              </button>
            </div>

            <!-- Sample payload preview -->
            {#if selectedEventDetails}
              <div class="bg-gray-50 border border-gray-200 rounded-lg p-3">
                <div class="flex items-center justify-between mb-2">
                  <span
                    class="text-xs font-semibold text-gray-500 uppercase tracking-wider"
                    >Preview</span
                  >
                  <button
                    type="button"
                    onclick={testTemplate}
                    disabled={previewLoading || !form.transformTemplate}
                    class="px-3 py-1 text-xs font-medium text-white bg-gray-900 rounded-md hover:bg-gray-800 disabled:bg-gray-400 transition"
                  >
                    {previewLoading ? "Running..." : "Run Preview"}
                  </button>
                </div>
                <div class="grid grid-cols-2 gap-3">
                  <div>
                    <p class="text-[10px] text-gray-500 mb-1 font-medium">
                      Input (Sample)
                    </p>
                    <pre
                      class="bg-white p-2 border border-gray-200 rounded text-[10px] overflow-auto max-h-32 font-mono">{JSON.stringify(
                        selectedEventDetails.samplePayload,
                        null,
                        2,
                      )}</pre>
                  </div>
                  <div>
                    <p class="text-[10px] text-gray-500 mb-1 font-medium">
                      Output (Transformed)
                    </p>
                    <div
                      class="bg-white p-2 border border-gray-200 rounded text-[10px] overflow-auto max-h-32 min-h-[40px] font-mono"
                    >
                      {#if dryRunResult}
                        <pre>{dryRunResult}</pre>
                      {:else if dryRunError}
                        <pre class="text-red-600">{dryRunError}</pre>
                      {:else}
                        <span class="text-gray-400 italic"
                          >Click "Run Preview" to see results</span
                        >
                      {/if}
                    </div>
                  </div>
                </div>
              </div>
            {/if}

            <div
              class="grid gap-3"
              style="grid-template-columns: {showTemplateDocs
                ? '1fr 1fr'
                : '1fr'}"
            >
              <div>
                <textarea
                  id="modal-template"
                  bind:value={form.transformTemplate}
                  rows="8"
                  placeholder={'{\n  "user_id": "{{ .Payload.id }}",\n  "email": "{{ .Payload.email | urlencode }}"\n}'}
                  class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900 font-mono"
                ></textarea>
                <p class="text-xs text-gray-400 mt-1">
                  Go template syntax for payload transformation
                </p>
              </div>

              {#if showTemplateDocs}
                <div class="border border-gray-200 rounded-lg overflow-hidden">
                  <div class="bg-gray-50 px-3 py-2 border-b border-gray-200">
                    <h4
                      class="text-xs font-semibold text-gray-700 uppercase tracking-wide"
                    >
                      Template Functions
                    </h4>
                  </div>
                  <div class="overflow-y-auto max-h-72">
                    {#if loadingTemplateFunctions}
                      <div class="p-4 text-center text-sm text-gray-500">
                        Loading...
                      </div>
                    {:else}
                      <div class="divide-y divide-gray-100">
                        {#each templateFunctions as func}
                          {@const parsed = parseMarkdown(func.description)}
                          <button
                            type="button"
                            onclick={() =>
                              (selectedFunction =
                                selectedFunction?.name === func.name
                                  ? null
                                  : func)}
                            class="w-full text-left px-3 py-2 hover:bg-gray-50 transition"
                          >
                            <code class="text-xs font-semibold text-blue-600"
                              >{func.name}</code
                            >
                            {#if selectedFunction?.name === func.name}
                              <div class="mt-1.5 space-y-1.5">
                                <p class="text-xs text-gray-600">
                                  {parsed.summary}
                                </p>
                                {#if parsed.usage}
                                  <pre
                                    class="text-[10px] bg-gray-800 text-gray-100 p-2 rounded overflow-x-auto">{parsed.usage}</pre>
                                {/if}
                                {#if parsed.example}
                                  <pre
                                    class="text-[10px] bg-gray-800 text-gray-100 p-2 rounded overflow-x-auto">{parsed.example}</pre>
                                {/if}
                              </div>
                            {:else}
                              <p
                                class="text-[10px] text-gray-500 mt-0.5 truncate"
                              >
                                {parsed.summary}
                              </p>
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

        <!-- Headers -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <span class="text-sm font-medium text-gray-700">Custom Headers</span
            >
          </div>

          {#if Object.keys(form.headers).length > 0}
            <div class="space-y-1.5 mb-2">
              {#each Object.entries(form.headers) as [key, value]}
                <div class="flex items-center gap-2">
                  <span
                    class="flex-1 text-xs font-mono bg-gray-50 px-2 py-1.5 rounded border border-gray-200 truncate"
                    >{key}</span
                  >
                  <span
                    class="flex-1 text-xs font-mono bg-gray-50 px-2 py-1.5 rounded border border-gray-200 truncate"
                    >{value}</span
                  >
                  <button
                    onclick={() => removeHeader(key)}
                    class="shrink-0 p-1 text-gray-400 hover:text-red-600 rounded transition"
                    aria-label="Remove header {key}"
                  >
                    <svg
                      class="w-3.5 h-3.5"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                </div>
              {/each}
            </div>
          {/if}

          <div class="flex items-center gap-2">
            <input
              type="text"
              bind:value={newHeaderKey}
              placeholder="Header name"
              class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
            />
            <input
              type="text"
              bind:value={newHeaderValue}
              placeholder="Header value"
              class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
            />
            <button
              onclick={addHeader}
              disabled={!newHeaderKey.trim() || !newHeaderValue.trim()}
              class="shrink-0 px-3 py-1.5 text-sm font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 disabled:bg-gray-300 transition"
            >
              Add
            </button>
           </div>
        </div>

        <!-- Label Filters -->
        <div>
          <div class="flex items-center justify-between mb-2">
            <span class="text-sm font-medium text-gray-700">Label Filters</span>
          </div>

          <!-- Informational callout explaining label filter behavior -->
          <div class="bg-amber-50 border border-amber-200 rounded-lg p-3 mb-3">
            <p class="text-xs text-amber-800 font-medium mb-1.5">How label filtering works</p>
            <p class="text-xs text-amber-700 leading-relaxed mb-2">
              Only events whose labels contain <strong>all</strong> of these key-value pairs will be delivered (AND logic).
              If no filters are set, this subscription matches <strong>every</strong> event of the selected type.
            </p>
            <details class="group">
              <summary class="text-xs text-amber-600 cursor-pointer hover:text-amber-800 font-medium select-none">
                Show example
              </summary>
              <div class="mt-2 bg-white rounded border border-amber-100 p-2.5 space-y-1.5">
                <p class="text-[11px] text-gray-600">
                  <span class="font-medium">Subscription filters:</span>
                  <code class="bg-amber-50 px-1 rounded text-amber-700">region=us-east</code>
                  <code class="bg-amber-50 px-1 rounded text-amber-700">env=prod</code>
                </p>
                <div class="flex items-center gap-2 text-[11px]">
                  <span class="text-green-600 font-medium">Matches:</span>
                  <span class="text-gray-600">event with labels <code class="bg-gray-100 px-1 rounded">region=us-east, env=prod, team=payments</code></span>
                </div>
                <div class="flex items-center gap-2 text-[11px]">
                  <span class="text-red-600 font-medium">Skipped:</span>
                  <span class="text-gray-600">event with labels <code class="bg-gray-100 px-1 rounded">region=us-east, env=staging</code></span>
                </div>
              </div>
            </details>
          </div>

          <!-- Validation hints -->
          <p class="text-xs text-gray-400 mb-2">
            Max 20 filters. Keys: 1-64 chars, alphanumeric with <code class="bg-gray-100 px-0.5 rounded">._-/</code>. Values: up to 256 chars.
          </p>

          {#if Object.keys(form.labelFilters).length > 0}
            <div class="space-y-1.5 mb-2">
              {#each Object.entries(form.labelFilters) as [key, value]}
                <div class="flex items-center gap-2">
                  <span
                    class="flex-1 text-xs font-mono bg-amber-50 px-2 py-1.5 rounded border border-amber-200 truncate"
                    >{key}</span
                  >
                  <span
                    class="flex-1 text-xs font-mono bg-amber-50 px-2 py-1.5 rounded border border-amber-200 truncate"
                    >{value}</span
                  >
                  <button
                    onclick={() => removeLabelFilter(key)}
                    class="shrink-0 p-1 text-gray-400 hover:text-red-600 rounded transition"
                    aria-label="Remove label filter {key}"
                  >
                    <svg
                      class="w-3.5 h-3.5"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        stroke-width="2"
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                </div>
              {/each}
            </div>
          {/if}

          <div class="flex items-center gap-2">
            <input
              type="text"
              bind:value={newLabelFilterKey}
              placeholder="Label key"
              class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
            />
            <input
              type="text"
              bind:value={newLabelFilterValue}
              placeholder="Label value"
              class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
            />
            <button
              onclick={addLabelFilter}
              disabled={!newLabelFilterKey.trim() || !newLabelFilterValue.trim()}
              class="shrink-0 px-3 py-1.5 text-sm font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 disabled:bg-gray-300 transition"
            >
              Add
            </button>
          </div>
        </div>
      </div>

      <!-- Modal footer -->
      <div
        class="flex items-center justify-end gap-3 px-6 py-4 border-t border-gray-200"
      >
        <button
          onclick={() => {
            modalOpen = false;
            resetForm();
          }}
          class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition"
        >
          Cancel
        </button>
        <button
          onclick={saveSubscription}
          disabled={!form.eventName}
          class="px-4 py-2 text-sm font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 disabled:bg-gray-300 transition shadow-sm"
        >
          {modalMode === "create"
            ? "Create Subscription"
            : "Update Subscription"}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Delete Confirmation -->
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
