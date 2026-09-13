---
type: Go Package
title: internal/webhooks/client
description: HTTP client for webhook delivery — transport, signing, SSRF protection
tags: [http-client, delivery, signing]
timestamp: 2026-06-22T00:00:00Z
---

# internal/webhooks/client

HTTP delivery client with connection pooling, payload signing, and SSRF
protection. Template transforms are delegated to `pkg/template`; the client
re-exports nothing — callers construct `template.WebhookTemplateContext` and
call `WebhookClient.TransformPayload`.

## Key Types

- `WebhookClient` — HTTP client with signed transport, delegates transforms to `pkg/template`, metrics
- `DeliveryRequest` — full delivery context (URL, headers, payload, secret, keys)
- `WebhookEnvelope` — default JSON body with version, event_id, event_name, timestamp, attempt, payload
- `Metrics` — client-side request counters and response time tracking

## Signing

Every delivery is dual-signed:
- **HMAC-SHA256** — prefix `v1,`, uses webhook secret
- **Ed25519** — prefix `v1a,`, uses per-webhook keypair

Standard Webhooks headers: `webhook-id`, `webhook-timestamp`, `webhook-signature`

## Citations

- `internal/webhooks/client/` — 11 files
