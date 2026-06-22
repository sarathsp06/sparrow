---
type: Concept
title: Webhook Health
description: Rolling health state machine with health events, summaries, and state tracking
tags: [health, monitoring, state-machine]
timestamp: 2026-06-22T00:00:00Z
---

# Webhook Health

Tracks the health of each webhook target URL using a state machine with three states:

## Health State Machine

```
healthy (consecutive failures = 0)
    │
    ▼ failure
degraded (consecutive failures > 0, < threshold)
    │
    ▼ continued failures
unhealthy (consecutive failures >= threshold)
    │
    ▼ success
healthy
```

## Tables

| Table | Purpose |
|-------|---------|
| `webhook_health_events` | Per-delivery health event with response time and error category |
| `webhook_health_summaries` | Rolling window summaries with p95, success rate, error breakdown |
| `webhook_health_state` | Current state per webhook (consecutive failures, last success) |

## Citations

- `db/migrations/000001.up.sql` — initial schema
- `db/migrations/000018.up.sql` — unexpected status errors
