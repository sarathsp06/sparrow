# Sparrow -- Complete Codebase Reference

> **Keep this file up to date.** Whenever a relevant change is made to the codebase -- new features, architectural changes, schema migrations, configuration changes, or dependency updates -- update the corresponding section(s) in this file so it remains a reliable single-source-of-truth for AI-assisted development.

**Module**: `github.com/sarathsp06/sparrow`  
**Go**: 1.25  
**Repository**: `/Users/sarathsadasivanpillai/projects/httpqueue`

---

## Architecture Overview

Sparrow is a **multi-tenant webhook delivery platform** with an event-driven architecture. It accepts webhook registrations, event definitions, and subscriptions (with Go-template payload transformation), then fans out events into a River job queue for async HTTP delivery with retries, health tracking, and error classification.

```
                         ┌─────────────────────┐
     Clients             │    Sparrow Server    │
  ┌──────────┐           │                      │
  │ gRPC     │──:50051──>│  internal/grpc       │──┐
  │ clients  │           │  (9 services)        │  │
  └──────────┘           │                      │  │
  ┌──────────┐           │  internal/connect    │  │  ┌──────────────┐
  │ HTTP/    │──:8080───>│  (Connect-RPC        │  ├─>│ Service Layer│
  │ Connect  │           │   adapter)           │  │  │ webhooks.Svc │
  └──────────┘           │                      │  │  │ tenant.Svc   │
  ┌──────────┐           │  internal/ui         │  │  │ namespace.Svc│
  │ Browser  │──:8080───>│  (SvelteKit embed)   │  │  └──────┬───────┘
  └──────────┘           └──────────────────────┘  │         │
                                                   │         v
                         ┌─────────────────────┐   │  ┌──────────────┐
                         │  internal/auth       │<─┘  │  Repository  │
                         │  JWT + API Key auth  │     │  (sqlx/DBTX) │
                         │  RBAC (5 roles)      │     └──────┬───────┘
                         └─────────────────────┘            │
                                                            v
                         ┌─────────────────────┐     ┌──────────────┐
                         │  internal/webhooks/  │     │  PostgreSQL  │
                         │  queue (River)       │     │  13 tables   │
                         │  EventWorker         │     │  10 migrations│
                         │  WebhookWorker       │     └──────────────┘
                         └─────────┬───────────┘
                                   │
                                   v
                         ┌─────────────────────┐
                         │  internal/webhooks/  │
                         │  client (HTTP)       │
                         │  HMAC signing        │
                         │  Response capture    │
                         └─────────────────────┘
```

---

## Directory Structure

```
.
├── cmd/
│   ├── server/          # Main entry point (main.go) -- wires everything
│   ├── migrate/         # DB migration runner
│   └── benchmark/       # Load testing tool with reservoir sampling
├── internal/
│   ├── audit/           # Audit logging (model, repository, service)
│   ├── auth/            # JWT/API-key auth, RBAC, interceptors, caching resolvers
│   ├── config/          # Config loading (DATABASE_URL from env)
│   ├── connect/         # Connect-RPC adapter (delegates to grpc servers)
│   ├── grpc/            # gRPC service implementations (handlers)
│   ├── health/          # Health check endpoint
│   ├── logger/          # Structured slog setup with OTel bridge
│   ├── namespace/       # Namespace CRUD (service, repository, models)
│   ├── observability/   # OTel setup (traces, metrics, logs via OTLP)
│   ├── tenant/          # Tenant + API key management, auto-provisioning
│   ├── ui/              # Embedded SvelteKit frontend (go:embed)
│   └── webhooks/
│       ├── client/      # HTTP client for webhook delivery (HMAC, redirects, SSL)
│       ├── queue/       # River job queue (EventWorker, WebhookWorker)
│       └── store/       # Repository (75+ methods), models, interface
├── pkg/
│   ├── errors/          # Error classification (9 categories, retryability)
│   ├── storage/         # DB/DBTX interfaces, WithTransaction, error translation
│   └── types/           # Shared types
├── proto/               # Generated protobuf Go + JS/TS code
│   ├── protoconnect/    # Generated Connect-RPC Go code
│   ├── webhook_pb.js    # protoc-gen-es output (web UI)
│   └── webhook_pb.d.ts  # protoc-gen-es types (web UI)
├── client/              # Generated clients (Go, JS, Python)
├── db/
│   └── migrations/      # 10 migration pairs (.up.sql / .down.sql)
├── web/                 # SvelteKit frontend source
├── buf.gen.yaml         # Buf code generation config (Go, JS/TS clients, protoc-gen-es for web UI)
├── buf.yaml             # Buf module config
├── webhook.proto        # Single proto file defining all 9 services
├── Dockerfile           # 3-stage: Node frontend -> Go build -> distroless
├── docker-compose.yml   # Postgres + migrate + sparrow + OTel collector
├── Makefile             # build, test, lint, migrate, generate, docker targets
└── go.mod / go.sum
```

