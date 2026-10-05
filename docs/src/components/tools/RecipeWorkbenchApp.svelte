<script lang="ts">
  import { renderGoTemplate, type EventContext } from '../../lib/go-template-engine.js';
  import { RECIPES, RECIPE_META } from '../../lib/recipes.js';
  import {
    generateSendgridTemplate,
    generateTwilioTemplate,
    generateSlackTemplate,
    generateDiscordTemplate,
    generateNtfyTemplate,
    generatePagerdutyTemplate,
    smsSegments,
    payloadPaths,
    ENVELOPE_PATHS,
  } from '../../lib/compose.js';
  import { parseTemplate, writeField, balanced, detectRecipe, type TemplateField } from '../../lib/template-fields.js';
  import { onMount } from 'svelte';

  // The transform template is the input: paste one (from Sparrow, the CLI, or
  // an AI draft) or start from a recipe's starter. Its message fields are
  // parsed out for editing, and edits are written straight back into it.

  // ---- Starter templates (what a recipe's tab loads before you paste your own) ----
  const STARTERS: Record<string, () => string> = {
    sendgrid: () =>
      generateSendgridTemplate({
        toMode: 'list',
        listPath: '.payload.alert_recipients',
        listField: 'email',
        to: '',
        cc: '',
        bcc: '',
        subject: 'Sparrow: webhook for {{.payload.consumer}} is now {{.payload.new_health}}',
        bodyText:
          'Webhook {{.payload.webhook_id}} ({{.payload.url}}) health changed: {{.payload.old_health}} -> {{.payload.new_health}}',
        format: 'text',
      }),
    twilio: () => generateTwilioTemplate('Sparrow {{.event_name}}: {{.payload.active_incidents}} active incidents in {{.payload.datacenter}}'),
    slack: () =>
      generateSlackTemplate({
        header: '{{.event_name}}',
        body: 'Service *{{.payload.service}}* deployed `{{.payload.version}}` to {{.payload.environment}} in {{.payload.duration_seconds}}s',
        includeContext: true,
      }),
    discord: () =>
      generateDiscordTemplate({
        title: '{{.event_name}}',
        description: 'New signup: **{{.payload.user_id}}** on the {{.payload.plan}} plan (via {{.payload.referrer}}).',
        includePayload: true,
      }),
    ntfy: () =>
      generateNtfyTemplate({
        title: 'High CPU on {{.payload.host}}',
        message: 'CPU usage hit {{.payload.usage_percent}}% — check the host.',
        tags: 'warning, computer',
        priority: 4,
        includePayload: false,
      }),
    pagerduty: () => generatePagerdutyTemplate({ summary: 'Sparrow: {{.event_name}}', severity: 'error', source: 'sparrow' }),
  };
  const starterFor = (id: string) => STARTERS[id]?.() ?? RECIPES.find((r) => r.name === id)!.transform_template;

  // ---- Recipe selection (the destination the preview renders as) ----
  let selectedId = $state('sendgrid');
  let activeRecipe = $derived(RECIPES.find((r) => r.name === selectedId)!);
  let meta = $derived(RECIPE_META[selectedId]);
  // Where the message lands, as people name it. Recipe titles describe the
  // output ("Slack Message"); the tabs and receiver name the destination.
  const DEST: Record<string, string> = {
    sendgrid: 'SendGrid',
    slack: 'Slack',
    twilio: 'Twilio SMS',
    discord: 'Discord',
    ntfy: 'ntfy',
    pagerduty: 'PagerDuty',
    clickhouse: 'ClickHouse',
    cloudevents: 'CloudEvents',
  };
  let dest = $derived(DEST[selectedId] ?? meta.title);

  // ---- Parameters (seeded with demo values, namespaced per recipe so
  // recipes sharing a param name — e.g. webhook_url — don't collide) ----
  let paramValues = $state<Record<string, Record<string, string>>>(
    Object.fromEntries(Object.entries(RECIPE_META).map(([k, m]) => [k, { ...m.demoParams }])),
  );
  let params = $derived(paramValues[selectedId] ?? {});

  // Tiny Slack mrkdwn renderer for the preview: bold / code / italics / newlines.
  function mrkdwn(t: string): string {
    return t
      .replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
      .replace(/`([^`\n]+)`/g, '<code>$1</code>')
      .replace(/\*([^*\n]+)\*/g, '<strong>$1</strong>')
      .replace(/_([^_\n]+)_/g, '<em>$1</em>')
      .replace(/\n/g, '<br>');
  }

  // ---- Event context ----
  let eventName = $state(RECIPE_META.sendgrid.presets[0].event_name);
  let eventId = $state('evt_0195c2a17f30');
  let timestamp = $state(new Date().toISOString());
  let attempt = $state(1);
  let payloadStr = $state(JSON.stringify(RECIPE_META.sendgrid.presets[0].payload, null, 2));
  let selectedPresetId = $state(RECIPE_META.sendgrid.presets[0].id);

  // ---- Template (the input) ----
  let template = $state(starterFor('sendgrid'));
  let parsed = $derived(parseTemplate(template));
  let primaryFields = $derived(parsed.fields.map((f, i) => ({ f, i })).filter(({ f }) => f.primary));
  let otherFields = $derived(parsed.fields.map((f, i) => ({ f, i })).filter(({ f }) => !f.primary));

  // ---- Derived rendering ----
  let parsedPayload = $derived.by(() => {
    try {
      return JSON.parse(payloadStr);
    } catch {
      return null;
    }
  });

  let eventContext = $derived<EventContext>({
    event_id: eventId,
    event_name: eventName,
    timestamp,
    attempt,
    payload: parsedPayload || {},
  });

  let renderResult = $derived.by(() => {
    if (!parsedPayload) return { result: '', error: 'Invalid JSON in event payload', jsonObj: undefined as any };
    return renderGoTemplate(template, eventContext, params);
  });

  let chips = $derived([...ENVELOPE_PATHS, ...payloadPaths(parsedPayload || {})]);

  let smsRendered = $derived.by(() => {
    try {
      return new URLSearchParams(renderResult.result.trim()).get('Body') ?? '';
    } catch {
      return '';
    }
  });
  let smsInfo = $derived(smsSegments(smsRendered));

  let emailIsHtml = $derived(renderResult.jsonObj?.content?.[0]?.type === 'text/html');
  let emailBodyRendered = $derived(renderResult.jsonObj?.content?.[0]?.value ?? '');
  let emailAddresses = $derived.by(() => {
    const out: Record<'to' | 'cc' | 'bcc', string[]> = { to: [], cc: [], bcc: [] };
    for (const p of renderResult.jsonObj?.personalizations ?? []) {
      for (const key of ['to', 'cc', 'bcc'] as const) {
        for (const t of p?.[key] ?? []) if (t?.email) out[key].push(t.email);
      }
    }
    return out;
  });

  let cliCommand = $derived.by(() => {
    let cmd = `sparrow use ${selectedId}`;
    for (const p of activeRecipe.params) {
      const val = params[p.name] ?? p.default ?? '';
      cmd += ` \\\n  --param ${p.name}="${p.secret ? '<secret>' : val}"`;
    }
    cmd += ` \\\n  --event ${eventName}`;
    return cmd;
  });

  // ---- Actions ----
  function switchRecipe(id: string) {
    selectedId = id;
    selectPreset(RECIPE_META[id].presets[0].id);
    template = starterFor(id);
    drafts = {};
  }

  function selectPreset(presetId: string) {
    const p = meta.presets.find((x) => x.id === presetId) ?? RECIPE_META[selectedId].presets[0];
    selectedPresetId = p.id;
    eventName = p.event_name;
    payloadStr = JSON.stringify(p.payload, null, 2);
  }

  // A pasted template usually targets a known destination; preview it as that.
  function onTemplatePaste() {
    queueMicrotask(() => {
      const r = detectRecipe(template);
      if (r && r !== selectedId && RECIPE_META[r]) selectedId = r;
    });
  }

  // ---- Field editing ----
  // While a field is focused its text lives in `drafts`, so a half-typed
  // placeholder ("{{.pay") isn't written into the template until it closes.
  let drafts = $state<Record<number, string>>({});

  function editField(i: number, f: TemplateField, value: string) {
    drafts[i] = value;
    if (f.kind === 'number' && !/^-?\d+(\.\d+)?$/.test(value.trim())) return;
    if (!balanced(value)) return;
    template = writeField(template, f, f.kind === 'number' ? value.trim() : value);
  }

  function rowsFor(text: string): number {
    return Math.min(8, Math.max(1, text.split('\n').length, Math.ceil(text.length / 48)));
  }

  // Variable chips insert into the last focused text field.
  let lastField: HTMLInputElement | HTMLTextAreaElement | null = null;
  function trackFocus(e: FocusEvent) {
    const t = e.target as HTMLElement;
    if (t instanceof HTMLTextAreaElement || (t instanceof HTMLInputElement && t.type !== 'checkbox')) lastField = t;
  }
  function insertChip(path: string) {
    if (!lastField) return;
    const start = lastField.selectionStart ?? lastField.value.length;
    lastField.setRangeText(`{{${path}}}`, start, lastField.selectionEnd ?? start, 'end');
    lastField.dispatchEvent(new Event('input', { bubbles: true }));
    lastField.focus();
  }

  // ---- Modal state ----
  let payloadModalOpen = $state(false);
  let templateModalOpen = $state(false);
  let fieldsModalOpen = $state(false);

  let copiedToast = $state<string | null>(null);
  function copyText(txt: string, msg: string) {
    navigator.clipboard.writeText(txt);
    copiedToast = msg;
    setTimeout(() => (copiedToast = null), 2000);
  }

  // A caller (e.g. Sparrow's template editor) can deep-link its state:
  //   ?template=<base64-utf8>  the transform template to load
  //   ?sample=<base64-utf8 JSON> a sample event payload
  //   ?event=<name>            the event name
  //   ?recipe=<name>           the destination to preview as (else detected)
  const b64 = (raw: string) => decodeURIComponent(escape(atob(raw)));
  onMount(() => {
    const q = new URLSearchParams(location.search);
    let tmpl: string | null = null;
    try {
      tmpl = q.get('template') && b64(q.get('template')!);
    } catch {
      /* malformed param — keep the starter */
    }
    const recipe = q.get('recipe') ?? (tmpl ? detectRecipe(tmpl) : null);
    if (recipe && RECIPE_META[recipe]) switchRecipe(recipe);
    if (tmpl) template = tmpl;
    const sample = q.get('sample');
    if (sample) {
      try {
        payloadStr = JSON.stringify(JSON.parse(b64(sample)), null, 2);
      } catch {
        /* malformed param — keep the preset payload */
      }
    }
    if (q.get('event')) eventName = q.get('event')!;
  });
</script>

<div class="rw-root">
  <header class="rw-header">
    <h1>Recipe workbench</h1>
    <p>
      Paste a transform template, from Sparrow, an AI draft, or a recipe starter, and edit its message while you watch
      what {dest} receives.
    </p>
  </header>

  {#if copiedToast}
    <div class="rw-toast" role="status">{copiedToast}</div>
  {/if}

  <nav class="rw-tabs" aria-label="Destination">
    {#each RECIPES as r}
      <button class="rw-tab" aria-current={selectedId === r.name ? 'true' : undefined} onclick={() => switchRecipe(r.name)}>
        {DEST[r.name] ?? RECIPE_META[r.name].title}
      </button>
    {/each}
  </nav>

  <div class="rw-grid">
    <!-- Left: the worksheet -->
    <div class="rw-sheet">
      <section class="rw-section">
        <div class="rw-section-head">
          <h2>Transform template</h2>
          <div class="rw-actions">
            <button class="rw-btn" onclick={() => (templateModalOpen = true)}>Open editor</button>
            <button class="rw-btn" onclick={() => copyText(template, 'Copied template')}>Copy template</button>
          </div>
        </div>
        <p class="rw-help">The Go template Sparrow stores on the subscription. Paste your own, or edit this {dest} starter.</p>
        <textarea
          id="rw_template"
          bind:value={template}
          onpaste={onTemplatePaste}
          rows="12"
          spellcheck="false"
          class="rw-code"
          aria-label="Transform template"
        ></textarea>
        <div class="rw-row">
          <span class="rw-note" class:rw-note-warn={parsed.format === 'unknown'}>
            {#if parsed.format === 'unknown'}
              Not a JSON or form body, so no fields can be pulled out. Edit the template directly.
            {:else}
              {parsed.format === 'json' ? 'JSON body' : 'Form body'}, {primaryFields.length} message field{primaryFields.length === 1 ? '' : 's'}{otherFields.length ? ` and ${otherFields.length} other value${otherFields.length === 1 ? '' : 's'}` : ''}
            {/if}
          </span>
          <button class="rw-link" onclick={() => { template = starterFor(selectedId); drafts = {}; }}>Reset to starter</button>
        </div>
      </section>

      <section class="rw-section" onfocusin={trackFocus}>
        <div class="rw-section-head">
          <h2>Message</h2>
          {#if parsed.fields.length}
            <button class="rw-btn" onclick={() => (fieldsModalOpen = true)}>Open editor</button>
          {/if}
        </div>
        {@render fieldsForm('')}
      </section>

      <section class="rw-section">
        <h2>Sample event</h2>
        <p class="rw-help">The event Sparrow delivers. Pick a preset or paste a payload; its fields become the placeholders above.</p>
        {#if meta.presets.length > 1}
          <div class="rw-presets">
            {#each meta.presets as p}
              <button class="rw-preset" aria-pressed={selectedPresetId === p.id} onclick={() => selectPreset(p.id)}>
                {p.label}
              </button>
            {/each}
          </div>
        {/if}
        <div class="rw-two">
          <div class="rw-field">
            <label for="rw_event_name">Event name</label>
            <input id="rw_event_name" type="text" bind:value={eventName} />
          </div>
          <div class="rw-field">
            <label for="rw_attempt">Attempt</label>
            <input id="rw_attempt" type="number" bind:value={attempt} min="1" />
          </div>
        </div>
        <div class="rw-field">
          <div class="rw-row">
            <label for="rw_payload">Payload</label>
            <div class="rw-actions">
              <span class="rw-note" class:rw-note-warn={!parsedPayload}>{parsedPayload ? 'Valid JSON' : 'Invalid JSON'}</span>
              <button class="rw-btn" onclick={() => (payloadModalOpen = true)}>Open editor</button>
            </div>
          </div>
          <textarea id="rw_payload" bind:value={payloadStr} rows="10" spellcheck="false" class="rw-code"></textarea>
        </div>
      </section>

      <section class="rw-section">
        <h2>Parameters</h2>
        <p class="rw-help">Asked once when you apply the recipe. Secrets are encrypted at rest.</p>
        <div class="rw-stack">
          {#each activeRecipe.params as p}
            <div class="rw-field">
              <label for={'p_' + p.name}>{p.prompt}</label>
              <input
                id={'p_' + p.name}
                type={p.secret ? 'password' : 'text'}
                bind:value={paramValues[selectedId][p.name]}
                placeholder={p.default || ''}
              />
            </div>
          {/each}
        </div>
      </section>

      <section class="rw-section">
        <h2>Setting up {dest}</h2>
        <p class="rw-help">Where the parameters above come from.</p>
        <ol class="rw-steps">
          {#each meta.setup.steps as s}
            <li>{s}</li>
          {/each}
        </ol>
        <a class="rw-link" href={meta.setup.docsUrl} target="_blank" rel="noopener noreferrer">{meta.setup.docsLabel}</a>
      </section>
    </div>

    <!-- Right: the receiver. Sticky so the result stays in view while editing. -->
    <aside class="rw-receiver" aria-live="polite">
      <h2 class="rw-receiver-title">What {dest} receives</h2>
      {@render preview()}

      <details class="rw-drawer">
        <summary>Request body</summary>
        <div class="rw-row">
          <span class="rw-note">Sent to {activeRecipe.webhook.url.split('/')[2] || 'the destination'}</span>
          <button class="rw-link" onclick={() => copyText(renderResult.result, 'Copied request body')}>Copy request body</button>
        </div>
        <pre class="rw-out"><code>{renderResult.result}</code></pre>
        <p class="rw-note">Rendered by a simplified in-browser engine. The server uses Go's text/template.</p>
      </details>

      <details class="rw-drawer">
        <summary>Apply with the CLI</summary>
        <p class="rw-note">
          Registers the stock {dest} recipe on your server. To use the template you edited here, copy it into the
          subscription's template editor in Sparrow.
        </p>
        <pre class="rw-out"><code>{cliCommand}</code></pre>
        <button class="rw-link" onclick={() => copyText(cliCommand, 'Copied command')}>Copy command</button>
      </details>
    </aside>
  </div>
</div>

<!-- ============ Destination preview ============ -->
{#snippet preview()}
  {#if renderResult.error}
    <div class="rw-error">{renderResult.error}</div>
  {/if}
  <div class="rw-mock">
    {#if selectedId === 'sendgrid' && renderResult.jsonObj}
      <div class="rw-email">
        <div class="rw-email-head">
          <div><strong>Subject</strong> {renderResult.jsonObj?.subject || '(no subject)'}</div>
          <div><strong>From</strong> {renderResult.jsonObj?.from?.name} &lt;{renderResult.jsonObj?.from?.email}&gt;</div>
          <div><strong>To</strong> {emailAddresses.to.length ? emailAddresses.to.join(', ') : '(no recipients)'}</div>
          {#if emailAddresses.cc.length}
            <div><strong>Cc</strong> {emailAddresses.cc.join(', ')}</div>
          {/if}
          {#if emailAddresses.bcc.length}
            <div><strong>Bcc</strong> {emailAddresses.bcc.join(', ')}</div>
          {/if}
        </div>
        <div class="rw-email-body" class:rw-email-html={emailIsHtml}>{#if emailIsHtml}{@html emailBodyRendered}{:else}{emailBodyRendered}{/if}</div>
      </div>

    {:else if selectedId === 'twilio' && smsRendered}
      <div class="rw-sms">
        <div class="rw-sms-to">To +{params.to_number}</div>
        <div class="rw-sms-bubble">{smsRendered}</div>
        <div class="rw-sms-meta" class:rw-note-warn={smsInfo.segments > 1}>
          {smsInfo.chars} characters, {smsInfo.segments} segment{smsInfo.segments === 1 ? '' : 's'}, {smsInfo.encoding}{smsInfo.segments > 1 ? '. Billed per segment.' : ''}
        </div>
      </div>

    {:else if selectedId === 'slack' && renderResult.jsonObj?.blocks}
      <div class="rw-slack">
        <div class="rw-slack-bot">
          <span class="rw-slack-avatar">S</span>
          <span class="rw-slack-name">Sparrow</span>
          <span class="rw-slack-app">APP</span>
        </div>
        <div class="rw-slack-blocks">
          {#each renderResult.jsonObj.blocks as block}
            {#if block.type === 'header'}
              <div class="rw-slack-header">{block.text?.text}</div>
            {:else if block.type === 'section' && block.fields}
              <div class="rw-slack-fields">
                {#each block.fields as f}
                  <div class="rw-slack-text">{@html mrkdwn(f.text ?? '')}</div>
                {/each}
              </div>
            {:else if block.type === 'section' && block.text}
              <div class="rw-slack-text">{@html mrkdwn(block.text?.text ?? '')}</div>
            {:else if block.type === 'context' && block.elements}
              <div class="rw-slack-context">{@html block.elements.map((e: any) => mrkdwn(e.text ?? '')).join('  ')}</div>
            {:else if block.type === 'divider'}
              <hr class="rw-slack-divider" />
            {/if}
          {/each}
        </div>
      </div>

    {:else if selectedId === 'discord' && renderResult.jsonObj?.embeds}
      <div class="rw-discord">
        {#if renderResult.jsonObj.content}
          <div class="rw-discord-content">{renderResult.jsonObj.content}</div>
        {/if}
        {#each renderResult.jsonObj.embeds as embed}
          <div class="rw-discord-embed">
            <div class="rw-discord-title">{embed?.title ?? ''}</div>
            <pre class="rw-discord-desc"><code>{embed?.description ?? ''}</code></pre>
            {#each embed?.fields ?? [] as f}
              <div class="rw-discord-field"><strong>{f.name}</strong><div>{f.value}</div></div>
            {/each}
            <div class="rw-discord-footer">{embed?.timestamp ?? ''}</div>
          </div>
        {/each}
      </div>

    {:else if selectedId === 'ntfy' && renderResult.jsonObj}
      <div class="rw-ntfy">
        <div class="rw-ntfy-top">
          <span>{(params.server_url || '').replace(/^https?:\/\//, '')}/{renderResult.jsonObj?.topic}</span>
          {#if renderResult.jsonObj?.priority}<span>priority {renderResult.jsonObj.priority}</span>{/if}
        </div>
        <div class="rw-ntfy-title">{renderResult.jsonObj?.title ?? eventName}</div>
        <pre class="rw-ntfy-body">{renderResult.jsonObj?.message ?? ''}</pre>
        {#if renderResult.jsonObj?.tags?.length}
          <div class="rw-ntfy-tags">{renderResult.jsonObj.tags.map((t: string) => `#${t}`).join(' ')}</div>
        {/if}
      </div>

    {:else if selectedId === 'pagerduty' && renderResult.jsonObj?.payload}
      <div class="rw-pd">
        <div class="rw-pd-head">
          <span>Incident</span>
          <span class="rw-pd-sev">{String(renderResult.jsonObj.payload.severity || '')}</span>
        </div>
        <div class="rw-pd-summary">{renderResult.jsonObj.payload.summary}</div>
        <div class="rw-note">Dedup key <code>{renderResult.jsonObj.dedup_key}</code></div>
        <pre class="rw-pd-details"><code>{JSON.stringify(renderResult.jsonObj.payload.custom_details, null, 2)}</code></pre>
      </div>

    {:else if selectedId === 'clickhouse' && renderResult.jsonObj}
      <div class="rw-ch">
        <div class="rw-note">One row inserted into <code>{params.table}</code></div>
        <table class="rw-table">
          <tbody>
            {#each Object.entries(renderResult.jsonObj) as [col, val]}
              <tr><th>{col}</th><td><code>{typeof val === 'string' ? val : JSON.stringify(val)}</code></td></tr>
            {/each}
          </tbody>
        </table>
      </div>

    {:else}
      {#if !renderResult.jsonObj && parsed.format === 'json' && renderResult.result}
        <div class="rw-error">The rendered body isn't valid JSON, so the destination would reject it.</div>
      {/if}
      <pre class="rw-out"><code>{renderResult.jsonObj ? JSON.stringify(renderResult.jsonObj, null, 2) : renderResult.result}</code></pre>
    {/if}
  </div>
{/snippet}

<!-- ============ Field editor ============ -->
{#snippet fieldRow(f: TemplateField, i: number, idp: string)}
  <div class="rw-field">
    <div class="rw-row">
      <label for="{idp}fld_{i}" class="rw-path">{f.path}</label>
      {#if f.kind === 'raw'}<span class="rw-note" title="Not plain text, so it is edited as a Go template expression">expression</span>{/if}
    </div>
    {#if f.when}
      <div class="rw-when" title={f.when}>{f.when}</div>
    {/if}
    {#if f.kind === 'bool'}
      <label class="rw-check">
        <input
          id="{idp}fld_{i}"
          type="checkbox"
          checked={f.text === 'true'}
          onchange={(e) => editField(i, f, (e.currentTarget as HTMLInputElement).checked ? 'true' : 'false')}
        />
        {f.text}
      </label>
    {:else if f.kind === 'number'}
      <input
        id="{idp}fld_{i}"
        type="text"
        inputmode="decimal"
        value={drafts[i] ?? f.text}
        oninput={(e) => editField(i, f, (e.currentTarget as HTMLInputElement).value)}
        onblur={() => delete drafts[i]}
      />
    {:else}
      {@const value = drafts[i] ?? f.text}
      <textarea
        id="{idp}fld_{i}"
        rows={rowsFor(value)}
        class="rw-text"
        class:rw-text-code={f.kind === 'raw'}
        class:rw-pending={!balanced(value)}
        spellcheck="false"
        {value}
        oninput={(e) => editField(i, f, (e.currentTarget as HTMLTextAreaElement).value)}
        onblur={() => delete drafts[i]}
      ></textarea>
      {#if !balanced(value)}
        <span class="rw-note rw-note-warn">Close the {'{{ }}'} to update the template.</span>
      {/if}
    {/if}
  </div>
{/snippet}

{#snippet fieldsForm(idp: string)}
  {#if parsed.format === 'unknown'}
    <p class="rw-help">Nothing to edit here. The template isn't a JSON object or form body, so edit it directly above.</p>
  {:else if parsed.fields.length === 0}
    <p class="rw-help">Nothing to edit here. The template has no literal values.</p>
  {:else}
    <p class="rw-help">
      Every value in the template. Write placeholders like <code>{'{{.payload.x}}'}</code>, or click one below to insert
      it where you last typed. A value inside an <code>{'{{if}}'}</code> is listed once per branch.
    </p>
    <div class="rw-chips">
      {#each chips as c}
        <button class="rw-chip" onclick={() => insertChip(c)} title={'Insert {{' + c + '}}'}>{c}</button>
      {/each}
    </div>
    <div class="rw-stack">
      {#each primaryFields as { f, i } (i)}
        {@render fieldRow(f, i, idp)}
      {/each}
      {#if primaryFields.length === 0}
        <p class="rw-help">No message text in this template. Its values are listed below.</p>
      {/if}
    </div>
    {#if otherFields.length}
      <details class="rw-drawer" open={primaryFields.length === 0}>
        <summary>{otherFields.length} other value{otherFields.length === 1 ? '' : 's'}: structure, parameters, envelope</summary>
        <div class="rw-stack">
          {#each otherFields as { f, i } (i)}
            {@render fieldRow(f, i, idp)}
          {/each}
        </div>
      </details>
    {/if}
  {/if}
{/snippet}

<!-- ============ Editors (full-width modals) ============ -->
{#snippet modal(title: string, close: () => void, editor: import('svelte').Snippet, actions: import('svelte').Snippet | undefined)}
  <div
    class="rw-modal"
    role="dialog"
    aria-modal="true"
    aria-label={title}
    tabindex="-1"
    onkeydown={(e) => { if (e.key === 'Escape') close(); }}
    {@attach (el) => { el.focus(); }}
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="rw-scrim" onclick={close}></div>
    <div class="rw-panel">
      <div class="rw-panel-bar">
        <span class="rw-panel-title">{title}</span>
        <div class="rw-actions">
          {#if actions}{@render actions()}{/if}
          <button class="rw-btn" onclick={close}>Close <kbd>Esc</kbd></button>
        </div>
      </div>
      <div class="rw-panel-body">
        <div class="rw-pane">{@render editor()}</div>
        <div class="rw-pane rw-pane-preview">
          <span class="rw-pane-label">What {dest} receives</span>
          <div class="rw-pane-scroll">
            {@render preview()}
            <details class="rw-drawer">
              <summary>Request body</summary>
              <pre class="rw-out"><code>{renderResult.result}</code></pre>
            </details>
          </div>
        </div>
      </div>
    </div>
  </div>
{/snippet}

{#if payloadModalOpen}
  {#snippet payloadEditor()}
    <label class="rw-pane-label" for="rw_payload_modal">Payload</label>
    <textarea id="rw_payload_modal" bind:value={payloadStr} spellcheck="false" class="rw-code rw-pane-fill"></textarea>
  {/snippet}
  {#snippet payloadActions()}
    <span class="rw-note" class:rw-note-warn={!parsedPayload}>{parsedPayload ? 'Valid JSON' : 'Invalid JSON'}</span>
  {/snippet}
  {@render modal('Sample event payload', () => (payloadModalOpen = false), payloadEditor, payloadActions)}
{/if}

{#if templateModalOpen}
  {#snippet templateEditor()}
    <label class="rw-pane-label" for="rw_template_modal">Transform template</label>
    <textarea id="rw_template_modal" bind:value={template} onpaste={onTemplatePaste} spellcheck="false" class="rw-code rw-pane-fill"></textarea>
  {/snippet}
  {#snippet templateActions()}
    <button class="rw-btn" onclick={() => copyText(template, 'Copied template')}>Copy template</button>
  {/snippet}
  {@render modal('Transform template', () => (templateModalOpen = false), templateEditor, templateActions)}
{/if}

{#if fieldsModalOpen}
  {#snippet fieldsEditor()}
    <span class="rw-pane-label">Message</span>
    <div class="rw-pane-scroll" onfocusin={trackFocus}>{@render fieldsForm('m_')}</div>
  {/snippet}
  {#snippet fieldsActions()}
    <button class="rw-btn" onclick={() => copyText(template, 'Copied template')}>Copy template</button>
  {/snippet}
  {@render modal('Message', () => (fieldsModalOpen = false), fieldsEditor, fieldsActions)}
{/if}

<style>
  /* Brand tokens from sparrow-tokens.css; the fallbacks keep the page legible
     without them. The editors render outside .rw-root, so they get them too. */
  .rw-root, .rw-modal {
    --ink: var(--sp-on-surface, #1a1a1a);
    --ink-2: var(--sp-on-surface-variant, #55524c);
    --ink-3: var(--sp-outline, #8a867e);
    --rule: var(--sp-outline-variant, #e4e2dd);
    --paper: var(--sp-surface, #faf9f7);
    --sheet: var(--sp-surface-container-lowest, #ffffff);
    --well: var(--sp-surface-container-low, #f7f5f1);
    --amber: var(--sp-primary-container, #f2a93b);
    --amber-ink: var(--sp-primary, #b06a10);
    --amber-wash: var(--sp-brand-primary-10, rgba(242, 169, 59, 0.1));
    --caution: var(--sp-tertiary, #b45309);
    --danger: var(--sp-error, #c0392b);
    --danger-wash: var(--sp-error-container, #fbe3e0);
    --inverse: var(--sp-inverse-surface, #1a1a1a);
    --inverse-ink: var(--sp-inverse-on-surface, #faf9f7);
    --display: var(--sp-font-display, 'Space Grotesk', system-ui, sans-serif);
    --body: var(--sp-font-body, 'Inter', system-ui, sans-serif);
    --mono: var(--sp-font-mono, 'Fira Code', ui-monospace, monospace);

    font-family: var(--body);
    color: var(--ink);
    font-size: 14px;
    line-height: 1.5;
  }
  .rw-root { margin: 2.5rem 0 4rem; }
  .rw-root *:focus-visible, .rw-modal *:focus-visible { outline: 2px solid var(--amber); outline-offset: 2px; }
  .rw-root code, .rw-modal code { font-family: var(--mono); font-size: 0.92em; }

  /* ─── Title ──────────────────────────────────────────────── */
  .rw-header { margin-bottom: 1.75rem; }
  .rw-header h1 { font-family: var(--display); font-size: clamp(1.6rem, 4vw, 2rem); font-weight: 600; letter-spacing: -0.015em; line-height: 1.1; margin: 0 0 0.6rem; }
  .rw-header p { margin: 0; color: var(--ink-2); font-size: 1rem; max-width: 62ch; }

  .rw-toast { position: fixed; bottom: 1.5rem; right: 1.5rem; background: var(--inverse); color: var(--inverse-ink); padding: 0.5rem 0.9rem; font-size: 13px; z-index: 50; border-radius: 4px; }

  /* ─── Destination tabs: underlined names, not pills ──────── */
  .rw-tabs { display: flex; flex-wrap: wrap; gap: 0 1.5rem; border-bottom: 1px solid var(--rule); margin-bottom: 1.5rem; }
  .rw-tab { background: none; border: 0; border-bottom: 2px solid transparent; margin-bottom: -1px; padding: 0.5rem 0 0.6rem; font: inherit; font-weight: 500; color: var(--ink-2); cursor: pointer; }
  .rw-tab:hover { color: var(--ink); }
  .rw-tab[aria-current='true'] { color: var(--ink); border-bottom-color: var(--amber); }

  /* ─── Two columns: worksheet and receiver ────────────────── */
  .rw-grid { display: grid; grid-template-columns: minmax(0, 7fr) minmax(0, 5fr); gap: 2rem; align-items: start; }
  @media (max-width: 900px) { .rw-grid { grid-template-columns: 1fr; gap: 1.5rem; } }

  /* One sheet, sections separated by rules. No numbering: the
     template and the message can each be the starting point. */
  .rw-sheet { background: var(--sheet); border: 1px solid var(--rule); min-width: 0; }
  .rw-section { padding: 1.25rem 1.5rem 1.5rem; }
  .rw-section + .rw-section { border-top: 1px solid var(--rule); }
  .rw-section h2 { font-family: var(--body); font-size: 15px; font-weight: 600; margin: 0 0 0.25rem; line-height: 1.3; }
  .rw-section-head { display: flex; flex-wrap: wrap; justify-content: space-between; align-items: baseline; gap: 0.5rem 1rem; }
  .rw-section-head h2 { margin-bottom: 0.25rem; }
  .rw-help { color: var(--ink-2); font-size: 13px; margin: 0 0 0.9rem; max-width: 68ch; }
  .rw-help code { background: var(--well); padding: 0 0.3em; }

  /* The receiver is the one bold element: the only amber edge on the page. */
  .rw-receiver { position: sticky; top: 1.25rem; max-height: calc(100vh - 2.5rem); overflow-y: auto; background: var(--sheet); border: 1px solid var(--rule); border-top: 3px solid var(--amber); padding: 1.25rem 1.5rem 1.5rem; min-width: 0; }
  @media (max-width: 900px) { .rw-receiver { position: static; max-height: none; } }
  .rw-receiver-title { font-family: var(--display); font-size: 1.3rem; font-weight: 600; letter-spacing: -0.01em; margin: 0 0 0.25rem; line-height: 1.2; }
  .rw-receiver-title { margin-bottom: 1rem; }
  .rw-receiver .rw-drawer { margin-top: 1.1rem; padding-top: 0.9rem; border-top: 1px solid var(--rule); }

  /* ─── Text, notes, links, buttons ────────────────────────── */
  .rw-row { display: flex; justify-content: space-between; align-items: center; gap: 0.75rem; min-height: 1.6rem; }
  .rw-actions { display: flex; align-items: center; gap: 0.5rem; flex-shrink: 0; }
  .rw-note { font-size: 12px; color: var(--ink-3); margin: 0; }
  .rw-note code { color: var(--ink-2); }
  .rw-note-warn { color: var(--caution); }
  .rw-link { background: none; border: 0; padding: 0; font: inherit; font-size: 13px; color: var(--amber-ink); text-decoration: underline; text-underline-offset: 2px; cursor: pointer; }
  .rw-link:hover { color: var(--ink); }
  .rw-btn { font: inherit; font-size: 12.5px; font-weight: 500; color: var(--ink); background: var(--sheet); border: 1px solid var(--rule); padding: 0.25rem 0.6rem; cursor: pointer; border-radius: 3px; white-space: nowrap; }
  .rw-btn:hover { border-color: var(--ink-3); }
  .rw-btn kbd { font-family: var(--mono); font-size: 10.5px; color: var(--ink-3); margin-left: 0.2em; }

  /* ─── Inputs ─────────────────────────────────────────────── */
  .rw-stack { display: flex; flex-direction: column; gap: 0.9rem; }
  .rw-two { display: grid; grid-template-columns: 1fr 1fr; gap: 0.9rem; margin-bottom: 0.9rem; }
  .rw-field { display: flex; flex-direction: column; gap: 0.3rem; }
  .rw-field label { font-size: 12.5px; font-weight: 500; color: var(--ink-2); }
  .rw-field input, .rw-text, .rw-code { font: inherit; font-size: 13.5px; color: var(--ink); background: var(--sheet); border: 1px solid var(--rule); border-radius: 3px; padding: 0.45rem 0.6rem; width: 100%; box-sizing: border-box; }
  .rw-field input:hover, .rw-text:hover, .rw-code:hover { border-color: var(--ink-3); }
  .rw-code { font-family: var(--mono); font-size: 12.5px; line-height: 1.55; resize: vertical; tab-size: 2; }
  .rw-text { line-height: 1.5; resize: vertical; field-sizing: content; max-height: 14rem; }
  .rw-text-code { font-family: var(--mono); font-size: 12.5px; }
  .rw-pending { border-color: var(--caution); }
  .rw-check { display: flex; align-items: center; gap: 0.5rem; font-size: 13.5px; }
  .rw-check input { accent-color: var(--amber-ink); }

  /* Field paths are code, so they are set in mono; the branch
     condition sits beneath in the caution colour because it
     changes which value is sent. */
  .rw-path { font-family: var(--mono); font-size: 12.5px; font-weight: 500; color: var(--ink); }
  .rw-when { font-family: var(--mono); font-size: 11.5px; color: var(--caution); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

  .rw-chips { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-bottom: 1rem; }
  .rw-chip { font-family: var(--mono); font-size: 11.5px; color: var(--amber-ink); background: var(--amber-wash); border: 1px solid transparent; padding: 0.15rem 0.5rem; border-radius: 3px; cursor: pointer; }
  .rw-chip:hover { border-color: var(--amber-ink); }

  .rw-presets { display: flex; flex-wrap: wrap; gap: 0.3rem; margin-bottom: 0.9rem; }
  .rw-preset { font: inherit; font-size: 12.5px; color: var(--ink-2); background: var(--sheet); border: 1px solid var(--rule); padding: 0.25rem 0.6rem; border-radius: 3px; cursor: pointer; }
  .rw-preset[aria-pressed='true'] { color: var(--ink); border-color: var(--ink); }

  .rw-steps { margin: 0 0 0.75rem; padding-left: 1.2rem; display: flex; flex-direction: column; gap: 0.35rem; font-size: 13px; color: var(--ink-2); max-width: 68ch; }

  /* Drawers hold what you need sometimes: raw body, CLI, other values. */
  .rw-drawer { margin-top: 1rem; font-size: 13px; }
  .rw-drawer summary { cursor: pointer; color: var(--ink-2); font-weight: 500; list-style: none; display: flex; align-items: center; gap: 0.4rem; }
  .rw-drawer summary::-webkit-details-marker { display: none; }
  .rw-drawer summary::before { content: ''; width: 0.4em; height: 0.4em; border-right: 1.5px solid currentColor; border-bottom: 1.5px solid currentColor; transform: rotate(-45deg); transition: transform 120ms; }
  .rw-drawer[open] summary::before { transform: rotate(45deg); }
  .rw-drawer summary:hover { color: var(--ink); }
  .rw-drawer > :not(summary) { margin-top: 0.75rem; }
  @media (prefers-reduced-motion: reduce) { .rw-drawer summary::before { transition: none; } }

  .rw-out { background: var(--inverse); color: var(--inverse-ink); font-family: var(--mono); font-size: 12px; line-height: 1.55; padding: 0.75rem 0.9rem; margin: 0; overflow: auto; max-height: 320px; border-radius: 3px; }
  .rw-error { background: var(--danger-wash); color: var(--danger); padding: 0.5rem 0.75rem; font-size: 13px; margin-bottom: 0.75rem; border-radius: 3px; }

  /* ─── Destination mocks: each wears its own chrome ───────── */
  .rw-mock { min-width: 0; }

  .rw-email { border: 1px solid var(--rule); border-radius: 4px; overflow: hidden; }
  .rw-email-head { padding: 0.75rem 1rem; background: var(--well); border-bottom: 1px solid var(--rule); font-size: 13px; display: grid; gap: 0.2rem; }
  .rw-email-head strong { font-weight: 500; color: var(--ink-3); display: inline-block; width: 3.5em; }
  .rw-email-body { padding: 1rem; font-size: 14px; white-space: pre-wrap; line-height: 1.6; }
  .rw-email-html { white-space: normal; }

  .rw-sms { background: var(--well); border-radius: 14px; padding: 1rem; max-width: 340px; }
  .rw-sms-to { font-size: 12px; color: var(--ink-3); text-align: center; margin-bottom: 0.75rem; }
  .rw-sms-bubble { background: #34c759; color: #fff; padding: 0.6rem 0.9rem; border-radius: 18px 18px 4px 18px; font-size: 14px; line-height: 1.45; margin-left: auto; max-width: 85%; width: fit-content; white-space: pre-wrap; }
  .rw-sms-meta { font-size: 12px; color: var(--ink-3); margin-top: 0.75rem; text-align: right; }

  .rw-slack { border: 1px solid var(--rule); border-radius: 4px; padding: 0.9rem 1rem; }
  .rw-slack-bot { display: flex; align-items: center; gap: 0.45rem; margin-bottom: 0.5rem; }
  .rw-slack-avatar { width: 24px; height: 24px; background: var(--amber); color: #1a1204; display: inline-flex; align-items: center; justify-content: center; border-radius: 4px; font-size: 13px; font-weight: 700; }
  .rw-slack-name { font-weight: 700; font-size: 14px; }
  .rw-slack-app { font-size: 9.5px; background: var(--well); color: var(--ink-3); padding: 1px 4px; border-radius: 3px; font-weight: 700; }
  .rw-slack-blocks { display: flex; flex-direction: column; gap: 0.5rem; }
  .rw-slack-header { font-size: 17px; font-weight: 800; }
  .rw-slack-fields { display: grid; grid-template-columns: 1fr 1fr; gap: 0.4rem; }
  .rw-slack-text { font-size: 14px; white-space: pre-wrap; line-height: 1.45; }
  .rw-slack-context { font-size: 12px; color: var(--ink-3); }
  .rw-slack-divider { border: 0; border-top: 1px solid var(--rule); margin: 0.25rem 0; }

  .rw-discord { background: #313338; border-radius: 4px; padding: 1rem; }
  .rw-discord-content { color: #dbdee1; font-size: 14px; margin-bottom: 0.5rem; white-space: pre-wrap; }
  .rw-discord-embed { border-left: 4px solid #5865f2; background: #2b2d31; border-radius: 4px; padding: 0.75rem 1rem; color: #dbdee1; }
  .rw-discord-embed + .rw-discord-embed { margin-top: 0.5rem; }
  .rw-discord-title { font-weight: 700; margin-bottom: 0.5rem; font-size: 14px; }
  .rw-discord-desc { background: #1e1f22; border-radius: 4px; padding: 0.5rem; font-size: 12px; overflow-x: auto; margin: 0 0 0.5rem; }
  .rw-discord-field { font-size: 12.5px; margin-bottom: 0.4rem; }
  .rw-discord-footer { font-size: 11px; color: #949ba4; }

  .rw-ntfy { border: 1px solid var(--rule); border-radius: 4px; padding: 0.9rem 1rem; }
  .rw-ntfy-top { display: flex; justify-content: space-between; margin-bottom: 0.5rem; font-family: var(--mono); font-size: 12px; color: var(--ink-3); }
  .rw-ntfy-title { font-weight: 600; font-size: 14px; margin-bottom: 0.25rem; }
  .rw-ntfy-body { font: inherit; font-size: 13.5px; white-space: pre-wrap; margin: 0; }
  .rw-ntfy-tags { margin-top: 0.4rem; font-size: 12px; color: var(--ink-3); }

  .rw-pd { border: 1px solid var(--rule); border-left: 4px solid var(--danger); border-radius: 4px; padding: 0.9rem 1rem; }
  .rw-pd-head { display: flex; justify-content: space-between; margin-bottom: 0.5rem; font-size: 12px; color: var(--ink-3); }
  .rw-pd-sev { font-weight: 700; color: var(--danger); }
  .rw-pd-summary { font-weight: 700; font-size: 14px; margin-bottom: 0.35rem; }
  .rw-pd-details { background: var(--well); padding: 0.5rem; font-size: 11.5px; overflow-x: auto; margin: 0.5rem 0 0; border-radius: 3px; }

  .rw-ch { border: 1px solid var(--rule); border-radius: 4px; padding: 0.9rem 1rem; }
  .rw-table { width: 100%; border-collapse: collapse; font-size: 12.5px; margin-top: 0.5rem; }
  .rw-table th { text-align: left; font-family: var(--mono); font-weight: 500; color: var(--ink-3); padding: 0.3rem 0.75rem 0.3rem 0; vertical-align: top; white-space: nowrap; }
  .rw-table td { padding: 0.3rem 0; word-break: break-all; }
  .rw-table tr + tr { border-top: 1px solid var(--rule); }

  /* ─── Editors ────────────────────────────────────────────── */
  .rw-modal { position: fixed; inset: 0; z-index: 100; display: flex; align-items: center; justify-content: center; }
  .rw-scrim { position: fixed; inset: 0; background: rgba(26, 26, 26, 0.45); }
  .rw-panel { position: relative; z-index: 1; width: 92vw; max-width: 84rem; height: 86vh; background: var(--sheet); border: 1px solid var(--rule); border-top: 3px solid var(--amber); display: flex; flex-direction: column; overflow: hidden; }
  .rw-panel-bar { display: flex; align-items: center; justify-content: space-between; gap: 0.75rem; padding: 0.7rem 1.25rem; border-bottom: 1px solid var(--rule); flex-shrink: 0; }
  .rw-panel-title { font-family: var(--display); font-size: 1.05rem; font-weight: 600; }
  .rw-panel-body { display: grid; grid-template-columns: 1fr 1fr; flex: 1; min-height: 0; overflow: hidden; }
  @media (max-width: 700px) { .rw-panel-body { grid-template-columns: 1fr; grid-template-rows: 1fr 1fr; } }
  .rw-pane { display: flex; flex-direction: column; padding: 1rem 1.25rem; min-height: 0; overflow: hidden; }
  .rw-pane + .rw-pane { border-left: 1px solid var(--rule); }
  @media (max-width: 700px) { .rw-pane + .rw-pane { border-left: 0; border-top: 1px solid var(--rule); } }
  .rw-pane-label { font-size: 12.5px; font-weight: 500; color: var(--ink-2); margin-bottom: 0.5rem; flex-shrink: 0; }
  .rw-pane-fill { flex: 1; resize: none; min-height: 0; }
  .rw-pane-scroll { flex: 1; overflow-y: auto; min-height: 0; }
</style>
