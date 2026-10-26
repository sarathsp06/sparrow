<script lang="ts">
  import { page } from "$app/state";
  import SubscriptionEditPage from "#lib/components/SubscriptionEditPage.svelte";
</script>

<SubscriptionEditPage webhookId={page.params.webhookId ?? ""} subscriptionId={page.params.subscriptionId ?? ""} />
