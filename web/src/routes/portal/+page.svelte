<script lang="ts">
  import { onMount } from "svelte";
  import { api, unwrap, portal } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import favicon from "$lib/assets/favicon.svg";
  import HealthBadge from "$lib/components/HealthBadge.svelte";
  import StatusBadge from "$lib/components/StatusBadge.svelte";
  import EmptyState from "$lib/components/EmptyState.svelte";
  import CopyableId from "$lib/components/CopyableId.svelte";
  import ConfirmDialog from "$lib/components/ConfirmDialog.svelte";
  import SubscriptionManager from "$lib/components/SubscriptionManager.svelte";
  import type { components } from "$lib/api-types";

  type WebhookOut = components["schemas"]["WebhookOut"];
  type DeliveryItem = components["schemas"]["DeliveryItem"];
  type SubscriptionItem = components["schemas"]["SubscriptionItem"];
  type EventTypeItem = components["schemas"]["EventTypeItem"];

  const consumer = portal?.consumer ?? "";

  // --- Theme: ?theme in the fragment wins, then saved choice, then OS. ---
  const hashTheme = typeof window !== "undefined"
    ? (window.location.hash.match(/(?:^#|[#&])theme=(dark|light)/)?.[1] ?? "")
    : "";
  let theme = $state<"light" | "dark">(
    (hashTheme ||
      (typeof localStorage !== "undefined" && localStorage.getItem("sparrow_portal_theme")) ||
      (typeof window !== "undefined" && window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light")) as "light" | "dark",
  );
  $effect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("sparrow_portal_theme", theme);
    return () => {
      delete document.documentElement.dataset.theme;
    };
  });

  // --- Data ---
  let loading = $state(true);
  let error = $state("");
  let webhooks: WebhookOut[] = $state([]);
  let stats = $state<{ total_webhooks: number; successful_deliveries: number; failed_deliveries: number; pending_deliveries: number; success_rate: number } | null>(null);
  let eventTypes: EventTypeItem[] = $state([]);

  // Selected webhook detail
  let selectedId = $state<string | null>(null);
  let detailTab = $state<"deliveries" | "subscriptions">("deliveries");
  let deliveries: DeliveryItem[] = $state([]);
  let deliveriesLoading = $state(false);
  let subscriptions: SubscriptionItem[] = $state([]);

  // Register form
  let registerOpen = $state(false);
  let regUrl = $state("");
  let regDescription = $state("");
  let regEvents = $state<string[]>([]);
  let regBusy = $state(false);

  // Delete confirmation
  let confirmDeleteId = $state<string | null>(null);

  const selected = $derived(webhooks.find((w) => w.webhook_id === selectedId));

  async function refresh() {
    error = "";
    try {
      const [whRes, statsRes] = await Promise.all([
        api.GET("/v1/consumers/{consumer}/webhooks", { params: { path: { consumer } } }),
        api.GET("/v1/consumers/{consumer}/stats", { params: { path: { consumer } } }),
      ]);
      webhooks = unwrap(whRes).items || [];
      stats = unwrap(statsRes);
    } catch (e) {
      error = formatAPIError(e, "Failed to load portal data");
    } finally {
      loading = false;
    }
  }

  async function fetchDeliveries(webhookId: string) {
    deliveriesLoading = true;
    try {
      const res = unwrap(
        await api.GET("/v1/consumers/{consumer}/deliveries", {
          params: { path: { consumer }, query: { webhook_id: webhookId, limit: 25 } },
        }),
      );
      deliveries = res.items || [];
    } catch (e) {
      error = formatAPIError(e, "Failed to load deliveries");
    } finally {
      deliveriesLoading = false;
    }
  }

  async function fetchSubscriptions(webhookId: string) {
    try {
      const res = unwrap(
        await api.GET("/v1/consumers/{consumer}/subscriptions", {
          params: { path: { consumer }, query: { webhook_id: webhookId } },
        }),
      );
      subscriptions = res.items || [];
    } catch (e) {
      error = formatAPIError(e, "Failed to load subscriptions");
    }
  }

  function selectWebhook(id: string) {
    if (selectedId === id) {
      selectedId = null;
      return;
    }
    selectedId = id;
    detailTab = "deliveries";
    deliveries = [];
    subscriptions = [];
    fetchDeliveries(id);
    fetchSubscriptions(id);
  }

  async function toggleActive(w: WebhookOut) {
    try {
      const op = w.active ? ":pause" : ":resume";
      unwrap(
        await api.POST(`/v1/consumers/{consumer}/webhooks/{webhook_id}${op}` as never, {
          params: { path: { consumer, webhook_id: w.webhook_id } },
        } as never),
      );
      await refresh();
    } catch (e) {
      error = formatAPIError(e, "Failed to update endpoint");
    }
  }

  async function executeDelete() {
    if (!confirmDeleteId) return;
    try {
      const res = await api.DELETE("/v1/consumers/{consumer}/webhooks/{webhook_id}", {
        params: { path: { consumer, webhook_id: confirmDeleteId } },
      });
      if (res.error) throw new Error("Failed to remove endpoint");
      if (selectedId === confirmDeleteId) selectedId = null;
      await refresh();
    } catch (e) {
      error = formatAPIError(e, "Failed to remove endpoint");
    } finally {
      confirmDeleteId = null;
    }
  }

  async function registerWebhook(e: SubmitEvent) {
    e.preventDefault();
    regBusy = true;
    try {
      unwrap(
        await api.POST("/v1/consumers/{consumer}/webhooks", {
          params: { path: { consumer } },
          body: { url: regUrl, description: regDescription || undefined, events: regEvents.length ? regEvents : undefined },
        }),
      );
      regUrl = "";
      regDescription = "";
      regEvents = [];
      registerOpen = false;
      await refresh();
    } catch (err) {
      error = formatAPIError(err, "Failed to add endpoint");
    } finally {
      regBusy = false;
    }
  }

  async function retryDelivery(deliveryId: string) {
    try {
      unwrap(
        await api.POST("/v1/consumers/{consumer}/deliveries/{delivery_id}:retry", {
          params: { path: { consumer, delivery_id: deliveryId } },
        }),
      );
      if (selectedId) await fetchDeliveries(selectedId);
    } catch (e) {
      error = formatAPIError(e, "Failed to retry delivery");
    }
  }

  onMount(() => {
    if (!portal) {
      loading = false;
      return;
    }
    refresh();
    api
      .GET("/v1/event-types", { params: { query: { active_only: true } } })
      .then((res) => {
        eventTypes = unwrap(res).items || [];
      })
      .catch(() => {});
  });

  const expired = $derived(portal?.expiresAt != null && portal.expiresAt.getTime() < Date.now());
</script>

<svelte:head>
  <title>Webhook Portal{consumer ? ` — ${consumer}` : ""}</title>
</svelte:head>

<div class="min-h-screen bg-ink text-text">
  <!-- Header -->
  <header class="border-b border-line bg-panel/60 backdrop-blur-sm sticky top-0 z-30">
    <div class="max-w-5xl mx-auto px-4 h-14 flex items-center gap-3">
      <span class="grid place-items-center w-8 h-8 rounded-lg border border-line bg-panel-2">
        <img src={favicon} alt="" class="w-4.5 h-4.5" />
      </span>
      <span class="flex flex-col leading-none">
        <span class="font-display font-bold tracking-[0.16em] text-sm">WEBHOOK PORTAL</span>
        {#if consumer}<span class="eyebrow mt-1" style="font-size:9.5px">{consumer}</span>{/if}
      </span>
      <div class="ml-auto flex items-center gap-3">
        {#if portal?.expiresAt && !expired}
          <span class="chip inline-flex" title={portal.expiresAt.toLocaleString()}>
            Access until {portal.expiresAt.toLocaleDateString()}
          </span>
        {/if}
        <button
          onclick={() => (theme = theme === "dark" ? "light" : "dark")}
          class="grid place-items-center w-8 h-8 rounded-md border border-line text-muted hover:text-text hover:border-line-strong transition-colors"
          title="Toggle theme"
          aria-label="Toggle color theme"
        >
          {#if theme === "dark"}
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true"><circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M4.9 4.9l1.4 1.4m11.4 11.4 1.4 1.4M2 12h2m16 0h2M4.9 19.1l1.4-1.4m11.4-11.4 1.4-1.4"/></svg>
          {:else}
            <svg class="w-4 h-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/></svg>
          {/if}
        </button>
      </div>
    </div>
  </header>

  <main class="max-w-5xl mx-auto px-4 py-6 space-y-5">
    {#if !portal || expired}
      <div class="panel">
        <EmptyState
          icon="link"
          title={expired ? "Access link expired" : "Missing access link"}
          description={expired
            ? "This portal link has expired. Ask for a fresh link to keep managing your webhook endpoints."
            : "Open this page through the personalized portal link you were given — it carries your access token."}
        />
      </div>
    {:else}
      {#if error}
        <div class="panel p-3 flex items-start justify-between" role="alert" aria-live="assertive" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
          <p class="text-sm" style="color:var(--color-bad)">{error}</p>
          <button onclick={() => (error = "")} class="ml-3 shrink-0 text-faint hover:text-text" aria-label="Dismiss error">✕</button>
        </div>
      {/if}

      <!-- Stats -->
      {#if stats}
        <div class="panel readout overflow-hidden">
          <div class="cell"><div class="val">{stats.total_webhooks}</div><div class="key">Endpoints</div></div>
          <div class="cell"><div class="val" style="color:var(--color-ok)">{stats.successful_deliveries}</div><div class="key">Delivered</div></div>
          <div class="cell"><div class="val" style="color:var(--color-bad)">{stats.failed_deliveries}</div><div class="key">Failed</div></div>
          <div class="cell"><div class="val tnum">{Math.round((stats.success_rate ?? 0) * 100)}%</div><div class="key">Success rate</div></div>
        </div>
      {/if}

      <!-- Endpoints -->
      <section class="space-y-3">
        <div class="flex items-center justify-between">
          <h2 class="eyebrow">Your Endpoints</h2>
          <button class="btn btn-beacon" onclick={() => (registerOpen = !registerOpen)}>
            <span class="text-base leading-none">+</span> Add Endpoint
          </button>
        </div>

        {#if registerOpen}
          <form class="panel p-4 space-y-4" onsubmit={registerWebhook}>
            <div>
              <label class="field-label" for="portal-reg-url">Endpoint URL</label>
              <input id="portal-reg-url" class="input" type="url" required placeholder="https://example.com/webhooks" bind:value={regUrl} />
            </div>
            <div>
              <label class="field-label" for="portal-reg-desc">Description (optional)</label>
              <input id="portal-reg-desc" class="input" type="text" placeholder="What is this endpoint for?" bind:value={regDescription} />
            </div>
            {#if eventTypes.length > 0}
              <fieldset>
                <legend class="field-label">Subscribe to events</legend>
                <div class="flex flex-wrap gap-2">
                  {#each eventTypes as et}
                    <label class="chip cursor-pointer select-none {regEvents.includes(et.name) ? '!border-beacon !text-text' : ''}">
                      <input type="checkbox" class="sr-only" value={et.name} bind:group={regEvents} />
                      {et.name}
                    </label>
                  {/each}
                </div>
                <p class="text-xs text-faint mt-2">Leave empty to add subscriptions later.</p>
              </fieldset>
            {/if}
            <div class="flex justify-end gap-2">
              <button type="button" class="btn btn-ghost" onclick={() => (registerOpen = false)}>Cancel</button>
              <button type="submit" class="btn btn-beacon" disabled={regBusy}>{regBusy ? "Adding…" : "Add Endpoint"}</button>
            </div>
          </form>
        {/if}

        {#if loading}
          <div class="panel p-8 text-center text-muted text-sm">Loading…</div>
        {:else if webhooks.length === 0}
          <div class="panel">
            <EmptyState icon="link" title="No endpoints yet" description="Add an HTTPS endpoint to start receiving webhook deliveries." />
          </div>
        {:else}
          <div class="space-y-2">
            {#each webhooks as w (w.webhook_id)}
              <div class="panel overflow-hidden">
                <button
                  type="button"
                  class="w-full flex items-center gap-3 px-4 py-3 text-left row-hover transition"
                  onclick={() => selectWebhook(w.webhook_id)}
                  aria-expanded={selectedId === w.webhook_id}
                >
                  <svg class="w-3.5 h-3.5 shrink-0 text-faint transition-transform {selectedId === w.webhook_id ? 'rotate-90' : ''}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="m9 18 6-6-6-6"/></svg>
                  <span class="min-w-0 flex-1">
                    <span class="block mono text-sm truncate">{w.url}</span>
                    {#if w.description}<span class="block text-xs text-muted truncate mt-0.5">{w.description}</span>{/if}
                  </span>
                  <HealthBadge health={w.health} />
                  {#if !w.active}
                    <span class="chip" style="color:var(--color-warn)">paused</span>
                  {/if}
                </button>

                {#if selectedId === w.webhook_id}
                  <div class="border-t border-line px-4 py-4 space-y-4 bg-panel-2/50">
                    <div class="flex flex-wrap items-center gap-2">
                      <CopyableId id={w.webhook_id} truncate={14} />
                      <span class="flex-1"></span>
                      <button class="btn btn-ghost !px-3 !py-1.5" onclick={() => toggleActive(w)}>
                        {w.active ? "Pause" : "Resume"}
                      </button>
                      <button class="btn btn-danger !px-3 !py-1.5" onclick={() => (confirmDeleteId = w.webhook_id)}>
                        Remove
                      </button>
                    </div>

                    <div class="flex gap-4 border-b border-line">
                      {#each [["deliveries", "Deliveries"], ["subscriptions", "Subscriptions"]] as [id, label]}
                        <button
                          class="pb-2 text-sm font-medium border-b-2 -mb-px transition {detailTab === id ? 'border-beacon text-text' : 'border-transparent text-muted hover:text-text'}"
                          onclick={() => (detailTab = id as typeof detailTab)}
                        >
                          {label}
                        </button>
                      {/each}
                    </div>

                    {#if detailTab === "deliveries"}
                      {#if deliveriesLoading}
                        <p class="text-sm text-muted py-4">Loading deliveries…</p>
                      {:else if deliveries.length === 0}
                        <EmptyState icon="send" title="No deliveries yet" description="Deliveries to this endpoint will appear here." />
                      {:else}
                        <div class="overflow-x-auto -mx-4">
                          <table class="w-full text-left">
                            <thead>
                              <tr>
                                <th class="th">Status</th>
                                <th class="th">Delivery</th>
                                <th class="th hidden sm:table-cell">Attempts</th>
                                <th class="th hidden md:table-cell">Response</th>
                                <th class="th hidden md:table-cell">Created</th>
                                <th class="th"></th>
                              </tr>
                            </thead>
                            <tbody>
                              {#each deliveries as d (d.delivery_id)}
                                <tr class="row-line">
                                  <td class="td"><StatusBadge status={d.status} /></td>
                                  <td class="td"><CopyableId id={d.delivery_id} truncate={12} /></td>
                                  <td class="td hidden sm:table-cell tnum">{d.attempt_count}/{d.max_attempts}</td>
                                  <td class="td hidden md:table-cell mono text-xs">
                                    {d.response_code ?? "—"}
                                    {#if d.error_category}<span class="text-faint"> · {d.error_category}</span>{/if}
                                  </td>
                                  <td class="td hidden md:table-cell text-xs text-muted">{new Date(d.created_at).toLocaleString()}</td>
                                  <td class="td text-right">
                                    {#if d.status === "failed" || d.status === "expired"}
                                      <button class="btn btn-ghost !px-2.5 !py-1 text-xs" onclick={() => retryDelivery(d.delivery_id)}>Retry</button>
                                    {/if}
                                  </td>
                                </tr>
                              {/each}
                            </tbody>
                          </table>
                        </div>
                      {/if}
                    {:else}
                      <SubscriptionManager
                        webhookId={w.webhook_id}
                        {consumer}
                        bind:subscriptions
                        onRefresh={() => fetchSubscriptions(w.webhook_id)}
                      />
                    {/if}
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      </section>
    {/if}
  </main>
</div>

<ConfirmDialog
  open={confirmDeleteId !== null}
  title="Remove Endpoint"
  message="This permanently removes the endpoint, its subscriptions, and its delivery history. This cannot be undone."
  confirmLabel="Remove"
  variant="danger"
  onconfirm={executeDelete}
  oncancel={() => (confirmDeleteId = null)}
/>
