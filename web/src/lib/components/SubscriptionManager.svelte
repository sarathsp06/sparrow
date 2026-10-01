<script lang="ts">
  // Subscriptions list for one webhook. Add and Edit navigate to the
  // subscription page; the template editor lives there.
  import { goto } from "$app/navigation";
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import EmptyState from "./EmptyState.svelte";
  import CopyableId from "./CopyableId.svelte";
  import ConfirmDialog from "./ConfirmDialog.svelte";
  import type { components } from "$lib/api-types";

  type SubscriptionItem = components["schemas"]["SubscriptionItem"];

  let {
    webhookId,
    consumer,
    subscriptions = $bindable([]),
    onRefresh,
  }: {
    webhookId: string;
    consumer: string;
    subscriptions: SubscriptionItem[];
    onRefresh?: () => void;
  } = $props();

  let error: string = $state("");
  let loading = $state(false);
  let confirmDeleteOpen = $state(false);
  let subscriptionToDelete = $state<string | null>(null);

  const CATCH_ALL_EVENT = "*";

  function openCreateModal() {
    goto(`/webhooks/${webhookId}/subscriptions/new`);
  }

  function openEditModal(subscription: SubscriptionItem) {
    goto(`/webhooks/${webhookId}/subscriptions/${subscription.subscription_id}/edit`);
  }

  async function fetchSubscriptions() {
    if (!webhookId) return;
    try {
      loading = true;
      const response = unwrap(await api.GET('/v1/consumers/{consumer}/subscriptions', {
        params: { path: { consumer: consumer || "default" }, query: { webhook_id: webhookId } },
      }));
      subscriptions = response.items || [];
      onRefresh?.();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to fetch subscriptions');
    } finally {
      loading = false;
    }
  }

  // Pause and resume. A paused subscription's deliveries are recorded as
  // paused and not sent; resuming does not send them, so after a resume we
  // offer to retry them.
  let pauseTarget = $state<SubscriptionItem | null>(null);
  let pauseReason = $state("");
  let resumeNotice = $state<{ count: number; since: string; subscriptionId: string } | null>(null);

  async function pauseSubscription() {
    if (!pauseTarget) return;
    try {
      unwrap(await api.POST('/v1/consumers/{consumer}/subscriptions/{subscription_id}:pause', {
        params: { path: { consumer: pauseTarget.consumer, subscription_id: pauseTarget.subscription_id } },
        body: { reason: pauseReason.trim() || undefined },
      }));
      pauseTarget = null;
      pauseReason = "";
      await fetchSubscriptions();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to pause subscription');
    }
  }

  async function resumeSubscription(subscription: SubscriptionItem) {
    try {
      const res = unwrap(await api.POST('/v1/consumers/{consumer}/subscriptions/{subscription_id}:resume', {
        params: { path: { consumer: subscription.consumer, subscription_id: subscription.subscription_id } },
      }));
      resumeNotice = res.paused_deliveries > 0
        ? { count: res.paused_deliveries, since: res.paused_since ?? "", subscriptionId: subscription.subscription_id }
        : null;
      await fetchSubscriptions();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to resume subscription');
    }
  }

  let retryingPaused = $state(false);
  async function retryPaused() {
    if (!resumeNotice) return;
    retryingPaused = true;
    try {
      const ns = consumer || "default";
      const list = unwrap(await api.GET('/v1/consumers/{consumer}/deliveries', {
        params: {
          path: { consumer: ns },
          query: { status: "paused", subscription_id: resumeNotice.subscriptionId, created_after: resumeNotice.since || undefined, prepare_retry: true, limit: 1 },
        },
      }));
      if (list.retry_id) {
        unwrap(await api.POST('/v1/consumers/{consumer}/deliveries:retryBatch', {
          params: { path: { consumer: ns } },
          body: { repush_id: list.retry_id },
        }));
      }
      resumeNotice = null;
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to retry paused deliveries');
    } finally {
      retryingPaused = false;
    }
  }

  function promptDelete(subscriptionId: string) {
    subscriptionToDelete = subscriptionId;
    confirmDeleteOpen = true;
  }

  async function executeDelete() {
    if (!subscriptionToDelete) return;
    try {
      unwrap(await api.DELETE('/v1/consumers/{consumer}/subscriptions/{subscription_id}', {
        params: { path: { consumer: consumer || "default", subscription_id: subscriptionToDelete } },
      }));
      confirmDeleteOpen = false;
      subscriptionToDelete = null;
      await fetchSubscriptions();
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to delete subscription');
      confirmDeleteOpen = false;
    }
  }

  function formatTemplate(template: string) {
    try {
      return JSON.stringify(JSON.parse(template), null, 2);
    } catch {
      return template;
    }
  }

  function formatCreatedAt(createdAt: string | undefined): string {
    if (!createdAt) return "N/A";
    const d = new Date(createdAt);
    return isNaN(d.getTime()) ? "N/A" : d.toLocaleDateString();
  }
