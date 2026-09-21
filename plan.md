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
| REST/OpenAPI API | Complete | Webhook, Event, Subscription, Delivery, Health resources, versioned under `/v1`; spec generated from Go (Huma), interactive docs at `/docs` (Scalar) |
| SvelteKit admin UI | Complete | Webhooks, events, deliveries, health, event instances |
| Go template transforms | Complete | Per-subscription payload transformation with caching |
| Standard Webhooks signing | Complete | `webhook-id`, `webhook-timestamp`, `webhook-signature` with `v1,`/`v1a,` base64 format |
| Ed25519 signing | Complete | Opt-in via `signature_type: ed25519` (adds `v1a,` alongside HMAC `v1,`), per-webhook keypair, hex public key via API |
| Signature verification helpers | Complete | Go `pkg/signature`, Python + TS/JS verifiers in `client/verify/`, documented in README "Verifying Webhook Signatures" |
| SSRF protection | Complete | Blocks private/loopback/metadata IPs, validates redirects |
| Envelope encryption | Complete | AES-256-GCM for webhook secrets + secret headers, with KEK keyring rotation support |
| Health tracking | Complete | State machine (healthy/degraded/unhealthy), rolling summaries |
| Soft schema validation | Complete | Warnings not errors, `schema_valid` flag on events |
| Batch re-push/retry | Complete | Deterministic snapshot-based, up to 10K items, async via River |
| Idempotency keys | Complete | Optional dedup on PushEvent, partial unique index |
| Per-webhook rate limiting | Complete | Leaky bucket (`rate_limit_rps`), 429 Retry-After parsing |
| Error classification | Complete | Categorized with retryability flags, including `rate_limited` |
| API key auth | Complete | Optional `SPARROW_API_KEY`, constant-time compare |
| Security headers | Complete | nosniff, DENY framing, strict referrer, no FLoC |
| OTel observability | Complete | Traces, metrics, logs, job trace propagation, gowrap wrappers |
| DB migrations | Complete | Automated on startup, see `db/migrations/` |
| CI/CD + GoReleaser | Complete | Cross-platform binaries, Docker image |
| Data retention | Complete | `RetentionWorker` purges `event_records` (cascades to deliveries) via `SPARROW_EVENT_RETENTION_DAYS` |
| CLI tool | Complete | `satellites/sparrow`, separate Go module, REST over HTTP |

### Known Gaps (vs Svix and general best practices)

| Gap | Priority | Notes |
|-----|----------|-------|
| No scheduled/delayed webhooks | Low | Not in Svix OSS either |
| Limited client SDKs | Medium | Python generated from OpenAPI; no Go/JS/Java/Ruby/C#/PHP |
| OKF bundle | Complete | `okf/` generated with 47 concepts across 58 files, 0 errors |

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

## Part 10: OKF Bundle + Docs Sync

**Status**: Complete

Generated OKF knowledge bundle at `okf/` with 47 concepts across 58 files covering architecture, packages, services, domain concepts, database schema, configuration, frontend, and DevOps.

Condensed `opencode.md` from 872 to 129 lines — removed sections fully covered by OKF (DB schema, River queue, error classification, observability, deployment, Makefile, etc.), kept unique content (design principles, code patterns, known gaps, dev history).

Updated `plan.md` inaccuracies: RPC count 34→36, migrations 22→23, gap table.

---

## Part 11: Dead Letter Queue

**Status**: Rejected -- not needed

`status = 'failed'` is already a permanent, queryable, filterable, retryable
terminal state on `webhook_deliveries` (`ListDeliveries?status=failed`,
single `retry`, bulk `prepare_retry`+`retryBatch`). A DLQ as a distinct
concept exists in queue systems (SQS, Kafka) to solve two problems Sparrow
doesn't have: a poison message blocking the queue behind it, and no
persistent storage for give-ups. Deliveries are independent rows, not a
blocking queue, and failed deliveries are already fully retained and
actionable. There is no second mechanism to build here -- "DLQ" would just
be a new name for data and endpoints that already exist.

---

## Part 12: Data Retention & Cleanup

**Status**: Complete

