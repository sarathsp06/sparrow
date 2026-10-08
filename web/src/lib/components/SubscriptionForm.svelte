<script lang="ts">
  // Create/edit a subscription on its own page. Everything about the
  // template lives behind "Edit template", which opens TemplateEditor.
  import { goto } from "$app/navigation";
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import { listAllEventTypes } from "$lib/system";
  import { onMount } from "svelte";
  import Disclosure from "./Disclosure.svelte";
  import TransformSettings from "./TransformSettings.svelte";
  import type { TemplateSaveMeta } from "./TemplateEditor.svelte";
  import type { components } from "$lib/api-types";

  type SubscriptionItem = components["schemas"]["SubscriptionItem"];
  type EventTypeItem = components["schemas"]["EventTypeItem"];

  let {
    webhookId,
    consumer,
    mode,
    subscription = null,
    requiresTransform = false,
  }: {
    webhookId: string;
    consumer: string;
    mode: "create" | "edit";
    subscription?: SubscriptionItem | null;
    requiresTransform?: boolean;
  } = $props();

  const CATCH_ALL = "*";
  const backHref = $derived(`/webhooks/${webhookId}?tab=subscriptions`);

  let eventName = $state(subscription?.event_name ?? "");
  let catchAll = $state(subscription?.event_name === CATCH_ALL);
  let method = $state(subscription?.method || "POST");
  let timeout = $state(subscription?.timeout || 30);
  let headers = $state<Record<string, string>>({ ...(subscription?.headers || {}) });
  let labelFilters = $state<Record<string, string>>({ ...(subscription?.label_filters || {}) });
  let transformEnabled = $state(subscription?.transform_enabled ?? false);
  let transformTemplate = $state(subscription?.transform_template || "");
  let onTransformError = $state<"fail" | "fallback">((subscription?.on_transform_error as "fail" | "fallback") ?? "fail");
  let templateMissingKey = $state<"error" | "zero">((subscription?.template_missing_key as "error" | "zero") ?? "error");
  let templateMeta: TemplateSaveMeta | null = $state(null);

  let availableEvents: EventTypeItem[] = $state([]);
  let error = $state("");
  let submitting = $state(false);
  let newHeaderKey = $state(""), newHeaderValue = $state("");
  let newFilterKey = $state(""), newFilterValue = $state("");

  let filterCount = $derived(Object.keys(labelFilters).length);
  let headerCount = $derived(Object.keys(headers).length);
  // Collapsed sections start open only when they hold non-default values.
  const filtersOpen = Object.keys(subscription?.label_filters || {}).length > 0;
  const deliveryOpen = Object.keys(subscription?.headers || {}).length > 0
    || (subscription?.method || "POST") !== "POST" || (subscription?.timeout || 30) !== 30;


  onMount(async () => {
    try {
      // Only the events this consumer can subscribe to (sparrow.* for _sparrow).
      availableEvents = await listAllEventTypes({ active_only: true, consumer });
    } catch (e: any) {
      error = formatAPIError(e, "Failed to load event types");
    }
  });

  function addHeader() {
    if (!newHeaderKey.trim() || !newHeaderValue.trim()) return;
    headers = { ...headers, [newHeaderKey.trim()]: newHeaderValue.trim() };
    newHeaderKey = ""; newHeaderValue = "";
  }
  function removeHeader(k: string) { const { [k]: _, ...rest } = headers; headers = rest; }
  function addFilter() {
    if (!newFilterKey.trim() || !newFilterValue.trim()) return;
    labelFilters = { ...labelFilters, [newFilterKey.trim()]: newFilterValue.trim() };
    newFilterKey = ""; newFilterValue = "";
  }
  function removeFilter(k: string) { const { [k]: _, ...rest } = labelFilters; labelFilters = rest; }

  async function save() {
    error = "";
    const name = catchAll ? CATCH_ALL : eventName.trim();
    if (!name) { error = "Pick an event type or enable catch-all."; return; }
    if (requiresTransform && !transformTemplate.trim()) { error = "This webhook requires a payload transform: write a template before saving."; return; }
    try {
      submitting = true;
      if (mode === "create") {
        unwrap(await api.POST('/v1/consumers/{consumer}/subscriptions', {
          params: { path: { consumer } },
          body: {
            webhook_id: webhookId,
            event_name: name,
            headers,
            method,
            timeout,
            transform_enabled: transformEnabled,
            transform_template: transformTemplate,
            label_filters: labelFilters,
            on_transform_error: onTransformError,
            template_missing_key: templateMissingKey,
          },
        }));
      } else if (subscription) {
        unwrap(await api.PATCH('/v1/consumers/{consumer}/subscriptions/{subscription_id}', {
          params: { path: { consumer, subscription_id: subscription.subscription_id } },
          body: {
            headers,
            method,
            timeout,
            transform_enabled: transformEnabled,
            transform_template: transformTemplate,
            label_filters: labelFilters,
            on_transform_error: onTransformError,
            template_missing_key: templateMissingKey,
            template_source: templateMeta?.source ?? "manual",
            template_notes: templateMeta?.notes || undefined,
          },
        }));
      }
      await goto(backHref);
    } catch (e: any) {
      error = formatAPIError(e, `Failed to ${mode === "create" ? "create" : "update"} subscription`);
    } finally {
      submitting = false;
    }
  }
