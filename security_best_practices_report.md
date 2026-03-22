# Sparrow Security Best Practices Report

**Project**: Sparrow (Multi-Tenant Webhook Delivery Platform)  
**Date**: 2026-03-22  
**Scope**: Go backend (`internal/`, `cmd/`, `pkg/`), SvelteKit frontend (`web/`), deployment config  
**Languages**: Go 1.25, TypeScript/Svelte 5  

---

## Executive Summary

Sparrow demonstrates solid security engineering in many areas: parameterized SQL queries, SHA-256 API key hashing with `crypto/rand`, JWT RS256 algorithm restriction, and a comprehensive SSRF URL validator. However, the audit identified **22 findings** across the backend and frontend, including 2 critical, 5 high, 6 medium, and 9 low-severity issues.

The most impactful findings are:

1. **SSRF bypass via HTTP redirects** -- webhook delivery follows redirects without re-validating against the SSRF blocklist, allowing exfiltration of cloud metadata and internal service responses.
2. **DNS rebinding TOCTOU** -- URL validation occurs at registration time but not at delivery time, allowing DNS record changes to redirect deliveries to internal networks.
3. **Committed credentials** -- `.env.docker` is tracked in git with database passwords; `web/.env` contains a Clerk secret key on disk.
4. **No request body size limits** -- Connect-RPC and HTTP endpoints accept arbitrarily large payloads.
5. **Auth disabled by default** -- combined with wildcard CORS, any unauthenticated deployment is fully open to any origin.

| Severity | Count |
|----------|-------|
| Critical | 2 |
| High | 5 |
| Medium | 6 |
| Low | 9 |
| **Total** | **22** |

---

## Critical Findings

### SEC-001: HTTP Redirect SSRF Bypass

**Rule**: GO-SSRF-001  
**File**: `internal/webhooks/client/client.go:41-45`  
**Impact**: An attacker can exfiltrate cloud metadata (AWS/GCP IAM credentials), internal service data, or map internal networks through a registered webhook.

The `http.Client` is created without a `CheckRedirect` function. Go's default behavior follows up to 10 redirects automatically. An attacker registers a webhook URL `https://attacker.com/hook` that responds with `302 Location: http://169.254.169.254/latest/meta-data/iam/security-credentials/`. The HTTP client follows this redirect, fetches the cloud metadata, and stores the response body in the delivery record -- which the attacker can then read via `GetDeliveryAttempts`.

The `FollowRedirects` config field exists in the data model (`internal/webhooks/store/models.go:36`) but is **never wired** into the HTTP client.

**Remediation**:
```go
client := &http.Client{
    CheckRedirect: func(req *http.Request, via []*http.Request) error {
        if !config.FollowRedirects || len(via) >= 3 {
            return http.ErrUseLastResponse
        }
        return validateWebhookURL(req.URL.String()) // re-validate each hop
    },
}
```

---

### SEC-002: DNS Rebinding / TOCTOU on Webhook URLs

**Rule**: GO-SSRF-001  
**Files**: `internal/webhooks/urlvalidation.go:50-61`, `internal/webhooks/client/client.go:35-38`  
**Impact**: Same as SEC-001 -- internal network access and cloud metadata exfiltration.

`ValidateWebhookURL()` performs DNS resolution at registration time, but the actual HTTP delivery happens later (potentially hours later) via a standard `net.Dialer` with no IP re-validation. An attacker can:

1. Register `evil.example.com` pointing to a public IP -- passes validation.
2. Change DNS to point to `169.254.169.254` or `10.0.0.1`.
3. Webhook delivery resolves the new IP and connects to the internal network.

**Remediation**: Add a `Control` function to the `net.Dialer` that re-validates resolved IPs:
```go
dialer := &net.Dialer{
    Timeout: config.DialTimeout,
    Control: func(network, address string, c syscall.RawConn) error {
        host, _, _ := net.SplitHostPort(address)
        ip := net.ParseIP(host)
        return validateIP(ip) // reuse existing validation logic
    },
}
```

---

## High Findings

### SEC-003: Committed Credentials in `.env.docker`

**Rule**: GO-SECRET-001  
**File**: `.env.docker:7-12`  
**Impact**: Credentials visible in git history to anyone with repository access.

