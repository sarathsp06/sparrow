<script lang="ts">
  import favicon from '$lib/assets/favicon.svg';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import {
    webhookClient,
    deliveryClient,
    healthClient,
    subscriptionClient,
  } from '$lib/services';
  import { onMount } from 'svelte';
  import { namespaceState } from '$lib/namespace.svelte';
  import type {
    EventSubscription,
    RegisteredWebhook,
    WebhookDelivery,
    WebhookHealthMetrics,
  } from '../../../../../proto/webhook_pb.js';
  import type { Timestamp } from '@bufbuild/protobuf/wkt';
  import { WebhookDeliveryStatus, WebhookHealth } from '../../../../../proto/webhook_pb.js';
  import HealthBadge from '$lib/components/HealthBadge.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';

  let webhook: RegisteredWebhook | undefined = $state();
  let deliveries: WebhookDelivery[] = $state([]);
  let healthMetrics: WebhookHealthMetrics | undefined = $state();
  let subscriptions: EventSubscription[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let expandedDeliveries: Set<string> = $state(new Set());
  let deliveryDetails: Map<string, any> = $state(new Map());
  let activeTab = $state<'deliveries' | 'config'>('deliveries');

  // Inline URL editing
  let editingUrl = $state(false);
  let editedUrl = $state('');
  let savingUrl = $state(false);

  // Unregister confirmation
  let confirmUnregister = $state(false);

  // Pagination
  let limit = $state(25);
  let offset = $state(0);
  let totalCount = $state(0);

  const webhookId = page.params.webhookId;

  let currentPage = $derived(Math.floor(offset / limit) + 1);
  let totalPages = $derived(Math.max(1, Math.ceil(totalCount / limit)));

  function formatPayload(payload: string): string {
    try { return JSON.stringify(JSON.parse(payload), null, 2); }
    catch { return payload; }
  }

  function formatTimestamp(timestamp: Timestamp | undefined | null): string {
    if (!timestamp) return 'N/A';
    return new Date(Number(timestamp.seconds) * 1000).toLocaleString();
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
      await fetchData();
    } catch (e: any) {
      error = `Failed to update status: ${e.message}`;
    }
  }

  async function executeUnregister() {
    if (!webhook) return;
    try {
      await webhookClient.unregisterWebhook({
        webhookId,
        namespace: namespaceState.current,
      });
      confirmUnregister = false;
      goto('/webhooks');
    } catch (e: any) {
      error = `Failed to unregister webhook: ${e.message}`;
      confirmUnregister = false;
    }
  }

  async function resendDelivery(deliveryId: string) {
    try {
      await deliveryClient.retryDelivery({
        deliveryId,
        namespace: namespaceState.current,
      });
      await fetchData();
    } catch (e: any) {
      error = `Failed to resend delivery: ${e.message}`;
    }
  }

  function startEditUrl() {
    if (!webhook) return;
    editedUrl = webhook.url;
    editingUrl = true;
    error = '';
  }

  function cancelEditUrl() {
    editingUrl = false;
    editedUrl = '';
  }

  async function saveWebhookUrl() {
    if (!webhook) return;
    const trimmedUrl = editedUrl.trim();
    if (!trimmedUrl) { error = 'URL is required'; return; }
    try { new URL(trimmedUrl); } catch { error = 'Enter a valid URL'; return; }

    savingUrl = true;
    error = '';
    try {
      await webhookClient.updateWebhookConfig({
        webhookId,
        namespace: namespaceState.current,
        updates: { url: trimmedUrl, active: webhook.active },
      });
      editingUrl = false;
      await fetchData();
    } catch (e: any) {
      error = `Failed to update URL: ${e.message}`;
    } finally {
      savingUrl = false;
    }
  }

  async function toggleDeliveryExpansion(deliveryId: string) {
    if (expandedDeliveries.has(deliveryId)) {
      expandedDeliveries.delete(deliveryId);
      expandedDeliveries = new Set(expandedDeliveries);
    } else {
      expandedDeliveries.add(deliveryId);
      expandedDeliveries = new Set(expandedDeliveries);

      if (!deliveryDetails.has(deliveryId)) {
        try {
          const detailRes = await deliveryClient.getDeliveryStatus({
            deliveryId,
            namespace: namespaceState.current,
          });
          deliveryDetails.set(deliveryId, detailRes.delivery);
          deliveryDetails = new Map(deliveryDetails);
        } catch (e: any) {
          console.error('Failed to load delivery details:', e);
        }
      }
    }
  }

  function handlePageChange(pageNum: number) {
    offset = (pageNum - 1) * limit;
    fetchData();
  }

  // Health metrics helpers
  let successRatePercent = $derived(healthMetrics ? (healthMetrics.successRate * 100).toFixed(1) : '0');
  let successRateColor = $derived.by(() => {
    if (!healthMetrics) return 'text-gray-400';
    if (healthMetrics.successRate >= 0.95) return 'text-green-600';
    if (healthMetrics.successRate >= 0.8) return 'text-yellow-600';
    return 'text-red-600';
  });
