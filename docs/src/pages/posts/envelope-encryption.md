---
layout: ../../layouts/BlogArticleLayout.astro
title: A letter that keeps its own secret
kicker: On envelope encryption
author: Sparrow team
description: How Sparrow protects webhook credentials with envelope encryption. A database backup turns up somewhere it should not be, so we open it and see what a stranger would find.
pubDate: 2026-10-01
tags: [security, encryption, architecture]
---

For most of history, a private letter was protected by two things: the honesty of the people who carried it, and a blob of wax.

The wax could not stop anyone from opening the letter. What it could do was tell the truth afterwards. A broken seal meant someone had looked. The letter did not need every courier to be trustworthy; it only needed to be impossible to tamper with quietly.

That is a better model for security than the one we usually reach for. We like to imagine walls that nobody gets past. Real systems leak. Backups get copied, replicas get shared, someone runs a diagnostic export and leaves it in the wrong bucket. The question worth designing for is not "will this ever escape?" but "when it does, what does it give away?"

## Friday afternoon

A message lands in your team channel: "Found a copy of last month's Postgres dump in the shared analytics bucket. Not sure who put it there."

Everyone asks the same thing. What is in it?

For a webhook server, the frightening answer would be "every credential we use to call our partners." Sparrow stores webhook signing secrets and secret header values, such as `Authorization: Bearer ...` tokens, because it needs them to deliver. If those sat in the dump as plain text, whoever found the file could impersonate you to every system you integrate with.

So let's open the dump and look.

## What a stranger would find

Find a webhook row and look at its `secret_headers` column. There is no token there. There is an opaque run of bytes laid out like this:

```text
[version] [key-id length] [key ID] [wrapped-DEK length] [wrapped DEK] [data nonce] [ciphertext + tag]
```

That is an *envelope*. It carries everything needed to read the secret, except the one thing that matters: the key. That key was never in the database, so it is not in the dump.

## Two locks, kept apart

Envelope encryption uses two layers of keys:

- A **data encryption key (DEK)** encrypts one value. Every stored secret gets its own fresh, random DEK.
- A **key encryption key (KEK)** encrypts, or *wraps*, the DEK.

The wrapped DEK travels inside the envelope. The KEK stays home. Sparrow reads its KEKs from a small keyring supplied at startup through `SPARROW_ENCRYPTION_KEYS`, which you keep in a secret manager, a Kubernetes Secret, or something equivalent:

```bash
export SPARROW_ENCRYPTION_KEYS="old=$(openssl rand -hex 32),new=$(openssl rand -hex 32)"
export SPARROW_ENCRYPTION_PRIMARY_KEY_ID=new
```

Each key is 32 bytes, hex-encoded. The stranger with the dump has a pile of locked boxes and the locked keys to those boxes, but not the master key. Without the KEK the wrapped DEK is useless, and without the DEK the ciphertext is useless.

## How one secret gets sealed

When you register a webhook with a secret header, Sparrow seals it before the row is written. Each `EnvelopeEncrypt` call:

1. generates a random 256-bit DEK;
2. generates a random nonce and encrypts the value with the DEK using AES-256-GCM;
3. generates a second nonce and wraps the DEK with the primary KEK;
4. writes the version, key ID, wrapped DEK, nonce, and ciphertext with its authentication tag into one envelope.

The plain text exists only in memory, and only for as long as it takes to seal it. At the other end, delivery opens the envelope only at the moment a request needs to be signed or authenticated. Secret header values are never returned by the API, and webhook secrets come back masked.

## The seal that tells the truth

Suppose the stranger is more ambitious. They edit a few bytes in an envelope, put the dump back, and hope Sparrow will decrypt it into a value they chose.

It will not. AES-256-GCM is authenticated encryption, and the authentication tag is the modern wax seal. It is computed from the contents, and Sparrow checks it before returning any plain text. Change one byte of the ciphertext, the wrapped DEK, or a nonce, and decryption fails. The same check catches the wrong key and a corrupted record. Sparrow's test suite flips a single ciphertext byte and confirms that decryption refuses.

What happens next matters just as much. When decryption fails, delivery fails closed. Sparrow records the failure and does not quietly send the request unsigned or without its auth header. A request that looks like it worked but was quietly downgraded is a kind of lie, and a visible error is always better than a comfortable lie.

## Changing the locks

On Monday your team decides to rotate the KEK anyway. The dump never contained it, but changing the locks after a scare is a reasonable instinct.

The keyring makes this a configuration change rather than an outage. Add a new key, make it primary, and leave the old one in the list. New secrets are wrapped with the new key. Each existing envelope records which key ID wrapped it, so Sparrow can still open it with the old key. Once every stored secret has been rewritten under the new key, remove the old one.

And if no keyring is configured at all, Sparrow refuses to encrypt or decrypt. It never falls back to storing a secret in plain text just to keep going.

## What the envelope cannot do

Every protection has edges, and it is more honest to name them. Envelope encryption protects data at rest: backups, replicas, exports, snapshots, and anyone reading rows they should not be able to use. It does not protect a running Sparrow process that holds the KEK, because that process has to be able to read the letter in order to deliver it. That is why the keyring belongs in a real secret store, and why encryption is one layer among several: API authentication, response masking, URL redaction, TLS, and signed outbound requests.

It also covers credentials, not events. Event payloads are stored as plain text, so if they are sensitive, pair a retention window (`SPARROW_EVENT_RETENTION_DAYS`) with storage-level encryption.

## Back to Friday

So, what was in the dump? Event payloads and configuration, which still deserve a proper incident review. And a stack of sealed credential envelopes, useless without a key that was never there.

The courier was not trustworthy, as it turned out. The seal held anyway. That is the whole idea: build things that stay safe even when people make ordinary mistakes, because people always will.
