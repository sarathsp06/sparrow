# Sparrow Implementation Plan

> This plan captures all design decisions made during planning. Implementation should follow this document. Update it as decisions change.

> **This plan must be evaluated and updated frequently.** After completing any part, re-assess priorities, update statuses, and revise remaining items based on what was learned. Stale plans lead to wasted effort.

## Design Principles

These principles apply globally to Sparrow, not just this feature set:

1. **Deterministic bulk operations** -- All bulk actions (re-push events, retry deliveries) use a snapshot-based batch pattern. When a user searches with filters and opts into a bulk action, the matching IDs are snapshotted into a `batch_jobs` row at query time. The bulk action operates on that snapshot, NOT a live re-query. This guarantees what-you-see = what-you-act-on. No race conditions from new data arriving between search and action.

2. **Soft validation over hard rejection** -- Schema validation produces warnings, not errors. Events are always accepted and stored. Invalid payloads are tagged (`schema_valid=false`), not discarded. The caller gets per-field validation messages as warnings in the response.

3. **Graceful degradation** -- When a non-critical step fails (e.g., Go template transform), fall back to a safe default (envelope payload) rather than failing the entire operation. Log a warning, continue delivery.

4. **Generic infrastructure over per-feature tables** -- Shared concerns (batch jobs, future: scheduled jobs, etc.) use generic tables with `job_type` + JSONB `data` columns. Each job type defines its own data schema within JSONB.

5. **Implicit infrastructure, explicit actions** -- Batch jobs are an implementation detail. Users see "re-push ID" and "retry ID", not "batch job IDs". The batch mechanism is invisible; the user's mental model is: search -> act on results.

6. **Postgres-only, no Redis** -- Sparrow's operational simplicity is a competitive advantage over Svix. All queuing, state, and caching uses PostgreSQL (via River). Do not introduce Redis or other external dependencies without strong justification.

7. **Self-hosted first** -- Features should be useful for teams running Sparrow behind a VPN for internal webhook delivery. Multi-tenant SaaS features are lower priority than operational excellence.

8. **Explicit and verbose execution** -- When performing tasks (especially multi-step ones like UI redesigns, refactors, or debugging), narrate every action clearly: what is being done, why, what the expected outcome is, and what was actually observed. This applies to both human and AI-assisted development. Silent changes lead to confusion; over-communication is preferred.

---

## Current State (as of v1.2.1)

### What's Implemented

| Area | Status | Details |
|------|--------|---------|
| Core webhook pipeline | Complete | Register, subscribe, push, fan-out, deliver, retry |
| 5 proto services, 34 RPCs | Complete | Webhook, Event, Subscription, Delivery, Health + Go-only Namespace |
| Dual protocol (gRPC + Connect-RPC) | Complete | :50051 gRPC, :8080 HTTP/Connect |
| SvelteKit admin UI | Complete | 13 pages: webhooks, events, deliveries, health, event instances |
| Go template transforms | Complete | Per-subscription payload transformation with caching |
| HMAC-SHA256 signing | Complete | `X-Sparrow-Signature-256` + `X-Sparrow-Timestamp` |
| Ed25519 signing | Complete | Dual signing (HMAC + Ed25519) on every delivery, per-webhook keypair, public key via API |
| SSRF protection | Complete | Blocks private/loopback/metadata IPs, validates redirects |
| Envelope encryption | Complete | AES-256-GCM for webhook secrets + secret headers |
| Health tracking | Complete | State machine (healthy/degraded/unhealthy), rolling summaries |
| Soft schema validation | Complete | Warnings not errors, `schema_valid` flag on events |
| Batch re-push/retry | Complete | Deterministic snapshot-based, up to 10K items, async via River |
| Idempotency keys | Complete | Optional dedup on PushEvent, partial unique index |
| Per-webhook rate limiting | Complete | Leaky bucket (`rate_limit_rps`), 429 Retry-After parsing |
| Error classification | Complete | 10 categories including `rate_limited`, retryability flags |
| API key auth | Complete | Optional `SPARROW_API_KEY`, HTTP + gRPC, constant-time compare |
| Security headers | Complete | nosniff, DENY framing, strict referrer, no FLoC |
| OTel observability | Complete | Traces, metrics, logs, job trace propagation, gowrap wrappers |
| 22 DB migrations | Complete | 11 tables including `webhook_rate_limit_state` |
| Helm chart | Complete | `charts/sparrow/` |
| CI/CD + GoReleaser | Complete | Cross-platform binaries, Helm chart artifact |

### Known Gaps (vs Svix and general best practices)

