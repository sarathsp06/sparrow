---
type: Go Package
title: pkg/crypto
description: Envelope encryption (AES-256-GCM with per-record DEK) for webhook secrets and Ed25519 keys
tags: [crypto, encryption, security]
timestamp: 2026-06-22T00:00:00Z
---

# pkg/crypto

Envelope encryption service. Each `Encrypt` call generates a random 256-bit Data Encryption Key (DEK), encrypts plaintext with AES-256-GCM using the DEK, then wraps the DEK with the primary KEK from the `SPARROW_ENCRYPTION_KEYS` keyring.

Ciphertext carries a key ID for deterministic decrypt during rotations.

## Key Exports

- `NewService(key []byte) (*Service, error)` — creates a convenience single-key keyed-envelope service
- `NewKeyring(keys []Key, primaryID string) (*Keyring, error)` — validates a KEK ring with one primary key
- `NewServiceFromKeyring(keyring *Keyring) (*Service, error)` — creates a rotation-friendly service
- `ParseKey(rawHex string) ([]byte, error)` — decodes 64-char hex
- `Encrypt(plaintext) ([]byte, error)` / `Decrypt(ciphertext) ([]byte, error)`
- `EncryptString` / `DecryptString`
- `EncryptJSON(v any) ([]byte, error)` / `DecryptJSON(ciphertext, v any) error`
- `EnvelopeEncrypt` / `EnvelopeDecrypt`

## Citations

- `pkg/crypto/` — 4 files
