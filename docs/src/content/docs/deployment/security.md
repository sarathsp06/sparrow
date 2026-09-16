---
title: Securing Sparrow
description: Sparrow's security model, the trust assumptions of the embedded dashboard, and how to add SSO (username/password, Microsoft Entra, Google) with an identity-aware proxy.
---

Sparrow's built-in authentication is deliberately minimal: one shared secret
(`SPARROW_API_KEY`) checked on every API request — optional in development,
required when `ENVIRONMENT=production` (the server refuses to start without
it). Everything beyond that — per-user logins, SSO, MFA, audit of *who* did
what — is a deployment concern, solved by putting an identity-aware proxy in
front of Sparrow rather than by building an identity provider into it.

This page covers what Sparrow does natively, what it assumes about your
network, and the recommended way to add real user authentication. For the
implementation details -- SSRF protection, HTTP hardening headers, secret
masking, tenant isolation -- see the [Security
Model](/sparrow/reference/security/) reference page.

## What Sparrow provides natively

| Control | Mechanism |
|---|---|
| API authentication | Shared secret: `SPARROW_API_KEY` → `X-API-Key` header, constant-time compare; mandatory in production |
| Secrets at rest | Envelope encryption (AES-256-GCM, per-record DEK) keyed by `SPARROW_ENCRYPTION_KEY` |
| Outbound signing | Standard Webhooks signatures (`v1` HMAC-SHA256, `v1a` Ed25519) on every delivery |
| SSRF protection | Private/loopback/link-local/metadata IPs blocked at dial time; redirects re-validated |
| Browser access | `CORS_ALLOWED_ORIGINS` allowlist; security headers on all responses |

What it does **not** provide:

- **User accounts.** There is one key. Everyone who has it is equally trusted.
- **Tenant isolation via consumers.** Consumers organize resources; they are
  *not* a security boundary. Any valid API key can read and write every consumer.
- **Rate limiting on the API.** Put a reverse proxy in front if you need it.

## Data retention and compliance posture

Sparrow stores event payloads and delivery request bodies as **plaintext in
PostgreSQL** — they must stay queryable and transformable. Secrets and
signing keys are envelope-encrypted; payloads are not. If payloads carry
regulated data (PHI, PII), plan for it:

- **Encrypt at the storage layer.** Encryption-at-rest requirements
  (HIPAA-style) are satisfied below the application: encrypted
  disks/volumes, or the managed encryption RDS/Cloud SQL enable by default,
  plus TLS to PostgreSQL.
- **Bound how long data lives.** Set `SPARROW_EVENT_RETENTION_DAYS` to purge
  events — and, via cascade, their deliveries — older than N days. Runs
  hourly in the background. Unset (the default), data is kept forever.
- **Keep response capture off.** `capture_response_body` is off by default;
  leave it off if endpoints may echo sensitive data back.
- **Self-hosting is the compliance boundary.** Payloads never leave your
  infrastructure — there is no vendor to sign a BAA with.

## Trust model of the embedded dashboard

The embedded web UI (`SPARROW_SERVE_UI=true`) is designed for **trusted,
private networks** — a team dashboard on a VPN or internal network, not a
public-facing app:

- The dashboard and its assets are served **without authentication** so the
  login-free UI works out of the box.
- The dashboard needs the API key to call the API, and the server currently
  injects it into the served HTML (`window.__SPARROW_CONFIG__`). **Anyone who
  can load the dashboard can read the API key.** Enabling the UI on a network
  segment is equivalent to sharing the API key with that segment.

Consequences:

- On a private/VPN network where everyone with network access is trusted:
  fine as-is.
- On a shared or internet-facing network: **do not expose the port directly.**
  Put an authenticating proxy in front (next section) or disable the UI.

## Recommended: SSO via an identity-aware proxy

For per-user login — username/password, Microsoft Entra ID, Google, or any
OIDC/SAML provider — run an open-source auth overlay in front of Sparrow. The
proxy authenticates humans, then injects the `X-API-Key` header on every
authenticated request. Sparrow needs zero configuration changes beyond setting
the key, and the browser never sees the key at all.

