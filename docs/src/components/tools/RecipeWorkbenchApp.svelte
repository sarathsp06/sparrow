<script lang="ts">
  import { renderGoTemplate, type EventContext } from '../../lib/go-template-engine.js';
  import { RECIPES, type RecipeDef } from '../../lib/recipe-templates.js';

  // Active state
  let selectedRecipeId = $state('sendgrid');
  let activeRecipe = $derived(RECIPES.find(r => r.id === selectedRecipeId)!);

  let paramValues = $state<Record<string, string>>({
    api_key: 'SG.demo_key_123',
    from_email: 'alerts@example.com',
    from_name: 'Sparrow Webhooks',
    webhook_url: 'https://hooks.slack.com/services/T00/B00/X123',
    topic_url: 'https://ntfy.sh/sparrow_alerts',
    routing_key: 'pd-key-prod-9988',
    base_url: 'http://clickhouse.internal:8123',
    table: 'webhook_events',
    user: 'default',
    password: 'secret123',
    account_sid: 'AC1234567890',
    basic_auth: 'QUMxMjM0NTY3ODkwOnNlY3JldA==',
    from_number: '+15551230000',
    to_number: '+15559876543'
  });

  let eventName = $state('sparrow.webhook.health_changed');
  let eventId = $state('evt_0195c2a17f30');
  let timestamp = $state(new Date().toISOString());
  let attempt = $state(1);
  let payloadStr = $state(JSON.stringify(RECIPES[0].sampleEvent.payload, null, 2));

  let customTransformTemplate = $state(RECIPES[0].transform_template);
  let copiedToast = $state<string | null>(null);

  function switchRecipe(recipeId: string) {
    selectedRecipeId = recipeId;
    const r = RECIPES.find(x => x.id === recipeId)!;
    eventName = r.sampleEvent.event_name;
    payloadStr = JSON.stringify(r.sampleEvent.payload, null, 2);
    customTransformTemplate = r.transform_template;
  }

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
    payload: parsedPayload || {}
  });

  let renderResult = $derived.by(() => {
    if (!parsedPayload) {
      return { result: '', error: 'Invalid JSON in event payload' };
    }
    return renderGoTemplate(customTransformTemplate, eventContext, paramValues);
  });

  let cliCommand = $derived.by(() => {
    let cmd = `sparrow use ${activeRecipe.id}`;
    for (const p of activeRecipe.params) {
      const val = paramValues[p.name] || p.default || '';
      cmd += ` \\\n  --param ${p.name}="${val}"`;
    }
    cmd += ` \\\n  --event ${eventName}`;
    return cmd;
  });

  function copyText(txt: string, msg: string) {
    navigator.clipboard.writeText(txt);
    copiedToast = msg;
    setTimeout(() => { copiedToast = null; }, 2000);
  }
</script>

