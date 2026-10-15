---
format: 1920x1080
duration: 70s
message: "Simplicity with feature fullness: an API-first webhook delivery server you self-host"
arc: Title → Run it → Deliver it → Infer it → Transform it → Route it → Trust it → Build on it → CTA
audience: developers and platform teams browsing Product Hunt
mode: autonomous
music: confident minimal tech underscore, steady pulse, no vocals
---

## Video direction

- **Core idea:** simplicity with feature fullness. Sparrow is an **API-first** webhook delivery server: services push events and register webhooks over HTTP; the UI is an assistive console, not the product. Every screen frame says what a service does **via the API** and shows the UI as the window onto it.
- **Split-screen law (every frame with a capture or terminal):** a vertical split. **Left story panel** (~36% width, x 0–690): the mono kicker top-left (`label`, orange), then a one-or-two-line story in `quote-text` (lowercase Space Grotesk 700, cream, ~3.2cqw), then one mono **API line** in orange (e.g. `POST /v1/consumers/{consumer}/events`), then optionally one muted mono sub-line (cream-muted). The story reveals first, on its own cue; the API line types on behind a caret after it. **Right stage** (~62% width, x 740–1880, vertically centred, top ≥ 90px, bottom ≤ 900px): the capture or terminal(s). The panel and the stage never overlap; a 1px `border-dark` hairline divides them at x≈715.
- **Every capture and terminal is framed:** browser chrome (three dim dots + URL bar) or terminal chrome (three dim dots + title), background `ink-black-alt`, a **1px solid `fire-orange` border at 70% opacity** around the whole window, 8px radius, and a soft outer glow (`0 0 40px rgba(234,157,43,0.12)`) so the light capture never bleeds into the dark ground. Captures sit at fit-width inside the window; punch-ins use scale + counter-translate on an inner wrapper and stay clipped by the window.
- **Palette (frame.md, broadside remix):** dark register only — ground `ink-black`, text `cream`, one accent `fire-orange`; `success-green` for success states; `cream-muted`/`cream-hint` for secondary lines.
- **Type:** `display` on title and closing only; `quote-text` for the story lines; `label` mono for kickers and API lines; Fira Code for everything typed.
- **Motion grammar:** `power3` long-tail settles, nothing bouncy. Reveals are cue-paced: story line → API line types → the screen answers. Nothing front-loads; holds are still (caret blink at most).
- **Rhythm:** Frames 1 and 6 are held cards; Frame 3 is the busiest; Frames 5 and 5b are calm.
- **Negative list:** no bouncy eases, breathing, back-half drift, random particles, purple "AI" gradients, fake cursors, unframed captures, text over a capture, slideshow dumps or screensaver float. Nothing in the bottom 17%.

## Frame 1 — Title card

- scene: Sparrow mark and lowercase wordmark settle on ink-black; "self-hosted webhook delivery" types beneath, then the thesis line "simple to run. complete in features. api-first."
- voiceover: ""
- duration: 3.5s
- transition_in: cut
- status: animated
- src: compositions/frames/01-title.html
- type: hook
- persuasion: Authority by clarity
- beat: curiosity
- blueprint: titlecard-reveal (Reproduce)
- focal: assets/sparrow-logo.svg
- roles: sparrow-logo = cutout
- asset_candidates: assets/sparrow-logo.svg — the Sparrow brand mark, blue flying sparrow

narrativeRole: Name the product, the category and the thesis before any UI appears.
keyMessage: Sparrow is self-hosted, API-first webhook delivery — simple to run, complete in features.

Scene 1 (0.0–0.4s): static ink-black field — Centered template.
Scene 2 (0.4–1.3s): the ONE move — the sparrow mark (~22% of frame height) and the lowercase wordmark "sparrow" in `display` fade-in + settle up together as a centered lockup (`scale-swap-transition`, restrained, `power3`).
Scene 3 (1.3–2.3s): the kicker "self-hosted webhook delivery" types on behind an orange caret beneath the lockup (`discrete-text-sequence`), label mono, orange.
Scene 4 (2.3–3.0s): a second line fades in below in cream-muted `lead`: "simple to run · complete in features · api-first" (`dynamic-content-sequencing`).
Scene 5 (3.0–3.5s): held read. Caret blinks only.

## Frame 2 — Start the server

