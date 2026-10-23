<script lang="ts">
  import { api, unwrap, webhookHref, webhooksHref } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import { onMount } from "svelte";
  import SubscriptionForm from "./SubscriptionForm.svelte";

  // The add-subscription page, shared by the console and the portal routes;
  // its links stay in whichever one it is opened from (see webhookHref).
  let { webhookId }: { webhookId: string } = $props();
  let consumer = $state("");
  let webhookUrl = $state("");
  let requiresTransform = $state(false);
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
    } catch (e: any) {
      error = formatAPIError(e, "Failed to load webhook");
    } finally {
      loading = false;
    }
  });
</script>

<svelte:head><title>New Subscription | Sparrow</title></svelte:head>

<main class="max-w-6xl mx-auto px-4 py-6">
  <nav class="text-xs text-muted mb-3"><a href={webhooksHref} class="hover:text-text">Webhooks</a> / <a href={webhookHref(webhookId, "subscriptions")} class="hover:text-text">{webhookUrl || webhookId}</a> / New subscription</nav>
  <h1 class="text-xl font-semibold text-text mb-4">Add subscription</h1>
  {#if loading}
    <p class="text-sm text-muted">Loading…</p>
  {:else if error}
    <p class="text-sm" style="color:var(--color-bad)">{error}</p>
  {:else}
    <SubscriptionForm {webhookId} {consumer} mode="create" {requiresTransform} />
  {/if}
</main>