- `RetentionWorker` (`internal/webhooks/queue/retention_worker.go`), a River periodic job, `RunOnStart: true`, runs hourly
- `SPARROW_EVENT_RETENTION_DAYS` (default `0` = disabled) purges `event_records` older than N days; deliveries cascade via FK
- Logs rows deleted per run

`webhook_health_events` and `batch_jobs` are not purged by this worker --
not treated as a gap: health events roll up into summaries and batch jobs
already expire after a 15min TTL, so neither has been observed to grow
large. Extend the same worker with config-gated deletes if that changes.

---

## Part 13: Payload Size Limits

**Status**: Rejected -- already covered

Payloads are already hard-capped at the HTTP layer: Huma enforces its 1 MiB
per-operation body limit on every `/v1` operation (no handler overrides it),
and `SPARROW_MAX_BODY_BYTES` (default 5 MiB, min 1 MiB) backstops the whole
router via `http.MaxBytesReader`. Oversized pushes get a `413` before any
service code runs. A separate `SPARROW_MAX_PAYLOAD_BYTES` check inside
`PushEvent` would be a second knob enforcing a stricter copy of the same
limit. Template output needs no cap either: Go `text/template` cannot loop
unboundedly, so transformed output is bounded by template size x input size,
and both are already bounded.

---

## Part 14: Asymmetric Webhook Signatures (Ed25519)

**Status**: Complete

### Design (as implemented)

- **Opt-in per webhook**: `signature_type` selects the scheme -- `hmac` (default) signs HMAC-SHA256 only; `ed25519` adds an Ed25519 `v1a,` signature alongside the HMAC `v1,` one. (Originally shipped as unconditional dual signing; a `signature_type` column was added later.)
- **Ed25519 keypair generated once** at webhook registration, private key envelope-encrypted (AES-256-GCM) and stored in `ed25519_private_key` column.
- **Public key derived at runtime** from the private key (`ed25519.PrivateKey.Public()`). Not stored separately.
- **Signing**: Standard Webhooks format. Message: `{msg_id}.{timestamp}.{payload}`, HMAC = `v1,<base64>`, Ed25519 = `v1a,<base64>`.
- **Headers**: `webhook-id`, `webhook-timestamp`, `webhook-signature` (space-delimited signatures).
- **Public key exposed** via the `signing_public_key` field on the webhook resource (hex-encoded).
- **Consumers choose** which signature to verify -- HMAC (requires shared secret, `v1,` prefix) or Ed25519 (requires only the public key, `v1a,` prefix).

### Migration

- 000022: Add `ed25519_private_key BYTEA` column to `webhook_registrations`

### Key decisions vs original plan

| Original plan | Actual implementation | Rationale |
|---|---|---|
| `signature_type` column | Initially no column (always dual-sign); `signature_type` added later to make Ed25519 opt-in | Simpler default, Ed25519 only where consumers verify it |
| `signing_public_key` column | Derived at runtime | Public key is always derivable from private key |
| Store private key in `webhook_secret` | Separate `ed25519_private_key` column | Different values; HMAC secret and Ed25519 key coexist |

---

## Part 15: CLI Tool

**Status**: Complete

Ships as `satellites/sparrow`, a separate Go module (see `docs/adr/0002-cli-module-split.md`), released with its own path-prefixed tags. Cobra-based, subcommands: `init`, `push`, `events`, `webhooks`, `tail`, `stats`, `listen`, `use` (recipes), `template`, `functions`, `version`. REST over HTTP against the same API, reusing API key auth. Connection config resolves env > flags > `~/.sparrow/config.yaml` > defaults.

---

## Part 16: API-Level Rate Limiting

**Status**: Complete

- `internal/middleware/rate_limit.go`: token bucket (`golang.org/x/time/rate`,
  burst = rps) per client -- API key when present, else remote IP
- `SPARROW_API_RATE_LIMIT` (requests/second, default `0` = disabled; the
  planned default of 100 was dropped -- self-hosted deployments shouldn't get
  surprise throttling from an upgrade)