- scene: split — left: "one compose file. that is the whole install." with a small console chip under it running `curl localhost:8080/health`; right: a terminal types `docker compose up -d`, then a framed browser window shows the console at localhost:8080
- voiceover: "One compose file. Postgres and the server come with it. Nothing else to run."
- duration: 5.5s
- transition_in: crossfade
- status: animated
- src: compositions/frames/02-start-server.html
- type: product_intro
- persuasion: Friction reduction
- beat: ease
- blueprint: prompt-type-submit-generate (Adapt)
- focal: assets/ui-webhooks.png
- roles: ui-webhooks = cutout
- asset_candidates: assets/ui-webhooks.png — Webhooks console page, the first screen after start

narrativeRole: Prove setup is one compose file (it brings Postgres and the server) and the API plus its console are immediately there.
keyMessage: One compose file is the whole install; `docker compose up -d` and the API answers on :8080.

Adapt: keep "command types → the machine answers" on the right stage; on the left, the story is followed by a small framed console chip whose curl call answers too. No cursor. The health response is real: `{"status":"healthy","service":"sparrow"}`.
Scene 1 (0.0–0.6s): left panel: kicker "01 / start the server" fades in; the story "one compose file. that's the whole install. postgres and the server come with it." reveals in `quote-text`. Right stage: a framed terminal window (~60% of the stage height, top of stage) settles in (a smooth spring-pop entrance, no overshoot).
Scene 2 (0.6–1.8s): at the right terminal's orange `$` prompt, `docker compose up -d` types behind a caret (`discrete-text-sequence`).
Scene 3 (1.8–3.0s): output prints one line per cue in cream-muted (`dynamic-content-sequencing`): ` ✔ Network sparrow_default  Created`, ` ✔ Container sparrow-postgres-1  Started`, ` ✔ Container sparrow-sparrow-1  Started` (last check in `success-green`).
Scene 4 (3.0–4.3s): under the story in the left panel, a small framed console chip (full panel width, ~150px tall) settles in; at its orange `$` prompt `curl -s localhost:8080/health | jq .status` types (`discrete-text-sequence`), then `"healthy"` prints in `success-green`. Meanwhile on the right the terminal shrinks toward the top of the stage as a framed browser window pushes up from below it (`cut-the-curve`, matched upward velocity) with `assets/ui-webhooks.png` at fit-width, URL bar `localhost:8080`; the window takes ~75% of the stage height.
Scene 5 (4.3–5.5s): held read: story + curl chip left, terminal + console right. Still.

## Frame 3 — Push first, infer the schema

- scene: split — left: "push an event nobody defined yet. find it in the console. infer its schema from the real payload." + API line; right: a framed terminal pushes `payment.refunded`, the framed Events list shows it with an "Infer schema" link, then the Update Event Type page generates the schema
- voiceover: ""
- duration: 9s
- transition_in: push-slide LEFT
- status: animated
- src: compositions/frames/03b-schema-infer.html
- type: feature_showcase
- persuasion: Friction reduction
- beat: ease
- blueprint: prompt-type-submit-generate (Adapt)
- focal: assets/ui-schema-infer.png
- roles: ui-events-list = cutout · ui-schema-infer = cutout
- asset_candidates: assets/ui-events-list.png — Events list with the payment.refunded row and its "Infer schema" link; assets/ui-schema-infer.png — Update Event Type page with the schema helper, pushed payload on the right and the generated schema on the left

narrativeRole: Show that you never hand-write definitions — push first, the console finds the event, the schema is inferred from real payloads.
keyMessage: Unknown events are auto-registered on push and their schema is inferred from real payloads.

