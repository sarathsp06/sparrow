<script lang="ts">
  // Under /portal so a proxy that forwards only /portal serves it.
  import { page } from "$app/state";
  import SubscriptionEditPage from "#lib/components/SubscriptionEditPage.svelte";
</script>

<div class="min-h-screen bg-ink text-text">
  <SubscriptionEditPage webhookId={page.params.webhookId ?? ""} subscriptionId={page.params.subscriptionId ?? ""} />
</div>
