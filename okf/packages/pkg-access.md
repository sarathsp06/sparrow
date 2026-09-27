---
type: Go Package
title: pkg/access
description: Reusable token/invite library — separate Go module (stdlib-only), issues and checks revocable access tokens and one-time invites
tags: [access, tokens, invites, auth, security]
timestamp: 2026-09-27T00:00:00Z
---

# pkg/access

A separate Go module (`pkg/access/go.mod`, stdlib-only — a test enforces it imports nothing from Sparrow) for issuing and checking revocable access tokens and one-time invites. Reusable by any application.

## Concepts

- **Realm**: isolation boundary (a tenant, an organization). Every token and invite belongs to exactly one realm.
- **Scope**: optional application-defined restriction. `nil` = full access within the realm; non-nil = opaque to this package.
- **Root key**: static secret from configuration. Authenticates as a full-access principal, needs no storage.
- **Token**: stored, named, optional expiry, individually revocable. Only a SHA-256 hash is stored.
- **Invite**: stored, single-use, expiring. Redeeming creates a token; the invite is then spent.

## Key Types

- `Service` — issues and checks tokens/invites. Caches successful lookups (default 30s), throttles `last_used_at` writes (default 1 min).
- `Store` — interface for token/invite persistence (`CreateToken`, `GetTokenByHash`, `ListTokens`, `RevokeToken`, `TouchToken`, `CreateInvite`, `GetInviteByHash`, `ListInvites`, `CancelInvite`, `RedeemInvite`).
- `Principal` — who a request authenticated as (realm, scope, name, token ID, root flag). `FullAccess()` reports unrestricted access.
- `Token`, `Invite` — stored records (never contain the secret).
- `AuthError` — carries a `Reason` (`missing`, `invalid`, `expired`, `revoked`).
- `RootKey`, `Config` — service configuration.

## Subpackages

| Package | Purpose |
|---------|---------|
| `memstore` | In-memory Store (tests) |
| `pgstore` | Postgres Store via `database/sql`; ships `pgstore.Schema` (the DDL for `access_tokens` + `access_invites`) |
| `storetest` | Conformance test suite every Store must pass |
| `httpauth` | net/http adapter: `Credential()` reads `Authorization: Bearer` or `X-API-Key` (never query params); `Authenticator` + `Middleware`; `WriteError` (401 with `reason`, 503 + `Retry-After` on store outage); `RedeemHandler` |

## Citations

- `pkg/access/*.go`
- `pkg/access/httpauth/`
- `pkg/access/memstore/`
- `pkg/access/pgstore/`
- `pkg/access/storetest/`
