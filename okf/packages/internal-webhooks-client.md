---
type: Go Package
title: internal/webhooks/client
description: HTTP client for webhook delivery — transport, signing, SSRF protection, TLS opt-out, URL redaction
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

## Transports

Four `http.Client`s share one SSRF-guarded dialer: {verify TLS, skip TLS} ×
{follow redirects, don't}. A delivery picks one from `DeliveryRequest`:
`SkipTLSVerify` (set from the webhook's `verify_ssl=false`; the zero value
verifies) and `FollowRedirects`. Skipping verification keeps the SSRF dialer
and redirect re-validation.

## URL redaction

Webhook URLs often carry secrets in the path (Slack, Discord) or query, so
`RedactURL` reduces them to `scheme://host` (plus `/…` when anything followed)
and fully redacts unparseable URLs. It is applied to service and worker logs
and span attributes, to the `*url.Error` returned by `Send` (the error type and
wrapped cause are kept for classification), and — via `redactSpanURL` inside
the otelhttp transport — to the delivery span's `url.full` attribute.

## Signing

Every delivery is dual-signed:
- **HMAC-SHA256** — prefix `v1,`, uses webhook secret
- **Ed25519** — prefix `v1a,`, uses per-webhook keypair

Standard Webhooks headers: `webhook-id`, `webhook-timestamp`, `webhook-signature`

## Citations

- `internal/webhooks/client/` — 11 files
- `internal/webhooks/client/client.go` — transports, `redactSpanURL`, `RedactURL`
- `internal/webhooks/client/client_test.go` — self-signed TLS, redaction, and span tests
