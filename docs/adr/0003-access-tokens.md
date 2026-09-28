# 3. Replace the single shared API key with revocable access tokens and one-time invites

- Status: accepted
- Date: 2026-09-27

## Context

Sparrow's authentication was a single shared secret (`SPARROW_API_KEY`):

- **One key, many holders.** Every operator, CI job, and script shares the same
  credential. Rotating it invalidates everyone at once. Off-boarding someone
  means rotating the key and re-distributing it to everyone who still needs it.
- **Key injected into the embedded UI.** With `SPARROW_SERVE_UI=true`, the
  server writes `SPARROW_API_KEY` into `window.__SPARROW_CONFIG__` so the
  dashboard works without a sign-in step. Anyone who can load the page can read
  the key — acceptable on a VPN, problematic on a shared network.
- **Portal tokens not individually revocable.** Stateless `spt_v2` portal
  tokens (HMAC-SHA256, consumer-scoped, 7-day default) can only be
  "revoked" by rotating the signing key, which invalidates all portal tokens
  at once.
- **Access links (unreleased) had limits.** The earlier "access link" prototype
  exchanged a one-time link for the raw API key. The browser then held the
  master key in `localStorage`, so revoking meant rotating
  `SPARROW_API_KEY`. Access links were never released.

The VPN-first deployment model means full-blown user accounts and OIDC are
out of scope — Sparrow is an infrastructure service, not a multi-user app —
but operators need a way to give individual people and machines their own
credentials without sharing the master key.

## Decision

### 1. Reusable library: `pkg/access`

A separate Go module (`pkg/access`, own `go.mod`, stdlib-only — a test enforces
it imports nothing from Sparrow) that any application can use. Concepts:

- **Realm** — an isolation boundary (Sparrow maps it to the default tenant).
- **Scope** — `nil` = full access in the realm; non-nil = application-defined
  restriction (Sparrow maps it to a consumer name).
- **Root keys** — static secrets from configuration (the master key). They
  authenticate as full-access principals and need no storage.
- **Tokens** — stored, named, optional expiry, individually revocable. Only a
  SHA-256 hash of the secret is stored.
- **Invites** — stored, single-use, expiring. Redeeming one creates a token.

Subpackages: `memstore` (in-memory, tests), `pgstore` (Postgres via
`database/sql`, ships `pgstore.Schema`), `storetest` (conformance suite every
Store implementation must pass), `httpauth` (net/http adapter:
`Credential()` reads `Authorization: Bearer` or `X-API-Key`, never query
params; `Authenticator` + `Middleware`; `WriteError` with structured 401
reasons; `RedeemHandler`).

The `Service` caches successful token lookups for 30 seconds and throttles
`last_used_at` writes to once per minute.

### 2. Sparrow adapter: `internal/accessauth`

Fixes the Sparrow-specific constants:

- Realm = default tenant ID.
- Scope = consumer name.
- Secret prefixes: `sparrow_tk_` (tokens), `sparrow_inv_` (invites).
- `SPARROW_API_KEY` registers as the root key named "master key".
- Lifetime rules: tenant-wide tokens never expire unless a TTL is given;
  consumer tokens default 7 days (max 30); invites default 24 hours (max
  7 days).

### 3. Two scopes only — no roles

A token is either full-access (tenant-wide, same power as `SPARROW_API_KEY`)
or pinned to one consumer (portal API only). There is no role-based access
control, no fine-grained permission model, and no allow/deny on individual
endpoints beyond the portal's existing allow-list. This keeps the model
simple enough that operators can reason about it without a matrix.

### 4. Stored invites instead of signed links

Invites are stored rows with a hashed secret, not stateless signed tokens
(the earlier "access link" approach). Consequences:

- An invite can be cancelled before it is used.
- Invite status is visible in `GET /v1/invites`.
- Invite validity does not depend on the master key: rotating
  `SPARROW_API_KEY` does not invalidate pending invites or existing tokens.
- One extra row per invite (tens, not millions — acceptable).

### 5. No cascade revocation

Revoking a token revokes that one token. The token that created it (or the
master key that created it) is unaffected, and tokens created by the revoked
token are also unaffected. There is no parent-child tree. This is simple and
predictable; operators who want "revoke everything Alice created" can list and
revoke individually.

### 6. Tokens independent of master key

Rotating `SPARROW_API_KEY` does not invalidate tokens. Tokens are verified
by their stored hash, not by a relationship to the master key. This lets an
operator rotate the master key without disrupting existing token holders.

### 7. 503, not 401, when the database is down

If the token store is unreachable, the server returns `503 Service Unavailable`
with `Retry-After: 5`, not `401`. Browsers keep their stored credential and
retry. The master key (a root key) still works without the database — it is
checked in memory.

### 8. `SPARROW_UI_INJECT_KEY` (default `true`)