</script>

<div>
  <div class="flex items-center justify-between mb-4">
    <h3 class="eyebrow">Event Subscriptions</h3>
    <button
      onclick={openCreateModal}
      class="btn btn-beacon !px-3 !py-1.5"
    >
      <span class="text-lg leading-none">+</span>
      Add Subscription
    </button>
  </div>

  {#if error}
    <div class="panel p-3 mb-4 flex items-start justify-between" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
      <p class="text-sm" style="color:var(--color-bad)">{error}</p>
      <button onclick={() => { error = ""; }} class="ml-3 shrink-0 text-faint hover:text-bad transition-colors" aria-label="Dismiss error">
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24" aria-hidden="true">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>
    </div>
  {/if}

  {#if loading}
    <div class="space-y-3">
      {#each Array(3) as _}
        <div class="panel p-4 animate-pulse">
          <div class="flex items-center gap-2 mb-2">
          <div class="h-4 bg-black/5 rounded w-32"></div>
            <div class="h-4 bg-black/[0.03] rounded w-16"></div>
            <div class="h-4 bg-black/[0.03] rounded w-12"></div>
          </div>
          <div class="flex gap-4">
            <div class="h-3 bg-black/[0.03] rounded w-24"></div>
            <div class="h-3 bg-black/[0.03] rounded w-32"></div>
          </div>
        </div>
      {/each}
    </div>
  {:else if subscriptions.length === 0}
    <div class="panel">
      <EmptyState icon="link" title="No subscriptions yet" description="Create subscriptions to define which events this webhook receives and how payloads are transformed.">
        {#snippet action()}
          <button onclick={openCreateModal} class="btn btn-beacon">
            Create First Subscription
          </button>
        {/snippet}
      </EmptyState>
    </div>
  {:else}
    {#if resumeNotice}
      <div class="panel-2 p-3 mb-3 flex flex-wrap items-center justify-between gap-3">
        <p class="text-sm text-text">
          {resumeNotice.count === 1
            ? "1 delivery was held while paused. It is not sent automatically."
            : `${resumeNotice.count} deliveries were held while paused. They are not sent automatically.`}
        </p>
        <div class="flex gap-2">
          <button class="btn btn-ghost !px-3 !py-1.5" onclick={() => (resumeNotice = null)}>Leave them</button>
          <button class="btn btn-beacon !px-3 !py-1.5" disabled={retryingPaused} onclick={retryPaused}>Retry paused deliveries</button>
        </div>
      </div>
    {/if}
    {#if pauseTarget}
      <div class="panel-2 p-4 mb-3 space-y-3">
        <p class="text-sm text-text">
          Pause <span class="mono">{pauseTarget.event_name}</span>? Events keep arriving and are recorded as paused deliveries, not sent. The webhook's health is not affected.
        </p>
        <label class="block">
          <span class="field-label">Reason (optional)</span>
          <input class="input" maxlength="500" placeholder="Receiver maintenance until Friday" bind:value={pauseReason} />
        </label>
        <div class="flex justify-end gap-2">
          <button class="btn btn-ghost !px-3 !py-1.5" onclick={() => (pauseTarget = null)}>Cancel</button>
          <button class="btn btn-danger !px-3 !py-1.5" onclick={pauseSubscription}>Pause subscription</button>
        </div>
      </div>
    {/if}
    <div class="space-y-3">
      {#each subscriptions as subscription}
        <div class="panel-2 hover:border-line-strong transition">
          <div class="p-4">
            <div class="flex items-start justify-between gap-4">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 mb-2 flex-wrap">
                  <h3 class="text-sm font-semibold text-text">
                    {#if subscription.event_name === CATCH_ALL_EVENT}
                      All Events
                    {:else}
                      {subscription.event_name}
                    {/if}
                  </h3>
                  {#if subscription.event_name === CATCH_ALL_EVENT}
                    <span class="chip" style="color:var(--color-beacon);border-color:color-mix(in srgb,var(--color-beacon) 35%,transparent);background:color-mix(in srgb,var(--color-beacon) 12%,var(--color-panel))">
                      Catch-All
                    </span>
                  {/if}
                  <span class="chip">
                    {subscription.consumer}
                  </span>
                  <span class="chip">
                    {subscription.method || "POST"}
                  </span>
                  {#if subscription.transform_enabled}
                    <span class="chip" style="color:var(--color-ok);border-color:color-mix(in srgb,var(--color-ok) 35%,transparent);background:color-mix(in srgb,var(--color-ok) 12%,var(--color-panel))">
                      Template
                    </span>
                    {#if subscription.on_transform_error === "fallback"}
                      <span class="chip" title="If the template fails, the default envelope is sent instead">Fallback on error</span>
                    {/if}
                  {/if}
                  {#if subscription.paused}
                    <span class="chip" style="color:var(--color-warn);border-color:color-mix(in srgb,var(--color-warn) 35%,transparent);background:color-mix(in srgb,var(--color-warn) 12%,var(--color-panel))">
                      Paused
                    </span>
                  {/if}
                </div>
                {#if subscription.paused}
                  <p class="text-xs text-muted mb-2">
                    Paused {subscription.paused_at ? formatCreatedAt(subscription.paused_at) : ""}{subscription.paused_reason ? `: ${subscription.paused_reason}` : ""}. New deliveries are recorded as paused and not sent.
                  </p>
                {/if}

                <div class="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-muted mono tnum">
                  <span>Timeout: {subscription.timeout || 30}s</span>
                  <span>Created: {formatCreatedAt(subscription.created_at)}</span>
                  <CopyableId id={subscription.subscription_id} truncate={12} />
                </div>

                {#if subscription.transform_enabled && subscription.transform_template}
                  <details class="mt-3">
                    <summary class="text-xs font-medium text-muted cursor-pointer hover:text-text select-none">
                      View Template
                    </summary>
                    <pre class="mt-1.5 panel-2 p-3 text-xs overflow-x-auto max-h-32 mono text-text">{formatTemplate(subscription.transform_template)}</pre>
                  </details>
                {/if}

                {#if Object.keys(subscription.headers || {}).length > 0}
                  <div class="mt-2 flex flex-wrap gap-1">
                    {#each Object.entries(subscription.headers || {}) as [key, value]}
                      <span class="chip">
                        {key}: {value}
                      </span>
                    {/each}
                  </div>
                {/if}

                {#if Object.keys(subscription.label_filters || {}).length > 0}
                  <div class="mt-2 flex flex-wrap items-center gap-1">
                    <span class="text-xs text-muted font-medium mr-1">Filters:</span>
                    {#each Object.entries(subscription.label_filters || {}) as [key, value]}
                      <span class="chip" style="color:var(--color-warn);border-color:color-mix(in srgb,var(--color-warn) 35%,transparent);background:color-mix(in srgb,var(--color-warn) 12%,var(--color-panel))">
                        {key}={value}
                      </span>
                    {/each}
                  </div>
                {:else}
                  <div class="mt-2">
                    <span class="text-xs text-faint italic">No label filters — matches all events</span>
                  </div>
                {/if}
              </div>

              <div class="flex items-center gap-1.5 shrink-0">
                <button onclick={() => openEditModal(subscription)} class="btn btn-ghost !px-3 !py-1.5">
                  Edit
                </button>
                {#if subscription.paused}
                  <button onclick={() => resumeSubscription(subscription)} class="btn btn-ghost !px-3 !py-1.5">Resume</button>
                {:else}
                  <button onclick={() => { pauseTarget = subscription; pauseReason = ""; }} class="btn btn-ghost !px-3 !py-1.5">Pause</button>
                {/if}
                <button onclick={() => promptDelete(subscription.subscription_id)} class="btn btn-danger !px-3 !py-1.5">
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}
</div>

<ConfirmDialog
  open={confirmDeleteOpen}
  title="Delete Subscription"
  message="This will permanently remove this event subscription. The webhook will no longer receive events for this subscription."
  confirmLabel="Delete"
  variant="danger"
  onconfirm={executeDelete}
  oncancel={() => {
    confirmDeleteOpen = false;
    subscriptionToDelete = null;
  }}
/>
