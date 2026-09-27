---
type: Go Package
title: internal/middleware
description: Authentication (master key + access tokens), CORS, portal gateway, and security headers middleware for the REST API
tags: [middleware, auth, cors, security, tokens]
timestamp: 2026-09-27T00:00:00Z
---

# internal/middleware

Provides authentication (master key + access tokens), CORS, portal gateway, and security headers for the HTTP server.

## Auth

```go
type Auth struct {
    Enabled              bool
    Realm                string
    Authn                *httpauth.Authenticator
    ExcludedPathPrefixes []string
}
```

Replaces the former `APIKeyAuth` (`apikey.go` is deleted). When `SPARROW_API_KEY` is set, every `/v1/*` request must include either the master key or a tenant-wide access token via `X-API-Key` or `Authorization: Bearer`. Consumer-scoped tokens are refused with 403 ("consumer-scoped tokens can only call the portal API under /portal/api/"). When auth is disabled, requests get an anonymous principal with root access.

HTTP query-parameter keys are intentionally not accepted; URLs are commonly logged by proxies, stored in browser history, and leaked through referrers. Excluded paths: `/health`, `/ready`, `/docs`, `/openapi`, UI catch-all.

## PortalVerifier

`NewPortalVerifier(pt, svc, realm)` builds a verifier that accepts both stateless portal tokens (`spt_v2`, minted by `POST /v1/consumers/{c}/portal-token`) and consumer-scoped access tokens. Full-access tokens are rejected with `ErrPortalFullAccessToken` (they should use `/v1` instead). The portal gateway uses this to resolve a bearer credential to a consumer name.

## PortalGateway

Serves the consumer portal API under one static public prefix, `/portal/api/`.
It verifies the consumer-scoped bearer token (stateless portal token or consumer access token via `PortalVerifier`), derives
the consumer from the token rather than the URL, maps `/portal/api/<rest>` to
the real `/v1` route (`portalTarget`), marks the request pre-authorized, and
re-dispatches into the main router so the existing handlers run unchanged.

Because the consumer rides in the token, an operator exposing the portal
allowlists a single API prefix with no per-consumer or deny rules, and
cross-consumer access is structurally impossible. `portalTarget` refuses event
injection (`POST events`) and token minting (`portal-token`), and allows the
read-only global helpers the portal UI needs (event-type catalog, template
functions, template dry-run). `Auth.HTTPMiddleware` honors the
`PortalAuthorized(ctx)` flag the gateway sets, so portal traffic reuses the
`/v1` handlers without the admin key.

## CORS

`CORS(origins []string, production bool) (func(http.Handler) http.Handler, CORSMode)` builds the cross-origin policy:

- **Origins set**: allow-list mode; only the listed origins may call the API. Trailing slashes are normalized away (`NormalizeOrigins`). Allowed headers: `Authorization`, `Content-Type`, `X-API-Key`. Credentials are not allowed (the UI uses API-key or bearer headers, not cookies).
- **Origins empty, production**: block all cross-origin requests (the embedded UI is same-origin).
- **Origins empty, non-production**: allow all origins for local development (e.g. Vite on :5173 talking to :8080).

Returns a `CORSMode` (`CORSAllowList`, `CORSBlockAll`, `CORSAllowAll`) for startup logging.

## SecurityHeaders

Sets standard security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY` (except `/portal` paths, which allow framing so operators can embed the portal in an iframe)
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy: interest-cohort=()` (opts out of FLoC/Topics)
- `Content-Security-Policy`: `default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data:; connect-src 'self'`. Portal paths get `frame-ancestors *`; all other paths get `frame-ancestors 'none'`.

## Citations

- `internal/middleware/auth.go`
- `internal/middleware/portal.go`
- `internal/middleware/portal_gateway.go`
- `internal/middleware/portal_test.go`
- `internal/middleware/auth_test.go`
- `internal/middleware/security_headers.go`
- `internal/middleware/cors.go`
- `internal/middleware/cors_test.go`
