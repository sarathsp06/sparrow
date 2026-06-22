---
type: Concept
title: System Architecture
description: High-level architecture of Sparrow — dual-protocol server, event-driven pipeline, PostgreSQL-backed job queue
tags: [architecture, core]
timestamp: 2026-06-22T00:00:00Z
---

# System Architecture

Sparrow is an event-driven webhook delivery platform with three main layers: API surface, service layer, and async job queue.

## Dual-Protocol API

Two ports serve the same 5 protobuf-defined services plus 1 Go-only service:

| Port | Protocol | Use case |
|------|----------|----------|
| `:8080` | HTTP/1.1 + Connect-RPC | Web UI, curl, client SDKs |
| `:50051` | gRPC | gRPC-native clients, grpcurl |

The HTTP server uses [chi](https://github.com/go-chi/chi) for routing with middleware groups. Connect-RPC handlers delegate directly to the same gRPC server implementations.

## Layers

```
Client (gRPC / HTTP-Connect / Browser)
    │
    ▼
API Layer
├── internal/connect  — Connect-RPC adapter (HTTP :8080)
├── internal/grpc     — gRPC handlers (:50051)
├── internal/middleware — Auth + security headers
    │
    ▼
Service Layer
├── internal/webhooks — Core business logic (WebhookServiceInterface)
├── internal/namespace — Namespace CRUD
├── internal/tenant   — Tenant bootstrap + default tenant
    │
    ▼
Repository Layer
├── internal/webhooks/store — ~80 methods, sqlx-based
├── internal/namespace/store
├── internal/tenant/store
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
- [OpenTelemetry](/packages/internal-observability.md) — traces, metrics, logs via OTLP
- More in [references/dependencies.md](/references/dependencies.md)

## Citations

- Architecture diagram in `opencode.md`
- Route configuration in `internal/middleware/api_key.go` and `cmd/server/main.go`
