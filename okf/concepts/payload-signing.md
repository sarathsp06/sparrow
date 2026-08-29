---
type: Concept
title: Payload Signing
description: Dual HMAC-SHA256 + Ed25519 signing on every delivery following Standard Webhooks format
tags: [signing, hmac, ed25519, security]
timestamp: 2026-07-06T16:46:48Z
---

# Payload Signing

Every delivery is **dual-signed** with both HMAC-SHA256 and Ed25519.

## Standard Webhooks Format

Message to sign: `{msg_id}.{timestamp}.{payload}`

| Algorithm | Header Prefix | Verification Requires |
|-----------|--------------|----------------------|
| HMAC-SHA256 | `v1,` | Shared secret (`webhook_secret`) |
| Ed25519 | `v1a,` | Public key (derived from `ed25519_private_key`, exposed via API) |

## Headers

- `webhook-id`
- `webhook-timestamp`
- `webhook-signature` — space-delimited: `v1,<base64> v1a,<base64>`

## Key Management

- Ed25519 keypair generated at webhook registration
- Private key stored envelope-encrypted in `ed25519_private_key` column
- Public key derived at runtime — not stored separately
- Exposed as `signing_public_key` in `RegisterWebhookResponse` and `RegisteredWebhook`
- Public-key presentation is handled behind the [webhooks service](/packages/internal-webhooks-service.md) seam via `WebhookSigningPublicKeyHex`, so transport modules do not decrypt private-key material directly

## Citations

- `db/migrations/000022.up.sql` — Ed25519 key column
- `internal/webhooks/client/` — signing implementation
- `internal/webhooks/webhook_service.go` — signing public-key presentation
- `internal/rest/conversions.go` — transport response conversion delegates signing-key presentation