---

## Package Dependency Graph

```
cmd/server/main.go
├── internal/config
├── internal/logger
├── internal/observability
├── internal/auth
├── internal/audit
├── internal/tenant
├── internal/namespace
├── internal/webhooks
│   ├── internal/webhooks/store
│   ├── internal/webhooks/queue
│   └── internal/webhooks/client
├── internal/grpc
├── internal/connect
├── internal/health
├── internal/ui
├── pkg/storage/postgres
└── proto / proto/protoconnect

internal/grpc ──────────> internal/audit
              ──────────> internal/auth
              ──────────> internal/webhooks (service interface)
              ──────────> internal/tenant
              ──────────> internal/namespace
              ──────────> pkg/storage
              ──────────> proto

internal/webhooks ──────> internal/auth
                  ──────> internal/webhooks/store
                  ──────> internal/webhooks/queue
                  ──────> internal/webhooks/client
                  ──────> internal/observability

internal/webhooks/queue ─> internal/webhooks/store
                         ─> internal/webhooks/client
                         ─> internal/observability
                         ─> pkg/errors

internal/webhooks/store ──> pkg/storage
                          ──> pkg/types

internal/tenant ──────────> internal/auth
                ──────────> pkg/storage

internal/namespace ───────> internal/auth
                   ───────> pkg/storage

internal/audit ───────────> internal/auth
               ───────────> pkg/storage

internal/auth ────────────> (leaf: no internal deps)
internal/observability ───> (leaf: only OTel externals)
pkg/storage ──────────────> (leaf: database/sql, sqlx)
pkg/errors ───────────────> (leaf: net, syscall)
```

---

## 9 gRPC/Connect-RPC Services

| Service | Server Struct | RPCs | File |
|---------|---------------|------|------|
| **WebhookService** | `WebhookServer` | RegisterWebhook, UnregisterWebhook, ListWebhooks, UpdateWebhookConfig, PauseWebhook, ResumeWebhook, GetNamespaceStats, GetTemplateFunctions | `webhook_handlers.go` |
| **EventService** | `WebhookServer` | RegisterEvent, ListEvents, UpdateEvent, DeleteEvent, GetEvent, PushEvent, ListEventReports | `event_handlers.go` |
| **SubscriptionService** | `WebhookServer` | CreateSubscription, GetSubscription, ListSubscriptions, UpdateSubscription, DeleteSubscription, TestSubscriptionTemplate | `subscription_handlers.go` |
| **DeliveryService** | `WebhookServer` | GetDeliveryStatus, ListDeliveries, RetryDelivery, GetDeliveryAttempts | `delivery_handlers.go` |
| **HealthService** | `WebhookServer` | GetWebhookHealth, ListWebhooksByHealth, GetHealthSummary | `health_handlers.go` |
| **TenantService** | `TenantServer` | CreateTenant, GetTenant, ListTenants, UpdateTenant, DeleteTenant | `tenant_server.go` |
| **APIKeyService** | `TenantServer` | CreateAPIKey, GetAPIKey, ListAPIKeys, RevokeAPIKey | `tenant_server.go` |
| **NamespaceService** | `NamespaceServer` | CreateNamespace, GetNamespace, ListNamespaces, UpdateNamespace, DeleteNamespace | `namespace_server.go` |
| **NamespaceMembershipService** | `NamespaceServer` | AssignNamespaceRole, RemoveNamespaceRole, ListNamespaceMembers, GetUserNamespaces | `namespace_server.go` |

**Serving**: gRPC on `:50051` (with reflection), Connect-RPC on `:8080` (with CORS + embedded UI).

---

## Authentication & Authorization

### Auth Flow
```
Request
  │
  ├─ Authorization: Bearer <JWT>  ──> JWTAuthenticator (JWKS validation)
  │                                    ├─ Extract org_id -> TenantLookup (cached 5min)
  │                                    ├─ Extract sub -> MembershipResolver (cached 30s)
  │                                    └─ Build AuthInfo
  │
  └─ Authorization: Bearer sk_*   ──> APIKeyAuthenticator
                                       ├─ SHA-256 hash -> APIKeyStore (cached 30s)
                                       └─ Build AuthInfo from stored key record
```

