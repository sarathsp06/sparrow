<script lang="ts">
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import type { components } from "$lib/api-types";

  type ImportResult = components["schemas"]["ImportEventTypesOutputBody"];
  type ImportItem = components["schemas"]["ImportItemResult"];

  interface Props {
    open: boolean;
    onclose: () => void;
    /** Called after an import is written. */
    onimported: () => void;
  }

  let { open, onclose, onimported }: Props = $props();

  let fileName = $state("");
  let bundle = $state<Record<string, unknown> | null>(null);
  let preview = $state<ImportResult | null>(null);
  let result = $state<ImportResult | null>(null);
  let error = $state("");
  let busy = $state(false);

  // Acknowledgements, keyed by stamp warning code.
  let acknowledged = $state<Record<string, boolean>>({});
  // Breaking changes are confirmed by typing each event type's name.
  let typedNames = $state<Record<string, string>>({});
  let pauseFailing = $state(false);

  const warningText: Record<string, (p: ImportResult) => string> = {
    version_differs: (p) =>
      `This file was exported by Sparrow ${p.stamp.exported_by || "(unknown)"}; this server is ${p.stamp.server}. Import anyway.`,
    format_unsupported: () =>
      "This file uses a bundle format newer than this server understands. Import anyway.",
    items_changed: () => "This file was edited after it was exported. Import anyway.",
  };

  let breakingItems = $derived<ImportItem[]>(
    (preview?.items ?? []).filter((i) => i.compatibility?.result === "breaking" && i.blocked),
  );
  let hasTemplateFailures = $derived(
    (preview?.items ?? []).some((i) => (i.subscriptions?.failures?.length ?? 0) > 0),
  );
  let warnings = $derived(preview?.stamp.warnings ?? []);
  let allAcknowledged = $derived(warnings.every((w) => acknowledged[w]));
  let breakingConfirmed = $derived(breakingItems.every((i) => (typedNames[i.name] ?? "").trim() === i.name));
  let itemCount = $derived(preview?.items?.length ?? 0);
  let canImport = $derived(!!preview && !busy && allAcknowledged && breakingConfirmed);

  function reset() {
    fileName = "";
    bundle = null;
    preview = null;
    result = null;
    error = "";
    acknowledged = {};
    typedNames = {};
    pauseFailing = false;
  }

  function close() {
    reset();
    onclose();
  }

  async function onFile(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    reset();
    if (!file) return;
    fileName = file.name;
    try {
      const parsed = JSON.parse(await file.text());
      if (!parsed || typeof parsed !== "object" || !Array.isArray(parsed.items)) {
        error = `${file.name} is not an event type bundle: it has no items list.`;
        return;
      }
      bundle = parsed;
    } catch (err: any) {
      error = `${file.name} is not valid JSON: ${err.message}`;
      return;
    }
    await runPreview();
  }

  function requestBody(dryRun: boolean) {
    const b = bundle as any;
    return {
      ...(b.apiVersion ? { apiVersion: b.apiVersion, kind: b.kind } : {}),
      ...(b.stamp ? { stamp: b.stamp } : {}),
      items: b.items,
      dry_run: dryRun,
      acknowledge: Object.keys(acknowledged).filter((k) => acknowledged[k]) as any,
      allow_breaking: !dryRun && breakingItems.length > 0,
      subscription_policy: pauseFailing ? ("pause" as const) : ("keep_active" as const),
    };
  }

  async function runPreview() {
    if (!bundle) return;
    busy = true;
    error = "";
    try {
      preview = unwrap(await api.POST("/v1/event-types:import", { body: requestBody(true) }));
    } catch (err: any) {
      error = formatAPIError(err, "The bundle cannot be imported");
      preview = null;
    } finally {
      busy = false;
    }
  }

  async function runImport() {
    if (!canImport) return;
    busy = true;
    error = "";
    try {
      const res = unwrap(await api.POST("/v1/event-types:import", { body: requestBody(false) }));
      if (!res.applied) {
        error = `Nothing was imported. Blocked by: ${(res.blocked_by ?? []).join(", ")}.`;
        preview = res;
        return;
      }
      result = res;
      onimported();
    } catch (err: any) {
      error = formatAPIError(err, "Import failed");
    } finally {
      busy = false;
    }
  }

  const actionTone: Record<string, string> = {
    created: "ok",
    new_version: "warn",
    updated: "beacon",
    unchanged: "idle",
  };
  const actionLabel: Record<string, string> = {
    created: "Create",
    new_version: "New version",
    updated: "Update",
    unchanged: "Unchanged",
  };

  function versionLabel(i: ImportItem) {
    return i.previous_version && i.previous_version !== i.version
      ? `v${i.previous_version} → v${i.version}`
      : `v${i.version}`;
  }

  // Deliveries that may fail after the change are template errors from today.
  function failuresLink(importedAt: string) {
    const day = importedAt.slice(0, 10);
    return `/deliveries?status=failed&error_category=template_error&created_after=${day}`;
  }
