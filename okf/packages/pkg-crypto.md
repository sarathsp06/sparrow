---
type: Go Package
title: pkg/crypto
description: Envelope encryption (AES-256-GCM with per-record DEK) for webhook secrets and Ed25519 keys
tags: [crypto, encryption, security]
timestamp: 2026-06-22T00:00:00Z
---

# pkg/crypto

Envelope encryption service. Each `Encrypt` call generates a random 256-bit Data Encryption Key (DEK), encrypts plaintext with AES-256-GCM using the DEK, then wraps the DEK with the KEK (Key Encryption Key from `SPARROW_ENCRYPTION_KEY`).

Supports backward-compatible legacy direct AES-256-GCM fallback.

## Key Exports

- `NewService(key []byte) (*Service, error)` — creates from 32-byte KEK
- `ParseKey(rawHex string) ([]byte, error)` — decodes 64-char hex
- `Encrypt(plaintext) ([]byte, error)` / `Decrypt(ciphertext) ([]byte, error)`
- `EncryptString` / `DecryptString`
- `EncryptJSON(v any) ([]byte, error)` / `DecryptJSON(ciphertext, v any) error`
- `EnvelopeEncrypt` / `EnvelopeDecrypt`

## Citations

- `pkg/crypto/` — 4 files
