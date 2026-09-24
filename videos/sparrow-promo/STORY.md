# Sparrow promo — story & prompt

## Prompt (the brief this video was built from)

> Build a 45-second, silent (autoplay-safe), 1920×1080 promo for Sparrow — a free,
> MIT-licensed, **self-hosted webhook delivery service** written in Go — for the hero
> spot on its open-source website. Sparrow must read as a service you host inside your
> system ecosystem, NOT a CLI tool: your services publish events to its REST API, and
> it fans them out — signed, retried, health-tracked — to your services and your
> partners' endpoints. Payload transformation (Go-template recipes) reshapes one event
> for Slack, Discord, email, SMS, and more. The CLI appears only as a helper footnote.
> Every command, endpoint, and claim on screen must be verifiable in the repo. Open
> with the problem in one line, get to "here it is working" by ~11 seconds, stack
> short feature payoffs, and end with a visual CTA. No narration, no music, no
> invented metrics or logos.

## Research grounding

- First 8 seconds decide retention → hook states the problem immediately; frame 1 is legible without audio.
- Launch reels that convert are "ruthless about lead time to the feature working" (Linear/Figma pattern) → the architecture flow starts at ~11s and gets 15s, the longest beat.
- 45–90s format works best as stacked micro-payoffs, not a feature checklist → 5 payoff cards, 9s total.
- CTA must be visual → typed `docker compose up -d` + `MIT licensed · github.com/sarathsp06/sparrow`.

## Beats (45s @ 30fps = 1350 frames)

| # | t | Beat | On screen (all repo-verified) |
|---|---|------|-------------------------------|
| 1 | 0–4.8s | Pain | Kicker "✱ WEBHOOKS, AGAIN?" · "Retries. Signing. Health checks." coral underline |
| 2 | 4.8–10.8s | Old way → name the fix | "Postgres. ~~+ Redis.~~ ~~+ a worker fleet to babysit.~~" collapses into the Sparrow lockup: "a self-hosted webhook delivery service · one Go binary · one Postgres database · open source" |
| 3 | 10.8–25.8s | Proof (centerpiece) | Architecture flow: `orders-api` → `POST /v1/consumers/acme/events` (internal/rest/event.go) → Sparrow service card (queue · retries · signing · health) → fan-out pulses to partner API (✓ signed), internal service (✓ delivered), Slack #ops and email via transform. Caption: "Your services publish. Sparrow delivers — to your services and your partners." |
| 4 | 25.8–33.8s | Transforms | `order.created` JSON → Go template transform → Slack · Discord · Email (SendGrid) · SMS (Twilio) · ntfy · PagerDuty · ClickHouse (satellites/recipes). Footnote: `$ sparrow use slack --event order.created` — one command via the helper CLI |
| 5 | 33.8–39.8s | Micro-payoffs | At-least-once delivery · HMAC + Ed25519 signing · Encryption at rest (AES-256-GCM) · Per-webhook health tracking · Embedded dashboard / consumer portal / OpenAPI-first (README features) |
| 6 | 39.8–45s | CTA | "One binary. One database. Self-hosted." · typed `$ docker compose up -d` (deploy/docker-compose.yml) · "MIT licensed · github.com/sarathsp06/sparrow" |

## Design system

- Canvas: warm paper `#F4F2ED` (persistent `Backdrop`: drifting dot grid, slow teal/coral glows, film grain, vignette), ink `#0B0F14`, terminal navy `#181615` (the Hook opens dark), Go teal `#00ADD8` accent, coral `#F97316` as the voltage color.
- Type: Inter (display), JetBrains Mono (code/kickers), Instrument Serif (the DIY beat) via `@remotion/google-fonts`.
- Motion: blur-in word staggers, scramble-decode kickers, curved flowing connectors with comet-tail packets and arrival ripples (`components/Wire.tsx`), slow camera push-ins, a custom blur-fade transition (`components/blurFade.tsx`), and a sparrow fly-by synced to three of the whoosh SFX.
- Micro-demos in the feature cards are illustrative; the `webhook-signature: v1,… v1a,…` format comes from `internal/webhooks/client/request.go`, and the destination formats and the `{{.event_name | json}}` line come from `satellites/recipes/*.yaml`.

## Render

```
npx remotion render SparrowPromo out/video.mp4
```
