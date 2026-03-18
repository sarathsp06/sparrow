# Architecture Review: Package Structure & Design Philosophy

This document captures the architectural review of Sparrow's three core domain packages
(`internal/tenant`, `internal/namespace`, `internal/webhooks`), their interdependencies,
and an honest assessment of whether the current design is the right approach.

---

## Verdict

**The three-package separation is structurally correct — but it is masking serious
atomicity and performance problems that need to be fixed.**

The packages do not import each other. The dependency graph is acyclic. The `auth`
package cleanly decouples them. These are good things. Merging the packages would not
fix the real problems and would create new ones (god objects, circular dependencies).

**However, the current implementation has concrete bugs:**

1. Most multi-write operations within `webhooks` are **not transactional** — partial
   failures leave the database inconsistent.
2. Namespace delete/rename **orphans all webhook data** silently — no FK, no cascade,
   no application-level cleanup.
3. The JWT auth hot path makes **1 uncached DB query per request** for namespace
   memberships.
4. The connection pool is **unconfigured** (Go defaults: unlimited open, 2 idle).

These are not theoretical concerns. They are bugs in the current code. The rest of this
document explains the architecture, then details each problem with concrete failure
scenarios and fixes.

---

## 1. Dependency Graph (Actual)

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

**Critical observation:** `tenant`, `namespace`, and `webhooks` never import each other.
Their only shared dependency is `internal/auth`, which defines narrow interfaces that each
package implements independently. This is dependency inversion done right.

### How They Actually Connect

| Producer (implements) | Interface (in `auth`) | Consumer |
|---|---|---|
| `tenant.pgRepository` | `auth.APIKeyStore` | `auth.APIKeyAuthenticator` |
| `tenant.pgRepository` | `auth.TenantLookup` | `auth.CachingTenantResolver` |
| `tenant.AutoProvisioner` | `auth.TenantProvisioner` | `auth.CachingTenantResolver` |
| `namespace.pgRepository` | `auth.MembershipResolver` | `auth.JWTAuthenticator` |

All of this wiring happens in `cmd/server/main.go` — the only place that knows about all
three packages simultaneously. This is the composition root pattern.

---

## 2. Why Three Separate Packages Is Right

### 2.1 Different Lifecycles

- **Tenants** are long-lived administrative entities. They are created once (via bootstrap
  or auto-provisioning) and rarely change. The tenant package also manages API keys — a
  security domain that should not be mixed with webhook delivery logic.

- **Namespaces** are organizational units within a tenant. They have their own RBAC
  (membership assignments). The namespace package implements `auth.MembershipResolver`,
  meaning it is part of the authentication pipeline — namespace memberships are resolved
  on every JWT-authenticated request. Mixing this with webhook CRUD would couple security
  concerns with business logic.

- **Webhooks** are the core business domain: event registration, subscription management,
  delivery pipeline, health tracking, HTTP client, queue workers. This is ~7,700 lines
  across 27 files. It is correctly isolated as its own bounded context.

### 2.2 Different Access Patterns

| Package | Read/Write Ratio | Hot Path? | Transaction Scope |
|---|---|---|---|
| `tenant` | Mostly reads (cached) | Auth pipeline (every request) | Single-row |
| `namespace` | Mostly reads (membership resolution) | Auth pipeline (JWT requests) | Single-row |
| `webhooks` | High write volume (events, deliveries) | Event processing pipeline | Multi-row transactions |

Combining these into one repository would create a god object with ~80+ methods, where
auth-critical reads compete with high-throughput event writes for code attention and
review scrutiny.

### 2.3 Each Package Owns Its Schema

| Package | Tables Owned |
|---|---|
| `tenant` | `tenants`, `api_keys` |
| `namespace` | `namespaces`, `namespace_memberships` |
| `webhooks` | `webhook_registrations`, `event_registrations`, `event_subscriptions`, `event_records`, `webhook_deliveries`, `webhook_health_events`, `webhook_health_summaries`, `webhook_health_state` |

This per-package schema ownership is a healthy pattern. Each package knows its own tables
and queries. Changes to the webhook schema don't require touching the tenant repository.

---

## 3. The `auth` Package as Contract Layer

`internal/auth` is the architectural linchpin. It defines the interfaces and types that
decouple the three domain packages:

