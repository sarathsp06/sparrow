---
type: Go Package
title: internal/health
description: HTTP health check and readiness endpoints
tags: [health, monitoring]
timestamp: 2026-06-22T00:00:00Z
---

# internal/health

Provides `/health` and `/ready` HTTP endpoints for liveness and readiness probes.

## Checker

```go
type Checker struct {
    dbPool       storage.DB
    queueManager *queue.Manager
    startTime    time.Time
}
```

- `Health()` — DB ping + queue status, returns JSON with status, version, uptime, checks
- `Ready()` — DB ping only

## Citations

- `internal/health/health.go`
