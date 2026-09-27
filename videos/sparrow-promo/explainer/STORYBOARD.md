---
title: The life of one webhook
format: 1920x1080 @ 30fps
duration: ~3:00 (derived from narration — src/explainer/timing.json)
arc: concept-explainer with process
audience: backend engineers who send (or are about to send) webhooks and have been burned by lost ones
message: Push an event once; Sparrow makes sure it lands — or tells you exactly why it didn't.
music: soft ambient pad (public/bgm/pad.mp3, synthesized by scripts/bgm.py)
voice: Edge neural · en-US-AndrewMultilingualNeural (scripts/voiceover.mjs)
---

# The life of one webhook — storyboard

Built with the **faceless-explainer** method (`.agents/skills/faceless-explainer`),
rendered with Remotion instead of HyperFrames so it reuses this project's
design system, components and SFX. Narration lives in `src/explainer/script.json`
(the SCRIPT); `scripts/voiceover.mjs` turns every cue into a clip and writes
`timing.json`, so **scene lengths and every reveal follow the voice**.

Every claim is repo-verifiable:

| Claim | Source |
|---|---|
| `POST /v1/consumers/{consumer}/events` returns immediately (201), async delivery | `internal/rest/event.go` (`pushEvent`) |
| idempotency key → original event back, `duplicate: true` | `internal/rest/event.go` `pushEventOutput` |
| fan-out by event name + consumer + label filters, one delivery job each, River in Postgres | `docs/.../how-it-works.mdx`, `internal/webhooks/queue/events_worker.go` |
| Go-template transform, fallback envelope `{version,event_id,event_name,timestamp,attempt,payload}` | `docs/FLOWS.md §5`, `internal/webhooks/queue/webhook_worker.go` |
| Standard Webhooks: `webhook-id` / `webhook-timestamp` / `webhook-signature`, `v1,` HMAC-SHA256 or `v1a,` Ed25519 over `{id}.{timestamp}.{payload}` | `internal/webhooks/client/request.go` |
| 10 error categories; retryable: server_error, timeout, connection_refused, network_error, rate_limited; terminal: client_error, dns_error, tls_error, unexpected_status | `docs/.../reference/error-classification.md`, `docs/FLOWS.md` |
| defaults max_retries 3, backoff 60s, delay = base·2^(n-1), capped 24h | `internal/webhooks/models.go:80`, `webhook_worker.go NextRetry` |
| 429 → snooze for Retry-After, not counted as an attempt | `docs/FLOWS.md §5` |
| health: healthy >90% & <5 consecutive fails; degraded 80–90%; unhealthy <80% or ≥5 consecutive | `internal/webhooks/store/health_repository.go` |
| `sparrow.webhook.health_changed` / `delivery_failed` → alert email | `guides/webhook-health-alerts.mdx` |
| retry a delivery by ID | `internal/rest/delivery.go` (`retryDelivery`) |
| recipes: slack, discord, sendgrid, twilio, ntfy, pagerduty, clickhouse; `sparrow use slack --event order.created` | `satellites/recipes/*.yaml`, `docs/.../satellites/cli.mdx` |
| embedded Svelte dashboard in the same binary (webhooks, events, deliveries, health), opt-in `SPARROW_SERVE_UI=true` | `README.md`, `web/src/routes/`, `internal/ui` |
| consumer portal `/portal` — consumers manage their own webhooks, subscriptions, deliveries via scoped expiring token | `docs/.../guides/portal-embedding.mdx` |
| PostgreSQL is the only required dependency; River queue in the same DB; no Redis/broker | `README.md` |
| OpenTelemetry traces propagate through every queue job; metrics + logs via OTLP | docs landing (`docs/src/pages/index.astro`), `why-sparrow.mdx` |
| comparison vs Svix / Convoy / Hookdeck / DIY (directional snapshot) | docs landing "How Sparrow stacks up", `why-sparrow.mdx` |
| MIT licensed; no per-message pricing | `LICENSE`, `why-sparrow.mdx` (promo Compare scene) |

