<script lang="ts">
  // Full-viewport template editor: editor pane, live-rendered pane, a toolbar
  // with the AI drawer (draft or copy-prompt), the helper reference, the
  // Workbench link, and the subscription's saved template history. It edits a
  // copy and hands the result back through onSave; nothing is persisted here.
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import type { components } from "$lib/api-types";

  type EventTypeItem = components["schemas"]["EventTypeItem"];
  type RecipeItem = components["schemas"]["Recipe"];
  type VersionItem = components["schemas"]["TemplateVersionItem"];
  export type TemplateSaveMeta = { source: "manual" | "ai_draft"; notes: string; rendersOk: boolean };

  let {
    open = $bindable(false),
    template = "",
    eventName = "",
    consumer = "default",
    subscriptionId = "",
    onSave,
  }: {
    open?: boolean;
    template?: string;
    eventName?: string;
    consumer?: string;
    subscriptionId?: string;
    onSave: (template: string, meta: TemplateSaveMeta) => void;
  } = $props();

  const CATCH_ALL = "*";
  const WORKBENCH_URL = "https://sarathsp06.github.io/sparrow/tools/recipe-workbench/";

  let draft = $state("");
  let source: "manual" | "ai_draft" = $state("manual");
  let sourceNotes = $state("");
  let eventDetails: EventTypeItem | null = $state(null);
  let rendered = $state("");
  let renderError = $state("");
  let rendering = $state(false);
  let strict = $state(true);
  let viewAs: "raw" | "json" = $state("json");
  let renderTimer: ReturnType<typeof setTimeout> | null = null;

  // Helper reference
  let showFunctions = $state(false);
  let functions: { name: string; description: string }[] = $state([]);
  let selectedFunction: string | null = $state(null);

  // History
  let versions: VersionItem[] = $state([]);
  let showHistory = $state(false);

  // AI drawer
  let aiCapability = $state<{ enabled: boolean; model?: string; provider?: string; prompt_only?: boolean }>({ enabled: false });
  let aiOpen = $state(false);
  let aiInstructions = $state("");
  let aiRecipe = $state("");
  let aiTargetExample = $state("");
  let aiShowTarget = $state(false);
  let aiDocsUrl = $state("");
  let aiSamplePayload = $state("");
  let aiRefine = $state(false);
  let aiLoading = $state(false);
  let aiNotes = $state("");
  let aiError = $state("");
  let aiAttempts = $state(0);
  let aiSampleSource = $state("");
  let aiConfirmReplace = $state(false);
  let preDraft: { template: string; rendered: string; error: string } | null = $state(null);
  let recipeOptions: RecipeItem[] = $state([]);
  let promptCopied = $state(false);
  let promptText = $state("");
  let promptLoading = $state(false);

  let canRender = $derived(!!eventName && eventName !== CATCH_ALL);
  let registeredSampleText = $derived(JSON.stringify((eventDetails as EventTypeItem | null)?.sample_payload ?? {}, null, 2));
  let aiSampleEdited = $derived.by(() => {
    try { return JSON.stringify(JSON.parse(aiSamplePayload)) !== JSON.stringify((eventDetails as EventTypeItem | null)?.sample_payload ?? {}); }
    catch { return true; }
  });
  let workbenchUrl = $derived.by(() => {
    const src = (eventDetails as EventTypeItem | null)?.sample_payload ?? (eventDetails as EventTypeItem | null)?.event_schema;
    if (!src) return WORKBENCH_URL;
    const b64 = btoa(unescape(encodeURIComponent(JSON.stringify(src))));
    return `${WORKBENCH_URL}?sample=${encodeURIComponent(b64)}`;
  });
  let prettyRendered = $derived.by(() => {
    if (viewAs !== "json" || !rendered) return rendered;
    try { return JSON.stringify(JSON.parse(rendered), null, 2); } catch { return rendered; }
  });
  let renderedIsJson = $derived.by(() => { try { JSON.parse(rendered); return true; } catch { return false; } });

  // Reset working state every time the editor opens.
  $effect(() => {
    if (!open) return;
    draft = template;
    source = "manual";
    sourceNotes = "";
    rendered = "";
    renderError = "";
    aiOpen = false;
    aiNotes = "";
    aiError = "";
    aiAttempts = 0;
    aiConfirmReplace = false;
    preDraft = null;
    promptCopied = false;
    promptText = "";
    aiRefine = !!template.trim();
    aiSamplePayload = "";
    void loadContext();
  });

  async function loadContext() {
    try {
      const caps = unwrap(await api.GET('/v1/capabilities'));
      aiCapability = caps.ai_drafting ?? { enabled: false };
    } catch { aiCapability = { enabled: false }; }
    if (canRender) {
      try {
        eventDetails = unwrap(await api.GET('/v1/event-types/{name}', { params: { path: { name: eventName } } }));
      } catch { eventDetails = null; }
    } else {
      eventDetails = null;
    }
    if (subscriptionId) {
      try {
        const res = unwrap(await api.GET('/v1/consumers/{consumer}/subscriptions/{subscription_id}/templateVersions', {
          params: { path: { consumer, subscription_id: subscriptionId } },
        }));
        versions = res.items || [];
      } catch { versions = []; }
    } else {
      versions = [];
    }
    scheduleRender();
  }

  // Live render on a short debounce, through the same dry-run a delivery uses.
  $effect(() => {
    void draft; void strict;
    if (open) scheduleRender();
  });

  function scheduleRender() {
    if (renderTimer) clearTimeout(renderTimer);
    renderTimer = setTimeout(renderNow, 400);
  }

  async function renderNow() {
    if (!open) return;
    if (!canRender) { rendered = ""; renderError = ""; return; }
    if (!draft.trim()) { rendered = ""; renderError = ""; return; }
    try {
      rendering = true;
      const res = unwrap(await api.POST('/v1/subscriptions:testTemplate', {
        body: { event_name: eventName, template: draft, template_missing_key: strict ? "error" : "zero" },
      }));
      rendered = res.rendered;
      renderError = "";
    } catch (e: any) {
      rendered = "";
      renderError = formatAPIError(e, "Template did not render");
    } finally {
      rendering = false;
    }
  }

  async function toggleFunctions() {
    showFunctions = !showFunctions;
    if (showFunctions && functions.length === 0) {
      try { functions = unwrap(await api.GET('/v1/template-functions')).items || []; } catch { functions = []; }
    }
  }

  function summary(description: string): string {
    const lines = description.split("\n").map((l) => l.trim());
    return lines.find((l) => l && !l.startsWith("#")) ?? "";
  }

  async function toggleAI() {
    aiOpen = !aiOpen;
    if (aiOpen && !aiSamplePayload.trim()) aiSamplePayload = registeredSampleText;
    if (aiOpen && recipeOptions.length === 0) {
      try { recipeOptions = (unwrap(await api.GET('/v1/recipes')).items || []).filter((r) => r.subscription?.transform_template); }
      catch { recipeOptions = []; }
    }
  }

  function aiRequestBody() {
    let samplePayload: Record<string, unknown> | undefined;
    if (aiSamplePayload.trim()) {
      try {
        const parsed = JSON.parse(aiSamplePayload);
        if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) throw new Error("not an object");
        if (aiSampleEdited) samplePayload = parsed;
      } catch {
        aiError = "The sample payload must be a JSON object, e.g. {\"id\": \"ord_1\"}.";
        return null;
      }
    }
    return {
      event_name: eventName,
      instructions: aiInstructions.trim(),
      recipe: aiRecipe || undefined,
      target_example: aiTargetExample.trim() || undefined,
      docs_url: aiDocsUrl.trim() || undefined,
      sample_payload: samplePayload,
      current_template: aiRefine && draft.trim() ? draft : undefined,
    };
  }

  async function draftWithAI(confirmed = false) {
    if (!canRender || !aiInstructions.trim()) return;
    aiError = "";
    if (!confirmed && !aiRefine && draft.trim()) { aiConfirmReplace = true; return; }
    aiConfirmReplace = false;
    const body = aiRequestBody();
    if (!body) return;
    try {
      aiLoading = true;
      aiNotes = "";
      const res = unwrap(await api.POST('/v1/subscriptions:draftTemplate', { body }));
      if (draft.trim() && draft !== res.template) preDraft = { template: draft, rendered, error: renderError };
      draft = res.template;
      rendered = res.rendered;
      renderError = "";
      aiNotes = res.notes || "";
      aiAttempts = res.attempts;
      aiSampleSource = res.sample_source;
      aiRefine = true;
      source = "ai_draft";
      sourceNotes = res.notes || "";
    } catch (e: any) {
      aiError = formatAPIError(e, "Failed to draft template");
    } finally {
      aiLoading = false;
    }
  }

  function undoDraft() {
    if (!preDraft) return;
    draft = preDraft.template;
    rendered = preDraft.rendered;
    renderError = preDraft.error;
    preDraft = null;
    aiNotes = "";
    source = "manual";
    sourceNotes = "";
  }

  async function copyPrompt() {
    if (!canRender || !aiInstructions.trim()) return;
    aiError = "";
    const body = aiRequestBody();
    if (!body) return;
    try {
      promptLoading = true;
      promptCopied = false;
      const res = unwrap(await api.POST('/v1/subscriptions:draftTemplatePrompt', { body }));
      promptText = res.prompt;
      try { await navigator.clipboard.writeText(res.prompt); promptCopied = true; } catch { promptCopied = false; }
    } catch (e: any) {
      aiError = formatAPIError(e, "Failed to build prompt");
    } finally {
      promptLoading = false;
    }
  }

  function loadVersion(v: VersionItem) {
    if (draft.trim() && draft !== v.template) preDraft = { template: draft, rendered, error: renderError };
    draft = v.template;
    source = "manual";
    sourceNotes = `Loaded version from ${new Date(v.created_at).toLocaleString()}`;
  }

  function save() {
    onSave(draft, { source, notes: sourceNotes, rendersOk: !renderError && (!!rendered || !draft.trim()) });
    open = false;
  }

  function cancel() {
    open = false;
  }

  function formatWhen(iso: string): string {
    const d = new Date(iso);
    return isNaN(d.getTime()) ? iso : d.toLocaleString();
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_no_static_element_interactions -->
  <div class="fixed inset-0 z-50 flex flex-col bg-panel" role="dialog" aria-modal="true" aria-label="Template editor" tabindex="-1" data-testid="template-editor"
       onkeydown={(e) => { if (e.key === "Escape") cancel(); }}>
    <div class="flex items-center justify-between gap-3 px-4 py-3 border-b border-line shrink-0">
      <div class="min-w-0">
        <span class="eyebrow">Transform template</span>
        <p class="text-sm text-text truncate">{canRender ? eventName : "Catch-all subscription (no single event to render against)"}</p>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <button type="button" onclick={cancel} class="btn btn-ghost !px-3 !py-1.5">Cancel</button>
        <button type="button" onclick={save} class="btn btn-beacon !px-3 !py-1.5" data-testid="use-template">Use this template</button>
      </div>
    </div>

    <div class="flex flex-wrap items-center gap-3 px-4 py-2 border-b border-line shrink-0 text-xs">
      <button type="button" onclick={toggleAI} disabled={!canRender} class="font-medium transition disabled:opacity-50 {aiOpen ? '' : 'text-muted hover:text-text'}" style={aiOpen ? 'color:var(--color-beacon)' : ''} data-testid="ai-toggle">
        ✦ {aiOpen ? "Hide" : aiCapability.enabled ? "Draft with AI" : "Copy prompt for AI"}
      </button>
      <button type="button" onclick={toggleFunctions} class="font-medium text-muted hover:text-text transition">ƒ {showFunctions ? "Hide functions" : "Functions"}</button>
      {#if subscriptionId}
        <button type="button" onclick={() => (showHistory = !showHistory)} class="font-medium text-muted hover:text-text transition">⟲ History{versions.length ? ` (${versions.length})` : ""}</button>
      {/if}
      <a href={workbenchUrl} target="_blank" rel="noopener" class="font-medium text-muted hover:text-text transition">Open in Workbench ↗</a>
      <div class="ml-auto flex items-center gap-3">
        <label class="flex items-center gap-1.5 text-muted cursor-pointer" title="Fail when the template reads a payload field the sample does not have">
          <input type="checkbox" bind:checked={strict} /> Strict missing keys
        </label>
        <span class="text-faint">View as</span>
        <div class="flex rounded border border-line overflow-hidden">
          <button type="button" onclick={() => (viewAs = "raw")} class="px-2 py-0.5 {viewAs === 'raw' ? 'bg-line text-text' : 'text-muted hover:text-text'}">raw</button>
          <button type="button" onclick={() => (viewAs = "json")} class="px-2 py-0.5 {viewAs === 'json' ? 'bg-line text-text' : 'text-muted hover:text-text'}">JSON</button>
        </div>
      </div>
    </div>

    {#if aiOpen}
      <div class="px-4 py-3 border-b border-line shrink-0 space-y-2 max-h-[45vh] overflow-y-auto" data-testid="ai-draft-panel" style="background:color-mix(in srgb,var(--color-beacon) 6%,var(--color-panel))">
        <div class="flex items-center justify-between">
          <span class="eyebrow">{aiCapability.enabled ? "Draft with AI" : "Prompt for AI"}</span>
          <span class="text-[10px] text-faint">{aiCapability.enabled ? aiCapability.model : "No AI provider on this server — copy the prompt into any chat assistant"}</span>
        </div>
        <div class="grid gap-3 lg:grid-cols-2">
          <div>
            <div class="flex items-center justify-between">
              <label for="ai-sample" class="text-[10px] text-muted font-medium">Sample <code class="mono">{eventName}</code> payload the draft is written and checked against</label>
              {#if aiSampleEdited}
                <button type="button" onclick={() => (aiSamplePayload = registeredSampleText)} class="text-[10px] font-medium text-muted hover:text-text transition">Reset to registered sample</button>
              {/if}
            </div>
            <textarea id="ai-sample" bind:value={aiSamplePayload} rows="6" class="input mono !text-xs w-full" style="resize:vertical"></textarea>
            <p class="text-[10px] text-faint mt-0.5">Edit values so they show what they really are: a field the schema only calls a string might be an email, a currency code, or an ISO date. Used for this draft only.</p>
          </div>
          <div class="space-y-2">
            <textarea bind:value={aiInstructions} rows="3" placeholder="Describe what the receiver should get, e.g. “a short Slack message with the order id and total, prefixed with ⚠️ when status is refunded”" class="input !text-xs w-full" style="resize:vertical" data-testid="ai-instructions"></textarea>
            <div class="grid grid-cols-2 gap-2">
              <div>
                <label for="ai-recipe" class="text-[10px] text-muted font-medium">Destination format</label>
                <select id="ai-recipe" bind:value={aiRecipe} class="select w-full !text-xs">
                  <option value="">Custom receiver (JSON)</option>
                  {#each recipeOptions as r}<option value={r.name}>{r.name} — {r.description}</option>{/each}
                </select>
              </div>
              <div class="flex flex-col justify-end gap-1 text-[10px] text-muted">
                <button type="button" onclick={() => (aiShowTarget = !aiShowTarget)} class="text-left font-medium hover:text-text transition">{aiShowTarget ? "− Hide" : "+ Describe"} what the receiver expects (example, description, or docs link)</button>
                {#if draft.trim()}
                  <label class="flex items-center gap-1.5 cursor-pointer"><input type="checkbox" bind:checked={aiRefine} /> Refine the current template instead of starting over</label>
                {/if}
              </div>
            </div>
            {#if aiShowTarget}
              <textarea bind:value={aiTargetExample} rows="4" placeholder={'Paste an example body the receiver accepts, or describe its shape in words'} class="input mono !text-xs w-full" style="resize:vertical"></textarea>
              <div>
                <label for="ai-docs-url" class="text-[10px] text-muted font-medium">Receiver documentation URL (optional)</label>
                <input id="ai-docs-url" type="url" bind:value={aiDocsUrl} placeholder="https://docs.receiver.example/webhooks#payload" class="input !text-xs w-full" />
                <p class="text-[10px] text-faint mt-0.5">Sparrow reads the page and gives the model an excerpt. Same network rules as deliveries apply.</p>
              </div>
            {/if}
            <div class="flex items-center justify-between gap-3">
              {#if aiCapability.enabled}
                <p class="text-[10px] text-faint">Only the schema, the sample, and what you type here go to the model. Never stored events, headers, or secrets.</p>
                <button type="button" onclick={() => draftWithAI()} disabled={aiLoading || !aiInstructions.trim()} class="btn btn-beacon !px-3 !py-1 shrink-0" data-testid="ai-draft">
                  {aiLoading ? "Drafting…" : aiRefine && draft.trim() ? "Refine template" : "Draft template"}
                </button>
              {:else}
                <p class="text-[10px] text-faint">Sparrow sends nothing anywhere. Paste the prompt into any chat assistant, then paste the template it returns into the editor.</p>
                <button type="button" onclick={copyPrompt} disabled={promptLoading || !aiInstructions.trim()} class="btn btn-beacon !px-3 !py-1 shrink-0" data-testid="copy-prompt">
                  {promptLoading ? "Building…" : promptCopied ? "Copied ✓" : "Copy prompt"}
                </button>
              {/if}
            </div>
            {#if aiConfirmReplace}
              <div class="flex items-center justify-between gap-3 panel p-2 rounded" data-testid="ai-confirm-replace">
                <p class="text-xs text-text">Replace the current template with a fresh draft? Tick "Refine" instead to build on it. You can undo the draft until you save.</p>
                <div class="flex gap-2 shrink-0">
                  <button type="button" onclick={() => (aiConfirmReplace = false)} class="btn btn-ghost !px-3 !py-1">Keep</button>
                  <button type="button" onclick={() => draftWithAI(true)} class="btn btn-beacon !px-3 !py-1">Replace</button>
                </div>
              </div>
            {/if}
            {#if aiNotes}
              <p class="text-xs text-muted" data-testid="ai-draft-notes">{aiNotes}{#if aiAttempts > 1}{" "}<span class="text-faint">(repaired after {aiAttempts - 1} failed render{aiAttempts > 2 ? "s" : ""})</span>{/if}{#if aiSampleSource === "provided"}{" "}<span class="text-faint">(rendered against your edited sample)</span>{/if}</p>
            {/if}
            {#if aiNotes}
              <p class="text-[10px] text-faint">Edit the template directly, or describe a change and refine.</p>
            {/if}
            {#if aiError}<p class="text-xs" style="color:var(--color-bad)">{aiError}</p>{/if}
            {#if !aiCapability.enabled && promptText}
              <details open={!promptCopied}>
                <summary class="text-[10px] text-muted cursor-pointer">{promptCopied ? "Prompt copied to the clipboard · show it" : "Clipboard unavailable — select and copy the prompt below"}</summary>
                <textarea readonly rows="6" class="input mono !text-[10px] w-full mt-1" style="resize:vertical" onclick={(e) => (e.currentTarget as HTMLTextAreaElement).select()}>{promptText}</textarea>
              </details>
            {/if}
          </div>
        </div>
      </div>
    {/if}

    {#if showHistory && subscriptionId}
      <div class="px-4 py-2 border-b border-line shrink-0 max-h-[30vh] overflow-y-auto" data-testid="template-history">
        {#if versions.length === 0}
          <p class="text-xs text-faint">No saved versions yet. A version is recorded each time the subscription is saved with a changed template.</p>
        {:else}
          <table class="w-full text-xs">
            <tbody>
              {#each versions as v}
                <tr class="row-line">
                  <td class="py-1.5 pr-3 whitespace-nowrap text-muted mono tnum">{formatWhen(v.created_at)}</td>
                  <td class="py-1.5 pr-3 whitespace-nowrap"><span class="chip">{v.source === "ai_draft" ? "AI draft" : "manual"}</span>{#if v.current}{" "}<span class="chip" style="color:var(--color-ok)">current</span>{/if}</td>
                  <td class="py-1.5 pr-3 text-muted truncate max-w-[20rem]">{v.notes || v.template.split("\n")[0]}</td>
                  <td class="py-1.5 pr-3 text-faint whitespace-nowrap">{v.saved_by || ""}</td>
                  <td class="py-1.5 text-right"><button type="button" onclick={() => loadVersion(v)} class="btn btn-ghost !px-2 !py-0.5 text-xs" disabled={v.template === draft}>Load</button></td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}
      </div>
    {/if}

    <div class="flex-1 min-h-0 grid lg:grid-cols-2 {showFunctions ? 'lg:grid-cols-[1fr_1fr_18rem]' : ''}">
      <div class="flex flex-col min-h-0 border-r border-line">
        <div class="px-4 py-1.5 text-[10px] uppercase tracking-wider text-muted border-b border-line flex items-center justify-between">
          <span>Template</span>
          <span class="text-faint normal-case tracking-normal">{#if preDraft}<button type="button" data-testid="ai-undo" onclick={undoDraft} class="font-medium underline hover:text-text">Undo last change</button> · {/if}{source === "ai_draft" ? "AI draft, editable" : "Go text/template"}</span>
        </div>
        <textarea bind:value={draft} class="flex-1 min-h-0 w-full mono text-xs p-4 bg-transparent text-text outline-none resize-none" spellcheck="false" placeholder={'{\n  "id": {{ .payload.id | json }},\n  "event": {{ .event_name | json }}\n}'} data-testid="template-draft"></textarea>
      </div>
      <div class="flex flex-col min-h-0">
        <div class="px-4 py-1.5 text-[10px] uppercase tracking-wider text-muted border-b border-line flex items-center justify-between">
          <span>Rendered · {rendering ? "rendering…" : canRender ? "live" : "unavailable"}</span>
          {#if rendered}<span class="text-faint normal-case tracking-normal" style={renderedIsJson ? 'color:var(--color-ok)' : ''}>{renderedIsJson ? "✓ valid JSON" : "text"}</span>{/if}
        </div>
        <div class="flex-1 min-h-0 overflow-auto p-4">
          {#if !canRender}
            <p class="text-xs text-muted">A catch-all subscription has no single event type to render against. Pick a specific event to see output, or save and rely on the delivery log.</p>
          {:else if renderError}
            <pre class="text-xs mono whitespace-pre-wrap" style="color:var(--color-bad)" data-testid="render-error">{renderError}</pre>
          {:else if rendered}
            <pre class="text-xs mono whitespace-pre-wrap text-text" data-testid="rendered">{prettyRendered}</pre>
          {:else}
            <p class="text-xs text-faint italic">Output appears here as you type.</p>
          {/if}
        </div>
      </div>
      {#if showFunctions}
        <div class="hidden lg:flex flex-col min-h-0 border-l border-line">
          <div class="px-3 py-1.5 text-[10px] uppercase tracking-wider text-muted border-b border-line">Template functions</div>
          <div class="flex-1 min-h-0 overflow-y-auto">
            {#each functions as f}
              <button type="button" onclick={() => (selectedFunction = selectedFunction === f.name ? null : f.name)} class="w-full text-left px-3 py-2 row-line row-hover transition">
                <code class="text-xs font-semibold" style="color:var(--color-beacon)">{f.name}</code>
                {#if selectedFunction === f.name}
                  <pre class="text-[10px] mt-1 whitespace-pre-wrap text-muted">{f.description}</pre>
                {:else}
                  <p class="text-[10px] text-faint mt-0.5 truncate">{summary(f.description)}</p>
                {/if}
              </button>
            {/each}
          </div>
        </div>
      {/if}
    </div>
  </div>
{/if}