```
auth.Role                    — used by tenant (API key roles) and namespace (membership roles)
auth.Permission              — used by webhooks (service-level permission checks)
auth.AuthInfo                — carried in context, consumed by all three packages
auth.APIKeyStore             — implemented by tenant.pgRepository
auth.TenantLookup            — implemented by tenant.pgRepository
auth.TenantProvisioner       — implemented by tenant.AutoProvisioner
auth.MembershipResolver      — implemented by namespace.pgRepository
auth.Authorize()             — called by webhooks service methods
auth.AuthInfo.Require()      — called by webhooks service methods
auth.AuthInfo.CanAccessNamespace() — called by webhooks service methods
```

**This is the Dependency Inversion Principle in practice**: high-level policy (auth) defines
interfaces; low-level details (tenant store, namespace store) implement them.

### Invariant to Maintain

The `auth` package must NEVER import `tenant`, `namespace`, or `webhooks`. If you find
yourself wanting to add a tenant import to auth, you have a design problem — define an
interface in auth instead.

---

## 4. Tensions and Things to Watch

### 4.1 Namespace Delete/Rename Orphans All Webhook Data [BUG — CRITICAL]

The `webhook_registrations`, `event_subscriptions`, and `event_records` tables store
namespace as a plain TEXT string with **no foreign key** to the `namespaces` table.

**What happens on namespace delete** (`namespace/service.go:100-109`):
- Only `namespaces` and `namespace_memberships` rows are deleted (FK CASCADE).
- `webhook_registrations`, `event_subscriptions`, `event_records` with that namespace
  name are **left behind** — orphaned, invisible through normal API access.

**What happens on namespace rename** (`namespace/service.go:72-97`):
- `namespaces.name` is updated. Nothing else is updated.
- All webhook data still references the old name. It becomes unreachable.

**Fix options (pick one):**
1. **SQL FK constraint** (preferred): Add a composite FK from `webhook_registrations
   (tenant_id, namespace)` → `namespaces(tenant_id, name)` with `ON DELETE CASCADE`
   and `ON UPDATE CASCADE`. Requires a migration to backfill any namespace strings not
   yet registered in the `namespaces` table. Does not require Go code changes.
2. **Application-level cascade**: Have the namespace service call into a cleanup
   interface (implemented by webhooks store) on delete/rename. Requires a new interface
   in `auth` or a shared contracts package.
3. **Prevent namespace delete if webhooks exist**: Add a pre-delete check that queries
   webhook_registrations by (tenant_id, namespace) and refuses deletion if any exist.

### 4.2 Most Webhook Write Operations Are Not Transactional [BUG — CRITICAL]

This is NOT caused by the three-package split — these are all within the `webhooks`
package itself. No repository method (in any package) ever calls `db.Beginx()`.

**Affected operations:**

| Operation | What Breaks on Partial Failure | File |
|---|---|---|
| `RegisterWebhook` | Webhook created, 2 of 5 subscriptions created. Subscription errors are logged and swallowed (`continue`). | `webhook_service.go:162-182` |
| `CreateWebhook` | Same as above. | `webhook_service.go:302-324` |
| `UpdateWebhookConfig` | Old subscriptions deleted, new ones partially created. Webhook can end up with **zero subscriptions**. | `webhook_service.go:1530-1551` |
| `PushEvent` | Event stored in `event_records`, but River job insertion fails. Event is permanently lost from the processing pipeline. (`StoreEventTx` exists but is not used here.) | `webhook_service.go:551-568` |
| `RetryDelivery` | Delivery status reset to "pending", but job not enqueued. Delivery is permanently stuck. | `webhook_service.go:740-883` |
| `EventProcessingWorker` | Delivery record created via `w.store`, but River job created in a separate pgx tx. Not in the same transaction. | `queue/events_worker.go:113-130` |
| `WebhookWorker` | Delivery updated, health event recorded, health state upserted — three separate non-transactional writes. | `queue/webhook_worker.go:59-282` |

**Fix:** Add a `WithTx` or `RunInTransaction` method to the `storage.DB` interface and
use it to wrap multi-write operations. The `Beginx()` method already exists on the
interface but is never called. Priority should be:
1. `PushEvent` — use `StoreEventTx` which already exists but isn't being called.
2. `UpdateWebhookConfig` — the delete-all-then-recreate pattern is dangerous without a tx.
3. `RegisterWebhook` / `CreateWebhook` — wrap webhook + subscription creation.
4. `EventProcessingWorker` — delivery + job insertion should share the River tx.

### 4.3 JWT Auth: 1 Uncached DB Query Per Request [PERFORMANCE]

The JWT authentication path (`jwt.go:253-254`) calls `ResolveNamespaceMemberships` on
every request. This queries `namespace_memberships WHERE tenant_id = $1 AND subject_id = $2`.

