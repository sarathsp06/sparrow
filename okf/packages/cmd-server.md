---
type: Go Package
title: cmd/server
description: Main entry point — wires all dependencies, starts the REST/OpenAPI HTTP server
tags: [entrypoint, di]
timestamp: 2026-08-29T00:00:00Z
---

# cmd/server

The binary entry point for the Sparrow server. Performs manual dependency injection in `main.go`.

## Responsibilities

1. Load config via `config.Load()`
2. Set up logging and observability
3. Connect to PostgreSQL (sqlx pool + pgxpool for River)
4. Bootstrap default tenant
5. Create the webhook repository: `webhookRepo := store.NewRepository(db)`
6. Wrap with OTel tracing (gowrap-generated wrappers)
7. Create the webhook service
8. Register HTTP routes with chi (`rest.Mount` registers every `/v1` Huma operation + `/openapi.*` + `/docs`)
9. Register health + readiness handlers and the embedded UI SPA fallback
10. Start River queue manager
11. Start the HTTP server with graceful shutdown

## HTTP Route Table

| Pattern | Handler | Auth |
|---------|---------|------|
| `/v1/*` | Huma REST API (`internal/rest`) | Yes |
| `/docs`, `/openapi.*` | Huma-served Scalar UI + spec | No |
| `GET /health` | Health handler | No |
| `GET /ready` | Readiness handler | No |
| `* (NotFound)` | UI SPA | No |

## Citations

- `cmd/server/main.go`
