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
    dbPool    *pgxpool.Pool
    startTime time.Time
    draining  atomic.Bool
}
```

- `Health()` — liveness: process only, never pings the database; always 200 with status, version, uptime
- `Ready()` — 503 `draining` once `SetDraining()` was called (on SIGTERM), otherwise a DB ping
- `SetDraining()` — called by `cmd/server` at the start of a graceful shutdown

## Citations

- `internal/health/checker.go`
