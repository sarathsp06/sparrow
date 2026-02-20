<script lang="ts">
    import favicon from '$lib/assets/favicon.svg';

  import { goto } from "$app/navigation";
  import { page } from "$app/state";
  import {
    webhookClient,
    deliveryClient,
    healthClient,
    subscriptionClient
  } from "$lib/services";
  import { onMount } from "svelte";
  import { namespaceState } from "$lib/namespace.svelte";
  import type {
    EventSubscription,
    RegisteredWebhook,
    WebhookDelivery,
    WebhookHealthMetrics,
  } from "../../../../../proto/webhook_pb.js";
  import type { Timestamp } from "@bufbuild/protobuf/wkt";

  import {
    WebhookDeliveryStatus,
    WebhookHealth,
  } from "../../../../../proto/webhook_pb.js";

  let webhook: RegisteredWebhook | undefined = $state();
  let deliveries: WebhookDelivery[] = $state([]);
  let healthMetrics: WebhookHealthMetrics | undefined = $state();
  let subscriptions: EventSubscription[] = $state([]);
  let loading = $state(true);
  let error = $state("");
  let expandedDeliveries: Set<string> = $state(new Set());
  let deliveryDetails: Map<string, any> = $state(new Map());
  let activeTab = $state("deliveries");
  let editingUrl = $state(false);
  let editedUrl = $state("");
  let savingUrl = $state(false);

  // Pagination state
  let limit = $state(50);
  let offset = $state(0);
  let totalCount = $state(0);

  const webhookId = page.params.webhookId;

  const healthColor: Record<WebhookHealth, string> = {
    [WebhookHealth.HEALTH_UNSPECIFIED]: "bg-gray-400",
    [WebhookHealth.HEALTH_HEALTHY]: "bg-green-500",
    [WebhookHealth.HEALTH_DEGRADED]: "bg-yellow-500",
    [WebhookHealth.HEALTH_UNHEALTHY]: "bg-red-500",
  };

  const statusColor: Record<WebhookDeliveryStatus, string> = {
    [WebhookDeliveryStatus.DELIVERY_UNSPECIFIED]: "text-gray-500",
    [WebhookDeliveryStatus.DELIVERY_PENDING]: "text-yellow-500",
    [WebhookDeliveryStatus.DELIVERY_SENDING]: "text-blue-500",
    [WebhookDeliveryStatus.DELIVERY_SUCCESS]: "text-green-500",
    [WebhookDeliveryStatus.DELIVERY_FAILED]: "text-red-500",
    [WebhookDeliveryStatus.DELIVERY_RETRYING]: "text-yellow-600",
    [WebhookDeliveryStatus.DELIVERY_EXPIRED]: "text-gray-600",
  };


  function formatPayload(payload: string): string {
    try {
      const obj = JSON.parse(payload);
      return JSON.stringify(obj, null, 2);
    } catch {
      return payload
    }
  }


  async function fetchData() {
    if (!webhookId) return;
    try {
      const [webhookRes, deliveriesRes, healthRes, subscriptionsRes] = await Promise.all([
        webhookClient.listWebhooks({ namespace: namespaceState.current, webhookId }),
        deliveryClient.listDeliveries({
          webhookId,
          namespace: namespaceState.current,
          pagination: { limit, offset },
        }),
        healthClient.getWebhookHealth({ webhookId, namespace: namespaceState.current }),
        subscriptionClient.listSubscriptions({ webhookId, namespace: namespaceState.current }),
      ]);

      webhook = webhookRes.webhooks?.[0];
      deliveries = deliveriesRes.deliveries || [];
      totalCount = deliveriesRes.pagination?.totalCount || 0;
      healthMetrics = healthRes.metrics;
      subscriptions = subscriptionsRes.subscriptions || [];
    } catch (e: any) {
      error = `Failed to load data: ${e.message}`;
    } finally {
      loading = false;
    }
  }

  onMount(fetchData);

  async function toggleWebhookStatus() {
    if (!webhook) return;
    try {
      if (webhook.active) {
        await webhookClient.pauseWebhook({ webhookId, namespace: namespaceState.current });
      } else {
        await webhookClient.resumeWebhook({ webhookId, namespace: namespaceState.current });
      }
      await fetchData(); // Refresh data
    } catch (e: any) {
      error = `Failed to update status: ${e.message}`;
    }
  }

  async function resendThisDelivery(identifier: string) {
    if (!webhookId) return;
    try {
      await deliveryClient.retryDelivery({
        deliveryId: identifier,
        namespace: namespaceState.current,
      });
      await fetchData(); // Refresh data
    } catch (e: any) {
      error = `Failed to resend webhooks: ${e.message}`;
    }
  }

  function startEditUrl() {
    if (!webhook) return;
    editedUrl = webhook.url;
    editingUrl = true;
    error = "";
  }

  function cancelEditUrl() {
    editingUrl = false;
    editedUrl = "";
  }

  async function saveWebhookUrl() {
    if (!webhook) return;

    const trimmedUrl = editedUrl.trim();
    if (!trimmedUrl) {
      error = "URL is required";
      return;
    }

    try {
      new URL(trimmedUrl);
    } catch {
      error = "Please enter a valid URL";
      return;
    }

    savingUrl = true;
    error = "";

    try {
      await webhookClient.updateWebhookConfig({
        webhookId,
        namespace: namespaceState.current,
        updates: {
          url: trimmedUrl,
          active: webhook.active,
        },
      });

      editingUrl = false;
      await fetchData();
    } catch (e: any) {
      error = `Failed to update webhook URL: ${e.message}`;
    } finally {
      savingUrl = false;
    }
  }

  function formatTimestamp(timestamp: Timestamp | undefined | null): string {
    if (!timestamp) return "N/A";
    return new Date(Number(timestamp.seconds) * 1000).toLocaleString();
  }

  async function toggleDeliveryExpansion(deliveryId: string) {
    if (expandedDeliveries.has(deliveryId)) {
      expandedDeliveries.delete(deliveryId);
      expandedDeliveries = new Set(expandedDeliveries);
    } else {
      expandedDeliveries.add(deliveryId);
      expandedDeliveries = new Set(expandedDeliveries);
      
      // Fetch detailed delivery information if not already loaded
      if (!deliveryDetails.has(deliveryId)) {
        try {
          const detailRes = await deliveryClient.getDeliveryStatus({
            deliveryId,
            namespace: namespaceState.current
          });
          deliveryDetails.set(deliveryId, detailRes.delivery);
          deliveryDetails = new Map(deliveryDetails);
        } catch (e: any) {
          console.error('Failed to load delivery details:', e);
          // Still show expanded view with available data
        }
      }
    }
  }

  function nextPage() {
    if (offset + limit < totalCount) {
      offset += limit;
      fetchData();
    }
  }

  function prevPage() {
    if (offset >= limit) {
      offset -= limit;
      fetchData();
    }
  }
