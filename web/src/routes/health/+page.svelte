<script lang="ts">
  import favicon from "$lib/assets/favicon.svg";
  import { client } from "$lib/services";

  import { onMount } from "svelte";
  import type {
    HealthSummary,
    NamespaceStats,
  } from "../../../../proto/webhook_pb.js";

  let healthSummary: HealthSummary | undefined = $state();
  let namespaceStats: NamespaceStats | undefined = $state();
  let loading = $state(true);
  let error = $state("");

  async function fetchData() {
    loading = true;
    error = "";
    try {
      const summaryReq = {};
      const summaryRes = await client.getHealthSummary(summaryReq);
      healthSummary = summaryRes.summary;

      const statsReq = { namespace: "default" };
      const statsRes = await client.getNamespaceStats(statsReq);
      namespaceStats = statsRes.stats;
    } catch (e: any) {
      error = `Failed to load health data: ${e.message}`;
    } finally {
      loading = false;
    }
  }

  onMount(fetchData);
</script>

<div class="min-h-screen bg-gray-50 font-display">
  <main class="p-6">
    {#if loading}
      <div class="flex justify-center items-center h-40">
        <span class="material-symbols-outlined animate-spin">
          <img src={favicon} alt="favicon" class="inline-block w-8 h-8" />
        </span>
      </div>
    {:else if error}
      <div
        class="bg-red-100 border border-red-300 text-red-700 rounded-lg p-4 mb-6 flex items-center gap-3 shadow-sm"
      >
        <span class="material-symbols-outlined">error</span>
        <p>{error}</p>
      </div>
    {:else}
      {#if healthSummary}
        <div class="mb-6">
          <h2 class="text-xl font-bold text-gray-800 mb-4">Overall Health</h2>
          <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
            <div class="bg-white p-4 rounded-lg shadow-sm border text-center">
              <p class="text-2xl font-bold text-green-500">
                {healthSummary.healthyCount}
              </p>
              <p class="text-gray-600">Healthy</p>
            </div>
            <div class="bg-white p-4 rounded-lg shadow-sm border text-center">
              <p class="text-2xl font-bold text-yellow-500">
                {healthSummary.degradedCount}
              </p>
              <p class="text-gray-600">Degraded</p>
            </div>
            <div class="bg-white p-4 rounded-lg shadow-sm border text-center">
              <p class="text-2xl font-bold text-red-500">
                {healthSummary.unhealthyCount}
              </p>
              <p class="text-gray-600">Unhealthy</p>
            </div>
            <div class="bg-white p-4 rounded-lg shadow-sm border text-center">
              <p class="text-2xl font-bold text-gray-500">
                {healthSummary.unknownCount}
              </p>
              <p class="text-gray-600">Unknown</p>
            </div>
          </div>
        </div>
      {/if}

      {#if namespaceStats}
        <div>
          <h2 class="text-xl font-bold text-gray-800 mb-4">
            Namespace: <span class="font-mono text-primary">default</span>
          </h2>
          <div class="grid grid-cols-2 sm:grid-cols-3 gap-4">
            <div class="bg-white p-4 rounded-lg shadow-sm border">
              <p class="font-semibold text-gray-600">Total Webhooks</p>
              <p class="text-2xl font-bold">{namespaceStats.totalWebhooks}</p>
            </div>
            <div class="bg-white p-4 rounded-lg shadow-sm border">
              <p class="font-semibold text-gray-600">Active Webhooks</p>
              <p class="text-2xl font-bold">{namespaceStats.activeWebhooks}</p>
            </div>
            <div class="bg-white p-4 rounded-lg shadow-sm border">
              <p class="font-semibold text-gray-600">Success Rate</p>
              <p class="text-2xl font-bold">
                {(namespaceStats.successRate * 100).toFixed(2)}%
              </p>
            </div>
            <div class="bg-white p-4 rounded-lg shadow-sm border">
              <p class="font-semibold text-gray-600">Successful Deliveries</p>
              <p class="text-2xl font-bold">
                {namespaceStats.successfulDeliveries}
              </p>
            </div>
            <div class="bg-white p-4 rounded-lg shadow-sm border">
              <p class="font-semibold text-gray-600">Failed Deliveries</p>
              <p class="text-2xl font-bold">
                {namespaceStats.failedDeliveries}
              </p>
            </div>
            <div class="bg-white p-4 rounded-lg shadow-sm border">
              <p class="font-semibold text-gray-600">Pending Deliveries</p>
              <p class="text-2xl font-bold">
                {namespaceStats.pendingDeliveries}
              </p>
            </div>
          </div>
        </div>
      {/if}
    {/if}
  </main>
</div>

<style>
  .bg-primary {
    background-color: #1d4ed8;
  }
  .text-primary {
    color: #1d4ed8;
  }
</style>
