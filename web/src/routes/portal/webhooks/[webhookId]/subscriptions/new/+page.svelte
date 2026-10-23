<script lang="ts">
  // Under /portal so a proxy that forwards only /portal serves it.
  import { page } from "$app/state";
  import SubscriptionNewPage from "$lib/components/SubscriptionNewPage.svelte";
</script>

<div class="min-h-screen bg-ink text-text">
  <SubscriptionNewPage webhookId={page.params.webhookId ?? ""} />
</div>
