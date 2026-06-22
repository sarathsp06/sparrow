---
type: Concept
title: Envelope Encryption
description: AES-256-GCM encryption with per-record DEK for webhook secrets and Ed25519 keys
tags: [encryption, crypto, security]
timestamp: 2026-06-22T00:00:00Z
---

# Envelope Encryption

Used to encrypt sensitive fields stored in the database:
- `webhook_registrations.webhook_secret` — HMAC signing secret
- `webhook_registrations.secret_headers` — custom headers sent with delivery
- `webhook_registrations.ed25519_private_key` — Ed25519 signing keypair

## Algorithm

1. `SPARROW_ENCRYPTION_KEY` (64-char hex → 32 bytes) is the **Key Encryption Key (KEK)**
2. Each `Encrypt` call generates a random 256-bit **Data Encryption Key (DEK)**
3. Plaintext encrypted with AES-256-GCM using the DEK
4. DEK is wrapped (encrypted) with the KEK
5. Format: `[encrypted_DEK][nonce][ciphertext]`

## Citations

- `pkg/crypto/` — implementation
