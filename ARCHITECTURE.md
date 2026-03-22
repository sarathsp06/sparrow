# Sparrow -- Architecture

This document describes Sparrow's package structure, dependency graph, and design principles.

---

## Dependency Graph

```
cmd/server/main.go  (composition root — wires everything)
    │
    ├── internal/tenant   ──→ internal/auth, pkg/storage
    ├── internal/namespace ──→ internal/auth, pkg/storage
    ├── internal/webhooks  ──→ internal/auth, pkg/storage, pkg/errors
    │       ├── store/     ──→ pkg/storage, pkg/types
    │       ├── queue/     ──→ store, client, pkg/errors, internal/observability
    │       └── client/    ──→ store (models only), pkg/errors
    ├── internal/auth      ──→ pkg/storage (for interface types only)
    └── internal/grpc      ──→ all three domain packages (transport layer)
```

`tenant`, `namespace`, and `webhooks` never import each other. Their only shared dependency is `internal/auth`, which defines narrow interfaces that each package implements independently.

### Dependency Matrix

```
              imports:  tenant  namespace  webhooks  auth
  tenant         -        No       No       Yes
  namespace      No        -       No       Yes
  webhooks       No       No        -       Yes
  auth           No       No       No        -
  grpc          Yes      Yes      Yes       Yes
  main.go       Yes      Yes      Yes       Yes
```

Zero cycles. The `auth` package must never import `tenant`, `namespace`, or `webhooks`.

---

## Package Responsibilities

### Domain Packages

**`internal/tenant`** -- Tenant lifecycle and API key management.

- Tables: `tenants`, `api_keys`
- Implements `auth.APIKeyStore`, `auth.TenantLookup`, `auth.ExternalTenantLookup`
- Auto-provisioning of tenants on first JWT login (`AutoProvisioner` implements `auth.TenantProvisioner`)

**`internal/namespace`** -- Namespace CRUD and membership management.

- Tables: `namespaces`, `namespace_memberships`
- Implements `auth.MembershipResolver` for namespace role resolution
- Syncs role changes to external identity providers via `auth.IdentityProvider`

**`internal/webhooks`** -- Core business domain: events, subscriptions, deliveries, health tracking.

- Tables: `webhook_registrations`, `event_registrations`, `event_subscriptions`, `event_records`, `webhook_deliveries`, `webhook_health_events`, `webhook_health_summaries`, `webhook_health_state`
- Sub-packages:
  - `store/` -- Data access (repository pattern, SQL queries)
  - `queue/` -- Async processing (River workers: `EventWorker`, `WebhookWorker`)
  - `client/` -- HTTP delivery (request building, HMAC signing, templating)

### Infrastructure Packages

**`internal/auth`** -- JWT/API-key authentication, RBAC, interceptors.

- Defines interfaces implemented by domain packages (dependency inversion)
- `AuthInfo` struct carried in request context, consumed by all packages
- Caching resolvers for tenant lookups (5min TTL) and namespace memberships (30s TTL)
- Pluggable `IdentityProvider` interface (Clerk implementation, noop default)

**`internal/grpc`** -- gRPC service implementations (transport layer).

- 9 service handlers delegating to domain services
- The only package that imports all three domain packages

**`internal/connect`** -- Connect-RPC adapter.

- Wraps gRPC handlers for HTTP/JSON access on `:8080`

**`internal/audit`** -- Audit logging.

- Tables: `audit_logs`
- Async (fire-and-forget) and sync modes for use within transactions

**`internal/observability`** -- OpenTelemetry setup (traces, metrics, logs via OTLP).

**`internal/ui`** -- Embedded SvelteKit frontend (`go:embed`).

**`internal/config`** -- Environment variable loading.

**`internal/health`** -- Health check endpoint.

### Shared Packages

- `pkg/storage` -- `DB`/`DBTX` interfaces, `WithTransaction` helper, SQL error translation
- `pkg/errors` -- Error classification (9 categories, retryability determination)
- `pkg/types` -- Shared utility types

---

## Interface Contracts

The `auth` package defines interfaces that domain packages implement. All wiring happens in `cmd/server/main.go`.

```
auth.APIKeyStore          ← tenant.pgRepository
auth.TenantLookup         ← tenant.pgRepository
auth.ExternalTenantLookup ← tenant.pgRepository
auth.TenantProvisioner    ← tenant.AutoProvisioner
auth.MembershipResolver   ← namespace.pgRepository
auth.IdentityProvider     ← auth.ClerkIdentityProvider (or NoopIdentityProvider)
```

### Auth Context as Communication Channel

The three domain packages don't call each other at runtime. The auth pipeline builds an `AuthInfo` struct (tenant ID, role, namespace memberships) and stores it in the request context. Service methods extract this context and enforce authorization independently.

```
Request → Auth Interceptor → [tenant repo: validate API key or JWT]
                            → [namespace repo: resolve memberships]
                            → AuthInfo injected into context
                            → Service method extracts AuthInfo
                            → Service enforces permissions
                            → Repository queries scoped by tenant_id + namespace
```

---

## Design Principles

### Bounded Contexts with Interface Contracts

Each domain package is a bounded context. They communicate through interfaces defined in `internal/auth`, not through direct imports. New cross-package interactions should follow this pattern: define the interface in `auth`, implement it in the producing package, consume it in the other.

### Composition Root in `main.go`

`cmd/server/main.go` is the only file that imports all three domain packages. It constructs repositories, services, and wires them together. No framework -- just constructor functions and explicit wiring.

### Repository Per Domain, Not Per Table

Each domain owns a `Repository` interface and `pgRepository` implementation. The repository encapsulates all SQL for that domain's tables. This avoids per-table repositories (too granular) or a single shared repository (too broad).

### Schema Ownership

Each domain package owns its tables:

- `internal/tenant` -- `tenants`, `api_keys`
- `internal/namespace` -- `namespaces`, `namespace_memberships`
- `internal/webhooks` -- all 8 webhook/event/delivery/health tables
- `internal/audit` -- `audit_logs`

### No Shared Models

There is no shared "models" package and no ORM. Each package defines its own models matching its own SQL schemas. The only shared infrastructure is `pkg/storage` (DB abstraction) and `pkg/types` (generic utilities).