- Over-limit requests get `429` + `Retry-After: 1`
- Mounted on the `/v1` group only -- `/health`, `/ready`, UI, and portal
  gateway are exempt
- State is in-memory and process-local — correct fit for the single-container
  Docker Compose deployment model this app ships (no Helm/k8s replicas)

## Part 17: REST/OpenAPI Migration (removed gRPC + Connect-RPC)

**Status**: Complete

Replaced the dual gRPC (`:50051`) + Connect-RPC (HTTP/JSON on `:8080`) transport with a single REST/OpenAPI interface, generated from Go via [Huma](https://github.com/danielgtaylor/huma). Deleted `proto/`, `internal/grpc/`, `internal/connect/`, `buf.yaml`/`buf.gen.yaml`, and the standalone Astro docs site (`docs/`).

- **Handlers**: `internal/rest/` — one file per resource (`webhook.go`, `event.go`, `subscription.go`, `delivery.go`, `health.go`), all versioned under `/v1`.
- **Spec**: exported from Go via `cmd/openapi-export`, committed at `api/openapi.{yaml,json}`; `internal/rest/openapi_drift_test.go` fails CI if it drifts from the handlers.
- **Docs**: interactive reference at `/docs` (Scalar), replacing the old Astro/Starlight site and its GitHub Pages deployment.
- **Web UI**: rewritten to call the REST API via `openapi-fetch`, typed from a generated `api-types.d.ts` (replacing the Connect-RPC generated client).
- **Client SDKs**: only the Python client is still generated (`client/python/`, via `openapi-python-client`); the old Go/JS Connect-RPC clients were deleted with no REST replacement generated yet (tracked in Known Gaps).
- **e2e/integration tests**: `e2e/libs/sparrow_api.py` rewritten to REST paths; some retry/batch/fan-out integration coverage that existed against Connect-RPC was not yet ported (see `internal/integration/e2e_retry_test.go`).

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
| More client SDKs (Java, Ruby, C#, PHP) | Could auto-generate from the committed OpenAPI spec (`api/openapi.yaml`) with the respective language's OpenAPI generator, same as the Python client. Prioritize when there's user demand. |

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
  Part 10 (docs sync + OKF bundle)
  Part 14 (Ed25519 signing)
  Part 17 (REST/OpenAPI migration)
  Part 12 (retention)
  Part 15 (CLI tool)
  Part 16 (API rate limiting)

Rejected:
  Part 11 (DLQ) -- status='failed' already is the DLQ
  Part 13 (payload limits) -- covered by Huma 1 MiB per-op + SPARROW_MAX_BODY_BYTES
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
| DLQ approach | Rejected -- no separate concept needed | `status='failed'` is already permanent, queryable, retryable; deliveries aren't a blocking queue, so DLQ's isolation rationale doesn't apply |
| Retention strategy | River periodic job, batched deletes | Avoids long locks, configurable per table |
| Ed25519 key storage | Reuse envelope encryption | Consistent with existing secret storage pattern |
| Ed25519 signing model | Always dual-sign (HMAC + Ed25519) | No config needed, negligible cost, consumer chooses which to verify |
| Ed25519 public key storage | Derived at runtime from private key | One fewer column, public key always derivable |
| CLI tool | Separate module (`satellites/sparrow`), REST over HTTP | Own release cadence, reuses API key auth, same protocol as web UI |
| API rate limiting state | In-memory (single instance) | Correct fit for the actual deployment model: single-container Docker Compose, no Helm/k8s replicas. Revisit only if a multi-instance topology is added |
| Payload size limits | Rejected -- already covered | Huma's 1 MiB per-op body limit + `SPARROW_MAX_BODY_BYTES` already bound payloads at the HTTP layer; template output is bounded by template x input |
| API rate limit default | `0` (disabled) instead of planned 100 rps | Self-hosted deployments shouldn't get surprise throttling from an upgrade; opt-in for public-facing setups |
| Redis dependency | No | Postgres-only is a competitive advantage over Svix OSS |
| Multi-tenant activation | Deferred | Different product; infrastructure retained but not activated |
| Consumer portal | Deferred | Requires multi-tenancy; admin UI serves current use case |
