---
type: Concept
title: Batch Jobs
description: Snapshot-based bulk operations for re-push and retry — deterministic, cancellable, up to 10K items
tags: [batch, re-push, retry, snapshot]
timestamp: 2026-06-22T00:00:00Z
---

# Batch Jobs

All bulk operations use a **deterministic snapshot-based batch pattern**. When a user searches with filters and opts in, the matching IDs are snapshotted into a `batch_jobs` row at query time. The bulk action operates on that snapshot — NOT a live re-query.

## Table

`batch_jobs` — generic table with `job_type` + JSONB `data`.

| Column | Purpose |
|--------|---------|
| `job_type` | `event_repush` or `delivery_retry` |
| `data` | JSONB: snapshot of IDs |
| `total`, `processed`, `failed` | Progress tracking |
| `status` | pending, processing, completed, failed, cancelled, expired |
| `ttl`, `expires_at` | 15-minute TTL |

## Constraints

- Max 10,000 items per batch
- 15-minute TTL
- Cancellation checks every 25 items

## Flow

```
Search → prepare_* flag → snapshot into batch_jobs → return repush_id/retry_id
    → BatchJobWorker processes snapshot → poll GetRepushStatus/GetRetryStatus
```

## Citations

- `db/migrations/000017.up.sql` — batch_jobs table
- `internal/webhooks/queue/batch_job_worker.go` — processing
