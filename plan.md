# Sparrow Implementation Plan: Search, Filter, Re-push & Retry

> This plan captures all design decisions made during planning. Implementation should follow this document. Update it as decisions change.

## Design Principles

These principles apply globally to Sparrow, not just this feature set:

1. **Deterministic bulk operations** -- All bulk actions (re-push events, retry deliveries) use a snapshot-based batch pattern. When a user searches with filters and opts into a bulk action, the matching IDs are snapshotted into a `batch_jobs` row at query time. The bulk action operates on that snapshot, NOT a live re-query. This guarantees what-you-see = what-you-act-on. No race conditions from new data arriving between search and action.

2. **Soft validation over hard rejection** -- Schema validation produces warnings, not errors. Events are always accepted and stored. Invalid payloads are tagged (`schema_valid=false`), not discarded. The caller gets per-field validation messages as warnings in the response.

3. **Graceful degradation** -- When a non-critical step fails (e.g., Go template transform), fall back to a safe default (envelope payload) rather than failing the entire operation. Log a warning, continue delivery.

4. **Generic infrastructure over per-feature tables** -- Shared concerns (batch jobs, future: scheduled jobs, etc.) use generic tables with `job_type` + JSONB `data` columns. Each job type defines its own data schema within JSONB.

5. **Implicit infrastructure, explicit actions** -- Batch jobs are an implementation detail. Users see "re-push ID" and "retry ID", not "batch job IDs". The batch mechanism is invisible; the user's mental model is: search -> act on results.

---

## Part 1: Soft Schema Validation + Template Fallback

**Status**: Not started

### Migration 000016: `add_schema_valid`
- Add `schema_valid BOOLEAN NOT NULL DEFAULT true` to `event_records`

### Proto Changes
- `PushEventResponse`: add `repeated string warnings = 4`
- `EventReport`: add `bool schema_valid = 12`

### Service Changes
- `PushEvent` signature: `(eventID string, warnings []string, err error)`
- Always stores event regardless of schema match
- Sets `schema_valid=false` + populates per-field warnings on validation failure
- Remove hard rejection path (currently returns InvalidArgument)

### Repository Changes
- Add `schema_valid` to all event INSERT/SELECT queries
- Fix `StoreEventTx` missing `labels` column (latent bug from migration 000012)

### gRPC Handler Changes
- Wire `warnings` from service into `PushEventResponse`
- Remove `SchemaValidationError` -> `codes.InvalidArgument` special case in event_handlers.go

### Worker Changes
- `WebhookWorker`: on template transform failure, log warning, fall back to `BuildEnvelopePayload`, continue delivery (do NOT return error to River)

---

## Part 2: Search Filters

**Status**: Not started

### ListEventReports Enhancements
Request fields:
- `optional bool schema_valid` -- filter by validation status
- `map<string, string> labels` -- JSONB containment filter (`labels @> ?`)
- `google.protobuf.Timestamp created_after` / `created_before` -- time range
- `bool prepare_repush` -- opt-in: snapshot matching IDs into batch_jobs, return repush_id

Response fields:
- `string repush_id` -- populated only when `prepare_repush=true`

### ListDeliveries Enhancements
Request fields:
- `optional string status` -- filter by delivery status
- `optional string error_category` -- filter by error classification
- `optional string subscription_id` -- filter by subscription
- `google.protobuf.Timestamp created_after` / `created_before`
- `bool prepare_retry` -- opt-in: snapshot matching IDs, return retry_id

Response fields:
- `string retry_id` -- populated only when `prepare_retry=true`

---

## Part 3: Single Event Re-push

**Status**: Not started

### New RPC: `EventService.RePushEvent`
- Input: `event_id` (UUID of original event)
- Loads original `event_record` (payload, namespace, event name)
- Calls PushEvent logic with same payload -> schema validates against CURRENT schema
- Returns new `event_id` + `warnings`
- This is "replay this event as if it were pushed fresh"

---

## Part 4: Deterministic Batch Operations