Adapt: keep "command types → the machine answers"; the answer is the auto-created line; then two real screens inside one framed window show the find-it and infer-it beats. No cursor. Terminal text verbatim: `event type "payment.refunded" auto-created` then `446b0e2a-c996-4878-8ba5-19c7b232fcda`.
Scene 1 (0.0–0.5s): left panel: kicker "02 / push first" fades in; story "push an event nobody defined yet. find it in the console. infer its schema from the real payload." reveals. Right stage: a framed slim terminal (top ~30% of the stage) settles in.
Scene 2 (0.5–2.0s): prompt types `sparrow push payment.refunded -d '{"payment_id":"pay_8f31","order_id":"42","amount":99,"currency":"USD"}'` (`discrete-text-sequence`).
Scene 3 (2.0–2.8s): two lines print one per cue (`dynamic-content-sequencing`): `event type "payment.refunded" auto-created` with `auto-created` in `success-green`, then the event id in cream-muted; the left API line types on: `POST /v1/consumers/{consumer}/events`.
Scene 4 (2.8–4.8s): a framed browser window rises below the terminal and fills the stage (`localhost:8080/events`, `assets/ui-events-list.png`); the camera pushes in (`coordinate-target-zoom`) onto the `payment.refunded` row; an orange marker outline draws around its amber "Infer schema" link; the left sub-line fades in: `GET /v1/event-types · auto-registered, v1, no schema yet`.
Scene 5 (4.8–7.2s): hard cut inside the window to `assets/ui-schema-infer.png` (URL bar `localhost:8080/events/payment.refunded/update`); the camera pushes in onto the right-hand helper panel — the pushed payload above the amber "Generate Schema" button — then pans left to the generated schema tree (`type: object`, `properties: {6 props}`, `required: [6 items]`); the left sub-line swaps to `Generated from 1 sample · versioned on change`.
Scene 6 (7.2–9s): held read on the schema. Still.

## Frame 4 — Subscribe and deliver

- scene: split — left: "register a webhook, subscribe it to the event, push again. signed, delivered, recorded." + API line; right: two framed terminals (`sparrow listen --event payment.refunded`, `sparrow push payment.refunded`), the signed delivery arriving, then the Deliveries row punching in with its green Success badge
- voiceover: ""
- duration: 11s
- transition_in: crossfade
- status: animated
- src: compositions/frames/03-register-deliver.html
- type: feature_showcase
- persuasion: Show-don't-tell proof
- beat: confidence
- blueprint: prompt-type-submit-generate (Adapt)
- focal: assets/ui-deliveries-refund.png
- roles: ui-deliveries-refund = cutout
- asset_candidates: assets/ui-deliveries-refund.png — Deliveries list with the payment.refunded Success row at the top

narrativeRole: The core loop: register + subscribe, push, receive HMAC-signed, see it recorded — all API calls.
keyMessage: Register a webhook over the API, push, and get a signed delivery with a receipt in the console.

Adapt: keep "command types → machine answers" twice across two stacked terminals in the right stage; the final answer is a punch-in onto the real Deliveries row. No cursor.
Terminal text is verbatim CLI output:
- top (listen): `listening on [::]:4568, registered webhook 8aaaf4d7… -> http://host.docker.internal:4568 (events: [payment.refunded])` / `press Ctrl-C to stop and delete the webhook`
- bottom (push): `86a095c1-b0e3-4928-90e9-18f0760d7d9b`
- top on delivery: `── delivery 09:42:02 ── POST /` / `signature: verified (v1, HMAC-SHA256)` / `Webhook-Id: msg_9c8f7f36-d798-5020-a8dc-f1a9b5937f15` / `X-Sparrow-Event-Id: 86a095c1-b0e3-4928-90e9-18f0760d7d9b` / `{ "event_name": "payment.refunded", "attempt": 1, "payload": { "amount": 99, "currency": "USD", "order_id": "42", "payment_id": "pay_8f31", "reason": "requested_by_customer" } }`
Scene 1 (0.0–0.6s): left panel: kicker "03 / subscribe and deliver" fades in; story "register a webhook and subscribe it. push again. signed, delivered, retried, recorded." reveals. Right stage: two framed terminals stacked (each ~46% of the stage height), the top one lit, the bottom dimmed to ~55%.
Scene 2 (0.6–2.2s): top prompt types `sparrow listen --event payment.refunded` (`discrete-text-sequence`); the left API line types on: `POST /v1/consumers/{consumer}/webhooks`.
Scene 3 (2.2–3.4s): top prints its two "listening / registered webhook" lines one per cue (`dynamic-content-sequencing`), webhook id and `payment.refunded` in orange; the bottom terminal brightens.
Scene 4 (3.4–5.2s): bottom prompt types `sparrow push payment.refunded -d '{"payment_id":"pay_8f31","order_id":"42","amount":99}'`; the event id prints in cream-muted.
Scene 5 (5.2–7.4s): the top terminal answers line by line (`dynamic-content-sequencing`): the `── delivery ──` header, `signature: verified (v1, HMAC-SHA256)` with a green keyword glow, then the headers and the payload with `"order_id": "42"` and `"amount": 99` in orange; the left sub-line fades in: `Webhook-Signature: v1,HMAC-SHA256 · retries with backoff`.
Scene 6 (7.4–9.2s): a framed browser window rises over both terminals inside the stage (`localhost:8080/deliveries`, `assets/ui-deliveries-refund.png`); the camera pushes in (`coordinate-target-zoom`) onto the TOP row — `payment.refunded · sparrow listen (temporary) · Success · 200` — so the row fills the stage width; the terminals fade beneath; a thin orange underline draws under the row and the green Success badge gets one finite glow pulse.
Scene 7 (9.2–11s): held read on the punched-in row. Still.