## Video direction

- **Palette** (`src/theme.ts`): cream paper ground (persistent `Backdrop`), ink text,
  navy for terminals and the dark opening, **teal = success / Sparrow's path**,
  **coral = the event / attention / the current step**, red only for failure.
- **Type**: Inter display (hero words 96–180px, titles 76px), Fira Code for
  code / kickers / labels, Instrument Serif for the one human aside (Frame 2).
- **Motion grammar**: critically-damped settles (`useReveal`, damping 200) —
  never bouncy. **Every piece reveals on the cue that names it** (`cueFrame(id, i)`);
  nothing is on screen before the VO reaches it. Holds are still; the only
  permitted aliveness is flowing wire dashes and comet packets (the subject doing
  something), not breathing cards. No back-half camera pushes.
- **Consistent stage**: Frames 4–8 (plus 6b) are the five-step run — `StepRail` across the
  top, `StepTitle` upper-left, the working diagram in the centre-right 60%.
  Seam = push-slide from the right, repeated. Everything else = blur-crossfade.
- **Held beats**: end of Frame 3 (the event pill sits alone), end of Frame 8
  (the timeline resolves ✓), Frame 11 (the callback lands, still).
- **Caption band**: bottom 17% (y > 880) is the caption pill; primary content
  stays above it.
- **Negative list**: no purple/blue AI gradients, no bokeh, no invented metrics or
  logos, no screensaver float, no front-load-then-freeze, no `Math.random`
  (use Remotion `random(seed)`).

---

## Frame 1 — Webhooks, again (follows the promo Hook — src/scenes/Hook.tsx)
- id: 01-hook · transition_in: cut · type: hook · beat: recognition + mild dread
- hook strategy: Direct address → pain validation
- persuasion: Pain validation + rule of three
- focal: the two-line headline · roles: navy ground with the promo's scrolling ghost DIY delivery log (503 / timeout / 429 / retry n/5 / dead-letter) = background, "WEBHOOK DELIVERY" kicker = supporting

narrativeRole: Positions the video immediately as being about webhook delivery, in the viewer's own words.
keyMessage: Every product that emits events ends up owning webhook delivery — and its chores.

Reproduce the promo Hook's look exactly (dark ground, scrolling ghost log, radial dim, grain, `Kicker text="WEBHOOK DELIVERY"` visible from frame 0 so frame 1 reads without sound), but retimed to the VO:
- c0 "Your app emits events.": line 1 word-reveals (cream at 70%).
- c1 "The rest of your stack expects webhooks.": line 2 word-reveals, "webhooks." in coral.
- c2 "Which means retries. Signing. Health checks. Every time.": the mono pain line reveals one item per spoken word — `Retries.` `Signing.` `Health checks.` `Every time.` (last in coral) — then the coral underline sweeps under it. Hold.

## Frame 2 — The DIY stack collapses (follows the promo OldWay — src/scenes/OldWay.tsx)
- id: 02-pain · transition_in: push-slide UP (slide from-bottom, as in the promo) · type: pain_point → product_intro · beat: recognition → relief
- persuasion: Subtractive framing + concept announcement
- focal: the pile of DIY chips, then the Sparrow lockup · roles: cream backdrop; PostgreSQL chip = the survivor

narrativeRole: Shows the homemade stack teams build, then collapses it to "just Postgres" and names Sparrow.
keyMessage: Sparrow replaces the whole DIY delivery stack with one binary on Postgres.

