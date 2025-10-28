<script lang="ts">
  import { page } from '$app/stores';
  import { createClient } from "@connectrpc/connect";
  import { createConnectTransport } from "@connectrpc/connect-web";
  import { onMount } from 'svelte';
  import type { WebhookDelivery } from "../../../../../proto/webhook_pb.js";
  import { WebhookService } from '../../../../../proto/webhook_pb.js';

  let delivery: WebhookDelivery | undefined;
  let loading = true;
  let error = '';

  const transport = createConnectTransport({
    baseUrl: "http://localhost:8080",
  });
  const client = createClient(WebhookService, transport);

  onMount(async () => {
    const deliveryId = $page.params.deliveryId;
    try {
      const req = {
          deliveryId,
          namespace: 'default',
      };
      const res = await client.getWebhookDeliveryStatus({
        deliveryId: deliveryId, namespace: 'default',
      })
      delivery = res.delivery;
    } catch (e) {
      error = 'Failed to load delivery details';
    }
    loading = false;
  });
</script>

<div class="min-h-screen bg-gradient-to-br from-white via-gray-50 to-gray-100 font-display">
  <div class="p-6 max-w-lg mx-auto">
    <h1 class="text-2xl font-bold mb-4">Delivery Details</h1>
    {#if loading}
      <p>Loading...</p>
    {:else if error}
      <p class="text-red-500">{error}</p>
    {:else if delivery}
        <div class="bg-white rounded-lg shadow p-6">
            <div class="mb-4">
                <p class="text-sm font-medium text-gray-700">Delivery ID</p>
                <p class="text-lg">{delivery.deliveryId}</p>
            </div>
            <div class="mb-4">
                <p class="text-sm font-medium text-gray-700">Webhook ID</p>
                <p class="text-lg">{delivery.webhookId}</p>
            </div>
            <div class="mb-4">
                <p class="text-sm font-medium text-gray-700">Event ID</p>
                <p class="text-lg">{delivery.eventId}</p>
            </div>
            <div class="mb-4">
                <p class="text-sm font-medium text-gray-700">Status</p>
                <p class="text-lg">{delivery.status}</p>
            </div>
            <div class="mb-4">
                <p class="text-sm font-medium text-gray-700">Response Code</p>
                <p class="text-lg">{delivery.responseCode}</p>
            </div>
            <div class="mb-4">
                <p class="text-sm font-medium text-gray-700">Response Body</p>
                <p class="text-lg">{delivery.responseBody}</p>
            </div>
        </div>
    {/if}
  </div>
</div>
