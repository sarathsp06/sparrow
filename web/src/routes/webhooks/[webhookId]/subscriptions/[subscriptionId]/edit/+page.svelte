<script lang="ts">
  import { page } from "$app/state";
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import { onMount } from "svelte";
  import SubscriptionForm from "$lib/components/SubscriptionForm.svelte";
  import type { components } from "$lib/api-types";

  type SubscriptionItem = components["schemas"]["SubscriptionItem"];

  const webhookId = $derived(page.params.webhookId ?? "");
  const subscriptionId = $derived(page.params.subscriptionId ?? "");
  let consumer = $state("");
  let webhookUrl = $state("");
  let requiresTransform = $state(false);
  let subscription: SubscriptionItem | null = $state(null);
  let error = $state("");
  let loading = $state(true);

  onMount(async () => {
    try {
      const res = unwrap(await api.GET('/v1/webhooks', { params: { query: { webhook_id: webhookId } } }));
      const wh = res.items?.[0];
      if (!wh) { error = "Webhook not found"; return; }
      consumer = wh.consumer;
      webhookUrl = wh.url;
      requiresTransform = wh.requires_transform;
      subscription = unwrap(await api.GET('/v1/consumers/{consumer}/subscriptions/{subscription_id}', {
        params: { path: { consumer: wh.consumer, subscription_id: subscriptionId } },
      }));
    } catch (e: any) {
      error = formatAPIError(e, "Failed to load subscription");
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head><title>Edit Subscription | Sparrow</title></svelte:head>

<main class="max-w-6xl mx-auto px-4 py-6">
  <nav class="text-xs text-muted mb-3"><a href="/webhooks" class="hover:text-text">Webhooks</a> / <a href="/webhooks/{webhookId}?tab=subscriptions" class="hover:text-text">{webhookUrl || webhookId}</a> / Edit subscription</nav>
  <h1 class="text-xl font-semibold text-text mb-4">Edit subscription {#if subscription}<span class="text-muted font-normal text-base">· {subscription.event_name === "*" ? "all events" : subscription.event_name}</span>{/if}</h1>
  {#if loading}
    <p class="text-sm text-muted">Loading…</p>
  {:else if error}
    <p class="text-sm" style="color:var(--color-bad)">{error}</p>
  {:else if subscription}
    <SubscriptionForm {webhookId} {consumer} mode="edit" {requiresTransform} {subscription} />
  {/if}
</main>
