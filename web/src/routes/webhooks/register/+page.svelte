<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { api, unwrap } from '$lib/services';
  import { formatAPIError } from '$lib/utils';
  import ConsumerPicker from '$lib/components/ConsumerPicker.svelte';
  import { onMount } from 'svelte';
  import type { components } from '$lib/api-types';
  import { substituteParams, type Recipe, type RecipeParam } from '$lib/recipes';
  import Disclosure from '$lib/components/Disclosure.svelte';
  import TransformSettings from '$lib/components/TransformSettings.svelte';
  import type { TemplateSaveMeta } from '$lib/components/TemplateEditor.svelte';
  import SigningSecretReveal from '$lib/components/SigningSecretReveal.svelte';
  import { ALERT_EVENT_TYPES, ALERT_SETUP_HREF, ALERTS_GUIDE_URL, SYSTEM_CONSUMER, isEmail, isSystemConsumer, listAllEventTypes } from '$lib/system';

  type EventTypeItem = components["schemas"]["EventTypeItem"];

  // ?recipe=sendgrid&consumer=_sparrow opens the form pre-set, e.g. from the
  // alert setup links.
  const query = page.url.searchParams;
  // A tenant can have hundreds of consumers, so none is assumed: the consumer
  // comes from the link (?consumer=) or is picked here.
  let consumer = $state(query.get('consumer') ?? '');
  let events: string[] = $state([]);
  let url = $state('');
  let description = $state('');
  let active = $state(true);
  let allEvents: EventTypeItem[] = $state([]);
  let error = $state('');
  let submitting = $state(false);
  let eventSearch = $state('');
  let alertEmail = $state('');
  // Set once registration succeeds: the server returns the signing secret in
  // full only in that response, so the page shows it before moving on.
  let created: components["schemas"]["WebhookOut"] | null = $state(null);
  let alertError = $state('');
  let alertEmailError = $state('');
  // Whether alert emails are actually sent (a _sparrow alert-delivery
  // webhook exists); unknown until /v1/capabilities answers.
  let alertDeliveryConfigured: boolean | undefined = $state();
  // _sparrow's own webhooks never raise alerts about themselves.
  let alertsApply = $derived(!isSystemConsumer(consumer.trim()));


  // HTTP Configuration
  let showAdvanced = $state(false);
  let maxRetries = $state(3);
  let retryBackoffSeconds = $state(60);
  let captureResponseBody = $state(false);
  let followRedirects = $state(true);
  let verifySSL = $state(true);
  let requestTimeoutSeconds = $state(30);
  let expectedStatusCodes = $state('200,201,202,204');
  let userAgent = $state('Sparrow-Webhook/1.0');
  let contentType = $state('application/json');

  // HTTP Headers
  let headers: { key: string; value: string }[] = $state([]);

  // Secret Headers (encrypted server-side)
  let secretHeaders: { key: string; value: string }[] = $state([]);

  // Recipe pre-fill
  let recipesOpen = $state(false);
  let selectedRecipe: Recipe | null = $state(null);
  let recipeParams: Record<string, string> = $state({});
  let appliedRecipe = $state('');
  let recipeError = $state('');
  let transformEnabled = $state(false);
  // The receiver only accepts a transformed payload; recipes turn this on.
  let requiresTransform = $state(false);
  let transformTemplate = $state('');
  let onTransformError = $state<'fail' | 'fallback'>('fail');
  let templateMissingKey = $state<'error' | 'zero'>('error');
  let templateMeta: TemplateSaveMeta | null = $state(null);
  let recipes: Recipe[] = $state([]);

  // Validation
  let urlError = $state('');
  let consumerError = $state('');
  let eventsError = $state('');

  let filteredEvents = $derived(
    eventSearch.trim()
      ? allEvents.filter(e => e.name.toLowerCase().includes(eventSearch.toLowerCase()))
      : allEvents
  );

  // One template cannot fit several event schemas, so a hand-picked transform
  // only applies to a single event. A webhook that requires a transform gets
  // the template on every subscription instead, checked per event on submit.
  let transformLocked = $derived(events.length > 1 && !requiresTransform
    ? `Transforms apply to a single event: these events have different payloads. Register first, then add a transform to each subscription from the webhook's page${transformTemplate.trim() ? ' (the template is kept if you go back to one event)' : ''}.`
    : '');

  let headerSummary = $derived.by(() => {
    const n = headers.filter((h) => h.key.trim()).length;
    const m = secretHeaders.filter((h) => h.key.trim()).length;
    if (!n && !m) return 'None';
    return [n && `${n} header${n === 1 ? '' : 's'}`, m && `${m} secret`].filter(Boolean).join(' · ');
  });

  onMount(async () => {
    try {
      recipes = unwrap(await api.GET('/v1/recipes')).items || [];
      const wanted = recipes.find((r) => r.name === query.get('recipe'));
      if (wanted) {
        recipesOpen = true;
        pickRecipe(wanted);
      }
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to load form data');
    }
    try {
      alertDeliveryConfigured = unwrap(await api.GET('/v1/capabilities')).alert_delivery.configured;
    } catch {
      // Leave unknown: the alert copy then stays neutral.
    }
  });

  // The server lists only the events this consumer can subscribe to: _sparrow
  // gets Sparrow's own sparrow.* events, every other consumer the rest.
  // Reloaded as the consumer is typed; picks the new consumer cannot have are
  // dropped.
  $effect(() => {
    const c = consumer.trim();
    let stale = false;
    const timer = setTimeout(async () => {
      try {
        const items = await listAllEventTypes({ active_only: true, consumer: c || undefined });
        if (stale) return;
        allEvents = items;
        const names = new Set(allEvents.map((e) => e.name));
        events = events.filter((e) => names.has(e));
      } catch (e: any) {
        if (!stale) error = formatAPIError(e, 'Failed to load event types');
      }
    }, 250);
    return () => { stale = true; clearTimeout(timer); };
  });

  function addHeader() {
    headers = [...headers, { key: '', value: '' }];
  }

  function removeHeader(index: number) {
    headers = headers.filter((_, i) => i !== index);
  }

  function addSecretHeader() {
    secretHeaders = [...secretHeaders, { key: '', value: '' }];
  }

  function removeSecretHeader(index: number) {
    secretHeaders = secretHeaders.filter((_, i) => i !== index);
  }

  function pickRecipe(r: Recipe) {
    selectedRecipe = r;
    // Inputs start empty and fall back to the default (shown as a hint);
    // a select starts on its default.
    recipeParams = Object.fromEntries(
      (r.params ?? []).map((p) => [p.name, p.enum?.length && !p.must_override_default ? p.default ?? '' : ''])
    );
    recipeError = '';
  }

  function recipeParamError(p: RecipeParam, value: string): string {
    if ((p.required || p.activation_required) && !value) return `${p.prompt || p.name} is required`;
    if (p.must_override_default && p.default && value === p.default) return `${p.prompt || p.name} must be changed from ${p.default}`;
    if (p.enum?.length && value && !p.enum.includes(value)) return `${p.prompt || p.name} must be one of ${p.enum.join(', ')}`;
    return '';
  }

  function recipeParamValue(p: RecipeParam): string {
    return (recipeParams[p.name]?.trim() || p.default || '').trim();
  }

  function applyRecipe() {
    const r = selectedRecipe;
    if (!r) return;
    const params: Record<string, string> = {};
    for (const p of r.params ?? []) {
      const value = recipeParamValue(p);
      const error = recipeParamError(p, value);
      if (error) {
        recipeError = error;
        return;
      }
      params[p.name] = value;
    }
    url = substituteParams(r.webhook.url, params);
    headers = Object.entries(r.webhook.headers ?? {}).map(([key, value]) => ({ key, value: substituteParams(value, params) }));
    secretHeaders = Object.entries(r.webhook.secret_headers ?? {}).map(([key, value]) => ({ key, value: substituteParams(value, params) }));
    transformTemplate = substituteParams(r.subscription?.transform_template ?? '', params);
    transformEnabled = !!transformTemplate.trim();
    requiresTransform = !!r.webhook.requires_transform;
    templateMeta = null;
    description = `recipe ${r.name}: ${r.description}`;
    appliedRecipe = r.name;
    recipesOpen = false;
    selectedRecipe = null;
  }

  function clearRecipe() {
    appliedRecipe = '';
    transformTemplate = '';
    transformEnabled = false;
    requiresTransform = false;
    templateMeta = null;
  }

  function validateUrl(val: string): boolean {
    if (!val.trim()) { urlError = 'URL is required'; return false; }
    try { new URL(val); urlError = ''; return true; }
    catch { urlError = 'Enter a valid URL (e.g. https://api.example.com/webhook)'; return false; }
  }

  function validateConsumer(val: string): boolean {
    if (!val.trim()) { consumerError = 'Consumer is required'; return false; }
    consumerError = '';
    return true;
  }

  async function registerWebhook(e: Event) {
    e.preventDefault();
    error = '';

    const urlValid = validateUrl(url);
    const nsValid = validateConsumer(consumer);
    eventsError = events.length === 0 ? 'Select at least one event' : '';
    // Checked up front: once the webhook exists, a bad address could only
    // fail half way through.
    const wantsAlert = alertsApply && !!alertEmail.trim();
    alertEmailError = wantsAlert && !isEmail(alertEmail) ? 'Enter an email address like ops@example.com.' : '';
    if (!urlValid || !nsValid || eventsError || alertEmailError) return;
    const applyTransform = !transformLocked && (transformEnabled || requiresTransform) && !!transformTemplate.trim();
    if (requiresTransform && !applyTransform) {
      error = 'This receiver only accepts a transformed payload: write a template, or turn off "Requires a transform".';
      return;
    }

    submitting = true;
    try {
      // The same template goes to every event: check it renders against each
      // one's sample payload before anything is created.
      if (applyTransform && events.length > 1) {
        const results = await Promise.allSettled(events.map(async (name) => {
          try {
            unwrap(await api.POST('/v1/subscriptions:testTemplate', {
              body: { event_name: name, template: transformTemplate, template_missing_key: templateMissingKey },
            }));
          } catch (e: any) {
            throw new Error(`${name}: ${formatAPIError(e, 'did not render')}`);
          }
        }));
        const failures = results.flatMap((r) => (r.status === 'rejected' ? [(r.reason as Error).message] : []));
        if (failures.length) {
          error = `The template does not render for every selected event. Fix it, or pick fewer events.\n${failures.join('\n')}`;
          return;
        }
      }

      const headersMap: Record<string, string> = {};
      headers.forEach(h => {
        if (h.key.trim() && h.value.trim()) {
          headersMap[h.key.trim()] = h.value.trim();
        }
      });

      const secretHeadersMap: Record<string, string> = {};
      secretHeaders.forEach(h => {
        if (h.key.trim() && h.value.trim()) {
          secretHeadersMap[h.key.trim()] = h.value.trim();
        }
      });

      const statusCodes = expectedStatusCodes
        .split(',')
        .map(code => parseInt(code.trim()))
        .filter(code => !isNaN(code) && code >= 100 && code < 600);

      const reg = unwrap(await api.POST('/v1/consumers/{consumer}/webhooks', {
        params: { path: { consumer } },
        body: {
          events,
          url,
          description,
          active,
          headers: headersMap,
          secret_headers: Object.keys(secretHeadersMap).length > 0 ? secretHeadersMap : undefined,
          requires_transform: requiresTransform || undefined,
          transform_template: applyTransform ? transformTemplate : undefined,
          on_transform_error: applyTransform ? onTransformError : undefined,
          template_missing_key: applyTransform ? templateMissingKey : undefined,
          template_source: applyTransform ? templateMeta?.source ?? 'manual' : undefined,
          template_notes: applyTransform ? templateMeta?.notes || undefined : undefined,
          http_config: showAdvanced ? {
            max_retries: maxRetries,
            retry_backoff_seconds: retryBackoffSeconds,
            capture_response_body: captureResponseBody,
            follow_redirects: followRedirects,
            verify_ssl: verifySSL,
            request_timeout_seconds: requestTimeoutSeconds,
            expected_status_codes: statusCodes,
            user_agent: userAgent.trim() || 'Sparrow-Webhook/1.0',
            content_type: contentType.trim() || 'application/json',
          } : undefined,
        },
      }));

      created = reg;
      if (wantsAlert) {
        // The webhook exists now; a failed alert must not hide its secret.
        try {
          unwrap(await api.POST('/v1/consumers/{consumer}/alert-configs', {
            params: { path: { consumer } },
            body: {
              webhook_id: reg.webhook_id,
              email: alertEmail.trim(),
              event_types: ALERT_EVENT_TYPES,
            },
          }));
        } catch (e: any) {
          alertError = formatAPIError(e, 'Failed to set up the failure alert');
        }
      }
    } catch (e: any) {
      error = formatAPIError(e, 'Failed to register webhook');
    } finally {
      submitting = false;
    }
  }