`.env.docker` is tracked in git (committed in `64de75a`) despite being in `.gitignore`. Contains `POSTGRES_PASSWORD=riverpass`, `PGADMIN_DEFAULT_PASSWORD=admin123`, and `DATABASE_URL` with credentials.

**Remediation**: `git rm --cached .env.docker` and replace with `.env.docker.example` containing placeholder values. Note: credentials remain in git history and should be rotated if this is a public/shared repository.

---

### SEC-004: No Request Body Size Limits

**Rule**: GO-HTTP-002  
**Files**: `cmd/server/main.go:363-369` (HTTP), `cmd/server/main.go:283-286` (gRPC), `cmd/server/main.go:331-341` (Connect-RPC)  
**Impact**: Denial of service via OOM -- an attacker can send multi-GB payloads to crash the server.

- HTTP server: no `MaxHeaderBytes` set (defaults to 1MB for headers, but no body limit).
- gRPC server: no `grpc.MaxRecvMsgSize()` configured (implicit 4MB default, undocumented).
- Connect-RPC handlers: no `connect.WithReadMaxBytes()` or `connect.WithSendMaxBytes()`.
- No `http.MaxBytesReader` middleware anywhere in the serving path.

**Remediation**:
```go
// Connect-RPC handlers
options := connect.WithInterceptors(otelInterceptor, authConnectInterceptor)
readLimit := connect.WithReadMaxBytes(4 * 1024 * 1024) // 4MB
mux.Handle(pbconnect.NewWebhookServiceHandler(srv, options, readLimit))

// HTTP server
httpServer := &http.Server{
    MaxHeaderBytes: 1 << 20, // 1MB
    // ...
}

// gRPC server
grpcServer := grpc.NewServer(
    grpc.MaxRecvMsgSize(4 * 1024 * 1024),
    // ...
)
```

---

### SEC-005: Auth Disabled by Default with Wildcard CORS

**Rules**: GO-HTTP-006, GO-CORS-001  
**Files**: `cmd/server/main.go:122` (auth default), `cmd/server/main.go:436-453` (CORS), `internal/auth/context.go:143-149` (default auth info)  
**Impact**: Any deployment that does not explicitly enable auth is fully open to any website on the internet.

When `SPARROW_AUTH_ENABLED` is unset (default `false`), every request gets `DefaultAuthInfo()` with `tenant:admin` on a well-known UUID. Combined with `cors.AllowAll()` when `CORS_ALLOWED_ORIGINS` is unset, any website can make fully authenticated admin API calls.

**Remediation**: Either (a) default auth to enabled, or (b) emit a prominent startup warning when auth is disabled outside of `ENVIRONMENT=development`, and require explicit opt-in for wildcard CORS.

---

### SEC-006: JWT Claims Logged to Browser Console

**Rule**: JS-LOG-001  
**File**: `web/src/lib/services.ts:28-36`  
**Impact**: Every API request decodes and logs JWT claims (`org_id`, `sub`, `org_role`) to `console.log`, exposing tenant and user identifiers in browser DevTools.

```javascript
const payload = JSON.parse(atob(token.split(".")[1]));
console.log("[auth] JWT claims:", {
    org_id: payload.org_id,
    sub: payload.sub,
    org_role: payload.org_role,
});
```

**Remediation**: Remove this debug logging block entirely. If needed for development, gate behind an environment check.

---

### SEC-007: Cached API Keys Not Invalidated on Revocation

**Rule**: GO-AUTH-001  
**File**: `internal/auth/authenticator.go:134-143`  
**Impact**: Revoked API keys remain valid for up to 30 seconds (the cache TTL).

The `fromCache()` method only checks time expiry, not revocation status. After `RevokeAPIKey` is called, the cached `AuthInfo` continues to authenticate successfully until the cache entry expires.

**Remediation**: Either add a revocation check on cache hits, provide an explicit cache invalidation path from `RevokeAPIKey`, or reduce the cache TTL and document the accepted window.

---

## Medium Findings

### SEC-008: gRPC Reflection Enabled Unconditionally

**File**: `cmd/server/main.go:382`  
**Impact**: Anyone with network access to port 50051 can enumerate the entire API schema.

`reflection.Register(grpcServer)` is always called regardless of environment. In production, this leaks service structure.