### AuthInfo (extracted per-request, stored in context)
```go
type AuthInfo struct {
    TenantID        uuid.UUID
    SubjectID       string             // JWT sub or ""
    KeyID           *uuid.UUID         // API key UUID or nil
    IsPlatformAdmin bool
    TenantRole      Role               // tenant:admin | tenant:member
    NamespaceRoles  map[string]Role    // namespace -> role
}
```

### 5 Roles
| Role | Scope |
|------|-------|
| `tenant:admin` | Full tenant access |
| `tenant:member` | Read + limited write |
| `namespace:admin` | Full namespace access |
| `namespace:member` | Read + write within namespace |
| `namespace:viewer` | Read-only namespace access |

### ~25 Permissions
Domains: `tenant`, `event_type`, `namespace`, `webhook`, `subscription`, `event`, `delivery`, `health`, `namespace_membership`

### Interceptors
- `NewGRPCUnaryInterceptor(cfg)` -- for native gRPC on :50051
- `NewConnectInterceptor(cfg)` -- for Connect-RPC on :8080
- Both share `authenticate()` helper; when auth disabled, inject `DefaultAuthInfo()`

---

## Data Flow: Event Push -> Delivery

```
PushEvent RPC
    │
    v
EventService.PushEvent()
    │  1. Validate payload against registered event schema
    │  2. Insert event_record
    │  3. Enqueue EventArgs job (River, "events" queue)
    v
EventProcessingWorker.Work()
    │  1. Load event from DB
    │  2. Query matching subscriptions (tenant_id + namespace + event_name)
    │  3. For each subscription: apply Go template transform (if enabled)
    │  4. Batch-insert all webhook_delivery records (single multi-row INSERT)
    │  5. Batch-enqueue all WebhookArgs jobs (River InsertMany)
    v
WebhookWorker.Work()
    │  1. Load delivery from DB
    │  2. HTTP POST to webhook URL (with headers, HMAC, timeout)
    │  3. Record delivery_attempt
    │  4. Update delivery status (success/failed/retrying)
    │  5. Update webhook_health_events + webhook_health_state
    │  6. If failed + retries remaining: re-enqueue with backoff
    v
Target URL receives webhook payload
```

---

## Repository / Storage Patterns

### Core Interfaces (`pkg/storage`)
```go
type DBTX interface {
    GetContext(ctx, dest, query, args...) error
    SelectContext(ctx, dest, query, args...) error
    NamedExecContext(ctx, query, arg) (sql.Result, error)
    ExecContext(ctx, query, args...) (sql.Result, error)
}

type DB interface {
    DBTX
    Ping() error
    Close() error
    Beginx() (*sqlx.Tx, error)
}
```

### WithConn Pattern (identical across all repos)
```go
type Repository struct {
    db   storage.DB    // original pool
    conn storage.DBTX  // current connection (pool or tx)
}

func (r *Repository) WithConn(conn storage.DBTX) *Repository {
    return &Repository{db: r.db, conn: conn}
}
```

### Transaction Helper
```go
storage.WithTransaction(db, func(tx storage.DBTX) error {
    // use repo.WithConn(tx) for transactional operations
    return nil
})
```

### SQL Error Translation (`pkg/storage/errors.go`)
| PostgreSQL Error | Semantic Error |
|------------------|---------------|
| `sql.ErrNoRows` | `ErrNotFound` |
| PG 23505 (unique_violation) | `ErrAlreadyExists` |
| PG 23502 (not_null_violation) | `ErrInvalidInput` |
| PG 23503 (foreign_key_violation) | `ErrForeignKeyViolation` |

---

## Error Classification (`pkg/errors`)

```go
type ErrorCategory string

const (
    CategorySuccess           = "success"
    CategoryClientError       = "client_error"      // 4xx
    CategoryServerError       = "server_error"       // 5xx -- RETRYABLE
    CategoryTimeout           = "timeout"            // RETRYABLE
    CategoryDNSError          = "dns_error"
    CategoryTLSError          = "tls_error"
    CategoryConnectionRefused = "connection_refused"  // RETRYABLE
    CategoryNetworkError      = "network_error"       // RETRYABLE
    CategoryUnknown           = "unknown"
)
```

**Retryable**: `server_error`, `timeout`, `connection_refused`, `network_error`  
**Not retryable**: `client_error`, `dns_error`, `tls_error`