A new config variable. When `true` (today's behaviour), the embedded UI gets
`SPARROW_API_KEY` injected so it works without signing in. When `false`, the
UI shows a sign-in prompt instead — the operator can paste the master key
(which the UI exchanges for a browser token, so the master key is not stored)
or open an invite link.

Default is `true` to avoid breaking existing deployments. Operators on shared
networks should set it to `false`.

## Consequences

- **Per-person, per-machine credentials.** Each operator and CI job gets its
  own named token. Off-boarding = revoke one token; no key rotation needed.
- **Invite flow replaces key-sharing.** `sparrow invite alice` prints a link;
  Alice opens it and gets her own token without seeing the master key.
- **Consumer tokens for the portal.** A consumer invite creates a token scoped
  to that consumer's portal — revocable, unlike stateless portal tokens.
  (Superseded, see "Update: portal links are access tokens" below.)
- **30-second revocation window on other instances.** The 30-second cache TTL
  means a revoked token may still work on instances that cached it. Immediate
  on the revoking instance (cache is cleared synchronously).
- **Identity is possession-based.** A token proves you have the secret, not
  who you are. There are no accounts, no passwords, no MFA. This is the same
  model as `SPARROW_API_KEY` but per-credential.
- **`pkg/access` is a separate Go module** with its own release tag
  (`pkg/access/vX.Y.Z`), managed by `scripts/release-submodules.sh`.
- **Migration 000027** creates `access_tokens` and `access_invites` tables;
  a test enforces the migration matches `pgstore.Schema` exactly.

## Alternatives considered

### Docs-only / auth proxy

Tell operators to use an identity-aware proxy (Authentik, oauth2-proxy) and
inject `X-API-Key` there. This works and is already documented, but does not
help with CI tokens, CLI access, off-boarding granularity, or the embedded
dashboard key injection. Tokens and invites solve those without requiring proxy
infrastructure.

### Tokens without invites

Issue tokens via the API and tell people to paste them. Works for CI; awkward
for people. Invites wrap the exchange in a link and avoid key-in-chat.

### Signed (stateless) invites

A signed token with an expiry, like the portal tokens. Simpler (no storage),
but cannot be cancelled, cannot be listed, and ties validity to the signing
key. Stored invites are more operational: `sparrow invites list`, `sparrow
invites cancel`.

### Built-in OIDC / user accounts

Full identity provider inside Sparrow. Far too heavy for a self-hosted webhook
relay. The auth-proxy path covers this when needed.

### Trusted proxy user header

Accept `X-Forwarded-User` from a trusted proxy and use it as the principal
name. Not implemented now but compatible with the current model — could be
added later as another `httpauth.Verifier` without changing tokens or invites.

### Moving portal embedding to stored tokens

Replace stateless `spt_v2` portal tokens with stored consumer tokens for
iframe embedding. Rejected: each portal page view would create or look up a
stored row, which is wasteful at scale. Stateless portal tokens remain the
right fit for embedding. (Later adopted, see "Update: portal links are access
tokens" below.)

## Known limits

- **Possession-based identity.** No user accounts, no MFA. A leaked token is
  a valid token until revoked.
- **No inbound API rate limiting.** Brute-forcing token secrets is bounded
  only by SHA-256 lookup cost and network round-trip, not by explicit
  throttling. Put rate limiting at the proxy if the API is internet-facing.
- **30-second revocation window** on instances that cached the token.
- **XSS can read `localStorage`.** The browser token is stored in
  `localStorage`. A successful XSS attack on the Sparrow UI domain can
  exfiltrate it — same risk as the master key injection, but scoped to one
  user's token instead of the master key.

## Amendment (2026-09-27)

`SPARROW_UI_INJECT_KEY` has been removed. The server no longer writes
`SPARROW_API_KEY` into the embedded UI's pages under any configuration.
When `SPARROW_API_KEY` is set, the embedded UI always shows the sign-in
prompt (a pasted master key is exchanged for a browser token). Setting
`SPARROW_UI_INJECT_KEY` in the environment now has no effect; the server
logs a deprecation warning if it is still present. Section 8 above
describes the original design; this amendment supersedes it.

## Update: portal links are access tokens (2026-09-28)

Portal links are now consumer-scoped access tokens from `POST /v1/tokens`
(which returns a `portal_path` for them); the separate
`POST /v1/consumers/{consumer}/portal-token` endpoint and stateless `spt_v2`
tokens are gone (nothing had shipped to production). One endpoint means one
set of lifetime, validation and idempotency rules. A security review found that `spt_v2` tokens, signed with the
data-encryption key and valid for up to 30 days, could only be revoked by
removing that key, which also breaks decryption.

The scale concern that rejected stored tokens above is handled by:

- `idempotency_key` (consumer tokens only): while a token minted for a consumer with the same key is
  valid, the call returns it (secret included) instead of storing a new row.
  To make that possible, such a token's secret is stored envelope-encrypted
  (`access.SecretSealer`, backed by the `SPARROW_ENCRYPTION_KEYS` keyring) and
  deleted on revocation. Tokens without a key still store only a hash.
- A daily purge (River job) deletes tokens expired or revoked more than 7 days
  ago.
- Short lifetimes chosen by the embedding backend (`ttl_seconds`).

Lookups hit the database at most once per token per 30 seconds (the existing
cache), which the portal's traffic easily absorbs.