## Frame 5 — Transform and route

- scene: split — left: "reshape any event for its receiver. go templates, rendered live." + API line; right: the framed template editor with the AI prompt, the live Slack render, then the `sparrow use slack` chip
- voiceover: ""
- duration: 10s
- transition_in: crossfade
- status: animated
- src: compositions/frames/04-transform.html
- type: feature_showcase
- persuasion: Feature-to-benefit translation
- beat: control
- blueprint: panel-edit-live-sync (Adapt)
- focal: assets/ui-template-editor-ai.png
- roles: ui-template-editor-ai = cutout
- asset_candidates: assets/ui-template-editor-ai.png — Template editor with AI drawer, Slack template and live render

narrativeRole: Show that payloads can be reshaped per receiver, tested over the API, with AI help or one recipe command.
keyMessage: Transform events into what the receiver wants — test it over the API, draft it with AI, or apply a recipe.

Adapt: keep the live-sync couple — editor pane and RENDERED · LIVE pane change together — pointed at by camera punch-ins and marker outlines on the real capture, all clipped inside the framed window on the right stage. No cursor.
Scene 1 (0.0–0.6s): left panel: kicker "03 / transform" fades in; story "reshape any event for its receiver. go templates, rendered live against the sample." reveals. Right stage: a framed browser window settles in (`localhost:8080/…/subscriptions/…/edit`, `assets/ui-template-editor-ai.png`).
Scene 2 (0.6–2.8s): the camera pushes in (`coordinate-target-zoom`) onto the "PROMPT FOR AI" box (upper-left of the capture); an orange marker outline draws around the prompt text (`css-marker-patterns`).
Scene 3 (2.8–5.0s): the camera travels (`viewport-change`) to the TEMPLATE pane and the outline redraws around the `{{ printf "*Order #%s* — $%v" … }}` line; then continues right (`cut-the-curve`) to the RENDERED · LIVE pane where `"text": "New order #42 for $99"` gets an orange keyword glow; the left API line types on: `POST /v1/subscriptions:testTemplate`.
Scene 4 (5.0–6.4s): the camera pulls back to the full editor (one decelerating pull, then still); the left sub-line fades in: `POST /v1/subscriptions:draftTemplate · AI optional`.
Scene 5 (6.4–8.8s): a framed terminal chip slides up over the lower third of the window (inside the stage, above the caption band); at its `$` prompt `sparrow use slack --param webhook_url=https://hooks.slack.com/… --event order.created` types (`discrete-text-sequence`), then prints `recipe "slack" applied in consumer "default"` in `success-green`.
Scene 6 (8.8–10s): held read. Still.

## Frame 6 — Recipes for API receivers

- scene: split — left: "receivers that want an api call, not a webhook. one recipe each." + API line; right: the framed Register webhook page with the recipe grid, outlines stepping across SendGrid, Twilio, PagerDuty, Discord
- voiceover: ""
- duration: 8.5s
- transition_in: push-slide LEFT
- status: animated
- src: compositions/frames/04b-recipes.html
- type: feature_showcase
- persuasion: Value stacking
- beat: power
- blueprint: device-surface-showcase (Adapt)
- focal: assets/ui-register-recipes.png
- roles: ui-register-recipes = cutout
- asset_candidates: assets/ui-register-recipes.png — Register webhook page with the recipe grid open (Clickhouse, Cloudevents, Discord, Ntfy, Pagerduty, Sendgrid, Slack, Twilio)

narrativeRole: Widen the transform story: the same event becomes an email, an SMS, a page, a chat message — each recipe ships destination and transform.
keyMessage: Send webhooks as API requests to SendGrid, Twilio, PagerDuty and more — one recipe each.

