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
    type EmailCompose,
    type SlackCompose,
    type DiscordCompose,
    type NtfyCompose,
    type PagerdutyCompose,
  } from '../../lib/compose.js';
  import { onMount } from 'svelte';

  // ---- Recipe selection ----
  let selectedId = $state('sendgrid');
  let activeRecipe = $derived(RECIPES.find((r) => r.name === selectedId)!);
  let meta = $derived(RECIPE_META[selectedId]);

  const COMPOSERS = new Set(['sendgrid', 'twilio', 'slack', 'discord', 'ntfy', 'pagerduty']);
  // Step-2 heading per recipe; clickhouse is a pure data sink (no composer).
  const STEP2_TITLE: Record<string, string> = {
    sendgrid: 'Compose the email',
    twilio: 'Compose the SMS',
    slack: 'Compose the Slack message',
    discord: 'Compose the Discord embed',
    ntfy: 'Compose the notification',
    pagerduty: 'Compose the incident',
    clickhouse: 'Column mapping',
  };

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

  // ---- Composer state ----
  let email = $state<EmailCompose>({
    toMode: 'list',
    listPath: '.payload.alert_recipients',
    listField: 'email',
    to: '{{.payload.customer_email}}',
    cc: '',
    bcc: '',
    subject: 'Sparrow: webhook for {{.payload.consumer}} is now {{.payload.new_health}}',
    bodyText:
      'Webhook {{.payload.webhook_id}} ({{.payload.url}}) health changed: {{.payload.old_health}} -> {{.payload.new_health}}',
  });
  let sms = $state('Sparrow {{.event_name}}: {{.payload.active_incidents}} active incidents in {{.payload.datacenter}}');
  let slack = $state<SlackCompose>({
    header: '{{.event_name}}',
    body: 'Service *{{.payload.service}}* deployed `{{.payload.version}}` to {{.payload.environment}} in {{.payload.duration_seconds}}s',
    includeContext: true,
  });
  let discord = $state<DiscordCompose>({
    title: '{{.event_name}}',
    description: 'New signup: **{{.payload.user_id}}** on the {{.payload.plan}} plan (via {{.payload.referrer}}).',
    includePayload: true,
  });
  let ntfy = $state<NtfyCompose>({
    title: 'High CPU on {{.payload.host}}',
    message: 'CPU usage hit {{.payload.usage_percent}}% — check the host.',
    tags: 'warning, computer',
    priority: 4,
    includePayload: false,
  });
  let pd = $state<PagerdutyCompose>({
    summary: 'Sparrow: {{.event_name}}',
    severity: 'error',
    source: 'sparrow',
  });

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

  let template = $derived.by(() => {
    if (selectedId === 'sendgrid') return generateSendgridTemplate(email);
    if (selectedId === 'twilio') return generateTwilioTemplate(sms);
    if (selectedId === 'slack') return generateSlackTemplate(slack);
    if (selectedId === 'discord') return generateDiscordTemplate(discord);
    if (selectedId === 'ntfy') return generateNtfyTemplate(ntfy);
    if (selectedId === 'pagerduty') return generatePagerdutyTemplate(pd);
    return activeRecipe.transform_template; // clickhouse: pure data sink, no composer
  });

  let renderResult = $derived.by(() => {
    if (!parsedPayload) return { result: '', error: 'Invalid JSON in event payload', jsonObj: undefined as any };
    return renderGoTemplate(template, eventContext, params);
  });

  let chips = $derived([...ENVELOPE_PATHS, ...payloadPaths(parsedPayload || {})]);

  let smsRendered = $derived.by(() => (parsedPayload ? renderGoTemplate(sms, eventContext, params).result : ''));
  let smsInfo = $derived(smsSegments(smsRendered));

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
  }

  function selectPreset(presetId: string) {
    const p = meta.presets.find((x) => x.id === presetId) ?? RECIPE_META[selectedId].presets[0];
    selectedPresetId = p.id;
    eventName = p.event_name;
    payloadStr = JSON.stringify(p.payload, null, 2);
    if (selectedId === 'sendgrid') {
      email.toMode = 'alert_recipients' in p.payload ? 'list' : 'addresses';
    }
  }

  // Variable chips insert into the last focused text field of the compose card.
  let lastField: HTMLInputElement | HTMLTextAreaElement | null = null;
  function trackFocus(e: FocusEvent) {
    const t = e.target as HTMLElement;
    if (t instanceof HTMLInputElement || t instanceof HTMLTextAreaElement) lastField = t;
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
  // Editable copy of the template shown in the template modal.
  // Initialized from the auto-generated template when the modal opens;
  // edits here re-render the preview in real time inside the modal.
  let templateDraft = $state('');
  let templateDraftRender = $derived.by(() => {
    if (!parsedPayload || !templateDraft) return { result: '', error: '' };
    return renderGoTemplate(templateDraft, eventContext, params);
  });

  function openTemplateModal() {
    templateDraft = template;
    templateModalOpen = true;
  }

  let copiedToast = $state<string | null>(null);
  function copyText(txt: string, msg: string) {
    navigator.clipboard.writeText(txt);
    copiedToast = msg;
    setTimeout(() => (copiedToast = null), 2000);
  }

  // A caller (e.g. Sparrow's subscription editor) can deep-link a sample event
  // payload as ?sample=<base64-utf8 JSON>; seed the payload field with it.
  onMount(() => {
    const raw = new URLSearchParams(location.search).get('sample');
    if (!raw) return;
    try {
      const json = decodeURIComponent(escape(atob(raw)));
      payloadStr = JSON.stringify(JSON.parse(json), null, 2);
    } catch {
      /* malformed param — keep the preset payload */
    }
  });
</script>

<div class="rw-root">
  <header class="rw-header">
    <span class="rw-badge">SATELLITE WORKBENCH</span>
    <h1>Satellite Recipe Workbench</h1>
    <p>
      Compose the message, and the workbench writes the Sparrow transform template for you. Preview exactly what the
      destination receives, then apply it with one CLI command.
    </p>
  </header>

  {#if copiedToast}
    <div class="rw-toast">{copiedToast}</div>
  {/if}

  <nav class="rw-recipe-tabs">
    {#each RECIPES as r}
      <button class="rw-tab {selectedId === r.name ? 'rw-tab-active' : ''}" onclick={() => switchRecipe(r.name)}>
        {RECIPE_META[r.name].title}
      </button>
    {/each}
  </nav>

  <div class="rw-tagline-bar"><strong>{meta.title}:</strong> {activeRecipe.description}</div>

  <div class="rw-grid">
    <!-- Left: compose -->
    <div class="rw-col">
      <!-- Sample event -->
      <div class="rw-card">
        <h3>1. Sample event</h3>
        <p class="rw-sub">The event Sparrow delivers. Pick a preset or paste your own payload — its fields become the variable chips below.</p>
        {#if meta.presets.length > 1}
          <div class="rw-pills">
            {#each meta.presets as p}
              <button class="rw-pill {selectedPresetId === p.id ? 'rw-pill-active' : ''}" onclick={() => selectPreset(p.id)}>
                {p.label}
              </button>
            {/each}
          </div>
        {/if}
        <div class="rw-form-grid">
          <div class="rw-field">
            <label for="rw_event_name">Event name</label>
            <input id="rw_event_name" type="text" bind:value={eventName} />
          </div>
          <div class="rw-field">
            <label for="rw_attempt">Attempt</label>
            <input id="rw_attempt" type="number" bind:value={attempt} min="1" />
          </div>
        </div>
        <div class="rw-field" style="margin-top: 0.75rem;">
          <div class="rw-field-header">
            <label for="rw_payload">Payload JSON</label>
            <div class="rw-field-actions">
              <span class="rw-status-text">{parsedPayload ? '✓ Valid JSON' : '⚠️ Invalid JSON'}</span>
              <button class="rw-btn rw-btn-secondary rw-btn-sm" onclick={() => (payloadModalOpen = true)}>Edit ↗</button>
            </div>
          </div>
          <textarea id="rw_payload" bind:value={payloadStr} rows="10" class="rw-code-textarea"></textarea>
        </div>
      </div>

      <!-- Composer -->
      <div class="rw-card" onfocusin={trackFocus}>
        <h3>2. {STEP2_TITLE[selectedId] ?? 'Compose the payload'}</h3>

        {#if COMPOSERS.has(selectedId)}
          <p class="rw-sub">Chips insert the event field at the cursor of the last text field you clicked into.</p>
          <div class="rw-chips">
            {#each chips as c}
              <button class="rw-chip" onclick={() => insertChip(c)} title={'{{' + c + '}}'}>{c}</button>
            {/each}
          </div>
        {/if}

        {#if selectedId === 'sendgrid'}
          <div class="rw-field">
            <label for="em_to">To</label>
            <div class="rw-to-row">
              <select bind:value={email.toMode}>
                <option value="addresses">Addresses</option>
                <option value="list">Everyone in a payload list</option>
              </select>
              {#if email.toMode === 'addresses'}
                <input id="em_to" type="text" bind:value={email.to} placeholder={'jane@acme.com, {{.payload.customer_email}}'} />
              {:else}
                <input id="em_to" type="text" bind:value={email.listPath} placeholder=".payload.alert_recipients" title="Go accessor of a payload array" />
                <input class="rw-to-field" type="text" bind:value={email.listField} placeholder="email" title="Address field on each list item; leave empty when the list holds plain strings" />
              {/if}
            </div>
            {#if email.toMode === 'list'}
              <p class="rw-sub">
                One email per address in the list — recipients never see each other. Falls back to the
                <code>default_recipient</code> parameter when the list is missing. Second box: the field on each item
                holding the address (empty for a list of plain strings).
              </p>
            {:else}
              <div class="rw-form-grid" style="margin-top: 0.5rem;">
                <div class="rw-field">
                  <label for="em_cc">Cc (optional)</label>
                  <input id="em_cc" type="text" bind:value={email.cc} placeholder="manager@acme.com" />
                </div>
                <div class="rw-field">
                  <label for="em_bcc">Bcc (optional)</label>
                  <input id="em_bcc" type="text" bind:value={email.bcc} placeholder="audit@acme.com" />
                </div>
              </div>
            {/if}
          </div>
          <div class="rw-field">
            <label for="em_subject">Subject</label>
            <input id="em_subject" type="text" bind:value={email.subject} />
          </div>
          <div class="rw-field">
            <label for="em_body">Body</label>
            <textarea id="em_body" rows="6" bind:value={email.bodyText} class="rw-body-textarea"></textarea>
          </div>

        {:else if selectedId === 'twilio'}
          <div class="rw-field">
            <div class="rw-field-header">
              <label for="sms_msg">Message</label>
              <span class="rw-status-text" class:rw-multi-segment={smsInfo.segments > 1}>{smsInfo.chars} chars · {smsInfo.segments} segment{smsInfo.segments === 1 ? '' : 's'} ({smsInfo.encoding}){smsInfo.segments > 1 ? ' — billed per segment' : ''}</span>
            </div>
            <textarea id="sms_msg" rows="4" bind:value={sms} class="rw-body-textarea"></textarea>
          </div>

        {:else if selectedId === 'slack'}
          <div class="rw-field">
            <label for="sl_header">Header</label>
            <input id="sl_header" type="text" bind:value={slack.header} />
          </div>
          <div class="rw-field">
            <label for="sl_body">Message (mrkdwn: *bold*, `code`)</label>
            <textarea id="sl_body" rows="4" bind:value={slack.body} class="rw-body-textarea"></textarea>
          </div>
          <label class="rw-check">
            <input type="checkbox" bind:checked={slack.includeContext} />
            Include event context fields (ID, timestamp, attempt)
          </label>

        {:else if selectedId === 'discord'}
          <div class="rw-field">
            <label for="dc_title">Title</label>
            <input id="dc_title" type="text" bind:value={discord.title} />
          </div>
          <div class="rw-field">
            <label for="dc_desc">Description (Markdown: **bold**, `code`)</label>
            <textarea id="dc_desc" rows="3" bind:value={discord.description} class="rw-body-textarea"></textarea>
          </div>
          <label class="rw-check">
            <input type="checkbox" bind:checked={discord.includePayload} />
            Append the full event payload as a JSON code block
          </label>

        {:else if selectedId === 'ntfy'}
          <div class="rw-field">
            <label for="nt_title">Title</label>
            <input id="nt_title" type="text" bind:value={ntfy.title} />
          </div>
          <div class="rw-field">
            <label for="nt_msg">Message</label>
            <textarea id="nt_msg" rows="3" bind:value={ntfy.message} class="rw-body-textarea"></textarea>
          </div>
          <div class="rw-form-grid" style="margin-top: 0.5rem;">
            <div class="rw-field">
              <label for="nt_tags">Tags (comma-separated; ntfy renders emoji)</label>
              <input id="nt_tags" type="text" bind:value={ntfy.tags} placeholder="warning, computer" />
            </div>
            <div class="rw-field">
              <label for="nt_prio">Priority</label>
              <select id="nt_prio" bind:value={ntfy.priority}>
                <option value={5}>5 — max / urgent</option>
                <option value={4}>4 — high</option>
                <option value={3}>3 — default</option>
                <option value={2}>2 — low</option>
                <option value={1}>1 — min</option>
              </select>
            </div>
          </div>
          <label class="rw-check">
            <input type="checkbox" bind:checked={ntfy.includePayload} />
            Append the full event payload to the message
          </label>

        {:else if selectedId === 'pagerduty'}
          <div class="rw-field">
            <label for="pd_summary">Summary</label>
            <input id="pd_summary" type="text" bind:value={pd.summary} />
          </div>
          <div class="rw-form-grid" style="margin-top: 0.5rem;">
            <div class="rw-field">
              <label for="pd_sev">Severity</label>
              <select id="pd_sev" bind:value={pd.severity}>
                <option value="critical">critical</option>
                <option value="error">error</option>
                <option value="warning">warning</option>
                <option value="info">info</option>
              </select>
            </div>
            <div class="rw-field">
              <label for="pd_source">Source</label>
              <input id="pd_source" type="text" bind:value={pd.source} />
            </div>
          </div>
          <p class="rw-sub">The full event payload is attached as <code>custom_details</code>; the dedup key is the event ID.</p>

        {:else}
          <p class="rw-sub">
            A straight data sink — nothing to compose. Each event is inserted as one row via ClickHouse's
            JSONEachRow HTTP endpoint, mapping envelope fields to fixed columns:
          </p>
          <table class="rw-ch-table">
            <tbody>
              <tr><th>event_id</th><td>envelope event ID</td></tr>
              <tr><th>event_name</th><td>envelope event name</td></tr>
              <tr><th>timestamp</th><td>delivery timestamp</td></tr>
              <tr><th>payload</th><td>full event payload as a JSON string</td></tr>
            </tbody>
          </table>
          <p class="rw-sub">Set the destination with the <code>table</code> parameter below; the stored template is shown next.</p>
        {/if}
      </div>

      <!-- Go transform template (what Sparrow stores) -->
      <div class="rw-card">
        <div class="rw-card-header">
          <h3>Go transform template (what Sparrow stores)</h3>
          <div class="rw-field-actions">
            <button class="rw-btn rw-btn-secondary rw-btn-sm" onclick={openTemplateModal}>Edit ↗</button>
            <button class="rw-btn rw-btn-secondary rw-btn-sm" onclick={() => copyText(template, 'Copied template')}>Copy</button>
          </div>
        </div>
        <pre class="rw-raw-code"><code>{template}</code></pre>
      </div>

      <!-- Parameters -->
      <div class="rw-card">
        <h3>3. Recipe parameters</h3>
        <p class="rw-sub">Asked once when you apply the recipe; secrets are envelope-encrypted at rest.</p>
        <div class="rw-form-list">
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
      </div>

      <!-- Setup guide -->
      <div class="rw-card">
        <h3>4. How to set up {meta.title}</h3>
        <p class="rw-sub">Where the credentials above come from, per the official docs.</p>
        <ol class="rw-setup-steps">
          {#each meta.setup.steps as s}
            <li>{s}</li>
          {/each}
        </ol>
        <a class="rw-btn-text" href={meta.setup.docsUrl} target="_blank" rel="noopener noreferrer">
          {meta.setup.docsLabel} →
        </a>
      </div>
    </div>

    <!-- Right: preview + apply -->
    <div class="rw-col">
      <div class="rw-card rw-preview-container">
        <div class="rw-card-header">
          <h3>What {meta.title} receives</h3>
          <span class="rw-badge-sm">{selectedId}</span>
        </div>

        {#if renderResult.error}
          <div class="rw-error-box">{renderResult.error}</div>
        {/if}

        <div class="rw-visual-box">
          {#if selectedId === 'sendgrid'}
            <div class="rw-email-card">
              <div class="rw-email-header">
                <div><strong>Subject:</strong> {renderResult.jsonObj?.subject || '(No subject)'}</div>
                <div><strong>From:</strong> {params.from_name} &lt;{params.from_email}&gt;</div>
                <div>
                  <strong>To:</strong>
                  {emailAddresses.to.length ? emailAddresses.to.join(', ') : '(no recipients)'}
                </div>
                {#if emailAddresses.cc.length}
                  <div><strong>Cc:</strong> {emailAddresses.cc.join(', ')}</div>
                {/if}
                {#if emailAddresses.bcc.length}
                  <div><strong>Bcc:</strong> {emailAddresses.bcc.join(', ')}</div>
                {/if}
              </div>
              <div class="rw-email-body">{emailBodyRendered}</div>
            </div>

          {:else if selectedId === 'twilio'}
            <div class="rw-twilio-card">
              <div class="rw-sms-header">To: +{params.to_number}</div>
              <div class="rw-sms-bubble">{smsRendered}</div>
            </div>

          {:else if selectedId === 'slack'}
            <div class="rw-slack-card">
              <div class="rw-slack-bot">
                <span class="rw-slack-avatar">S</span>
                <span class="rw-slack-name">Sparrow Bot</span>
                <span class="rw-slack-badge">APP</span>
              </div>
              {#if renderResult.jsonObj?.blocks}
                <div class="rw-slack-blocks">
                  {#each renderResult.jsonObj.blocks as block}
                    {#if block.type === 'header'}
                      <div class="rw-slack-header">{block.text?.text}</div>
                    {:else if block.type === 'section' && block.fields}
                      <div class="rw-slack-fields">
                        {#each block.fields as f}
                          <div class="rw-slack-field">{@html mrkdwn(f.text ?? '')}</div>
                        {/each}
                      </div>
                    {:else if block.type === 'section' && block.text}
                      <div class="rw-slack-text">{@html mrkdwn(block.text?.text ?? '')}</div>
                    {/if}
                  {/each}
                </div>
              {:else}
                <pre><code>{renderResult.result}</code></pre>
              {/if}
            </div>

          {:else if selectedId === 'discord'}
            <div class="rw-discord-card">
              <div class="rw-discord-embed">
                <div class="rw-discord-title">{renderResult.jsonObj?.embeds?.[0]?.title || eventName}</div>
                <pre class="rw-discord-desc"><code>{renderResult.jsonObj?.embeds?.[0]?.description}</code></pre>
                <div class="rw-discord-footer">{renderResult.jsonObj?.embeds?.[0]?.timestamp}</div>
              </div>
            </div>

          {:else if selectedId === 'ntfy'}
            <div class="rw-ntfy-card">
              <div class="rw-ntfy-top">
                <span class="rw-ntfy-topic">{(params.server_url || '').replace(/^https?:\/\//, '')}/{params.topic}</span>
              </div>
              <div class="rw-ntfy-title">{renderResult.jsonObj?.title ?? eventName}</div>
              <pre class="rw-ntfy-body">{renderResult.jsonObj?.message ?? renderResult.result}</pre>
              {#if renderResult.jsonObj?.tags?.length}
                <div class="rw-ntfy-tags">{renderResult.jsonObj.tags.map((t: string) => `#${t}`).join(' ')}</div>
              {/if}
            </div>

          {:else if selectedId === 'pagerduty'}
            <div class="rw-pd-card">
              <div class="rw-pd-header">
                <span class="rw-pd-badge">PAGERDUTY INCIDENT</span>
                <span class="rw-pd-sev">{String(renderResult.jsonObj?.payload?.severity || '').toUpperCase()}</span>
              </div>
              <div class="rw-pd-summary">{renderResult.jsonObj?.payload?.summary}</div>
              <div class="rw-pd-meta">Dedup key: <code>{renderResult.jsonObj?.dedup_key}</code></div>
              <pre class="rw-pd-details"><code>{JSON.stringify(renderResult.jsonObj?.payload?.custom_details, null, 2)}</code></pre>
            </div>

          {:else if selectedId === 'clickhouse'}
            <div class="rw-ch-card">
              <div class="rw-ch-title">Row inserted into <code>{params.table}</code> (JSONEachRow)</div>
              {#if renderResult.jsonObj}
                <table class="rw-ch-table">
                  <tbody>
                    <tr><th>event_id</th><td>{renderResult.jsonObj.event_id}</td></tr>
                    <tr><th>event_name</th><td>{renderResult.jsonObj.event_name}</td></tr>
                    <tr><th>timestamp</th><td>{renderResult.jsonObj.timestamp}</td></tr>
                    <tr><th>payload</th><td><code>{renderResult.jsonObj.payload}</code></td></tr>
                  </tbody>
                </table>
              {:else}
                <pre><code>{renderResult.result}</code></pre>
              {/if}
            </div>
          {/if}
        </div>

        <details class="rw-raw-toggle">
          <summary>Rendered request body</summary>
          <div class="rw-raw-section">
            <div class="rw-raw-head">
              <span>Rendered request body sent to {activeRecipe.webhook.url.split('/')[2] || 'destination'}</span>
              <button class="rw-btn-text" onclick={() => copyText(renderResult.result, 'Copied output')}>Copy</button>
            </div>
            <pre class="rw-raw-code"><code>{renderResult.result}</code></pre>
            <p class="rw-sub">Preview is rendered by a simplified in-browser engine; the server uses Go's text/template.</p>
          </div>
        </details>
      </div>

      <div class="rw-card">
        <div class="rw-card-header">
          <h3>Apply with the Sparrow CLI</h3>
          <button class="rw-btn rw-btn-secondary rw-btn-sm" onclick={() => copyText(cliCommand, 'Copied CLI command')}>
            Copy command
          </button>
        </div>
        <p class="rw-sub">
          Registers this recipe on your Sparrow server. To use a customized template, save the generated template above
          into your own recipe file first.
        </p>
        <pre class="rw-cli-block"><code>{cliCommand}</code></pre>
      </div>
    </div>
  </div>
</div>

<!-- ============ Payload Edit Modal ============ -->
{#if payloadModalOpen}
  <div
    class="rw-modal-backdrop"
    role="dialog"
    aria-modal="true"
    aria-label="Edit Payload JSON"
    tabindex="-1"
    onkeydown={(e) => { if (e.key === 'Escape') payloadModalOpen = false; }}
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="rw-modal-scrim" onclick={() => (payloadModalOpen = false)}></div>
    <div class="rw-modal-panel">
      <div class="rw-modal-bar">
        <span class="rw-badge">PAYLOAD EDITOR</span>
        <div class="rw-field-actions">
          <span class="rw-status-text">{parsedPayload ? '✓ Valid JSON' : '⚠️ Invalid JSON'}</span>
          <button class="rw-btn rw-btn-secondary rw-btn-sm" onclick={() => (payloadModalOpen = false)}>Close <span class="rw-kbd">Esc</span></button>
        </div>
      </div>
      <div class="rw-modal-body">
        <div class="rw-modal-editor">
          <label class="rw-modal-pane-label" for="rw_payload_modal">Payload JSON</label>
          <textarea id="rw_payload_modal" bind:value={payloadStr} class="rw-code-textarea rw-modal-textarea"></textarea>
        </div>
        <div class="rw-modal-preview">
          <span class="rw-modal-pane-label">Live preview — what {meta.title} receives</span>
          <div class="rw-modal-preview-scroll">
            {#if renderResult.error}
              <div class="rw-error-box">{renderResult.error}</div>
            {/if}
            <div class="rw-visual-box">
              {#if selectedId === 'sendgrid'}
                <div class="rw-email-card">
                  <div class="rw-email-header">
                    <div><strong>Subject:</strong> {renderResult.jsonObj?.subject || '(No subject)'}</div>
                    <div><strong>From:</strong> {params.from_name} &lt;{params.from_email}&gt;</div>
                    <div><strong>To:</strong> {emailAddresses.to.length ? emailAddresses.to.join(', ') : '(no recipients)'}</div>
                  </div>
                  <div class="rw-email-body">{emailBodyRendered}</div>
                </div>
              {:else if selectedId === 'twilio'}
                <div class="rw-twilio-card"><div class="rw-sms-header">To: +{params.to_number}</div><div class="rw-sms-bubble">{smsRendered}</div></div>
              {:else if selectedId === 'slack'}
                <div class="rw-slack-card">
                  <div class="rw-slack-bot"><span class="rw-slack-avatar">S</span><span class="rw-slack-name">Sparrow Bot</span><span class="rw-slack-badge">APP</span></div>
                  {#if renderResult.jsonObj?.blocks}
                    <div class="rw-slack-blocks">
                      {#each renderResult.jsonObj.blocks as block}
                        {#if block.type === 'header'}<div class="rw-slack-header">{block.text?.text}</div>
                        {:else if block.type === 'section' && block.fields}<div class="rw-slack-fields">{#each block.fields as f}<div class="rw-slack-field">{@html mrkdwn(f.text ?? '')}</div>{/each}</div>
                        {:else if block.type === 'section' && block.text}<div class="rw-slack-text">{@html mrkdwn(block.text?.text ?? '')}</div>{/if}
                      {/each}
                    </div>
                  {:else}<pre><code>{renderResult.result}</code></pre>{/if}
                </div>
              {:else if selectedId === 'discord'}
                <div class="rw-discord-card"><div class="rw-discord-embed"><div class="rw-discord-title">{renderResult.jsonObj?.embeds?.[0]?.title || eventName}</div><pre class="rw-discord-desc"><code>{renderResult.jsonObj?.embeds?.[0]?.description}</code></pre></div></div>
              {:else if selectedId === 'ntfy'}
                <div class="rw-ntfy-card"><div class="rw-ntfy-title">{renderResult.jsonObj?.title ?? eventName}</div><pre class="rw-ntfy-body">{renderResult.jsonObj?.message ?? renderResult.result}</pre></div>
              {:else if selectedId === 'pagerduty'}
                <div class="rw-pd-card"><div class="rw-pd-header"><span class="rw-pd-badge">PAGERDUTY INCIDENT</span><span class="rw-pd-sev">{String(renderResult.jsonObj?.payload?.severity || '').toUpperCase()}</span></div><div class="rw-pd-summary">{renderResult.jsonObj?.payload?.summary}</div><pre class="rw-pd-details"><code>{JSON.stringify(renderResult.jsonObj?.payload?.custom_details, null, 2)}</code></pre></div>
              {:else}
                <pre class="rw-raw-code"><code>{renderResult.result}</code></pre>
              {/if}
            </div>
            <details class="rw-raw-toggle" style="margin-top:0.75rem">
              <summary>Rendered request body (JSON)</summary>
              <pre class="rw-raw-code" style="margin-top:0.5rem"><code>{renderResult.result}</code></pre>
            </details>
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}

<!-- ============ Template Edit Modal ============ -->
{#if templateModalOpen}
  <div
    class="rw-modal-backdrop"
    role="dialog"
    aria-modal="true"
    aria-label="Edit Transform Template"
    tabindex="-1"
    onkeydown={(e) => { if (e.key === 'Escape') templateModalOpen = false; }}
  >
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <!-- svelte-ignore a11y_no_static_element_interactions -->
    <div class="rw-modal-scrim" onclick={() => (templateModalOpen = false)}></div>
    <div class="rw-modal-panel">
      <div class="rw-modal-bar">
        <span class="rw-badge">TEMPLATE EDITOR</span>
        <div class="rw-field-actions">
          <button class="rw-btn rw-btn-secondary rw-btn-sm" onclick={() => { templateDraft = template; }}>Reset to generated</button>
          <button class="rw-btn rw-btn-secondary rw-btn-sm" onclick={() => copyText(templateDraft, 'Copied template')}>Copy</button>
          <button class="rw-btn rw-btn-secondary rw-btn-sm" onclick={() => (templateModalOpen = false)}>Close <span class="rw-kbd">Esc</span></button>
        </div>
      </div>
      <div class="rw-modal-body">
        <div class="rw-modal-editor">
          <label class="rw-modal-pane-label" for="rw_template_modal">Go transform template</label>
          <textarea id="rw_template_modal" bind:value={templateDraft} class="rw-code-textarea rw-modal-textarea"></textarea>
        </div>
        <div class="rw-modal-preview">
          <span class="rw-modal-pane-label">Live preview — rendered request body</span>
          <div class="rw-modal-preview-scroll">
            {#if templateDraftRender.error}
              <div class="rw-error-box">{templateDraftRender.error}</div>
            {/if}
            <pre class="rw-raw-code"><code>{templateDraftRender.result}</code></pre>
          </div>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .rw-root { display: flex; flex-direction: column; gap: 1rem; font-family: var(--sp-font-body, system-ui, sans-serif); color: var(--sp-on-surface, #1f2937); margin: 1.5rem 0 3rem; }
  .rw-header { padding: 1.25rem 1.5rem; background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; }
  .rw-header h1 { margin: 0.35rem 0 0.5rem; font-size: 1.6rem; }
  .rw-header p { margin: 0; color: #4b5563; font-size: 0.95rem; max-width: 56rem; }
  .rw-badge { font-family: var(--sp-font-mono, monospace); font-size: 10px; font-weight: 700; letter-spacing: 0.08em; color: #b06a10; background: #fef3c7; padding: 2px 8px; border-radius: 4px; }
  .rw-toast { position: fixed; bottom: 1.5rem; right: 1.5rem; background: #111827; color: #fff; padding: 0.5rem 1rem; border-radius: 8px; font-size: 13px; z-index: 50; }

  .rw-recipe-tabs { display: flex; flex-wrap: wrap; gap: 0.4rem; }
  .rw-tab { padding: 0.45rem 0.9rem; border: 1px solid #e5e7eb; background: #fff; border-radius: 999px; font-size: 13px; font-weight: 600; cursor: pointer; color: #4b5563; }
  .rw-tab-active { background: #b06a10; border-color: #b06a10; color: #fff; }
  .rw-tagline-bar { font-size: 13px; color: #4b5563; padding: 0.5rem 0.75rem; background: #f9fafb; border: 1px solid #e5e7eb; border-radius: 8px; }

  .rw-grid { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: 1rem; align-items: start; }
  @media (max-width: 900px) { .rw-grid { grid-template-columns: 1fr; } }
  .rw-col { display: flex; flex-direction: column; gap: 1rem; min-width: 0; }

  .rw-card { background: #fff; border: 1px solid #e5e7eb; border-radius: 12px; padding: 1rem 1.25rem; }
  .rw-card h3 { margin: 0 0 0.25rem; font-size: 1rem; }
  .rw-card-header { display: flex; justify-content: space-between; align-items: center; gap: 0.5rem; }
  .rw-sub { font-size: 12.5px; color: #6b7280; margin: 0.25rem 0 0.75rem; }
  .rw-badge-sm { font-family: monospace; font-size: 11px; background: #f3f4f6; padding: 2px 8px; border-radius: 4px; }

  .rw-pills { display: flex; flex-wrap: wrap; gap: 0.4rem; margin-bottom: 0.75rem; }
  .rw-pill { font-size: 12px; padding: 0.3rem 0.7rem; border-radius: 999px; border: 1px solid #e5e7eb; background: #fff; cursor: pointer; color: #4b5563; }
  .rw-pill-active { border-color: #b06a10; color: #b06a10; background: #fffbeb; font-weight: 600; }

  .rw-form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem; }
  .rw-form-list { display: flex; flex-direction: column; gap: 0.75rem; }
  .rw-field { display: flex; flex-direction: column; gap: 0.3rem; }
  .rw-field label, .rw-label { font-size: 12px; font-weight: 600; color: #374151; }
  .rw-field input, .rw-field select { padding: 0.45rem 0.6rem; border: 1px solid #d1d5db; border-radius: 6px; font-size: 13px; font-family: inherit; }
  .rw-field-header { display: flex; justify-content: space-between; align-items: center; }
  .rw-status-text { font-size: 11.5px; color: #6b7280; }
  .rw-to-row { display: flex; gap: 0.5rem; }
  .rw-to-row select { flex: 0 0 auto; }
  .rw-to-row input { flex: 1; }
  .rw-to-row .rw-to-field { flex: 0 0 110px; }
  .rw-check { display: flex; align-items: center; gap: 0.5rem; font-size: 13px; margin-top: 0.5rem; }

  .rw-chips { display: flex; flex-wrap: wrap; gap: 0.35rem; margin-bottom: 0.75rem; }
  .rw-chip { font-family: monospace; font-size: 11.5px; padding: 0.2rem 0.55rem; background: #eef2ff; color: #4338ca; border: 1px solid #c7d2fe; border-radius: 999px; cursor: pointer; }
  .rw-chip:hover { background: #e0e7ff; }
  .rw-code-textarea, .rw-body-textarea { font-family: var(--sp-font-mono, monospace); font-size: 12.5px; padding: 0.6rem; border: 1px solid #d1d5db; border-radius: 6px; resize: vertical; line-height: 1.5; width: 100%; box-sizing: border-box; }
  .rw-body-textarea { font-family: inherit; font-size: 13.5px; }
  .rw-btn-text { background: none; border: none; color: #b06a10; font-size: 12px; font-weight: 600; cursor: pointer; padding: 0.25rem 0; align-self: flex-start; }
  a.rw-btn-text { text-decoration: none; }
  .rw-setup-steps { margin: 0 0 0.5rem; padding-left: 1.1rem; display: flex; flex-direction: column; gap: 0.35rem; font-size: 12.5px; color: #374151; line-height: 1.45; }
  .rw-setup-steps li { padding-left: 0.15rem; }
  .rw-btn { padding: 0.4rem 0.9rem; border-radius: 6px; font-size: 13px; font-weight: 600; cursor: pointer; border: 1px solid transparent; }
  .rw-btn-secondary { background: #f3f4f6; border-color: #e5e7eb; color: #1f2937; }
  .rw-btn-sm { padding: 0.3rem 0.7rem; font-size: 12px; }

  .rw-error-box { background: #fef2f2; border: 1px solid #fecaca; color: #b91c1c; padding: 0.5rem 0.75rem; border-radius: 6px; font-size: 12.5px; margin-bottom: 0.75rem; }
  .rw-visual-box { margin-top: 0.5rem; }

  /* Email preview */
  .rw-email-card { border: 1px solid #e5e7eb; border-radius: 8px; overflow: hidden; }
  .rw-email-header { padding: 0.75rem 1rem; background: #f9fafb; border-bottom: 1px solid #e5e7eb; font-size: 13px; display: flex; flex-direction: column; gap: 0.25rem; }
  .rw-email-body { padding: 1rem; font-size: 13.5px; white-space: pre-wrap; line-height: 1.6; }

  /* SMS preview */
  .rw-twilio-card { background: #f3f4f6; border-radius: 12px; padding: 1rem; max-width: 340px; }
  .rw-sms-header { font-size: 12px; color: #6b7280; text-align: center; margin-bottom: 0.75rem; }
  .rw-sms-bubble { background: #34c759; color: #fff; padding: 0.6rem 0.9rem; border-radius: 18px 18px 4px 18px; font-size: 14px; line-height: 1.45; margin-left: auto; max-width: 85%; width: fit-content; white-space: pre-wrap; }

  /* Slack preview */
  .rw-slack-card { border: 1px solid #e5e7eb; border-radius: 8px; padding: 0.9rem 1rem; }
  .rw-slack-bot { display: flex; align-items: center; gap: 0.45rem; margin-bottom: 0.5rem; }
  .rw-slack-avatar { width: 24px; height: 24px; background: #b06a10; color: #fff; display: inline-flex; align-items: center; justify-content: center; border-radius: 4px; font-size: 13px; font-weight: 700; }
  .rw-slack-name { font-weight: 700; font-size: 13.5px; }
  .rw-slack-badge { font-size: 9.5px; background: #e5e7eb; color: #6b7280; padding: 1px 4px; border-radius: 3px; font-weight: 700; }
  .rw-slack-blocks { display: flex; flex-direction: column; gap: 0.5rem; }
  .rw-slack-header { font-size: 16px; font-weight: 800; }
  .rw-slack-fields { display: grid; grid-template-columns: 1fr 1fr; gap: 0.4rem; }
  .rw-slack-field, .rw-slack-text { font-size: 13px; white-space: pre-wrap; line-height: 1.45; }

  /* Discord preview */
  .rw-discord-card { background: #313338; border-radius: 8px; padding: 1rem; }
  .rw-discord-embed { border-left: 4px solid #5865f2; background: #2b2d31; border-radius: 4px; padding: 0.75rem 1rem; color: #dbdee1; }
  .rw-discord-title { font-weight: 700; margin-bottom: 0.5rem; font-size: 14px; }
  .rw-discord-desc { background: #1e1f22; border-radius: 4px; padding: 0.5rem; font-size: 12px; overflow-x: auto; margin: 0 0 0.5rem; }
  .rw-discord-footer { font-size: 11px; color: #949ba4; }

  /* ntfy preview */
  .rw-ntfy-card { border: 1px solid #e5e7eb; border-radius: 8px; padding: 0.9rem 1rem; }
  .rw-ntfy-top { display: flex; justify-content: space-between; margin-bottom: 0.5rem; }
  .rw-ntfy-topic { font-family: monospace; font-size: 12px; color: #6b7280; }
  .rw-ntfy-body { font-size: 13px; white-space: pre-wrap; margin: 0; font-family: inherit; }
  .rw-ntfy-title { font-weight: 600; font-size: 14px; margin-bottom: 0.25rem; }
  .rw-ntfy-tags { margin-top: 0.4rem; font-size: 12px; color: #6b7280; }
  .rw-multi-segment { color: #b45309; font-weight: 600; }

  /* PagerDuty preview */
  .rw-pd-card { border: 1px solid #e5e7eb; border-left: 4px solid #dc2626; border-radius: 8px; padding: 0.9rem 1rem; }
  .rw-pd-header { display: flex; justify-content: space-between; margin-bottom: 0.5rem; }
  .rw-pd-badge { font-size: 10px; font-weight: 700; letter-spacing: 0.05em; color: #6b7280; }
  .rw-pd-sev { font-size: 10px; font-weight: 800; color: #dc2626; }
  .rw-pd-summary { font-weight: 700; font-size: 14px; margin-bottom: 0.35rem; }
  .rw-pd-meta { font-size: 12px; color: #6b7280; margin-bottom: 0.5rem; }
  .rw-pd-details { background: #f9fafb; border-radius: 6px; padding: 0.5rem; font-size: 11.5px; overflow-x: auto; margin: 0; }

  /* ClickHouse preview */
  .rw-ch-card { border: 1px solid #e5e7eb; border-radius: 8px; padding: 0.9rem 1rem; }
  .rw-ch-title { font-size: 13px; margin-bottom: 0.6rem; }
  .rw-ch-table { width: 100%; border-collapse: collapse; font-size: 12px; }
  .rw-ch-table th { text-align: left; font-family: monospace; color: #6b7280; padding: 0.3rem 0.6rem 0.3rem 0; vertical-align: top; white-space: nowrap; }
  .rw-ch-table td { padding: 0.3rem 0; word-break: break-all; }

  .rw-raw-toggle { margin-top: 0.9rem; font-size: 13px; }
  .rw-raw-toggle summary { cursor: pointer; color: #6b7280; font-weight: 600; }
  .rw-raw-section { display: flex; flex-direction: column; gap: 0.4rem; margin-top: 0.6rem; }
  .rw-raw-head { display: flex; justify-content: space-between; align-items: center; font-size: 12px; font-weight: 600; color: #374151; }
  .rw-raw-code { background: #0f172a; color: #f8fafc; padding: 0.75rem; border-radius: 6px; font-size: 12px; overflow-x: auto; margin: 0; max-height: 320px; }
  .rw-cli-block { background: #0f172a; color: #f8fafc; padding: 0.75rem; border-radius: 6px; font-family: monospace; font-size: 12px; margin: 0; overflow-x: auto; }

  /* Field actions (inline row of buttons beside a label) */
  .rw-field-actions { display: flex; align-items: center; gap: 0.5rem; }

  /* Keyboard shortcut badge */
  .rw-kbd { font-family: var(--sp-font-mono, monospace); font-size: 10px; padding: 1px 4px; border: 1px solid #d1d5db; border-radius: 3px; background: #f3f4f6; color: #6b7280; margin-left: 2px; }

  /* ─── Wide edit modal ─────────────────────────────────────── */
  .rw-modal-backdrop { position: fixed; inset: 0; z-index: 100; display: flex; align-items: center; justify-content: center; }
  .rw-modal-scrim { position: fixed; inset: 0; background: rgba(0,0,0,0.45); backdrop-filter: blur(2px); }
  .rw-modal-panel {
    position: relative; z-index: 1;
    width: 90vw; max-width: 80rem;
    height: 85vh;
    background: #fff; border: 1px solid #e5e7eb; border-radius: 12px;
    display: flex; flex-direction: column; overflow: hidden;
    box-shadow: 0 20px 60px rgba(0,0,0,0.18);
  }
  .rw-modal-bar {
    display: flex; align-items: center; justify-content: space-between; gap: 0.75rem;
    padding: 0.65rem 1rem; border-bottom: 1px solid #e5e7eb; background: #f9fafb; flex-shrink: 0;
  }
  .rw-modal-body {
    display: grid; grid-template-columns: 1fr 1fr; gap: 0;
    flex: 1; min-height: 0; overflow: hidden;
  }
  @media (max-width: 700px) {
    .rw-modal-body { grid-template-columns: 1fr; grid-template-rows: 1fr 1fr; }
  }
  .rw-modal-editor {
    display: flex; flex-direction: column; padding: 0.75rem; border-right: 1px solid #e5e7eb;
    min-height: 0; overflow: hidden;
  }
  @media (max-width: 700px) {
    .rw-modal-editor { border-right: none; border-bottom: 1px solid #e5e7eb; }
  }
  .rw-modal-preview {
    display: flex; flex-direction: column; padding: 0.75rem;
    min-height: 0; overflow: hidden;
  }
  .rw-modal-pane-label { font-size: 11px; font-weight: 700; letter-spacing: 0.06em; text-transform: uppercase; color: #6b7280; margin-bottom: 0.5rem; display: block; flex-shrink: 0; }
  .rw-modal-textarea {
    flex: 1; resize: none; width: 100%; min-height: 0;
    font-family: var(--sp-font-mono, monospace); font-size: 12.5px;
    padding: 0.6rem; border: 1px solid #d1d5db; border-radius: 6px; line-height: 1.5;
  }
  .rw-modal-preview-scroll { flex: 1; overflow-y: auto; min-height: 0; }
</style>