Reproduce the promo OldWay's chips, wobble, drop-out and lockup, retimed to the VO:
- c0 "So teams build it: a Redis queue, retry crons, signing code, dashboards, alerts.": the promo's chip board piles up; chips named in the line land on their words (Redis · queues, retry cron jobs, HMAC signing code, consumer dashboard, monitoring + alerts); the rest (client SDKs, RabbitMQ · retries, rate limiting, payload audit logs, PostgreSQL, dead-letter queue, key rotation) fill in between, staggered across the cue.
- c1 "And events still quietly get lost.": the board wobbles/tilts with rising stress (as promo), a couple of chips flicker.
- c2 "With Sparrow, it's just Postgres.": everything but the PostgreSQL chip drops out of frame; the line "With Sparrow, it's just [PostgreSQL]" forms around the surviving chip (as promo).
- c3 "Sparrow: webhook delivery you run yourself.": the lockup lands — logo + "Sparrow" wordmark, subtitle **"webhook delivery you run yourself"** (the docs landing hero), and chips `MIT licensed` · `Go + PostgreSQL` · `OpenAPI 3.1` (landing hero badges). Hold.

## Frame 3 — Follow one event
- id: 03-meet · transition_in: blur-crossfade · type: product_intro · beat: orientation
- persuasion: Frame-then-fill
- focal: the `order.created` event pill · roles: small Sparrow lockup (continuity from Frame 2) upper third, journey line = supporting

narrativeRole: Promises the path the body follows.
keyMessage: We'll follow one event from push to delivered.

