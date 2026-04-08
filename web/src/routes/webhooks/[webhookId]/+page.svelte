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
  import { getCategoryBadge, ERROR_CATEGORIES, formatAPIError } from '$lib/utils';
  import { onMount, onDestroy } from 'svelte';

  import type {
    EventSubscription,
    RegisteredWebhook,
    WebhookDelivery,
    WebhookHealthMetrics,
  } from '../../../../../proto/webhook_pb.js';
  import type { Timestamp } from '@bufbuild/protobuf/wkt';
  import { timestampFromDate, FieldMaskSchema } from '@bufbuild/protobuf/wkt';
  import { create } from '@bufbuild/protobuf';
  import { WebhookDeliveryStatus, WebhookHealth } from '../../../../../proto/webhook_pb.js';
  import HealthBadge from '$lib/components/HealthBadge.svelte';
  import CopyableId from '$lib/components/CopyableId.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import SubscriptionManager from '$lib/components/SubscriptionManager.svelte';
  import BatchProgress from '$lib/components/BatchProgress.svelte';

  let webhook: RegisteredWebhook | undefined = $state();
  let deliveries: WebhookDelivery[] = $state([]);
  let healthMetrics: WebhookHealthMetrics | undefined = $state();
  let subscriptions: EventSubscription[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let expandedDeliveries: Set<string> = $state(new Set());
  let deliveryDetails: Map<string, any> = $state(new Map());
  let activeTab = $state<'deliveries' | 'config' | 'subscriptions'>('deliveries');

  // Inline URL editing
  let editingUrl = $state(false);
  let editedUrl = $state('');
  let savingUrl = $state(false);

  // Config editing
  let editingConfig = $state(false);
  let savingConfig = $state(false);
  let configForm = $state({
    description: '',
    url: '',
    active: true,
    maxRetries: 3,
    retryBackoffSeconds: 60,
    requestTimeoutSeconds: 30,
    captureResponseBody: false,
    followRedirects: true,
    verifySsl: true,
    expectedStatusCodes: '200, 201, 202, 204',
    webhookSecret: '',
    userAgent: 'Sparrow-Webhook/1.0',
    contentType: 'application/json',
    headers: {} as Record<string, string>,
  });
  let configHeaderKey = $state('');
  let configHeaderValue = $state('');
  let configSecretHeaderKey = $state('');
  let configSecretHeaderValue = $state('');
  // Track existing secret header keys that haven't been replaced (display-only, never sent back)
  let existingSecretHeaderKeys = $state<Set<string>>(new Set());
  // Track newly added or replaced secret headers (these are the only ones sent to the backend)
  let newSecretHeaders = $state<Record<string, string>>({});

  // Unregister confirmation
  let confirmUnregister = $state(false);

  // Pagination
  let limit = $state(25);
  let offset = $state(0);
  let totalCount = $state(0);

  // Delivery filters
  let deliveryStatusFilter = $state('');
  let deliveryErrorCategoryFilter = $state('');
  let deliveryCreatedAfterFilter = $state('');
  let deliveryCreatedBeforeFilter = $state('');

  // Batch retry state
  let retryId = $state('');
  let batchStatus = $state<{ status: string; total: number; processed: number; failed: number } | undefined>();
  let preparingRetry = $state(false);
  let confirmRetry = $state(false);
  let retryTotal = $state(0);
  let pollingTimer: ReturnType<typeof setInterval> | undefined;

  const webhookId = page.params.webhookId ?? '';

  let currentPage = $derived(Math.floor(offset / limit) + 1);
  let totalPages = $derived(Math.max(1, Math.ceil(totalCount / limit)));

  let hasDeliveryFilters = $derived(
    deliveryStatusFilter !== '' ||
    deliveryErrorCategoryFilter !== '' ||
    deliveryCreatedAfterFilter !== '' ||
    deliveryCreatedBeforeFilter !== ''
  );

  onDestroy(() => {
    if (pollingTimer) clearInterval(pollingTimer);
  });

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
      // Fetch webhook first with empty namespace (backend looks up by ID)
      const webhookRes = await webhookClient.listWebhooks({ namespace: '', webhookId });
      webhook = webhookRes.webhooks?.[0];

      if (!webhook) {
        error = 'Webhook not found';
        return;
      }

      // Use the webhook's own namespace for related data
      const ns = webhook.namespace;

      // Build delivery request with filters
      const deliveryReq: Record<string, any> = {
        webhookId,
        namespace: ns,
        pagination: { limit, offset },
      };
      if (deliveryStatusFilter) deliveryReq.status = deliveryStatusFilter;
      if (deliveryErrorCategoryFilter) deliveryReq.errorCategory = deliveryErrorCategoryFilter;
      if (deliveryCreatedAfterFilter) deliveryReq.createdAfter = timestampFromDate(new Date(deliveryCreatedAfterFilter));
      if (deliveryCreatedBeforeFilter) deliveryReq.createdBefore = timestampFromDate(new Date(deliveryCreatedBeforeFilter));

      const [deliveriesRes, healthRes, subscriptionsRes] = await Promise.all([
        deliveryClient.listDeliveries(deliveryReq),
        healthClient.getWebhookHealth({ webhookId, namespace: ns }),
        subscriptionClient.listSubscriptions({ webhookId, namespace: ns }),
      ]);

      deliveries = deliveriesRes.deliveries || [];
      totalCount = deliveriesRes.pagination?.totalCount || 0;
      healthMetrics = healthRes.metrics;
      subscriptions = subscriptionsRes.subscriptions || [];
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to load data');
    } finally {
      loading = false;
    }
  }

  onMount(fetchData);

  async function toggleWebhookStatus() {
    if (!webhook) return;
    try {
      if (webhook.active) {
        await webhookClient.pauseWebhook({ webhookId, namespace: webhook.namespace });
      } else {
        await webhookClient.resumeWebhook({ webhookId, namespace: webhook.namespace });
      }
      await fetchData();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to update status');
    }
  }

  async function executeUnregister() {
    if (!webhook) return;
    try {
      await webhookClient.unregisterWebhook({
        webhookId,
        namespace: webhook.namespace,
      });
      confirmUnregister = false;
      goto('/webhooks');
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to unregister webhook');
      confirmUnregister = false;
    }
  }

  async function resendDelivery(deliveryId: string) {
    try {
      await deliveryClient.retryDelivery({
        deliveryId,
        namespace: webhook?.namespace || '',
      });
      await fetchData();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to resend delivery');
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
        namespace: webhook.namespace,
        updates: { url: trimmedUrl, active: webhook.active },
        updateMask: create(FieldMaskSchema, { paths: ['url', 'active'] }),
      });
      editingUrl = false;
      await fetchData();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to update URL');
    } finally {
      savingUrl = false;
    }
  }

  function startEditConfig() {
    if (!webhook) return;
    configForm = {
      description: webhook.description || '',
      url: webhook.url || '',
      active: webhook.active,
      maxRetries: webhook.httpConfig?.maxRetries ?? 3,
      retryBackoffSeconds: webhook.httpConfig?.retryBackoffSeconds ?? 60,
      requestTimeoutSeconds: webhook.httpConfig?.requestTimeoutSeconds ?? 30,
      captureResponseBody: webhook.httpConfig?.captureResponseBody ?? false,
      followRedirects: webhook.httpConfig?.followRedirects ?? true,
      verifySsl: webhook.httpConfig?.verifySsl ?? true,
      expectedStatusCodes: (webhook.httpConfig?.expectedStatusCodes || [200, 201, 202, 204]).join(', '),
      webhookSecret: '',
      userAgent: webhook.httpConfig?.userAgent || 'Sparrow-Webhook/1.0',
      contentType: webhook.httpConfig?.contentType || 'application/json',
      headers: { ...(webhook.headers || {}) },
    };
    // Track existing secret header keys (masked values — never sent back)
    existingSecretHeaderKeys = new Set(Object.keys(webhook.secretHeaders || {}));
    // No new secret headers initially
    newSecretHeaders = {};
    configHeaderKey = '';
    configHeaderValue = '';
    configSecretHeaderKey = '';
    configSecretHeaderValue = '';
    editingConfig = true;
    error = '';
  }

  function cancelEditConfig() {
    editingConfig = false;
  }

  function addConfigHeader() {
    if (configHeaderKey.trim() && configHeaderValue.trim()) {
      configForm.headers = { ...configForm.headers, [configHeaderKey.trim()]: configHeaderValue.trim() };
      configHeaderKey = '';
      configHeaderValue = '';
    }
  }

  function removeConfigHeader(key: string) {
    const { [key]: _, ...rest } = configForm.headers;
    configForm.headers = rest;
  }

  function addConfigSecretHeader() {
    if (configSecretHeaderKey.trim() && configSecretHeaderValue.trim()) {
      const key = configSecretHeaderKey.trim();
      // If replacing an existing key, remove it from the "existing" set
      existingSecretHeaderKeys.delete(key);
      existingSecretHeaderKeys = new Set(existingSecretHeaderKeys);
      // Track as a new/replaced header with the actual plaintext value
      newSecretHeaders = { ...newSecretHeaders, [key]: configSecretHeaderValue.trim() };
      configSecretHeaderKey = '';
      configSecretHeaderValue = '';
    }
  }

  function removeConfigSecretHeader(key: string) {
    // Remove from both tracking sets
    existingSecretHeaderKeys.delete(key);
    existingSecretHeaderKeys = new Set(existingSecretHeaderKeys);
    const { [key]: _, ...rest } = newSecretHeaders;
    newSecretHeaders = rest;
  }

  async function saveConfig() {
    if (!webhook) return;
    savingConfig = true;
    error = '';
    try {
      // Parse expected status codes
      const statusCodes = configForm.expectedStatusCodes
        .split(',')
        .map((s: string) => parseInt(s.trim(), 10))
        .filter((n: number) => !isNaN(n) && n >= 100 && n <= 599);

      if (statusCodes.length === 0) {
        error = 'At least one valid expected status code is required';
        savingConfig = false;
        return;
      }

      // Validate URL
      const trimmedUrl = configForm.url.trim();
      if (!trimmedUrl) { error = 'URL is required'; savingConfig = false; return; }
      try { new URL(trimmedUrl); } catch { error = 'Enter a valid URL'; savingConfig = false; return; }

      // Build the field mask: always include non-sensitive fields,
      // only include secrets when the user explicitly provided new values.
      const maskPaths = ['url', 'active', 'description', 'headers', 'http_config'];
      const hasNewSecretHeaders = Object.keys(newSecretHeaders).length > 0;
      if (hasNewSecretHeaders) {
        maskPaths.push('secret_headers');
      }
      if (configForm.webhookSecret) {
        maskPaths.push('http_config.webhook_secret');
      }

      await webhookClient.updateWebhookConfig({
        webhookId,
        namespace: webhook.namespace,
        updateMask: create(FieldMaskSchema, { paths: maskPaths }),
        updates: {
          url: trimmedUrl,
          active: configForm.active,
          description: configForm.description,
          headers: configForm.headers,
          ...(hasNewSecretHeaders ? { secretHeaders: newSecretHeaders } : {}),
          httpConfig: {
            maxRetries: configForm.maxRetries,
            retryBackoffSeconds: configForm.retryBackoffSeconds,
            requestTimeoutSeconds: configForm.requestTimeoutSeconds,
            captureResponseBody: configForm.captureResponseBody,
            followRedirects: configForm.followRedirects,
            verifySsl: configForm.verifySsl,
            expectedStatusCodes: statusCodes,
            ...(configForm.webhookSecret ? { webhookSecret: configForm.webhookSecret } : {}),
            userAgent: configForm.userAgent,
            contentType: configForm.contentType,
          },
        },
      });
      editingConfig = false;
      await fetchData();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to update configuration');
    } finally {
      savingConfig = false;
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
            namespace: webhook?.namespace || '',
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

  function applyDeliveryFilters() {
    offset = 0;
    fetchData();
  }

  function clearDeliveryFilters() {
    deliveryStatusFilter = '';
    deliveryErrorCategoryFilter = '';
    deliveryCreatedAfterFilter = '';
    deliveryCreatedBeforeFilter = '';
    applyDeliveryFilters();
  }

  // -- Batch Retry --

  async function prepareRetryBatch() {
    if (!webhook) return;
    preparingRetry = true;
    try {
      const req: Record<string, any> = {
        webhookId,
        namespace: webhook.namespace,
        pagination: { limit: 1, offset: 0 },
        prepareRetry: true,
      };
      if (deliveryStatusFilter) req.status = deliveryStatusFilter;
      if (deliveryErrorCategoryFilter) req.errorCategory = deliveryErrorCategoryFilter;
      if (deliveryCreatedAfterFilter) req.createdAfter = timestampFromDate(new Date(deliveryCreatedAfterFilter));
      if (deliveryCreatedBeforeFilter) req.createdBefore = timestampFromDate(new Date(deliveryCreatedBeforeFilter));

      const res = await deliveryClient.listDeliveries(req);
      if (res.retryId) {
        retryId = res.retryId;
        retryTotal = res.pagination?.totalCount || 0;
        confirmRetry = true;
      } else {
        error = 'No matching deliveries to retry.';
      }
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to prepare retry');
    } finally {
      preparingRetry = false;
    }
  }

  async function executeRetry() {
    confirmRetry = false;
    if (!retryId) return;
    try {
      const res = await deliveryClient.retryDeliveries({ retryId });
      batchStatus = { status: res.status, total: res.total, processed: 0, failed: 0 };
      startRetryPolling();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to start retry');
    }
  }

  function startRetryPolling() {
    if (pollingTimer) clearInterval(pollingTimer);
    pollingTimer = setInterval(async () => {
      if (!retryId) { stopRetryPolling(); return; }
      try {
        const res = await deliveryClient.getRetryStatus({ retryId });
        if (res.batch) {
          batchStatus = {
            status: res.batch.status,
            total: res.batch.total,
            processed: res.batch.processed,
            failed: res.batch.failed,
          };
          if (res.batch.status === 'completed' || res.batch.status === 'failed' || res.batch.status === 'cancelled') {
            stopRetryPolling();
          }
        }
      } catch {
        stopRetryPolling();
      }
    }, 2000);
  }

  function stopRetryPolling() {
    if (pollingTimer) { clearInterval(pollingTimer); pollingTimer = undefined; }
  }

  async function cancelRetryBatch() {
    if (!retryId) return;
    try {
      await deliveryClient.cancelRetry({ retryId });
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to cancel retry');
    }
  }

  function onBatchDone() {
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
          <div class="flex items-baseline justify-between mb-4">
            <h2 class="text-sm font-semibold text-gray-900 uppercase tracking-wide">Health Metrics</h2>
            <span class="text-[10px] text-gray-400 font-mono">Last 24 hours</span>
          </div>
          <div class="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-4">
            <div>
              <p class="text-xs text-gray-500">Success Rate</p>
              <p class="text-xl font-bold {successRateColor}">{successRatePercent}%</p>
            </div>
            <div>
              <p class="text-xs text-gray-500">Deliveries</p>
              <p class="text-xl font-bold text-gray-900">{healthMetrics.totalDeliveries}</p>
            </div>
            <div>
              <p class="text-xs text-gray-500">Succeeded</p>
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

          <!-- Error Category Breakdown -->
          {#if healthMetrics.failedDeliveries > 0}
            <div class="mt-4 pt-4 border-t border-gray-100">
              <h3 class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3">Error Breakdown</h3>
              <div class="grid grid-cols-2 sm:grid-cols-5 gap-3">
                <div class="bg-orange-50 rounded-lg px-3 py-2">
                  <p class="text-xs text-orange-600 font-medium">Client (4xx)</p>
                  <p class="text-lg font-bold text-orange-700">{healthMetrics.clientErrors}</p>
                  <p class="text-[10px] text-orange-500">Not retried</p>
                </div>
                <div class="bg-red-50 rounded-lg px-3 py-2">
                  <p class="text-xs text-red-600 font-medium">Server (5xx)</p>
                  <p class="text-lg font-bold text-red-700">{healthMetrics.serverErrors}</p>
                  <p class="text-[10px] text-red-500">Retried</p>
                </div>
                <div class="bg-yellow-50 rounded-lg px-3 py-2">
                  <p class="text-xs text-yellow-600 font-medium">Timeout</p>
                  <p class="text-lg font-bold text-yellow-700">{healthMetrics.timeoutErrors}</p>
                  <p class="text-[10px] text-yellow-500">Retried</p>
                </div>
                <div class="bg-purple-50 rounded-lg px-3 py-2">
                  <p class="text-xs text-purple-600 font-medium">Network</p>
                  <p class="text-lg font-bold text-purple-700">{healthMetrics.networkErrors}</p>
                  <p class="text-[10px] text-purple-500">DNS / TLS / Conn</p>
                </div>
                <div class="bg-amber-50 rounded-lg px-3 py-2">
                  <p class="text-xs text-amber-600 font-medium">Unexpected Status</p>
                  <p class="text-lg font-bold text-amber-700">{healthMetrics.unexpectedStatusErrors}</p>
                  <p class="text-[10px] text-amber-500">Not retried</p>
                </div>
              </div>

              <!-- Error distribution bar -->
              {#if (healthMetrics.clientErrors || 0) + (healthMetrics.serverErrors || 0) + (healthMetrics.timeoutErrors || 0) + (healthMetrics.networkErrors || 0) + (healthMetrics.unexpectedStatusErrors || 0) > 0}
                {@const totalErrors = (healthMetrics.clientErrors || 0) + (healthMetrics.serverErrors || 0) + (healthMetrics.timeoutErrors || 0) + (healthMetrics.networkErrors || 0) + (healthMetrics.unexpectedStatusErrors || 0)}
                <div class="mt-3 w-full h-2 rounded-full overflow-hidden flex">
                  {#if healthMetrics.clientErrors > 0}
                    <div class="bg-orange-400 h-full" style="width: {(healthMetrics.clientErrors / totalErrors) * 100}%" title="Client errors: {healthMetrics.clientErrors}"></div>
                  {/if}
                  {#if healthMetrics.serverErrors > 0}
                    <div class="bg-red-400 h-full" style="width: {(healthMetrics.serverErrors / totalErrors) * 100}%" title="Server errors: {healthMetrics.serverErrors}"></div>
                  {/if}
                  {#if healthMetrics.timeoutErrors > 0}
                    <div class="bg-yellow-400 h-full" style="width: {(healthMetrics.timeoutErrors / totalErrors) * 100}%" title="Timeouts: {healthMetrics.timeoutErrors}"></div>
                  {/if}
                  {#if healthMetrics.networkErrors > 0}
                    <div class="bg-purple-400 h-full" style="width: {(healthMetrics.networkErrors / totalErrors) * 100}%" title="Network errors: {healthMetrics.networkErrors}"></div>
                  {/if}
                  {#if healthMetrics.unexpectedStatusErrors > 0}
                    <div class="bg-amber-400 h-full" style="width: {(healthMetrics.unexpectedStatusErrors / totalErrors) * 100}%" title="Unexpected status: {healthMetrics.unexpectedStatusErrors}"></div>
                  {/if}
                </div>
              {/if}
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
            class="pb-3 text-sm font-medium border-b-2 transition {activeTab === 'subscriptions'
              ? 'border-gray-900 text-gray-900'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'}"
            onclick={() => (activeTab = 'subscriptions')}
          >
            Subscriptions
            <span class="ml-1 px-1.5 py-0.5 text-xs rounded-full bg-gray-100 text-gray-600">{subscriptions.length}</span>
          </button>
        </nav>
      </div>

      <!-- Deliveries Tab -->
      {#if activeTab === 'deliveries'}
        <!-- Delivery filters -->
        <div class="bg-white rounded-lg border border-gray-200 p-4 mb-4">
          <div class="flex flex-wrap items-end gap-3">
            <div class="w-full sm:w-32">
              <label for="del-status" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Status</label>
              <select
                id="del-status"
                bind:value={deliveryStatusFilter}
                onchange={applyDeliveryFilters}
                class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
              >
                <option value="">All</option>
                <option value="pending">Pending</option>
                <option value="sending">Sending</option>
                <option value="success">Success</option>
                <option value="failed">Failed</option>
                <option value="retrying">Retrying</option>
                <option value="expired">Expired</option>
              </select>
            </div>
            <div class="w-full sm:w-36">
              <label for="del-error" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Error Category</label>
              <select
                id="del-error"
                bind:value={deliveryErrorCategoryFilter}
                onchange={applyDeliveryFilters}
                class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
              >
                <option value="">All</option>
                {#each ERROR_CATEGORIES as cat}
                    <option value={cat.value}>{cat.label}</option>
                {/each}
              </select>
            </div>
            <div class="w-full sm:w-44">
              <label for="del-after" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Created After</label>
              <input
                id="del-after"
                type="datetime-local"
                bind:value={deliveryCreatedAfterFilter}
                onchange={applyDeliveryFilters}
                class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
              />
            </div>
            <div class="w-full sm:w-44">
              <label for="del-before" class="block text-[10px] font-medium text-gray-500 uppercase tracking-wider mb-1">Created Before</label>
              <input
                id="del-before"
                type="datetime-local"
                bind:value={deliveryCreatedBeforeFilter}
                onchange={applyDeliveryFilters}
                class="w-full px-3 py-1.5 text-sm border border-gray-300 rounded-lg bg-white focus:ring-2 focus:ring-gray-900 focus:border-gray-900"
              />
            </div>
            <div class="flex items-center gap-2">
              {#if hasDeliveryFilters}
                <button
                  onclick={clearDeliveryFilters}
                  class="px-3 py-1.5 text-xs font-medium text-gray-600 bg-gray-100 rounded-lg hover:bg-gray-200 transition"
                >
                  Clear
                </button>
              {/if}
              {#if totalCount > 0}
                <button
                  onclick={prepareRetryBatch}
                  disabled={preparingRetry}
                  class="px-3 py-1.5 text-xs font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 disabled:opacity-50 transition"
                >
                  {preparingRetry ? 'Preparing...' : 'Retry All Matching'}
                </button>
              {/if}
            </div>
          </div>
        </div>

        <!-- Batch progress -->
        {#if batchStatus}
          <div class="mb-4">
            <BatchProgress
              batch={batchStatus}
              label="Retry Deliveries"
              oncancel={cancelRetryBatch}
              ondone={onBatchDone}
            />
          </div>
        {/if}

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
                        <CopyableId id={delivery.deliveryId} href="/deliveries/{delivery.deliveryId}" truncate={12} />
                        <!-- Show event ID inline on mobile -->
                        <span class="block sm:hidden mt-0.5"><CopyableId id={delivery.eventId} truncate={12} /></span>
                      </td>
                      <td class="px-4 py-3 hidden sm:table-cell"><CopyableId id={delivery.eventId} truncate={16} /></td>
                      <td class="px-4 py-3">
                        <div class="flex items-center gap-1.5">
                          <StatusBadge status={delivery.status} />
                          {#if delivery.errorCategory && delivery.errorCategory !== '' && delivery.errorCategory !== 'success'}
                            {@const badge = getCategoryBadge(delivery.errorCategory)}
                            <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                          {/if}
                        </div>
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
                                  <p class="text-xs font-medium text-gray-500 mb-0.5">Error Category</p>
                                  {#if details.errorCategory && details.errorCategory !== '' && details.errorCategory !== 'success'}
                                    {@const badge = getCategoryBadge(details.errorCategory)}
                                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium border {badge.classes}">{badge.label}</span>
                                  {:else}
                                    <span class="text-sm text-gray-400">—</span>
                                  {/if}
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
          {#if editingConfig}
            <!-- Editable Configuration Form -->
            <div class="bg-white rounded-lg border border-gray-200 p-5">
              <div class="flex items-center justify-between mb-4">
                <h3 class="text-sm font-semibold text-gray-900 uppercase tracking-wide">Edit Configuration</h3>
              </div>

              <div class="space-y-5">
                <!-- Description -->
                <div>
                  <label for="config-description" class="block text-xs font-medium text-gray-700 mb-1">Description</label>
                  <input
                    id="config-description"
                    type="text"
                    bind:value={configForm.description}
                    class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900"
                    placeholder="Webhook description"
                  />
                </div>

                <!-- URL -->
                <div>
                  <label for="config-url" class="block text-xs font-medium text-gray-700 mb-1">URL</label>
                  <input
                    id="config-url"
                    type="url"
                    bind:value={configForm.url}
                    class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900 font-mono"
                    placeholder="https://example.com/webhook"
                  />
                </div>

                <!-- HTTP Config Fields -->
                <div class="border-t border-gray-100 pt-4">
                  <h4 class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3">HTTP Configuration</h4>
                  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                    <div>
                      <label for="config-max-retries" class="block text-xs font-medium text-gray-700 mb-1">Max Retries (0-10)</label>
                      <input id="config-max-retries" type="number" bind:value={configForm.maxRetries} min="0" max="10" class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                    </div>
                    <div>
                      <label for="config-retry-backoff" class="block text-xs font-medium text-gray-700 mb-1">Retry Backoff (seconds)</label>
                      <input id="config-retry-backoff" type="number" bind:value={configForm.retryBackoffSeconds} min="1" max="3600" class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                    </div>
                    <div>
                      <label for="config-timeout" class="block text-xs font-medium text-gray-700 mb-1">Request Timeout (seconds)</label>
                      <input id="config-timeout" type="number" bind:value={configForm.requestTimeoutSeconds} min="1" max="300" class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                    </div>
                    <div>
                      <label for="config-content-type" class="block text-xs font-medium text-gray-700 mb-1">Content Type</label>
                      <input id="config-content-type" type="text" bind:value={configForm.contentType} class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900 font-mono" />
                    </div>
                    <div>
                      <label for="config-user-agent" class="block text-xs font-medium text-gray-700 mb-1">User Agent</label>
                      <input id="config-user-agent" type="text" bind:value={configForm.userAgent} class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900 font-mono" />
                    </div>
                    <div>
                      <label for="config-status-codes" class="block text-xs font-medium text-gray-700 mb-1">Expected Status Codes</label>
                      <input id="config-status-codes" type="text" bind:value={configForm.expectedStatusCodes} placeholder="200, 201, 202, 204" class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900 font-mono" />
                      <p class="text-[10px] text-gray-400 mt-0.5">Comma-separated HTTP status codes</p>
                    </div>
                  </div>
                </div>

                <!-- Webhook Secret -->
                <div>
                  <label for="config-secret" class="block text-xs font-medium text-gray-700 mb-1">Webhook Secret</label>
                  <input id="config-secret" type="password" bind:value={configForm.webhookSecret} placeholder="Leave empty to keep existing" class="w-full text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900 font-mono" />
                  <p class="text-[10px] text-gray-400 mt-0.5">Used for HMAC signature verification</p>
                </div>

                <!-- Boolean toggles -->
                <div class="border-t border-gray-100 pt-4">
                  <h4 class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3">Options</h4>
                  <div class="space-y-3">
                    {#each [
                      { key: 'followRedirects', label: 'Follow Redirects', desc: 'Follow HTTP 3xx redirects' },
                      { key: 'verifySsl', label: 'Verify SSL', desc: 'Validate SSL/TLS certificates' },
                      { key: 'captureResponseBody', label: 'Capture Response Body', desc: 'Off: stores up to 1 KB of response per delivery. On: stores up to 1 MB.' },
                    ] as toggle}
                      <div class="flex items-center gap-3">
                        <button
                          type="button"
                          onclick={() => { (configForm as any)[toggle.key] = !(configForm as any)[toggle.key]; }}
                          aria-label="Toggle {toggle.label}"
                          class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out {(configForm as any)[toggle.key] ? 'bg-green-500' : 'bg-gray-300'}"
                        >
                          <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {(configForm as any)[toggle.key] ? 'translate-x-4' : 'translate-x-0'}"></span>
                        </button>
                        <div>
                          <span class="text-sm font-medium text-gray-700">{toggle.label}</span>
                          <p class="text-xs text-gray-500">{toggle.desc}</p>
                        </div>
                      </div>
                    {/each}
                  </div>
                </div>

                <!-- Custom Headers -->
                <div class="border-t border-gray-100 pt-4">
                  <h4 class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3">Custom Headers</h4>
                  {#if Object.keys(configForm.headers).length > 0}
                    <div class="space-y-1.5 mb-3">
                      {#each Object.entries(configForm.headers) as [key, value]}
                        <div class="flex items-center gap-2">
                          <span class="flex-1 text-xs font-mono bg-gray-50 px-2 py-1.5 rounded border border-gray-200 truncate">{key}</span>
                          <span class="flex-1 text-xs font-mono bg-gray-50 px-2 py-1.5 rounded border border-gray-200 truncate">{value}</span>
                          <button
                            onclick={() => removeConfigHeader(key)}
                            class="shrink-0 p-1 text-gray-400 hover:text-red-600 rounded transition"
                            aria-label="Remove header {key}"
                          >
                            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                            </svg>
                          </button>
                        </div>
                      {/each}
                    </div>
                  {/if}
                  <div class="flex items-center gap-2">
                    <input type="text" bind:value={configHeaderKey} placeholder="Header name" class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                    <input type="text" bind:value={configHeaderValue} placeholder="Header value" class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                    <button
                      onclick={addConfigHeader}
                      disabled={!configHeaderKey.trim() || !configHeaderValue.trim()}
                      class="shrink-0 px-3 py-1.5 text-sm font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 disabled:bg-gray-300 transition"
                    >
                      Add
                    </button>
                  </div>
                </div>

                <!-- Secret Headers -->
                <div class="border-t border-gray-100 pt-4">
                  <h4 class="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-1">Secret Headers</h4>
                  <p class="text-[10px] text-gray-400 mb-3">Encrypted headers for sensitive values (API keys, tokens). Existing values are preserved unless you remove or replace them.</p>
                  {#if existingSecretHeaderKeys.size > 0 || Object.keys(newSecretHeaders).length > 0}
                    <div class="space-y-1.5 mb-3">
                      {#each [...existingSecretHeaderKeys] as key}
                        <div class="flex items-center gap-2">
                          <span class="flex-1 text-xs font-mono bg-gray-50 px-2 py-1.5 rounded border border-gray-200 truncate">{key}</span>
                          <span class="flex-1 text-xs font-mono bg-gray-50 px-2 py-1.5 rounded border border-gray-200 truncate text-gray-400">••••••</span>
                          <button
                            onclick={() => removeConfigSecretHeader(key)}
                            class="shrink-0 p-1 text-gray-400 hover:text-red-600 rounded transition"
                            aria-label="Remove secret header {key}"
                          >
                            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                            </svg>
                          </button>
                        </div>
                      {/each}
                      {#each Object.entries(newSecretHeaders) as [key, value]}
                        <div class="flex items-center gap-2">
                          <span class="flex-1 text-xs font-mono bg-green-50 px-2 py-1.5 rounded border border-green-200 truncate">{key}</span>
                          <span class="flex-1 text-xs font-mono bg-green-50 px-2 py-1.5 rounded border border-green-200 truncate text-green-600">new value set</span>
                          <button
                            onclick={() => removeConfigSecretHeader(key)}
                            class="shrink-0 p-1 text-gray-400 hover:text-red-600 rounded transition"
                            aria-label="Remove secret header {key}"
                          >
                            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                            </svg>
                          </button>
                        </div>
                      {/each}
                    </div>
                  {/if}
                  <div class="flex items-center gap-2">
                    <input type="text" bind:value={configSecretHeaderKey} placeholder="Header name (e.g. Authorization)" class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                    <input type="password" bind:value={configSecretHeaderValue} placeholder="Header value (e.g. Bearer sk-...)" class="flex-1 text-sm rounded-lg border-gray-300 shadow-sm focus:border-gray-900 focus:ring-gray-900" />
                    <button
                      onclick={addConfigSecretHeader}
                      disabled={!configSecretHeaderKey.trim() || !configSecretHeaderValue.trim()}
                      class="shrink-0 px-3 py-1.5 text-sm font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 disabled:bg-gray-300 transition"
                    >
                      Add
                    </button>
                  </div>
                </div>

                <!-- Save / Cancel -->
                <div class="flex items-center justify-end gap-3 pt-4 border-t border-gray-100">
                  <button
                    onclick={cancelEditConfig}
                    disabled={savingConfig}
                    class="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition"
                  >
                    Cancel
                  </button>
                  <button
                    onclick={saveConfig}
                    disabled={savingConfig}
                    class="px-4 py-2 text-sm font-medium text-white bg-gray-900 rounded-lg hover:bg-gray-800 disabled:bg-gray-400 transition shadow-sm"
                  >
                    {savingConfig ? 'Saving...' : 'Save Configuration'}
                  </button>
                </div>
              </div>
            </div>
          {:else}
            <!-- Read-Only Configuration View -->
            <!-- HTTP Configuration -->
            <div class="bg-white rounded-lg border border-gray-200 p-5">
              <div class="flex items-center justify-between mb-4">
                <h3 class="text-sm font-semibold text-gray-900 uppercase tracking-wide">HTTP Configuration</h3>
                <button
                  onclick={startEditConfig}
                  class="px-3 py-1.5 text-xs font-medium text-gray-700 bg-gray-100 rounded-md hover:bg-gray-200 transition"
                >
                  Edit
                </button>
              </div>
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

            <!-- Secret Headers -->
            <div class="bg-white rounded-lg border border-gray-200 p-5">
              <h3 class="text-sm font-semibold text-gray-900 uppercase tracking-wide mb-2">Secret Headers</h3>
              <p class="text-xs text-gray-400 mb-4">Encrypted headers for sensitive values. Values are never exposed.</p>
              {#if Object.keys(webhook.secretHeaders || {}).length > 0}
                <div class="space-y-1.5">
                  {#each Object.entries(webhook.secretHeaders) as [key, value]}
                    <div class="flex items-center gap-2 text-sm">
                      <span class="font-mono text-gray-900 font-medium">{key}:</span>
                      <span class="font-mono text-gray-400">••••••</span>
                    </div>
                  {/each}
                </div>
              {:else}
                <p class="text-sm text-gray-500">No secret headers configured.</p>
              {/if}
            </div>
          {/if}
        </div>
      {/if}

      <!-- Subscriptions Tab -->
      {#if activeTab === 'subscriptions'}
        <SubscriptionManager
          {webhookId}
          namespace={webhook.namespace}
          bind:subscriptions
          onRefresh={fetchData}
        />
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

<!-- Confirm Retry Dialog -->
<ConfirmDialog
  open={confirmRetry}
  title="Retry Deliveries"
  message="This will retry {retryTotal} matching deliver{retryTotal !== 1 ? 'ies' : 'y'} for this webhook. Continue?"
  confirmLabel="Retry"
  variant="warning"
  onconfirm={executeRetry}
  oncancel={() => { confirmRetry = false; retryId = ''; }}
/>
