<script lang="ts">
  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import {
    subscriptionClient as client,
    webhookClient
  } from "$lib/services";
  import { onMount } from "svelte";
  import { type EventSubscription, type TemplateFunction } from "../../../../../../proto/webhook_pb";

  let webhookId: string = "";
  let webhook: any = null;
  let subscriptions: EventSubscription[] = [];
  let error: string = "";
  let loading = false;
  let showCreateModal = false;
  let editingSubscription: any = null;
  let showEditModal = false;
  let showTemplateDocs = false;
  let templateFunctions: TemplateFunction[] = [];
  let loadingTemplateFunctions = false;
  let selectedFunction: TemplateFunction | null = null;

  let newSubscription = {
    eventName: "",
    namespace: "default",
    transformEnabled: false,
    transformTemplate: "",
    method: "POST",
    timeout: 30,
    headers: {} as Record<string, string>,
  };

  let editSubscription = {
    subscriptionId: "",
    eventName: "",
    namespace: "default",
    transformEnabled: false,
    transformTemplate: "",
    method: "POST",
    timeout: 30,
    headers: {} as Record<string, string>,
  };

  // Header management for forms
  let newHeaderKey = "";
  let newHeaderValue = "";
  let editHeaderKey = "";
  let editHeaderValue = "";

  function addHeaderToNew() {
    if (newHeaderKey && newHeaderValue) {
      newSubscription.headers[newHeaderKey] = newHeaderValue;
      newHeaderKey = "";
      newHeaderValue = "";
      newSubscription = { ...newSubscription }; // Trigger reactivity
    }
  }

  function removeHeaderFromNew(key: string) {
    delete newSubscription.headers[key];
    newSubscription = { ...newSubscription };
  }

  function addHeaderToEdit() {
    if (editHeaderKey && editHeaderValue) {
      editSubscription.headers[editHeaderKey] = editHeaderValue;
      editHeaderKey = "";
      editHeaderValue = "";
      editSubscription = { ...editSubscription };
    }
  }

  function removeHeaderFromEdit(key: string) {
    delete editSubscription.headers[key];
    editSubscription = { ...editSubscription };
  }

  async function fetchWebhook() {
    try {
      const res = await webhookClient.listWebhooks({
        namespace: "default",
        webhookId
      });
      webhook = res.webhooks?.[0];
    } catch (e: any) {
      console.error('Failed to fetch webhook:', e);
    }
  }

  async function fetchSubscriptions() {
    if (!webhookId) return;
    try {
      loading = true;
      const response = await client.listSubscriptions({
        webhookId,
        namespace: "default"
      });
      subscriptions = response.subscriptions
      console.log(subscriptions);
    } catch (e: any) {
      error = `Failed to fetch subscriptions: ${e.message}`;
    } finally {
      loading = false;
    }
  }

  async function createSubscription() {
    try {
      await client.createSubscription({
        webhookId,
        eventName: newSubscription.eventName,
        namespace: newSubscription.namespace,
        transformEnabled: newSubscription.transformEnabled,
        transformTemplate: newSubscription.transformTemplate,
        method: newSubscription.method,
        timeout: newSubscription.timeout,
        headers: newSubscription.headers,
      });

      showCreateModal = false;
      await fetchSubscriptions();

      // Reset form
      newSubscription = {
        eventName: "",
        namespace: "default",
        transformEnabled: false,
        transformTemplate: "",
        method: "POST",
        timeout: 30,
        headers: {},
      };
    } catch (e: any) {
      error = `Failed to create subscription: ${e.message}`;
    }
  }

  async function updateSubscription() {
    try {
      await client.updateSubscription({
        subscriptionId: editSubscription.subscriptionId,
        namespace: editSubscription.namespace,
        headers: editSubscription.headers,
        method: editSubscription.method,
        timeout: editSubscription.timeout,
        transformEnabled: editSubscription.transformEnabled,
        transformTemplate: editSubscription.transformTemplate,
      });

      showEditModal = false;
      editingSubscription = null;
      await fetchSubscriptions();
    } catch (e: any) {
      error = `Failed to update subscription: ${e.message}`;
    }
  }

  async function deleteSubscription(subscriptionId: string) {
    if (!confirm("Are you sure you want to delete this subscription?")) return;

    try {
      await client.deleteSubscription({
        subscriptionId,
        namespace: "default"
      });
      await fetchSubscriptions();
    } catch (e: any) {
      error = `Failed to delete subscription: ${e.message}`;
    }
  }

  function openEditModal(subscription: any) {
    editingSubscription = subscription;
    editSubscription = {
      subscriptionId: subscription.subscriptionId,
      eventName: subscription.eventName,
      namespace: subscription.namespace,
      transformEnabled: subscription.transformEnabled,
      transformTemplate: subscription.transformTemplate || "",
      method: subscription.method || "POST",
      timeout: subscription.timeout || 30,
      headers: subscription.headers || {},
    };
    showEditModal = true;
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
      error = `Failed to fetch template functions: ${e.message}`;
    } finally {
      loadingTemplateFunctions = false;
    }
  }

  function parseMarkdown(description: string): { title: string; summary: string; usage: string; example: string } {
    const lines = description.split('\n');
    let title = '';
    let summary = '';
    let usage = '';
    let example = '';
    let currentSection = '';
    let inCodeBlock = false;
    
    for (const line of lines) {
      if (line.trim().startsWith('# ')) {
        title = line.replace(/^#\s*/, '').trim();
      } else if (line.trim().startsWith('## Usage')) {
        currentSection = 'usage';
      } else if (line.trim().startsWith('## Example')) {
        currentSection = 'example';
      } else if (line.trim() === '```') {
        inCodeBlock = !inCodeBlock;
        if (currentSection === 'usage') usage += '\n';
        if (currentSection === 'example') example += '\n';
      } else if (line.trim().startsWith('##')) {
        currentSection = 'other';
      } else {
        if (!currentSection && !line.trim().startsWith('#') && line.trim()) {
          summary += line + '\n';
        } else if (currentSection === 'usage') {
          usage += line + '\n';
        } else if (currentSection === 'example') {
          example += line + '\n';
        }
      }
    }
    
    return {
      title: title || 'Function',
      summary: summary.trim(),
      usage: usage.trim(),
      example: example.trim()
    };
  }

  onMount(async () => {
    webhookId = page.params.webhookId || "";
    if (webhookId) {
      await Promise.all([fetchWebhook(), fetchSubscriptions()]);
    } else {
      error = "No webhook ID provided";
    }
  });
</script>

<svelte:head>
  <title>Subscriptions - {webhookId} | HTTPQueue</title>
</svelte:head>

<div class="min-h-screen bg-gray-50 font-display">
  <main class="p-6">
    <!-- Breadcrumb -->
    <nav class="mb-6">
      <ol class="flex items-center space-x-2 text-sm text-gray-500">
        <li><a href="/" class="hover:text-blue-600">Dashboard</a></li>
        <li>›</li>
        <li><a href="/webhooks/{webhookId}" class="hover:text-blue-600">Webhook {webhookId}</a></li>
        <li>›</li>
        <li class="text-gray-800 font-medium">Subscriptions</li>
      </ol>
    </nav>

    <div class="mb-6">
      <h1 class="text-3xl font-bold text-gray-800 mb-2">Event Subscriptions</h1>
      {#if webhook}
        <p class="text-gray-600">
          Manage event subscriptions for webhook: <code class="bg-gray-100 px-2 py-1 rounded text-sm">{webhook.url}</code>
        </p>
      {/if}
    </div>

    {#if error}
      <div class="bg-red-100 border border-red-300 text-red-700 rounded-lg p-4 mb-6">
        <p>{error}</p>
      </div>
    {/if}

    {#if loading}
      <div class="bg-white rounded-lg shadow-sm border p-12 text-center">
        <p class="text-gray-500">Loading...</p>
      </div>
    {:else if webhook}
      <!-- Webhook Info -->
      <div class="bg-white rounded-lg shadow-sm border p-6 mb-6">
        <div class="flex items-center justify-between">
          <div>
            <h2 class="text-lg font-semibold text-gray-800">Webhook Details</h2>
            <p class="text-sm text-gray-600 mt-1">{webhook.url}</p>
            <div class="mt-2 flex items-center gap-4">
              <span class="text-sm font-medium text-gray-700">Status:</span>
              <span class={`text-sm px-2 py-1 rounded ${webhook.active ? 'bg-green-100 text-green-800' : 'bg-red-100 text-red-800'}`}>
                {webhook.active ? 'Active' : 'Inactive'}
              </span>
              {#if webhook.events && webhook.events.length > 0}
                <span class="text-sm font-medium text-gray-700">Events:</span>
                <div class="flex gap-1">
                  {#each webhook.events as event}
                    <span class="bg-gray-100 text-gray-600 px-2 py-1 text-xs rounded-md">{event}</span>
                  {/each}
                </div>
              {/if}
            </div>
          </div>
          <button
            onclick={() => goto(`/webhooks/${webhookId}`)}
            class="text-blue-600 hover:text-blue-800 text-sm font-medium"
          >
            ← Back to Webhook
          </button>
        </div>
      </div>

      <!-- Subscriptions -->
      <div class="bg-white rounded-lg shadow-sm border p-6">
        <div class="flex justify-between items-center mb-6">
          <h2 class="text-xl font-bold text-gray-800">Subscriptions ({subscriptions.length})</h2>
          <button
            onclick={() => (showCreateModal = true)}
            class="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 text-sm font-medium"
          >
            + Add Subscription
          </button>
        </div>

        {#if subscriptions.length === 0}
          <div class="text-center py-12 text-gray-500">
            <p class="text-lg font-medium">No subscriptions yet</p>
            <p class="text-sm mt-2">Create subscriptions to define which events this webhook should receive and how to transform them.</p>
            <button
              onclick={() => (showCreateModal = true)}
              class="mt-4 bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700 font-medium"
            >
              Create First Subscription
            </button>
          </div>
        {:else}
          <div class="space-y-4">
            {#each subscriptions as subscription}
              <div class="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition">
                <div class="flex items-start justify-between">
                  <div class="flex-1">
                    <div class="flex items-center gap-3 mb-3">
                      <h3 class="font-semibold text-gray-800 text-lg">{subscription.eventName}</h3>
                      <span class="text-xs bg-gray-100 text-gray-600 px-2 py-1 rounded font-medium">
                        {subscription.namespace}
                      </span>
                      {#if subscription.transformEnabled}
                        <span class="text-xs bg-green-100 text-green-700 px-2 py-1 rounded font-medium">
                          🎭 Template Enabled
                        </span>
                      {/if}
                    </div>
                    
                    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4 text-sm text-gray-600 mb-3">
                      <div>
                        <span class="font-medium text-gray-700">Method:</span>
                        <span class="ml-1">{subscription.method || 'POST'}</span>
                      </div>
                      <div>
                        <span class="font-medium text-gray-700">Timeout:</span>
                        <span class="ml-1">{subscription.timeout || 30}s</span>
                      </div>
                      <div>
                        <span class="font-medium text-gray-700">Created:</span>
                        <span class="ml-1">{subscription.createdAt ? new Date(Number(subscription.createdAt.seconds) * 1000).toLocaleDateString() : 'N/A'}</span>
                      </div>
                      <div>
                        <span class="font-medium text-gray-700">ID:</span>
                        <code class="ml-1 text-xs bg-gray-100 px-1 py-0.5 rounded">{subscription.subscriptionId.substring(0, 8)}...</code>
                      </div>
                    </div>
                    
                    {#if subscription.transformEnabled && subscription.transformTemplate}
                      <div class="mt-3">
                        <span class="text-sm font-medium text-gray-700">Template:</span>
                        <pre class="mt-1 bg-gray-50 p-3 rounded text-xs overflow-x-auto max-h-32 border">{formatTemplate(subscription.transformTemplate)}</pre>
                      </div>
                    {/if}
                    
                    {#if Object.keys(subscription.headers || {}).length > 0}
                      <div class="mt-3">
                        <span class="text-sm font-medium text-gray-700">Custom Headers:</span>
                        <div class="mt-1 space-y-1">
                          {#each Object.entries(subscription.headers) as [key, value]}
                            <div class="text-xs bg-blue-50 text-blue-700 px-2 py-1 rounded inline-block mr-2">
                              <span class="font-medium">{key}:</span> {value}
                            </div>
                          {/each}
                        </div>
                      </div>
                    {/if}
                  </div>
                  
                  <div class="flex gap-2 ml-4">
                    <button
                      onclick={() => openEditModal(subscription)}
                      class="bg-blue-100 text-blue-800 hover:bg-blue-200 px-3 py-1 rounded text-sm font-medium transition"
                    >
                      Edit
                    </button>
                    <button
                      onclick={() => deleteSubscription(subscription.subscriptionId)}
                      class="bg-red-100 text-red-800 hover:bg-red-200 px-3 py-1 rounded text-sm font-medium transition"
                    >
                      Delete
                    </button>
                  </div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>
    {/if}
  </main>
</div>

<!-- Create Subscription Modal -->
{#if showCreateModal}
  <div class="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
    <div class="relative top-20 mx-auto p-5 border w-11/12 max-w-2xl shadow-lg rounded-md bg-white">
      <div class="mt-3">
        <h3 class="text-lg font-medium text-gray-900 mb-4">Create New Subscription</h3>
        
        <div class="space-y-4">
          <!-- Event Name -->
          <div>
            <label for="create-event-name" class="block text-sm font-medium text-gray-700 mb-1">Event Name</label>
            <input
              id="create-event-name"
              type="text"
              bind:value={newSubscription.eventName}
              placeholder="e.g., user.created, order.completed"
              class="w-full border border-gray-300 rounded-md px-3 py-2"
            />
          </div>

          <!-- Namespace -->
          <div>
            <label for="create-namespace" class="block text-sm font-medium text-gray-700 mb-1">Namespace</label>
            <input
              id="create-namespace"
              type="text"
              bind:value={newSubscription.namespace}
              placeholder="default"
              class="w-full border border-gray-300 rounded-md px-3 py-2"
            />
          </div>

          <!-- Method and Timeout -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="create-method" class="block text-sm font-medium text-gray-700 mb-1">Method</label>
              <select id="create-method" bind:value={newSubscription.method} class="w-full border border-gray-300 rounded-md px-3 py-2">
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="PATCH">PATCH</option>
              </select>
            </div>
            <div>
              <label for="create-timeout" class="block text-sm font-medium text-gray-700 mb-1">Timeout (seconds)</label>
              <input
                id="create-timeout"
                type="number"
                bind:value={newSubscription.timeout}
                min="1"
                max="300"
                class="w-full border border-gray-300 rounded-md px-3 py-2"
              />
            </div>
          </div>

          <!-- Transform Toggle -->
          <div class="flex items-center">
            <input
              type="checkbox"
              id="transform-enabled"
              bind:checked={newSubscription.transformEnabled}
              class="h-4 w-4 text-blue-600 border-gray-300 rounded"
            />
            <label for="transform-enabled" class="ml-2 text-sm font-medium text-gray-700">
              Enable Payload Transformation
            </label>
          </div>

          <!-- Transform Template -->
          {#if newSubscription.transformEnabled}
            <div>
              <div class="flex justify-between items-center mb-1">
                <label for="create-template" class="block text-sm font-medium text-gray-700">Transform Template (Go Template)</label>
                <button
                  type="button"
                  onclick={fetchTemplateFunctions}
                  class="text-xs text-blue-600 hover:text-blue-800 font-medium"
                >
                  {showTemplateDocs ? '✕ Hide' : '📖 Show'} Template Functions
                </button>
              </div>
              
              <div class="grid gap-4" style="grid-template-columns: {showTemplateDocs ? '1fr 1fr' : '1fr'}">
                <div>
                  <textarea
                    id="create-template"
                    bind:value={newSubscription.transformTemplate}
                    rows="10"
                    placeholder={'{\n  "user_id": "{{ .Payload.id }}",\n  "email": "{{ .Payload.email | urlencode }}",\n  "timestamp": "{{ now | formatTime \\"2006-01-02T15:04:05Z07:00\\" }}"\n}'}
                    class="w-full border border-gray-300 rounded-md px-3 py-2 font-mono text-sm"
                  ></textarea>
                  <p class="text-xs text-gray-500 mt-1">Use Go template syntax to transform the webhook payload</p>
                </div>
                
                {#if showTemplateDocs}
                  <div class="border border-gray-200 rounded-md overflow-hidden">
                    <div class="bg-gray-50 p-3 border-b border-gray-200">
                      <h4 class="font-semibold text-sm text-gray-800">Available Template Functions</h4>
                      <p class="text-xs text-gray-600 mt-1">Click on a function to see details</p>
                    </div>
                    <div class="overflow-y-auto" style="max-height: 400px;">
                      {#if loadingTemplateFunctions}
                        <div class="p-4 text-center text-gray-500 text-sm">Loading...</div>
                      {:else if templateFunctions.length > 0}
                        <div class="divide-y divide-gray-100">
                          {#each templateFunctions as func}
                            {@const parsed = parseMarkdown(func.description)}
                            <button
                              type="button"
                              onclick={() => selectedFunction = selectedFunction?.name === func.name ? null : func}
                              class="w-full text-left p-3 hover:bg-blue-50 transition"
                            >
                              <div class="flex justify-between items-start">
                                <div class="flex-1">
                                  <code class="text-sm font-semibold text-blue-600">{func.name}</code>
                                  {#if selectedFunction?.name === func.name}
                                    <div class="mt-2 space-y-2">
                                      <p class="text-xs text-gray-600">{parsed.summary}</p>
                                      {#if parsed.usage}
                                        <div>
                                          <span class="text-xs font-medium text-gray-700">Usage:</span>
                                          <pre class="mt-1 text-xs bg-gray-800 text-gray-100 p-2 rounded overflow-x-auto">{parsed.usage}</pre>
                                        </div>
                                      {/if}
                                      {#if parsed.example}
                                        <div>
                                          <span class="text-xs font-medium text-gray-700">Example:</span>
                                          <pre class="mt-1 text-xs bg-gray-800 text-gray-100 p-2 rounded overflow-x-auto">{parsed.example}</pre>
                                        </div>
                                      {/if}
                                    </div>
                                  {:else}
                                    <p class="text-xs text-gray-500 mt-0.5">{parsed.summary}</p>
                                  {/if}
                                </div>
                                <span class="text-gray-400 text-xs ml-2">{selectedFunction?.name === func.name ? '▼' : '▶'}</span>
                              </div>
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
            <span class="block text-sm font-medium text-gray-700 mb-1">Custom Headers</span>
            <div class="space-y-2">
              {#each Object.entries(newSubscription.headers) as [key, value]}
                <div class="flex items-center gap-2">
                  <input type="text" value={key} disabled class="flex-1 border border-gray-300 rounded-md px-3 py-2 bg-gray-50" />
                  <input type="text" value={value} disabled class="flex-1 border border-gray-300 rounded-md px-3 py-2 bg-gray-50" />
                  <button onclick={() => removeHeaderFromNew(key)} class="text-red-600 hover:text-red-800">✕</button>
                </div>
              {/each}
              
              <div class="flex items-center gap-2">
                <input
                  type="text"
                  bind:value={newHeaderKey}
                  placeholder="Header name"
                  class="flex-1 border border-gray-300 rounded-md px-3 py-2"
                />
                <input
                  type="text"
                  bind:value={newHeaderValue}
                  placeholder="Header value"
                  class="flex-1 border border-gray-300 rounded-md px-3 py-2"
                />
                <button
                  onclick={addHeaderToNew}
                  class="bg-blue-600 text-white px-3 py-2 rounded-md hover:bg-blue-700"
                >
                  Add
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <button
            onclick={() => (showCreateModal = false)}
            class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            onclick={createSubscription}
            disabled={!newSubscription.eventName}
            class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-gray-400"
          >
            Create Subscription
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- Edit Subscription Modal -->
{#if showEditModal}
  <div class="fixed inset-0 bg-gray-600 bg-opacity-50 overflow-y-auto h-full w-full z-50">
    <div class="relative top-20 mx-auto p-5 border w-11/12 max-w-2xl shadow-lg rounded-md bg-white">
      <div class="mt-3">
        <h3 class="text-lg font-medium text-gray-900 mb-4">Edit Subscription</h3>
        
        <div class="space-y-4">
          <!-- Event Name -->
          <div>
            <label for="edit-event-name" class="block text-sm font-medium text-gray-700 mb-1">Event Name</label>
            <input
              id="edit-event-name"
              type="text"
              bind:value={editSubscription.eventName}
              placeholder="e.g., user.created, order.completed"
              class="w-full border border-gray-300 rounded-md px-3 py-2"
              disabled
            />
          </div>

          <!-- Namespace -->
          <div>
            <label for="edit-namespace" class="block text-sm font-medium text-gray-700 mb-1">Namespace</label>
            <input
              id="edit-namespace"
              type="text"
              bind:value={editSubscription.namespace}
              placeholder="default"
              class="w-full border border-gray-300 rounded-md px-3 py-2"
              disabled
            />
          </div>

          <!-- Method and Timeout -->
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label for="edit-method" class="block text-sm font-medium text-gray-700 mb-1">Method</label>
              <select id="edit-method" bind:value={editSubscription.method} class="w-full border border-gray-300 rounded-md px-3 py-2">
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="PATCH">PATCH</option>
              </select>
            </div>
            <div>
              <label for="edit-timeout" class="block text-sm font-medium text-gray-700 mb-1">Timeout (seconds)</label>
              <input
                id="edit-timeout"
                type="number"
                bind:value={editSubscription.timeout}
                min="1"
                max="300"
                class="w-full border border-gray-300 rounded-md px-3 py-2"
              />
            </div>
          </div>

          <!-- Transform Toggle -->
          <div class="flex items-center">
            <input
              type="checkbox"
              id="edit-transform-enabled"
              bind:checked={editSubscription.transformEnabled}
              class="h-4 w-4 text-blue-600 border-gray-300 rounded"
            />
            <label for="edit-transform-enabled" class="ml-2 text-sm font-medium text-gray-700">
              Enable Payload Transformation
            </label>
          </div>

          <!-- Transform Template -->
          {#if editSubscription.transformEnabled}
            <div>
              <div class="flex justify-between items-center mb-1">
                <label for="edit-template" class="block text-sm font-medium text-gray-700">Transform Template (Go Template)</label>
                <button
                  type="button"
                  onclick={fetchTemplateFunctions}
                  class="text-xs text-blue-600 hover:text-blue-800 font-medium"
                >
                  {showTemplateDocs ? '✕ Hide' : '📖 Show'} Template Functions
                </button>
              </div>
              
              <div class="grid gap-4" style="grid-template-columns: {showTemplateDocs ? '1fr 1fr' : '1fr'}">
                <div>
                  <textarea
                    id="edit-template"
                    bind:value={editSubscription.transformTemplate}
                    rows="10"
                    placeholder={'{\n  "user_id": "{{ .Payload.id }}",\n  "email": "{{ .Payload.email | urlencode }}",\n  "timestamp": "{{ now | formatTime \\"2006-01-02T15:04:05Z07:00\\" }}"\n}'}
                    class="w-full border border-gray-300 rounded-md px-3 py-2 font-mono text-sm"
                  ></textarea>
                  <p class="text-xs text-gray-500 mt-1">Use Go template syntax to transform the webhook payload</p>
                </div>
                
                {#if showTemplateDocs}
                  <div class="border border-gray-200 rounded-md overflow-hidden">
                    <div class="bg-gray-50 p-3 border-b border-gray-200">
                      <h4 class="font-semibold text-sm text-gray-800">Available Template Functions</h4>
                      <p class="text-xs text-gray-600 mt-1">Click on a function to see details</p>
                    </div>
                    <div class="overflow-y-auto" style="max-height: 400px;">
                      {#if loadingTemplateFunctions}
                        <div class="p-4 text-center text-gray-500 text-sm">Loading...</div>
                      {:else if templateFunctions.length > 0}
                        <div class="divide-y divide-gray-100">
                          {#each templateFunctions as func}
                            {@const parsed = parseMarkdown(func.description)}
                            <button
                              type="button"
                              onclick={() => selectedFunction = selectedFunction?.name === func.name ? null : func}
                              class="w-full text-left p-3 hover:bg-blue-50 transition"
                            >
                              <div class="flex justify-between items-start">
                                <div class="flex-1">
                                  <code class="text-sm font-semibold text-blue-600">{func.name}</code>
                                  {#if selectedFunction?.name === func.name}
                                    <div class="mt-2 space-y-2">
                                      <p class="text-xs text-gray-600">{parsed.summary}</p>
                                      {#if parsed.usage}
                                        <div>
                                          <span class="text-xs font-medium text-gray-700">Usage:</span>
                                          <pre class="mt-1 text-xs bg-gray-800 text-gray-100 p-2 rounded overflow-x-auto">{parsed.usage}</pre>
                                        </div>
                                      {/if}
                                      {#if parsed.example}
                                        <div>
                                          <span class="text-xs font-medium text-gray-700">Example:</span>
                                          <pre class="mt-1 text-xs bg-gray-800 text-gray-100 p-2 rounded overflow-x-auto">{parsed.example}</pre>
                                        </div>
                                      {/if}
                                    </div>
                                  {:else}
                                    <p class="text-xs text-gray-500 mt-0.5">{parsed.summary}</p>
                                  {/if}
                                </div>
                                <span class="text-gray-400 text-xs ml-2">{selectedFunction?.name === func.name ? '▼' : '▶'}</span>
                              </div>
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
            <span class="block text-sm font-medium text-gray-700 mb-1">Custom Headers</span>
            <div class="space-y-2">
              {#each Object.entries(editSubscription.headers) as [key, value]}
                <div class="flex items-center gap-2">
                  <input type="text" value={key} disabled class="flex-1 border border-gray-300 rounded-md px-3 py-2 bg-gray-50" />
                  <input type="text" value={value} disabled class="flex-1 border border-gray-300 rounded-md px-3 py-2 bg-gray-50" />
                  <button onclick={() => removeHeaderFromEdit(key)} class="text-red-600 hover:text-red-800">✕</button>
                </div>
              {/each}
              
              <div class="flex items-center gap-2">
                <input
                  type="text"
                  bind:value={editHeaderKey}
                  placeholder="Header name"
                  class="flex-1 border border-gray-300 rounded-md px-3 py-2"
                />
                <input
                  type="text"
                  bind:value={editHeaderValue}
                  placeholder="Header value"
                  class="flex-1 border border-gray-300 rounded-md px-3 py-2"
                />
                <button
                  onclick={addHeaderToEdit}
                  class="bg-blue-600 text-white px-3 py-2 rounded-md hover:bg-blue-700"
                >
                  Add
                </button>
              </div>
            </div>
          </div>
        </div>

        <div class="flex justify-end gap-3 mt-6">
          <button
            onclick={() => (showEditModal = false)}
            class="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            onclick={updateSubscription}
            disabled={!editSubscription.eventName}
            class="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:bg-gray-400"
          >
            Update Subscription
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}
