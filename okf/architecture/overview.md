---
type: Concept
title: System Architecture
description: High-level architecture of Sparrow — REST/OpenAPI server, event-driven pipeline, PostgreSQL-backed job queue
tags: [architecture, core]
timestamp: 2026-08-29T00:00:00Z
---

# System Architecture

Sparrow is an event-driven webhook delivery platform with three main layers: API surface, service layer, and async job queue.

## REST/OpenAPI API

A single port serves the whole API surface, versioned under `/v1`:

| Port | Protocol | Use case |
|------|----------|----------|
| `:8080` | HTTP/1.1 + REST/JSON | Web UI, curl, client SDKs, interactive docs at `/docs` |

The HTTP server uses [chi](https://github.com/go-chi/chi) for routing with middleware groups. [Huma](https://github.com/danielgtaylor/huma) registers every `/v1` operation on top of chi and generates the OpenAPI 3.1 document from the same Go handler structs, served at `/openapi.yaml`/`/openapi.json` and rendered interactively at `/docs` (Scalar).

## Layers

```
Client (curl / Browser / client SDKs)
    │
    ▼
API Layer
├── internal/rest       — Huma REST handlers, one file per resource (HTTP :8080)
├── internal/middleware  — Auth + security headers
    │
    ▼
Service Layer
├── internal/webhooks — Core business logic (WebhookServiceInterface)
├── internal/tenant   — Tenant bootstrap + default tenant
    │
    ▼
Repository Layer
├── internal/webhooks/store — ~80 methods, sqlx-based
    │
    ▼
Job Queue Layer (River)
├── EventProcessingWorker — Fans out events to subscriptions
├── WebhookWorker — HTTP delivery with retries
├── BatchJobWorker — Bulk re-push / retry
    │
    ▼
HTTP Client Layer
├── internal/webhooks/client — Delivery HTTP client, HMAC/Ed25519 signing
```

## Dependencies

- [PostgreSQL](/database/schema.md) — single data store (no Redis)
- [River](https://riverqueue.com) — job queue backed by PostgreSQL
- [sqlx](https://github.com/jmoiron/sqlx) — database queries
- [pgxpool](https://github.com/jackc/pgx) — River's PG pool
- [chi](https://github.com/go-chi/chi) — HTTP router
- [Huma](https://github.com/danielgtaylor/huma) — REST/OpenAPI framework
- [OpenTelemetry](/packages/internal-observability.md) — traces, metrics, logs via OTLP
- More in [references/dependencies.md](/references/dependencies.md)

## Citations

- Architecture diagram in `opencode.md`
- Route configuration in `internal/middleware/apikey.go` and `cmd/server/main.go`