<div class="rw-root">
  <header class="rw-header">
    <div class="rw-header-title">
      <span class="rw-badge">SATELLITE WORKBENCH</span>
      <h1>Satellite Recipe Workbench</h1>
      <p>Interactive playground to inspect, test, and render all 7 Sparrow satellite recipes.</p>
    </div>
    <div class="rw-header-links">
      {#if selectedRecipeId === 'sendgrid'}
        <a href="/sparrow/tools/email-studio/" class="rw-btn rw-btn-primary">
          Open Email Studio &rarr;
        </a>
      {/if}
    </div>
  </header>

  {#if copiedToast}
    <div class="rw-toast">{copiedToast}</div>
  {/if}

  <!-- Recipe Selector Tabs -->
  <nav class="rw-recipe-tabs">
    {#each RECIPES as r}
      <button
        class="rw-tab {selectedRecipeId === r.id ? 'rw-tab-active' : ''}"
        onclick={() => switchRecipe(r.id)}
      >
        <span class="rw-tab-name">{r.name}</span>
      </button>
    {/each}
  </nav>

  <div class="rw-tagline-bar">
    <strong>{activeRecipe.name}:</strong> {activeRecipe.tagline}
  </div>

  <div class="rw-grid">
    <!-- Left Col: Controls & Schema -->
    <div class="rw-col">

      <!-- Recipe Parameters -->
      <div class="rw-card">
        <h3>1. Recipe Parameters</h3>
        <div class="rw-form-list">
          {#each activeRecipe.params as p}
            <div class="rw-field">
              <label for={"p_" + p.name}>{p.prompt} (<code>{p.name}</code>)</label>
              <input
                id={"p_" + p.name}
                type={p.secret ? 'password' : 'text'}
                bind:value={paramValues[p.name]}
                placeholder={p.default || ''}
              />
            </div>
          {/each}
        </div>
      </div>

      <!-- Event Context -->
      <div class="rw-card">
        <h3>2. Event Envelope Context</h3>
        <div class="rw-form-grid">
          <div class="rw-field">
            <label for="rw_event_name"><code>.event_name</code></label>
            <input id="rw_event_name" type="text" bind:value={eventName} />
          </div>
          <div class="rw-field">
            <label for="rw_event_id"><code>.event_id</code></label>
            <input id="rw_event_id" type="text" bind:value={eventId} />
          </div>
          <div class="rw-field">
            <label for="rw_attempt"><code>.attempt</code></label>
            <input id="rw_attempt" type="number" bind:value={attempt} min="1" />
          </div>
          <div class="rw-field">
            <label for="rw_timestamp"><code>.timestamp</code></label>
            <input id="rw_timestamp" type="text" bind:value={timestamp} />
          </div>
        </div>

        <div class="rw-field" style="margin-top: 0.75rem;">
          <div class="rw-field-header">
            <label for="rw_payload">Payload JSON (<code>.payload</code>)</label>
            <span class="rw-status-text">{parsedPayload ? '✓ Valid JSON' : '⚠️ Invalid JSON'}</span>
          </div>
          <textarea id="rw_payload" bind:value={payloadStr} rows="6" class="rw-code-textarea"></textarea>
        </div>
      </div>

      <!-- Template Editor -->
      <div class="rw-card">
        <div class="rw-card-header">
          <h3>3. Go Transform Template</h3>
          <button class="rw-btn-text" onclick={() => customTransformTemplate = activeRecipe.transform_template}>
            Reset Template
          </button>
        </div>
        <textarea bind:value={customTransformTemplate} rows="8" class="rw-code-textarea"></textarea>
      </div>

    </div>

    <!-- Right Col: Rendered Visual Output -->
    <div class="rw-col">

      <div class="rw-card rw-preview-container">
        <div class="rw-card-header">
          <h3>Simulated Target Preview ({activeRecipe.name})</h3>
          <span class="rw-badge-sm">{activeRecipe.id}</span>
        </div>

        {#if renderResult.error}
          <div class="rw-error-box">{renderResult.error}</div>
        {/if}

        <!-- Visual Destination Renderers -->
        <div class="rw-visual-box">
          {#if selectedRecipeId === 'sendgrid'}
            <!-- SendGrid Email Card -->
            <div class="rw-email-card">
              <div class="rw-email-header">
                <div><strong>Subject:</strong> {renderResult.jsonObj?.subject || '(No subject)'}</div>
                <div><strong>From:</strong> {paramValues.from_name} &lt;{paramValues.from_email}&gt;</div>
                <div><strong>To:</strong> {renderResult.jsonObj?.personalizations?.[0]?.to?.[0]?.email || 'N/A'}</div>
              </div>
              <div class="rw-email-body">
                {renderResult.jsonObj?.content?.[0]?.value || renderResult.result}
              </div>
            </div>

          {:else if selectedRecipeId === 'slack'}
            <!-- Slack Block Kit Card -->
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
                          <div class="rw-slack-field">{f.text}</div>
                        {/each}
                      </div>
                    {:else if block.type === 'section' && block.text}
                      <div class="rw-slack-code">{block.text?.text}</div>
                    {/if}
                  {/each}
                </div>
              {:else}
                <pre><code>{renderResult.result}</code></pre>
              {/if}
            </div>

          {:else if selectedRecipeId === 'discord'}
            <!-- Discord Embed Card -->
            <div class="rw-discord-card">
              <div class="rw-discord-embed">
                <div class="rw-discord-title">{renderResult.jsonObj?.embeds?.[0]?.title || eventName}</div>
                <pre class="rw-discord-desc"><code>{renderResult.jsonObj?.embeds?.[0]?.description}</code></pre>
                <div class="rw-discord-footer">{renderResult.jsonObj?.embeds?.[0]?.timestamp}</div>
              </div>
            </div>

          {:else if selectedRecipeId === 'ntfy'}
            <!-- ntfy Card -->
            <div class="rw-ntfy-card">
              <div class="rw-ntfy-top">
                <span class="rw-ntfy-topic">ntfy.sh / {paramValues.topic_url.split('/').pop()}</span>
                <span class="rw-ntfy-tag">PRIORITY HIGH</span>
              </div>
              <pre class="rw-ntfy-body">{renderResult.result}</pre>
            </div>

          {:else if selectedRecipeId === 'pagerduty'}
            <!-- PagerDuty Incident Card -->
            <div class="rw-pd-card">
              <div class="rw-pd-header">
                <span class="rw-pd-badge">PAGERDUTY INCIDENT</span>
                <span class="rw-pd-sev">ERROR</span>
              </div>
              <div class="rw-pd-summary">{renderResult.jsonObj?.payload?.summary}</div>
              <div class="rw-pd-meta">Dedup Key: <code>{renderResult.jsonObj?.dedup_key}</code></div>
              <pre class="rw-pd-details"><code>{JSON.stringify(renderResult.jsonObj?.payload?.custom_details, null, 2)}</code></pre>
            </div>

          {:else if selectedRecipeId === 'clickhouse'}
            <!-- ClickHouse Table Row Card -->
            <div class="rw-ch-card">
              <div class="rw-ch-title">ClickHouse Row Insertion (JSONEachRow)</div>
              <div class="rw-ch-row">
                {#if renderResult.jsonObj}
                  <table>
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
            </div>

          {:else if selectedRecipeId === 'twilio'}
            <!-- Twilio SMS Bubble -->
            <div class="rw-twilio-card">
              <div class="rw-sms-header">To: {paramValues.to_number}</div>
              <div class="rw-sms-bubble">{renderResult.result}</div>
            </div>
          {/if}
        </div>

        <div class="rw-raw-toggle">
          <details>
            <summary>Raw Rendered Output (JSON / Text)</summary>
            <pre class="rw-raw-code"><code>{renderResult.result}</code></pre>
          </details>
        </div>
      </div>

      <!-- CLI Command Output -->
      <div class="rw-card">
        <div class="rw-card-header">
          <h3>Apply Command</h3>
          <button class="rw-btn rw-btn-secondary rw-btn-sm" onclick={() => copyText(cliCommand, 'Copied CLI Command')}>
            Copy CLI Command
          </button>
        </div>
        <pre class="rw-cli-block"><code>{cliCommand}</code></pre>
      </div>

    </div>
  </div>
</div>

<style>
  .rw-root {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
    font-family: var(--sp-font-body, system-ui, sans-serif);
    color: var(--sp-on-surface, #1f2937);
    margin: 1.5rem 0 3rem;
  }

  .rw-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1.25rem;
    background: #ffffff;
    border: 1px solid #e5e7eb;
    border-radius: 12px;
  }

  .rw-badge {
    display: inline-block;
    font-family: monospace;
    font-size: 10px;
    font-weight: 700;
    padding: 2px 8px;
    border-radius: 999px;
    background: #e0f2fe;
    color: #0369a1;
    margin-bottom: 0.25rem;
  }

  .rw-header h1 { margin: 0 0 0.25rem; font-size: 1.4rem; font-weight: 700; }
  .rw-header p { margin: 0; font-size: 0.85rem; color: #6b7280; }

  .rw-toast {
    position: fixed;
    bottom: 24px;
    right: 24px;
    background: #10b981;
    color: #ffffff;
    padding: 10px 18px;
    border-radius: 8px;
    font-weight: 600;
    font-size: 13px;
    z-index: 1000;
  }

  /* Recipe Tabs */
  .rw-recipe-tabs {
    display: flex;
    gap: 0.4rem;
    overflow-x: auto;
    padding-bottom: 0.25rem;
  }

  .rw-tab {
    padding: 0.5rem 1rem;
    border: 1px solid #e5e7eb;
    background: #ffffff;
    border-radius: 8px;
    font-size: 13px;
    font-weight: 600;
    color: #4b5563;
    cursor: pointer;
    white-space: nowrap;
  }
  .rw-tab:hover { background: #f9fafb; }
  .rw-tab-active {
    background: #0f172a;
    color: #ffffff;
    border-color: #0f172a;
  }

  .rw-tagline-bar {
    padding: 0.6rem 1rem;
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    font-size: 13px;
    color: #334155;
  }

  /* Grid */
  .rw-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 1.25rem;
  }

  @media (max-width: 1024px) {
    .rw-grid { grid-template-columns: 1fr; }
  }

  .rw-col {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .rw-card {
    background: #ffffff;
    border: 1px solid #e5e7eb;
    border-radius: 12px;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .rw-card h3 { margin: 0; font-size: 0.95rem; font-weight: 700; flex: 1; }

  .rw-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .rw-badge-sm {
    font-family: monospace;
    font-size: 11px;
    background: #f1f5f9;
    padding: 2px 6px;
    border-radius: 4px;
  }

  /* Forms */
  .rw-form-list, .rw-form-grid {
    display: grid;
    gap: 0.6rem;
  }
  .rw-form-grid { grid-template-columns: 1fr 1fr; }

  .rw-field { display: flex; flex-direction: column; gap: 0.25rem; }
  .rw-field label { font-size: 12px; font-weight: 600; color: #374151; }
  .rw-field input {
    padding: 0.45rem 0.65rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 13px;
  }

  .rw-field-header { display: flex; justify-content: space-between; align-items: center; }
  .rw-status-text { font-size: 11px; font-weight: 600; color: #10b981; }

  .rw-code-textarea {
    width: 100%;
    font-family: monospace;
    font-size: 12px;
    padding: 0.6rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    background: #f8fafc;
    box-sizing: border-box;
  }

  .rw-btn {
    padding: 0.45rem 0.9rem;
    font-size: 13px;
    font-weight: 600;
    border-radius: 6px;
    cursor: pointer;
    border: none;
    text-decoration: none;
  }
  .rw-btn-primary { background: #d97706; color: #ffffff; }
  .rw-btn-secondary { background: #f3f4f6; color: #374151; border: 1px solid #d1d5db; }
  .rw-btn-sm { padding: 0.25rem 0.5rem; font-size: 11px; }
  .rw-btn-text { background: none; border: none; color: #d97706; font-size: 12px; cursor: pointer; }

  /* Visual Preview Box */
  .rw-visual-box {
    background: #f8fafc;
    border: 1px solid #e2e8f0;
    border-radius: 8px;
    padding: 1.25rem;
    min-height: 200px;
  }

  .rw-error-box {
    padding: 0.75rem;
    background: #fef2f2;
    border: 1px solid #fecaca;
    color: #991b1b;
    font-size: 12px;
    border-radius: 6px;
  }

  /* Target Card Renderers */
  .rw-email-card {
    background: #ffffff;
    border: 1px solid #cbd5e1;
    border-radius: 8px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .rw-email-header { font-size: 12px; display: flex; flex-direction: column; gap: 0.2rem; }
  .rw-email-body { padding: 0.75rem; background: #f1f5f9; border-radius: 6px; font-size: 13px; white-space: pre-wrap; }

  /* Slack */
  .rw-slack-card { background: #1a1d21; color: #d1d2d3; padding: 1rem; border-radius: 8px; font-family: sans-serif; }
  .rw-slack-bot { display: flex; align-items: center; gap: 6px; margin-bottom: 0.5rem; }
  .rw-slack-avatar { width: 20px; height: 20px; background: #611f69; color: #fff; border-radius: 4px; display: flex; align-items: center; justify-content: center; font-size: 11px; font-weight: 700; }
  .rw-slack-name { font-weight: 700; font-size: 13px; color: #fff; }
  .rw-slack-badge { font-size: 9px; background: #2c2d30; padding: 1px 4px; border-radius: 3px; }
  .rw-slack-header { font-size: 15px; font-weight: 700; color: #fff; margin-bottom: 0.4rem; }
  .rw-slack-fields { display: grid; grid-template-columns: 1fr 1fr; gap: 0.5rem; font-size: 12px; margin-bottom: 0.5rem; }
  .rw-slack-code { background: #222529; padding: 0.5rem; border-radius: 4px; font-family: monospace; font-size: 11px; }

  /* Discord */
  .rw-discord-card { background: #313338; color: #dbdee1; padding: 1rem; border-radius: 8px; }
  .rw-discord-embed { border-left: 4px solid #5865f2; background: #2b2d31; padding: 0.75rem 1rem; border-radius: 4px; }
  .rw-discord-title { font-weight: 700; color: #fff; margin-bottom: 0.4rem; }
  .rw-discord-desc { margin: 0; font-family: monospace; font-size: 11px; background: #1e1f22; padding: 0.5rem; border-radius: 4px; }
  .rw-discord-footer { font-size: 10px; color: #949ba4; margin-top: 0.4rem; }

  /* ntfy */
  .rw-ntfy-card { background: #ffffff; border: 1px solid #e2e8f0; border-radius: 8px; padding: 1rem; }
  .rw-ntfy-top { display: flex; justify-content: space-between; font-size: 11px; margin-bottom: 0.5rem; }
  .rw-ntfy-topic { font-weight: 700; color: #0284c7; }
  .rw-ntfy-tag { background: #fee2e2; color: #991b1b; padding: 1px 6px; border-radius: 4px; font-weight: 700; }
  .rw-ntfy-body { margin: 0; font-family: monospace; font-size: 12px; white-space: pre-wrap; }

  /* PagerDuty */
  .rw-pd-card { background: #002d24; color: #e6fffa; padding: 1rem; border-radius: 8px; }
  .rw-pd-header { display: flex; justify-content: space-between; font-size: 11px; font-weight: 700; margin-bottom: 0.5rem; }
  .rw-pd-sev { background: #e53e3e; color: #fff; padding: 1px 6px; border-radius: 4px; }
  .rw-pd-summary { font-size: 14px; font-weight: 700; margin-bottom: 0.4rem; }
  .rw-pd-meta { font-size: 11px; opacity: 0.8; margin-bottom: 0.5rem; }
  .rw-pd-details { margin: 0; font-family: monospace; font-size: 11px; background: #001f19; padding: 0.5rem; border-radius: 4px; }

  /* ClickHouse */
  .rw-ch-card { background: #ffffff; border: 1px solid #cbd5e1; border-radius: 8px; padding: 1rem; }
  .rw-ch-title { font-weight: 700; font-size: 13px; margin-bottom: 0.5rem; }
  .rw-ch-row table { width: 100%; border-collapse: collapse; font-size: 12px; }
  .rw-ch-row th, .rw-ch-row td { padding: 4px 8px; border: 1px solid #e2e8f0; text-align: left; }

  /* Twilio */
  .rw-twilio-card { display: flex; flex-direction: column; gap: 0.4rem; }
  .rw-sms-header { font-size: 11px; font-weight: 600; color: #64748b; }
  .rw-sms-bubble { background: #2563eb; color: #ffffff; padding: 0.75rem 1rem; border-radius: 16px 16px 2px 16px; font-size: 13px; max-width: 85%; align-self: flex-end; }

  .rw-raw-toggle summary { font-size: 12px; font-weight: 600; color: #64748b; cursor: pointer; }
  .rw-raw-code { background: #0f172a; color: #f8fafc; padding: 0.75rem; border-radius: 6px; font-family: monospace; font-size: 11px; overflow-x: auto; margin-top: 0.4rem; }

  .rw-cli-block { background: #0f172a; color: #f8fafc; padding: 0.75rem; border-radius: 6px; font-family: monospace; font-size: 12px; margin: 0; overflow-x: auto; }
</style>
