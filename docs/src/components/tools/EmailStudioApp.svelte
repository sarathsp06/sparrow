<script lang="ts">
  import { renderGoTemplate, type EventContext } from '../../lib/go-template-engine.js';

  const baseUrl = (import.meta.env?.BASE_URL || '/sparrow/').replace(/\/?$/, '/');
  const recipeWorkbenchUrl = `${baseUrl}tools/recipe-workbench/`;

  // Sample event presets
  const PRESETS = [
    {
      id: 'health_changed',
      name: 'Webhook Health Alert (Health Changed)',
      event_name: 'sparrow.webhook.health_changed',
      payload: {
        alert_recipients: [
          { email: 'devops-team@acme.com' },
          { email: 'oncall@acme.com' }
        ],
        consumer: 'billing-service',
        webhook_id: 'wh_01h8x9p3',
        url: 'https://api.acme.com/v1/webhooks/billing',
        old_health: 'healthy',
        new_health: 'degraded'
      }
    },
    {
      id: 'delivery_failed',
      name: 'Delivery Failed Alert (Permanent Failure)',
      event_name: 'sparrow.delivery_failed',
      payload: {
        alert_recipients: [
          { email: 'sre-alerts@acme.com' }
        ],
        consumer: 'shipping-service',
        webhook_id: 'wh_01h8x9p8',
        url: 'https://shipping.acme.com/events',
        delivery_id: 'del_99f23a10',
        attempt: 5,
        error_message: 'HTTP 500 Internal Server Error (connection reset)',
        error_category: 'HTTP 5xx'
      }
    },
    {
      id: 'order_created',
      name: 'Order Created (Custom E-Commerce Event)',
      event_name: 'order.created',
      payload: {
        alert_recipients: [
          { email: 'customer-care@acme.com' }
        ],
        order_id: 'ord_98124',
        customer_name: 'Jane Doe',
        total_amount: '$149.50',
        items: [
          { name: 'Developer Pro Plan', qty: 1, price: '$149.50' }
        ]
      }
    }
  ];

  // Selected state
  let selectedPresetId = $state('health_changed');
  let eventName = $state(PRESETS[0].event_name);
  let eventId = $state('evt_0195c2a17f3078b9');
  let timestamp = $state(new Date().toISOString());
  let attempt = $state(1);
  let payloadJsonStr = $state(JSON.stringify(PRESETS[0].payload, null, 2));

  // Recipe parameters
  let apiKey = $state('SG.x89123_demo_key_abcdef');
  let fromEmail = $state('alerts@sparrow-platform.dev');
  let fromName = $state('Sparrow Webhooks');

  // Go Template
  let defaultTemplate = `{
  "personalizations": [{{range $i, $r := .payload.alert_recipients}}{{if $i}},{{end}}{"to": [{"email": {{$r.email | json}}}]}{{end}}],
  "from": {"email": "{{param "from_email"}}", "name": "{{param "from_name"}}"},
  "subject": {{if eq .event_name "sparrow.webhook.health_changed"}}{{if eq .payload.new_health "healthy"}}{{printf "Sparrow: webhook for %v recovered" .payload.consumer | json}}{{else}}{{printf "Sparrow: webhook for %v is now %v" .payload.consumer .payload.new_health | json}}{{end}}{{else}}{{printf "Sparrow: delivery to %v failed permanently" .payload.consumer | json}}{{end}},
  "content": [{"type": "text/plain", "value": {{if eq .event_name "sparrow.webhook.health_changed"}}{{printf "Webhook %v (%v) health changed: %v -> %v" .payload.webhook_id .payload.url .payload.old_health .payload.new_health | json}}{{else}}{{printf "Webhook %v (%v) delivery %v failed permanently after %v attempt(s): %v (%v)" .payload.webhook_id .payload.url .payload.delivery_id .payload.attempt .payload.error_message .payload.error_category | json}}{{end}}}]
}`;

  let transformTemplate = $state(defaultTemplate);

  // UI state
  let activeView = $state<'preview' | 'json'>('preview');
  let copiedNotice = $state<string | null>(null);
  let showImportModal = $state(false);
  let importedSchemaPasted = $state('');

  // Derived computations
  let parsedPayload = $derived.by(() => {
    try {
      return JSON.parse(payloadJsonStr);
    } catch {
      return null;
    }
  });

  let eventContext = $derived<EventContext>({
    event_id: eventId,
    event_name: eventName,
    timestamp: timestamp,
    attempt: attempt,
    payload: parsedPayload || {}
  });

  let params = $derived<Record<string, string>>({
    api_key: apiKey,
    from_email: fromEmail,
    from_name: fromName
  });

  let renderOutcome = $derived.by(() => {
    if (!parsedPayload) {
      return { result: '', error: 'Invalid JSON in Payload editor' };
    }
    return renderGoTemplate(transformTemplate, eventContext, params);
  });

  let emailToRecipients = $derived.by(() => {
    if (!renderOutcome.jsonObj || !Array.isArray(renderOutcome.jsonObj.personalizations)) {
      return [];
    }
    const recipients: string[] = [];
    for (const p of renderOutcome.jsonObj.personalizations) {
      if (Array.isArray(p.to)) {
        for (const item of p.to) {
          if (item?.email) recipients.push(item.email);
        }
      }
    }
    return recipients;
  });

  let emailSubject = $derived<string>(renderOutcome.jsonObj?.subject || '(No Subject)');
  let emailContentValue = $derived.by(() => {
    if (!renderOutcome.jsonObj || !Array.isArray(renderOutcome.jsonObj.content)) {
      return '';
    }
    return renderOutcome.jsonObj.content[0]?.value || '';
  });

  let cliCommand = $derived(
    `sparrow use sendgrid \\\n  --param api_key=${apiKey} \\\n  --param from_email=${fromEmail} \\\n  --param from_name="${fromName}" \\\n  --event ${eventName}`
  );

  function selectPreset(presetId: string) {
    const p = PRESETS.find(x => x.id === presetId);
    if (!p) return;
    selectedPresetId = presetId;
    eventName = p.event_name;
    payloadJsonStr = JSON.stringify(p.payload, null, 2);
  }

  function copyToClipboard(text: string, label: string) {
    navigator.clipboard.writeText(text);
    copiedNotice = label;
    setTimeout(() => {
      copiedNotice = null;
    }, 2000);
  }

  function handleImportSchema() {
    try {
      const parsed = JSON.parse(importedSchemaPasted);
      if (parsed.payload) {
        payloadJsonStr = JSON.stringify(parsed.payload, null, 2);
        if (parsed.event_name) eventName = parsed.event_name;
      } else {
        payloadJsonStr = JSON.stringify(parsed, null, 2);
      }
      showImportModal = false;
      importedSchemaPasted = '';
      copiedNotice = 'Schema / Payload Imported!';
      setTimeout(() => { copiedNotice = null; }, 2000);
    } catch {
      alert('Invalid JSON schema provided. Please check the JSON format.');
    }
  }

  function resetToDefault() {
    transformTemplate = defaultTemplate;
    selectPreset('health_changed');
    apiKey = 'SG.x89123_demo_key_abcdef';
    fromEmail = 'alerts@sparrow-platform.dev';
    fromName = 'Sparrow Webhooks';
  }
