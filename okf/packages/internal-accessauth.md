---
type: Go Package
title: internal/accessauth
description: Sparrow adapter for pkg/access — fixes realm, scope, prefixes, and lifetime rules
tags: [access, tokens, auth, adapter]
timestamp: 2026-09-27T00:00:00Z
---

# internal/accessauth

Sparrow's adapter for `pkg/access`. Everything generic (issuing, hashing, redeeming, caching) lives in `pkg/access`; this package fixes the Sparrow-specific constants.

## Configuration

- **Realm**: `tenant.DefaultTenantID` (the single default tenant).
- **Scope**: a consumer name (`nil` = tenant-wide / full access).
- **Secret prefixes**: `sparrow_tk_` (tokens), `sparrow_inv_` (invites).
- **Root key**: `SPARROW_API_KEY` is registered as a root key named `"master key"` when set.

## Lifetime Rules

| Credential | Default TTL | Maximum TTL |
|------------|-------------|-------------|
| Tenant-wide token | `SPARROW_TOKEN_DEFAULT_TTL` (90 days; 0 = never) | No limit; `never_expires` for none |
| Consumer token | 7 days | 30 days |
| Invite | 24 hours | 7 days |

`TokenTTL(consumer, requested, neverExpires, tenantDefault)` and `InviteTTL(requested)` apply these rules.

## Constructor

`New(db, apiKey)` builds an `access.Service` over Postgres (`pgstore`). `NewWithStore(store, apiKey)` accepts any `access.Store` (tests use `memstore`).

## Tests

- `TestMigrationMatchesPgstoreSchema` — enforces that `db/migrations/000027_access_tokens.up.sql` contains `pgstore.Schema` verbatim.
- `TestTTLRules` — validates lifetime defaults and limits.
- `TestMasterKeyIsRootInDefaultTenant` — verifies the master key authenticates as a full-access root principal.
- `TestPgstoreConformance` — runs `storetest.Run` against real Postgres in a throwaway schema.

## Citations

- `internal/accessauth/accessauth.go`
- `internal/accessauth/accessauth_test.go`
