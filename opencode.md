# Sparrow -- Complete Codebase Reference

> **Keep this file up to date.** Whenever a relevant change is made to the codebase -- new features, architectural changes, schema migrations, configuration changes, or dependency updates -- update the corresponding section(s) in this file so it remains a reliable single-source-of-truth for AI-assisted development.

**Module**: `github.com/sarathsp06/sparrow`  
**Go**: 1.25  
**Repository**: `/Users/sarathsadasivanpillai/projects/httpqueue`

---

## Architecture Overview

Sparrow is a **self-hosted webhook delivery platform** with an event-driven architecture. It accepts webhook registrations, event definitions, and subscriptions (with Go-template payload transformation), then fans out events into a River job queue for async HTTP delivery with retries, health tracking, and error classification.

There is **no authentication or authorization layer** -- all endpoints are open. A default tenant is auto-provisioned on startup and used for all operations.

```
                         ┌─────────────────────┐
     Clients             │    Sparrow Server    │
  ┌──────────┐           │                      │
  │ gRPC     │──:50051──>│  internal/grpc       │──┐
  │ clients  │           │  (6 services)        │  │
  └──────────┘           │                      │  │
  ┌──────────┐           │  internal/connect    │  │  ┌──────────────┐
  │ HTTP/    │──:8080───>│  (Connect-RPC        │  ├─>│ Service Layer│
  │ Connect  │           │   adapter)           │  │  │ webhooks.Svc │
  └──────────┘           │                      │  │  │ namespace.Svc│
  ┌──────────┐           │  internal/ui         │  │  └──────┬───────┘
  │ Browser  │──:8080───>│  (SvelteKit embed)   │  │         │
  └──────────┘           └──────────────────────┘  │         v
                                                   │  ┌──────────────┐
                                                   │  │  Repository  │
                                                   │  │  (sqlx/DBTX) │
                                                   │  └──────┬───────┘
                                                   │         │
                                                   │         v
                         ┌─────────────────────┐   │  ┌──────────────┐
                         │  internal/webhooks/  │   │  │  PostgreSQL  │
                         │  queue (River)       │<──┘  │  10 tables   │
                         │  EventWorker         │      │  11 migrations│
                         │  WebhookWorker       │      └──────────────┘
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
│   ├── config/          # Config loading (DATABASE_URL from env)
│   ├── connect/         # Connect-RPC adapter (delegates to grpc servers)
│   ├── grpc/            # gRPC service implementations (handlers)
│   ├── health/          # Health check endpoint
│   ├── logger/          # Structured slog setup with OTel bridge
│   ├── namespace/       # Namespace CRUD (service, repository, models)
│   ├── observability/   # OTel setup (traces, metrics, logs via OTLP)
│   ├── tenant/          # Tenant bootstrap (default tenant auto-creation)
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
│   └── migrations/      # 11 migration pairs (.up.sql / .down.sql)
├── web/                 # SvelteKit frontend source
├── buf.gen.yaml         # Buf code generation config (Go, JS/TS clients, protoc-gen-es for web UI)
├── buf.yaml             # Buf module config
├── webhook.proto        # Single proto file defining all 6 services
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

internal/grpc ──────────> internal/webhooks (service interface)
              ──────────> internal/namespace
              ──────────> internal/tenant
              ──────────> pkg/storage
              ──────────> proto

internal/webhooks ──────> internal/webhooks/store
                  ──────> internal/webhooks/queue
                  ──────> internal/webhooks/client
                  ──────> internal/observability

internal/webhooks/queue ─> internal/webhooks/store
                         ─> internal/webhooks/client
                         ─> internal/observability
                         ─> pkg/errors

internal/webhooks/store ──> pkg/storage
                          ──> pkg/types

internal/tenant ──────────> pkg/storage

internal/namespace ───────> pkg/storage

internal/observability ───> (leaf: only OTel externals)
pkg/storage ──────────────> (leaf: database/sql, sqlx)
pkg/errors ───────────────> (leaf: net, syscall)
```

---