```text
 browser ──▶ auth proxy (login: password / Entra / Google)
                 │  injects X-API-Key: <SPARROW_API_KEY>
                 ▼
             Sparrow :8080  (not directly reachable)

 CI / services ──▶ Sparrow (internal route, X-API-Key directly)
```

Rules for this topology:

1. Sparrow's port must be reachable **only** from the proxy (and trusted
   machine clients). If clients can bypass the proxy, the overlay is decoration.
2. Set `SPARROW_API_KEY`; give it only to the proxy config and machine clients.
3. Set `CORS_ALLOWED_ORIGINS` to the proxy's public origin (or leave unset if
   the UI is served through the same origin).

### Option A — Authentik (username/password + Entra + Google in one tool)

[Authentik](https://goauthentik.io) is a self-hosted identity provider with a
built-in **Proxy Provider** mode, so it is both the IdP and the overlay:

- Local user database (passwords, passkeys, MFA) **and** federated sources
  (Microsoft Entra ID, Google, generic OIDC/SAML) side by side.
- Proxy Provider runs as an embedded outpost or standalone container; speaks
  forward-auth natively with Traefik, nginx, and Caddy.
- In the Proxy Provider settings, add a custom header injection so every
  proxied request carries `X-API-Key: <SPARROW_API_KEY>`.

Deployment: two containers (server + worker) plus PostgreSQL — you already run
PostgreSQL for Sparrow. Verify that the external OAuth/SAML source types you
need are available in the open-source tier of your Authentik version.

### Option B — oauth2-proxy (single IdP, smallest footprint)

If **all** users live in one IdP (an Entra tenant or a Google Workspace
domain), [oauth2-proxy](https://github.com/oauth2-proxy/oauth2-proxy) is a
single Go binary that does the whole job:

```text
oauth2-proxy \
  --provider=oidc \
  --oidc-issuer-url=https://login.microsoftonline.com/<tenant>/v2.0 \
  --upstream=http://sparrow:8080 \
  --email-domain=yourcompany.com
```

Inject the API key upstream via its header configuration (e.g. an
`injectRequestHeaders` entry in the alpha config, or terminate at nginx and add
`proxy_set_header X-API-Key ...`). No local username/password support — that's
the tradeoff for the small footprint.

### Option C — Keycloak + oauth2-proxy (maximum boring)

[Keycloak](https://www.keycloak.org) if you want the battle-tested enterprise
IdP: local users, identity brokering (Entra, Google, SAML), fine-grained roles
— all free. Keycloak is only the IdP; pair it with oauth2-proxy (pointing at
Keycloak as its OIDC issuer) as the actual overlay. Two moving parts instead
of one, but every part is thoroughly documented and widely deployed.

### Not a fit

- **Authelia** — local users only; it [cannot consume external OIDC
  providers](https://www.authelia.com/configuration/identity-providers/openid-connect/provider/)
  (no Relying Party role), so no "login with Entra/Google".
- **Pomerium** — excellent identity-aware proxy, but delegates to a single
  upstream IdP and has no built-in username/password store.

## Hardening checklist

Whether or not you add SSO:

- [ ] Set `SPARROW_API_KEY` (generate: `openssl rand -hex 32`) — mandatory
      when `ENVIRONMENT=production`, and without it anyone who can reach the
      port owns the instance regardless of environment.
- [ ] Set `SPARROW_ENCRYPTION_KEY` and store it in a secret manager, never in
      the database or repo.
- [ ] Use TLS to PostgreSQL (`sslmode=require` or stronger) whenever the
      database is not on localhost.
- [ ] Set `CORS_ALLOWED_ORIGINS` explicitly for any browser-based access.
- [ ] Leave `SPARROW_ALLOW_PRIVATE_NETWORKS=false` unless webhook targets are
      genuinely on your LAN — enabling it disables SSRF protection globally.
- [ ] Set `SPARROW_EVENT_RETENTION_DAYS` if payloads carry regulated data or
      you have any retention policy — the default keeps events forever.
- [ ] Terminate TLS and apply rate limiting at a reverse proxy; Sparrow serves
      plain HTTP.
- [ ] Keep `/health`, `/ready`, `/docs`, and `/openapi.*` in mind: they are
      intentionally unauthenticated. Restrict at the proxy if the OpenAPI spec
      is sensitive in your environment.
