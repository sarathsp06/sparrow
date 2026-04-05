# Sparrow -- Architecture

This document describes Sparrow's package structure, dependency graph, and design principles.

---

## Dependency Graph

```
cmd/server/main.go  (composition root — wires everything)
    │
    ├── internal/tenant    ──→ pkg/storage
    ├── internal/webhooks  ──→ pkg/storage, pkg/errors
    │       ├── store/     ──→ pkg/storage, pkg/types
    │       ├── queue/     ──→ store, client, pkg/errors, internal/observability
    │       └── client/    ──→ store (models only), pkg/errors
    └── internal/grpc      ──→ both domain packages (transport layer)
```

`tenant` and `webhooks` never import each other.

### Dependency Matrix

```
              imports:  tenant  webhooks
  tenant         -       No
  webhooks       No       -
  grpc          Yes     Yes
  main.go       Yes     Yes
```

Zero cycles.

---

## Package Responsibilities

### Domain Packages

**`internal/tenant`** -- Tenant lifecycle. A default tenant is bootstrapped on first boot.

- Tables: `tenants`

**`internal/webhooks`** -- Core business domain: namespaces, events, subscriptions, deliveries, health tracking.

- Tables: `namespaces`, `webhook_registrations`, `event_registrations`, `event_subscriptions`, `event_records`, `webhook_deliveries`, `webhook_health_events`, `webhook_health_summaries`, `webhook_health_state`
- Sub-packages:
  - `store/` -- Data access (repository pattern, SQL queries)
  - `queue/` -- Async processing (River workers: `EventWorker`, `WebhookWorker`)
  - `client/` -- HTTP delivery (request building, HMAC signing, templating)

### Infrastructure Packages

**`internal/grpc`** -- gRPC service implementations (transport layer).

- 5 proto-defined service handlers delegating to domain services
- The only package that imports both domain packages

**`internal/connect`** -- Connect-RPC adapter.

- Wraps gRPC handlers for HTTP/JSON access on `:8080`

**`internal/observability`** -- OpenTelemetry setup (traces, metrics, logs via OTLP).

**`internal/ui`** -- Embedded SvelteKit frontend (`go:embed`).

**`internal/config`** -- Environment variable loading.

**`internal/health`** -- Health check endpoint.

### Shared Packages

- `pkg/storage` -- `DB`/`DBTX` interfaces, `WithTransaction` helper, SQL error translation
- `pkg/errors` -- Error classification (9 categories, retryability determination)
- `pkg/types` -- Shared utility types

---

## Design Principles

### Composition Root in `main.go`

`cmd/server/main.go` is the only file that imports both domain packages. It constructs repositories, services, and wires them together. No framework -- just constructor functions and explicit wiring.

### Repository Per Domain, Not Per Table

Each domain owns a `Repository` interface and implementation. The repository encapsulates all SQL for that domain's tables.

### Schema Ownership

Each domain package owns its tables:

- `internal/tenant` -- `tenants`
- `internal/webhooks` -- `namespaces` + all 8 webhook/event/delivery/health tables

### No Shared Models

There is no shared "models" package and no ORM. Each package defines its own models matching its own SQL schemas. The only shared infrastructure is `pkg/storage` (DB abstraction) and `pkg/types` (generic utilities).
