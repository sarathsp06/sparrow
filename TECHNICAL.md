# Sparrow -- Technical Reference

This document covers Sparrow's internal architecture, deployment guides, and detailed configuration. For quick start and basic usage, see [README.md](README.md).

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Deployment Guide](#deployment-guide)
  - [Docker Compose (Recommended)](#docker-compose-recommended)
  - [Binary Deployment](#binary-deployment)
  - [Enabling Authentication](#enabling-authentication)
  - [Setting Up Clerk](#setting-up-clerk)
  - [Setting Up Other OIDC Providers](#setting-up-other-oidc-providers)
- [Auth Architecture](#auth-architecture)
  - [API Key Authentication](#api-key-authentication)
  - [JWT Authentication](#jwt-authentication)
  - [RBAC Model](#rbac-model)
  - [Web UI Auth Providers](#web-ui-auth-providers)
- [Architecture Deep Dive](#architecture-deep-dive)
  - [Data Model](#data-model)
  - [Tenant Isolation](#tenant-isolation)
  - [Error Classification & Retry Logic](#error-classification--retry-logic)
  - [Health State Machine](#health-state-machine)
  - [HTTP Client Design](#http-client-design)
  - [Observability](#observability)
  - [Server Boot Sequence](#server-boot-sequence)
  - [Web UI Architecture](#web-ui-architecture)

---

## Architecture Overview

```mermaid
graph LR
    App[Your Application] -->|PushEvent<br>HTTP / gRPC| API

    subgraph Sparrow
        API[Dual-Protocol API<br>gRPC :50051 · HTTP :8080]
        Auth[Auth Interceptor<br>JWT + API Key]
        DB[(PostgreSQL)]
        EQ[Events Queue]
        WQ[Webhooks Queue]
        EW[EventWorker]
        WW[WebhookWorker]
        Health[Health Tracker]
        Tenants[Tenant Isolation]

        API --> Auth
        Auth -->|Tenant-scoped| Tenants
        Tenants -->|Store event| DB
        API -->|Enqueue| EQ
        EQ --> EW
        EW -->|Find matching<br>subscriptions| DB
        EW -->|Fan-out jobs| WQ
        WQ --> WW
        WW -->|Load config +<br>transform payload| DB
        WW -->|Record outcome| DB
        WW -->|Update| Health
        Health -->|LISTEN/NOTIFY| DB
    end

    WW -->|HTTP POST<br>HMAC-signed| Endpoint[External Endpoints]
    UI[Embedded SvelteKit UI] -->|Connect-RPC| API
```

### Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go 1.25 |
| Database | PostgreSQL 15 |
| Job Queue | River (Postgres-based) |
| API | gRPC (`:50051`) + Connect-RPC/HTTP (`:8080`) |
| Protobuf | buf.build toolchain |
| Web UI | SvelteKit 5 + TypeScript + Tailwind CSS 4 (embedded static build) |
| Observability | OpenTelemetry (traces, metrics, logs via OTLP) |
| DB Access | pgx/v5 + sqlx (OTel-instrumented) |
| Container | Multi-stage Dockerfile (distroless nonroot) |

---

## Deployment Guide

### Docker Compose (Recommended)

The simplest way to run Sparrow. This starts PostgreSQL, runs migrations, and launches the server with the embedded web UI.

```bash
docker compose up -d
```

That's it. The server is available at:
- **Web UI:** http://localhost:8080
- **HTTP API (Connect-RPC):** http://localhost:8080
- **gRPC API:** localhost:50051

On first boot, a root API key is printed to the logs:

```bash
docker compose logs sparrow
```

To stop:

```bash
docker compose down        # stop containers
docker compose down -v     # stop and delete data
```

### Binary Deployment

Build from source and run directly:

```bash
# Build with embedded UI
make build-with-ui

# Start infrastructure (or provide your own Postgres)
export DATABASE_URL=postgres://user:pass@localhost:5432/sparrow?sslmode=disable

# Run migrations
./build/server-* migrate  # or: make migrate

# Start the server
SPARROW_SERVE_UI=true ./build/server-*
```

### Enabling Authentication

By default, Sparrow runs without authentication. To enable it:

**1. API keys only (simplest):**

```bash
# docker-compose.yml or environment
SPARROW_AUTH_ENABLED=true
```

On first boot, Sparrow prints a root API key. Use it to create additional keys:

```bash
curl -s -X POST http://localhost:8080/webhook.APIKeyService/CreateAPIKey \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{
    "tenant_id": "<tenant_id>",
    "name": "Production Key",
    "role": "tenant:admin"
  }'
```

**2. API keys + JWT (for web UI with identity provider):**

```bash
SPARROW_AUTH_ENABLED=true
SPARROW_JWKS_URL=https://your-provider.example.com/.well-known/jwks.json
SPARROW_JWT_TENANT_CLAIM=org_id
SPARROW_JWT_ROLE_CLAIM=org_role
```

When both are enabled, the interceptor tries JWT first, then API key. This lets the web UI use JWTs while scripts use API keys.

### Setting Up Clerk

Clerk is a managed identity provider. Sparrow auto-provisions tenants when users create Clerk organizations -- no webhooks or manual setup needed.

**1. Create a Clerk application** at [clerk.com](https://clerk.com) and enable Organizations.

**2. Create a JWT template** in Clerk Dashboard > JWT Templates:

- Template name: `sparrow` (or any name)
- Claims (JSON):
  ```json
  {
    "org_id": "{{org.id}}",
    "org_role": "{{org_membership.role}}"
  }
  ```

**3. Configure the backend:**

```bash
SPARROW_AUTH_ENABLED=true
SPARROW_JWKS_URL=https://<your-instance>.clerk.accounts.dev/.well-known/jwks.json
SPARROW_JWT_TENANT_CLAIM=org_id
SPARROW_JWT_ROLE_CLAIM=org_role
SPARROW_JWT_ISSUER=https://<your-instance>.clerk.accounts.dev
```

**4. Configure the frontend** (in `web/.env` or environment):

```bash
PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_your-key-here
PUBLIC_API_URL=http://localhost:8080   # or your deployment URL
```

### Setting Up Other OIDC Providers

Sparrow's backend is **provider-agnostic**. It validates JWTs using standard JWKS and reads configurable claims. No provider SDK is linked into the Go binary.

**Any OIDC provider** works as long as it:

1. Publishes a JWKS endpoint with RS256 keys
2. Includes a tenant/org identifier claim in the JWT
3. Optionally includes a role claim

---

## Auth Architecture

### API Key Authentication

API keys use the format `sk_<tenant_slug>_<random>` and are verified via SHA-256 hash lookup with an in-memory cache (5-minute TTL).

```mermaid
sequenceDiagram
    participant Client
    participant Interceptor as Auth Interceptor
    participant Cache as Key Cache (5min TTL)
    participant DB as PostgreSQL

    Client->>Interceptor: Request + Authorization: Bearer sk_acme_...
    Interceptor->>Cache: Lookup key hash
    alt Cache hit
        Cache-->>Interceptor: AuthInfo
    else Cache miss
        Interceptor->>DB: SELECT by key_hash
        DB-->>Interceptor: API key record
        Interceptor->>Cache: Store AuthInfo
    end
    Interceptor->>Interceptor: Check expiry, revocation
    Interceptor->>Interceptor: Inject AuthInfo into context
    Note over Interceptor: Request proceeds with tenant scope
```

**API key management:**

```bash
# Create a key
curl -s -X POST http://localhost:8080/webhook.APIKeyService/CreateAPIKey \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{"tenant_id": "<id>", "name": "CI Key", "role": "tenant:member"}'

# List keys for a tenant
curl -s -X POST http://localhost:8080/webhook.APIKeyService/ListAPIKeys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{"tenant_id": "<id>"}'

# Revoke a key
curl -s -X POST http://localhost:8080/webhook.APIKeyService/RevokeAPIKey \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{"id": "<api_key_id>"}'
```

### JWT Authentication

Provider-agnostic RS256 JWT verification. No external JWT library -- verification is implemented with Go's `crypto/rsa` stdlib.

```mermaid
sequenceDiagram
    participant Browser
    participant Frontend as SvelteKit Frontend
    participant IdP as Identity Provider<br>(Clerk / Auth0 / etc.)
    participant API as Sparrow API
    participant JWKS as JWKS Endpoint
    participant Resolver as Tenant Resolver
    participant DB as PostgreSQL

    Browser->>IdP: Sign in
    IdP-->>Browser: Session
    Browser->>Frontend: Access dashboard
    Frontend->>IdP: getToken()
    IdP-->>Frontend: JWT (RS256)
    Frontend->>API: Request + Authorization: Bearer <JWT>
    API->>API: Parse JWT header (kid, alg)
    API->>JWKS: Fetch public keys (cached 1hr)
    JWKS-->>API: RSA public keys
    API->>API: Verify RS256 signature
    API->>API: Validate exp, nbf, iss, aud
    API->>API: Extract tenant claim (org_id)
    API->>Resolver: ResolveTenant(org_id, sub)
    alt Tenant exists
        Resolver->>DB: Lookup by external_id (cached 5min)
        DB-->>Resolver: Internal tenant UUID
    else Unknown org_id + provisioner configured
        Resolver->>DB: Auto-provision new tenant
        DB-->>Resolver: New tenant UUID
    end
    Resolver-->>API: Tenant ID
    API->>API: Map role claim -> Sparrow role
    API->>API: Inject AuthInfo into context
    Note over API: Request proceeds with tenant scope
```

**Key components:**

| Component | File | Purpose |
|-----------|------|---------|
| `JWKSProvider` | `internal/auth/jwks.go` | Fetches and caches RSA public keys from a JWKS URL |
| `JWTAuthenticator` | `internal/auth/jwt.go` | Validates JWT signature and claims, maps to AuthInfo |
| `CachingTenantResolver` | `internal/auth/tenant_resolver.go` | Maps external org IDs to internal tenant UUIDs with caching |
| `AutoProvisioner` | `internal/tenant/provisioner.go` | Creates tenants on first JWT login, enforces per-user limits |
| `JWTClaimsConfig` | `internal/auth/jwt.go` | Configurable claim names, role mapping, issuer/audience |

### RBAC Model

5 roles and 25 permissions. `auth.Authorize(authInfo, permission)` checks whether the authenticated identity has a specific permission.

| Role | Scope | Description |
|---|---|---|
| `tenant:admin` | Tenant-wide | Full access to all resources within the tenant |
| `tenant:member` | Tenant-wide | Read/write access to webhooks, events, subscriptions |
| `namespace:admin` | Single namespace | Full access within a specific namespace |
| `namespace:member` | Single namespace | Read/write access within a specific namespace |
| `namespace:viewer` | Single namespace | Read-only access within a specific namespace |

**JWT role mapping (configurable):** `org:admin` -> `tenant:admin`, `org:member` -> `tenant:member`.

**Platform admin keys** (`is_platform_admin: true`) bypass tenant scoping entirely, allowing cross-tenant management operations.

### Web UI Auth Providers

The SvelteKit web dashboard has a pluggable auth provider system. The active provider is selected via environment variables.

**Provider selection** (via `PUBLIC_AUTH_PROVIDER` or auto-detected from provider-specific keys):

| `PUBLIC_AUTH_PROVIDER` | Provider-Specific Key | Result |
|---|---|---|
| *(unset)* | *(unset)* | No authentication (open access) |
| *(unset)* | `PUBLIC_CLERK_PUBLISHABLE_KEY=pk_...` | Clerk (auto-detected) |
| `clerk` | `PUBLIC_CLERK_PUBLISHABLE_KEY=pk_...` | Clerk (explicit) |
| `none` | *(any)* | No authentication (forced) |


**Adding a new auth provider** (e.g., Auth0):

1. Create `web/src/lib/auth/providers/auth0/Auth0AuthShell.svelte` with the same snippet contract
2. Call `registerTokenProvider()` from `web/src/lib/auth.ts` with your provider's token getter
3. Add `"auth0"` to `AuthProviderType` in `web/src/lib/auth/types.ts`
4. Add detection logic in `web/src/lib/auth/provider.ts`
5. Add a case in `web/src/lib/auth/AuthShell.svelte`

The services layer and backend remain completely unchanged -- they only see JWTs.

## Architecture Deep Dive

### Dual-Protocol API

The same gRPC service implementations back both protocols -- no code duplication:

- **gRPC** on `:50051` for high-performance programmatic access
- **Connect-RPC (HTTP/JSON)** on `:8080` for curl, browsers, and any HTTP client

Seven domain services: `WebhookService`, `EventService`, `SubscriptionService`, `DeliveryService`, `HealthService`, `TenantService`, `APIKeyService`.

### Data Model

```mermaid
erDiagram
    tenants ||--o{ api_keys : "has"
    tenants ||--o{ event_registrations : "owns"
    tenants ||--o{ webhook_registrations : "owns"
    tenants ||--o{ event_subscriptions : "owns"
    tenants ||--o{ event_records : "owns"

    tenants {
        uuid id PK
        string name
        string slug UK
        string external_id UK
        string created_by
        string status
        jsonb settings
        timestamp created_at
        timestamp updated_at
    }

    api_keys {
        uuid id PK
        uuid tenant_id FK
        string key_prefix
        bytes key_hash
        string role
        string namespace_scope
        bool is_platform_admin
        timestamp expires_at
        timestamp last_used_at
        timestamp revoked_at
    }
```

### Tenant Isolation

Every domain table (`event_registrations`, `webhook_registrations`, `event_subscriptions`, `event_records`) has a `tenant_id` column with a foreign key to `tenants`. All queries are scoped by tenant ID, extracted from the authenticated context (API key or JWT).

- **External ID mapping:** Tenants have an `external_id` that maps to an identity provider's organization ID (e.g., Clerk `org_id`). JWT-authenticated requests are scoped to the correct tenant via this mapping.
- **Auto-provisioning:** When a JWT contains an unknown `org_id`, the `AutoProvisioner` creates a new tenant automatically. A per-user limit of 2 tenants is enforced via the `created_by` column (JWT `sub` claim).
- **Default tenant:** Auto-created on first boot with UUID `00000000-0000-0000-0000-000000000001`.

---

### Error Classification & Retry Logic

The `pkg/errors/` package implements a taxonomy-based error classifier that inspects Go errors to determine retryability:

```mermaid
flowchart LR
    Err[Go error] --> Unwrap[Unwrap error chain]

    Unwrap --> URLErr[*url.Error]
    Unwrap --> NetErr[net.Error]
    Unwrap --> TLS[TLS errors]
    Unwrap --> DNS[*net.DNSError]
    Unwrap --> OpErr[*net.OpError]
    Unwrap --> Sys[*os.SyscallError]
    Unwrap --> Errno[syscall.Errno]
    Unwrap --> Str[String pattern fallback]

    URLErr & NetErr & OpErr & Sys & Errno & Str --> Retryable
    TLS & DNS --> NonRetryable

    subgraph Retryable [Retryable -> River retries]
        R1[5xx server error]
        R2[Timeout]
        R3[Connection refused]
        R4[Network error<br>ECONNRESET / EPIPE / EHOSTUNREACH]
    end

    subgraph NonRetryable [Non-retryable -> stop]
        NR1[4xx client error]
        NR2[DNS resolution failure]
        NR3[TLS/SSL handshake failure]
    end
```

Non-retryable errors return `nil` to River (stopping retries); retryable errors return an `error` to trigger River's built-in retry mechanism.

---

### Health State Machine

Event-sourced health calculation with a 24-hour lookback window:

```mermaid
stateDiagram-v2
    [*] --> unknown

    unknown --> healthy : >=90% success rate\n(3+ events)
    unknown --> degraded : <90% success rate\n(5+ events)
    unknown --> unhealthy : 5+ consecutive failures\nOR <80% rate (10+ events)

    healthy --> degraded : Success rate drops <90%
    healthy --> unhealthy : 5+ consecutive failures

    degraded --> healthy : Success rate recovers >=90%
    degraded --> unhealthy : 5+ consecutive failures\nOR <80% rate (10+ events)

    unhealthy --> degraded : Partial recovery
    unhealthy --> healthy : Full recovery >=90%

    healthy --> unknown : No events in 24h
    degraded --> unknown : No events in 24h
    unhealthy --> unknown : No events in 24h
```

**How it works:**
1. Each delivery outcome is recorded as a health event
2. Health state is atomically upserted (tracks consecutive failures, last success/failure timestamps)
3. Webhook health status is recalculated and persisted
4. Hourly aggregation computes per-webhook summaries (p95 response time, error category breakdown)

**Real-time reactivity:** PostgreSQL `LISTEN/NOTIFY` pushes health events for async processing.

---

### HTTP Client Design

A centralized, OTel-instrumented HTTP client (`internal/webhooks/client/`):

- **Connection pooling**: 100 max idle connections, 10 per host, 90s idle timeout
- **HMAC signing**: `X-Sparrow-Signature-256` header using `HMAC-SHA256(timestamp + "." + payload, secret)` (Stripe/GitHub pattern)
- **Template engine**: Go `text/template` with LRU cache (100 entries, SHA-256 keyed), ~20 built-in helper functions (json, base64, urlencode, string manipulation, etc.)
- **Object pooling**: `sync.Pool` for `bytes.Buffer`, `[]byte` slices, and header maps to reduce GC pressure
- **Header merging**: Subscription-level headers override webhook-level defaults
- **In-process metrics**: Lock-free atomic counters for request totals, error categories, cache hit rates, and response time statistics

### Web UI Architecture

The web dashboard is a SvelteKit application that compiles to static files and is embedded into the Go binary via `go:embed`. See [web/README.md](web/README.md) for development setup.

**Build pipeline:**
1. `cd web && npm run build` -- compiles SvelteKit to static files in `internal/ui/dist/`
2. `go build ./cmd/server` -- embeds `internal/ui/dist/` via `go:embed`
3. At runtime, `internal/ui/embed.go` serves the SPA with proper fallback routing

The Docker image builds the frontend automatically -- no manual build step needed.