Adapt: keep the held hero surface; the "screen advance" is a sequence of marker outlines stepping across the recipe cards inside the framed window, synced to the left panel's sub-line swapping. No cursor.
Scene 1 (0.0–0.5s): left panel: kicker "03b / recipes" fades in; story "receivers that want an api call, not a webhook. one recipe each: destination + transform." reveals. Right stage: a framed browser window settles in (`localhost:8080/webhooks/register`, `assets/ui-register-recipes.png`).
Scene 2 (0.5–1.6s): the camera pushes in (`coordinate-target-zoom`) onto the "START FROM A RECIPE" grid so the 3×3 cards fill the stage width; the left API line types on: `sparrow use sendgrid --event order.created`.
Scene 3 (1.6–4.4s): four orange marker outlines draw one per cue across the cards (`css-marker-patterns`, each fading as the next draws): Sendgrid → Twilio → Pagerduty → Discord (`dynamic-content-sequencing`); the left sub-line swaps in step (cream-muted mono): `email via SendGrid` → `SMS via Twilio` → `alerts via PagerDuty` → `chat via Discord`.
Scene 4 (4.4–5.6s): the outlines clear; one wider outline draws around the whole grid; the sub-line settles on `8 recipes · ClickHouse, CloudEvents, ntfy, Slack too`.
Scene 5 (5.6–6.5s): held read. Still.

## Frame 7 — Health and portal

- scene: split — left: "health per endpoint, a scoped portal for each consumer, prometheus metrics." + API line; right: the framed console with health badges, crossfading to the consumer portal, then a flash of raw `/metrics`
- voiceover: ""
- duration: 8s
- transition_in: push-slide LEFT
- status: animated
- src: compositions/frames/05-health-portal.html
- type: benefit_highlight
- persuasion: Risk reversal
- beat: trust
- blueprint: device-surface-showcase (Adapt)
- focal: assets/ui-webhooks.png
- roles: ui-webhooks = cutout · ui-portal = cutout
- asset_candidates: assets/ui-webhooks.png — Webhooks list with Health badges (Idle, Healthy); assets/ui-portal.png — consumer portal with endpoints, 95% success rate

narrativeRole: Operations proof — health, a consumer-scoped portal, and metrics, all readable over the API too.
keyMessage: Health, a consumer portal, and Prometheus metrics come built in.

Adapt: keep the held hero surface whose screens cycle; the surface is one framed browser window on the right stage, the screens are two real captures swapped by crossfade, and the third "screen" is the real metrics text. No cursor.
Metrics lines (verbatim from the real `/metrics` scrape; labels trimmed):
`# HELP sparrow_delivery_attempts_total Webhook delivery attempts that reached the receiver (or failed to), by result and error_category`
`sparrow_delivery_attempts_total{result="success"} 2`
`sparrow_delivery_attempts_total{result="failure",error_category="client_error"} 1`
`# HELP sparrow_delivery_attempt_duration_seconds Time from sending a webhook delivery attempt to the receiver's response, by result`
`sparrow_delivery_attempt_duration_seconds_count{result="success"} 2`
`sparrow_delivery_attempt_duration_seconds_sum{result="success"} 0.015`
Scene 1 (0.0–0.5s): left panel: kicker "04 / health and portal" fades in; story "health per endpoint. a scoped portal per consumer. metrics for prometheus." reveals. Right stage: a framed browser window settles in with `assets/ui-webhooks.png` (`localhost:8080/webhooks`).
Scene 2 (0.5–2.6s): the camera pushes in (`coordinate-target-zoom`) onto the HEALTH column; the four badges get staggered orange outlines that draw and fade (`css-marker-patterns`, `dynamic-content-sequencing`), ending on the green "Healthy" badge; the left API line types on: `GET /v1/webhooks?health=unhealthy`.
Scene 3 (2.6–3.2s): the camera pulls back; the URL bar swaps to `localhost:8080/portal`.
Scene 4 (3.2–5.6s): the screen crossfades to `assets/ui-portal.png`; the stat strip gets a gentle push-in and the "95%" a green keyword glow; the left sub-line fades in: `POST /v1/invites --consumer acme → one-time portal link`.
Scene 5 (5.6–7.0s): hard cut inside the window to the `/metrics` screen (URL bar `localhost:8080/metrics`); mono metrics lines print one per cue, orange names, cream values; the left sub-line swaps to `GET /metrics · OpenTelemetry traces + logs`.
Scene 6 (7.0–8s): held read. Still.

## Frame 8 — Built for developers