</script>

<div class="min-h-screen bg-gray-50 font-display">
  <main class="p-6">
    {#if loading}
      <div class="flex justify-center items-center h-40">
        <span
          class="material-symbols-outlined animate-spin text-4xl text-primary"
          >
          <img src={favicon} alt="favicon" class="inline-block w-8 h-8" />
          </span
        >
      </div>
    {:else if error}
      <div
        class="bg-red-100 border border-red-300 text-red-700 rounded-lg p-4 mb-6 flex items-center gap-3 shadow-sm"
      >
        <span class="material-symbols-outlined">error</span>
        <p>{error}</p>
      </div>
    {:else if webhook}
      <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div class="lg:col-span-2">
          <!-- Webhook Details -->
          <div class="bg-white rounded-lg shadow-sm border p-6 mb-6">
            <div class="flex items-center gap-3 mb-4">
              <span
                class={`w-4 h-4 rounded-full ${healthColor[webhook.health]}`}
              >
              </span>
              <h1 class="text-2xl font-bold text-gray-800">
                {webhook.description || 'Webhook Details'}
              </h1>
            </div>
            <p class="text-gray-500 font-mono text-sm break-all mb-4">
              {webhook.url}
            </p>
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
              <div>
                <p class="font-semibold text-gray-600">Status</p>
                <p>{webhook.active ? "Active" : "Inactive"}</p>
              </div>
              <div>
                <p class="font-semibold text-gray-600">Namespace</p>
                <p>{webhook.namespace}</p>
              </div>
              <div>
                <p class="font-semibold text-gray-600">Created</p>
                <p>{formatTimestamp(webhook.createdAt)}</p>
              </div>
              <div>
                <p class="font-semibold text-gray-600">Timeout</p>
                <p>{webhook.timeout}s</p>
              </div>
            </div>
            <div class="mt-4">
              <p class="font-semibold text-gray-600 mb-2">Subscribed Events</p>
              <div class="flex flex-wrap gap-2">
                {#each webhook.events as event}
                  <span
                    class="bg-gray-100 text-gray-600 px-2 py-1 text-xs rounded-md"
                    >{event}</span
                  >
                {/each}
              </div>
            </div>
          </div>
        </div>
        <div>
          <!-- Actions -->
          <div class="bg-white rounded-lg shadow-sm border p-6">
            <h2 class="text-xl font-bold text-gray-800 mb-4">Actions</h2>
            <div class="flex flex-col gap-3">
              {#if editingUrl}
                <div class="space-y-2">
                  <label for="webhook-url" class="text-sm font-semibold text-gray-700">Webhook URL</label>
                  <input
                    id="webhook-url"
                    type="url"
                    bind:value={editedUrl}
                    class="w-full px-3 py-2 border border-gray-300 rounded-lg text-sm"
                    placeholder="https://example.com/webhook"
                  />
                  <div class="flex gap-2">
                    <button
                      onclick={saveWebhookUrl}
                      disabled={savingUrl}
                      class="px-3 py-2 rounded-lg text-sm font-semibold bg-blue-100 text-blue-800 hover:bg-blue-200 disabled:opacity-60"
                    >
                      {savingUrl ? 'Saving...' : 'Save URL'}
                    </button>
                    <button
                      onclick={cancelEditUrl}
                      disabled={savingUrl}
                      class="px-3 py-2 rounded-lg text-sm font-semibold bg-gray-100 text-gray-800 hover:bg-gray-200 disabled:opacity-60"
                    >
                      Cancel
                    </button>
                  </div>
                </div>
              {:else}
                <button
                  onclick={startEditUrl}
                  class="w-full text-left font-semibold px-4 py-2 rounded-lg transition bg-blue-100 text-blue-800 hover:bg-blue-200"
                >
                  Edit Webhook URL
                </button>
              {/if}
              <button
                onclick={toggleWebhookStatus}
                class="w-full text-left font-semibold px-4 py-2 rounded-lg transition {webhook.active
                  ? 'bg-yellow-100 text-yellow-800 hover:bg-yellow-200'
                  : 'bg-green-100 text-green-800 hover:bg-green-200'}"
              >
                {webhook.active ? "Pause Webhook" : "Resume Webhook"}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Health Metrics -->
      {#if healthMetrics}
        <div class="bg-white rounded-lg shadow-sm border p-6 mb-6">
          <h2 class="text-xl font-bold text-gray-800 mb-4">Health Metrics</h2>
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-4 text-sm">
            <div>
              <p class="font-semibold text-gray-600">Success Rate</p>
              <p>{(healthMetrics.successRate * 100).toFixed(2)}%</p>
            </div>
            <div>
              <p class="font-semibold text-gray-600">Total Deliveries</p>
              <p>{healthMetrics.totalDeliveries}</p>
            </div>
            <div>
              <p class="font-semibold text-gray-600">Successful</p>
              <p>{healthMetrics.successfulDeliveries}</p>
            </div>
            <div>
              <p class="font-semibold text-gray-600">Failed</p>
              <p>{healthMetrics.failedDeliveries}</p>
            </div>
            <div>
              <p class="font-semibold text-gray-600">Avg. Response</p>
              <p>{healthMetrics.avgResponseTime}ms</p>
            </div>
            <div>
              <p class="font-semibold text-gray-600">Last Success</p>
              <p>
                {healthMetrics.lastSuccessAt
                  ? formatTimestamp(healthMetrics.lastSuccessAt)
                  : "N/A"}
              </p>
            </div>
            <div>
              <p class="font-semibold text-gray-600">Last Failure</p>
              <p>
                {healthMetrics.lastFailureAt
                  ? formatTimestamp(healthMetrics.lastFailureAt)
                  : "N/A"}
              </p>
            </div>
          </div>
        </div>
      {/if}

      <!-- Tabs Navigation -->
      <div class="bg-white rounded-lg shadow-sm border mb-6">
        <div class="border-b border-gray-200">
          <nav class="-mb-px flex">
            <button
              class="py-2 px-4 border-b-2 font-medium text-sm"
              class:border-blue-500={activeTab === 'deliveries'}
              class:text-blue-600={activeTab === 'deliveries'}
              class:border-transparent={activeTab !== 'deliveries'}
              class:text-gray-500={activeTab !== 'deliveries'}
              onclick={() => (activeTab = 'deliveries')}
            >
              Deliveries
            </button>
            <button
              class="py-2 px-4 border-b-2 font-medium text-sm"
              class:border-blue-500={activeTab === 'subscriptions'}
              class:text-blue-600={activeTab === 'subscriptions'}
              class:border-transparent={activeTab !== 'subscriptions'}
              class:text-gray-500={activeTab !== 'subscriptions'}
              onclick={() => goto(`/webhooks/${webhookId}/subscriptions`)}
            >
              Event Subscriptions ({subscriptions.length})
            </button>
          </nav>
        </div>
      </div>

      {#if activeTab === 'deliveries'}
      <!-- Delivery History -->
      <div class="bg-white rounded-lg shadow-sm border p-6">
        <h2 class="text-xl font-bold text-gray-800 mb-8">Delivery History</h2>
        {#if deliveries.length === 0}
          <p class="text-gray-500">No deliveries found for this webhook.</p>
        {:else}
          <div class="overflow-x-auto">
            <table class="w-full text-sm text-left">
              <thead class="text-xs text-gray-700 uppercase bg-gray-50">
                <tr>
                  <th class="px-4 py-3">Delivery ID</th>
                  <th class="px-4 py-3">Event ID</th>
                  <th class="px-4 py-3">Status</th>
                  <th class="px-4 py-3">Attempts</th>
                  <th class="px-4 py-3">Last Attempt</th>
                  <th class="px-4 py-3"></th>
                </tr>
              </thead>
              <tbody>
                {#each deliveries as delivery}
                  <tr class="border-b hover:bg-gray-50">
                    <td class="px-4 py-3 font-mono text-xs"
                      >{delivery.deliveryId}</td
                    >
                    <td class="px-4 py-3 font-mono text-xs"
                      >{delivery.eventId}</td
                    >
                    <td class="px-4 py-3">
                      <span
                        class={`font-semibold ${statusColor[delivery.status]}`}
                      >
                        {WebhookDeliveryStatus[delivery.status].replace(
                          "DELIVERY_",
                          ""
                        )}
                      </span>
                    </td>
                    <td class="px-4 py-3">{delivery.attemptCount}</td>
                    <td class="px-4 py-3"
                      >{formatTimestamp(delivery.lastAttemptedAt)}</td
                    >
                    <td class="px-4 py-3">
                      <button
                        onclick={() => toggleDeliveryExpansion(delivery.deliveryId)}
                        class="text-blue-600 hover:text-blue-800 font-medium"
                      >
                        {expandedDeliveries.has(delivery.deliveryId) ? 'Hide' : 'Details'}
                      </button>
                    </td>
                  </tr>
                  {#if expandedDeliveries.has(delivery.deliveryId)}
                    <tr class="border-b bg-gray-50">
                      <td colspan="6" class="px-1 py-6">
                        <div class="space-y-2">
                          {#if deliveryDetails.has(delivery.deliveryId)}
                            {@const details = deliveryDetails.get(delivery.deliveryId)}
                            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-2 text-sm">
                              <div>
                                <p class="font-semibold text-gray-700">Response Code</p>
                                <p class="font-mono bg-white px-2 py-1 rounded border"
                                   class:text-green-600={details.responseCode >= 200 && details.responseCode < 300}
                                   class:text-yellow-600={details.responseCode >= 300 && details.responseCode < 400}
                                   class:text-red-600={details.responseCode >= 400}
                                >{details.responseCode || 'N/A'}</p>
                              </div>
                              <div>
                                <p class="font-semibold text-gray-700">Created At</p>
                                <p class="bg-white px-2 py-1 rounded border">
                                  {details.createdAt ? formatTimestamp(details.createdAt) : 'N/A'}
                                </p>
                              </div>
                            </div>
                             <div>
                                <p class="font-semibold text-gray-700 mb-2">Request Body</p>
                                <pre class="bg-white font-family-mono p-2 rounded border text-xs overflow-auto font-mono">{formatPayload(details.requestBody)}</pre>
                              </div>
                            {#if details.responseBody}
                              <div>
                                <p class="font-semibold text-gray-700 mb-2">Response Body</p>
                                <pre class="bg-white p-3 rounded border text-xs overflow-auto max-h-32 font-mono">{details.responseBody}</pre>
                              </div>
                            {/if}
                            
                            {#if details.errorMessage}
                              <div>
                                <p class="font-semibold text-gray-700 mb-2">Error Message </p>
                                <p class="bg-red-50 text-red-800 p-3 rounded border border-red-200 text-sm">
                                  {details.errorMessage}
                                </p>
                              </div>
                              <button
                                onclick={() => resendThisDelivery(delivery.deliveryId)}
                                class="mt-2 px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition"
                              >
                                Resend
                              </button>
                            {/if}
                          {:else}
                            <div class="text-center py-4">
                              <p class="text-gray-500">Loading detailed information...</p>
                            </div>
                          {/if}
                        </div>
                      </td>
                    </tr>
                  {/if}
                {/each}
              </tbody>
            </table>
          </div>

          <!-- Pagination controls -->
          <div class="mt-6 flex items-center justify-between border-t pt-4">
            <div class="text-sm text-gray-500">
              Showing {offset + 1} to {Math.min(offset + limit, totalCount)} of {totalCount} deliveries
            </div>
            <div class="flex gap-2">
              <button
                class="px-4 py-2 border rounded-md text-sm disabled:opacity-50 hover:bg-gray-50"
                onclick={prevPage}
                disabled={offset === 0}
              >
                Previous
              </button>
              <button
                class="px-4 py-2 border rounded-md text-sm disabled:opacity-50 hover:bg-gray-50"
                onclick={nextPage}
                disabled={offset + limit >= totalCount}
              >
                Next
              </button>
            </div>
          </div>
        {/if}
      </div>
      {/if}
    {/if}
  </main>
</div>

<style>
  
  .border-primary {
    border-color: #1d4ed8;
  }
</style>