**Status**: Not started

### Migration 000017: `add_batch_jobs`
```sql
CREATE TABLE batch_jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    namespace     VARCHAR(255) NOT NULL,
    job_type      VARCHAR(50) NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'pending',
    data          JSONB NOT NULL,
    total         INTEGER NOT NULL DEFAULT 0,
    processed     INTEGER NOT NULL DEFAULT 0,
    failed        INTEGER NOT NULL DEFAULT 0,
    ttl_seconds   INTEGER NOT NULL DEFAULT 900,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_batch_jobs_tenant_status ON batch_jobs (tenant_id, status);
CREATE INDEX idx_batch_jobs_expires_at ON batch_jobs (expires_at) WHERE status NOT IN ('completed', 'cancelled');
```

### Job Types
- `event_repush` -- data: `{item_ids: [...], filter: {...}}`
- `delivery_retry` -- data: `{item_ids: [...], filter: {...}}`

### Batch Size Cap
- 10,000 items maximum per batch
- If filter matches > 10K, return error asking user to narrow filter

### Batch TTL
- 15 minutes (900 seconds) default
- `ttl_seconds` stored on row, `expires_at = created_at + ttl_seconds`
- Periodic cleanup of expired rows

### Batch Creation Flow (opt-in via `prepare_repush` / `prepare_retry`)
1. Run filter query WITHOUT pagination (up to 10K cap)
2. Snapshot all matching IDs into `data.item_ids`
3. Store filter params in `data.filter` for audit
4. Return batch ID as `repush_id` / `retry_id`

### New EventService RPCs
- `RePushEvents(repush_id)` -> validate batch, set status=processing, enqueue River job
- `GetRepushStatus(repush_id)` -> return status/total/processed/failed
- `CancelRepush(repush_id)` -> set status=cancelled

### New DeliveryService RPCs
- `RetryDeliveries(retry_id)` -> same pattern
- `GetRetryStatus(retry_id)` -> same
- `CancelRetry(retry_id)` -> same

### River Worker: `BatchJobWorker`
- Queue: `batch_jobs`
- Reads batch, dispatches by `job_type`
- Checks batch status before each item (abort if `cancelled`)
- Updates progress counters periodically
- Sets final status: `completed` or `failed`

### Cleanup
- Opportunistic: on batch creation, delete expired rows
- Or periodic River cron job

---

## Part 5: Delivery Retry by Filter

**Status**: Not started

Covered by Part 2 (filters on ListDeliveries) + Part 4 (batch-based retry via `prepare_retry` + `RetryDeliveries`).

Existing single-delivery `RetryDelivery` RPC remains unchanged.

---

## Part 6: Web UI

**Status**: Not started

### Event Reports Page (`/events/[eventName]/reports`)
- Add filter controls: schema_valid toggle, labels input, date range picker
- "Re-push All Matching" button: sets `prepare_repush=true`, shows count, confirms, calls `RePushEvents`
- Progress bar via polling `GetRepushStatus`
- Per-row "Re-push" button (single `RePushEvent`)

### New Deliveries Page (`/deliveries`)
- Full delivery list with filters: status, error_category, webhook, event, subscription, date range
- "Retry All Matching" button with batch pattern
- Progress bar via polling `GetRetryStatus`
- Per-row "Retry" button (existing `RetryDelivery`)

### Webhook Detail Page (`/webhooks/[webhookId]`)
- Add delivery filters: status, error_category
- Bulk retry button

---

## Part 7: Docs Fixes

**Status**: Not started

- Fix Python HMAC indentation in `architecture.mdx` (lines 350-351)
- Fix OTel sidecar reference in `docker-compose.md` (line 81)
- Fix `opencode.md`: 5 services not 6, landing page is redirect, no namespace package

---

## Implementation Order

```
Part 1 --> Part 2 --> Part 4 --> Part 6
                  \
                   Part 3 (parallel with Part 4)

Part 7 (parallel with everything)
```

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