- scene: split — left: "the api is the product. the console is the assistive layer." + API line; right: a framed terminal runs the install one-liner, three fact cards assemble (one binary · postgres only · curl for everything), then the framed API Console punches in on its curl entries
- voiceover: ""
- duration: 10s
- transition_in: crossfade
- status: animated
- src: compositions/frames/05b-developer.html
- type: benefit_highlight
- persuasion: Friction reduction
- beat: ease + trust
- blueprint: grid-card-assemble (Adapt)
- focal: assets/ui-api-console.png
- roles: ui-api-console = cutout
- asset_candidates: assets/ui-api-console.png — the API Console drawer listing the UI's own requests as curl commands

narrativeRole: Close the proof with the operational simplicity story and the API-first stance: one binary and Postgres, and the console teaches you the API as you use it.
keyMessage: API-first and developer-friendly: single binary, Postgres only, and every console action shown as curl.

Adapt: keep the staggered cascade signature — three fact cards assemble one per cue inside the right stage — but open with a typed install command and finish by punching into the real API Console capture. No cursor.
Scene 1 (0.0–0.5s): left panel: kicker "05 / built for developers" fades in; story "the api is the product. the console is the assistive layer on top." reveals. Right stage: a framed slim terminal (top ~30% of the stage) settles in.
Scene 2 (0.5–2.0s): `curl -fsSL https://raw.githubusercontent.com/sarathsp06/sparrow/main/scripts/install.sh | sh` types at the `$` prompt (`discrete-text-sequence`).
Scene 3 (2.0–2.9s): two lines print one per cue (`dynamic-content-sequencing`): `Downloading sparrow-cli-0.13.0-darwin-arm64.tar.gz (v0.13.0)...` in cream-muted, then `Installed sparrow v0.13.0 to /usr/local/bin/sparrow` with `Installed` in `success-green`.
Scene 4 (2.9–5.2s): beneath the terminal, inside the stage, three `stat-card` fact cards assemble left-to-right one per cue (`dynamic-content-sequencing`), each a `label` kicker over a `lead` line: "01 / one binary" — "a single static Go binary, distroless image" · "02 / postgres only" — "queue, state and cache in PostgreSQL. no Redis, no broker" · "03 / curl for everything" — "the API Console shows every console request as copy-ready curl"; the left API line types on: `GET /openapi.yaml · generate your client`.
Scene 5 (5.2–7.6s): a framed browser window rises over the cards (`localhost:8080/webhooks`, `assets/ui-api-console.png`); the camera pushes in (`coordinate-target-zoom`) onto the API CONSOLE drawer so its four `$ curl 'http://localhost:8080/v1/…'` entries fill the stage width; an orange marker outline draws around the first curl block; the left sub-line fades in: `every UI action = one documented API call`.
Scene 6 (7.6–9s): held read on the console. Still.

## Frame 9 — Closing card

- scene: "simple to run. complete in features." then "mit licensed." with github.com/sarathsp06/sparrow and the docs URL, under the sparrow mark
- voiceover: ""
- duration: 4.5s
- transition_in: zoom-through
- status: animated
- src: compositions/frames/06-closing.html
- type: cta
- persuasion: Risk reversal
- beat: motivation
- blueprint: titlecard-reveal (Reproduce)
- focal: assets/sparrow-logo.svg
- roles: sparrow-logo = supporting
- asset_candidates: assets/sparrow-logo.svg — the Sparrow brand mark

narrativeRole: Send the viewer to the repo and docs, with the thesis and the license as the closer.
keyMessage: Simple to run, complete in features, MIT licensed — go get it.

Scene 1 (0.0–0.4s): ink-black field; the small sparrow mark + "sparrow" wordmark sit top-center as a persistent anchor (already settled at t=0).
Scene 2 (0.4–1.3s): the ONE move — "mit licensed." in `display` (lowercase, cream) slide-up crossfades into center (`discrete-text-sequence`); above it, a `label` mono line in orange reads "simple to run · complete in features · api-first".
Scene 3 (1.3–2.6s): beneath, `github.com/sarathsp06/sparrow` types on in mono, orange, behind a caret (`discrete-text-sequence` + `context-sensitive-cursor`); a 36×2 orange stub rule draws in above it (`svg-path-draw`).
Scene 4 (2.6–3.4s): a second mono line `sarathsp06.github.io/sparrow` fades in cream-muted below (`dynamic-content-sequencing`).
Scene 5 (3.4–4.5s): held to the last frame. Caret blinks only.
