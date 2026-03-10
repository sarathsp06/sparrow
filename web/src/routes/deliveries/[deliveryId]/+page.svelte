<script lang="ts">
  import { page } from '$app/state';
  import { deliveryClient as client } from "$lib";
  import StatusBadge from '$lib/components/StatusBadge.svelte';
  import { onMount } from 'svelte';
  import type { WebhookDelivery } from "../../../../../proto/webhook_pb.js";
  import { WebhookDeliveryStatus } from "../../../../../proto/webhook_pb.js";

  let delivery: WebhookDelivery | undefined = $state();
  let loading = $state(true);
  let error = $state('');

  const deliveryId = page.params.deliveryId;

  onMount(async () => {
    try {
      const res = await client.getDeliveryStatus({
        deliveryId: deliveryId,
        namespace: '',
      });
      delivery = res.delivery;
    } catch (e: any) {
      error = `Failed to load delivery details: ${e.message}`;
    } finally {
      loading = false;
    }
  });

  function formatResponseCode(code: number): string {
    if (code === 0) return 'N/A';
    return String(code);
  }

  function formatResponseBody(body: string): string {
    if (!body) return 'No response body';
    try {
      return JSON.stringify(JSON.parse(body), null, 2);
    } catch {
      return body;
    }
  }

  function getCategoryDisplay(category: string): { label: string; color: string; bgColor: string; borderColor: string } {
    switch (category) {
      case 'client_error': return { label: '4xx Client Error', color: 'text-orange-700', bgColor: 'bg-orange-50', borderColor: 'border-orange-200' };
      case 'server_error': return { label: '5xx Server Error', color: 'text-red-700', bgColor: 'bg-red-50', borderColor: 'border-red-200' };
      case 'timeout': return { label: 'Timeout', color: 'text-yellow-700', bgColor: 'bg-yellow-50', borderColor: 'border-yellow-200' };
      case 'dns_error': return { label: 'DNS Error', color: 'text-purple-700', bgColor: 'bg-purple-50', borderColor: 'border-purple-200' };
      case 'tls_error': return { label: 'TLS Error', color: 'text-purple-700', bgColor: 'bg-purple-50', borderColor: 'border-purple-200' };
      case 'connection_refused': return { label: 'Connection Refused', color: 'text-purple-700', bgColor: 'bg-purple-50', borderColor: 'border-purple-200' };
      case 'network_error': return { label: 'Network Error', color: 'text-purple-700', bgColor: 'bg-purple-50', borderColor: 'border-purple-200' };
      case 'success': return { label: 'Success', color: 'text-green-700', bgColor: 'bg-green-50', borderColor: 'border-green-200' };
      default: return { label: category || 'Unknown', color: 'text-gray-700', bgColor: 'bg-gray-50', borderColor: 'border-gray-200' };
    }
  }
</script>

<svelte:head>
  <title>Delivery {deliveryId.substring(0, 8)}... | Sparrow</title>
</svelte:head>