</script>

{#if open}
  <div class="fixed inset-0 z-50 flex items-start justify-center p-4 overflow-y-auto" role="dialog" aria-modal="true" aria-labelledby="import-title">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="fixed inset-0 bg-black/40 backdrop-blur-sm" role="presentation" onclick={close}></div>
    <div class="panel relative w-full max-w-4xl p-6 my-8 space-y-5">
      <div class="flex items-start justify-between gap-4">
        <div>
          <p class="eyebrow mb-1.5">Catalog / Import</p>
          <h2 id="import-title" class="text-lg font-semibold text-text">Import event types</h2>
          <p class="text-sm text-muted mt-1">
            Each entry replaces the event type with the same name. Types not in the file are left alone. Nothing is written until you confirm.
          </p>
        </div>
        <button onclick={close} class="link text-2xl leading-none" aria-label="Close">&times;</button>
      </div>

      {#if result}
        {@const imported = result.items ?? []}
        {@const pausedCount = imported.reduce((n, i) => n + (i.subscriptions?.paused?.length ?? 0), 0)}
        {@const failingCount = imported.reduce((n, i) => n + (i.subscriptions?.failures?.length ?? 0), 0)}
        <div class="panel-2 p-4 space-y-2">
          <p class="text-sm text-text">Imported {imported.length} event type{imported.length === 1 ? "" : "s"}.</p>
          {#if pausedCount > 0}
            <p class="text-sm text-muted">
              Paused {pausedCount} subscription{pausedCount === 1 ? "" : "s"} whose template failed. Their deliveries are held until you fix the template, resume the subscription, and retry them.
            </p>
          {:else if failingCount > 0}
            <p class="text-sm text-muted">
              {failingCount} subscription template{failingCount === 1 ? "" : "s"} no longer fit{failingCount === 1 ? "s" : ""} the new schema; their deliveries fail with a template error until the template is fixed.
              <a class="link-beacon" href={failuresLink(result.imported_at ?? "")}>Review template failures</a>
            </p>
          {/if}
          <div class="flex justify-end">
            <button class="btn btn-beacon" onclick={close}>Done</button>
          </div>
        </div>
      {:else}
        <div>
          <label for="bundle-file" class="field-label">Bundle file</label>
          <input id="bundle-file" type="file" accept="application/json,.json" onchange={onFile} class="input" />
          {#if fileName && busy && !preview}
            <p class="text-xs text-muted mt-1.5">Checking {fileName}…</p>
          {/if}
        </div>

        {#if error}
          <div class="panel p-3" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
            <p class="text-sm" style="color:var(--color-bad)">{error}</p>
          </div>
        {/if}

        {#if preview}
          {#if preview.stamp.status === "unsigned"}
            <p class="text-xs text-muted">This file has no stamp, so it was written by hand or by an older Sparrow. That is fine.</p>
          {/if}

          <div class="overflow-x-auto border border-line rounded">
            <table class="w-full text-left">
              <thead>
                <tr class="border-b border-line">
                  <th class="th">Event type</th>
                  <th class="th">Action</th>
                  <th class="th">Version</th>
                  <th class="th">Details</th>
                </tr>
              </thead>
              <tbody>
                {#each preview.items ?? [] as item}
                  {@const tone = item.blocked ? "bad" : actionTone[item.action]}
                  <tr class="row-line align-top">
                    <td class="td font-medium text-text mono text-sm">{item.name}</td>
                    <td class="td">
                      <span class="chip" style="color:var(--color-{tone});border-color:color-mix(in srgb,var(--color-{tone}) 35%,transparent);background:color-mix(in srgb,var(--color-{tone}) 12%,var(--color-panel-2))">
                        {item.blocked ? "Breaking" : actionLabel[item.action]}
                      </span>
                    </td>
                    <td class="td mono text-xs text-muted whitespace-nowrap">{versionLabel(item)}</td>
                    <td class="td text-sm space-y-1.5">
                      {#if item.changes?.length}
                        <p class="text-muted">Changes: {item.changes.join(", ")}</p>
                      {/if}
                      {#if item.active_change === "deactivates"}
                        <p style="color:var(--color-warn)">Deactivates this event type: pushes will be rejected.</p>
                      {:else if item.active_change === "reactivates"}
                        <p style="color:var(--color-warn)">Reactivates this event type.</p>
                      {/if}
                      {#if item.compatibility?.result === "breaking"}
                        <ul class="list-disc pl-4" style="color:var(--color-bad)">
                          {#each item.compatibility.reasons ?? [] as reason}
                            <li class="mono text-xs">{reason}</li>
                          {/each}
                        </ul>
                      {:else if item.compatibility?.result === "compatible"}
                        <p class="text-muted">Compatible with existing subscriptions.</p>
                      {/if}
                      {#if item.subscriptions}
                        {#each item.subscriptions.failures ?? [] as f}
                          <p class="text-xs" style="color:var(--color-bad)">
                            Template fails ({f.payload === "required_only" ? "when optional fields are absent" : "against the new schema"}) for subscription
                            <span class="mono">{f.subscription_id.slice(0, 8)}</span> in {f.consumer}: <span class="mono">{f.error}</span>
                          </p>
                        {/each}
                        {#if item.subscriptions.passed}
                          <p class="text-xs text-muted">{item.subscriptions.passed} subscription template{item.subscriptions.passed === 1 ? "" : "s"} still render.</p>
                        {/if}
                        {#if item.subscriptions.without_transform}
                          <p class="text-xs text-muted">
                            {item.subscriptions.without_transform} subscription{item.subscriptions.without_transform === 1 ? "" : "s"} without a transform will receive the new payload as is. Sparrow cannot tell whether the receiving system copes.
                          </p>
                        {/if}
                      {/if}
                    </td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>

          {#if warnings.length}
            <fieldset class="space-y-2">
              <legend class="field-label">Version stamp</legend>
              {#each warnings as w}
                <label class="flex items-start gap-2 text-sm text-text cursor-pointer">
                  <input type="checkbox" bind:checked={acknowledged[w]} class="mt-0.5 accent-[color:var(--color-beacon)]" />
                  <span>{warningText[w]?.(preview) ?? w}</span>
                </label>
              {/each}
            </fieldset>
          {/if}

          {#if breakingItems.length}
            <fieldset class="panel-2 p-4 space-y-3" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent)">
              <legend class="field-label">Breaking changes</legend>
              <p class="text-sm text-muted">
                These schema changes could break subscriptions that receive the event type. To apply them, type each event type's name.
              </p>
              {#each breakingItems as item}
                <label class="block">
                  <span class="text-xs text-muted">Type <span class="mono text-text">{item.name}</span> to confirm</span>
                  <input class="input mono mt-1" autocomplete="off" spellcheck="false" bind:value={typedNames[item.name]} />
                </label>
              {/each}
            </fieldset>
          {/if}

          {#if hasTemplateFailures}
            <label class="flex items-start gap-2 text-sm text-text cursor-pointer">
              <input type="checkbox" bind:checked={pauseFailing} class="mt-0.5 accent-[color:var(--color-beacon)]" />
              <span>
                Pause the subscriptions whose template fails. Their deliveries are held as paused until you fix the template, resume, and retry them.
                <span class="text-muted">Otherwise those deliveries fail with a template error, which you can retry after fixing the template.</span>
              </span>
            </label>
          {/if}

          <div class="flex items-center justify-end gap-3">
            <button class="btn btn-ghost" onclick={close}>Cancel</button>
            <button class="btn btn-beacon" disabled={!canImport} onclick={runImport}>
              {busy ? "Importing…" : `Import ${itemCount} event type${itemCount === 1 ? "" : "s"}`}
            </button>
          </div>
        {/if}
      {/if}
    </div>
  </div>
{/if}
