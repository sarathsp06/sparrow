---
type: Go Package
title: internal/middleware
description: API key authentication and security headers middleware for HTTP and gRPC
tags: [middleware, auth, security]
timestamp: 2026-06-22T00:00:00Z
---

# internal/middleware

Provides optional API key authentication and security headers for the HTTP and gRPC servers.

## APIKeyAuth

```go
type APIKeyAuth struct {
    APIKey               string
    ExcludedPathPrefixes []string
}
```

When `SPARROW_API_KEY` is set, every API request must include the key via:
- HTTP: `X-API-Key` header or `api_key` query parameter
- gRPC: `x-api-key` metadata key

Uses `crypto/subtle.ConstantTimeCompare` to prevent timing attacks. Excluded paths: `/health`, `/ready`, UI catch-all.

## SecurityHeaders

Sets standard security headers:
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Permissions-Policy` with restricted defaults

## Citations

- `internal/middleware/api_key.go`, `internal/middleware/security.go`