- start (before c0): the Sparrow lockup is already present, centred (match-cut from Frame 2's end), no subtitle.
- c0 "Let's follow one event, order.created,": lockup lifts to the upper third and shrinks; the coral `order.created` pill with its JSON preview pops in centre.
- c1 "from push, to delivered.": journey line draws with `push` / `delivered ✓` ends and five faint station dots. Hold.

## Frame 4 — Step 1: Push
- id: 04-push · transition_in: push-slide LEFT · type: feature_showcase · beat: comprehension
- persuasion: Demonstration + question→answer
- focal: terminal with the POST and its response · roles: StepRail (active 0), StepTitle "Push", Postgres drum = supporting

narrativeRole: Shows the push is one fast, safe call.
keyMessage: One POST; Sparrow stores it and answers immediately — and retries of the push itself are deduplicated.

- c0: StepRail + StepTitle "Push" in.
- c1 "One POST. … answers right away.": right 60%: Terminal types
  `curl -X POST /v1/consumers/acme/events?event=order.created` then `-d '{"payload":{…},"idempotency_key":"idem_ord_123"}'`; response block pops: `201 Created` · `{"event_id":"evt_7c1…","duplicate":false}`; a small "stored" drum (Postgres) left of terminal receives the packet (teal ripple). A tiny "≈ ms" is NOT shown (no invented metric).
- c2 "Push it again with the same idempotency key?": the same command re-types quickly (ghost second line).
- c3 "You get the original back. No duplicate.": response 2: `{"event_id":"evt_7c1…","duplicate":true}` — `duplicate:true` highlighted coral, matching event_id underlined in both responses; a "✕ no second delivery" stamp.

## Frame 5 — Step 2: Fan-out
- id: 05-fanout · transition_in: push-slide LEFT · type: feature_showcase · beat: "aha"
- persuasion: Progressive disclosure (one filter at a time) + causal chain
- focal: the event node splitting into delivery lanes · roles: StepRail (active 1), subscription cards = foreground, Postgres queue strip = supporting

narrativeRole: Explains how one event becomes N deliveries — and why some subscribers don't get it.
keyMessage: Matching is by event name, consumer and labels; each match becomes its own queued delivery.

- c0: StepRail + StepTitle "Fan-out".
- c1 "…every subscription that matches:": the `order.created` pill (with label chip `env=prod`) at left-centre; four subscription cards fan out on the right: `billing-svc · order.created`, `partner-acme · order.created`, `slack-ops · order.created`, `analytics · order.created · env=staging`.
- c2 "the event name, the consumer, even labels like env=prod,": three filter chips light in sequence on the pill → cards: `event ✓`, `consumer: acme ✓`, `labels env=prod` — the `analytics` card (env=staging) greys out with a ✕.
- c3 "…queues one delivery for each. Right in Postgres.": wires draw from pill to the three matching cards; three comets travel; beneath, a "river · postgres" queue strip receives three job tiles `delivery #1..#3`. Label "no broker · no Redis".

## Frame 6 — Step 3: Build the request
- id: 06-sign · transition_in: push-slide LEFT · type: feature_showcase · beat: fascination → "aha"
- persuasion: Build-up (one request assembles part by part) + generalization (webhook → any HTTP API)
- focal: ONE outgoing HTTP request card · roles: StepRail (active 2), StepTitle "Build the request", three generic destination endpoints = supporting, signature seal = hero at the end
- no brand logos, no Slack. Endpoints are generic: `hooks.partner.example`, `api.crm.example`, `billing.internal`.

narrativeRole: Shows that what Sparrow sends is a fully configurable HTTP request — shaped body, secret headers, any method — so it can call real APIs, not just webhooks; and it's always signed.
keyMessage: Envelope or template body + encrypted secret headers + any method = one event can drive almost any HTTP API, signed.

Sources: transform + envelope (FLOWS.md §5), secret headers envelope-encrypted and merged last (`internal/rest/conversions.go`, FLOWS.md §5), subscription method enum GET/POST/PUT/PATCH/DELETE (`internal/rest/subscription.go`), Standard Webhooks signing (`client/request.go`).

- c0 "Step three: build the request.": StepRail + StepTitle; an empty request card (≈55% of frame, centre-right) outlines itself: a method line slot, a HEADERS block, a BODY block.
- c1 "The standard envelope, or your own shape with a Go template.": BODY fills with the envelope JSON (`version`, `event_id`, `event_name`, `timestamp`, `attempt`, `payload`) key by key; on "your own shape" a small Go-template chip (`{{ .payload.order_id }}` …) slides over and the body morphs into a custom JSON (`{"order": "ord_123", "amount": 49.99, "status": "created"}`).
- c2 "Headers too, even an API token, encrypted at rest.": HEADERS fill: `Content-Type: application/json`, `X-Tenant: acme`, then `Authorization: Bearer ••••••••` with a lock icon and a small teal "encrypted at rest · AES-256-GCM" tag.
- c3 "Any method: POST, PUT, PATCH, DELETE.": the method token cycles POST → PUT → PATCH → DELETE on the spoken words (in-place token swap), then settles on `PUT https://api.crm.example/orders/ord_123`.
- c4 "So one event can call almost any HTTP API.": three generic destination endpoints appear to the right/below (partner webhook `POST hooks.partner.example`, CRM API `PUT api.crm.example`, internal `PATCH billing.internal`); dotted wires draw from the card and comets travel to each.
- c5 "And every request is signed: HMAC or Ed25519.": `webhook-id: msg_…`, `webhook-timestamp: …`, `webhook-signature: v1,K5o…=` stamp onto the headers block; two chips `HMAC-SHA256` · `Ed25519`; teal "signed" seal. Hold.

## Frame 6b — Recipes
- id: 06b-recipes · transition_in: push-slide LEFT · type: benefit_highlight · beat: delight
- persuasion: Callback to Frame 6 ("exactly that, pre-packaged") + enumeration
- focal: the request card from Frame 6 collapsing into a recipe file, then the recipe grid · roles: StepRail (active 2), terminal = supporting
- Slack must NOT lead: it's just one tile among seven, never highlighted.

narrativeRole: Recipes are Frame 6's configurable request, pre-packaged per destination — no template writing.
keyMessage: One YAML file per destination bundles URL + headers + transform template; one CLI command applies it.

- c0 "Recipes are exactly that, pre-packaged.": a compact version of Frame 6's request card (method · headers · body) sits left and folds into a `pagerduty.yaml` file card mirroring `satellites/recipes/pagerduty.yaml`: `webhook:` → `url: https://events.pagerduty.com/v2/enqueue`, `headers:`; `subscription:` → `transform_template: |` with 2–3 template lines (`"event_action": "trigger",` · `"dedup_key": {{.event_id | json}},`).
- c1 "PagerDuty, Twilio, SendGrid, ClickHouse and more, one YAML file each.": the recipe grid pops in — PagerDuty, Twilio, SendGrid, ClickHouse land on their spoken names; Discord, ntfy, Slack fill in together on "and more" — each tile with icon + label + output format (Events API v2 · Twilio Messages · SendGrid v3 · JSONEachRow · embeds · ntfy topic · Block Kit).
- c2 "One command applies one: sparrow use pagerduty.": Terminal types `$ sparrow use pagerduty --event order.created`; the PagerDuty tile lights teal, wire + comet from terminal to it; footnote "helper CLI · satellites/sparrow". Hold.

## Frame 7 — Step 4: Send & classify
- id: 07-classify · transition_in: push-slide LEFT · type: feature_showcase · beat: clarity
- persuasion: Comparison of two options (sorting into two bins) + numbered enumeration
- focal: two bins — RETRY vs STOP · roles: StepRail (active 3), response tokens = foreground

narrativeRole: Teaches that Sparrow reads *why* a delivery failed and decides whether retrying can help.
keyMessage: Retryable failures get another try; terminal ones stop immediately.

- c0: StepRail + StepTitle "Send & classify". A request arrow leaves towards an endpoint.
- c1 "…one of ten categories.": ten category tokens spray in (mono chips) in a loose cloud: success, server_error, timeout, connection_refused, network_error, rate_limited, client_error, dns_error, tls_error, unexpected_status — a count "10" ticks up.
- c2 "Server errors and timeouts? Worth another try.": left bin "↻ retry" (teal) — server_error (5xx), timeout, connection_refused, network_error, rate_limited (429) fly into it.
- c3 "DNS, TLS, or a 4xx? Retrying won't help,": right bin "■ stop" (red-ish) — dns_error, tls_error, client_error (4xx), unexpected_status fly in.
- c4 "so Sparrow stops.": the stop bin locks (lid), `success` token goes to a small ✓ at top. Hold.

## Frame 8 — Step 5: Back off
- id: 08-backoff · transition_in: push-slide LEFT · type: feature_showcase · beat: confidence
- persuasion: Worked example with real numbers + value-scaled timeline
- focal: an exponential retry timeline · roles: StepRail (active 4), attempt markers = foreground, Retry-After snooze = supporting

narrativeRole: Makes the retry schedule concrete and shows 429s are respected.
keyMessage: 1m → 2m → 4m by default, doubling (capped at 24h); a 429 waits Retry-After without burning an attempt.

- c0: StepRail + StepTitle "Back off".
- c1 "three retries: one minute, two, then four.": a long horizontal timeline (full width strip, y≈520). Attempt 1 ✕ at t0; gaps draw proportional to 60s, 120s, 240s with labels `+1m`, `+2m`, `+4m` revealed with the words "one", "two", "four"; attempts 2 and 3 ✕.
- c2 "Doubling every time.": formula card `delay = 60s × 2ⁿ⁻¹  (max 24h)`; the gap brackets pulse once in sequence.
- c3 "And a 429, slow down?": above the timeline, a `429 Too Many Requests · Retry-After: 30` response chip appears on a side lane.
- c4 "Sparrow waits out Retry-After, without spending an attempt.": a snooze arc (dashed, "zZ") jumps over, attempt counter stays `3 / 4` (not incremented); attempt 4 lands ✓ teal with a ripple. Hold.

## Frame 9 — Health
- id: 09-health · transition_in: blur-crossfade · type: benefit_highlight · beat: foresight
- persuasion: Frame-then-fill (three states) + thresholds as data-viz
- focal: a health gauge / segmented arc · roles: state chips, alert email card = supporting

narrativeRole: Zooms out from one delivery to the webhook's overall health, and shows ops gets told.
keyMessage: Each webhook has a health state computed from success rate and consecutive failures; changes can alert on-call.

- c0: kicker "PER-WEBHOOK HEALTH"; a webhook card `partner-acme · https://api.acme.example/hooks` with a small state badge "unknown".
- c1 "Healthy, above ninety percent success.": large semicircular gauge draws; teal segment 90–100% labelled `healthy > 90%`; needle sweeps to 96%, badge → healthy.
- c2 "Degraded, between eighty and ninety.": amber segment 80–90 draws; needle drops to 86%, badge → degraded.
- c3 "Five failures in a row? Unhealthy.": five ✕ dots tick in a row under the gauge; red segment <80 draws; needle → unhealthy.
- c4 "And every change can email your on-call.": an email card slides in right: `sparrow.webhook.health_changed` · "partner-acme: degraded → unhealthy" · to `oncall@acme.example.com`.

## Frame 10 — The receipt
- id: 10-record · transition_in: blur-crossfade · type: social_proof · beat: trust
- persuasion: Demonstration (a delivery log) + callback
- focal: delivery attempts table · roles: retry button = supporting

narrativeRole: Proves nothing is hidden — every attempt is inspectable and re-drivable.
keyMessage: Every attempt is recorded, and you can retry any delivery by ID.

- c0 "Nothing is a mystery.": kicker "DELIVERY dlv_4e9…" and a table header fades in.
- c1 "Every attempt is recorded: status, timing, error category.": rows reveal one per word-group: `#1 503 · 1.2s · server_error`, `#2 —  · 30.0s · timeout`, `#3 connection refused · connection_refused`, `#4 200 · 84ms · success ✓`; column headers "status / time / category" highlight as named.
- c2 "When the partner is back, retry it by ID.": a mono command chip `POST /v1/consumers/acme/deliveries/dlv_4e9…:retry` appears with a button press; a fresh row `#5 200 ✓` slides in.

## Frame 10b — The dashboard
- id: 10b-dashboard · transition_in: blur-crossfade · type: benefit_highlight · beat: relief + delight
- persuasion: Demonstration (a UI reconstruction — the topic IS the interface here) + rule of three
- focal: a browser-framed dashboard mock · roles: sidebar nav = supporting, portal card = supporting

narrativeRole: Shows the operator view comes free — no extra service to deploy.
keyMessage: The dashboard ships in the same binary; the consumer portal lets customers self-serve.

- c0 "And you don't have to dig through logs.": a small mono log tail (a few grey JSON log lines) at left, quickly blurring/dimming.
- c1 "A dashboard ships inside the same binary: webhooks, events, deliveries, health.": a browser frame (`localhost:8080`) rises into the centre (≈60% of frame) showing a reconstruction of Sparrow's Svelte dashboard: left sidebar (Dashboard, Webhooks, Events, Deliveries, Portal — from web/src/routes), main area with a webhooks table (partner-acme healthy, billing-svc healthy, slack-ops degraded) and a delivery sparkline. Each of the four spoken nouns highlights its sidebar item / panel in turn. A chip "same binary · SPARROW_SERVE_UI=true".
- c2 "Plus a consumer portal, so your customers manage their own webhooks.": a second, smaller card slides in front-right: "acme · consumer portal" at `/portal#token=spt_…` showing only acme's webhooks + "Add endpoint" button.
- c3 "And OpenTelemetry traces every event, end to end.": a compact trace waterfall strip (OpenTelemetry) draws beneath/over the browser: spans `POST /v1/…/events` → `event.fanout` → `webhook.deliver` → `http.post partner-acme`, each bar extending left→right in sequence; label "OpenTelemetry · traces + metrics + logs". Hold.

## Frame 11 — Callback (plays after Frame 12b)
- id: 11-takeaway · transition_in: blur-crossfade · type: branding · beat: "now I get it"
- persuasion: Callback (the hook's packet) + distillation
- focal: the packet landing · roles: wire from hook = supporting

narrativeRole: Answers the hook's question with the image that opened it.
keyMessage: Push once; Sparrow does the rest.

- c0 "So, where did order.created end up?": the hook's composition returns in cream: service chip left, endpoint right, dotted wire; question in small mono above.
- c1 "Exactly where you sent it.": the coral packet travels, the endpoint is green now, packet lands with teal ripple and ✓ "delivered · attempt 4".
- c2 "Push once. Sparrow does the rest.": the headline "Push once. Sparrow does the rest." word-reveals (Inter 96px), "Sparrow" in teal. Hold still.

## Frame 12a — What it needs
- id: 12a-open · transition_in: blur-crossfade · type: benefit_highlight · beat: conviction
- persuasion: Subtractive framing (what you *don't* need) + distillation
- focal: a single PostgreSQL drum as the only dependency, then the MIT badge · roles: struck-through Redis/broker/worker tiles = supporting

narrativeRole: Removes the last objection — operational cost and licensing.
keyMessage: PostgreSQL is the only dependency, and it's MIT — no per-message pricing.

- c0 "So what does it need to run?": question in Inter 88px centred-upper.
- c1 "Just PostgreSQL. The queue lives there too. No Redis, no broker.": a single "PostgreSQL" database drum grows centre-left; inside it a "River queue" layer fills in on "the queue lives there too"; to the right, three ghost tiles `Redis`, `message broker`, `worker fleet` each get struck through (coral line) on "No Redis, no broker" (third on the same beat).
- c2 "And it's MIT licensed: free to run, fork and ship. No per-message pricing.": an "MIT License" badge stamps in (teal seal), three small verbs `run · fork · ship` reveal per word, and a price-tag chip "per-message pricing" struck through. Hold.

## Frame 12b — How it stacks up (adapts the promo Compare — src/scenes/Compare.tsx)
- id: 12b-compare · transition_in: blur-crossfade · type: social_proof · beat: conviction
- persuasion: Comparison of options + progressive disclosure
- focal: comparison table, Sparrow column highlighted · roles: vendor columns = supporting, footnote = supporting

narrativeRole: Positions Sparrow against the alternatives buyers already know.
keyMessage: Sparrow is the fully-MIT, Postgres-only option with dual signing, recipes, a portal and self-monitoring built in.

Table (docs landing "How Sparrow stacks up", verbatim values): columns Sparrow · Svix · Convoy · Hookdeck · DIY. Rows:
- Fully open source (MIT): Yes · Partial · No (Elastic 2.0) · Partial · Yes
- Core infrastructure: PostgreSQL only · PostgreSQL + Redis · PostgreSQL + Redis · SaaS · Varies
- Webhook signing: HMAC-SHA256 + Ed25519 · HMAC-SHA256 · HMAC-SHA256 · HMAC-SHA256 · Manual
- Prebuilt integrations: Recipes · Limited · Limited · Yes · Manual
- Consumer portal: Yes, self-hosted · Yes · Yes · Yes · No
- Self-monitoring alerts: System events + email · Operational webhooks · Alert configs · Issue alerts · Manual
Footnote (always, small): "Directional snapshot — vendors change packaging often; verify against their docs."

- c0 "How does that stack up?": kicker "HOW IT COMPARES", header row + Sparrow column highlight band.
- c1 "Svix and Convoy add Redis. Hookdeck's core is SaaS.": rows 1–2 reveal; the "PostgreSQL + Redis" cells (Svix, Convoy) and "SaaS" (Hookdeck) get a coral emphasis as named, Sparrow's "PostgreSQL only" teal.
- c2 "Sparrow is fully MIT, with dual signing, recipes, a portal and self-monitoring.": rows reveal on their spoken words — MIT (row 1 Sparrow cell pulses teal), signing, integrations, portal, self-monitoring. Hold.

## Frame 12 — Run it
- id: 12-cta · transition_in: blur-crossfade · type: cta · beat: resolve
- persuasion: Distillation + direct call to act
- focal: typed `docker compose up -d` · roles: logo lockup, github URL = supporting

narrativeRole: Gives the one action to take.
keyMessage: One binary, one Postgres, one command.

- c0 "One Go binary. One Postgres.": two tokens slam in side-by-side: `1 × Go binary` and `1 × PostgreSQL` with a "+" and "= Sparrow" lockup.
- c1 "docker compose up, and you're delivering.": terminal types `$ docker compose up -d`; beneath: "MIT licensed · github.com/sarathsp06/sparrow". Final frame is the only exit: gentle fade at the very end.