</script>

<div class="space-y-4">
  {#if error}
    <div class="panel p-3" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
      <p class="text-sm" style="color:var(--color-bad)">{error}</p>
    </div>
  {/if}

  <div class="grid gap-4 lg:grid-cols-2 items-start">
    <section class="panel p-5 space-y-4">
      <h3 class="eyebrow">Subscription</h3>
      <div>
        <label for="sub-event" class="field-label">Event type</label>
        {#if mode === "create"}
          <div class="flex items-center gap-3 mb-2">
            <button type="button" onclick={() => { catchAll = !catchAll; if (catchAll) eventName = CATCH_ALL; else if (eventName === CATCH_ALL) eventName = ""; }} aria-label="Toggle catch-all"
              class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors {catchAll ? 'bg-ok' : 'bg-line-strong'}">
              <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition {catchAll ? 'translate-x-4' : 'translate-x-0'}"></span>
            </button>
            <span class="text-sm text-text">Catch-all: receive every event in this consumer</span>
          </div>
          {#if !catchAll}
            <select id="sub-event" bind:value={eventName} class="select w-full">
              <option value="">Select an event type…</option>
              {#each availableEvents as ev}<option value={ev.name}>{ev.name}</option>{/each}
            </select>
          {/if}
        {:else}
          <input id="sub-event" type="text" value={catchAll ? "All events (catch-all)" : eventName} disabled class="input opacity-60 w-full" />
          <p class="text-xs text-faint mt-1">The event type cannot be changed. Delete and recreate the subscription instead.</p>
        {/if}
      </div>

      <Disclosure label="Label filters" summary={filterCount ? `${filterCount} filter${filterCount === 1 ? "" : "s"}` : "Every event"} open={filtersOpen}>
        <div>
          <p class="text-xs text-faint mb-2">Deliver only events carrying every label listed here.</p>
          {#if filterCount}
            <div class="flex flex-wrap gap-1 mb-2">
              {#each Object.entries(labelFilters) as [k, v]}
                <span class="chip" style="color:var(--color-warn)">{k}={v} <button type="button" onclick={() => removeFilter(k)} class="ml-1 text-faint hover:text-bad" aria-label="Remove filter {k}">×</button></span>
              {/each}
            </div>
          {/if}
          <div class="flex gap-2">
            <input type="text" placeholder="key" bind:value={newFilterKey} class="input flex-1 min-w-0" />
            <input type="text" placeholder="value" bind:value={newFilterValue} class="input flex-1 min-w-0" />
            <button type="button" onclick={addFilter} class="btn btn-ghost !px-3" aria-label="Add label filter">+</button>
          </div>
        </div>
      </Disclosure>

      <Disclosure label="Delivery options" summary="{method} · {timeout}s timeout{headerCount ? ` · ${headerCount} header${headerCount === 1 ? '' : 's'}` : ''}" open={deliveryOpen}>
        <div class="space-y-4">
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label for="sub-method" class="field-label">Method</label>
              <select id="sub-method" bind:value={method} class="select w-full">
                {#each ["POST", "PUT", "PATCH"] as m}<option value={m}>{m}</option>{/each}
              </select>
            </div>
            <div>
              <label for="sub-timeout" class="field-label">Timeout (seconds)</label>
              <input id="sub-timeout" type="number" min="1" max="300" bind:value={timeout} class="input w-full" />
            </div>
          </div>
          <div>
            <span class="field-label">Extra headers</span>
            {#if headerCount}
              <div class="flex flex-wrap gap-1 mb-2">
                {#each Object.entries(headers) as [k, v]}
                  <span class="chip">{k}: {v} <button type="button" onclick={() => removeHeader(k)} class="ml-1 text-faint hover:text-bad" aria-label="Remove header {k}">×</button></span>
                {/each}
              </div>
            {/if}
            <div class="flex gap-2">
              <input type="text" placeholder="Header" bind:value={newHeaderKey} class="input flex-1 min-w-0" />
              <input type="text" placeholder="value" bind:value={newHeaderValue} class="input flex-1 min-w-0" />
              <button type="button" onclick={addHeader} class="btn btn-ghost !px-3" aria-label="Add header">+</button>
            </div>
          </div>
        </div>
      </Disclosure>
    </section>

    <TransformSettings
      bind:enabled={transformEnabled}
      bind:template={transformTemplate}
      bind:onError={onTransformError}
      bind:missingKey={templateMissingKey}
      bind:meta={templateMeta}
      eventName={catchAll ? CATCH_ALL : eventName}
      {consumer}
      subscriptionId={subscription?.subscription_id ?? ""}
      required={requiresTransform}
    />
  </div>

  <div class="flex items-center justify-end gap-2">
    <a href={backHref} class="btn btn-ghost">Cancel</a>
    <button type="button" onclick={save} disabled={submitting} class="btn btn-beacon" data-testid="save-subscription">{submitting ? "Saving…" : mode === "create" ? "Create subscription" : "Save subscription"}</button>
  </div>
</div>

