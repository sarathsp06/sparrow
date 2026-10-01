---
title: Three nights of one webhook
format: 1920x1080 @ 30fps
duration: ~3:25 (derived from narration — src/explainer/timing.json)
arc: story-explainer with process
audience: backend engineers who send (or are about to send) webhooks and have been burned by lost ones
message: Push an event once; Sparrow makes sure it lands, tells you exactly why when it doesn't, and lets you fix history in one command.
music: soft ambient pad (public/bgm/pad.mp3, synthesized by scripts/bgm.py)
voice: Edge neural · en-US-AndrewMultilingualNeural (scripts/voiceover.mjs)
---

# Three nights of one webhook — storyboard

Built with the **faceless-explainer** method, rendered with Remotion so it reuses
this project's design system, components and SFX. Narration lives in
`src/explainer/script.json` (the SCRIPT); `scripts/voiceover.mjs` turns every cue
into a clip and writes `timing.json`, so **scene lengths and every reveal follow
the voice** (`cueFrame` / `wordFrame` in `src/explainer/timing.ts`).

The hero is one event, `item.shipped` — a parcel that has to reach every dock.
Frames 1–2 are the promo's opening, unchanged. The body is three short stories
about that parcel, each landing an outcome, instead of a feature tour.

Every claim is repo-verifiable:

| Claim | Source |
|---|---|
| `POST /v1/consumers/{consumer}/events` returns 201 immediately; `idempotency_key` → original event, `duplicate: true` | `internal/rest/event.go` (`pushEvent`, `pushEventOutput`) |
| fan-out by event name + consumer + labels; one delivery job each, River in Postgres | `docs/FLOWS.md`, `internal/webhooks/queue/events_worker.go` |
| envelope `{version,event_id,event_name,timestamp,attempt,payload}` or Go-template body; secret headers envelope-encrypted (AES-256-GCM); method GET/POST/PUT/PATCH/DELETE | `docs/FLOWS.md §5`, `internal/rest/conversions.go`, `internal/rest/subscription.go` |
| Standard Webhooks `webhook-id` / `webhook-timestamp` / `webhook-signature`; `v1,` HMAC-SHA256 and `v1a,` Ed25519 | `internal/webhooks/client/request.go` |
| one copy-in verify helper per language: Python, TypeScript, Java, Kotlin, Ruby, PHP, Rust, Elixir (+ the Go module); rejects missing headers, ±5 min timestamps, bad signatures | `client/verify/*`, `docs/.../guides/verify-signatures.mdx` |
| retryable: server_error, timeout, connection_refused, network_error, rate_limited; terminal: client_error, dns_error, tls_error, unexpected_status | `pkg/errors/category.go`, `docs/.../reference/error-classification.md` |
| defaults max_retries 3, backoff 60s, delay = base·2^(n-1), capped 24h | `internal/webhooks/models.go`, `webhook_worker.go NextRetry` |
| 429 → snooze for Retry-After, not counted as an attempt | `webhook_worker.go` ("Handle 429") |
| health: healthy >90% & <5 consecutive; degraded 80–90%; unhealthy <80% or ≥5 consecutive | `internal/webhooks/store/health_repository.go` |
| `sparrow.webhook.health_changed` → alert email | `internal/webhooks/queue/system_events.go`, `internal/rest/alert_config.go` |
| bulk retry: `GET …/deliveries?status=failed&created_after=…&prepare_retry=true` → `retry_id`; `POST …/deliveries:retry`; snapshot semantics | `internal/rest/delivery.go` (`DeliveryListParams`, `retryDeliveriesByWebhook`), AGENTS.md principle 1 |
| bulk re-push: `GET …/events?event=…&created_after&created_before&prepare_repush=true` → `repush_id`; `POST …/events:rePush`; job_type `event_repush` | `internal/rest/event.go` (`EventOccurrenceListParams`, `rePushEvents`) |
| pause / resume a webhook; paused deliveries snooze and resume | `internal/rest/webhook.go` (`pauseWebhook`, `resumeWebhook`), `webhook_worker.go` |
| consumer portal `/portal` with a scoped, expiring token link | `docs/.../guides/portal-embedding.mdx`, `internal/rest/access.go` |
| recipes: pagerduty, twilio, sendgrid, clickhouse, discord, ntfy, slack; `sparrow use <recipe> --event …` | `satellites/recipes/*.yaml`, `satellites/sparrow/use.go` |
| embedded dashboard, `SPARROW_SERVE_UI=true`; OpenTelemetry traces through every queue job | `README.md`, `internal/ui`, `docs/src/pages/index.astro` |
| PostgreSQL only; River queue in the same DB; MIT; no per-message pricing | `README.md`, `LICENSE`, `why-sparrow.mdx` |
| comparison vs Svix / Convoy / Hookdeck / DIY (directional snapshot) | docs landing "How Sparrow stacks up" |

