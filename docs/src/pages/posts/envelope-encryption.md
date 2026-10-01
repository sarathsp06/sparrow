---
layout: ../../layouts/BlogArticleLayout.astro
title: Protecting secrets with envelope encryption
author: Sparrow team
description: How Sparrow encrypts webhook credentials, detects tampering, and rotates keys without storing key material beside the data.
pubDate: 2026-10-01
tags: [security, encryption, architecture]
---

Webhook credentials are useful only when Sparrow can use them, which means the server must eventually decrypt them. That makes storage protection important: a database backup, replica, or accidental dump should not also become a list of credentials that can be replayed against downstream systems.

Sparrow protects webhook secrets and secret header values with envelope encryption. The short version is: each stored value gets its own data-encryption key, and that key is protected by a separately managed key-encryption key.

## What envelope encryption means

Envelope encryption uses two layers of keys:

- A **data encryption key (DEK)** encrypts one plaintext value.
- A **key encryption key (KEK)** encrypts, or *wraps*, the DEK.

The database stores the encrypted value and the wrapped DEK together. It does not store the KEK. Sparrow keeps KEKs in a small configured keyring, supplied through `SPARROW_ENCRYPTION_KEYS` rather than persisted with application data.

This split avoids using one long-lived key directly across every record. It also gives key management a clean boundary: rotating a KEK does not require using the new KEK to encrypt every plaintext value immediately.

## Why Sparrow needs it

Webhook secrets and secret headers are credentials, not ordinary configuration. Sparrow needs the plaintext at delivery time to sign requests or send authentication headers, but operators should not need to expose those values through normal reads, logs, or database access.

Encryption at rest addresses the database boundary. It helps protect credentials in:

- PostgreSQL backups and replicas
- Database exports used for diagnostics
- Rows read by someone who should not be able to use the underlying credential
- Storage media or snapshots outside the running Sparrow process

It is not a replacement for access control or secrets management. A running Sparrow instance with the KEK can decrypt a configured value when a delivery requires it, so the encryption keys must be provided through a protected secret manager, Kubernetes Secret, or equivalent deployment mechanism.

## How Sparrow implements it

Sparrow uses AES-256-GCM, an authenticated-encryption mode. Each `EnvelopeEncrypt` call performs these steps:

1. Generate a random 256-bit DEK.
2. Generate a random nonce and encrypt the plaintext with the DEK using AES-256-GCM.
3. Generate another nonce and wrap the DEK with the primary KEK.
4. Store the envelope version, key ID, wrapped DEK, data nonce, and ciphertext with its authentication tag.

The current keyed envelope is laid out as:

```text
[version] [key-id length] [key ID] [wrapped-DEK length] [wrapped DEK] [data nonce] [ciphertext + tag]
```

The key ID tells Sparrow which configured KEK to use when unwrapping the DEK. New writes use the configured primary key. During rotation, the old key remains available for decryption while the new key handles new records; the old key can be retired after existing values have been rewritten under the new key.

Configuration uses 32-byte keys encoded as hexadecimal:

```bash
export SPARROW_ENCRYPTION_KEYS="old=$(openssl rand -hex 32),new=$(openssl rand -hex 32)"
export SPARROW_ENCRYPTION_PRIMARY_KEY_ID=new
```

The server requires the keyring configuration. An encryption or decryption attempt without a configured key returns an error instead of silently storing or reading a plaintext secret.

At the application boundary, webhook registration encrypts `webhook_secret` and `secret_headers` before persistence. Delivery preparation decrypts them only when needed. Secret header values never round-trip through API responses; webhook secrets are masked after server-side decryption. If a configured secret cannot be decrypted, delivery fails closed rather than sending an unsigned or unauthenticated request.

## Encryption also detects changes

AES-256-GCM does more than hide bytes. It is an **AEAD** mode: authenticated encryption with associated data. Every ciphertext includes an authentication tag calculated from the encrypted content. Sparrow verifies that tag before returning plaintext.

That means a change to the ciphertext, wrapped DEK, nonce, or other authenticated encrypted data causes decryption to fail. An attacker cannot make a stored value decrypt to altered plaintext merely by editing the database. The same check catches the wrong key, a damaged record, and a tampered envelope.

This is integrity protection and key authentication, not a standalone public-key digital signature. The practical guarantee is the one Sparrow needs for stored secrets: decryptable data must also pass the cipher's authenticity check. The test suite exercises this by flipping a ciphertext byte and asserting that decryption returns an error.

## What this ensures

Envelope encryption gives Sparrow four useful guarantees for these credentials:

1. **Confidentiality at rest.** Database readers without the KEK cannot recover webhook secrets or secret header values.
2. **Tamper detection.** AES-GCM's authentication tag rejects modified or corrupted envelopes instead of returning silently altered plaintext.
3. **Rotation without an outage.** A keyring can retain old KEKs for reads while the primary KEK encrypts new values.
4. **Fail-closed delivery.** If a configured secret is missing, corrupted, or encrypted with an unavailable key, Sparrow records the failure and does not downgrade the request to an unsigned or secret-less delivery.

Encryption is one layer of the design. Sparrow combines it with API authentication, response masking, URL redaction, TLS, and outbound signature verification so the plaintext credential is useful only at the narrow point where delivery needs it.