<div class="min-h-screen bg-gray-50">
  <main class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
    <!-- Breadcrumb -->
    <nav class="flex items-center text-sm text-gray-500 mb-6">
      <a href="/webhooks" class="hover:text-gray-900 transition">Webhooks</a>
      <span class="mx-2">/</span>
      {#if delivery}
        <a href={`/webhooks/${delivery.webhookId}`} class="hover:text-gray-900 transition">
          {delivery.webhookId.substring(0, 8)}...
        </a>
        <span class="mx-2">/</span>
      {/if}
      <span class="text-gray-900 font-medium">Delivery {deliveryId.substring(0, 8)}...</span>
    </nav>

    {#if loading}
      <!-- Loading skeleton -->
      <div class="max-w-2xl">
        <div class="animate-pulse mb-6">
          <div class="h-7 bg-gray-200 rounded w-48 mb-2"></div>
          <div class="h-4 bg-gray-100 rounded w-64"></div>
        </div>
        <div class="bg-white rounded-lg border border-gray-200 p-6">
          <div class="animate-pulse space-y-6">
            {#each Array(6) as _}
              <div>
                <div class="h-3 bg-gray-200 rounded w-24 mb-2"></div>
                <div class="h-5 bg-gray-100 rounded w-64"></div>
              </div>
            {/each}
          </div>
        </div>
      </div>
    {:else if error}
      <div class="bg-red-50 border border-red-200 rounded-lg p-4 max-w-2xl">
        <div class="flex items-start gap-3">
          <svg class="w-5 h-5 text-red-500 mt-0.5 shrink-0" fill="currentColor" viewBox="0 0 20 20">
            <path fill-rule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clip-rule="evenodd" />
          </svg>
          <div>
            <p class="text-sm font-medium text-red-800">{error}</p>
            <button onclick={() => window.location.reload()} class="text-sm text-red-600 hover:text-red-800 underline mt-1">Retry</button>
          </div>
        </div>
      </div>
    {:else if delivery}
      <div class="max-w-2xl">
        <!-- Header -->
        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 mb-6">
          <div>
            <h1 class="text-2xl font-bold text-gray-900">Delivery Details</h1>
            <p class="text-sm text-gray-500 mt-0.5 font-mono">{delivery.deliveryId}</p>
          </div>
          <StatusBadge status={delivery.status} />
        </div>

        <!-- Details card -->
        <div class="bg-white rounded-lg border border-gray-200 divide-y divide-gray-100">
          <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-gray-100">
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Webhook ID</p>
              <a href={`/webhooks/${delivery.webhookId}`} class="text-sm font-mono text-blue-600 hover:text-blue-800 transition">
                {delivery.webhookId}
              </a>
            </div>
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Event ID</p>
              <p class="text-sm font-mono text-gray-900">{delivery.eventId}</p>
            </div>
          </div>

          <div class="grid grid-cols-1 sm:grid-cols-2 divide-y sm:divide-y-0 sm:divide-x divide-gray-100">
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Status</p>
              <span class="text-sm font-medium text-gray-900">
                {WebhookDeliveryStatus[delivery.status].replace("DELIVERY_", "")}
              </span>
            </div>
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Response Code</p>
              <span class="text-sm font-mono {delivery.responseCode >= 200 && delivery.responseCode < 300 ? 'text-green-700' : delivery.responseCode >= 400 ? 'text-red-700' : 'text-gray-900'}">
                {formatResponseCode(delivery.responseCode)}
              </span>
            </div>
          </div>

          {#if delivery.errorCategory && delivery.errorCategory !== '' && delivery.errorCategory !== 'success'}
            {@const cat = getCategoryDisplay(delivery.errorCategory)}
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-1">Error Category</p>
              <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium {cat.bgColor} {cat.color} border {cat.borderColor}">
                {cat.label}
              </span>
            </div>
          {/if}

          {#if delivery.responseBody}
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Response Body</p>
              <pre class="text-xs bg-gray-50 border border-gray-200 rounded-lg p-4 overflow-auto max-h-64 text-gray-800">{formatResponseBody(delivery.responseBody)}</pre>
            </div>
          {/if}

          {#if delivery.errorMessage}
            <div class="px-6 py-4">
              <p class="text-xs font-medium text-gray-500 uppercase tracking-wider mb-2">Error Message</p>
              <div class="bg-red-50 border border-red-200 rounded-lg p-3">
                <p class="text-sm text-red-800">{delivery.errorMessage}</p>
              </div>
            </div>
          {/if}
        </div>

        <!-- Back link -->
        <div class="mt-6">
          <a
            href={`/webhooks/${delivery.webhookId}`}
            class="inline-flex items-center gap-1 text-sm text-gray-600 hover:text-gray-900 transition"
          >
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
            </svg>
            Back to webhook
          </a>
        </div>
      </div>
    {/if}
  </main>
</div>
