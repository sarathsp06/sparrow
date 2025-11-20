<script lang="ts">
  import { client } from "$lib/services";
  import { onMount } from "svelte";
  import type {
    EventSubscription,
    RegisteredEvent,
  } from "../../../../proto/webhook_pb.js";

  let subscriptions: EventSubscription[] = $state([]);
  let allEvents: RegisteredEvent[] = $state([]);
  let loading = $state(true);
  let error = $state("");
  let selectedWebhookId = $state("");
  let showCreateModal = $state(false);

  // Create subscription form
  let newSubscription = $state({
    webhookId: "",
    eventName: "",
    namespace: "default",
    transformEnabled: false,
    transformTemplate: "",
    method: "POST",
    timeout: 30,
    headers: {} as Record<string, string>,
  });

  async function fetchData() {
    loading = true;
    error = "";
    try {
      const [eventsRes] = await Promise.all([
        client.listEvents({ activeOnly: true }),
      ]);
      allEvents = eventsRes.events || [];
    } catch (e: any) {
      error = `Failed to load data: ${e.message}`;
    } finally {
      loading = false;
    }
  }

  async function fetchSubscriptionsForWebhook(webhookId: string) {
    if (!webhookId) return;
    try {
      const res = await client.listSubscriptions({ webhookId });
      subscriptions = res.subscriptions || [];
    } catch (e: any) {
      error = `Failed to load subscriptions: ${e.message}`;
    }
  }

  async function createSubscription() {
    try {
      await client.createSubscription({
        webhookId: newSubscription.webhookId,
        eventName: newSubscription.eventName,
        namespace: newSubscription.namespace,
        transformEnabled: newSubscription.transformEnabled,
        transformTemplate: newSubscription.transformTemplate,
        method: newSubscription.method,
        timeout: newSubscription.timeout,
        headers: newSubscription.headers,
      });

      showCreateModal = false;
      await fetchSubscriptionsForWebhook(newSubscription.webhookId);

      // Reset form
      newSubscription = {
        webhookId: "",
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

  async function deleteSubscription(subscriptionId: string) {
    if (!confirm("Are you sure you want to delete this subscription?")) return;

    try {
      await client.deleteSubscription({ subscriptionId });
      await fetchSubscriptionsForWebhook(selectedWebhookId);
    } catch (e: any) {
      error = `Failed to delete subscription: ${e.message}`;
    }
  }

  onMount(fetchData);

  $effect(() => {
    if (selectedWebhookId) {
      fetchSubscriptionsForWebhook(selectedWebhookId);
    }
  });
</script>

<div class="min-h-screen bg-gray-50 font-display">
  <main class="p-6">
    <div class="mb-6">
      <h1 class="text-3xl font-bold text-gray-800 mb-2">Event Subscriptions</h1>
      <p class="text-gray-600">
        Manage event subscriptions and payload templates for webhooks
      </p>
    </div>

    {#if error}
      <div
        class="bg-red-100 border border-red-300 text-red-700 rounded-lg p-4 mb-6"
      >
        <p>{error}</p>
      </div>
    {/if}

    <!-- Webhook Selector -->
    <div class="bg-white rounded-lg shadow-sm border p-6 mb-6">
      <h2 class="text-lg font-semibold text-gray-800 mb-4">Select Webhook</h2>
      <div class="flex gap-4">
        <input
          type="text"
          placeholder="Enter webhook ID..."
          bind:value={selectedWebhookId}
          class="flex-1 border border-gray-300 rounded-md px-4 py-2"
        />
        <button
          onclick={() => fetchSubscriptionsForWebhook(selectedWebhookId)}
          class="bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700"
        >
          Load Subscriptions
        </button>
      </div>
    </div>

    {#if selectedWebhookId}
      <!-- Subscriptions List -->
      <div class="bg-white rounded-lg shadow-sm border p-6 mb-6">
        <div class="flex justify-between items-center mb-4">
          <h2 class="text-lg font-semibold text-gray-800">
            Subscriptions for {selectedWebhookId}
          </h2>
          <button
            onclick={() => {
              newSubscription.webhookId = selectedWebhookId;
              showCreateModal = true;
            }}
            class="bg-green-600 text-white px-4 py-2 rounded-md hover:bg-green-700"
          >
            + Add Subscription
          </button>
        </div>

        {#if subscriptions.length === 0}
          <div class="text-center py-8 text-gray-500">
            <p>No subscriptions found for this webhook.</p>
          </div>
        {:else}
          <div class="space-y-4">
            {#each subscriptions as subscription}
              <div class="border border-gray-200 rounded-lg p-4">
                <div class="flex items-start justify-between">
                  <div class="flex-1">
                    <div class="flex items-center gap-3 mb-2">
                      <h3 class="font-semibold text-gray-800">
                        {subscription.eventName}
                      </h3>
                      {#if subscription.transformEnabled}
                        <span
                          class="text-xs bg-green-100 text-green-700 px-2 py-1 rounded"
                        >
                          🎭 Template Enabled
                        </span>
                      {/if}
                    </div>

                    {#if subscription.transformEnabled && subscription.transformTemplate}
                      <div class="mt-3">
                        <span class="text-sm font-medium text-gray-700"
                          >Template:</span
                        >
                        <pre
                          class="mt-1 bg-gray-50 p-2 rounded text-xs overflow-x-auto max-h-32">{subscription.transformTemplate}</pre>
                      </div>
                    {/if}
                  </div>

                  <div class="flex gap-2">
                    <button
                      onclick={() =>
                        deleteSubscription(subscription.subscriptionId)}
                      class="text-red-600 hover:text-red-800 text-sm font-medium"
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

    <!-- Template Examples -->
    <div class="bg-white rounded-lg shadow-sm border p-6">
      <h2 class="text-lg font-semibold text-gray-800 mb-4">
        Template Examples
      </h2>
      <div class="grid md:grid-cols-2 gap-6">
        <div>
          <h3 class="font-medium text-gray-700 mb-2">Slack BlockKit</h3>
          <pre class="bg-gray-50 p-3 rounded text-xs overflow-x-auto">{`{
  "blocks": [
    {
      "type": "header",
      "text": {
        "type": "plain_text",
        "text": "🎉 {{ .Event.Event }}"
      }
    },
    {
      "type": "section",
      "fields": [
        {
          "type": "mrkdwn",
          "text": "*User:*\\n{{ .Payload.user_id }}"
        }
      ]
    }
  ]
}`}</pre>
        </div>

        <div>
          <h3 class="font-medium text-gray-700 mb-2">Custom API</h3>
          <pre class="bg-gray-50 p-3 rounded text-xs overflow-x-auto">{`{
  "event_type": "{{ .Event.Event }}",
  "timestamp": "{{ .Event.CreatedAt }}",
  "data": {{ json .Payload }},
  "metadata": {
    "event_id": "{{ .Event.ID }}",
    "namespace": "{{ .Event.Namespace }}"
  }
}`}</pre>
        </div>
      </div>
    </div>
  </main>
</div>

<!-- Create Subscription Modal -->
{#if showCreateModal}
  <div
    class="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
  >
    <div
      class="bg-white rounded-lg p-6 w-full max-w-2xl mx-4 max-h-[90vh] overflow-y-auto"
    >
      <h2 class="text-lg font-semibold text-gray-800 mb-4">
        Create New Subscription
      </h2>

      <form
        onsubmit={(e) => {
          e.preventDefault();
          createSubscription();
        }}
      >
        <div class="space-y-4">
          <div>
            <label class="block text-sm font-medium text-gray-700">Event</label>
            <select
              bind:value={newSubscription.eventName}
              class="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2"
            >
              <option value="">Select an event...</option>
              {#each allEvents as event}
                <option value={event.name}
                  >{event.name} - {event.description}</option
                >
              {/each}
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-gray-700">
              <input
                type="checkbox"
                bind:checked={newSubscription.transformEnabled}
                class="mr-2"
              />
              Enable Template Transformation
            </label>
          </div>

          {#if newSubscription.transformEnabled}
            <div>
              <label class="block text-sm font-medium text-gray-700"
                >Template</label
              >
              <textarea
                bind:value={newSubscription.transformTemplate}
                rows="8"
                class="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2 font-mono text-sm"
                placeholder="Enter Go template..."
              ></textarea>
            </div>
          {/if}

          <div class="grid grid-cols-2 gap-4">
            <div>
              <label
                for="method"
                class="block text-sm font-medium text-gray-700">Method</label
              >
              <select
                bind:value={newSubscription.method}
                class="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2"
              >
                <option value="POST">POST</option>
                <option value="PUT">PUT</option>
                <option value="PATCH">PATCH</option>
              </select>
            </div>

            <div>
              <label
                for="timeout"
                class="block text-sm font-medium text-gray-700"
                >Timeout (seconds)</label
              >
              <input
                type="number"
                bind:value={newSubscription.timeout}
                min="1"
                max="300"
                class="mt-1 block w-full border border-gray-300 rounded-md px-3 py-2"
              />
            </div>
          </div>
        </div>

        <div class="flex gap-4 mt-6">
          <button
            type="submit"
            class="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
          >
            Create Subscription
          </button>
          <button
            type="button"
            onclick={() => (showCreateModal = false)}
            class="border border-gray-300 px-4 py-2 rounded-md hover:bg-gray-50"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
