<script lang="ts">
  // Shown after resuming a paused webhook or subscription that held
  // deliveries: they are never sent automatically, so offer to retry them
  // (snapshot status=paused with the given filter, then a batch retry) or
  // leave them.
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";

  let {
    consumer,
    count,
    filter,
    onclose,
  }: {
    consumer: string;
    count: number;
    filter: { webhook_id?: string; subscription_id?: string; created_after?: string };
    onclose: () => void;
  } = $props();

  let retrying = $state(false);
  let error = $state("");

  async function retryHeld() {
    retrying = true;
    error = "";
    try {
      const list = unwrap(await api.GET("/v1/consumers/{consumer}/deliveries", {
        params: {
          path: { consumer },
          query: { ...filter, status: "paused", prepare_retry: true, limit: 1 },
        },
      }));
      if (list.retry_id) {
        unwrap(await api.POST("/v1/consumers/{consumer}/deliveries:retryBatch", {
          params: { path: { consumer } },
          body: { repush_id: list.retry_id },
        }));
      }
      onclose();
    } catch (e: any) {
      error = formatAPIError(e, "Failed to retry paused deliveries");
    } finally {
      retrying = false;
    }
  }
</script>

<div class="panel-2 p-3 mb-3 flex flex-wrap items-center justify-between gap-3">
  <p class="text-sm text-text">
    {count === 1
      ? "1 delivery was held while paused. It is not sent automatically."
      : `${count} deliveries were held while paused. They are not sent automatically.`}
    {#if error}<span class="block text-xs mt-1" style="color:var(--color-bad)">{error}</span>{/if}
  </p>
  <div class="flex gap-2">
    <button class="btn btn-ghost !px-3 !py-1.5" onclick={onclose}>Leave them</button>
    <button class="btn btn-beacon !px-3 !py-1.5" disabled={retrying} onclick={retryHeld}>Retry paused deliveries</button>
  </div>
</div>