</script>

<svelte:head>
  <title>{webhook?.description || 'Webhook'} - {webhookId} | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <main class="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    {#if loading}
      <!-- Loading skeleton -->
      <nav class="mb-4">
        <div class="h-4 bg-gray-200 rounded w-28 animate-pulse"></div>
      </nav>
      <div class="bg-white rounded-lg border border-gray-200 p-5 mb-6 animate-pulse">
        <div class="flex items-center gap-3 mb-3">
          <div class="h-6 bg-gray-200 rounded w-48"></div>
          <div class="h-5 bg-gray-200 rounded-full w-20"></div>
        </div>
        <div class="h-4 bg-gray-100 rounded w-72 mb-3"></div>
        <div class="flex gap-6">
          <div class="h-4 bg-gray-100 rounded w-32"></div>
          <div class="h-4 bg-gray-100 rounded w-40"></div>
        </div>
      </div>
      <div class="bg-white rounded-lg border border-gray-200 p-5 mb-6 animate-pulse">
        <div class="h-4 bg-gray-200 rounded w-32 mb-4"></div>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
          {#each Array(4) as _}
            <div><div class="h-3 bg-gray-100 rounded w-16 mb-1"></div><div class="h-6 bg-gray-200 rounded w-12"></div></div>
          {/each}
        </div>
      </div>
    {:else if error && !webhook}
      <div class="bg-red-50 border border-red-200 rounded-lg p-4">
        <p class="text-sm text-red-700">{error}</p>
        <a href="/webhooks" class="text-sm text-red-600 hover:text-red-800 underline mt-2 inline-block">Back to webhooks</a>
      </div>
    {:else if webhook}
      <!-- Breadcrumb -->
      <nav class="mb-4">
        <a href="/webhooks" class="text-sm text-gray-500 hover:text-gray-700 transition">&larr; All Webhooks</a>
      </nav>

      {#if error}
        <div class="bg-red-50 border border-red-200 rounded-lg p-3 mb-4 flex items-start justify-between">
          <p class="text-sm text-red-700">{error}</p>
          <button onclick={() => { error = ''; }} class="text-red-400 hover:text-red-600 ml-3 shrink-0" aria-label="Dismiss error">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      {/if}

      <!-- Webhook Header Card -->
      <div class="bg-white rounded-lg border border-gray-200 p-5 mb-6">
        <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-3 mb-2">
              <h1 class="text-xl font-bold text-gray-900 truncate">
                {webhook.description || 'Webhook'}
              </h1>
              <HealthBadge health={webhook.health} size="md" />
              {#if webhook.active}
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-50 text-green-700">Active</span>
              {:else}
                <span class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-500">Paused</span>
              {/if}
            </div>

            <!-- URL (editable) -->
            {#if editingUrl}
              <div class="flex items-center gap-2 mb-3">
                <input
                  type="url"
                  bind:value={editedUrl}
                  class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900 font-mono"
                  placeholder="https://example.com/webhook"
                />
                <button
                  onclick={saveWebhookUrl}
                  disabled={savingUrl}
                  class="px-3 py-1.5 text-xs font-medium bg-gray-900 text-white rounded-lg hover:bg-gray-800 disabled:opacity-50 transition"
                >
                  {savingUrl ? 'Saving...' : 'Save'}
                </button>
                <button
                  onclick={cancelEditUrl}
                  disabled={savingUrl}
                  class="px-3 py-1.5 text-xs font-medium bg-gray-100 text-gray-700 rounded-lg hover:bg-gray-200 disabled:opacity-50 transition"
                >
                  Cancel
                </button>
              </div>
            {:else}
              <button
                onclick={startEditUrl}
                class="group flex items-center gap-1.5 mb-3 text-left"
                title="Click to edit URL"
              >
                <span class="text-sm font-mono text-gray-600 break-all">{webhook.url}</span>
                <svg class="w-3.5 h-3.5 text-gray-400 opacity-0 group-hover:opacity-100 transition shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                </svg>
              </button>
            {/if}

            <div class="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-gray-500">
              <span>Namespace: <span class="font-medium text-gray-700">{webhook.namespace}</span></span>
              <span>Created: <span class="font-medium text-gray-700">{formatTimestamp(webhook.createdAt)}</span></span>
              <span class="font-mono text-xs text-gray-400">ID: {webhookId}</span>
            </div>
          </div>

          <!-- Actions -->
          <div class="flex items-center gap-2 shrink-0">
            <button
              onclick={toggleWebhookStatus}
              class="px-3 py-1.5 text-sm font-medium rounded-lg transition {webhook.active
                ? 'bg-yellow-50 text-yellow-700 hover:bg-yellow-100 border border-yellow-200'
                : 'bg-green-50 text-green-700 hover:bg-green-100 border border-green-200'}"
            >
              {webhook.active ? 'Pause' : 'Resume'}
            </button>
            <button
              onclick={() => { confirmUnregister = true; }}
              class="px-3 py-1.5 text-sm font-medium rounded-lg transition bg-red-50 text-red-700 hover:bg-red-100 border border-red-200"
            >
              Unregister
            </button>
          </div>
        </div>

        <!-- Subscribed events -->
        <div class="mt-4 pt-4 border-t border-gray-100">
          <p class="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">Subscribed Events</p>
          <div class="flex flex-wrap gap-1.5">
            {#each webhook.events as event}
              <span class="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-50 text-blue-700">
                {event}
              </span>
            {/each}
            {#if webhook.events.length === 0}
              <span class="text-xs text-gray-400">No events subscribed</span>
            {/if}
          </div>
        </div>
      </div>

      <!-- Health Metrics -->
      {#if healthMetrics}
        <div class="bg-white rounded-lg border border-gray-200 p-5 mb-6">
          <h2 class="text-sm font-semibold text-gray-900 uppercase tracking-wide mb-4">Health Metrics</h2>
          <div class="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-4">
            <div>
              <p class="text-xs text-gray-500">Success Rate</p>
              <p class="text-xl font-bold {successRateColor}">{successRatePercent}%</p>
            </div>
            <div>
              <p class="text-xs text-gray-500">Total</p>
              <p class="text-xl font-bold text-gray-900">{healthMetrics.totalDeliveries}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500">Successful</p>
              <p class="text-xl font-bold text-green-600">{healthMetrics.successfulDeliveries}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500">Failed</p>
              <p class="text-xl font-bold text-red-600">{healthMetrics.failedDeliveries}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500">Avg Response</p>
              <p class="text-xl font-bold text-gray-900">{healthMetrics.avgResponseTime}<span class="text-xs font-normal text-gray-400">ms</span></p>
            </div>
            <div>
              <p class="text-xs text-gray-500">Last Success</p>
              <p class="text-xs font-medium text-gray-700">{healthMetrics.lastSuccessAt ? formatTimestamp(healthMetrics.lastSuccessAt) : 'N/A'}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500">Last Failure</p>
              <p class="text-xs font-medium text-gray-700">{healthMetrics.lastFailureAt ? formatTimestamp(healthMetrics.lastFailureAt) : 'N/A'}</p>
            </div>
          </div>

          <!-- Success rate bar -->
          {#if healthMetrics.totalDeliveries > 0}
            <div class="mt-4">
              <div class="w-full bg-gray-100 rounded-full h-2 overflow-hidden">
                <div
                  class="h-full rounded-full transition-all duration-500 {healthMetrics.successRate >= 0.95 ? 'bg-green-500' : healthMetrics.successRate >= 0.8 ? 'bg-yellow-500' : 'bg-red-500'}"
                  style="width: {healthMetrics.successRate * 100}%"
                ></div>
              </div>
            </div>
          {/if}
        </div>
      {/if}

      <!-- Tabs -->
      <div class="border-b border-gray-200 mb-6">
        <nav class="flex gap-6">
          <button
            class="pb-3 text-sm font-medium border-b-2 transition {activeTab === 'deliveries'
              ? 'border-gray-900 text-gray-900'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}"
            onclick={() => (activeTab = 'deliveries')}
          >
            Deliveries
            <span class="ml-1 px-1.5 py-0.5 text-xs rounded-full bg-gray-100 text-gray-600">{totalCount}</span>
          </button>
          <button
            class="pb-3 text-sm font-medium border-b-2 transition {activeTab === 'config'
              ? 'border-gray-900 text-gray-900'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}"
            onclick={() => (activeTab = 'config')}
          >
            Configuration
          </button>
          <button
            class="pb-3 text-sm font-medium border-b-2 transition border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300"
            onclick={() => goto(`/webhooks/${webhookId}/subscriptions`)}
          >
            Subscriptions
            <span class="ml-1 px-1.5 py-0.5 text-xs rounded-full bg-gray-100 text-gray-600">{subscriptions.length}</span>
          </button>
        </nav>
      </div>

      <!-- Deliveries Tab -->
      {#if activeTab === 'deliveries'}
        <div class="bg-white rounded-lg border border-gray-200">
          {#if deliveries.length === 0}
            <EmptyState
              icon="send"
              title="No deliveries yet"
              description="Deliveries will appear here when events are pushed to this webhook."
            />
          {:else}
            <div class="overflow-x-auto">
              <table class="w-full text-sm text-left">
                <thead>
                  <tr class="border-b border-gray-200 bg-gray-50/50">
                    <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Delivery ID</th>
                    <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden sm:table-cell">Event ID</th>
                    <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider">Status</th>
                    <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden md:table-cell">Attempts</th>
                    <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider hidden lg:table-cell">Last Attempt</th>
                    <th class="px-4 py-3 text-xs font-semibold text-gray-500 uppercase tracking-wider"></th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-100">
                  {#each deliveries as delivery}
                    <tr class="hover:bg-gray-50 transition">
                      <td class="px-4 py-3">
                        <span class="font-mono text-xs text-gray-700">{delivery.deliveryId.substring(0, 12)}...</span>
                        <!-- Show event ID inline on mobile -->
                        <span class="block sm:hidden font-mono text-xs text-gray-400 mt-0.5">{delivery.eventId.substring(0, 12)}...</span>
                      </td>
                      <td class="px-4 py-3 font-mono text-xs text-gray-700 hidden sm:table-cell">{delivery.eventId.substring(0, 16)}...</td>
                      <td class="px-4 py-3">
                        <StatusBadge status={delivery.status} />
                      </td>
                      <td class="px-4 py-3 text-gray-700 hidden md:table-cell">{delivery.attemptCount}</td>
                      <td class="px-4 py-3 text-xs text-gray-500 hidden lg:table-cell">{formatTimestamp(delivery.lastAttemptedAt)}</td>
                      <td class="px-4 py-3">
                        <button
                          onclick={() => toggleDeliveryExpansion(delivery.deliveryId)}
                          class="text-xs font-medium text-gray-600 hover:text-gray-900 transition"
                        >
                          {expandedDeliveries.has(delivery.deliveryId) ? 'Hide' : 'Details'}
                        </button>
                      </td>
                    </tr>
                    {#if expandedDeliveries.has(delivery.deliveryId)}
                      <tr class="bg-gray-50/50">
                        <td colspan="6" class="px-4 py-4">
                          {#if deliveryDetails.has(delivery.deliveryId)}
                            {@const details = deliveryDetails.get(delivery.deliveryId)}
                            <div class="space-y-3">
                              <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 text-sm">
                                <div>
                                  <p class="text-xs font-medium text-gray-500 mb-0.5">Response Code</p>
                                  <span class="font-mono text-sm {details.responseCode >= 200 && details.responseCode < 300 ? 'text-green-600' : details.responseCode >= 400 ? 'text-red-600' : 'text-gray-700'}">
                                    {details.responseCode || 'N/A'}
                                  </span>
                                </div>
                                <div>
                                  <p class="text-xs font-medium text-gray-500 mb-0.5">Attempts</p>
                                  <span class="text-sm text-gray-700">{delivery.attemptCount}</span>
                                </div>
                                <div>
                                  <p class="text-xs font-medium text-gray-500 mb-0.5">Created</p>
                                  <span class="text-sm text-gray-700">{details.createdAt ? formatTimestamp(details.createdAt) : 'N/A'}</span>
                                </div>
                                <div>
                                  <p class="text-xs font-medium text-gray-500 mb-0.5">Last Attempt</p>
                                  <span class="text-sm text-gray-700">{formatTimestamp(delivery.lastAttemptedAt)}</span>
                                </div>
                              </div>

                              <div>
                                <p class="text-xs font-medium text-gray-500 mb-1">Request Body</p>
                                <pre class="bg-white p-3 rounded-lg border border-gray-200 text-xs overflow-auto max-h-48 font-mono text-gray-800">{formatPayload(details.requestBody)}</pre>
                              </div>

                              {#if details.responseBody}
                                <div>
                                  <p class="text-xs font-medium text-gray-500 mb-1">Response Body</p>
                                  <pre class="bg-white p-3 rounded-lg border border-gray-200 text-xs overflow-auto max-h-32 font-mono text-gray-800">{details.responseBody}</pre>
                                </div>
                              {/if}

                              {#if details.errorMessage}
                                <div>
                                  <p class="text-xs font-medium text-gray-500 mb-1">Error</p>
                                  <div class="bg-red-50 border border-red-200 rounded-lg p-3">
                                    <p class="text-sm text-red-700">{details.errorMessage}</p>
                                  </div>
                                </div>
                              {/if}

                              <!-- Actions row -->
                              <div class="flex items-center gap-2 pt-1">
                                {#if delivery.status === WebhookDeliveryStatus.DELIVERY_FAILED || details.errorMessage}
                                  <button
                                    onclick={() => resendDelivery(delivery.deliveryId)}
                                    class="inline-flex items-center px-3 py-1.5 text-xs font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 transition"
                                  >
                                    Retry Delivery
                                  </button>
                                {/if}
                                <a
                                  href="/deliveries/{delivery.deliveryId}"
                                  class="inline-flex items-center px-3 py-1.5 text-xs font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition"
                                >
                                  Full Details
                                </a>
                              </div>
                            </div>
                          {:else}
                            <div class="flex items-center justify-center py-4">
                              <img src={favicon} alt="Loading" class="w-4 h-4 animate-spin mr-2" />
                              <span class="text-sm text-gray-500">Loading details...</span>
                            </div>
                          {/if}
                        </td>
                      </tr>
                    {/if}
                  {/each}
                </tbody>
              </table>
            </div>

            <div class="border-t border-gray-200 px-4">
              <Pagination
                {currentPage}
                {totalPages}
                {totalCount}
                pageSize={limit}
                onPageChange={handlePageChange}
                itemLabel="deliveries"
              />
            </div>
          {/if}
        </div>
      {/if}

      <!-- Configuration Tab -->
      {#if activeTab === 'config'}
        <div class="space-y-4">
          <!-- HTTP Configuration -->
          <div class="bg-white rounded-lg border border-gray-200 p-5">
            <h3 class="text-sm font-semibold text-gray-900 uppercase tracking-wide mb-4">HTTP Configuration</h3>
            {#if webhook.httpConfig}
              <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                <div>
                  <p class="text-xs text-gray-500 mb-0.5">Max Retries</p>
                  <p class="text-sm font-medium text-gray-900">{webhook.httpConfig.maxRetries}</p>
                </div>
                <div>
                  <p class="text-xs text-gray-500 mb-0.5">Retry Backoff</p>
                  <p class="text-sm font-medium text-gray-900">{webhook.httpConfig.retryBackoffSeconds}s</p>
                </div>
                <div>
                  <p class="text-xs text-gray-500 mb-0.5">Request Timeout</p>
                  <p class="text-sm font-medium text-gray-900">{webhook.httpConfig.requestTimeoutSeconds}s</p>
                </div>
                <div>
                  <p class="text-xs text-gray-500 mb-0.5">Content Type</p>
                  <p class="text-sm font-medium text-gray-900 font-mono">{webhook.httpConfig.contentType || 'application/json'}</p>
                </div>
                <div>
                  <p class="text-xs text-gray-500 mb-0.5">User Agent</p>
                  <p class="text-sm font-medium text-gray-900 font-mono">{webhook.httpConfig.userAgent || 'Sparrow-Webhook/1.0'}</p>
                </div>
                <div>
                  <p class="text-xs text-gray-500 mb-0.5">Expected Status Codes</p>
                  <div class="flex flex-wrap gap-1">
                    {#each webhook.httpConfig.expectedStatusCodes as code}
                      <span class="px-1.5 py-0.5 text-xs font-mono bg-gray-100 text-gray-700 rounded">{code}</span>
                    {/each}
                    {#if webhook.httpConfig.expectedStatusCodes.length === 0}
                      <span class="text-sm text-gray-400">Default (2xx)</span>
                    {/if}
                  </div>
                </div>
              </div>

              <!-- Boolean flags -->
              <div class="mt-4 pt-4 border-t border-gray-100">
                <div class="flex flex-wrap gap-3">
                  <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium {webhook.httpConfig.followRedirects ? 'bg-green-50 text-green-700' : 'bg-gray-100 text-gray-500'}">
                    <span class="w-1.5 h-1.5 rounded-full {webhook.httpConfig.followRedirects ? 'bg-green-500' : 'bg-gray-400'}"></span>
                    Follow Redirects
                  </span>
                  <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium {webhook.httpConfig.verifySsl ? 'bg-green-50 text-green-700' : 'bg-red-50 text-red-700'}">
                    <span class="w-1.5 h-1.5 rounded-full {webhook.httpConfig.verifySsl ? 'bg-green-500' : 'bg-red-500'}"></span>
                    Verify SSL
                  </span>
                  <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium {webhook.httpConfig.captureResponseBody ? 'bg-blue-50 text-blue-700' : 'bg-gray-100 text-gray-500'}">
                    <span class="w-1.5 h-1.5 rounded-full {webhook.httpConfig.captureResponseBody ? 'bg-blue-500' : 'bg-gray-400'}"></span>
                    Capture Response
                  </span>
                  {#if webhook.httpConfig.webhookSecret}
                    <span class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium bg-purple-50 text-purple-700">
                      <span class="w-1.5 h-1.5 rounded-full bg-purple-500"></span>
                      Secret Configured
                    </span>
                  {/if}
                </div>
              </div>
            {:else}
              <p class="text-sm text-gray-500">Using default HTTP configuration.</p>
            {/if}
          </div>

          <!-- Custom Headers -->
          <div class="bg-white rounded-lg border border-gray-200 p-5">
            <h3 class="text-sm font-semibold text-gray-900 uppercase tracking-wide mb-4">Custom Headers</h3>
            {#if Object.keys(webhook.headers || {}).length > 0}
              <div class="space-y-1.5">
                {#each Object.entries(webhook.headers) as [key, value]}
                  <div class="flex items-center gap-2 text-sm">
                    <span class="font-mono text-gray-900 font-medium">{key}:</span>
                    <span class="font-mono text-gray-600">{value}</span>
                  </div>
                {/each}
              </div>
            {:else}
              <p class="text-sm text-gray-500">No custom headers configured.</p>
            {/if}
          </div>
        </div>
      {/if}
    {/if}
  </main>
</div>

<!-- Confirm Unregister Dialog -->
<ConfirmDialog
  open={confirmUnregister}
  title="Unregister Webhook"
  message="This will permanently remove the webhook and stop all future deliveries. This action cannot be undone."
  confirmLabel="Unregister"
  variant="danger"
  onconfirm={executeUnregister}
  oncancel={() => { confirmUnregister = false; }}
/>
