---
type: Go Package
title: cmd/server
description: Main entry point — wires all dependencies, starts gRPC + HTTP servers
tags: [entrypoint, di]
timestamp: 2026-06-22T00:00:00Z
---

# cmd/server

The binary entry point for the Sparrow server. Performs manual dependency injection in `main.go`.

## Responsibilities

1. Load config via `config.Load()`
2. Set up logging and observability
3. Connect to PostgreSQL (sqlx pool + pgxpool for River)
4. Bootstrap default tenant
5. Create repositories

   ```
   webhookRepo := store.NewRepository(db)
   nsRepo      := namespace.NewRepository(db)
   ```

6. Wrap with OTel tracing (gowrap-generated wrappers)
7. Create services
8. Create gRPC servers + Connect-RPC adapter
9. Register HTTP routes with chi (API routes + health + embedded UI SPA)
10. Start River queue manager
11. Start gRPC + HTTP servers with graceful shutdown

## HTTP Route Table

| Pattern | Handler | Auth |
|---------|---------|------|
| `/webhook.WebhookService/*` | Connect-RPC | Yes |
| `/webhook.EventService/*` | Connect-RPC | Yes |
| `/webhook.SubscriptionService/*` | Connect-RPC | Yes |
| `/webhook.DeliveryService/*` | Connect-RPC | Yes |
| `/webhook.HealthService/*` | Connect-RPC | Yes |
| `GET /health` | Health handler | No |
| `GET /ready` | Readiness handler | No |
| `* (NotFound)` | UI SPA | No |

## Citations

- `cmd/server/main.go`