| Gap | Priority | Notes |
|-----|----------|-------|
| No dead letter queue | High | Failed deliveries stuck in deliveries table forever |
| No API-level rate limiting | Medium | Per-webhook rate limiting exists, but API endpoints are unthrottled |
| No CLI tool | Medium | Users rely on grpcurl / curl |
| No payload size limits | Medium | Unbounded event payloads |
| No data retention / cleanup | High | No TTL-based purge of old events/deliveries |
| No scheduled/delayed webhooks | Low | Not in Svix OSS either |
| No API versioning | Low | Single proto, no v1/v2 namespacing |
| Limited client SDKs | Medium | Go/JS/Python generated; no Java/Ruby/C#/PHP |
| `opencode.md` stale | High | Several inaccuracies (see Part 10) |

---

## Completed Parts (v0.8.0 -- v1.2.1)

### Part 1: Soft Schema Validation + Template Fallback

**Status**: Complete (v0.8.0)

- Migration 000016: `schema_valid BOOLEAN` on `event_records`
- `PushEvent` always stores, sets `schema_valid=false` + warnings on mismatch
- `WebhookWorker` falls back to envelope payload on template failure

### Part 2: Search Filters

**Status**: Complete (v0.8.0)

- `ListEventReports`: schema_valid, labels, date range, `prepare_repush` opt-in
- `ListDeliveries`: status, error_category, subscription, date range, `prepare_retry` opt-in

### Part 3: Single Event Re-push

**Status**: Complete (v0.9.0)

- `EventService.RePushEvent`: replay original event through current schema

### Part 4: Deterministic Batch Operations

**Status**: Complete (v0.8.0)

- Migration 000017: `batch_jobs` table
- `BatchJobWorker` (queue: `batch_jobs`, 5 workers, 5s poll)
- RPCs: RePushEvents, GetRepushStatus, CancelRepush, RetryDeliveries, GetRetryStatus, CancelRetry
- 10K item cap, 15min TTL, cancellation checks every 25 items

### Part 5: Delivery Retry by Filter

**Status**: Complete (v0.8.0)

- Covered by Part 2 filters + Part 4 batch retry

### Part 6: Web UI

**Status**: Complete (v0.9.0)

- Event reports page with filters + bulk re-push
- Deliveries page with filters + bulk retry
- Webhook detail page with delivery filters

### Part 7: Docs Fixes

**Status**: Complete (v0.11.2)

### Part 8: Idempotency Keys on PushEvent

**Status**: Complete (v1.1.1)

- Migration 000020: `idempotency_key` column + partial unique index
- `PushEventResponse.duplicate` flag
- RePushEvent bypasses dedup (nil key)

### Part 9: Per-Webhook Rate Limiting

**Status**: Complete (v1.2.1)

- Migration 000021: `rate_limit_rps` on `webhook_registrations` + `webhook_rate_limit_state` table
- Leaky bucket algorithm in `WebhookWorker` via `AcquireDeliverySlot`
- HTTP 429 `Retry-After` header parsing (seconds and HTTP-date formats, capped at 15min)
- `rate_limited` error category (retryable) in `pkg/errors`

---

## Part 10: Docs Sync

**Status**: Pending

The `opencode.md` file has drifted from the actual codebase. Fix all inaccuracies:

- Rate limiting IS implemented (remove from "Known Gaps", add to features)
- 21 migrations, not 20; 11 tables, not 10
- 34 proto RPCs across 5 services (update RPC counts in service table)
- Error categories: 10 not 9 (add `rate_limited`)
- UI routes: remove `/namespaces` (doesn't exist), add `/events/register`, `/events/instances/[eventId]`, `/deliveries/[deliveryId]`
- Queue config: add `batch_jobs` queue (5 workers, 5s poll)
- Add SecurityHeaders middleware to middleware section
- Update Known Gaps to reflect current state

---

## Part 11: Dead Letter Queue

**Status**: Pending

**Priority**: High -- currently failed deliveries accumulate forever with no mechanism to surface, redrive, or purge them.

### Design

After all retry attempts are exhausted, a delivery enters `failed` terminal state. Today it sits in `webhook_deliveries` indefinitely. A DLQ mechanism should:

1. **Surface failed deliveries clearly** -- dedicated `ListDLQ` RPC filtered to terminal-failed deliveries past max attempts
2. **Redrive** -- `RedriveDLQ` RPC: re-enqueue failed deliveries for another round of attempts (resets attempt counter)
3. **Bulk redrive** -- reuse existing batch infrastructure (`batch_jobs` with `job_type = "dlq_redrive"`)
4. **Per-webhook DLQ depth metric** -- OTel gauge `sparrow.dlq.depth` by webhook_id
5. **UI** -- DLQ tab on webhook detail page + global DLQ view

### Why not a separate table?

Failed deliveries already have all the data (payload, headers, target URL, error info). A separate DLQ table would duplicate storage. Instead, use a filtered view: `status = 'failed' AND attempts >= max_attempts`.

### Migration

- Add index: `idx_webhook_deliveries_dlq ON webhook_deliveries (webhook_id, status) WHERE status = 'failed'`

### Proto Changes

- `DeliveryService.ListDLQ(namespace, webhook_id, page_size, page_token)` -> paginated failed deliveries
- `DeliveryService.RedriveDLQ(namespace, webhook_id)` -> redrive all failed for a webhook
- `DeliveryService.RedriveDelivery(delivery_id)` -> redrive single delivery

---

## Part 12: Data Retention & Cleanup

**Status**: Pending

**Priority**: High -- without cleanup, tables grow unbounded. Event records, deliveries, health events, and batch jobs all need TTL-based purging.

### Design

River periodic (cron) job that runs cleanup on a configurable schedule.

### Configuration

New env vars:
- `SPARROW_RETENTION_EVENTS_DAYS` -- default 30. Delete event_records older than N days.
- `SPARROW_RETENTION_DELIVERIES_DAYS` -- default 30. Delete webhook_deliveries + delivery attempts older than N days.
- `SPARROW_RETENTION_HEALTH_EVENTS_DAYS` -- default 7. Delete webhook_health_events older than N days.
- `SPARROW_RETENTION_BATCH_JOBS_DAYS` -- default 1. Delete completed/cancelled/expired batch_jobs older than N days.

### Implementation

- New `RetentionWorker` as a River periodic job (runs every hour)
- Deletes in batches (1000 rows per DELETE) to avoid long-held locks
- Logs rows deleted per table per run
- OTel counter: `sparrow.retention.rows_deleted` by table

### Migration

- Add `created_at` indexes on tables that lack them for efficient range deletes

---

## Part 13: Payload Size Limits

**Status**: Pending

**Priority**: Medium

### Design

- New config: `SPARROW_MAX_PAYLOAD_BYTES` (default: 256KB)
- Enforced in `PushEvent` service method before DB insert
- Returns `codes.InvalidArgument` with clear message
- Also enforced on template output (transformed payload)

### Proto Changes

None -- enforcement is server-side only.

---

## Part 14: Asymmetric Webhook Signatures (Ed25519)

**Status**: Complete

### Design (as implemented)

- **Dual signing**: Every delivery is signed with both HMAC-SHA256 and Ed25519. No `signature_type` configuration needed.
- **Ed25519 keypair generated once** at webhook registration, private key envelope-encrypted (AES-256-GCM) and stored in `ed25519_private_key` column.
- **Public key derived at runtime** from the private key (`ed25519.PrivateKey.Public()`). Not stored separately.
- **Signing**: `Ed25519Sign(privateKey, timestamp + "." + payload)`, same message format as HMAC.
- **Headers**: `X-Sparrow-Signature-Ed25519` (hex-encoded) alongside existing `X-Sparrow-Signature-256` and `X-Sparrow-Timestamp`.
- **Public key exposed** via `signing_public_key` field on `RegisterWebhookResponse` and `RegisteredWebhook` proto messages (hex-encoded).
- **Consumers choose** which signature to verify -- HMAC (requires shared secret) or Ed25519 (requires only the public key).

### Migration

- 000022: Add `ed25519_private_key BYTEA` column to `webhook_registrations`

### Key decisions vs original plan

| Original plan | Actual implementation | Rationale |
|---|---|---|
| `signature_type` column | No column; always dual-sign | Simpler, no config needed, negligible cost |
| `signing_public_key` column | Derived at runtime | Public key is always derivable from private key |
| Store private key in `webhook_secret` | Separate `ed25519_private_key` column | Different values; HMAC secret and Ed25519 key coexist |

---

## Part 15: CLI Tool

**Status**: Pending

**Priority**: Medium -- currently users need grpcurl or curl for all operations. A dedicated CLI improves DX significantly.

### Design

Standalone Go binary (`cmd/cli/`) using cobra. Connects to Sparrow server via Connect-RPC (HTTP).

### Commands

```
sparrow-cli webhooks list [--namespace NS]
sparrow-cli webhooks register --url URL --namespace NS [--secret SECRET]
sparrow-cli webhooks pause ID
sparrow-cli webhooks resume ID
sparrow-cli events list [--namespace NS]
sparrow-cli events register --name NAME [--schema FILE]
sparrow-cli events push --name NAME --namespace NS --payload FILE|STDIN
sparrow-cli deliveries list [--status STATUS] [--webhook-id ID]
sparrow-cli deliveries retry ID
sparrow-cli health summary [--namespace NS]
sparrow-cli config  # show server connection info
```

### Configuration

- `SPARROW_URL` env var (default: `http://localhost:8080`)
- `SPARROW_API_KEY` env var (reuse existing)
- Optional `~/.sparrow.yaml` config file

---

## Part 16: API-Level Rate Limiting

**Status**: Pending

**Priority**: Medium -- per-webhook delivery rate limiting exists, but API endpoints have no request throttling. Important for public-facing deployments.

### Design

- Token bucket per API key (or per-IP if no API key)
- Configurable via `SPARROW_API_RATE_LIMIT` (requests/second, default: 100)
- Returns HTTP 429 with `Retry-After` header
- Implemented as chi middleware (HTTP) + gRPC interceptor
- State stored in-memory (process-local) -- acceptable for single-instance deployments
- For multi-instance: optional PostgreSQL-backed limiter using `pg_advisory_lock`

---

## Future Considerations (Not Planned)

These are features identified from the Svix comparison that are **not currently planned** but documented for future reference:

| Feature | Rationale for deferring |
|---------|------------------------|
| Multi-tenant SaaS mode | Sparrow is designed for self-hosted internal use. Tenant infrastructure exists in the DB but activating it requires auth, isolation, and billing -- a different product. |
| Consumer app portal (embeddable UI) | Requires multi-tenancy. Sparrow's UI is admin-only. |
| FIFO endpoint ordering | Niche requirement. River doesn't support strict ordering without serialization. Would require per-endpoint queue partitioning. |
| Polling endpoints | Low demand for self-hosted use. Adds complexity to delivery model. |
| Object storage sinks (S3/GCS) | Specialized. Can be done via Go template + custom subscription type later. |
| Connector endpoints (Slack, etc.) | Better handled by users via template transforms + Slack webhook URLs. |
| Email notifications on failure | Requires email infrastructure (SMTP config, templates). Could add later with a webhook-to-email bridge pattern. |
| Operational webhooks (meta-webhooks) | Low priority for single-tenant. Could use existing subscription mechanism to subscribe to internal events. |
| More client SDKs (Java, Ruby, C#, PHP) | Generated clients work but are thin. Could auto-generate from proto using buf + connect-es ecosystem. Prioritize when there's user demand. |

---

## Implementation Order

```
Completed:
  Part 1 --> Part 2 --> Part 4 --> Part 6
                    \
                     Part 3 (parallel with Part 4)
  Part 7 (parallel)
  Part 8 (parallel)
  Part 9 (parallel)

Next:
  Part 10 (docs sync) -- immediate, no code changes
  Part 11 (DLQ) ──────> Part 12 (retention) -- DLQ first so retention doesn't delete un-redriven failures
  Part 13 (payload limits) -- independent
  Part 14 (Ed25519) -- COMPLETE
  Part 15 (CLI) -- independent
  Part 16 (API rate limiting) -- independent
```

---

## Decisions Log

| Question | Decision | Rationale |
|----------|----------|-----------|
| Batch storage | DB table (`batch_jobs`) | Persistent, horizontal-scaling safe, consistent with River |
| Batch TTL | 15 minutes | Generous for UI workflows |
| Batch size cap | 10,000 items | Balance of power vs safety |
| Batch execution | Async via River job | Large batches can't block HTTP |
| Delivery retry model | Batch-based (same as events) | Consistency, determinism |
| Batch on every list? | Opt-in via `prepare_*` flag | Avoid wasted rows |
| Batch scope | All matching IDs (not just page) | "Re-push all" means all |
| Batch table design | Generic `job_type` + JSONB `data` | Extensible for future batch types |
| Batch visibility | Implicit (repush_id/retry_id) | User sees actions, not infrastructure |
| Idempotency key column | Separate `idempotency_key` (nullable VARCHAR) | String flexibility, not coupled to event UUID |
| Re-push dedup | RePushEvent passes nil key | Re-push must never be deduplicated |
| Idempotency index | Partial unique index (WHERE NOT NULL) | No overhead for events without idempotency keys |
| Rate limiting algorithm | Leaky bucket (DB-backed) | Simple, predictable, no Redis needed |
| DLQ approach | Filtered view, not separate table | Failed deliveries already have all data; avoid duplication |
| Retention strategy | River periodic job, batched deletes | Avoids long locks, configurable per table |
| Ed25519 key storage | Reuse envelope encryption | Consistent with existing secret storage pattern |
| Ed25519 signing model | Always dual-sign (HMAC + Ed25519) | No config needed, negligible cost, consumer chooses which to verify |
| Ed25519 public key storage | Derived at runtime from private key | One fewer column, public key always derivable |
| CLI transport | Connect-RPC over HTTP | Same protocol as web UI, reuses API key auth |
| API rate limiting state | In-memory (single instance) | Simplest. Upgrade to PG-backed if multi-instance needed |
| Redis dependency | No | Postgres-only is a competitive advantage over Svix OSS |
| Multi-tenant activation | Deferred | Different product; infrastructure retained but not activated |
| Consumer portal | Deferred | Requires multi-tenancy; admin UI serves current use case |
