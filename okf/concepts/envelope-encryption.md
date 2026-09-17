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

1. A configured KEK ring is loaded from `SPARROW_ENCRYPTION_KEYS`
2. Each `Encrypt` call generates a random 256-bit **Data Encryption Key (DEK)**
3. Plaintext encrypted with AES-256-GCM using the DEK
4. DEK is wrapped (encrypted) with the primary KEK selected by `SPARROW_ENCRYPTION_PRIMARY_KEY_ID`
5. The format carries a key ID: `[version][kid_len][key_id][encrypted_DEK][nonce][ciphertext]`
6. During rotation, old KEKs stay in the ring for decryption until all ciphertext has been rewritten

## Citations

- `pkg/crypto/` — implementation
