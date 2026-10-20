<script lang="ts">
  // Sends test events to one webhook (POST :sendTestEvent): the sample, or a
  // payload the user edits. Shared by the admin webhook page and the portal;
  // the API client routes the same /v1 call through /portal/api/ there.
  import { JSONEditor, Mode, type Content } from "svelte-jsoneditor";
  import { api, unwrap } from "$lib/services";
  import { formatAPIError } from "$lib/utils";
  import { VERIFY_SIGNATURES_GUIDE_URL } from "$lib/system";
  import type { components } from "$lib/api-types";

  type TestResult = components["schemas"]["SendTestEventOutputBody"];

  interface Props {
    consumer: string;
    webhookId: string;
    /** The webhook's subscriptions, one test button each. */
    subscriptions: { subscription_id: string; event_name: string }[];
  }

  let { consumer, webhookId, subscriptions }: Props = $props();

  let tests = $state<Record<string, { busy: boolean; result?: TestResult; error?: string }>>({});
  // Per subscription: the payload being edited, when its editor is open.
  let payloads = $state<Record<string, { content: Content; loading: boolean }>>({});
  // Per catch-all (*) subscription: the event to send.
  let eventNames = $state<Record<string, string>>({});

  // The event a row sends: its own, or the one typed for a catch-all.
  const eventFor = (s: { subscription_id: string; event_name: string }) =>
    s.event_name === "*" ? (eventNames[s.subscription_id] ?? "").trim() : s.event_name;

  // Opens the payload editor with the event type's sample, or closes it.
  async function togglePayload(subscriptionId: string, eventName: string) {
    if (payloads[subscriptionId]) {
      delete payloads[subscriptionId];
      return;
    }
    payloads[subscriptionId] = { content: { json: {} }, loading: true };
    if (!eventName) {
      payloads[subscriptionId] = { content: { json: {} }, loading: false };
      return;
    }
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

  async function sendTest(subscriptionId: string, eventName: string) {
    if (!eventName) {
      tests[subscriptionId] = { busy: false, error: "Enter the event to send." };
      return;
    }
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
          body: {
            subscription_id: subscriptionId,
            event_name: eventName,
            ...(edited.payload ? { payload: edited.payload } : {}),
          },
        }),
      );
      tests[subscriptionId] = { busy: false, result };
    } catch (e) {
      tests[subscriptionId] = { busy: false, error: formatAPIError(e, "Failed to send the test event") };
    }
  }

  const ok = (r: TestResult) => r.status_code >= 200 && r.status_code < 300;
</script>

<section class="space-y-3" data-testid="test-events">
  <div class="space-y-1">
    <h3 class="eyebrow">Send a test event</h3>
    <p class="text-sm text-muted">
      Sends one signed delivery of the event type's sample, or of the payload you edit, rendered like a real delivery
      and marked <code class="mono text-xs">X-Sparrow-Test: true</code>. It is not recorded and never affects the
      endpoint's health. To verify signatures in your receiver, see
      <a class="link-beacon hover:underline" href={VERIFY_SIGNATURES_GUIDE_URL} target="_blank" rel="noreferrer">Verify webhook signatures</a>.
    </p>
  </div>
  {#if subscriptions.length === 0}
    <p class="text-sm text-muted">Subscribe this endpoint to an event type first.</p>
  {:else}
    <ul class="space-y-2">
      {#each subscriptions as s (s.subscription_id)}
        {@const t = tests[s.subscription_id]}
        <li class="flex flex-wrap items-center gap-3 text-sm" data-testid="test-event-row">
          <button class="btn btn-ghost !px-3 !py-1" disabled={t?.busy} onclick={() => sendTest(s.subscription_id, eventFor(s))}>
            {t?.busy ? "Sending…" : "Send test event"}
          </button>
          {#if s.event_name === "*"}
            <input
              class="input mono text-xs !py-1 w-48"
              placeholder="event, e.g. order.paid"
              aria-label="Event to send"
              bind:value={eventNames[s.subscription_id]}
            />
          {:else}
            <span class="mono text-xs">{s.event_name}</span>
          {/if}
          <button class="text-xs text-muted hover:text-text" onclick={() => togglePayload(s.subscription_id, eventFor(s))}>
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
</section>
