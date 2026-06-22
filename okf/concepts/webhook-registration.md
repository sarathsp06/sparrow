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
- `namespace` — FK to namespaces
- `active` — whether the webhook is accepting deliveries
- `webhook_secret` — HMAC secret (envelope-encrypted)
- `secret_headers` — encrypted headers sent with delivery
- `ed25519_private_key` — Ed25519 keypair (envelope-encrypted)
- `signature_type` — hmac | ed25519
- `rate_limit_rps` — max deliveries per second
- HTTP config: max_retries, retry_backoff, request_timeout, follow_redirects, verify_ssl, expected_status_codes, etc.

## Lifecycle

1. Register → optionally paused → active → unregister
2. Health: [healthy → degraded → unhealthy](/concepts/webhook-health.md)

[Subscriptions](/concepts/subscription.md) bind webhooks to events.

## Citations

- `db/migrations/000001.up.sql` — initial schema
- `db/migrations/000022.up.sql` — Ed25519 keys
- `db/migrations/000023.up.sql` — signature_type