**Remediation**: Gate behind `ENVIRONMENT == "development"`.

---

### SEC-009: No Content Security Policy

**Rule**: JS-CSP-001  
**Files**: `web/src/app.html`, `internal/ui/handler.go`  
**Impact**: No defense-in-depth against XSS. If any future XSS vector is introduced, there is no CSP to mitigate it.

No CSP header or meta tag is configured anywhere. The embedded SvelteKit UI is served without security headers.

**Remediation**: Add a CSP header when serving the embedded UI from the Go server. A reasonable starting policy:
```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com; img-src 'self' data: https:; connect-src 'self'
```

---

### SEC-010: Auto-Provisioning Triggered by Any Valid JWT

**File**: `internal/auth/tenant_resolver.go:123-139`, `internal/tenant/provisioner.go:42-75`  
**Impact**: Resource exhaustion -- an attacker with valid JWTs can auto-create tenants (limited to 2 per user, but unlimited users on a free IdP tier).

Any user with a valid JWT containing an unknown `org_id` triggers automatic tenant creation. With free Clerk accounts, this is exploitable at scale.

**Remediation**: Make auto-provisioning opt-in (`SPARROW_AUTO_PROVISION_TENANTS=true`) and add a global tenant limit.

---

### SEC-011: Missing `ReadHeaderTimeout` on HTTP Server

**Rule**: GO-HTTP-001  
**File**: `cmd/server/main.go:363-369`  
**Impact**: Slowloris-style attacks -- a client can hold connections open by slowly sending headers.

`ReadTimeout` is set (30s) but `ReadHeaderTimeout` is not. The `ReadTimeout` starts from connection acceptance, not from the start of header reading.

**Remediation**: Add `ReadHeaderTimeout: 10 * time.Second`.

---

### SEC-012: JWT Audience Validation Bug

**File**: `internal/auth/jwt.go:191-196`  
**Impact**: Only the first audience in `SPARROW_JWT_AUDIENCES` is validated; all others are silently ignored.

The loop iterates over configured audiences but `break`s after the first one, rendering multi-audience configurations non-functional.

**Remediation**: Remove the `break` statement to allow all audiences to be registered, or change the intent to explicitly document single-audience mode.

---

### SEC-013: Vite Dev Server `allowedHosts: true`

**Rule**: JS-CONFIG-001  
**File**: `web/vite.config.ts:11`  
**Impact**: DNS rebinding attacks against the dev server when running on a network-accessible interface.

**Remediation**: Remove `allowedHosts: true` or restrict to specific development hostnames.

---

## Low Findings

### SEC-014: Health Endpoint Leaks Internal Details

**File**: `internal/health/checker.go:88-107,112`

Exposes service endpoint addresses, database/queue technology names, and raw database error messages (which may contain hostnames, ports, or usernames).

**Remediation**: Return only health status (healthy/unhealthy) in production. Omit implementation details and raw error messages.

---

### SEC-015: Root API Key Logged to stdout

**File**: `internal/tenant/bootstrap.go:106-109`

The root API key is logged via `slog` at INFO level. In containerized deployments, this persists in log aggregation systems.

**Remediation**: Output the key via a separate mechanism (e.g., write to a file, stderr-only, or a one-time endpoint).

---

### SEC-016: `lastUsed` Map Grows Unbounded

**File**: `internal/auth/authenticator.go:80`

The `lastUsed map[uuid.UUID]time.Time` debounce map has no eviction logic, unlike the main cache (which evicts above 10,000 entries). Over time this leaks memory.

**Remediation**: Evict entries older than 5 minutes on the same schedule as the main cache.

---

### SEC-017: Clerk Secret Key in Frontend `.env`

**File**: `web/.env:4`

A Clerk backend secret key (`sk_test_*`) is stored in the frontend project's `.env`. While `.gitignore`d and not bundled by Vite (only `PUBLIC_` vars are exposed), its presence in a frontend directory is a hygiene risk.

**Remediation**: Move to the backend `.env` or a dedicated secrets manager. Remove from `web/.env`.

---

### SEC-018: `InsecureSkipVerify` Configurable / `VerifySSL` Not Wired

**Files**: `internal/webhooks/client/client.go:33`, `internal/webhooks/store/models.go:37`