---

## River Job Queue

### Queue Configuration
| Queue | Workers | Poll Interval |
|-------|---------|---------------|
| `default` | 5 | default |
| `events` | 20 | 2s |
| `webhooks` | 20 | 2s |

### Job Types
```go
// EventArgs -- kind: "event_processing", queue: "events"
type EventArgs struct {
    TenantID, EventID, Namespace, Event string
    TTLSeconds int
    Metadata   json.RawMessage
    CreatedAt  time.Time
}

// WebhookArgs -- kind: "webhook_delivery", queue: "webhooks"
type WebhookArgs struct {
    TenantID, DeliveryID, WebhookID, SubscriptionID, EventID string
    ExpiresAt  time.Time
    Namespace  string
    MaxAttempts int
}
```

**OTel propagation**: Trace context is serialized into `job.Metadata` as JSON `propagation.MapCarrier`.

---

## Audit Logging

### Actor Resolution
```
auth.AuthInfo
  ├─ KeyID != nil     -> ActorType: "api_key", ActorID: key UUID
  ├─ SubjectID != ""  -> ActorType: "user",    ActorID: JWT sub
  └─ fallback         -> ActorType: "system",  ActorID: tenant UUID
```

### 20 Actions Across 8 Resource Types
| Resource | Actions |
|----------|---------|
| `webhook` | `webhook.register`, `webhook.unregister`, `webhook.update`, `webhook.pause`, `webhook.resume` |
| `event` | `event.register`, `event.update`, `event.delete` |
| `subscription` | `subscription.create`, `subscription.update`, `subscription.delete` |
| `delivery` | `delivery.retry` |
| `tenant` | `tenant.create`, `tenant.update`, `tenant.delete` |
| `api_key` | `api_key.create`, `api_key.revoke` |
| `namespace` | `namespace.create`, `namespace.update`, `namespace.delete` |
| `membership` | `membership.assign`, `membership.remove` |

### Modes
- **`Log(ctx, entry)`** -- async fire-and-forget via goroutine + `context.WithoutCancel`
- **`LogSync(ctx, entry)`** -- synchronous, for use within transactions

---

## Dependency Injection (Manual, in `cmd/server/main.go`)

```go
// 1. Infrastructure
db := postgres.New(databaseURL)          // sqlx pool, MaxOpen=25
pgxPool := pgxpool.New(...)              // for River only

// 2. Repositories
webhookRepo := store.NewRepository(db)
tenantRepo  := tenant.NewRepository(db)
nsRepo      := namespace.NewRepository(db)
auditRepo   := audit.NewRepository(db)

// 3. OTel tracing wrappers (gowrap generated)
tracedRepo := store.NewRepositoryInterfaceWithTracing(webhookRepo, "sparrow.store")

// 4. Services
webhookSvc := webhooks.NewService(tracedRepo, jobInserter)
tenantSvc  := tenant.NewService(tenantRepo)
nsSvc      := namespace.NewService(nsRepo)
auditLogger := audit.NewLogger(auditRepo, slogLogger)

// 5. Auth
authenticators := []auth.Authenticator{jwtAuth, apiKeyAuth}

// 6. gRPC servers
webhookServer := grpc.NewWebhookServer(webhookSvc, auditLogger)
tenantServer  := grpc.NewTenantServer(tenantSvc, auditLogger)
nsServer      := grpc.NewNamespaceServer(nsSvc, auditLogger)
```

---

## Database Schema (13 Tables)

### Entity Relationship Diagram