## 6 gRPC/Connect-RPC Services

| Service | Server Struct | RPCs | File |
|---------|---------------|------|------|
| **WebhookService** | `WebhookServer` | RegisterWebhook, UnregisterWebhook, ListWebhooks, UpdateWebhookConfig, PauseWebhook, ResumeWebhook, GetNamespaceStats, GetTemplateFunctions | `webhook_handlers.go` |
| **EventService** | `WebhookServer` | RegisterEvent, ListEvents, UpdateEvent, DeleteEvent, GetEvent, PushEvent, ListEventReports | `event_handlers.go` |
| **SubscriptionService** | `WebhookServer` | CreateSubscription, GetSubscription, ListSubscriptions, UpdateSubscription, DeleteSubscription, TestSubscriptionTemplate | `subscription_handlers.go` |
| **DeliveryService** | `WebhookServer` | GetDeliveryStatus, ListDeliveries, RetryDelivery, GetDeliveryAttempts | `delivery_handlers.go` |
| **HealthService** | `WebhookServer` | GetWebhookHealth, ListWebhooksByHealth, GetHealthSummary | `health_handlers.go` |
| **NamespaceService** | `NamespaceServer` | CreateNamespace, GetNamespace, ListNamespaces, UpdateNamespace, DeleteNamespace | `namespace_server.go` |

**Serving**: gRPC on `:50051` (with reflection), Connect-RPC on `:8080` (with CORS + embedded UI).

---

## Tenant Model

There is no authentication. A **default tenant** (`00000000-0000-0000-0000-000000000001`) is auto-created on startup via `tenant.Bootstrap()`. All operations use this tenant ID. The tenant infrastructure (table, columns, foreign keys) is retained for future use.

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

## Dependency Injection (Manual, in `cmd/server/main.go`)

```go
// 1. Infrastructure
db := postgres.New(databaseURL)          // sqlx pool, MaxOpen=25
pgxPool := pgxpool.New(...)              // for River only

// 2. Bootstrap default tenant
tenant.Bootstrap(db)

// 3. Repositories
webhookRepo := store.NewRepository(db)
nsRepo      := namespace.NewRepository(db)

// 4. OTel tracing wrappers (gowrap generated)
tracedRepo := store.NewRepositoryInterfaceWithTracing(webhookRepo, "sparrow.store")

// 5. Services
webhookSvc := webhooks.NewService(tracedRepo, jobInserter)
nsSvc      := namespace.NewService(nsRepo)

// 6. gRPC servers
webhookServer := grpc.NewWebhookServer(webhookSvc)
nsServer      := grpc.NewNamespaceServer(nsSvc)
```

---

## Database Schema (10 Tables)

### Entity Relationship Diagram

```
                            ┌───────────┐
                            │  tenants  │
                            │───────────│
                            │ id (PK)   │
                            │ name      │
                            │ slug (UQ) │
                            │ status    │
                            └─────┬─────┘
           ┌──────────────────────┼──────────────────────┐
           │                      │                      │
           v                      v                      v
    ┌─────────┐            ┌─────────┐            ┌─────────┐
    │namespace│            │event_reg│            │event_rec│
    │─────────│            │─────────│            │─────────│
    │ id (PK) │            │tenant_id│            │ id (PK) │
    │tenant_id│            │name(PK) │            │tenant_id│
    │name(UQ) │            │schema   │            │event    │
    │descript.│            │active   │            │payload  │
    └────┬────┘            └─────────┘            │namespace│
         │                                        └────┬────┘
         v                                             │
  ┌───────────────┐                                    │
  │ webhook_regs  │                                    │
  │───────────────│                                    │
  │ id (PK)       │                                    │
  │ tenant_id     │                                    │
  │ namespace ────┤─── FK to namespaces                │
  │ url           │    (tenant_id, name)               │
  │ active        │                                    │
  │ health        │                                    │
  │ webhook_secret│                                    │
  └───────┬───────┘                                    │
       ┌──┼──────────┬──────────┐                      │
       │  │          │          │                      │
       v  v          v          v                      │
┌──────────┐  ┌──────────┐  ┌──────────┐              │
│event_sub │  │health_ev │  │health_sum│              │
│──────────│  │──────────│  │──────────│              │
│ id (PK)  │  │ id (PK)  │  │ id (PK)  │              │
│webhook_id│  │webhook_id│  │webhook_id│              │
│event_name│  │success   │  │success_  │              │
│namespace │  │resp_time │  │  rate    │              │
│transform │  │error_cat │  │p95_resp  │              │
│template  │  └──────────┘  └──────────┘              │
└──────────┘                                           │
       │          ┌──────────┐                         │
       │          │health_st │                         │
       │          │──────────│                         │
       │          │webhook_id│                         │
       │          │consec_   │                         │
       │          │ failures │                         │
       │          │last_succ │                         │
       │          └──────────┘                         │
       v                                               │
┌──────────┐                                           │
│deliveries│                                           │
│──────────│                                           │
│ id (PK)  │                                           │
│webhook_id│                                           │
│event_id──┤───────────────────────────────────────────┘
│subscr_id │
│status    │
│attempts  │
│error_cat │
└──────────┘
```