## Video direction

- **Palette** (`src/theme.ts`): cream paper ground (persistent `Backdrop`), ink text,
  navy for terminals and the dark opening, **teal = success / Sparrow's path**,
  **coral = the parcel / attention / the current step**, red only for failure.
  Night two adds a translucent indigo wash that lifts into a coral dawn.
- **Type**: Inter display (hero words 92–156px, titles 40–64px), Fira Code for
  code / kickers / labels (never below 17px; body code 19–26px), Instrument Serif
  for the two human asides ("DIY means…", "you: asleep").
- **Scale rule** (from the review): the hero of every frame fills 40–60% of the
  canvas; nothing important sits in the caption band (y > 880); no scene opens
  on an empty stage.
- **Motion grammar**: critically-damped settles (`useReveal`, damping 200) —
  never bouncy. **Every piece reveals on the word that names it** (`wordFrame`);
  nothing is on screen before the VO reaches it. Holds are still; the only
  aliveness is flowing wire dashes, comets and the parcel travelling.
- **Recurring props**: the coral `Parcel`, receiving `Dock` cards with a lamp and
  shutter, the `NightKicker` with a clock, navy `Terminal`s.
- **Seams**: push-slide from the right into and between the three nights
  (frames 4, 5, 7, 8); blur-crossfade everywhere else; night two enters on a
  blur-crossfade so the tint can settle.
- **Held beats**: end of Frame 4 (three green ticks), the "you: asleep" beat in
  Frame 6, Frame 12 (the headline).
- **Negative list**: no purple/blue AI gradients, no bokeh, no invented metrics
  beyond the illustrative counts (128, 2417), no logos except the recipe icons,
  no `Math.random`.

---

## Frame 1 — Webhooks, again
- id: 01-hook · transition_in: cut · type: hook · beat: recognition + mild dread
- Unchanged from the previous cut (promo Hook, retimed to VO).

## Frame 2 — The DIY stack collapses
- id: 02-pain · transition_in: push-slide UP · type: pain_point → product_intro · beat: recognition → relief
- Unchanged from the previous cut (promo OldWay chips → "just PostgreSQL" → lockup).

## Frame 3 — Three nights
- id: 03-three · transition_in: blur-crossfade · type: product_intro · beat: orientation + stakes
- persuasion: Frame-then-fill + stakes
- focal: the parcel + `item.shipped` pill · roles: lockup (continuity) shrinks to the top, three chapter cards = supporting

narrativeRole: Promises the arc: a happy path, a bad night, a bad deploy.
keyMessage: We'll follow one parcel through three nights.

- c0 "Let's follow one event, item.shipped, through three nights.": lockup lifts; on "item" the coral parcel + pill pop centre with the payload.
- c1 "The happy path. The night the partner went dark. And the Tuesday you shipped a bug.": three 540px chapter cards land on their phrases (teal / indigo / coral tops). Hold.

## Frame 4 — Night one: the happy path
- id: 04-happy · transition_in: push-slide LEFT · type: feature_showcase · beat: comprehension → satisfaction
- persuasion: Demonstration on one stage + build-up
- focal: the three-column stage (terminal · request card · docks)

narrativeRole: Shows the whole pipeline once, fast, at readable scale.
keyMessage: One POST; idempotent; fan-out by name/consumer/labels; a fully-shaped, signed request per dock.