```
                            ┌───────────┐
                            │  tenants  │
                            │───────────│
                            │ id (PK)   │
                            │ name      │
                            │ slug (UQ) │
                            │ status    │
                            │ external_id│
                            │ created_by│
                            └─────┬─────┘
           ┌──────────┬───────────┼──────────┬───────────┬────────────┐
           │          │           │          │           │            │
           v          v           v          v           v            v
    ┌──────────┐ ┌─────────┐ ┌────────┐ ┌─────────┐ ┌─────────┐ ┌──────────┐
    │ api_keys │ │namespace│ │ns_memb.│ │event_reg│ │event_rec│ │audit_logs│
    │──────────│ │─────────│ │────────│ │─────────│ │─────────│ │──────────│
    │ id (PK)  │ │ id (PK) │ │ id (PK)│ │tenant_id│ │ id (PK) │ │ id (PK)  │
    │ tenant_id│ │tenant_id│ │tenant_id│ │name(PK) │ │tenant_id│ │ tenant_id│
    │ key_hash │ │name(UQ) │ │subject │ │schema   │ │event    │ │ actor_id │
    │ role     │ │descript.│ │namespace│ │active   │ │payload  │ │ action   │
    │ expires  │ └────┬────┘ │role    │ └─────────┘ │namespace│ │ resource │
    └──────────┘      │      └────────┘              └────┬────┘ └──────────┘
                      │                                    │
                      v                                    │
              ┌───────────────┐                            │
              │ webhook_regs  │                            │
              │───────────────│                            │
              │ id (PK)       │                            │
              │ tenant_id     │                            │
              │ namespace ────┤─── FK to namespaces        │
              │ url           │    (tenant_id, name)       │
              │ active        │                            │
              │ health        │                            │
              │ webhook_secret│                            │
              └───────┬───────┘                            │
           ┌──────────┼──────────┬──────────┐              │
           │          │          │          │              │
           v          v          v          v              │
    ┌──────────┐┌──────────┐┌──────────┐┌──────────┐      │
    │event_sub ││deliveries││health_ev ││health_sum│      │
    │──────────││──────────││──────────││──────────│      │
    │ id (PK)  ││ id (PK)  ││ id (PK)  ││ id (PK)  │      │
    │webhook_id││webhook_id││webhook_id││webhook_id│      │
    │event_name││event_id──┤│──────────┤│──────────│      │
    │namespace ││subscr_id ││success   ││success_  │      │
    │transform ││status    ││resp_time ││  rate    │      │
    │template  ││attempts  ││error_cat ││p95_resp  │      │
    └──────────┘│request_  ││          │└──────────┘      │
                │  body    │└──────────┘                   │
                │error_cat │       ┌──────────┐            │
                └─────┬────┘       │health_st │            │
                      │            │──────────│            │
                      │            │webhook_id│            │
                      └────────────│consec_   │            │
                         FK        │ failures │            │
                       event_id────│last_succ │            │
                                   └──────────┘
```

### All Tables with Column Counts

| Table | Columns | Primary Key | Notable |
|-------|---------|-------------|---------|
| `tenants` | 8 | `id` | Slug unique, external_id partial unique |
| `api_keys` | 12 | `id` | FK tenant, key_hash indexed, role CHECK |
| `namespaces` | 6 | `id` | (tenant_id, name) UNIQUE |
| `namespace_memberships` | 7 | `id` | (tenant_id, subject_id, namespace) UNIQUE |
| `event_registrations` | 8 | `(tenant_id, name)` | Composite PK (no UUID) |
| `webhook_registrations` | 20 | `id` | (tenant_id, namespace) FK to namespaces |
| `event_subscriptions` | 11 | `id` | FK webhook_id, transform template |
| `event_records` | 9 | `id` | FK tenant, TTL + expires_at |
| `webhook_deliveries` | 16 | `id` | FK webhook+event+subscription, status enum |
| `webhook_health_events` | 9 | `id` | FK webhook, error_category |
| `webhook_health_summaries` | 17 | `id` | (webhook_id, window_start, window_end) UNIQUE |
| `webhook_health_state` | 8 | `id` | webhook_id UNIQUE |
| `audit_logs` | 11 | `id` | FK tenant, actor_type enum |

### Index Inventory (50+ indexes)

#### Hot-path composite indexes (added in migration 000010)
| Index | Columns | Purpose |
|-------|---------|---------|
| `idx_event_subscriptions_tenant_ns_event` | `(tenant_id, namespace, event_name)` | Fan-out query (~200 deliveries/min) |
| `idx_webhook_deliveries_webhook_created` | `(webhook_id, created_at DESC)` | Paginated delivery listing |
| `idx_webhook_deliveries_event_created` | `(event_id, created_at DESC)` | Delivery-by-event queries |
| `idx_event_records_tenant_ns_created` | `(tenant_id, namespace, created_at DESC)` | Event listing |
| `idx_event_records_tenant_event_created` | `(tenant_id, event, created_at DESC)` | Event name filtering |
| `idx_webhook_registrations_tenant_ns_active` | `(tenant_id, namespace, active)` | Filtered webhook listing |

#### Dropped (migration 000010)
| Index | Reason |
|-------|--------|
| `idx_webhook_deliveries_request_body_gin` | Unused GIN index, wasted write overhead |