### All Tables with Column Counts

| Table | Columns | Primary Key | Notable |
|-------|---------|-------------|---------|
| `tenants` | 5 | `id` | Slug unique, default tenant auto-created |
| `namespaces` | 6 | `id` | (tenant_id, name) UNIQUE |
| `event_registrations` | 8 | `(tenant_id, name)` | Composite PK (no UUID) |
| `webhook_registrations` | 20 | `id` | (tenant_id, namespace) FK to namespaces |
| `event_subscriptions` | 11 | `id` | FK webhook_id, transform template |
| `event_records` | 9 | `id` | FK tenant, TTL + expires_at |
| `webhook_deliveries` | 16 | `id` | FK webhook+event+subscription, status enum |
| `webhook_health_events` | 9 | `id` | FK webhook, error_category |
| `webhook_health_summaries` | 17 | `id` | (webhook_id, window_start, window_end) UNIQUE |
| `webhook_health_state` | 8 | `id` | webhook_id UNIQUE |

### Index Inventory

#### Hot-path composite indexes (added in migration 000010)
| Index | Columns | Purpose |
|-------|---------|---------|
| `idx_event_subscriptions_tenant_ns_event` | `(tenant_id, namespace, event_name)` | Fan-out query (~200 deliveries/min) |
| `idx_webhook_deliveries_webhook_created` | `(webhook_id, created_at DESC)` | Paginated delivery listing |
| `idx_webhook_deliveries_event_created` | `(event_id, created_at DESC)` | Delivery-by-event queries |
| `idx_event_records_tenant_ns_created` | `(tenant_id, namespace, created_at DESC)` | Event listing |
| `idx_event_records_tenant_event_created` | `(tenant_id, event, created_at DESC)` | Event name filtering |
| `idx_webhook_registrations_tenant_ns_active` | `(tenant_id, namespace, active)` | Filtered webhook listing |

