<script lang="ts">
  // Email alert recipients for one webhook: its own alert configs plus the
  // consumer-wide ones that also cover it. Sparrow only sends these emails
  // when an alert-delivery webhook exists under _sparrow (the capability).
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import type { components } from "$lib/api-types";
  import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
  import { ALERT_EVENT_TYPES, ALERT_SETUP_HREF, ALERTS_GUIDE_URL, isEmail } from "$lib/system";

  type AlertConfigItem = components["schemas"]["AlertConfigItem"];

  let { consumer, webhookId }: { consumer: string; webhookId: string } = $props();

  let configs: AlertConfigItem[] = $state([]);
  let deliveryConfigured: boolean | undefined = $state();
  let loading = $state(true);
  let error = $state("");
  let email = $state("");
  let adding = $state(false);
  let toDelete: AlertConfigItem | null = $state(null);

  const eventLabel: Record<string, string> = {
    "sparrow.webhook.health_changed": "health changes",
    "sparrow.webhook.delivery_failed": "permanent failures",
    "sparrow.webhook.disabled": "auto-disable",
  };

  async function load() {
    loading = true;
    error = "";
    try {
      const [list, caps] = await Promise.all([
        api.GET("/v1/consumers/{consumer}/alert-configs", { params: { path: { consumer } } }),
        api.GET("/v1/capabilities"),
      ]);
      // Without webhook_id the API lists the whole consumer: keep this
      // webhook's configs and the consumer-wide ones.
      configs = (unwrap(list).items ?? []).filter((c) => !c.webhook_id || c.webhook_id === webhookId);
      deliveryConfigured = unwrap(caps).alert_delivery.configured;
    } catch (e: any) {
      error = formatAPIError(e, "Failed to load alert recipients");
    } finally {
      loading = false;
    }
  }

  async function add(e: Event) {
    e.preventDefault();
    const value = email.trim();
    if (!isEmail(value)) {
      error = "Enter an email address like ops@example.com.";
      return;
    }
    adding = true;
    error = "";
    try {
      unwrap(await api.POST("/v1/consumers/{consumer}/alert-configs", {
        params: { path: { consumer } },
        body: { webhook_id: webhookId, email: value, event_types: ALERT_EVENT_TYPES },
      }));
      email = "";
      await load();
    } catch (e: any) {
      error = formatAPIError(e, "Failed to add alert recipient");
    } finally {
      adding = false;
    }
  }

  async function remove() {
    const c = toDelete;
    toDelete = null;
    if (!c) return;
    try {
      unwrap(await api.DELETE("/v1/consumers/{consumer}/alert-configs/{alert_config_id}", {
        params: { path: { consumer, alert_config_id: c.id } },
      }));
      await load();
    } catch (e: any) {
      error = formatAPIError(e, "Failed to delete alert recipient");
    }
  }

  $effect(() => {
    consumer;
    webhookId;
    load();
  });
</script>

<div class="panel p-5 mb-6" data-testid="alert-configs">
  <div class="flex items-baseline justify-between gap-3 mb-3">
    <h2 class="eyebrow">Alerts</h2>
    <a href={ALERTS_GUIDE_URL} target="_blank" rel="noreferrer" class="text-xs link-beacon">How alerts work</a>
  </div>

  {#if deliveryConfigured === false}
    <p class="text-xs mb-3 px-3 py-2 rounded-md" style="color:var(--color-warn);background:color-mix(in srgb,var(--color-warn) 10%,transparent)" data-testid="alert-delivery-missing">
      Alert email is not set up on this server: recipients below are stored, but nothing is sent until an operator adds an alert-delivery webhook.
      <a href={ALERT_SETUP_HREF} class="underline">Set it up</a>
    </p>
  {/if}

  {#if loading}
    <div class="h-8 bg-black/[0.03] rounded animate-pulse"></div>
  {:else}
    {#if configs.length === 0}
      <p class="text-sm text-muted mb-3">No one is emailed about this webhook.</p>
    {:else}
      <ul class="divide-y divide-line mb-3">
        {#each configs as c (c.id)}
          <li class="flex items-center justify-between gap-3 py-2">
            <div class="min-w-0">
              <p class="text-sm text-text truncate">{c.email}</p>
              <p class="text-xs text-muted">
                {(c.event_types ?? []).map((t) => eventLabel[t] ?? t).join(", ")}
                · {c.webhook_id ? "this webhook" : `every webhook in ${c.consumer}`}
              </p>
            </div>
            <button type="button" class="btn btn-ghost !px-2 !py-1 !text-xs" onclick={() => (toDelete = c)} aria-label="Remove alert recipient {c.email}">Remove</button>
          </li>
        {/each}
      </ul>
    {/if}

    <form onsubmit={add} class="flex gap-2">
      <input type="email" bind:value={email} placeholder="ops@example.com" aria-label="Alert email" class="input flex-1 min-w-0" />
      <button type="submit" disabled={adding || !email.trim()} class="btn btn-ghost !px-3">{adding ? "Adding…" : "Add recipient"}</button>
    </form>
    <p class="text-xs text-faint mt-1">Emailed when this webhook's health changes, a delivery fails permanently, or Sparrow pauses it.</p>
  {/if}

  {#if error}<p class="text-xs mt-2" style="color:var(--color-bad)">{error}</p>{/if}
</div>

<ConfirmDialog
  open={!!toDelete}
  title="Remove alert recipient"
  message={toDelete?.webhook_id
    ? `${toDelete?.email} stops getting alerts for this webhook.`
    : `${toDelete?.email} stops getting alerts for every webhook in ${toDelete?.consumer}, not just this one.`}
  confirmLabel="Remove"
  onconfirm={remove}
  oncancel={() => (toDelete = null)}
/>