**There is no cache.** Tenant resolution has a 5-minute cache. JWKS keys have a 1-hour
cache. API key lookups have a 30-second cache. But namespace memberships hit the DB
every single time.

| Auth Type | Cache State | Sync DB Queries Per Request |
|---|---|---|
| API Key | Cache hit | **0** |
| API Key | Cache miss | **1** |
| JWT (no memberships) | Tenant cached | **0** |
| JWT (with memberships) | Tenant cached | **1** (memberships, always) |
| JWT (with memberships) | Both miss | **2** (tenant + memberships) |

**Fix:** Add a `CachingMembershipResolver` wrapper (analogous to `CachingTenantResolver`)
keyed on `(tenantID, subjectID)` with a 30-60 second TTL. Namespace membership changes
are rare enough to tolerate short-lived caching.

**Bonus optimization:** On a full JWT cache miss, the two sequential queries (tenant
lookup + membership resolution) could be combined into a single SQL query with a CTE:

```sql
WITH t AS (SELECT id FROM tenants WHERE external_id = $1 AND status = 'active')
SELECT nm.namespace, nm.role
FROM namespace_memberships nm JOIN t ON nm.tenant_id = t.id
WHERE nm.subject_id = $2;
```

This would reduce the JWT cold-path from 2 round-trips to 1. However, this requires a
cross-repository query, which would need a dedicated "auth resolution" helper rather
than combining the two existing repositories.

### 4.4 API Key: Write-Per-Request for `last_used_at` [PERFORMANCE]

Every API key request (cache hit or miss) spawns a goroutine that runs:
```sql
UPDATE api_keys SET last_used_at = $1 WHERE id = $2
```
(`authenticator.go:131-136`, `authenticator.go:182-186`)

At high throughput, this creates one write per request. Consider debouncing: only update
`last_used_at` at most once per N seconds per key using a buffered channel or in-memory
timestamp check.

### 4.5 Connection Pool Is Unconfigured [PERFORMANCE]

`cmd/server/main.go:92` calls `postgres.Open(databaseURL, 3)` with no pool options.
Go's `database/sql` defaults apply: **unlimited open connections, 2 idle connections**.

The `pkg/storage/postgres/sqlx.go` file defines `WithMaxOpenConnections`,
`WithMaxIdleConnections`, etc. — but they are **never used**.

With 3 repositories + auth lookups + River workers all sharing the same pool with only
2 idle connections, every burst of concurrent requests pays connection setup costs.

**Fix:** Pass pool options in `main.go`:
```go
sqlxDB, err := postgres.Open(databaseURL, 3,
    postgres.WithMaxOpenConnections(25),
    postgres.WithMaxIdleConnections(10),
    postgres.WithConnectionMaxLifeTime(30 * time.Minute),
    postgres.WithSetConnMaxIdleTime(5 * time.Minute),
)
```

### 4.6 Event Registrations Are Tenant-Scoped, Not Namespace-Scoped [OK]

Unlike webhooks and subscriptions, `event_registrations` have `tenant_id` but no
`namespace` column. Event types are shared across all namespaces within a tenant. This is
intentional — event types (like "order.created") are a tenant-wide vocabulary, while
*subscriptions* to those events are namespace-specific.

This is the right design: it avoids duplicating event type definitions across namespaces
and allows one event to fan out to subscriptions in multiple namespaces.

### 4.7 Two Database Connection Pools [OK]

The server creates two database connection pools:
- `pgxpool` (for River queue, which requires pgx natively)
- `sqlx` wrapped around pgx (for all repository operations)

All three repositories share the same `sqlx` pool — there is zero pool fragmentation.
Having separate repository structs adds no overhead (3 extra pointer-sized structs).

### 4.8 Webhook Package Size [OK]

At ~7,700 lines across 27 files, the `webhooks` package is large but internally well-
structured with clear sub-packages:

```
webhooks/           — service layer (business logic, validation, API models)
webhooks/store/     — data access (repository pattern, SQL queries)
webhooks/queue/     — async processing (River workers)
webhooks/client/    — HTTP delivery (request building, templating, signing)
```

Each sub-package has a single responsibility. Breaking `webhooks` into more top-level
packages (e.g., `internal/events`, `internal/deliveries`) would create circular
dependencies because events, subscriptions, deliveries, and webhooks are tightly
coupled in the processing pipeline.

---

## 5. Design Philosophy (Principles to Follow)

### 5.1 Bounded Contexts with Interface Contracts

Each domain package (tenant, namespace, webhooks) is a bounded context. They communicate
through interfaces defined in `internal/auth`, not through direct imports. New cross-
package interactions should follow this pattern: define the interface in `auth` (or a new
shared `contracts` package), implement it in the producing package, consume it in the
other.