- c0 "Night one … one POST.": kicker NIGHT ONE · 22:04; the warehouse-app terminal types the POST with `idempotency_key: pkg_88a1`.
- c1 "…answers in the same breath: 201, event ID.": `201 Created` response, `event_id` glows on the word.
- c2 "Same parcel pushed twice? … no duplicate.": ghost re-type, `duplicate: true` in coral, stamp "same event · no duplicate".
- c3 "Then fan-out … even labels like region=eu": four docks slide in at right; chips `event ✓` `consumer ✓` `labels region=eu` land on their words; `analytics-us` greys with ✕ on "region".
- c4 "For each match, one request …": the request card assembles in the middle column: body envelope → template morph on "template", headers + lock on "secret"/"encrypted", the method token cycles on "any HTTP method".
- c5 "Signed, and out the door. Three parcels, three docks, three green ticks.": signature headers + seal on "signed"; three parcels ride comets to the three open docks, lamps go teal with ✓ on "three green ticks"; chip "3 / 3 delivered". Hold.

## Frame 5 — Night one: the other side
- id: 05-verify · transition_in: push-slide LEFT · type: benefit_highlight · beat: reassurance
- persuasion: Generalization (the receiver is solved too) + enumeration
- focal: the three signed headers at 28px · roles: customer dock (right), copy-in file (bottom-left), language chips

narrativeRole: Closes the loop: the customer can trust and verify the parcel with one file.
keyMessage: Standard Webhooks headers, one copy-in helper in nine languages, replay-safe.

- c0 "On the other side, your customer checks the signature.": parcel travels to the customer dock; `webhook-id` / `webhook-timestamp` / `webhook-signature` reveal on "signature".
- c1 "One file to copy in: Python, TypeScript, Java, Ruby, and five more.": `sparrow_verify.py` card (three lines); chips land on the four names, five ghost chips on "five more".
- c2 "Wrong key, replayed, or older than five minutes? Rejected at the door.": three red stamps slam onto the dock on their words; shutter closes red on "Rejected", then reopens teal "✓ verified". Hold.

## Frame 6 — Night two: the partner went dark
- id: 06-night · transition_in: blur-crossfade · type: feature_showcase → social_proof · beat: tension → relief
- persuasion: Story (setup → tension → turn → resolution) + worked example with real numbers
- focal: hub → partner dock on top; night stage = classify bins + retry timeline + health; dawn stage = the one-shot bulk retry terminal + attempt log

narrativeRole: Lives through an outage so the retry, health and recovery features are felt, not listed.
keyMessage: Sparrow classifies, backs off, honours 429, degrades health, wakes on-call, and lets you retry the whole night in one snapshot.

