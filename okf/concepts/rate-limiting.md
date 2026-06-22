---
type: Concept
title: Rate Limiting
description: Per-webhook leaky bucket rate limiting for delivery throughput
tags: [rate-limit, leaky-bucket, delivery]
timestamp: 2026-06-22T00:00:00Z
---

# Rate Limiting

Per-webhook delivery rate limiting using a **DB-backed leaky bucket** algorithm. No Redis dependency.

## Configuration

- `rate_limit_rps` on `webhook_registrations` — max deliveries per second
- Stored in `webhook_rate_limit_state` table per webhook

## Enforcement

The [WebhookWorker](/packages/internal-webhooks-queue.md) calls `AcquireDeliverySlot` before each delivery:

1. Check `webhook_rate_limit_state` for `next_delivery_at`
2. If current time < `next_delivery_at`, calculate wait duration
3. If within threshold, hold the delivery slot
4. If past threshold, skip delivery (expired)

## 429 Handling

Parses `Retry-After` from target responses (seconds and HTTP-date formats, capped at 15 minutes). Records `rate_limited` [error category](/concepts/error-classification.md).

## Citations

- `db/migrations/000021.up.sql` — rate_limit_rps + webhook_rate_limit_state