### Foreign Key Cascade Map
```
tenants.id
  ├── api_keys.tenant_id                    CASCADE
  ├── namespaces.tenant_id                  CASCADE
  ├── namespace_memberships.tenant_id       CASCADE
  ├── event_registrations.tenant_id         CASCADE
  ├── webhook_registrations.tenant_id       CASCADE
  ├── event_subscriptions.tenant_id         CASCADE
  ├── event_records.tenant_id               CASCADE
  └── audit_logs.tenant_id                  CASCADE

namespaces(tenant_id, name)
  ├── webhook_registrations(tenant_id, ns)  CASCADE
  └── namespace_memberships(tenant_id, ns)  CASCADE

webhook_registrations.id
  ├── event_subscriptions.webhook_id        CASCADE
  ├── webhook_deliveries.webhook_id         CASCADE
  ├── webhook_health_events.webhook_id      CASCADE
  ├── webhook_health_summaries.webhook_id   CASCADE
  └── webhook_health_state.webhook_id       CASCADE

event_records.id
  └── webhook_deliveries.event_id           CASCADE

event_subscriptions.id
  └── webhook_deliveries.subscription_id    SET NULL
```

---

## Observability Stack

```
Sparrow Server
  │
  ├── Traces (OpenTelemetry)
  │     ├── gowrap-generated repository/service/queue tracing
  │     ├── gRPC interceptor tracing
  │     └── River job trace propagation (via job.Metadata)
  │
  ├── Metrics (OpenTelemetry)
  │     ├── webhook_registrations (counter)
  │     ├── events_pushed (counter)
  │     ├── webhook_deliveries (counter, by status)
  │     ├── delivery_duration (histogram)
  │     ├── queue_depth (gauge)
  │     └── active_webhooks (gauge)
  │
  └── Logs (slog + OTel bridge)
        └── JSON structured logging -> OTLP HTTP export
                │
                v
        ┌───────────────┐
        │ OTel Collector │ :4317 (gRPC) / :4318 (HTTP)
        └───────────────┘
```

---

## Configuration

### Environment Variables
| Variable | Purpose | Default |
|----------|---------|---------|
| `DATABASE_URL` | PostgreSQL connection string | required |
| `SPARROW_AUTH_ENABLED` | Enable JWT/API-key auth | `"false"` |
| `SPARROW_JWKS_URL` | JWKS endpoint for JWT validation | -- |
| `SPARROW_JWT_TENANT_CLAIM` | JWT claim for tenant/org ID | `"org_id"` |
| `SPARROW_JWT_ROLE_CLAIM` | JWT claim for role | `"org_role"` |
| `SPARROW_JWT_SUBJECT_CLAIM` | JWT claim for user identifier | `"sub"` |
| `SPARROW_JWT_ISSUER` | Expected JWT issuer | -- |
| `SPARROW_JWT_AUDIENCES` | Comma-separated expected audience values | -- |
| `SPARROW_JWT_NAMESPACE_ROLES_CLAIM` | JWT claim for namespace roles. Set to `""` to disable. | `"namespace_roles"` |
| `SPARROW_JWT_ROLE_MAPPING` | Comma-separated `provider_role=sparrow_role` pairs | `"org:admin=tenant:admin,org:member=tenant:member"` |
| `SPARROW_ROOT_API_KEY` | Pre-configured root API key for bootstrap | -- |
| `SPARROW_SERVE_UI` | Serve embedded SvelteKit UI | `"false"` |
| `CORS_ALLOWED_ORIGINS` | CORS origins for Connect-RPC | -- |
| `ENVIRONMENT` | `"development"` or `"production"` | -- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP HTTP export endpoint | -- |
| `CLERK_SECRET_KEY` | Clerk secret key for namespace role sync | -- |

### Database Pools
| Pool | Library | Config | Purpose |
|------|---------|--------|---------|
| sqlx | `jmoiron/sqlx` | MaxOpen=25 | All application queries |
| pgxpool | `jackc/pgx/v5` | MaxConns=50, MinConns=10, 30min lifetime, 5min idle, 30s health check | River job queue only |

---

## Deployment Topology

```
┌──────────────────────────────────────────────────────────────────┐
│                      docker compose                              │
├──────────────┬──────────────┬───────────────┬────────────────────┤
│   postgres   │   migrate    │    sparrow    │   otel-collector   │
│  :5432       │ (run-once)   │  :8080 :50051 │   :4317 :4318     │
│              │              │               │                    │
│ postgres:15  │ /app/tools/  │ /app/server   │ otel/contrib       │
│  -alpine     │   migrate    │               │                    │
│              │              │ Env:          │ Config:            │
│ DB: sparrow  │ Depends:     │  DATABASE_URL │  otel-collector-   │
│ User:sparrow │  postgres    │  SPARROW_*    │   config.yml       │
│              │  (healthy)   │               │                    │
│ Healthcheck: │              │ Depends:      │                    │
│  pg_isready  │ restart: no  │  migrate      │                    │
│  5s interval │              │  (completed)  │                    │
└──────────────┴──────────────┴───────────────┴────────────────────┘

Startup order: postgres (healthy) -> migrate (exits) -> sparrow (starts)
```

