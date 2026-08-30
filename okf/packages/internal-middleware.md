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

## SecurityHeaders

Sets standard security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy` with restricted defaults

## Citations

- `internal/middleware/apikey.go`
- `internal/middleware/apikey_test.go`
- `internal/middleware/security_headers.go`
