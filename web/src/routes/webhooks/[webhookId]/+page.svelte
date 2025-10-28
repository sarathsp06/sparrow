<script lang="ts">
  import { page } from '$app/stores';
  import { createClient } from "@connectrpc/connect";
  import { createConnectTransport } from "@connectrpc/connect-web";
  import { onMount } from 'svelte';
  import type { WebhookDelivery, WebhookHealthMetrics } from "../../../../../proto/webhook_pb.js";
  import { WebhookService } from '../../../../../proto/webhook_pb.js';

  let healthMetrics: WebhookHealthMetrics | null;
  let deliveries: WebhookDelivery[] = [];
  let loading = true;
  let error = '';

  const transport = createConnectTransport({
    baseUrl: "http://localhost:8080",
  });
  const client = createClient(WebhookService, transport);

  onMount(async () => {
    const webhookId = $page.params.webhookId;
    try {
      const healthReq = ({
          webhookId,
          namespace: 'default',
      });
      const healthRes = await client.getWebhookHealth(healthReq);
      healthMetrics = healthRes.metrics ? healthRes.metrics : null;

      const deliveryReq = ({
          webhookId,
          namespace: 'default',
          limit: 20,
      });
      const deliveryRes = await client.getWebhookDeliveryHistory(deliveryReq);
      deliveries = deliveryRes.deliveries;

    } catch (e) {
      error = 'Failed to load webhook details';
    }
    loading = false;
  });
</script>

<div class="min-h-screen bg-gradient-to-br from-white via-gray-50 to-gray-100 font-display">
  <div class="p-6 max-w-3xl mx-auto">
    <h1 class="text-2xl font-bold mb-4">Webhook Details</h1>
    {#if loading}
      <p>Loading...</p>
    {:else if error}
      <p class="text-red-500">{error}</p>
    {:else if healthMetrics}
        <div class="bg-white rounded-lg shadow p-6 mb-6">
            <h2 class="text-xl font-bold mb-4">Health Metrics</h2>
            <div class="grid grid-cols-2 gap-4">
                <div>
                    <p class="text-sm font-medium text-gray-700">Avg Response Time</p>
                    <p class="text-lg">{healthMetrics.avgResponseTime} ms</p>
                </div>
                <div>
                    <p class="text-sm font-medium text-gray-700">Total Deliveries</p>
                    <p class="text-lg">{healthMetrics.totalDeliveries}</p>
                </div>
                <div>
                    <p class="text-sm font-medium text-gray-700">Successful Deliveries</p>
                    <p class="text-lg">{healthMetrics.successfulDeliveries}</p>
                </div>
                <div>
                    <p class="text-sm font-medium text-gray-700">Failed Deliveries</p>
                    <p class="text-lg">{healthMetrics.failedDeliveries}</p>
                </div>
            </div>
        </div>

        <h2 class="text-xl font-bold mb-4">Delivery History</h2>
        <div class="bg-white rounded-lg shadow p-6">
            {#if deliveries.length === 0}
                <p>No deliveries found.</p>
            {:else}
                <div class="grid gap-4">
                    {#each deliveries as d}
                      <div class="delivery-card flex flex-col md:flex-row justify-between items-center bg-white rounded-md shadow-sm p-4">
                        <div class="flex flex-col gap-1">
                          <span class="font-semibold text-gray-900">Delivery: {d.deliveryId}</span>
                          <span class="text-gray-500 text-sm">Status: {d.status}</span>
                          <span class="text-gray-500 text-sm">Response Code: {d.responseCode}</span>
                        </div>
                        <a href={`/deliveries/${d.deliveryId}`} class="text-gray-700 font-medium px-3 py-1 rounded hover:underline ml-4">Details</a>
                      </div>
                    {/each}
                  </div>
            {/if}
        </div>
    {/if}
  </div>
</div>
