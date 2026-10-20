<script lang="ts">
  // Generates a ready-to-run receiver for one webhook (POST
  // :generateReceiverCode) and sends test events to it (:sendTestEvent).
  // Shared by the admin webhook page and the portal's Integrate tab: the API
  // client routes the same /v1 calls through /portal/api/ in the portal.
  import { onMount } from "svelte";
  import { JSONEditor, Mode, type Content } from "svelte-jsoneditor";
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import { highlight, languageOf } from "$lib/highlight";
  import { makeZip } from "$lib/zip";
  import type { components } from "$lib/api-types";

  type Generated = components["schemas"]["GenerateReceiverCodeOutputBody"];
  type TestResult = components["schemas"]["SendTestEventOutputBody"];
  type Target = { id: string; language: string; framework: string; name: string; runtime: string };
  type SpecSubscription = { id: string; event_type: string; handler: string; format: string; model: string; notes?: string[]; render_error?: string };

  interface Props {
    consumer: string;
    webhookId: string;
    /** The webhook's subscriptions, for the test-event buttons. */
    subscriptions: { subscription_id: string; event_name: string }[];
  }

  let { consumer, webhookId, subscriptions }: Props = $props();

  // Used until /v1/capabilities answers (and if it cannot).
  const FALLBACK_TARGETS: Target[] = [
    { id: "go-nethttp", language: "go", framework: "nethttp", name: "Go (net/http)", runtime: "Go 1.22 or later" },
    { id: "node-express", language: "node", framework: "express", name: "Node.js (Express, TypeScript)", runtime: "Node.js 22.18 or later" },
    { id: "python-fastapi", language: "python", framework: "fastapi", name: "Python (FastAPI)", runtime: "Python 3.10 or later" },
  ];

  let targets = $state<Target[]>(FALLBACK_TARGETS);
  let targetId = $state("");
  let generated = $state<Generated | null>(null);
  let generating = $state(false);
  let error = $state("");
  let selectedPath = $state("");
  let copied = $state("");
  let tests = $state<Record<string, { busy: boolean; result?: TestResult; error?: string }>>({});
  // Per subscription: the payload being edited, when its editor is open.
  let payloads = $state<Record<string, { content: Content; loading: boolean }>>({});

  const target = $derived(targets.find((t) => t.id === targetId));
  const selectedFile = $derived(generated?.files.find((f) => f.path === selectedPath));
  const tokens = $derived(selectedFile ? highlight(selectedFile.content, languageOf(selectedFile.path)) : []);
  const specSubs = $derived(((generated?.spec as { subscriptions?: SpecSubscription[] } | undefined)?.subscriptions ?? []));
  const notes = $derived(specSubs.filter((s) => s.notes?.length || s.render_error));

  onMount(async () => {
    try {
      const caps = unwrap(await api.GET("/v1/capabilities"));
      if (caps.receiver_code?.targets?.length) targets = caps.receiver_code.targets;
    } catch {
      // Keep the fallback list: generation still validates the target.
    }
  });

  async function generate(id: string) {
    const t = targets.find((x) => x.id === id);
    if (!t) return;
    targetId = id;
    generating = true;
    error = "";
    generated = null;
    try {
      generated = unwrap(
        await api.POST("/v1/consumers/{consumer}/webhooks/{webhook_id}:generateReceiverCode", {
          params: { path: { consumer, webhook_id: webhookId } },
          body: { language: t.language, framework: t.framework },
        }),
      );
      // Open the file a developer edits first.
      const first = generated.files.find((f) => /^(src\/)?handlers\.(go|py|ts)$/.test(f.path)) ?? generated.files[0];
      selectedPath = first?.path ?? "";
    } catch (e) {
      error = formatAPIError(e, "Failed to generate the receiver");
    } finally {
      generating = false;
    }
  }

  async function copy(text: string, what: string) {
    try {
      await navigator.clipboard.writeText(text);
      copied = what;
      setTimeout(() => {
        if (copied === what) copied = "";
      }, 1500);
    } catch {
      // Clipboard blocked: the text stays visible to select by hand.
    }
  }

  function download() {
    if (!generated || !target) return;
    const name = `sparrow-receiver-${target.id}`;
    const zip = makeZip(generated.files, `${name}/`);
    const url = URL.createObjectURL(new Blob([zip as BlobPart], { type: "application/zip" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = `${name}.zip`;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  // Opens the payload editor with the event type's sample, or closes it.
  async function togglePayload(subscriptionId: string, eventName: string) {
    if (payloads[subscriptionId]) {
      delete payloads[subscriptionId];
      return;
    }
    payloads[subscriptionId] = { content: { json: {} }, loading: true };
    try {
      const ev = unwrap(await api.GET("/v1/event-types/{name}", { params: { path: { name: eventName } } }));
      payloads[subscriptionId] = { content: { json: ev.sample_payload ?? {} }, loading: false };
    } catch {
      // No sample to start from: edit an empty object.
      payloads[subscriptionId] = { content: { json: {} }, loading: false };
    }
  }

  // The edited payload as an object, or an error message.
  function editedPayload(content: Content): { payload?: Record<string, unknown>; error?: string } {
    let value: unknown;
    try {
      value = "json" in content ? content.json : JSON.parse(content.text);
    } catch {
      return { error: "The payload is not valid JSON." };
    }
    if (value === null || typeof value !== "object" || Array.isArray(value)) {
      return { error: "The payload must be a JSON object." };
    }
    return { payload: value as Record<string, unknown> };
  }

  async function sendTest(subscriptionId: string) {
    const edited = payloads[subscriptionId] ? editedPayload(payloads[subscriptionId].content) : {};
    if (edited.error) {
      tests[subscriptionId] = { busy: false, error: edited.error };
      return;
    }
    tests[subscriptionId] = { busy: true };
    try {
      const result = unwrap(
        await api.POST("/v1/consumers/{consumer}/webhooks/{webhook_id}:sendTestEvent", {
          params: { path: { consumer, webhook_id: webhookId } },
          body: { subscription_id: subscriptionId, ...(edited.payload ? { payload: edited.payload } : {}) },
        }),
      );
      tests[subscriptionId] = { busy: false, result };
    } catch (e) {
      tests[subscriptionId] = { busy: false, error: formatAPIError(e, "Failed to send the test event") };
    }
  }

  // The file list with a row for each folder before its first file.
  type Row = { dir: string; path?: undefined; depth: number } | { path: string; name: string; depth: number };
  const rows = $derived.by(() => {
    const out: Row[] = [];
    let current = "";
    for (const f of generated?.files ?? []) {
      const parts = f.path.split("/");
      const dir = parts.slice(0, -1).join("/");
      if (dir && dir !== current) out.push({ dir: dir + "/", depth: parts.length - 2 });
      current = dir;
      out.push({ path: f.path, name: parts[parts.length - 1], depth: parts.length - 1 });
    }
    return out;
  });
  const ok = (r: TestResult) => r.status_code >= 200 && r.status_code < 300;
</script>

<section class="space-y-5" data-testid="receiver-code">
  <div class="space-y-1">
    <h3 class="eyebrow">Generate a receiver</h3>
    <p class="text-sm text-muted">
      A ready-to-run project for this endpoint: it verifies Sparrow's signatures on the raw body, rejects stale and
      repeated deliveries, routes each event to its own handler and ships with tests. Your secret is never included:
      the receiver reads it from <code class="mono text-xs">SPARROW_WEBHOOK_SECRET</code>.
    </p>
  </div>

  <div class="flex flex-wrap gap-2" role="group" aria-label="Language and framework">
    {#each targets as t (t.id)}
      <button
        class="btn {targetId === t.id ? 'btn-beacon' : 'btn-ghost'} !px-3 !py-1.5"
        aria-pressed={targetId === t.id}
        disabled={generating}
        onclick={() => generate(t.id)}
      >
        {t.name}
      </button>
    {/each}
  </div>

  {#if generating}
    <p class="text-sm text-muted">Generating…</p>
  {/if}
  {#if error}
    <p class="text-sm text-bad" role="alert">{error}</p>
  {/if}

  {#if generated && target}
    {#if generated.has_more}
      <p class="text-sm text-warn" role="status" data-testid="receiver-code-has-more">
        This endpoint has more than 200 subscriptions: only the first 200 have handlers. Generate with the CLI's
        <code class="mono text-xs">--subscription</code> to choose them.
      </p>
    {/if}
    <div class="flex flex-wrap items-center gap-2">
      <button class="btn btn-beacon !px-3 !py-1.5" onclick={download}>Download .zip</button>
      {#if selectedFile}
        <button class="btn btn-ghost !px-3 !py-1.5" onclick={() => selectedFile && copy(selectedFile.content, "file")}>
          {copied === "file" ? "Copied" : `Copy ${selectedFile.path}`}
        </button>
      {/if}
      <span class="text-xs text-muted">{generated.files.length} files · {target.runtime}</span>
    </div>

    <div class="grid gap-3 md:grid-cols-[14rem_minmax(0,1fr)]">
      <nav class="panel-2 rounded-md p-2 overflow-auto max-h-[28rem]" aria-label="Generated files">
        <ul class="space-y-0.5">
          {#each rows as row (row.path ?? row.dir)}
            <li>
              {#if row.path === undefined}
                <span class="block mono text-xs text-faint px-2 py-1" style="padding-left: {0.5 + row.depth * 0.75}rem">{row.dir}</span>
              {:else}
                {@const path = row.path}
                <button
                  class="w-full text-left mono text-xs rounded px-2 py-1 truncate {selectedPath === path ? 'bg-panel text-text' : 'text-muted hover:text-text'}"
                  style="padding-left: {0.5 + row.depth * 0.75}rem"
                  aria-current={selectedPath === path ? "true" : undefined}
                  onclick={() => (selectedPath = path)}
                  title={path}
                >
                  {row.name}
                </button>
              {/if}
            </li>
          {/each}
        </ul>
      </nav>
      <div class="panel-2 rounded-md overflow-auto max-h-[28rem] min-w-0">
        {#if selectedFile}
          <div class="sticky top-0 border-b border-line bg-panel-2 px-3 py-1.5 mono text-xs text-muted">{selectedFile.path}</div>
          <pre class="mono text-xs leading-relaxed p-3" data-testid="receiver-code-preview"><code>{#each tokens as tok, i (i)}<span class="tok-{tok.kind}">{tok.text}</span>{/each}</code></pre>
        {/if}
      </div>
    </div>

    <div class="grid gap-3 sm:grid-cols-2">
      {#each [["Run", generated.run], ["Test", generated.test]] as [label, cmd] (label)}
        <div class="space-y-1">
          <div class="flex items-center justify-between">
            <span class="field-label !mb-0">{label}</span>
            <button class="text-xs text-muted hover:text-text" onclick={() => copy(cmd, label)}>{copied === label ? "Copied" : "Copy"}</button>
          </div>
          <pre class="panel-2 rounded-md mono text-xs p-2 overflow-x-auto">{cmd}</pre>
        </div>
      {/each}
    </div>

    {#if notes.length}
      <div class="space-y-2 text-sm">
        <span class="field-label">Check before you rely on the models</span>
        <ul class="space-y-1 text-muted list-disc pl-5">
          {#each notes as s (s.id)}
            {#each s.notes ?? [] as note, i (i)}
              <li><span class="mono text-xs">{s.event_type}</span>: {note}</li>
            {/each}
            {#if s.render_error}
              <li><span class="mono text-xs">{s.event_type}</span>: template error: <span class="mono text-xs">{s.render_error}</span></li>
            {/if}
          {/each}
        </ul>
      </div>
    {/if}
  {/if}

  <div class="space-y-2 border-t border-line pt-4">
    <h3 class="eyebrow">Send a test event</h3>
    <p class="text-sm text-muted">
      Sends one signed delivery of the event type's sample, or of the payload you edit, rendered like a real delivery
      and marked <code class="mono text-xs">X-Sparrow-Test: true</code>. It is not recorded and never affects the
      endpoint's health.
    </p>
    {#if subscriptions.length === 0}
      <p class="text-sm text-muted">Subscribe this endpoint to an event type first.</p>
    {:else}
      <ul class="space-y-2">
        {#each subscriptions as s (s.subscription_id)}
          {@const t = tests[s.subscription_id]}
          <li class="flex flex-wrap items-center gap-3 text-sm" data-testid="test-event-row">
            <button class="btn btn-ghost !px-3 !py-1" disabled={t?.busy} onclick={() => sendTest(s.subscription_id)}>
              {t?.busy ? "Sending…" : "Send test event"}
            </button>
            <span class="mono text-xs">{s.event_name}</span>
            <button class="text-xs text-muted hover:text-text" onclick={() => togglePayload(s.subscription_id, s.event_name)}>
              {payloads[s.subscription_id] ? "Use the sample" : "Edit payload"}
            </button>
            {#if t?.result}
              <span class={ok(t.result) ? "text-ok" : "text-bad"} data-testid="test-event-result">
                {ok(t.result) ? "✓" : "✗"}
                {t.result.status_code || "no response"}
                · {t.result.latency_ms > 0 ? `${t.result.latency_ms} ms` : "<1 ms"}
                {#if t.result.error_class}· {t.result.error_class}{/if}
              </span>
            {:else if t?.error}
              <span class="text-bad" role="alert">{t.error}</span>
            {/if}
            {#if payloads[s.subscription_id]}
              <div class="w-full h-48" data-testid="test-event-payload">
                {#if payloads[s.subscription_id].loading}
                  <p class="text-xs text-muted">Loading the sample…</p>
                {:else}
                  <JSONEditor bind:content={payloads[s.subscription_id].content} mode={Mode.text} mainMenuBar={false} />
                {/if}
              </div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
  </div>
</section>

<style>
  .tok-comment { color: var(--color-faint); font-style: italic; }
  .tok-string { color: var(--color-ok); }
  .tok-number { color: var(--color-beacon); }
  .tok-keyword { color: var(--color-text); font-weight: 600; }
  .tok-plain { color: var(--color-muted); }
</style>
