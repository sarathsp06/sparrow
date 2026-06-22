---
type: Go Package
title: internal/webhooks/queue
description: River job queue — event processing, webhook delivery, and batch job workers
tags: [queue, workers, river]
timestamp: 2026-06-22T00:00:00Z
---

# internal/webhooks/queue

Manages the River job queue with three worker types.

## Workers

| Worker | Queue | Workers | Poll | Job Args |
|--------|-------|---------|------|----------|
| `EventProcessingWorker` | `events` | 20 | 2s | `EventArgs` |
| `WebhookWorker` | `webhooks` | 20 | 2s | `WebhookArgs` |
| `BatchJobWorker` | `batch_jobs` | 5 | 5s | `BatchJobArgs` |

## EventProcessingWorker

[Consumes](/architecture/data-flow.md) from the events queue. On work:

1. Loads event from DB
2. Queries matching subscriptions (tenant_id + namespace + event_name)
3. Applies Go template transform per subscription
4. Batch-inserts delivery records
5. Batch-enqueues WebhookArgs

## WebhookWorker

[Consumes](/architecture/data-flow.md) from the webhooks queue. On work:

1. Acquires rate-limit slot (leaky bucket)
2. Performs HTTP delivery with [HMAC + Ed25519 dual signing](/concepts/payload-signing.md)
3. Records delivery attempt
4. Updates health tracking
5. Re-enqueues with backoff on failure

## BatchJobWorker

[Consumes](/architecture/data-flow.md) from the batch_jobs queue. Processes snapshot-based bulk operations (re-push/retry) with cancellation checks every 25 items.

## Job Types

- `EventArgs` — TenantID, EventID, Namespace, Event, TTLSeconds, Metadata, CreatedAt
- `WebhookArgs` — TenantID, DeliveryID, WebhookID, SubscriptionID, EventID, ExpiresAt, Namespace, MaxAttempts
- `BatchJobArgs` — TenantID, BatchID

## OTel Propagation

Trace context serialized into `job.Metadata` as JSON `propagation.MapCarrier`.

## Citations

- `internal/webhooks/queue/` — 9 files
