<script lang="ts">
  import { createClient } from "@connectrpc/connect";
  import { createConnectTransport } from "@connectrpc/connect-web";
  import { onMount } from 'svelte';
  import { ListWebhooksRequest, UnregisterWebhookRequest, GetWebhookDeliveryHistoryRequest, RegisteredWebhook, WebhookDelivery } from '../../../../proto/webhook_pb.js';
  import { WebhookService } from '../../../../proto/webhook_pb.js';

  let webhooks: RegisteredWebhook[] = [];
  let deliveries: WebhookDelivery[] = [];
  let loading = true;
  let error = '';
  const transport = createConnectTransport({
    baseUrl: "http://localhost:8080",
  });
  const client = createClient(WebhookService, transport);

  async function fetchWebhooks() {
    loading = true;
    error = '';
    try {
      const req = new ListWebhooksRequest({
        namespace: 'default',
        event: '',
        activeOnly: false,
      });
      const res = await client.listWebhooks(req);
      webhooks = res.webhooks || [];
    } catch (e) {
      error = 'Failed to load webhooks';
    }
    loading = false;
  }

  async function fetchDeliveries() {
    console.log("FETCH DELIVERIES");
    try {
      await Promise.all(webhooks.map(async webhook => {
        const req = new GetWebhookDeliveryHistoryRequest({
          namespace: 'default',
          webhookId: webhook.webhookId,
          limit: 5
        });
        const res = await client.getWebhookDeliveryHistory(req);
        deliveries.push(...(res.deliveries || []));
      }));
    } catch (e) {
      error = 'Failed to load deliveries';
      console.error(e);
    }
  }

  onMount(async () => {
    await fetchWebhooks();
    await fetchDeliveries();
    console.log("DELIVERIES",...deliveries);
  });

  async function unregisterWebhook(webhookId: string) {
    try {
      const req = new UnregisterWebhookRequest({ webhookId });
      await client.unregisterWebhook(req);
      await fetchWebhooks(); // Refresh the list
    } catch (e) {
      error = `Failed to unregister webhook: ${webhookId}`;
    }
  }
</script>

<!-- Paper-inspired Light Theme Webhook Dashboard -->
<div class="min-h-screen bg-gradient-to-br from-white via-gray-50 to-gray-100 font-display relative overflow-hidden">
  <div class="flex items-center bg-white/90 backdrop-blur-md p-6 pb-2 justify-between sticky top-0 z-10 border-b border-gray-200 shadow-sm">
    <h1 class="text-gray-900 text-2xl font-extrabold leading-tight tracking-tight flex items-center gap-2">
      <span class="material-symbols-outlined text-primary text-3xl">dashboard</span>
      Webhook Dashboard
    </h1>
    <a href="/webhooks/register" class="bg-primary/90 text-white px-5 py-2 rounded-lg font-semibold shadow hover:bg-primary transition">+ Register Webhook</a>
  </div>

  <div class="p-6 max-w-3xl mx-auto">
    {#if loading}
      <div class="flex justify-center items-center h-32">
        <span class="ml-3 text-lg text-gray-700">Loading...</span>
      </div>
    {:else if error}
      <div class="bg-red-50 rounded-lg p-4 text-red-600 shadow mb-4 flex items-center gap-2">
        <span class="material-symbols-outlined">error</span>{error}
      </div>
    {:else if webhooks.length === 0}
      <div class="bg-white rounded-lg p-4 text-gray-600 shadow mb-4 flex items-center gap-2">
        <span class="material-symbols-outlined text-gray-400">info</span>No webhooks found.
      </div>
    {:else}
      <div class="grid gap-6">
        {#each webhooks as wh}
          <div class="paper-card flex flex-col md:flex-row justify-between items-center bg-white rounded-xl shadow-md p-6 hover:shadow-lg transition">
            <div class="flex flex-col gap-1">
              <div class="flex items-center gap-2">
                <span class="material-symbols-outlined text-primary text-2xl">link</span>
                <span class="font-bold text-gray-900">{ wh.description ?? wh.url}</span>
              </div>
              <span class="text-gray-500 text-sm">{ wh.url}</span>
            </div>
            <div class="flex gap-2">
                <a href={`/webhooks/${wh.webhookId}`} class="bg-primary/10 text-primary font-semibold px-4 py-2 rounded-lg shadow hover:bg-primary/20 transition ml-4">Details</a>
                <button on:click={() => unregisterWebhook(wh.webhookId)} class="bg-red-500/10 text-red-500 font-semibold px-4 py-2 rounded-lg shadow hover:bg-red-500/20 transition">Delete</button>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <!-- Latest Deliveries Section -->
  <div class="p-6 max-w-3xl mx-auto">
    <h2 class="text-lg font-semibold text-gray-800 mb-4">Latest Deliveries (Namespace: default)</h2>
    {#if deliveries.length === 0}
      <div class="bg-gray-50 rounded-md p-3 text-gray-500 shadow mb-4 flex items-center gap-2">
        <span class="material-symbols-outlined text-gray-400">info</span>No deliveries found.
      </div>
    {:else}
      <div class="grid gap-4">
        {#each deliveries as d}
          <div class="delivery-card flex flex-col md:flex-row justify-between items-center bg-white rounded-md shadow-sm p-4">
            <div class="flex flex-col gap-1">
              <span class="font-semibold text-gray-900">Webhook: {d.webhookId}</span>
              <span class="text-gray-500 text-sm">Status: {d.status}</span>
              <span class="text-gray-500 text-sm">Delivered At: {d.deliveredAt}</span>
            </div>
            <a href={`/deliveries/${d.deliveryId}`} class="text-gray-700 font-medium px-3 py-1 rounded hover:underline ml-4">Details</a>
          </div>
        {/each}
      </div>
    {/if}
  </div>

  <style>
    .paper-card {
      background: #fff;
      border: 1px solid #f3f3f3;
      box-shadow: 0 2px 8px 0 rgba(60, 60, 100, 0.06);
      transition: box-shadow 0.2s, transform 0.2s;
    }
    .paper-card:hover {
      box-shadow: 0 8px 24px 0 rgba(60, 60, 100, 0.10);
      transform: translateY(-2px) scale(1.02);
    }
    .animate-spin {
      animation: spin 1s linear infinite;
    }
    @keyframes spin {
      100% { transform: rotate(360deg); }
    }
    .bg-primary { background-color: #2563eb; }
    .text-primary { color: #2563eb; }
    .delivery-card {
      background: #fff;
      border: 1px solid #ececec;
      box-shadow: 0 1px 4px 0 rgba(60, 60, 100, 0.04);
      transition: box-shadow 0.2s, transform 0.2s;
    }
    .delivery-card:hover {
      box-shadow: 0 4px 12px 0 rgba(60, 60, 100, 0.08);
      transform: translateY(-1px) scale(1.01);
    }
  </style>
</div>