- c0 "Night two. 3:12 a.m. The partner's dock is closed: 503.": indigo wash, moon, clock 03:12; the parcel comet hits the dock, the shutter slams (red lamp), `503` badge, comet bounces back.
- c1 "Sparrow reads the answer. Server error? Worth another try. DNS, TLS, 4xx? It stops.": two bins under the hub: `↻ retry` (5xx · timeout · refused · network · 429) on "worth", `■ stop` (4xx · DNS · TLS) on "stops".
- c2 "So it backs off: one minute, two, four, doubling, capped at a day.": full-width timeline; ✕ #1, `+1m` #2, `+2m` #3, `+4m` on their words; clock ticks 03:13 → 03:15 → 03:19; chip `delay = 60s × 2ⁿ⁻¹ · max 24h` on "capped".
- c3 "A 429 with Retry-After? It waits exactly that long, and it doesn't count as an attempt.": `429 · Retry-After: 120` chip; dashed snooze arc on "waits"; `#4 · still 3 / 4 attempts` on "doesn't".
- c4 "Fail after fail, the webhook's health slides: healthy, degraded, unhealthy.": a health bar under the dock; badge + colour change on each word (96% → 86% → 71%).
- c5 "Five in a row, and your on-call gets an email. You? You're asleep.": five ✕ dots on "five"; the email card (`sparrow.webhook.health_changed`, to oncall@) on "email"; serif "you: asleep · z z z" on "asleep". Held beat.
- c6 "6:40. The partner is back. Every failed delivery from tonight: one snapshot, one retry.": wash lifts to a coral dawn, clock 06:40, sun; shutter opens, teal ripple on "back"; terminal types `GET …/deliveries?status=failed&created_after=…&prepare_retry=true` → `{ retry_id, total: 128 }` on "snapshot"; `POST …/deliveries:retry` on "retry"; a stream of comets, counter `128 / 128 ✓` on the dock.
- c7 "A snapshot, not a live query: what you saw is exactly what gets retried.": SNAPSHOT card: "128 deliveries, frozen at 06:40 · new failures after 06:40 → not included" on "exactly".
- c8 "And every attempt is on record: status, timing, category. Nothing is a mystery.": attempt log strip (#1–#5 with times, status, category); column headers brighten on their words. Hold.

## Frame 7 — Night three: Tuesday's bug
- id: 07-tuesday · transition_in: push-slide LEFT · type: feature_showcase → benefit_highlight · beat: relief + control
- persuasion: Worked example + callback (same snapshot mechanism as Frame 6)
- focal: the re-push terminal · roles: parcel row with weights (top), partner dock with pause/resume (right), portal card (bottom-right)

narrativeRole: Shows Sparrow fixes history and gives operators and customers control.
keyMessage: Re-push a whole day from a snapshot; pause/resume a partner; customers self-serve in the portal.

- c0 "Night three. Tuesday's deploy put the wrong weight on every parcel.": six parcel cards; on "wrong" they turn red with `weight_kg: 0`; deploy tag + diff on the same word; "× every parcel, all day" on "every".
- c1 "Fix the bug. Then re-push every item.shipped from Tuesday: one snapshot, one job.": diff resolves green + `✓ fixed` on "fix"; row shrinks up; terminal types `GET …/events?event=item.shipped&created_after&created_before&prepare_repush=true` → `{ repush_id, total: 2417 }` on "snapshot"; `POST …/events:rePush` on "job"; progress bar `event_repush · 2417 / 2417 ✓`; parcels turn teal as the bar passes them.
- c2 "Partner mid-deploy? Pause their webhook. Deliveries wait. Resume when they're ready.": partner dock with `⏸ paused` on "pause"; queued bars count up to 37 on "wait"; `▶ resumed`, bars drain on "resume"; footnote `POST …/webhooks/{id}:pause · :resume`.
- c3 "And your customers watch all of it themselves, in a portal you hand them with one link.": portal browser card (`/portal#token=spt_…`, DHL · consumer portal, their webhook, "128 retried · 2417 re-pushed"); "one scoped, expiring link" line on "link". Hold.

## Frame 8 — Shape the request → any HTTP API → recipes
- id: 08-recipes · transition_in: push-slide LEFT · type: feature_showcase → benefit_highlight · beat: fascination → delight
- persuasion: Build-up (one request assembles part by part) → generalization (webhook → any HTTP API) → callback ("recipes are those templates, pre-built")
- focal: act 1 = ONE outgoing request card (780px) + its Go template; act 2 = the 7-tile recipe grid · roles: three generic endpoints = supporting, `pagerduty.yaml` = the template, pre-packaged

narrativeRole: Explains, in order, that the body is shaped by a Go template, headers and method are yours, so any HTTP API is reachable; then names recipes as those templates pre-built.
keyMessage: Template + your headers + any method = any HTTP API; recipes are that, packaged per destination.

- c0 "What goes out is yours to shape. A Go template turns the envelope into whatever the receiver expects.": request card with the envelope body on "shape"; the dark Go-template card (`{{ .payload.parcel }}`…) slides in on "template"; a coral comet crosses into the body and it morphs into the rendered JSON on "whatever".
- c1 "Add your own headers, even an API token, encrypted at rest. Any method: POST, PUT, PATCH, DELETE.": headers fill on "headers", `authorization: Bearer ••••` on "token", lock + `encrypted at rest · AES-256-GCM` on "encrypted"; the method token cycles POST → PUT → PATCH → DELETE on the words.
- c2 "So one event can call almost any HTTP API, not just a webhook.": template card leaves; three generic endpoints (POST hooks.partner / PUT api.crm / PATCH billing.internal) land at right, wires + comets on "almost"; chip "any HTTP API · not just a webhook" on "not".
- c3 "Recipes are those templates, pre-built: PagerDuty, Twilio, SendGrid, ClickHouse and more, one YAML file each.": act 1 recedes; tiles land on their names, Discord / ntfy / Slack on "more"; chip "same template + headers, pre-built per destination" on "pre-built"; `pagerduty.yaml` (url, Authorization header, transform_template) lands with "Recipes".
- c4 "One command: sparrow use pagerduty.": terminal types `sparrow use pagerduty --event item.shipped`; wire + comet to the PagerDuty tile, which glows teal. Hold.

## Frame 9 — Seeing it
- id: 09-seeing · transition_in: blur-crossfade · type: benefit_highlight · beat: relief
- persuasion: Demonstration (UI reconstruction) + rule of three
- focal: the dashboard mock at 1380×540 · roles: OpenTelemetry strip full width

- c0 "A dashboard ships inside the same binary: webhooks, events, deliveries, health.": browser frame rises; each nav item lights on its word.
- c1 "And OpenTelemetry traces every parcel, push to dock.": four spans draw left→right across the full width. Hold.

## Frame 10 — What it needs
- id: 10-needs · transition_in: blur-crossfade · type: benefit_highlight · beat: conviction
- persuasion: Subtractive framing + distillation
- focal: the PostgreSQL drum (420px) · roles: struck ghosts (Redis, message broker, worker fleet, per-message pricing), MIT stamp

- c0 "So what does it need to run?": headline word-reveals.
- c1 "Just PostgreSQL. The queue lives there too. No Redis, no broker.": drum on "just"; "River queue · in here" on "queue"; ghosts struck on "Redis" / "broker".
- c2 "MIT licensed: free to run, fork and ship. No per-message pricing.": MIT stamp on "MIT"; run · fork · ship on their words; "per-message pricing" ghost struck. Hold.

## Frame 11 — How it compares
- id: 11-compare · transition_in: blur-crossfade · type: social_proof · beat: conviction
- persuasion: Comparison of options + progressive disclosure
- focal: full-width table (1850px), Sparrow column band · roles: footnote (21px)

- c0 "How does that stack up?": kicker, header row, Sparrow band.
- c1 "Svix and Convoy add Redis. Hookdeck's core is SaaS.": rows 1–2; Redis / SaaS cells go coral on their words.
- c2 "Sparrow is fully MIT, with dual signing, recipes, a portal and self-monitoring.": remaining rows reveal on their words; MIT cell pulses teal. Hold.

## Frame 12 — Callback
- id: 12-takeaway · transition_in: blur-crossfade · type: branding · beat: "now I get it"
- persuasion: Callback + distillation
- focal: the headline · roles: warehouse-app parcel → three docks

- c0 "So, where did item.shipped end up?": parcel + warehouse-app chip at left, three docks at right, wires draw.
- c1 "Every dock. Every time. Even the nights you slept.": a comet lands on each dock on "Every" / "Every" / "Even"; lamps teal, `✓ delivered`.
- c2 "Push once. Sparrow does the rest.": 104px headline word-reveals, "Sparrow" teal. Hold still.

## Frame 13 — Run it
- id: 13-cta · transition_in: blur-crossfade · type: cta · beat: resolve
- persuasion: Distillation + direct call to act
- focal: `github.com/sarathsp06/sparrow` at 64px · roles: `1 × Go binary + 1 × PostgreSQL = Sparrow`, the typed `docker compose up -d`

- c0 "One Go binary. One Postgres.": tokens slam in on "One" / "One", "= Sparrow" after "Postgres".
- c1 "docker compose up, and you're delivering.": terminal types the command; the GitHub URL and badges land on "delivering". Gentle fade at the very end.