`InsecureSkipVerify` is a global config (default: false, good). However, the per-webhook `VerifySSL` field exists in the data model but is not applied at delivery time.

**Remediation**: Either wire `VerifySSL` per-webhook or remove the unused field to avoid confusion.

---

### SEC-019: Webhook Secrets Accept Weak Values

**Files**: `internal/webhooks/webhook_service.go:1588-1589`, `internal/grpc/webhook_conversions.go:25`

Webhook secrets are user-supplied with no minimum length or entropy enforcement. Users can set `""` or `"password"`. Secrets are stored in plaintext.

**Remediation**: Enforce a minimum length (e.g., 32 characters) or auto-generate cryptographically random secrets. Consider encrypted storage.

---

### SEC-020: External Resource Loads Without CSP

**Files**: `web/src/app.html:8-10` (Google Fonts), `web/src/lib/auth/providers/none/NoAuthShell.svelte:34` (external image)

External fonts and images leak user IPs to third parties and create runtime dependencies.

**Remediation**: Self-host fonts and images, or accept the privacy trade-off and allowlist in CSP.

---

### SEC-021: Missing `rel="noopener noreferrer"` on External Links

**File**: `web/src/routes/+page.svelte:132,137,200,372,394,395`

Six `target="_blank"` links without explicit `rel="noopener noreferrer"`. Modern browsers handle this automatically, but it's still best practice.

**Remediation**: Add `rel="noopener noreferrer"` to all external `target="_blank"` links.

---

### SEC-022: Excessive Console Logging in Production

**Files**: `web/src/lib/services.ts:30-36`, plus 13 other `console.*` calls across components.

Debug and error logging persists in production builds, potentially leaking operational details.

**Remediation**: Remove debug logging. For error logging, use a structured logger that can be disabled in production.

---

## Positive Security Observations

The following were implemented correctly and deserve recognition:

| Area | Implementation | Assessment |
|------|----------------|------------|
| SQL Injection | All queries use `$1`, `$2` parameterized placeholders | **Safe** |
| JWT Algorithm | `jwt.WithValidMethods([]string{"RS256"})` | **Correct** -- prevents algorithm confusion |
| JWT Expiration | `jwt.WithExpirationRequired()` | **Correct** |
| API Key Generation | `crypto/rand.Read()` (128 bits entropy) | **Correct** |
| API Key Hashing | SHA-256 (appropriate for high-entropy secrets) | **Correct** |
| HMAC Signing | HMAC-SHA256 with timestamp binding | **Correct** |
| URL Validation | Comprehensive SSRF blocklist (private IPs, metadata, multicast, IPv6-mapped) | **Good** (but bypassable, see SEC-001/002) |
| Template DoS | 1MB output limit, repeat capped at 1000 | **Good** |
| Template Safety | `text/template` with restricted FuncMap, plain `map[string]any` context | **Acceptable** (no method calls on data) |
| Concurrency | All cache maps protected by `sync.RWMutex` | **Correct** |
| Auth Context | Fire-and-forget goroutines use `context.WithoutCancel` | **Correct** |
| Weak Crypto | No MD5, SHA1, DES, RC4 usage found | **Good** |
| Debug Endpoints | No `net/http/pprof` or `expvar` imports | **Good** |
| Frontend XSS | Zero `{@html}`, `innerHTML`, `document.write`, `eval()` usage | **Good** |
| Token Storage | Memory-only token provider, no localStorage/sessionStorage | **Good** |
| Cookie Security | No server-side cookies set; header-based Bearer auth | **Good** |

---

## Remediation Status

All **Critical** and **High** findings have been remediated. The following table tracks the status of each finding:

| Finding | Severity | Status | Fix Description |
|---------|----------|--------|-----------------|
| SEC-001 | Critical | **FIXED** | Added `ssrfSafeCheckRedirect()` in `internal/webhooks/client/ssrf.go` that validates scheme, hostname, and resolved IPs for each redirect hop. Wired into `http.Client.CheckRedirect`. |
| SEC-002 | Critical | **FIXED** | Added `ssrfDialControl()` in `internal/webhooks/client/ssrf.go` that validates resolved IPs at TCP connect time via `net.Dialer.Control`, eliminating the DNS rebinding TOCTOU window. |
| SEC-003 | High | **FIXED** | Ran `git rm --cached .env.docker` to untrack the file (already in `.gitignore`). Created `.env.docker.example` with placeholder values. Note: credentials remain in git history -- rotate if repository is public/shared. |
| SEC-004 | High | **FIXED** | Added `MaxHeaderBytes: 1MB` and `ReadHeaderTimeout: 10s` to HTTP server. Added `grpc.MaxRecvMsgSize(4MB)` and `grpc.MaxSendMsgSize(4MB)` to gRPC server. Added `connect.WithReadMaxBytes(4MB)` and `connect.WithSendMaxBytes(4MB)` to all Connect-RPC handlers. |
| SEC-005 | High | **FIXED** | Added prominent startup warnings when auth is disabled in production, and a critical warning when both auth disabled and CORS wildcard are active in production. |
| SEC-006 | High | **FIXED** | Removed the JWT claims debug logging block from `web/src/lib/services.ts`. |
| SEC-007 | High | **FIXED** | Added `InvalidateKey()`, `InvalidateKeyByHash()`, `InvalidateAllKeys()` methods to `APIKeyAuthenticator`. Added `KeyCacheInvalidator` interface. `RevokeAPIKey` handler now invalidates the cache entry immediately on revocation. Wired via `WithKeyCacheInvalidator()` option in `cmd/server/main.go`. |
| SEC-008 | Medium | Open | gRPC reflection still unconditionally enabled. |
| SEC-009 | Medium | Open | No Content Security Policy header. |
| SEC-010 | Medium | Open | Auto-provisioning not gated. |
| SEC-011 | Medium | **FIXED** (as part of SEC-004) | `ReadHeaderTimeout: 10s` added to HTTP server. |
| SEC-012 | Medium | Open | JWT audience validation loop still breaks after first entry. |
| SEC-013 | Medium | Open | Vite dev server `allowedHosts: true`. |
| SEC-014 | Low | Open | Health endpoint leaks internal details. |
| SEC-015 | Low | Open | Root API key logged to stdout. |
| SEC-016 | Low | Open | `lastUsed` map grows unbounded. |
| SEC-017 | Low | Open | Clerk secret key in frontend `.env`. |
| SEC-018 | Low | Open | `VerifySSL` field not wired. |
| SEC-019 | Low | Open | Webhook secrets accept weak values. |
| SEC-020 | Low | Open | External resource loads without CSP. |
| SEC-021 | Low | Open | Missing `rel="noopener noreferrer"` on external links. |
| SEC-022 | Low | Open | Excessive console logging in production. |

### Files Changed

| File | Change |
|------|--------|
| `internal/webhooks/client/ssrf.go` | **NEW** -- SSRF-safe redirect validation, dial control, IP validation (SEC-001, SEC-002) |
| `internal/webhooks/client/client.go` | Added `CheckRedirect` and `Control` to HTTP client (SEC-001, SEC-002) |
| `internal/webhooks/urlvalidation.go` | Refactored to use `client.ValidateIP()` (SEC-002) |
| `cmd/server/main.go` | Body size limits (SEC-004), startup warnings (SEC-005), `ReadHeaderTimeout` (SEC-011), API key cache invalidation wiring (SEC-007) |
| `web/src/lib/services.ts` | Removed JWT debug logging (SEC-006) |
| `internal/auth/authenticator.go` | Added `KeyCacheInvalidator` interface and invalidation methods (SEC-007) |
| `internal/grpc/tenant_server.go` | Added cache invalidation on key revocation (SEC-007) |
| `.env.docker` | Untracked from git (SEC-003) |
| `.env.docker.example` | **NEW** -- Template with placeholder credentials (SEC-003) |

---

## Recommended Fix Priority (Remaining)

| Priority | Finding | Effort |
|----------|---------|--------|
| 1 | SEC-012: JWT audience bug | Low (remove `break`) |
| 2 | SEC-008: Gate gRPC reflection | Low (one `if` statement) |
| 3 | SEC-009: Add CSP header | Medium |
| 4 | SEC-010: Gate auto-provisioning | Medium |
| 5 | SEC-013: Vite allowedHosts | Low |
| 6-13 | Remaining low findings | Low each |