### Dockerfile (3-stage)
1. **`frontend`** (`node:22-alpine`): `npm run build` (SvelteKit static adapter)
2. **`builder`** (`golang:1.25-alpine`): Compiles `server` + `migrate` binaries with embedded UI
3. **Final** (`distroless/static-debian12:nonroot`): Minimal runtime, ports 50051 + 8080

---

## Web UI (SvelteKit)

**Stack**: SvelteKit 2, Svelte 5, Tailwind CSS v4, Vite 7, adapter-static (SPA mode)  
**Output**: `../internal/ui/dist` (embedded in Go binary via `go:embed`)  
**Auth**: Pluggable via `PUBLIC_AUTH_PROVIDER` env — `clerk` (Clerk SDK) or `none` (no auth)  
**Fonts**: Inter (`font-inter`) for marketing content, Fira Code (`font-fira`) for code/monospace  
**UI library**: flowbite-svelte  
**API layer**: Connect-RPC via `@connectrpc/connect-web`, protobuf types from `proto/webhook_pb.js`

### Route Structure
| Route | Access | Purpose |
|-------|--------|---------|
| `/` | Public | Marketing landing page (hero, features, getting started, architecture, CTA) |
| `/webhooks` | Auth required | Webhook list (default post-login page) |
| `/webhooks/register` | Auth required | Register new webhook |
| `/webhooks/[webhookId]` | Auth required | Webhook detail + deliveries |
| `/events` | Auth required | Event type list |
| `/events/push` | Auth required | Push event form |
| `/events/[eventName]/update` | Auth required | Edit event type |
| `/events/[eventName]/reports` | Auth required | Event delivery reports |
| `/health` | Auth required | Webhook health dashboard |
| `/team` | Auth required | Team/org management |
| `/deliveries` | Auth required | Delivery list |

### Auth Shell Hierarchy
```
+layout.svelte (defines public routes: ["/"])
  └── AuthShell.svelte (dispatches to provider)
        ├── ClerkAuthShell.svelte
        │     ├── <Show when="signed-in"> → OrgGate → nav + children
        │     └── <Show when="signed-out"> → SignInButton (forceRedirectUrl="/webhooks")
        └── NoAuthShell.svelte (no auth, renders nav + children directly)
```

### Post-login Redirect
- `OrgGate.svelte`: `afterSelectOrganizationUrl="/webhooks"`, `afterCreateOrganizationUrl="/webhooks"`
- `OrgGate.svelte`: org-switch reload → `window.location.href = "/webhooks"`
- `ClerkAuthShell.svelte`: `SignInButton` uses `forceRedirectUrl="/webhooks"`

---

## Makefile Targets

| Target | Description |
|--------|-------------|
| `build` | Build server binary |
| `build-ui` | Build frontend UI |
| `build-with-ui` | Build server + embedded UI |
| `build-all` | Cross-platform builds |
| `run` | Run server locally |
| `test` | Run tests |
| `migrate` | Run database migrations |
| `clean` | Remove build artifacts |
| `generate` | buf generate + go generate + docs |
| `lint` | Run linters (golangci-lint) |
| `docker-dev` | Docker compose dev environment |
| `docker-purge` | Purge docker dev resources |
| `fmt` | Format code |
| `example` | Run example |
| `run-web` | Run web UI dev server |

---

## Code Patterns & Conventions

### What to Check When Reviewing

1. **Auth checks first**: Every handler starts with `auth.MustFromContext(ctx)` then `info.Require(perm, namespace)`
2. **Error returns before success**: All error paths return before the happy path
3. **Audit after success**: `s.audit.Log(ctx, ...)` is called ONLY after successful mutations, never on error paths
4. **WithConn for transactions**: Use `repo.WithConn(tx)` inside `storage.WithTransaction` blocks
5. **No direct SQL in handlers**: All DB access goes through repository methods
6. **gRPC error translation**: Use `toGRPCError(ctx, err, msg)` or manual `status.Errorf` mapping
7. **Tenant scoping**: Every query must filter by `tenant_id` (multi-tenant isolation)
8. **Namespace scoping**: Where applicable, queries also filter by namespace
9. **Platform admin bypass**: `info.IsPlatformAdmin` bypasses tenant/namespace ownership checks
10. **OTel tracing wrappers**: Generated by gowrap, applied at DI time (not in business logic)