</script>

<div class="es-root">
  <!-- Top Bar / App Header -->
  <header class="es-header">
    <div class="es-header-info">
      <div class="es-badge">SATELLITE APP</div>
      <h1>SendGrid Email Recipe Studio</h1>
      <p>Design, verify, and preview SendGrid email recipes using Sparrow event payloads and schemas.</p>
    </div>
    <div class="es-header-actions">
      <button class="es-btn es-btn-secondary" onclick={() => copyToClipboard(JSON.stringify(parsedPayload, null, 2), 'Copied Schema JSON')}>
        <svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"/></svg>
        Copy Schema
      </button>
      <button class="es-btn es-btn-secondary" onclick={() => showImportModal = true}>
        <svg width="14" height="14" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"/></svg>
        Import Schema
      </button>
      <a href={recipeWorkbenchUrl} class="es-btn es-btn-ghost">
        Recipe Workbench &rarr;
      </a>
    </div>
  </header>

  {#if copiedNotice}
    <div class="es-toast">{copiedNotice}</div>
  {/if}

  <!-- Main Grid Layout -->
  <div class="es-grid">
    <!-- Left Column: Controls, Schema, Template -->
    <div class="es-col es-col-left">

      <!-- Preset Selector -->
      <div class="es-card">
        <div class="es-card-title">
          <span class="es-num">1</span>
          <h3>Event Schema Preset & Context</h3>
        </div>
        <div class="es-preset-pills">
          {#each PRESETS as p}
            <button
              class="es-pill {selectedPresetId === p.id ? 'es-pill-active' : ''}"
              onclick={() => selectPreset(p.id)}
            >
              {p.name}
            </button>
          {/each}
        </div>

        <div class="es-form-grid">
          <div class="es-field">
            <label for="event_name">Event Name (<code>.event_name</code>)</label>
            <input id="event_name" type="text" bind:value={eventName} />
          </div>
          <div class="es-field">
            <label for="event_id">Event ID (<code>.event_id</code>)</label>
            <input id="event_id" type="text" bind:value={eventId} />
          </div>
          <div class="es-field">
            <label for="attempt">Attempt (<code>.attempt</code>)</label>
            <input id="attempt" type="number" bind:value={attempt} min="1" />
          </div>
        </div>

        <div class="es-field" style="margin-top: 0.75rem;">
          <div class="es-field-header">
            <label for="payload_json">Payload JSON (<code>.payload</code>)</label>
            <span class="es-hint">{parsedPayload ? '✓ Valid JSON' : '⚠️ Invalid JSON'}</span>
          </div>
          <textarea id="payload_json" bind:value={payloadJsonStr} rows="7" class="es-code-textarea"></textarea>
        </div>
      </div>

      <!-- Parameters Card -->
      <div class="es-card">
        <div class="es-card-title">
          <span class="es-num">2</span>
          <h3>Recipe Parameters</h3>
        </div>
        <div class="es-form-grid">
          <div class="es-field">
            <label for="param_from_email"><code>from_email</code></label>
            <input id="param_from_email" type="email" bind:value={fromEmail} placeholder="alerts@example.com" />
          </div>
          <div class="es-field">
            <label for="param_from_name"><code>from_name</code></label>
            <input id="param_from_name" type="text" bind:value={fromName} placeholder="Sparrow" />
          </div>
          <div class="es-field es-field-full">
            <label for="param_api_key"><code>api_key</code> (Secret)</label>
            <input id="param_api_key" type="password" bind:value={apiKey} />
          </div>
        </div>
      </div>

      <!-- Transform Template Card -->
      <div class="es-card">
        <div class="es-card-title">
          <span class="es-num">3</span>
          <h3>Transform Go Template</h3>
          <button class="es-btn-text" onclick={resetToDefault}>Reset Template</button>
        </div>
        <p class="es-card-sub">Sparrow renders this server-side per delivery to produce SendGrid API v3 JSON.</p>
        <textarea bind:value={transformTemplate} rows="10" class="es-code-textarea"></textarea>
      </div>

    </div>

    <!-- Right Column: Live Email Preview & Inspector -->
    <div class="es-col es-col-right">

      <!-- Preview Frame -->
      <div class="es-card es-preview-card">
        <div class="es-preview-header">
          <div class="es-tabs">
            <button
              class="es-tab {activeView === 'preview' ? 'es-tab-active' : ''}"
              onclick={() => activeView = 'preview'}
            >
              <svg width="15" height="15" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path d="M3 8l7.89 5.26a2 2 0 002.22 0L21 8M5 19h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"/></svg>
              Simulated Email Inbox Preview
            </button>
            <button
              class="es-tab {activeView === 'json' ? 'es-tab-active' : ''}"
              onclick={() => activeView = 'json'}
            >
              <svg width="15" height="15" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="2"><path d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"/></svg>
              Rendered SendGrid JSON
            </button>
          </div>

          <div class="es-status-tag {renderOutcome.error ? 'es-status-error' : 'es-status-ok'}">
            {renderOutcome.error ? 'Error' : 'Valid Email Payload'}
          </div>
        </div>

        {#if renderOutcome.error}
          <div class="es-error-banner">
            <strong>Template Error:</strong> {renderOutcome.error}
          </div>
        {/if}

        {#if activeView === 'preview'}
          <!-- Realistic Email Client Preview -->
          <div class="es-email-client">
            <div class="es-email-client-toolbar">
              <span class="es-client-dot red"></span>
              <span class="es-client-dot yellow"></span>
              <span class="es-client-dot green"></span>
              <span class="es-client-title">Mail Reader — Sparrow Satellite Preview</span>
            </div>

            <div class="es-email-meta">
              <div class="es-email-subject">{emailSubject}</div>

              <div class="es-email-meta-row">
                <span class="es-meta-label">From:</span>
                <span class="es-meta-val"><strong>{fromName}</strong> &lt;{fromEmail}&gt;</span>
              </div>

              <div class="es-email-meta-row">
                <span class="es-meta-label">To:</span>
                <div class="es-recipient-badges">
                  {#if emailToRecipients.length > 0}
                    {#each emailToRecipients as email, i}
                      {#if i > 0} {' '} {/if}
                      <span class="es-badge-email">{email}</span>
                    {/each}
                  {:else}
                    <span class="es-badge-email empty">(No recipients found in personalizations)</span>
                  {/if}
                </div>
              </div>

              <div class="es-email-meta-row">
                <span class="es-meta-label">Date:</span>
                <span class="es-meta-val-muted">{new Date().toLocaleString()}</span>
              </div>
            </div>

            <div class="es-email-body">
              {#if emailContentValue}
                <div class="es-email-text">{emailContentValue}</div>
              {:else}
                <div class="es-email-placeholder">Email body content is empty or unparseable.</div>
              {/if}
            </div>
          </div>
        {:else}
          <!-- Raw JSON View -->
          <div class="es-json-viewer">
            <pre><code>{renderOutcome.result}</code></pre>
          </div>
        {/if}
      </div>

      <!-- CLI Quick Export Card -->
      <div class="es-card">
        <div class="es-card-title">
          <h3>Apply with Sparrow CLI</h3>
          <button class="es-btn es-btn-secondary es-btn-sm" onclick={() => copyToClipboard(cliCommand, 'Copied CLI Command')}>
            Copy Command
          </button>
        </div>
        <p class="es-card-sub">Run this command to register this SendGrid recipe on your Sparrow server:</p>
        <pre class="es-cli-block"><code>{cliCommand}</code></pre>
      </div>

    </div>
  </div>
</div>

<!-- Import Modal -->
{#if showImportModal}
  <div class="es-modal-backdrop" onclick={() => showImportModal = false} role="presentation">
    <div class="es-modal" onclick={(e) => e.stopPropagation()} role="dialog">
      <h3>Import Event Schema / Payload</h3>
      <p>Paste event JSON from your Sparrow subscription page or event log:</p>
      <textarea bind:value={importedSchemaPasted} rows="8" placeholder="Paste event or schema JSON here..." class="es-code-textarea"></textarea>
      <div class="es-modal-actions">
        <button class="es-btn es-btn-ghost" onclick={() => showImportModal = false}>Cancel</button>
        <button class="es-btn es-btn-primary" onclick={handleImportSchema}>Import into Studio</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .es-root {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
    font-family: var(--sp-font-body, system-ui, sans-serif);
    color: var(--sp-on-surface, #1f2937);
    margin: 1.5rem 0 3rem;
  }

  .es-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding: 1.5rem;
    background: var(--sp-surface-container-lowest, #ffffff);
    border: 1px solid var(--sp-outline-variant, #e5e7eb);
    border-radius: 12px;
    gap: 1rem;
    flex-wrap: wrap;
  }

  .es-badge {
    display: inline-block;
    font-family: var(--sp-font-mono, monospace);
    font-size: 10px;
    font-weight: 700;
    letter-spacing: 0.05em;
    padding: 2px 8px;
    border-radius: 999px;
    background: #fef3c7;
    color: #92400e;
    margin-bottom: 0.5rem;
  }

  .es-header h1 {
    font-size: 1.5rem;
    font-weight: 700;
    margin: 0 0 0.25rem;
    letter-spacing: -0.02em;
  }

  .es-header p {
    margin: 0;
    font-size: 0.9rem;
    color: var(--sp-on-surface-variant, #4b5563);
  }

  .es-header-actions {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    flex-wrap: wrap;
  }

  .es-toast {
    position: fixed;
    bottom: 24px;
    right: 24px;
    background: #10b981;
    color: #ffffff;
    padding: 10px 18px;
    border-radius: 8px;
    font-weight: 600;
    font-size: 13px;
    box-shadow: 0 10px 25px rgba(0,0,0,0.15);
    z-index: 1000;
  }

  /* Grid */
  .es-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1.5rem;
  }

  @media (max-width: 1024px) {
    .es-grid {
      grid-template-columns: 1fr;
    }
  }

  .es-col {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  /* Cards */
  .es-card {
    background: var(--sp-surface-container-lowest, #ffffff);
    border: 1px solid var(--sp-outline-variant, #e5e7eb);
    border-radius: 12px;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .es-card-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .es-card-title h3 {
    font-size: 1rem;
    font-weight: 700;
    margin: 0;
    flex: 1;
  }

  .es-card-sub {
    font-size: 0.85rem;
    color: var(--sp-on-surface-variant, #6b7280);
    margin: 0;
  }

  .es-num {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: #f3f4f6;
    color: #374151;
    font-size: 11px;
    font-weight: 700;
  }

  /* Buttons */
  .es-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.45rem 0.9rem;
    font-size: 13px;
    font-weight: 600;
    border-radius: 6px;
    cursor: pointer;
    border: none;
    text-decoration: none;
    transition: all 0.15s ease;
  }

  .es-btn-primary {
    background: #d97706;
    color: #ffffff;
  }
  .es-btn-primary:hover { background: #b45309; }

  .es-btn-secondary {
    background: #f3f4f6;
    color: #374151;
    border: 1px solid #d1d5db;
  }
  .es-btn-secondary:hover { background: #e5e7eb; }

  .es-btn-ghost {
    background: transparent;
    color: #374151;
    border: 1px solid #d1d5db;
  }
  .es-btn-ghost:hover { background: #f3f4f6; }

  .es-btn-sm { padding: 0.3rem 0.6rem; font-size: 12px; }

  .es-btn-text {
    background: none;
    border: none;
    color: #d97706;
    font-size: 12px;
    font-weight: 600;
    cursor: pointer;
  }

  /* Presets */
  .es-preset-pills {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }

  .es-pill {
    padding: 0.35rem 0.75rem;
    font-size: 12px;
    font-weight: 500;
    border-radius: 999px;
    border: 1px solid #e5e7eb;
    background: #f9fafb;
    color: #4b5563;
    cursor: pointer;
  }
  .es-pill:hover { background: #f3f4f6; }
  .es-pill-active {
    background: #fef3c7;
    border-color: #f59e0b;
    color: #92400e;
    font-weight: 600;
  }

  /* Forms */
  .es-form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
  }

  .es-field {
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .es-field-full { grid-column: span 2; }

  .es-field label {
    font-size: 12px;
    font-weight: 600;
    color: #374151;
  }

  .es-field input {
    padding: 0.45rem 0.65rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 13px;
    background: #ffffff;
  }

  .es-field-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .es-hint { font-size: 11px; font-weight: 600; color: #10b981; }

  .es-code-textarea {
    width: 100%;
    font-family: var(--sp-font-mono, monospace);
    font-size: 12px;
    line-height: 1.5;
    padding: 0.6rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    background: #f9fafb;
    color: #111827;
    resize: vertical;
    box-sizing: border-box;
  }

  /* Email Preview Frame */
  .es-preview-card {
    padding: 0;
    overflow: hidden;
  }

  .es-preview-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 1rem;
    background: #f8fafc;
    border-bottom: 1px solid #e2e8f0;
  }

  .es-tabs {
    display: flex;
    gap: 0.5rem;
  }

  .es-tab {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.4rem 0.75rem;
    font-size: 12px;
    font-weight: 600;
    border-radius: 6px;
    border: none;
    background: transparent;
    color: #64748b;
    cursor: pointer;
  }

  .es-tab-active {
    background: #ffffff;
    color: #0f172a;
    box-shadow: 0 1px 3px rgba(0,0,0,0.1);
  }

  .es-status-tag {
    font-size: 11px;
    font-weight: 700;
    padding: 3px 8px;
    border-radius: 999px;
  }
  .es-status-ok { background: #d1fae5; color: #065f46; }
  .es-status-error { background: #fee2e2; color: #991b1b; }

  .es-error-banner {
    padding: 0.75rem 1rem;
    background: #fef2f2;
    border-bottom: 1px solid #fecaca;
    color: #991b1b;
    font-size: 13px;
  }

  /* Email Client UI */
  .es-email-client {
    display: flex;
    flex-direction: column;
    background: #ffffff;
  }

  .es-email-client-toolbar {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 14px;
    background: #f1f5f9;
    border-bottom: 1px solid #e2e8f0;
  }

  .es-client-dot {
    width: 10px; height: 10px; border-radius: 50%;
  }
  .es-client-dot.red { background: #ef4444; }
  .es-client-dot.yellow { background: #f59e0b; }
  .es-client-dot.green { background: #10b981; }

  .es-client-title {
    margin-left: 8px;
    font-size: 11px;
    font-weight: 600;
    color: #64748b;
  }

  .es-email-meta {
    padding: 1.25rem 1.5rem;
    border-bottom: 1px solid #f1f5f9;
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
  }

  .es-email-subject {
    font-size: 1.2rem;
    font-weight: 700;
    color: #0f172a;
    margin-bottom: 0.25rem;
  }

  .es-email-meta-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 13px;
  }

  .es-meta-label {
    min-width: 45px;
    font-weight: 600;
    color: #64748b;
  }

  .es-meta-val { color: #334155; }
  .es-meta-val-muted { color: #94a3b8; font-size: 12px; }

  .es-recipient-badges {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
  }

  .es-badge-email {
    font-family: var(--sp-font-mono, monospace);
    font-size: 11px;
    background: #eff6ff;
    color: #1e40af;
    padding: 2px 8px;
    border-radius: 4px;
    border: 1px solid #bfdbfe;
    display: inline-block;
  }
  .es-badge-email.empty {
    background: #fef2f2;
    color: #991b1b;
    border-color: #fecaca;
  }

  .es-email-body {
    padding: 1.5rem;
    min-height: 180px;
    background: #fafafa;
  }

  .es-email-text {
    font-size: 14px;
    line-height: 1.6;
    color: #1e293b;
    white-space: pre-wrap;
    word-break: break-word;
    background: #ffffff;
    padding: 1.25rem;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    box-shadow: 0 1px 2px rgba(0,0,0,0.03);
  }

  .es-email-placeholder {
    color: #94a3b8;
    font-style: italic;
    font-size: 13px;
  }

  /* JSON Viewer */
  .es-json-viewer {
    padding: 1rem;
    background: #0f172a;
    color: #38bdf8;
    overflow-x: auto;
  }

  .es-json-viewer pre {
    margin: 0;
    font-family: var(--sp-font-mono, monospace);
    font-size: 12px;
    line-height: 1.5;
  }

  .es-cli-block {
    background: #0f172a;
    color: #f8fafc;
    padding: 0.85rem;
    border-radius: 6px;
    font-family: var(--sp-font-mono, monospace);
    font-size: 12px;
    margin: 0;
    overflow-x: auto;
  }

  /* Modal */
  .es-modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0,0,0,0.5);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 2000;
  }

  .es-modal {
    background: #ffffff;
    border-radius: 12px;
    padding: 1.5rem;
    width: 90%;
    max-width: 520px;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .es-modal h3 { margin: 0; font-size: 1.1rem; }
  .es-modal p { margin: 0; font-size: 0.85rem; color: #6b7280; }

  .es-modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 0.5rem;
  }
</style>
