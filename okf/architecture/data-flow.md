---
type: Concept
title: Event Pipeline Data Flow
description: End-to-end flow from PushEvent RPC through fan-out, delivery, and retry
tags: [data-flow, core]
timestamp: 2026-06-22T00:00:00Z
---

# Event Pipeline Data Flow

## PushEvent → Delivery

```
PushEvent RPC
    │
    ▼
1. Idempotency check (if idempotency_key provided)
    │
    ▼
2. Validate payload against registered event schema
   (soft validation — always stores, tags schema_valid=false on mismatch)
    │
    ▼
3. Insert event_record
    │
    ▼
4. Enqueue EventArgs job → River "events" queue
    │
    ▼
EventProcessingWorker.Work()
    ├── Load event from DB
    ├── Query matching subscriptions (tenant_id + namespace + event_name)
    ├── For each subscription: apply Go template transform (if enabled)
    ├── Batch-insert delivery records (single multi-row INSERT)
    └── Batch-enqueue WebhookArgs jobs → River "webhooks" queue
    │
    ▼
WebhookWorker.Work()
    ├── Acquire rate-limit slot (leaky bucket per webhook)
    ├── Load delivery from DB
    ├── Build HTTP request (headers, HMAC + Ed25519 dual signing)
    ├── POST to target URL (with timeout, redirect handling)
    ├── Record delivery_attempt
    ├── Update delivery status (success/failed/retrying)
    ├── Update health events + health state
    ├── Rate-limit handling: parse Retry-After on 429
    └── If failed + retries remain → re-enqueue with exponential backoff
```

## Batch Operations (Re-Push / Retry)

```
Search with filters
    │
    ▼
Opt-in "Prepare" flag snapshots matching IDs into batch_jobs row
    │
    ▼
Returns repush_id / retry_id (opaque to user)
    │
    ▼
BatchJobWorker processes snapshot:
    ├── Reads batch_jobs row (snapshot of IDs)
    ├── For re-push: replays events (bypasses idempotency dedup)
    ├── For retry: re-enqueues failed deliveries
    ├── Cancellation check every 25 items
    └── Reports progress / completion
```

## Citations

- Flow described in `opencode.md` Data Flow section
- `internal/webhooks/queue/event_worker.go` — fan-out implementation
- `internal/webhooks/queue/webhook_worker.go` — delivery with rate limiting
- `internal/webhooks/queue/batch_job_worker.go` — batch processing
