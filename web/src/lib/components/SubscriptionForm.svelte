<script lang="ts">
  // Create/edit a subscription on its own page. Everything about the
  // template lives behind "Edit template", which opens TemplateEditor.
  import { goto } from "$app/navigation";
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import { onMount } from "svelte";
  import TemplateEditor, { type TemplateSaveMeta } from "./TemplateEditor.svelte";
  import type { components } from "$lib/api-types";

  type SubscriptionItem = components["schemas"]["SubscriptionItem"];
  type EventTypeItem = components["schemas"]["EventTypeItem"];

  let {
    webhookId,
    consumer,
    mode,
    subscription = null,
  }: {
    webhookId: string;
    consumer: string;
    mode: "create" | "edit";
    subscription?: SubscriptionItem | null;
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
  let templateChanged = $state(false);
  let editorOpen = $state(false);

  let availableEvents: EventTypeItem[] = $state([]);
  let error = $state("");
  let submitting = $state(false);
  let newHeaderKey = $state(""), newHeaderValue = $state("");
  let newFilterKey = $state(""), newFilterValue = $state("");

  let templatePreviewLines = $derived(transformTemplate.split("\n").slice(0, 4));
  let templateLineCount = $derived(transformTemplate ? transformTemplate.split("\n").length : 0);

  onMount(async () => {
    try {
      const res = unwrap(await api.GET('/v1/event-types', { params: { query: { active_only: true } } }));
      availableEvents = res.items || [];
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

  function onTemplateSaved(t: string, meta: TemplateSaveMeta) {
    templateChanged = templateChanged || t !== transformTemplate;
    transformTemplate = t;
    templateMeta = meta;
    if (t.trim()) transformEnabled = true;
  }

  async function save() {
    error = "";
    const name = catchAll ? CATCH_ALL : eventName.trim();
    if (!name) { error = "Pick an event type or enable catch-all."; return; }
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

      <div>
        <span class="field-label">Label filters</span>
        <p class="text-xs text-faint mb-2">Only events carrying every listed label are delivered. None means every event.</p>
        <div class="flex flex-wrap gap-1 mb-2">
          {#each Object.entries(labelFilters) as [k, v]}
            <span class="chip" style="color:var(--color-warn)">{k}={v} <button type="button" onclick={() => removeFilter(k)} class="ml-1 text-faint hover:text-bad" aria-label="Remove filter {k}">×</button></span>
          {/each}
        </div>
        <div class="flex gap-2">
          <input type="text" placeholder="key" bind:value={newFilterKey} class="input flex-1" />
          <input type="text" placeholder="value" bind:value={newFilterValue} class="input flex-1" />
          <button type="button" onclick={addFilter} class="btn btn-ghost !px-3">+</button>
        </div>
      </div>

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
        <div class="flex flex-wrap gap-1 mb-2">
          {#each Object.entries(headers) as [k, v]}
            <span class="chip">{k}: {v} <button type="button" onclick={() => removeHeader(k)} class="ml-1 text-faint hover:text-bad" aria-label="Remove header {k}">×</button></span>
          {/each}
        </div>
        <div class="flex gap-2">
          <input type="text" placeholder="Header" bind:value={newHeaderKey} class="input flex-1" />
          <input type="text" placeholder="value" bind:value={newHeaderValue} class="input flex-1" />
          <button type="button" onclick={addHeader} class="btn btn-ghost !px-3">+</button>
        </div>
      </div>
    </section>

    <section class="panel p-5 space-y-3">
      <h3 class="eyebrow">Payload transform</h3>
      <div class="flex items-center gap-3">
        <button type="button" onclick={() => (transformEnabled = !transformEnabled)} aria-label="Toggle payload transformation"
          class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors {transformEnabled ? 'bg-ok' : 'bg-line-strong'}">
          <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition {transformEnabled ? 'translate-x-4' : 'translate-x-0'}"></span>
        </button>
        <span class="text-sm text-text">Transform the payload before delivery</span>
      </div>
      {#if transformEnabled}
        <div class="panel-2 p-3" data-testid="template-card">
          <div class="flex items-center justify-between gap-3 mb-2">
            <div class="text-[10px] text-muted flex flex-wrap gap-x-3">
              {#if transformTemplate.trim()}
                <span>{templateLineCount} line{templateLineCount === 1 ? "" : "s"}</span>
                {#if templateMeta}<span style={templateMeta.rendersOk ? 'color:var(--color-ok)' : 'color:var(--color-warn)'}>{templateMeta.rendersOk ? "✓ renders against sample" : "⚠ did not render in the editor"}</span>{/if}
                {#if templateMeta?.source === "ai_draft"}<span>AI draft</span>{/if}
                {#if templateChanged}<span>unsaved</span>{/if}
              {:else}
                <span>No template yet</span>
              {/if}
            </div>
            <button type="button" onclick={() => (editorOpen = true)} class="btn btn-beacon !px-3 !py-1" data-testid="edit-template">{transformTemplate.trim() ? "Edit template" : "Write template"}</button>
          </div>
          {#if transformTemplate.trim()}
            <pre class="text-xs mono text-text overflow-x-auto">{templatePreviewLines.join("\n")}{templateLineCount > 4 ? "\n…" : ""}</pre>
          {:else}
            <p class="text-xs text-faint">Open the editor to write one by hand, draft it with AI, or copy a prompt for any chat assistant.</p>
          {/if}
        </div>
        <p class="text-[10px] text-faint">Nothing is saved until you save the subscription. The editor shows a live render against the event's sample payload.</p>
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-3 pt-1">
          <div>
            <label for="sub-on-error" class="field-label">If the template fails</label>
            <select id="sub-on-error" bind:value={onTransformError} class="select w-full">
              <option value="fail">Fail the delivery (recommended)</option>
              <option value="fallback">Send the default envelope</option>
            </select>
            <p class="text-[10px] text-faint mt-1">{onTransformError === "fail" ? "Nothing is sent and the delivery is marked template_error; retry it after fixing the template." : "The default envelope is sent instead; the template error is still recorded on the delivery."}</p>
          </div>
          <div>
            <label for="sub-missing-key" class="field-label">Missing fields</label>
            <select id="sub-missing-key" bind:value={templateMissingKey} class="select w-full">
              <option value="error">Treat as an error (recommended)</option>
              <option value="zero">Render as &lt;no value&gt;</option>
            </select>
            <p class="text-[10px] text-faint mt-1">{templateMissingKey === "error" ? "A removed field fails the template. Read optional fields with index or dig." : "A missing field renders as <no value> and the delivery goes out."}</p>
          </div>
        </div>
      {:else}
        <p class="text-xs text-faint">The event payload is delivered as-is inside Sparrow's envelope.</p>
      {/if}
    </section>
  </div>

  <div class="flex items-center justify-end gap-2">
    <a href={backHref} class="btn btn-ghost">Cancel</a>
    <button type="button" onclick={save} disabled={submitting} class="btn btn-beacon" data-testid="save-subscription">{submitting ? "Saving…" : mode === "create" ? "Create subscription" : "Save subscription"}</button>
  </div>
</div>

<TemplateEditor
  bind:open={editorOpen}
  template={transformTemplate}
  eventName={catchAll ? CATCH_ALL : eventName}
  {consumer}
  subscriptionId={subscription?.subscription_id ?? ""}
  onSave={onTemplateSaved}
/>
