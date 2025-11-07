<script lang="ts">
    import favicon from '$lib/assets/favicon.svg';
    import { stopPropagation } from 'svelte/legacy';

  import { goto } from "$app/navigation";
  import { client } from "$lib/services";
  import { onMount } from "svelte";
  import type { RegisteredWebhook } from "../../../../proto/webhook_pb.js";
  import { WebhookHealth } from "../../../../proto/webhook_pb.js";

  let webhooks: RegisteredWebhook[] = $state([]);
  let loading = $state(true);
  let error = $state("");
  let namespace = $state("default");



  const healthColor: Record<WebhookHealth, string> = {
    [WebhookHealth.HEALTH_UNSPECIFIED]: "gray-500",
    [WebhookHealth.HEALTH_HEALTHY]: "green-500",
    [WebhookHealth.HEALTH_DEGRADED]: "yellow-500",
    [WebhookHealth.HEALTH_UNHEALTHY]: "red-500",
  };

  const healthText: Record<WebhookHealth, string> = {
    [WebhookHealth.HEALTH_UNSPECIFIED]: "Unknown",
    [WebhookHealth.HEALTH_HEALTHY]: "Healthy",
    [WebhookHealth.HEALTH_DEGRADED]: "Degraded",
    [WebhookHealth.HEALTH_UNHEALTHY]: "Unhealthy",
  };

  async function fetchWebhooks() {
    loading = true;
    error = "";
    try {
      if (namespace.trim() === "") {
        error = "Namespace cannot be empty.";
        webhooks = [];
      } else {
        const res = await client.listWebhooks({ namespace: namespace.trim() });
        webhooks = res.webhooks || [];
      }
    } catch (e: any) {
      console.error(e);
      error = `Failed to load webhooks: ${e.message}`;
    } finally {
      loading = false;
    }
  }

  onMount(fetchWebhooks);

  async function unregisterWebhook(webhookId: string) {
    try {
      const req = { webhookId };
      await client.unregisterWebhook(req);
      await fetchWebhooks(); // Refresh the list
    } catch (e: any) {
      error = `Failed to unregister webhook: ${e.message}`;
    }
  }
</script>

<div class="min-h-screen bg-gray-50 font-display">
  <main class="p-6">
    <div class="mb-6">
      <input
        type="text"
        placeholder="Filter by namespace..."
        bind:value={namespace}
        class="border border-gray-300 rounded-md px-4 py-2 w-full max-w-sm"
        onkeydown={(e) => {
          if (e.key === "Enter") {
            fetchWebhooks();
          }
        }}
      />
    </div>
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
    {:else if webhooks.length === 0}
      <div
        class="bg-white border rounded-lg p-8 text-center text-gray-500 shadow-sm flex flex-col items-center gap-4"
      >
        <span class="material-symbols-outlined text-5xl text-gray-300"
          >webhook</span
        >
        <h3 class="text-xl font-semibold">No webhooks found</h3>
        <p>Get started by registering a new webhook.</p>
      </div>
    {:else}
      <div class="overflow-x-auto rounded-lg border border-gray-400 p-4">
        <table class="w-full text-sm text-left">
          <thead class="text-xs text-gray-700 uppercase bg-gray-50">
            <tr class="text-center">
              <th>Namespace</th>
              <th>URL</th>
              <th>Events</th>
              <th>Health</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each webhooks as wh}
              <tr
                class="border-b border-gray-100 hover:bg-gray-100 hover:cursor-pointer"
                data-id={wh.webhookId}
                onclick={() => goto(`/webhooks/${wh.webhookId}`)}
              >
                <td class="px-4 py-2">{wh.namespace}</td>
                <td class="px-4 py-2">{wh.url}</td>
                <td class="px-4 py-2">{wh.events.join(", ")}</td>
                <td class="px-4 py-2">
                  <span class="text-{healthColor[wh.health]}">
                    {healthText[wh.health]}
                  </span>
                </td>
                <td class="px-4 py-2">
                  <button
                    class="bg-red-500 text-white px-3 py-2 rounded-md hover:bg-red-600"
                    onclick={stopPropagation(() =>
                      unregisterWebhook(wh.webhookId))}>Unregister</button
                  >
                </td>
              </tr>
            {/each}
          </tbody>
        </table> 
      </div>
    {/if}
  </main>
<a
  href="/webhooks/register"
  class="fixed bottom-4 right-4 hover:bg-primary/10 bg-primary text-white font-semibold p-2 rounded-lg"
>
  + Register Webhook
</a>

</div>

<style>
  .bg-primary {
    background-color: #13348f;
  }
  .text-primary {
    color: #1d4ed8;
  }
  .border-primary {
    border-color: #1d4ed8;
  }
</style>
