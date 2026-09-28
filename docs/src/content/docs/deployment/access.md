---
title: "Access: Tokens and Invites"
description: Give people and machines their own revocable credentials instead of sharing the master key.
---

Sparrow's master key (`SPARROW_API_KEY`) works like a root password: powerful,
but sharing it means you can never revoke one holder without rotating it for
everyone. **Access tokens** solve this — each person or CI job gets a named,
individually revocable credential. **Invites** let you hand out tokens without
pasting secrets into chat.

## First run

When `SPARROW_API_KEY` is set (always the case with the production Docker
Compose), the embedded UI shows a **Sign in to Sparrow** prompt on the first
`401`. You can paste the master key there -- the UI exchanges it for a browser
token behind the scenes, so the master key is never stored in the browser. An
access token or invite link also works.

## Invite a teammate

### From the web UI

1. Open the **Access** page in the sidebar.
2. Click **Invite**.
3. Enter a name (e.g. "alice"), choose **Full access** or a single consumer's
   portal, pick a link expiry (1 hour, 24 hours, or 7 days), and optionally
   set how long their access lasts.
4. Copy the link and send it to them.

When they open the link, the UI redeems the invite and signs them in
automatically. The invite works once; expired, cancelled, or already-used
invites show a clear message.

### From the CLI

```bash
sparrow invite alice
sparrow invite alice --ttl 15m --ui-url https://sparrow.example.com
sparrow invite "acme support" --consumer acme --ttl 7d
```

The printed link defaults to the server URL. Pass `--ui-url` when the UI is
[hosted separately](/sparrow/deployment/separate-ui/).

### From the API

```bash
curl -X POST http://localhost:8080/v1/invites \
  -H "X-API-Key: $SPARROW_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"name": "alice", "ttl_seconds": 900}'
```

The response includes a `path` (`/#invite=<secret>` for the console,
`/portal#invite=<secret>` for a consumer invite). Prepend the UI's base URL.

## Remove access

Revoke a token from the **Access** page (click **Revoke**), the CLI, or the
API:

```bash
sparrow tokens revoke <token-id>
```

The token stops working immediately on the revoking instance. Other instances
stop accepting it within 30 seconds (the cache TTL).

## CI and machine tokens

Create a token directly — no invite link needed:

```bash
sparrow tokens create --name ci-deploy
sparrow tokens create --name acme-sync --consumer acme --ttl 30d
```

The secret is printed once. Store it in your CI secret store and use it as
`X-API-Key` or `Authorization: Bearer`.

Tenant-wide tokens (no `--consumer`) expire after the server's default —
90 days unless `SPARROW_TOKEN_DEFAULT_TTL` changes it — unless `--ttl` is
given. For a CI credential you rotate by hand, `--ttl never` creates one that
does not expire; revoke it when it is retired. Consumer tokens default to 7
days, allow at most 30, and always expire.

## Consumer (portal) access

Two ways to give consumers access to the portal. Both produce a
consumer-scoped access token: listed in `sparrow tokens list`, revocable on its
own, and limited to that consumer's portal.

### Invites (for named people)

```bash
sparrow invite "acme support" --consumer acme
```

The link works once and opens the consumer's portal with a token named after
the invitee.

### Portal links (for embedding)

```bash
curl -X POST http://localhost:8080/v1/tokens \
  -H "X-API-Key: $SPARROW_API_KEY" -H "Content-Type: application/json" \
  -d '{"name": "portal", "consumer": "acme", "ttl_seconds": 3600, "external_id": "user-42"}'
```

Every consumer token comes with a ready-made `portal_path`. Your backend mints
one for a user who is already signed in to your product and hands them that
link; revoke it early with `DELETE /v1/tokens/{id}` (for example on logout).
Use a short `ttl_seconds` for embedding.

`external_id` (consumer tokens only) is your id for who the token is for, for
example your user id. There is at most one active token per consumer and
`external_id`, so the call is safe to repeat on every page view: while that
token is valid, the same token and link come back (`"reused": true`; `name`
and `ttl_seconds` are ignored); once it has expired or been revoked, a new one
is created. The secret of such a token is kept envelope-encrypted with
`SPARROW_ENCRYPTION_KEYS` so it can be returned again; tokens without an
`external_id` store only a hash. The `external_id` itself is not a secret.

Expired and revoked tokens stay listed for 7 days, then a daily job deletes
them, so frequently minted portal links do not grow the tokens table without
bound.

## What happens on master-key rotation

Rotating `SPARROW_API_KEY` invalidates the old master key but **does not
invalidate existing tokens or pending invites**. Tokens are verified by their
stored hash, not by a relationship to the master key.

## Lifetimes

| Credential | Default | Maximum |
|---|---|---|
| Tenant-wide token | `SPARROW_TOKEN_DEFAULT_TTL` (90 days) | No limit, or never with `--ttl never` / `never_expires` |
| Consumer token | 7 days | 30 days |
| Invite (link expiry) | 24 hours | 7 days |

Durations accept Go syntax plus a `d` suffix: `90d`, `12h`, `15m`.
Set `SPARROW_TOKEN_DEFAULT_TTL=0` to restore the previous behaviour, where
tenant-wide tokens never expire unless a TTL is given. Browser sign-ins (a
pasted master key or an invite) also get the default lifetime, so people sign
in again after it lapses.

## API endpoints

All under `/v1` (require the master key or a tenant-wide token):

| Endpoint | Method | Description |
|---|---|---|
| `/v1/whoami` | GET | Show which credential this request used |
| `/v1/tokens` | POST | Create a token (secret returned once) |
| `/v1/tokens` | GET | List tokens (`?consumer=`, `?include_inactive=`) |
| `/v1/tokens/{token_id}` | DELETE | Revoke a token (idempotent) |
| `/v1/invites` | POST | Create an invite |
| `/v1/invites` | GET | List invites (`?include_inactive=`) |
| `/v1/invites/{invite_id}` | DELETE | Cancel a pending invite (409 if not pending) |

Outside `/v1`, no auth required (the invite is the credential):

| Endpoint | Method | Description |
|---|---|---|
| `/invite/redeem` | POST | Redeem an invite (`{"invite": "..."}` -> token) |

## Security notes

- **Possession-based identity.** Tokens prove you have the secret, not who you
  are. There are no accounts, passwords, or MFA. A leaked token is valid until
  revoked.
- **SHA-256 hashing.** Only the hash is stored; the plaintext secret is shown
  once at creation. Prefixes (`sparrow_tk_`, `sparrow_inv_`) make leaked
  secrets easy to spot in logs and secret scanners.
- **503, not 401, on DB outage.** If the token store is unreachable, the
  server returns `503 Service Unavailable` with `Retry-After`, not `401`.
  Browsers keep their credential and retry. The master key still works without
  the database.
- **30-second revocation window.** Successful token lookups are cached for 30
  seconds. Revocation is immediate on the revoking instance; other instances
  honour it within 30 seconds.
- **XSS risk.** The browser token lives in `localStorage`. An XSS attack on
  the Sparrow UI domain could exfiltrate it. Set a tight CSP and keep the UI
  on a dedicated origin.
- **No inbound rate limiting.** Brute-forcing token secrets is bounded by
  SHA-256 cost and network round-trip, not by explicit throttling. Rate-limit
  at the proxy if the API is internet-facing.
