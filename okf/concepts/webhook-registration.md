---
type: Concept
title: Webhook Registration
description: Registered webhook target with URL, HTTP config, secrets, and health tracking
tags: [webhook, registration]
timestamp: 2026-06-22T00:00:00Z
---

# Webhook Registration

A webhook registration represents a target URL that receives event deliveries. It includes HTTP delivery configuration, HMAC secret, Ed25519 keypair, and health tracking.

## Key Fields (23 columns)

- `url` — target URL
- `consumer` — FK to consumers
- `active` — whether the webhook is accepting deliveries
- `webhook_secret` — HMAC secret (envelope-encrypted)
- `secret_headers` — encrypted headers sent with delivery
- `ed25519_private_key` — Ed25519 keypair (envelope-encrypted)
- `signature_type` — hmac | ed25519
- `rate_limit_rps` — max deliveries per second
- HTTP config: max_retries, retry_backoff, request_timeout, follow_redirects, verify_ssl, expected_status_codes, etc.

## HTTP config limits

Checked by `WebhookHTTPConfig.ValidateConfig` on create **and** on update (the
merged config after a PATCH is validated, so an update cannot bypass them):

| Field | Allowed | Default |
|---|---|---|
| `max_retries` | 0–10 | 3 |
| `retry_backoff_seconds` | 1–3600 | 60 |
| `request_timeout_seconds` | 1–300 | 30 |
| `expected_status_codes` | non-empty, each 100–599 | 200, 201, 202, 204 |
| `content_type` | non-empty | `application/json` |
| `rate_limit_rps` | unset or > 0 | unset (no limit) |
| `verify_ssl` | bool | `true` (`false` skips certificate checks; SSRF checks still apply) |

A PATCH changes only the fields it sends.

## Lifecycle

1. Register → optionally paused → active → unregister
2. Health: [healthy → degraded → unhealthy](/concepts/webhook-health.md)

[Subscriptions](/concepts/subscription.md) bind webhooks to events.

## Citations

- `internal/webhooks/models.go` — `DefaultWebhookHTTPConfig`, `ValidateConfig`
- `internal/webhooks/webhook_service_registration.go` — `UpdateWebhookConfig` merge + validation
- `db/migrations/000001.up.sql` — initial schema
- `db/migrations/000022.up.sql` — Ed25519 keys
- `db/migrations/000023.up.sql` — signature_type
