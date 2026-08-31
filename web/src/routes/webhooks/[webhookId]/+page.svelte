<script lang="ts">
  import favicon from '$lib/assets/favicon.svg';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { api, unwrap } from '$lib/services';
  import { getCategoryBadge, ERROR_CATEGORIES, formatAPIError } from '$lib/utils';
  import { onMount, onDestroy } from 'svelte';
  import type { components } from '$lib/api-types';
  import HealthBadge from '$lib/components/HealthBadge.svelte';
  import CopyableId from '$lib/components/CopyableId.svelte';
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import Pagination from '$lib/components/Pagination.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import ConfirmDialog from '$lib/components/ConfirmDialog.svelte';
  import SubscriptionManager from '$lib/components/SubscriptionManager.svelte';
  import BatchProgress from '$lib/components/BatchProgress.svelte';

  type WebhookOut = components["schemas"]["WebhookOut"];
  type DeliveryItem = components["schemas"]["DeliveryItem"];
  type WebhookHealthOutput = components["schemas"]["WebhookHealthOutputBody"];
  type SubscriptionItem = components["schemas"]["SubscriptionItem"];

  let webhook: WebhookOut | undefined = $state();
  let deliveries: DeliveryItem[] = $state([]);
  let healthMetrics: WebhookHealthOutput | undefined = $state();
  let subscriptions: SubscriptionItem[] = $state([]);
  let loading = $state(true);
  let error = $state('');
  let expandedDeliveries: Set<string> = $state(new Set());
  let deliveryDetails: Map<string, DeliveryItem> = $state(new Map());
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
  let existingSecretHeaderKeys = $state<Set<string>>(new Set());
  let newSecretHeaders = $state<Record<string, string>>({});

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

  function formatPayload(payload: string | undefined): string {
    if (!payload) return 'N/A';
    try { return JSON.stringify(JSON.parse(payload), null, 2); }
    catch { return payload; }
  }

  function formatTimestamp(timestamp: string | null | undefined): string {
    if (!timestamp) return 'N/A';
    const d = new Date(timestamp);
    return isNaN(d.getTime()) ? 'N/A' : d.toLocaleString();
  }

  async function fetchData() {
    if (!webhookId) return;
    try {
      // Look up by id across all namespaces (namespace not yet known).
      const webhookRes = unwrap(await api.GET('/v1/webhooks', { params: { query: { webhook_id: webhookId } } }));
      webhook = webhookRes.items?.[0];

      if (!webhook) {
        error = 'Webhook not found';
        return;
      }

      const ns = webhook.namespace;

      const [deliveriesRes, healthRes, subscriptionsRes] = await Promise.all([
        api.GET('/v1/namespaces/{namespace}/deliveries', {
          params: {
            path: { namespace: ns },
            query: {
              webhook_id: webhookId,
              status: deliveryStatusFilter || undefined,
              limit,
              offset,
            },
          },
        }),
        api.GET('/v1/namespaces/{namespace}/webhooks/{webhook_id}/health', {
          params: { path: { namespace: ns, webhook_id: webhookId } },
        }),
        api.GET('/v1/namespaces/{namespace}/subscriptions', {
          params: { path: { namespace: ns }, query: { webhook_id: webhookId } },
        }),
      ]);

      deliveries = unwrap(deliveriesRes).items || [];
      totalCount = unwrap(deliveriesRes).pagination?.total_count || 0;
      healthMetrics = unwrap(healthRes);
      subscriptions = unwrap(subscriptionsRes).items || [];
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
        unwrap(await api.POST('/v1/namespaces/{namespace}/webhooks/{webhook_id}:pause', {
          params: { path: { namespace: webhook.namespace, webhook_id: webhookId } },
        }));
      } else {
        unwrap(await api.POST('/v1/namespaces/{namespace}/webhooks/{webhook_id}:resume', {
          params: { path: { namespace: webhook.namespace, webhook_id: webhookId } },
        }));
      }
      await fetchData();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to update status');
    }
  }

  async function executeUnregister() {
    if (!webhook) return;
    try {
      unwrap(await api.DELETE('/v1/namespaces/{namespace}/webhooks/{webhook_id}', {
        params: { path: { namespace: webhook.namespace, webhook_id: webhookId } },
      }));
      confirmUnregister = false;
      goto('/webhooks');
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to unregister webhook');
      confirmUnregister = false;
    }
  }

  async function resendDelivery(deliveryId: string) {
    try {
      unwrap(await api.POST('/v1/deliveries/{delivery_id}:retry', {
        params: { path: { delivery_id: deliveryId } },
      }));
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
      unwrap(await api.PATCH('/v1/namespaces/{namespace}/webhooks/{webhook_id}', {
        params: { path: { namespace: webhook.namespace, webhook_id: webhookId } },
        body: { url: trimmedUrl, active: webhook.active },
      }));
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
      maxRetries: webhook.http_config?.max_retries ?? 3,
      retryBackoffSeconds: webhook.http_config?.retry_backoff_seconds ?? 60,
      requestTimeoutSeconds: webhook.http_config?.request_timeout_seconds ?? 30,
      captureResponseBody: webhook.http_config?.capture_response_body ?? false,
      followRedirects: webhook.http_config?.follow_redirects ?? true,
      verifySsl: webhook.http_config?.verify_ssl ?? true,
      expectedStatusCodes: (webhook.http_config?.expected_status_codes || [200, 201, 202, 204]).join(', '),
      webhookSecret: '',
      userAgent: webhook.http_config?.user_agent || 'Sparrow-Webhook/1.0',
      contentType: webhook.http_config?.content_type || 'application/json',
      headers: { ...(webhook.headers || {}) },
    };
    existingSecretHeaderKeys = new Set(Object.keys(webhook.secret_headers || {}));
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
      existingSecretHeaderKeys.delete(key);
      existingSecretHeaderKeys = new Set(existingSecretHeaderKeys);
      newSecretHeaders = { ...newSecretHeaders, [key]: configSecretHeaderValue.trim() };
      configSecretHeaderKey = '';
      configSecretHeaderValue = '';
    }
  }

  function removeConfigSecretHeader(key: string) {
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
      const statusCodes = configForm.expectedStatusCodes
        .split(',')
        .map((s: string) => parseInt(s.trim(), 10))
        .filter((n: number) => !isNaN(n) && n >= 100 && n <= 599);

      if (statusCodes.length === 0) {
        error = 'At least one valid expected status code is required';
        savingConfig = false;
        return;
      }

      const trimmedUrl = configForm.url.trim();
      if (!trimmedUrl) { error = 'URL is required'; savingConfig = false; return; }
      try { new URL(trimmedUrl); } catch { error = 'Enter a valid URL'; savingConfig = false; return; }

      const hasNewSecretHeaders = Object.keys(newSecretHeaders).length > 0;

      unwrap(await api.PATCH('/v1/namespaces/{namespace}/webhooks/{webhook_id}', {
        params: { path: { namespace: webhook.namespace, webhook_id: webhookId } },
        body: {
          url: trimmedUrl,
          active: configForm.active,
          description: configForm.description,
          headers: configForm.headers,
          ...(hasNewSecretHeaders ? { secret_headers: newSecretHeaders } : {}),
          http_config: {
            max_retries: configForm.maxRetries,
            retry_backoff_seconds: configForm.retryBackoffSeconds,
            request_timeout_seconds: configForm.requestTimeoutSeconds,
            capture_response_body: configForm.captureResponseBody,
            follow_redirects: configForm.followRedirects,
            verify_ssl: configForm.verifySsl,
            expected_status_codes: statusCodes,
            ...(configForm.webhookSecret ? { webhook_secret: configForm.webhookSecret } : {}),
            user_agent: configForm.userAgent,
            content_type: configForm.contentType,
          },
        },
      }));
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
          const detail = unwrap(await api.GET('/v1/deliveries/{delivery_id}', {
            params: { path: { delivery_id: deliveryId } },
          }));
          deliveryDetails.set(deliveryId, detail);
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

  async function prepareRetryBatch() {
    if (!webhook) return;
    preparingRetry = true;
    try {
      const res = unwrap(await api.GET('/v1/namespaces/{namespace}/deliveries', {
        params: {
          path: { namespace: webhook.namespace },
          query: {
            webhook_id: webhookId,
            status: deliveryStatusFilter || undefined,
            prepare_retry: true,
            limit: 1,
            offset: 0,
          },
        },
      }));
      if (res.retry_id) {
        retryId = res.retry_id;
        retryTotal = res.pagination?.total_count || 0;
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
    if (!retryId || !webhook) return;
    try {
      const res = unwrap(await api.POST('/v1/namespaces/{namespace}/deliveries:retryBatch', {
        params: { path: { namespace: webhook.namespace } },
        body: { repush_id: retryId },
      }));
      batchStatus = { status: res.status, total: res.total, processed: res.processed, failed: res.failed };
      startRetryPolling();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to start retry');
    }
  }

  function startRetryPolling() {
    if (pollingTimer) clearInterval(pollingTimer);
    pollingTimer = setInterval(async () => {
      if (!retryId || !webhook) { stopRetryPolling(); return; }
      try {
        const res = unwrap(await api.GET('/v1/namespaces/{namespace}/retry-jobs/{job_id}', {
          params: { path: { namespace: webhook.namespace, job_id: retryId } },
        }));
        batchStatus = { status: res.status, total: res.total, processed: res.processed, failed: res.failed };
        if (res.status === 'completed' || res.status === 'failed' || res.status === 'cancelled') {
          stopRetryPolling();
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
    if (!retryId || !webhook) return;
    try {
      await api.POST('/v1/namespaces/{namespace}/retry-jobs/{job_id}:cancel', {
        params: { path: { namespace: webhook.namespace, job_id: retryId } },
      });
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to cancel retry');
    }
  }

  function onBatchDone() {
    fetchData();
  }

  let successRatePercent = $derived(healthMetrics ? (healthMetrics.success_rate * 100).toFixed(1) : '0');
  let successRateColor = $derived.by(() => {
    if (!healthMetrics) return 'text-faint';
    if (healthMetrics.success_rate >= 0.95) return 'text-ok';
    if (healthMetrics.success_rate >= 0.8) return 'text-warn';
    return 'text-bad';
  });
</script>

<svelte:head>
  <title>{webhook?.description || 'Webhook'} - {webhookId} | Sparrow</title>
</svelte:head>

<main class="mx-auto max-w-6xl px-4 sm:px-8 py-8">
    {#if loading}
      <nav class="mb-6">
        <div class="h-4 bg-white/5 rounded w-28 animate-pulse"></div>
      </nav>
      <div class="panel p-5 mb-6 animate-pulse">
        <div class="h-6 bg-white/5 rounded w-48 mb-3"></div>
        <div class="h-4 bg-white/[0.03] rounded w-64 mb-3"></div>
        <div class="flex gap-6">
          <div class="h-4 bg-white/[0.03] rounded w-32"></div>
          <div class="h-4 bg-white/[0.03] rounded w-40"></div>
        </div>
      </div>
      <div class="panel p-5 mb-6 animate-pulse">
        <div class="h-4 bg-white/5 rounded w-32 mb-4"></div>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
          {#each Array(4) as _}
            <div><div class="h-3 bg-white/[0.03] rounded w-16 mb-1"></div><div class="h-6 bg-white/5 rounded w-12"></div></div>
          {/each}
        </div>
      </div>
    {:else if error && !webhook}
      <div class="panel p-4" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
        <p class="text-sm" style="color:var(--color-bad)">{error}</p>
        <a href="/webhooks" class="link-beacon text-sm underline mt-2 inline-block">Back to webhooks</a>
      </div>
    {:else if webhook}
      <nav class="flex items-center gap-2 text-sm text-muted mb-6">
        <a class="link" href="/webhooks">Webhooks</a>
        <span class="text-faint">/</span>
        <span class="text-text truncate max-w-xs">{webhook.description || 'Webhook'}</span>
      </nav>

      {#if error}
        <div class="panel p-3 mb-4 flex items-start justify-between" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
          <p class="text-sm" style="color:var(--color-bad)">{error}</p>
          <button onclick={() => { error = ''; }} class="ml-3 shrink-0 text-faint hover:text-bad transition-colors" aria-label="Dismiss error">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
      {/if}

      <div class="panel ticked p-5 mb-6">
        <div class="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
          <div class="flex-1 min-w-0">
            <p class="eyebrow mb-1.5">Fleet / Webhook</p>
            <div class="flex items-center gap-3 mb-2 flex-wrap">
              <h1 class="text-2xl truncate">
                {webhook.description || 'Webhook'}
              </h1>
              <HealthBadge health={webhook.health} size="md" />
              {#if webhook.active}
                <span class="chip" style="color:var(--color-ok);border-color:color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 12%,var(--color-panel-2))">
                  <span class="w-1.5 h-1.5 rounded-full" style="background:var(--color-ok)"></span>
                  Active
                </span>
              {:else}
                <span class="chip">
                  <span class="w-1.5 h-1.5 rounded-full" style="background:var(--color-idle)"></span>
                  Paused
                </span>
              {/if}
            </div>

            {#if editingUrl}
              <div class="flex items-center gap-2 mb-3">
                <input
                  type="url"
                  bind:value={editedUrl}
                  class="input flex-1"
                  placeholder="https://example.com/webhook"
                />
                <button onclick={saveWebhookUrl} disabled={savingUrl} class="btn btn-beacon !px-3 !py-1.5">
                  {savingUrl ? 'Saving…' : 'Save'}
                </button>
                <button onclick={cancelEditUrl} disabled={savingUrl} class="btn btn-ghost !px-3 !py-1.5">
                  Cancel
                </button>
              </div>
            {:else}
              <button onclick={startEditUrl} class="group flex items-center gap-1.5 mb-3 text-left" title="Click to edit URL" aria-label="Edit webhook URL">
                <span class="text-sm mono text-muted break-all">{webhook.url}</span>
                <svg class="w-3.5 h-3.5 text-faint opacity-0 group-hover:opacity-100 transition shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15.232 5.232l3.536 3.536m-2.036-5.036a2.5 2.5 0 113.536 3.536L6.5 21.036H3v-3.572L16.732 3.732z" />
                </svg>
              </button>
            {/if}

            <div class="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm text-muted">
              <span>Namespace: <span class="chip">{webhook.namespace}</span></span>
              <span>Created: <span class="mono tnum text-text">{formatTimestamp(webhook.created_at)}</span></span>
              <span class="mono text-xs text-faint">ID: {webhookId}</span>
            </div>
          </div>

          <div class="flex items-center gap-2 shrink-0">
            <button
              onclick={toggleWebhookStatus}
              class="btn btn-ghost !px-3 !py-1.5"
            >
              {webhook.active ? 'Pause' : 'Resume'}
            </button>
            <button
              onclick={() => { confirmUnregister = true; }}
              class="btn btn-danger !px-3 !py-1.5"
            >
              Unregister
            </button>
          </div>
        </div>

        <div class="mt-4 pt-4 border-t border-line">
          <p class="eyebrow mb-2">Subscribed Events</p>
          <div class="flex flex-wrap gap-1.5">
            {#each webhook.events ?? [] as ev}
              <span class="chip">{ev}</span>
            {/each}
            {#if (webhook.events ?? []).length === 0}
              <span class="text-xs text-faint">No events subscribed</span>
            {/if}
          </div>
        </div>
      </div>

      {#if healthMetrics}
        <div class="panel p-5 mb-6">
          <div class="flex items-baseline justify-between mb-4">
            <h2 class="eyebrow">Health Metrics</h2>
            <span class="text-[10px] text-faint mono">Last 24 hours</span>
          </div>
          <div class="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-3">
            <div class="panel-2 p-3">
              <p class="text-xs text-muted mb-0.5">Success Rate</p>
              <p class="text-xl font-semibold tnum {successRateColor}">{successRatePercent}%</p>
            </div>
            <div class="panel-2 p-3">
              <p class="text-xs text-muted mb-0.5">Deliveries</p>
              <p class="text-xl font-semibold tnum text-text">{healthMetrics.total_deliveries}</p>
            </div>
            <div class="panel-2 p-3">
              <p class="text-xs text-muted mb-0.5">Succeeded</p>
              <p class="text-xl font-semibold tnum" style="color:var(--color-ok)">{healthMetrics.successful_deliveries}</p>
            </div>
            <div class="panel-2 p-3">
              <p class="text-xs text-muted mb-0.5">Failed</p>
              <p class="text-xl font-semibold tnum" style="color:var(--color-bad)">{healthMetrics.failed_deliveries}</p>
            </div>
            <div class="panel-2 p-3">
              <p class="text-xs text-muted mb-0.5">Avg Response</p>
              <p class="text-xl font-semibold tnum text-text">{healthMetrics.avg_response_time}<span class="text-xs font-normal text-faint">ms</span></p>
            </div>
            <div class="panel-2 p-3">
              <p class="text-xs text-muted mb-0.5">Consecutive Failures</p>
              <p class="text-xl font-semibold tnum text-text">{healthMetrics.consecutive_failures}</p>
            </div>
          </div>

          {#if healthMetrics.total_deliveries > 0}
            <div class="mt-4">
              <div class="w-full bg-panel-2 border border-line rounded-full h-2 overflow-hidden">
                <div
                  class="h-full rounded-full transition-all duration-500 {healthMetrics.success_rate >= 0.95 ? 'bg-ok' : healthMetrics.success_rate >= 0.8 ? 'bg-warn' : 'bg-bad'}"
                  style="width: {healthMetrics.success_rate * 100}%"
                ></div>
              </div>
            </div>
          {/if}

          {#if healthMetrics.failed_deliveries > 0}
            <div class="mt-4 pt-4 border-t border-line">
              <h3 class="eyebrow mb-3">Error Breakdown</h3>
              <div class="grid grid-cols-2 sm:grid-cols-5 gap-3">
                <div class="panel-2 px-3 py-2" style="border-color:color-mix(in srgb,var(--color-warn) 30%,transparent)">
                  <p class="text-xs font-medium" style="color:var(--color-warn)">Client (4xx)</p>
                  <p class="text-lg font-semibold tnum text-text">{healthMetrics.client_errors}</p>
                  <p class="text-[10px] text-faint">Not retried</p>
                </div>
                <div class="panel-2 px-3 py-2" style="border-color:color-mix(in srgb,var(--color-bad) 30%,transparent)">
                  <p class="text-xs font-medium" style="color:var(--color-bad)">Server (5xx)</p>
                  <p class="text-lg font-semibold tnum text-text">{healthMetrics.server_errors}</p>
                  <p class="text-[10px] text-faint">Retried</p>
                </div>
                <div class="panel-2 px-3 py-2" style="border-color:color-mix(in srgb,var(--color-warn) 30%,transparent)">
                  <p class="text-xs font-medium" style="color:var(--color-warn)">Timeout</p>
                  <p class="text-lg font-semibold tnum text-text">{healthMetrics.timeout_errors}</p>
                  <p class="text-[10px] text-faint">Retried</p>
                </div>
                <div class="panel-2 px-3 py-2" style="border-color:color-mix(in srgb,var(--color-idle) 30%,transparent)">
                  <p class="text-xs font-medium" style="color:var(--color-idle)">Network</p>
                  <p class="text-lg font-semibold tnum text-text">{healthMetrics.network_errors}</p>
                  <p class="text-[10px] text-faint">DNS / TLS / Conn</p>
                </div>
                <div class="panel-2 px-3 py-2" style="border-color:color-mix(in srgb,var(--color-warn) 30%,transparent)">
                  <p class="text-xs font-medium" style="color:var(--color-warn)">Unexpected Status</p>
                  <p class="text-lg font-semibold tnum text-text">{healthMetrics.unexpected_status_errors}</p>
                  <p class="text-[10px] text-faint">Not retried</p>
                </div>
              </div>

              {#if (healthMetrics.client_errors || 0) + (healthMetrics.server_errors || 0) + (healthMetrics.timeout_errors || 0) + (healthMetrics.network_errors || 0) + (healthMetrics.unexpected_status_errors || 0) > 0}
                {@const totalErrors = (healthMetrics.client_errors || 0) + (healthMetrics.server_errors || 0) + (healthMetrics.timeout_errors || 0) + (healthMetrics.network_errors || 0) + (healthMetrics.unexpected_status_errors || 0)}
                <div class="mt-3 w-full h-2 rounded-full overflow-hidden flex bg-panel-2 border border-line">
                  {#if healthMetrics.client_errors > 0}
                    <div class="h-full" style="width: {(healthMetrics.client_errors / totalErrors) * 100}%;background:var(--color-warn)" title="Client errors: {healthMetrics.client_errors}"></div>
                  {/if}
                  {#if healthMetrics.server_errors > 0}
                    <div class="h-full" style="width: {(healthMetrics.server_errors / totalErrors) * 100}%;background:var(--color-bad)" title="Server errors: {healthMetrics.server_errors}"></div>
                  {/if}
                  {#if healthMetrics.timeout_errors > 0}
                    <div class="h-full" style="width: {(healthMetrics.timeout_errors / totalErrors) * 100}%;background:color-mix(in srgb,var(--color-warn) 70%,var(--color-bad))" title="Timeouts: {healthMetrics.timeout_errors}"></div>
                  {/if}
                  {#if healthMetrics.network_errors > 0}
                    <div class="h-full" style="width: {(healthMetrics.network_errors / totalErrors) * 100}%;background:var(--color-idle)" title="Network errors: {healthMetrics.network_errors}"></div>
                  {/if}
                  {#if healthMetrics.unexpected_status_errors > 0}
                    <div class="h-full" style="width: {(healthMetrics.unexpected_status_errors / totalErrors) * 100}%;background:color-mix(in srgb,var(--color-warn) 55%,var(--color-idle))" title="Unexpected status: {healthMetrics.unexpected_status_errors}"></div>
                  {/if}
                </div>
              {/if}
            </div>
          {/if}
        </div>
      {/if}

      <div class="border-b border-line mb-6">
        <nav class="flex gap-6">
          <button
            class="pb-3 text-sm font-medium border-b-2 -mb-px transition {activeTab === 'deliveries' ? 'border-beacon text-text' : 'border-transparent text-muted hover:text-text hover:border-line-strong'}"
            onclick={() => (activeTab = 'deliveries')}
          >
            Deliveries
            <span class="ml-1 chip tnum">{totalCount}</span>
          </button>
          <button
            class="pb-3 text-sm font-medium border-b-2 -mb-px transition {activeTab === 'config' ? 'border-beacon text-text' : 'border-transparent text-muted hover:text-text hover:border-line-strong'}"
            onclick={() => (activeTab = 'config')}
          >
            Configuration
          </button>
          <button
            class="pb-3 text-sm font-medium border-b-2 -mb-px transition {activeTab === 'subscriptions' ? 'border-beacon text-text' : 'border-transparent text-muted hover:text-text hover:border-line-strong'}"
            onclick={() => (activeTab = 'subscriptions')}
          >
            Subscriptions
            <span class="ml-1 chip tnum">{subscriptions.length}</span>
          </button>
        </nav>
      </div>

      {#if activeTab === 'deliveries'}
        <div class="panel p-4 mb-4">
          <div class="flex flex-wrap items-end gap-3">
            <div class="w-full sm:w-32">
              <label for="del-status" class="field-label">Status</label>
              <select id="del-status" bind:value={deliveryStatusFilter} onchange={applyDeliveryFilters} class="select">
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
              <label for="del-error" class="field-label">Error Category</label>
              <select id="del-error" bind:value={deliveryErrorCategoryFilter} onchange={applyDeliveryFilters} class="select">
                <option value="">All</option>
                {#each ERROR_CATEGORIES as cat}
                    <option value={cat.value}>{cat.label}</option>
                {/each}
              </select>
            </div>
            <div class="flex items-center gap-2">
              {#if hasDeliveryFilters}
                <button onclick={clearDeliveryFilters} class="btn btn-ghost !px-3 !py-1.5">Clear</button>
              {/if}
              {#if totalCount > 0}
                <button onclick={prepareRetryBatch} disabled={preparingRetry} class="btn btn-beacon !px-3 !py-1.5">
                  {preparingRetry ? 'Preparing…' : 'Retry All Matching'}
                </button>
              {/if}
            </div>
          </div>
        </div>

        {#if batchStatus}
          <div class="mb-4">
            <BatchProgress
              batch={batchStatus}
              oncancel={cancelRetryBatch}
              ondone={onBatchDone}
            />
          </div>
        {/if}

        <div class="panel overflow-hidden">
          {#if deliveries.length === 0}
            <EmptyState icon="send" title="No deliveries yet" description="Deliveries will appear here when events are pushed to this webhook." />
          {:else}
            <div class="overflow-x-auto">
              <table class="w-full text-left">
                <thead>
                  <tr class="border-b border-line">
                    <th class="th">Delivery ID</th>
                    <th class="th hidden sm:table-cell">Event ID</th>
                    <th class="th">Status</th>
                    <th class="th hidden md:table-cell">Attempts</th>
                    <th class="th hidden lg:table-cell">Last Attempt</th>
                    <th class="th"></th>
                  </tr>
                </thead>
                <tbody>
                  {#each deliveries as delivery}
                    <tr class="row-line row-hover transition">
                      <td class="td">
                        <CopyableId id={delivery.delivery_id} href="/deliveries/{delivery.delivery_id}" truncate={12} />
                        <span class="block sm:hidden mt-0.5"><CopyableId id={delivery.event_id} href="/events/instances/{delivery.event_id}" truncate={12} /></span>
                      </td>
                      <td class="td hidden sm:table-cell"><CopyableId id={delivery.event_id} href="/events/instances/{delivery.event_id}" truncate={16} /></td>
                      <td class="td">
                        <div class="flex items-center gap-1.5">
                          <StatusBadge status={delivery.status} />
                          {#if delivery.error_category && delivery.error_category !== 'success'}
                            {@const badge = getCategoryBadge(delivery.error_category)}
                            <span class="inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium border {badge.classes}">{badge.label}</span>
                          {/if}
                        </div>
                      </td>
                      <td class="td tnum text-muted hidden md:table-cell">{delivery.attempt_count}</td>
                      <td class="td hidden lg:table-cell"><span class="mono tnum text-muted text-xs">{formatTimestamp(delivery.last_attempted_at)}</span></td>
                      <td class="td">
                        <button onclick={() => toggleDeliveryExpansion(delivery.delivery_id)} class="text-xs mono text-muted hover:text-text transition">
                          {expandedDeliveries.has(delivery.delivery_id) ? 'Hide' : 'Details'}
                        </button>
                      </td>
                    </tr>
                    {#if expandedDeliveries.has(delivery.delivery_id)}
                      <tr class="row-line">
                        <td colspan="6" class="px-4 py-4 bg-panel-2">
                          {#if deliveryDetails.has(delivery.delivery_id)}
                            {@const details = deliveryDetails.get(delivery.delivery_id)!}
                            <div class="space-y-3">
                              <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-3 text-sm">
                                <div>
                                  <p class="eyebrow mb-0.5">Response Code</p>
                                  <span class="mono text-sm tnum" style="color:{(details.response_code ?? 0) >= 200 && (details.response_code ?? 0) < 300 ? 'var(--color-ok)' : (details.response_code ?? 0) >= 400 ? 'var(--color-bad)' : 'var(--color-text)'}">
                                    {details.response_code || 'N/A'}
                                  </span>
                                </div>
                                <div>
                                  <p class="eyebrow mb-0.5">Error Category</p>
                                  {#if details.error_category && details.error_category !== 'success'}
                                    {@const badge = getCategoryBadge(details.error_category)}
                                    <span class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium border {badge.classes}">{badge.label}</span>
                                  {:else}
                                    <span class="text-sm text-faint">—</span>
                                  {/if}
                                </div>
                                <div>
                                  <p class="eyebrow mb-0.5">Attempts</p>
                                  <span class="text-sm text-text tnum">{delivery.attempt_count}</span>
                                </div>
                                <div>
                                  <p class="eyebrow mb-0.5">Created</p>
                                  <span class="text-sm text-text mono tnum">{details.created_at ? formatTimestamp(details.created_at) : 'N/A'}</span>
                                </div>
                                <div>
                                  <p class="eyebrow mb-0.5">Last Attempt</p>
                                  <span class="text-sm text-text mono tnum">{formatTimestamp(delivery.last_attempted_at)}</span>
                                </div>
                              </div>

                              {#if details.response_body}
                                <div>
                                  <p class="eyebrow mb-1">Response Body</p>
                                  <pre class="panel p-3 text-xs overflow-auto max-h-32 mono text-text">{formatPayload(details.response_body)}</pre>
                                </div>
                              {/if}

                              {#if details.error_message}
                                <div>
                                  <p class="eyebrow mb-1">Error</p>
                                  <div class="panel p-3" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
                                    <p class="text-sm" style="color:var(--color-bad)">{details.error_message}</p>
                                  </div>
                                </div>
                              {/if}

                              <div class="flex items-center gap-2 pt-1">
                                {#if delivery.status === 'failed' || details.error_message}
                                  <button onclick={() => resendDelivery(delivery.delivery_id)} class="btn btn-beacon !px-3 !py-1.5">
                                    Retry Delivery
                                  </button>
                                {/if}
                                <a href="/deliveries/{delivery.delivery_id}" class="btn btn-ghost !px-3 !py-1.5">
                                  Full Details
                                </a>
                              </div>
                            </div>
                          {:else}
                            <div class="flex items-center justify-center py-4">
                              <img src={favicon} alt="" aria-hidden="true" class="w-4 h-4 animate-spin mr-2" />
                              <span class="text-sm text-muted">Loading details…</span>
                            </div>
                          {/if}
                        </td>
                      </tr>
                    {/if}
                  {/each}
                </tbody>
              </table>
            </div>

            <div class="border-t border-line px-4">
              <Pagination {currentPage} {totalPages} {totalCount} pageSize={limit} onPageChange={handlePageChange} />
            </div>
          {/if}
        </div>
      {/if}

      {#if activeTab === 'config'}
        <div class="space-y-4">
          {#if editingConfig}
            <div class="panel p-5">
              <div class="flex items-center justify-between mb-4">
                <h3 class="eyebrow">Edit Configuration</h3>
              </div>

              <div class="space-y-5">
                <div>
                  <label for="config-description" class="field-label">Description</label>
                  <input id="config-description" type="text" bind:value={configForm.description} class="input" placeholder="Webhook description" />
                </div>

                <div>
                  <label for="config-url" class="field-label">URL</label>
                  <input id="config-url" type="url" bind:value={configForm.url} class="input" placeholder="https://example.com/webhook" />
                </div>

                <div class="border-t border-line pt-4">
                  <h4 class="eyebrow mb-3">HTTP Configuration</h4>
                  <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                    <div>
                      <label for="config-max-retries" class="field-label">Max Retries (0-10)</label>
                      <input id="config-max-retries" type="number" bind:value={configForm.maxRetries} min="0" max="10" class="input" />
                    </div>
                    <div>
                      <label for="config-retry-backoff" class="field-label">Retry Backoff (seconds)</label>
                      <input id="config-retry-backoff" type="number" bind:value={configForm.retryBackoffSeconds} min="1" max="3600" class="input" />
                    </div>
                    <div>
                      <label for="config-timeout" class="field-label">Request Timeout (seconds)</label>
                      <input id="config-timeout" type="number" bind:value={configForm.requestTimeoutSeconds} min="1" max="300" class="input" />
                    </div>
                    <div>
                      <label for="config-content-type" class="field-label">Content Type</label>
                      <input id="config-content-type" type="text" bind:value={configForm.contentType} class="input" />
                    </div>
                    <div>
                      <label for="config-user-agent" class="field-label">User Agent</label>
                      <input id="config-user-agent" type="text" bind:value={configForm.userAgent} class="input" />
                    </div>
                    <div>
                      <label for="config-status-codes" class="field-label">Expected Status Codes</label>
                      <input id="config-status-codes" type="text" bind:value={configForm.expectedStatusCodes} placeholder="200, 201, 202, 204" class="input" />
                      <p class="text-[10px] text-faint mt-0.5">Comma-separated HTTP status codes</p>
                    </div>
                  </div>
                </div>

                <div>
                  <label for="config-secret" class="field-label">Webhook Secret</label>
                  <input id="config-secret" type="password" bind:value={configForm.webhookSecret} placeholder="Leave empty to keep existing…" class="input" />
                  <p class="text-[10px] text-faint mt-0.5">Used for HMAC signature verification</p>
                </div>

                <div class="border-t border-line pt-4">
                  <h4 class="eyebrow mb-3">Options</h4>
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
                          class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out {(configForm as any)[toggle.key] ? 'bg-ok' : 'bg-line-strong'}"
                        >
                          <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out {(configForm as any)[toggle.key] ? 'translate-x-4' : 'translate-x-0'}"></span>
                        </button>
                        <div>
                          <span class="text-sm font-medium text-text">{toggle.label}</span>
                          <p class="text-xs text-muted">{toggle.desc}</p>
                        </div>
                      </div>
                    {/each}
                  </div>
                </div>

                <div class="border-t border-line pt-4">
                  <h4 class="eyebrow mb-3">Custom Headers</h4>
                  {#if Object.keys(configForm.headers).length > 0}
                    <div class="space-y-1.5 mb-3">
                      {#each Object.entries(configForm.headers) as [key, value]}
                        <div class="flex items-center gap-2">
                          <span class="flex-1 text-xs mono panel-2 px-2 py-1.5 rounded truncate text-text">{key}</span>
                          <span class="flex-1 text-xs mono panel-2 px-2 py-1.5 rounded truncate text-text">{value}</span>
                          <button onclick={() => removeConfigHeader(key)} class="shrink-0 p-1 text-faint hover:text-bad rounded transition" aria-label="Remove header {key}">
                            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                            </svg>
                          </button>
                        </div>
                      {/each}
                    </div>
                  {/if}
                  <div class="flex items-center gap-2">
                    <input type="text" bind:value={configHeaderKey} placeholder="Header name" class="input flex-1" />
                    <input type="text" bind:value={configHeaderValue} placeholder="Header value" class="input flex-1" />
                    <button onclick={addConfigHeader} disabled={!configHeaderKey.trim() || !configHeaderValue.trim()} class="btn btn-ghost !px-3 !py-1.5 shrink-0">
                      Add
                    </button>
                  </div>
                </div>

                <div class="border-t border-line pt-4">
                  <h4 class="eyebrow mb-1">Secret Headers</h4>
                  <p class="text-[10px] text-faint mb-3">Encrypted headers for sensitive values (API keys, tokens). Existing values are preserved unless you remove or replace them.</p>
                  {#if existingSecretHeaderKeys.size > 0 || Object.keys(newSecretHeaders).length > 0}
                    <div class="space-y-1.5 mb-3">
                      {#each [...existingSecretHeaderKeys] as key}
                        <div class="flex items-center gap-2">
                          <span class="flex-1 text-xs mono panel-2 px-2 py-1.5 rounded truncate text-text">{key}</span>
                          <span class="flex-1 text-xs mono panel-2 px-2 py-1.5 rounded truncate text-faint">••••••</span>
                          <button onclick={() => removeConfigSecretHeader(key)} class="shrink-0 p-1 text-faint hover:text-bad rounded transition" aria-label="Remove secret header {key}">
                            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                            </svg>
                          </button>
                        </div>
                      {/each}
                      {#each Object.entries(newSecretHeaders) as [key, value]}
                        <div class="flex items-center gap-2">
                          <span class="flex-1 text-xs mono px-2 py-1.5 rounded truncate" style="color:var(--color-ok);border:1px solid color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 10%,var(--color-panel-2))">{key}</span>
                          <span class="flex-1 text-xs mono px-2 py-1.5 rounded truncate" style="color:var(--color-ok);border:1px solid color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 10%,var(--color-panel-2))">new value set</span>
                          <button onclick={() => removeConfigSecretHeader(key)} class="shrink-0 p-1 text-faint hover:text-bad rounded transition" aria-label="Remove secret header {key}">
                            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
                              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
                            </svg>
                          </button>
                        </div>
                      {/each}
                    </div>
                  {/if}
                  <div class="flex items-center gap-2">
                    <input type="text" bind:value={configSecretHeaderKey} placeholder="Header name (e.g. Authorization)" class="input flex-1" />
                    <input type="password" bind:value={configSecretHeaderValue} placeholder="Header value (e.g. Bearer sk-…)" class="input flex-1" />
                    <button onclick={addConfigSecretHeader} disabled={!configSecretHeaderKey.trim() || !configSecretHeaderValue.trim()} class="btn btn-ghost !px-3 !py-1.5 shrink-0">
                      Add
                    </button>
                  </div>
                </div>

                <div class="flex items-center justify-end gap-3 pt-4 border-t border-line">
                  <button onclick={cancelEditConfig} disabled={savingConfig} class="btn btn-ghost">
                    Cancel
                  </button>
                  <button onclick={saveConfig} disabled={savingConfig} class="btn btn-beacon">
                    {savingConfig ? 'Saving…' : 'Save Configuration'}
                  </button>
                </div>
              </div>
            </div>
          {:else}
            <div class="panel p-5">
              <div class="flex items-center justify-between mb-4">
                <h3 class="eyebrow">HTTP Configuration</h3>
                <button onclick={startEditConfig} class="btn btn-ghost !px-3 !py-1.5">
                  Edit
                </button>
              </div>
              {#if webhook.http_config}
                <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                  <div>
                    <p class="text-xs text-muted mb-0.5">Max Retries</p>
                    <p class="text-sm font-medium text-text tnum">{webhook.http_config.max_retries}</p>
                  </div>
                  <div>
                    <p class="text-xs text-muted mb-0.5">Retry Backoff</p>
                    <p class="text-sm font-medium text-text tnum">{webhook.http_config.retry_backoff_seconds}s</p>
                  </div>
                  <div>
                    <p class="text-xs text-muted mb-0.5">Request Timeout</p>
                    <p class="text-sm font-medium text-text tnum">{webhook.http_config.request_timeout_seconds}s</p>
                  </div>
                  <div>
                    <p class="text-xs text-muted mb-0.5">Content Type</p>
                    <p class="text-sm font-medium text-text mono">{webhook.http_config.content_type || 'application/json'}</p>
                  </div>
                  <div>
                    <p class="text-xs text-muted mb-0.5">User Agent</p>
                    <p class="text-sm font-medium text-text mono">{webhook.http_config.user_agent || 'Sparrow-Webhook/1.0'}</p>
                  </div>
                  <div>
                    <p class="text-xs text-muted mb-0.5">Expected Status Codes</p>
                    <div class="flex flex-wrap gap-1">
                      {#each webhook.http_config.expected_status_codes || [] as code}
                        <span class="chip tnum">{code}</span>
                      {/each}
                      {#if (webhook.http_config.expected_status_codes || []).length === 0}
                        <span class="text-sm text-faint">Default (2xx)</span>
                      {/if}
                    </div>
                  </div>
                </div>

                <div class="mt-4 pt-4 border-t border-line">
                  <div class="flex flex-wrap gap-3">
                    <span class="chip" style={webhook.http_config.follow_redirects ? 'color:var(--color-ok);border-color:color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 12%,var(--color-panel-2))' : ''}>
                      <span class="w-1.5 h-1.5 rounded-full" style="background:var(--color-{webhook.http_config.follow_redirects ? 'ok' : 'idle'})"></span>
                      Follow Redirects
                    </span>
                    <span class="chip" style={webhook.http_config.verify_ssl ? 'color:var(--color-ok);border-color:color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 12%,var(--color-panel-2))' : 'color:var(--color-bad);border-color:color-mix(in srgb,var(--color-bad) 35%,transparent);background:color-mix(in srgb,var(--color-bad) 12%,var(--color-panel-2))'}>
                      <span class="w-1.5 h-1.5 rounded-full" style="background:var(--color-{webhook.http_config.verify_ssl ? 'ok' : 'bad'})"></span>
                      Verify SSL
                    </span>
                    <span class="chip" style={webhook.http_config.capture_response_body ? 'color:var(--color-beacon);border-color:color-mix(in srgb,var(--color-beacon) 35%,transparent);background:color-mix(in srgb,var(--color-beacon) 12%,var(--color-panel-2))' : ''}>
                      <span class="w-1.5 h-1.5 rounded-full" style="background:var(--color-{webhook.http_config.capture_response_body ? 'beacon' : 'idle'})"></span>
                      Capture Response
                    </span>
                    {#if webhook.http_config.webhook_secret}
                      <span class="chip" style="color:var(--color-beacon);border-color:color-mix(in srgb,var(--color-beacon) 35%,transparent);background:color-mix(in srgb,var(--color-beacon) 12%,var(--color-panel-2))">
                        <span class="w-1.5 h-1.5 rounded-full" style="background:var(--color-beacon)"></span>
                        Secret Configured
                      </span>
                    {/if}
                  </div>
                </div>
              {:else}
                <p class="text-sm text-muted">Using default HTTP configuration.</p>
              {/if}
            </div>

            <div class="panel p-5">
              <h3 class="eyebrow mb-4">Custom Headers</h3>
              {#if Object.keys(webhook.headers || {}).length > 0}
                <div class="space-y-1.5">
                  {#each Object.entries(webhook.headers || {}) as [key, value]}
                    <div class="flex items-center gap-2 text-sm">
                      <span class="mono text-text font-medium">{key}:</span>
                      <span class="mono text-muted">{value}</span>
                    </div>
                  {/each}
                </div>
              {:else}
                <p class="text-sm text-muted">No custom headers configured.</p>
              {/if}
            </div>

            <div class="panel p-5">
              <h3 class="eyebrow mb-2">Secret Headers</h3>
              <p class="text-xs text-faint mb-4">Encrypted headers for sensitive values. Values are never exposed.</p>
              {#if Object.keys(webhook.secret_headers || {}).length > 0}
                <div class="space-y-1.5">
                  {#each Object.entries(webhook.secret_headers || {}) as [key, value]}
                    <div class="flex items-center gap-2 text-sm">
                      <span class="mono text-text font-medium">{key}:</span>
                      <span class="mono text-faint">••••••</span>
                    </div>
                  {/each}
                </div>
              {:else}
                <p class="text-sm text-muted">No secret headers configured.</p>
              {/if}
            </div>
          {/if}
        </div>
      {/if}

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

<ConfirmDialog
  open={confirmUnregister}
  title="Unregister Webhook"
  message="This will permanently remove the webhook and stop all future deliveries. This action cannot be undone."
  confirmLabel="Unregister"
  variant="danger"
  onconfirm={executeUnregister}
  oncancel={() => { confirmUnregister = false; }}
/>

<ConfirmDialog
  open={confirmRetry}
  title="Retry Deliveries"
  message="This will retry {retryTotal} matching deliver{retryTotal !== 1 ? 'ies' : 'y'} for this webhook. Continue?"
  confirmLabel="Retry"
  variant="warning"
  onconfirm={executeRetry}
  oncancel={() => { confirmRetry = false; retryId = ''; }}
/>