### Naming Conventions
- **Files**: `snake_case.go` (e.g., `webhook_handlers.go`, `tenant_server.go`)
- **Packages**: Lowercase single word (e.g., `audit`, `auth`, `namespace`)
- **Proto services**: PascalCase with `Service` suffix (e.g., `WebhookService`)
- **Actions**: `resource.verb` format (e.g., `webhook.register`, `tenant.delete`)
- **Roles**: `scope:level` format (e.g., `tenant:admin`, `namespace:viewer`)
- **API keys**: `sk_<tenant-slug>_<random>` prefix pattern

### Handler Structure Pattern
```go
func (s *XxxServer) DoSomething(ctx context.Context, req *pb.DoSomethingRequest) (*pb.DoSomethingResponse, error) {
    // 1. Auth check
    info := auth.MustFromContext(ctx)
    if err := info.Require(auth.PermXxx, ""); err != nil {
        return nil, status.Error(codes.PermissionDenied, err.Error())
    }

    // 2. Input validation
    id, err := uuid.Parse(req.GetId())
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid ID: %v", err)
    }

    // 3. Ownership/access check (if not platform admin)
    if !info.IsPlatformAdmin && ... {
        return nil, status.Error(codes.PermissionDenied, "...")
    }

    // 4. Business logic (service call)
    result, err := s.svc.DoSomething(ctx, ...)
    if err != nil {
        if storage.IsNotFound(err) {
            return nil, status.Errorf(codes.NotFound, "not found")
        }
        return nil, status.Errorf(codes.Internal, "do something: %v", err)
    }

    // 5. Audit log (async, fire-and-forget)
    s.audit.Log(ctx, audit.LogEntry{
        Action:       audit.ActionXxx,
        ResourceType: audit.ResourceXxx,
        ResourceID:   result.ID.String(),
        Namespace:    req.GetNamespace(),
        Metadata:     map[string]any{"key": "value"},
    })

    // 6. Return response
    return &pb.DoSomethingResponse{
        Result: resultToProto(result),
    }, nil
}
```

---

## Known Gaps / Not Yet Implemented

1. **No rate limiting** at the API level
2. **No dead letter queue** -- failed deliveries stay in deliveries table
3. **No integration/E2E tests**
4. **No API versioning** in proto
5. **No payload size limits** enforcement
6. **No tenant usage quotas** (events/month, deliveries, etc.)
7. **No scheduled/delayed webhooks**
8. **No AuditLogService RPC** -- audit logs are written but no query endpoint exists yet

---

## Development History

| Phase | Period | Key Work |
|-------|--------|----------|
| Foundation | Oct 2025 | gRPC + Connect-RPC, OTel tracing |
| Core | Oct-Nov 2025 | Webhooks, events, subscriptions, deliveries, SvelteKit UI |
| Observability | Nov 2025 | Metrics, sqlx migration, delivery retry, HMAC |
| Templating | Nov-Dec 2025 | Go template transforms, benchmarking tool |
| CI/Quality | Dec 2025-Jan 2026 | GitHub CI, golangci-lint, gowrap codegen |
| Proto Refactor | Feb 2026 | Split monolith service into 9 services, optimize DB |
| UI Redesign | Feb 2026 | Terminal aesthetic, namespace chooser, RBAC UI |
| Multi-Tenancy | Mar 2026 | Full RBAC, JWT/API-key auth, tenants, namespaces, audit logs |
| Proto Cleanup | Mar 2026 | Removed OpenAPI/Swagger, added Go/JS/Python gRPC client generation |
| Scaling & Audit | Mar 2026 | Batch fan-out (BatchCreateDeliveries), pgxpool tuning, audit logging for all 18 RPCs, 6 composite indexes |
| UI Modernization | Mar 2026 | Marketing landing page (light theme), Getting Started with real curl commands, post-login redirect to /webhooks, protoc-gen-es integration in buf.gen.yaml |
| Identity Provider | Mar 2026 | Pluggable IdentityProvider interface, Clerk namespace role sync (raw HTTP, no SDK), JWT namespace_roles claim extraction with DB fallback, CachingMembershipResolver (30s TTL), full env var configurability for self-hosted OIDC deployments |