### 5.2 Composition Root in `main.go`

`cmd/server/main.go` is the only file that imports all three domain packages. It
constructs repositories, services, and wires them together. This is where dependency
injection happens. No framework — just constructor functions and explicit wiring.

### 5.3 Repository Per Domain, Not Per Table

Each domain owns a `Repository` interface and `pgRepository` implementation. The
repository encapsulates all SQL for that domain's tables. This is the correct granularity.
Resist the urge to create per-table repositories (too granular) or a single shared
repository (too broad).

### 5.4 Auth Context as the Communication Channel

The three packages don't call each other's methods at runtime. Instead, the auth
pipeline builds an `AuthInfo` struct (containing tenant ID, role, namespace memberships)
and stores it in the request context. Service methods extract this context and enforce
authorization independently.

```
Request → Auth Interceptor → [tenant repo: validate API key or JWT]
                            → [namespace repo: resolve memberships]
                            → AuthInfo injected into context
                            → Service method extracts AuthInfo
                            → Service enforces permissions
                            → Repository queries scoped by tenant_id + namespace
```

This means the runtime coupling between packages flows through context values, not
function calls.

### 5.5 Schema Ownership

Each migration that adds tables should document which package owns the new tables.
The mapping is:

| Migration | Tables | Owning Package |
|---|---|---|
| 000001 | webhook_registrations, event_registrations, event_subscriptions, event_records, webhook_deliveries, webhook_health_* | `internal/webhooks` |
| 000004 | tenants, api_keys | `internal/tenant` |
| 000008 | namespaces, namespace_memberships | `internal/namespace` |

### 5.6 No "Smart" Base Packages

There is no shared "models" package, no shared "repository" package, no ORM. Each
package defines its own models matching its own SQL schemas. The only shared
infrastructure is `pkg/storage` (DB abstraction) and `pkg/types` (generic utilities
like `Map`). This avoids the "shared models" anti-pattern where changing one domain's
model breaks another.

---

## 6. Fix Priority (Ordered by Impact)

| Priority | Issue | Risk | Fix Effort |
|---|---|---|---|
| **P0** | `PushEvent` not using `StoreEventTx` — events silently lost | Silently lost events | Small — method exists, just not called |
| **P0** | `UpdateWebhookConfig` delete-then-recreate without tx | Webhook with zero subscriptions | Medium — wrap in `Beginx()` transaction |
| **P1** | Namespace delete/rename orphans webhook data | Silent data inconsistency | Medium — add FK migration or app-level cascade |
| **P1** | `RegisterWebhook`/`CreateWebhook` partial subscriptions | Webhook misses events silently | Medium — wrap in transaction |
| **P1** | `EventProcessingWorker` delivery + job not in same tx | Permanently stuck deliveries | Medium — use River tx for both |
| **P2** | No cache on namespace membership resolution | 1 extra DB query per JWT request | Small — add caching wrapper |
| **P2** | Connection pool unconfigured (2 idle conns) | Connection churn under load | Trivial — pass options to `Open()` |
| **P3** | `last_used_at` write per API key request | Write amplification at high RPS | Small — debounce with timestamp check |
| **P3** | Cache eviction is O(n) with write lock | Brief latency spike at >10k entries | Small — replace with LRU |

---

## 7. When to Reconsider the Package Structure

- **If namespace lifecycle must cascade to webhooks:** Add FK constraints at the DB
  level (preferred) or an application-level cascade interface. Do NOT have webhooks
  import namespace.
- **If you need multi-package transactions:** Create a coordinator in a new package
  that imports both, or use database-level constraints.
- **If the webhooks package exceeds ~15,000 lines:** Consider extracting the health
  tracking subsystem into its own package.
- **If you add more domains (e.g., billing, audit log):** Follow the same pattern —
  own package, own repository, own tables, communicate through `auth` interfaces.

---

## 8. Summary Dependency Matrix

| Package | Imports `tenant`? | Imports `namespace`? | Imports `webhooks`? | Imports `auth`? |
|---|---|---|---|---|
| `tenant` | — | No | No | Yes |
| `namespace` | No | — | No | Yes |
| `webhooks` | No | No | — | Yes |
| `auth` | No | No | No | — |
| `grpc` | Yes (transport) | Yes (transport) | Yes (transport) | Yes |
| `main.go` | Yes (wiring) | Yes (wiring) | Yes (wiring) | Yes |

There are zero cycles. This is the mark of a healthy Go codebase.
