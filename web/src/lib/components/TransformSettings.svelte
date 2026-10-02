<script lang="ts">
  // The "Payload transform" panel shared by the subscription form and webhook
  // registration: on/off toggle, template card that opens TemplateEditor, and
  // the error-handling settings. Owns no persistence; callers save the values.
  import Disclosure from "./Disclosure.svelte";
  import TemplateEditor, { type TemplateSaveMeta } from "./TemplateEditor.svelte";

  let {
    enabled = $bindable(false),
    template = $bindable(""),
    onError = $bindable("fail"),
    missingKey = $bindable("error"),
    meta = $bindable(null),
    eventName,
    consumer,
    subscriptionId = "",
    disabledReason = "",
    lockedReason = "",
    required = false,
  }: {
    enabled?: boolean;
    template?: string;
    onError?: "fail" | "fallback";
    missingKey?: "error" | "zero";
    meta?: TemplateSaveMeta | null;
    eventName: string;
    consumer: string;
    subscriptionId?: string;
    disabledReason?: string;
    // Set when a transform cannot apply at all (e.g. several events with
    // different schemas): the toggle is off and locked, the template is kept.
    lockedReason?: string;
    // The webhook requires a transform (its receiver only accepts a specific
    // format): no toggle, the transform is always on.
    required?: boolean;
  } = $props();

  $effect(() => {
    if (required && !enabled) enabled = true;
  });

  let editorOpen = $state(false);
  let changed = $state(false);

  let on = $derived(required || (enabled && !lockedReason));
  let lines = $derived(template ? template.split("\n") : []);

  function onSave(t: string, m: TemplateSaveMeta) {
    changed = changed || t !== template;
    template = t;
    meta = m;
    if (t.trim()) enabled = true;
  }
</script>

<section class="panel p-5 space-y-4">
  <h3 class="eyebrow">Payload transform</h3>
  {#if required}
    <p class="text-sm text-text flex items-center gap-2" data-testid="transform-required" title="This webhook's receiver only accepts a transformed payload, so every subscription needs a template.">
      <span class="chip" style="color:var(--color-beacon)">Required</span> This receiver only accepts a transformed payload.
    </p>
  {:else}
  <div class="flex items-center gap-3 {lockedReason ? 'cursor-not-allowed' : ''}" title={lockedReason || undefined}>
    <button type="button" onclick={() => (enabled = !enabled)} disabled={!!lockedReason} aria-label="Toggle payload transformation" aria-pressed={on}
      class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors disabled:cursor-not-allowed disabled:opacity-50 {on ? 'bg-ok' : 'bg-line-strong'}">
      <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition {on ? 'translate-x-4' : 'translate-x-0'}"></span>
    </button>
    <span class="text-sm text-text">Transform the payload before delivery</span>
  </div>
  {/if}
  {#if lockedReason && !required}
    <p class="text-xs text-muted" data-testid="transform-locked">{lockedReason}</p>
  {:else if on}
    <div class="panel-2 p-3" data-testid="template-card">
      <div class="flex items-center justify-between gap-3">
        <div class="text-xs text-muted flex flex-wrap gap-x-3 min-w-0">
          {#if template.trim()}
            <span>{lines.length} line{lines.length === 1 ? "" : "s"}</span>
            {#if meta}<span style={meta.rendersOk ? 'color:var(--color-ok)' : 'color:var(--color-warn)'}>{meta.rendersOk ? "✓ renders against sample" : "⚠ did not render"}</span>{/if}
            {#if meta?.source === "ai_draft"}<span>AI draft</span>{/if}
            {#if changed}<span>not saved yet</span>{/if}
          {:else}
            <span>No template yet</span>
          {/if}
        </div>
        <button type="button" onclick={() => (editorOpen = true)} disabled={!!disabledReason} title={disabledReason} class="btn btn-beacon !px-3 !py-1 shrink-0" data-testid="edit-template">{template.trim() ? "Edit template" : "Write template"}</button>
      </div>
      {#if template.trim()}
        <pre class="mt-2 text-xs mono text-text overflow-x-auto">{lines.slice(0, 4).join("\n")}{lines.length > 4 ? "\n…" : ""}</pre>
      {:else}
        <p class="mt-2 text-xs text-faint">{disabledReason || "Write it by hand or get a draft from AI. The editor previews it against the event's sample payload."}</p>
      {/if}
    </div>
    <Disclosure label="Error handling" summary="{onError === 'fail' ? 'Fail the delivery' : 'Send default envelope'} · {missingKey === 'error' ? 'strict fields' : 'lenient fields'}">
      <div>
        <label for="sub-on-error" class="field-label">If the template fails</label>
        <select id="sub-on-error" bind:value={onError} class="select w-full">
          <option value="fail">Fail the delivery (recommended)</option>
          <option value="fallback">Send the default envelope</option>
        </select>
        <p class="text-xs text-faint mt-1">{onError === "fail" ? "Nothing is sent; fix the template, then retry the delivery." : "The untransformed envelope is sent and the error is recorded on the delivery."}</p>
      </div>
      <div>
        <label for="sub-missing-key" class="field-label">If a field is missing</label>
        <select id="sub-missing-key" bind:value={missingKey} class="select w-full">
          <option value="error">Treat it as an error (recommended)</option>
          <option value="zero">Render it as &lt;no value&gt;</option>
        </select>
        <p class="text-xs text-faint mt-1">{missingKey === "error" ? "Read optional fields with index or dig so they don't fail the template." : "The delivery goes out with <no value> where the field was."}</p>
      </div>
    </Disclosure>
  {:else}
    <p class="text-xs text-faint">The event payload is delivered as-is inside Sparrow's envelope.</p>
  {/if}
</section>

<TemplateEditor bind:open={editorOpen} {template} {eventName} {consumer} {subscriptionId} {missingKey} {onSave} />