### Foreign Key Cascade Map
```
tenants.id
  ├── namespaces.tenant_id                  CASCADE
  ├── event_registrations.tenant_id         CASCADE
  ├── webhook_registrations.tenant_id       CASCADE
  ├── event_subscriptions.tenant_id         CASCADE
  └── event_records.tenant_id               CASCADE

namespaces(tenant_id, name)
  └── webhook_registrations(tenant_id, ns)  CASCADE

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
| `SPARROW_SERVE_UI` | Serve embedded SvelteKit UI | `"false"` |
| `CORS_ALLOWED_ORIGINS` | CORS origins for Connect-RPC | -- |
| `ENVIRONMENT` | `"development"` or `"production"` | -- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP HTTP export endpoint | -- |

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
**Fonts**: Inter (`font-inter`) for marketing content, Fira Code (`font-fira`) for code/monospace  
**UI library**: flowbite-svelte  
**API layer**: Connect-RPC via `@connectrpc/connect-web`, protobuf types from `proto/webhook_pb.js`

### Route Structure
| Route | Purpose |
|-------|---------|
| `/` | Marketing landing page (hero, features, getting started, architecture, CTA) |
| `/webhooks` | Webhook list |
| `/webhooks/register` | Register new webhook |
| `/webhooks/[webhookId]` | Webhook detail + deliveries |
| `/events` | Event type list |
| `/events/push` | Push event form |
| `/events/[eventName]/update` | Edit event type |
| `/events/[eventName]/reports` | Event delivery reports |
| `/health` | Webhook health dashboard |
| `/namespaces` | Namespace management |
| `/deliveries` | Delivery list |

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

1. **Error returns before success**: All error paths return before the happy path
2. **WithConn for transactions**: Use `repo.WithConn(tx)` inside `storage.WithTransaction` blocks
3. **No direct SQL in handlers**: All DB access goes through repository methods
4. **gRPC error translation**: Use `toGRPCError(ctx, err, msg)` or manual `status.Errorf` mapping
5. **Tenant scoping**: Every query must filter by `tenant_id` (using `tenant.DefaultTenantID`)
6. **Namespace scoping**: Where applicable, queries also filter by namespace
7. **OTel tracing wrappers**: Generated by gowrap, applied at DI time (not in business logic)

### Naming Conventions
- **Files**: `snake_case.go` (e.g., `webhook_handlers.go`, `namespace_server.go`)
- **Packages**: Lowercase single word (e.g., `namespace`, `tenant`)
- **Proto services**: PascalCase with `Service` suffix (e.g., `WebhookService`)

### Handler Structure Pattern
```go
func (s *WebhookServer) DoSomething(ctx context.Context, req *pb.DoSomethingRequest) (*pb.DoSomethingResponse, error) {
    // 1. Use default tenant
    tenantID := tenant.DefaultTenantID

    // 2. Input validation
    id, err := uuid.Parse(req.GetId())
    if err != nil {
        return nil, status.Errorf(codes.InvalidArgument, "invalid ID: %v", err)
    }

    // 3. Business logic (service call)
    result, err := s.svc.DoSomething(ctx, tenantID, ...)
    if err != nil {
        if storage.IsNotFound(err) {
            return nil, status.Errorf(codes.NotFound, "not found")
        }
        return nil, status.Errorf(codes.Internal, "do something: %v", err)
    }

    // 4. Return response
    return &pb.DoSomethingResponse{
        Result: resultToProto(result),
    }, nil
}
```

---

## Known Gaps / Not Yet Implemented

1. **No authentication/authorization** -- all endpoints are open
2. **No rate limiting** at the API level
3. **No dead letter queue** -- failed deliveries stay in deliveries table
4. **No integration/E2E tests**
5. **No API versioning** in proto
6. **No payload size limits** enforcement
7. **No tenant usage quotas** (events/month, deliveries, etc.)
8. **No scheduled/delayed webhooks**

---

## Development History

| Phase | Period | Key Work |
|-------|--------|----------|
| Foundation | Oct 2025 | gRPC + Connect-RPC, OTel tracing |
| Core | Oct-Nov 2025 | Webhooks, events, subscriptions, deliveries, SvelteKit UI |
| Observability | Nov 2025 | Metrics, sqlx migration, delivery retry, HMAC |
| Templating | Nov-Dec 2025 | Go template transforms, benchmarking tool |
| CI/Quality | Dec 2025-Jan 2026 | GitHub CI, golangci-lint, gowrap codegen |
| Proto Refactor | Feb 2026 | Split monolith service into 8 services, optimize DB |
| UI Redesign | Feb 2026 | Terminal aesthetic, namespace chooser |
| Multi-Tenancy | Mar 2026 | Tenants, namespaces, batch fan-out, pgxpool tuning, 6 composite indexes |
| UI Modernization | Mar 2026 | Marketing landing page, Getting Started with curl commands, protoc-gen-es |
| Auth Removal | Mar 2026 | Removed all auth/RBAC/audit code, simplified to open self-hosted deployment |
