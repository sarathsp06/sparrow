<script lang="ts">
  import { page } from "$app/state";
  import SubscriptionNewPage from "#lib/components/SubscriptionNewPage.svelte";
</script>

<SubscriptionNewPage webhookId={page.params.webhookId ?? ""} />
