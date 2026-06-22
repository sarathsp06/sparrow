---
type: Concept
title: Delivery
description: Webhook HTTP delivery lifecycle — pending through success/failure with retries
tags: [delivery, retry, status]
timestamp: 2026-06-22T00:00:00Z
---

# Delivery

Represents a single HTTP delivery of an event to a webhook target URL. Created by the [EventProcessingWorker](/packages/internal-webhooks-queue.md).

## Status Lifecycle

```
pending → sending → success
                → failed (terminal, if max_attempts reached)
                → retrying → sending → ...
                → expired
```

## Key Fields (16 columns)

- `webhook_id`, `event_id`, `subscription_id` — FKs
- `status` — pending, sending, success, failed, retrying, expired
- `attempts` — current attempt count
- `max_attempts` — retry limit from HTTP config
- `error_category` — [classified error](/concepts/error-classification.md)
- `response_status_code`, `response_body` — captured from target

## Citations

- `db/migrations/000001.up.sql` — initial schema
