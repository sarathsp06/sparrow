---
type: Go Package
title: internal/middleware
description: API key authentication and security headers middleware for the REST API
tags: [middleware, auth, security]
timestamp: 2026-08-29T00:00:00Z
---

# internal/middleware

Provides optional API key authentication and security headers for the HTTP server.

## APIKeyAuth

```go
type APIKeyAuth struct {
    APIKey               string
    ExcludedPathPrefixes []string
}
```

When `SPARROW_API_KEY` is set, every `/v1/*` request must include the key via the `X-API-Key` header.

HTTP query-parameter keys are intentionally not accepted; URLs are commonly logged by proxies, stored in browser history, and leaked through referrers. Uses `crypto/subtle.ConstantTimeCompare` to prevent timing attacks. Excluded paths: `/health`, `/ready`, `/docs`, `/openapi`, UI catch-all.

## PortalGateway

Serves the consumer portal API under one static public prefix, `/portal/api/`.
It verifies the consumer-scoped bearer token (`PortalTokens.Verify`), derives
the consumer from the token rather than the URL, maps `/portal/api/<rest>` to
the real `/v1` route (`portalTarget`), marks the request pre-authorized, and
re-dispatches into the main router so the existing handlers run unchanged.

Because the consumer rides in the token, an operator exposing the portal
allowlists a single API prefix with no per-consumer or deny rules, and
cross-consumer access is structurally impossible. `portalTarget` refuses event
injection (`POST events`) and token minting (`portal-token`), and allows the
read-only global helpers the portal UI needs (event-type catalog, template
functions, template dry-run). `APIKeyAuth.HTTPMiddleware` honors the
`PortalAuthorized(ctx)` flag the gateway sets, so portal traffic reuses the
`/v1` handlers without the admin key.

## SecurityHeaders

Sets standard security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy` with restricted defaults

## Citations

- `internal/middleware/apikey.go`
- `internal/middleware/portal.go`
- `internal/middleware/portal_gateway.go`
- `internal/middleware/portal_test.go`
- `internal/middleware/apikey_test.go`
- `internal/middleware/security_headers.go`