</script>

<svelte:head>
  <title>Register Webhook | Sparrow</title>
</svelte:head>

<main class="max-w-6xl mx-auto px-4 py-6">
  <nav class="text-xs text-muted mb-3"><a href="/webhooks" class="hover:text-text">Webhooks</a> / Register</nav>
  <h1 class="text-xl font-semibold text-text mb-4">Register webhook</h1>

  {#if created}
    <section class="panel p-5 space-y-4 max-w-2xl">
      <div>
        <h2 class="text-base font-semibold text-text">Webhook registered</h2>
        <p class="text-sm text-muted mt-1">Deliveries to <span class="mono break-all">{created.url}</span> are signed with this secret. The receiver needs it to verify them.</p>
      </div>
      <SigningSecretReveal secret={created.http_config.webhook_secret ?? ''} publicKey={created.signing_public_key} />
      {#if alertError}
        <p class="text-sm" style="color:var(--color-bad)">The webhook was registered, but the failure alert was not: {alertError}. Add it from the webhook page.</p>
      {/if}
      <div class="flex items-center justify-end gap-2">
        <a href="/webhooks" class="btn btn-ghost">Back to webhooks</a>
        <button type="button" onclick={() => goto(`/webhooks/${created?.webhook_id}`)} class="btn btn-beacon">I've saved it, open the webhook</button>
      </div>
    </section>
  {:else}
  <form onsubmit={registerWebhook} class="space-y-4">
    <section class="panel p-5">
      {#if appliedRecipe}
        <div class="flex items-center justify-between gap-3">
          <p class="text-sm text-text">Using the <span class="mono">{appliedRecipe}</span> recipe: URL, headers and transform are filled in below.</p>
          <button type="button" onclick={clearRecipe} class="btn btn-ghost !px-3 !py-1 shrink-0">Clear</button>
        </div>
      {:else}
        <button type="button" class="w-full flex items-center justify-between gap-4 text-left" aria-expanded={recipesOpen} onclick={() => (recipesOpen = !recipesOpen)}>
          <span>
            <span class="eyebrow">Start from a recipe</span>
            <span class="block text-xs text-muted mt-1">Preset destinations (Slack, SendGrid, …) with their transform template.</span>
          </span>
          <span class="text-xs text-muted shrink-0">{recipesOpen ? 'Hide' : 'Browse'} <span class="inline-block transition {recipesOpen ? 'rotate-90' : ''}">›</span></span>
        </button>

        {#if recipesOpen}
          {#if selectedRecipe}
            <div class="mt-4 pt-4 border-t border-line space-y-3 max-w-xl">
              <div>
                <p class="text-sm font-medium text-text capitalize">{selectedRecipe.name}</p>
                <p class="text-xs text-muted mt-0.5">{selectedRecipe.description}</p>
              </div>
              {#if selectedRecipe.consumer}
                {@const hint = selectedRecipe.consumer}
                <div class="panel-2 px-3 py-2 text-xs text-muted" data-testid="recipe-consumer-hint">
                  <p>{hint.note}</p>
                  {#if consumer.trim() === hint.name}
                    <p class="mt-1 text-text">Registering under <span class="mono">{hint.name}</span>.</p>
                  {:else}
                    <button type="button" class="link-beacon mt-1" onclick={() => (consumer = hint.name)}>Register under <span class="mono">{hint.name}</span> instead of <span class="mono">{consumer.trim() || '…'}</span></button>
                  {/if}
                </div>
              {/if}
              {#each selectedRecipe.params ?? [] as p}
                {@const optional = !(p.required || p.activation_required)}
                <div>
                  <label for={`recipe-param-${p.name}`} class="field-label">{p.prompt || p.name}{optional ? ' (optional)' : ''}</label>
                  {#if p.enum?.length}
                    <select id={`recipe-param-${p.name}`} bind:value={recipeParams[p.name]} class="select w-full">
                      {#if optional && !p.default}<option value="">None</option>{/if}
                      {#each p.enum as v}<option value={v}>{v}</option>{/each}
                    </select>
                  {:else}
                    <input id={`recipe-param-${p.name}`} type={p.secret ? 'password' : 'text'} bind:value={recipeParams[p.name]} placeholder={p.example ?? ''} autocomplete="off" class="input w-full" />
                  {/if}
                  {#if p.help || (p.default && !p.must_override_default && !p.enum?.length) || p.docs_url}
                    <p class="text-xs text-muted mt-1">
                      {p.help ?? ''}
                      {#if p.default && !p.must_override_default && !p.enum?.length}<span class="text-faint">Default: <span class="mono">{p.default}</span>.</span>{/if}
                      {#if p.docs_url}<a href={p.docs_url} target="_blank" rel="noreferrer" class="link-beacon whitespace-nowrap">Docs ↗</a>{/if}
                    </p>
                  {/if}
                </div>
              {/each}
              {#if recipeError}<p class="text-xs" style="color:var(--color-bad)">{recipeError}</p>{/if}
              <div class="flex gap-2">
                <button type="button" onclick={applyRecipe} class="btn btn-beacon !px-3 !py-1">Use recipe</button>
                <button type="button" onclick={() => (selectedRecipe = null)} class="btn btn-ghost !px-3 !py-1">Back</button>
              </div>
            </div>
          {:else}
            <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-2 mt-4">
              {#each recipes as r}
                <button type="button" onclick={() => pickRecipe(r)} class="panel-2 p-3 text-left hover:border-line-strong transition">
                  <p class="text-sm font-medium text-text capitalize">{r.name}</p>
                  <p class="text-xs text-muted mt-0.5 line-clamp-2">{r.description}</p>
                </button>
              {/each}
            </div>
          {/if}
        {/if}
      {/if}
    </section>

    <div class="grid gap-4 lg:grid-cols-2 items-start">
      <div class="space-y-4 min-w-0">
        <section class="panel p-5 space-y-4">
          <h3 class="eyebrow">Endpoint</h3>
          <div>
            <label for="url" class="field-label">Target URL</label>
            <input id="url" type="text" bind:value={url} placeholder="https://example.com/webhook" class="input w-full" style={urlError ? 'border-color:color-mix(in srgb,var(--color-bad) 55%,transparent)' : ''} />
            {#if urlError}<p class="text-xs mt-1" style="color:var(--color-bad)">{urlError}</p>{/if}
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label for="consumer" class="field-label">Consumer</label>
              <ConsumerPicker id="consumer" bind:value={consumer} live required placeholder="Search or type a consumer" />
              {#if consumerError}<p class="text-xs mt-1" style="color:var(--color-bad)">{consumerError}</p>
              {:else if consumer.trim() === '_sparrow'}<p class="text-xs text-faint mt-1">Sparrow's internal consumer: only its own <code>sparrow.*</code> alert events can be subscribed here.</p>{/if}
            </div>
            <div>
              <label for="description" class="field-label">Description</label>
              <input id="description" type="text" bind:value={description} placeholder="Optional" class="input w-full" />
            </div>
          </div>
          <div class="flex items-center gap-3">
            <button type="button" onclick={() => (active = !active)} aria-label="Toggle active" aria-pressed={active}
              class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors {active ? 'bg-ok' : 'bg-line-strong'}">
              <span class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow transition {active ? 'translate-x-4' : 'translate-x-0'}"></span>
            </button>
            <span class="text-sm text-text">{active ? 'Active: deliveries start right away' : 'Paused: no deliveries until activated'}</span>
          </div>
          <label class="flex items-start gap-2 cursor-pointer" title="Every subscription must then carry a transform template, and Sparrow never sends this receiver its default envelope.">
            <input type="checkbox" bind:checked={requiresTransform} class="mt-0.5 accent-[color:var(--color-beacon)]" data-testid="requires-transform" />
            <span>
              <span class="text-sm text-text">Requires a transform</span>
              <span class="block text-xs text-faint">The receiver only accepts its own format (Slack, SendGrid, …). Recipes turn this on.</span>
            </span>
          </label>
        </section>

        <section class="panel p-5">
          <div class="flex items-center justify-between gap-3 mb-2">
            <label for="event-search" class="eyebrow">Events</label>
            <span class="text-xs text-muted">{events.length ? `${events.length} selected` : 'Pick at least one'}</span>
          </div>
          {#if eventsError}<p class="text-xs mb-2" style="color:var(--color-bad)">{eventsError}</p>{/if}
          <input id="event-search" type="text" placeholder="Search events…" bind:value={eventSearch} class="input w-full mb-2" />
          <div class="space-y-0.5 max-h-56 overflow-y-auto -mx-2">
            {#each filteredEvents as ev}
              <label class="flex items-center gap-2 px-2 py-1.5 hover:bg-black/5 rounded cursor-pointer">
                <input
                  type="checkbox"
                  checked={events.includes(ev.name)}
                  onchange={() => {
                    events = events.includes(ev.name) ? events.filter(e => e !== ev.name) : [...events, ev.name];
                  }}
                  class="accent-[color:var(--color-beacon)]"
                />
                <span class="text-sm text-text truncate">{ev.name}</span>
              </label>
            {/each}
          </div>
          <p class="text-xs text-faint mt-2">One subscription is created per event; refine each one later from the webhook's page.</p>
        </section>
      </div>

      <div class="space-y-4 min-w-0">
        <TransformSettings
          bind:enabled={transformEnabled}
          bind:template={transformTemplate}
          bind:onError={onTransformError}
          bind:missingKey={templateMissingKey}
          bind:meta={templateMeta}
          eventName={events[0] ?? ''}
          {consumer}
          lockedReason={transformLocked}
          required={requiresTransform}
          disabledReason={events.length === 0 ? 'Pick an event first: the editor previews the template against its sample payload.' : ''}
        />
        {#if requiresTransform && events.length > 1}
          <p class="text-xs text-muted -mt-2 px-1" data-testid="multi-event-transform-note">This template is used for all {events.length} events. It is checked against each event's sample payload when you register.</p>
        {/if}

        <section class="panel p-5 space-y-3">
          <h3 class="eyebrow">Delivery</h3>
          <Disclosure label="Headers" summary={headerSummary} open={headers.length + secretHeaders.length > 0}>
            <div>
              {#each headers as h, i}
                <div class="flex gap-2 mb-2">
                  <input type="text" placeholder="Header" bind:value={h.key} class="input flex-1 min-w-0" />
                  <input type="text" placeholder="value" bind:value={h.value} class="input flex-1 min-w-0" />
                  <button type="button" onclick={() => removeHeader(i)} class="px-2 text-faint hover:text-bad transition-colors" aria-label="Remove header">&times;</button>
                </div>
              {/each}
              <button type="button" onclick={addHeader} class="btn btn-ghost !px-3 !py-1 text-xs">+ Add header</button>
            </div>
            <div>
              <span class="field-label">Secret headers</span>
              <p class="text-xs text-faint mb-2">Encrypted at rest and never shown again, e.g. an Authorization token.</p>
              {#each secretHeaders as h, i}
                <div class="flex gap-2 mb-2">
                  <input type="text" placeholder="Header" bind:value={h.key} class="input flex-1 min-w-0" />
                  <input type="password" placeholder="value" bind:value={h.value} class="input flex-1 min-w-0" />
                  <button type="button" onclick={() => removeSecretHeader(i)} class="px-2 text-faint hover:text-bad transition-colors" aria-label="Remove secret header">&times;</button>
                </div>
              {/each}
              <button type="button" onclick={addSecretHeader} class="btn btn-ghost !px-3 !py-1 text-xs">+ Add secret header</button>
            </div>
          </Disclosure>

          {#if alertsApply}
            <Disclosure label="Health alerts" summary={alertEmail.trim() || 'Off'} open={!!alertEmailError}>
              <div>
                <label for="alert-email" class="field-label">Alert email</label>
                <input id="alert-email" type="email" bind:value={alertEmail} oninput={() => (alertEmailError = '')} placeholder="ops@example.com" class="input w-full" style={alertEmailError ? 'border-color:color-mix(in srgb,var(--color-bad) 55%,transparent)' : ''} />
                {#if alertEmailError}<p class="text-xs mt-1" style="color:var(--color-bad)">{alertEmailError}</p>{/if}
                <p class="text-xs text-faint mt-1">Emails when this webhook's health changes, a delivery fails permanently, or Sparrow pauses it.</p>
                {#if alertDeliveryConfigured === false}
                  <p class="text-xs mt-1" style="color:var(--color-warn)" data-testid="alert-delivery-missing">
                    Alert email is not set up on this server yet: the recipient is saved, but nothing is sent until an operator adds an alert-delivery webhook under <span class="mono">{SYSTEM_CONSUMER}</span>.
                    <a href={ALERTS_GUIDE_URL} target="_blank" rel="noreferrer" class="underline">Guide</a> · <a href={ALERT_SETUP_HREF} class="underline" data-sveltekit-reload>Set it up</a>
                  </p>
                {/if}
              </div>
            </Disclosure>
          {/if}

          <Disclosure label="HTTP settings" summary={showAdvanced ? `${maxRetries} retries · ${requestTimeoutSeconds}s timeout` : 'Defaults'} bind:open={showAdvanced}>
            <div class="grid grid-cols-2 gap-3">
              <div>
                <label for="maxRetries" class="field-label">Max retries</label>
                <input id="maxRetries" type="number" bind:value={maxRetries} class="input w-full" />
              </div>
              <div>
                <label for="retryBackoff" class="field-label">Retry backoff (s)</label>
                <input id="retryBackoff" type="number" bind:value={retryBackoffSeconds} class="input w-full" />
              </div>
              <div>
                <label for="timeout" class="field-label">Request timeout (s)</label>
                <input id="timeout" type="number" bind:value={requestTimeoutSeconds} class="input w-full" />
              </div>
              <div>
                <label for="statusCodes" class="field-label">Success status codes</label>
                <input id="statusCodes" type="text" bind:value={expectedStatusCodes} class="input w-full" />
              </div>
              <div>
                <label for="userAgent" class="field-label">User-Agent</label>
                <input id="userAgent" type="text" bind:value={userAgent} class="input w-full" />
              </div>
              <div>
                <label for="contentType" class="field-label">Content-Type</label>
                <input id="contentType" type="text" bind:value={contentType} class="input w-full" />
              </div>
            </div>
            <div class="flex flex-wrap items-center gap-x-4 gap-y-1">
              <label class="flex items-center gap-2 text-sm text-text cursor-pointer"><input type="checkbox" bind:checked={captureResponseBody} class="accent-[color:var(--color-beacon)]" /> Capture response body</label>
              <label class="flex items-center gap-2 text-sm text-text cursor-pointer"><input type="checkbox" bind:checked={followRedirects} class="accent-[color:var(--color-beacon)]" /> Follow redirects</label>
              <label class="flex items-center gap-2 text-sm text-text cursor-pointer"><input type="checkbox" bind:checked={verifySSL} class="accent-[color:var(--color-beacon)]" /> Verify SSL</label>
            </div>
          </Disclosure>
        </section>
      </div>
    </div>

    {#if error}
      <div class="panel p-3" style="border-color:color-mix(in srgb,var(--color-bad) 40%,transparent);background:color-mix(in srgb,var(--color-bad) 8%,var(--color-panel))">
        <p class="text-sm whitespace-pre-line" style="color:var(--color-bad)">{error}</p>
      </div>
    {/if}

    <div class="flex items-center justify-end gap-2">
      <a href="/webhooks" class="btn btn-ghost">Cancel</a>
      <button type="submit" disabled={submitting} class="btn btn-beacon">{submitting ? 'Registering…' : 'Register webhook'}</button>
    </div>
  </form>
  {/if}
</main>
