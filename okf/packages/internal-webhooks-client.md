---
type: Go Package
title: internal/webhooks/client
description: HTTP client for webhook delivery — transport, signing, template transforms, SSRF protection
tags: [http-client, delivery, signing]
timestamp: 2026-06-22T00:00:00Z
---

# internal/webhooks/client

HTTP delivery client with connection pooling, payload signing, Go template transforms, and SSRF protection.

## Key Types

- `WebhookClient` — HTTP client with signed transport, template engine, metrics
- `DeliveryRequest` — full delivery context (URL, headers, payload, secret, keys)
- `WebhookEnvelope` — default JSON body with version, event_id, event_name, timestamp, attempt, payload
- `TemplateEngine` — cached Go template execution (1 MB output limit, 5s CPU timeout)
- `WebhookTemplateContext` — snake_case keys for templates
- `Metrics` — client-side request counters and response time tracking

## Signing

Every delivery is dual-signed:
- **HMAC-SHA256** — prefix `v1,`, uses webhook secret
- **Ed25519** — prefix `v1a,`, uses per-webhook keypair

Standard Webhooks headers: `webhook-id`, `webhook-timestamp`, `webhook-signature`

## Citations

- `internal/webhooks/client/` — 16 files
