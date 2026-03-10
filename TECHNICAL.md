# Sparrow - Technical Architecture

## What Is Sparrow?

Sparrow is an event-driven webhook delivery platform. It sits between your application and external HTTP endpoints, providing reliable, observable webhook delivery with retries, health tracking, and payload transformation.

**Problem it solves:** When your app needs to notify external systems (e.g., "user created", "order completed"), direct HTTP calls are fragile. Sparrow acts as a reliable intermediary that guarantees delivery, retries on failure, and provides full audit trails.

---

## Architecture Overview

```
Your App                    Sparrow                         External Endpoints
   |                          |                                    |
   |-- PushEvent -----------> |                                    |
   |   (HTTP/gRPC)            |-- Store event in DB                |
   |                          |-- Queue EventArgs job              |
   |                          |                                    |
   |                          |-- EventWorker picks up job         |
   |                          |   - Find matching subscriptions    |
   |                          |   - Create delivery records        |
   |                          |   - Queue WebhookArgs jobs         |
   |                          |                                    |
   |                          |-- WebhookWorker picks up job       |
   |                          |   - Load webhook config from DB    |
   |                          |   - Apply payload transformation   |
   |                          |   - HTTP POST to endpoint -------> |
   |                          |   - Record success/failure         |
   |                          |   - Update health metrics          |
   |                          |   - River retries on failure       |
```

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Backend | Go 1.25 |
| Database | PostgreSQL 15 |
| Job Queue | River (Postgres-based) |
| API Protocols | gRPC (`:50051`) + Connect-RPC/HTTP (`:8080`) |
| Protobuf | buf.build toolchain |
| Web UI | SvelteKit 5 + TypeScript + Tailwind CSS 4 (`:5173`) |
| Observability | OpenTelemetry (traces, metrics, logs via OTLP) |
| DB Access | pgx/v5 + sqlx (OTel-instrumented) |
| CI | GitHub Actions (lint, test with Postgres, build) |
| Container | Multi-stage Dockerfile (distroless nonroot) |

---

## Core Domain Model

Five concepts, eight database tables:

1. **Events** - Event types your system produces (e.g., `user.created`). Support optional JSON Schema validation and sample payload generation.
2. **Webhooks** - Registered HTTP endpoints with namespace-based multi-tenancy, configurable retries, timeouts, SSL settings, and HMAC signing.
3. **Subscriptions** - Link webhooks to events. Each subscription can override headers, HTTP method, timeout, and apply a Go template payload transformation.
4. **Deliveries** - Individual delivery attempts with full lifecycle tracking (pending -> sending -> success/failed/retrying/expired) and request/response audit.
5. **Health** - Automatic per-webhook health state (healthy/degraded/unhealthy/unknown) computed from delivery outcomes.

---

## Dual-Protocol API

The same gRPC service implementations back both protocols - no code duplication:

- **gRPC** on `:50051` for high-performance programmatic access
- **Connect-RPC (HTTP/JSON)** on `:8080` for curl, browsers, and any HTTP client

Five domain services: `WebhookService`, `EventService`, `SubscriptionService`, `DeliveryService`, `HealthService`. Service definitions live in `proto/webhook.proto`.

Connect-RPC URL pattern: `POST http://localhost:8080/webhook.{Service}/{Method}`

---

## Two-Stage Queue Pipeline

Uses River (Postgres-based job queue) with a fan-out architecture:

```
PushEvent
   |
   v
[events queue] --> EventProcessingWorker
                      |
                      |--> Find matching subscriptions
                      |--> Create delivery records
                      |--> Fan out: one WebhookArgs job per subscription
                              |
                              v
                   [webhooks queue] --> WebhookWorker
                                          |
                                          |--> Fetch webhook config + event payload from DB
                                          |--> Transform payload via Go templates
                                          |--> Send HTTP request (with HMAC signing)
                                          |--> Classify result -> update delivery status
                                          |--> Record health event
                                          |--> Return error to River if retryable
```

**Queue configuration:**

| Queue | Max Workers | Purpose |
|-------|-------------|---------|
| `events` | 20 | Event fan-out to subscriptions |
| `webhooks` | 20 | Webhook HTTP delivery |
| `default` | 5 | General purpose |

**Design decisions:**
- Job payloads are lightweight references (IDs only). Heavy data (webhook config, event payload) is fetched from the DB at work time. This keeps the queue lean.
- OTel trace context is propagated through job metadata using W3C TraceContext injection/extraction.
- Retry behavior is controlled per-webhook via `max_retries` (0-10, default 3) and `retry_backoff_seconds` (1-3600, default 60).

---

## Error Classification & Retry Logic

The `pkg/errors/` package implements a taxonomy-based error classifier that inspects Go errors to determine retryability:

| Category | Example | Retried? |
|----------|---------|----------|
| `server_error` | 5xx HTTP response | Yes |
| `timeout` | Connection/request timeout | Yes |
| `connection_refused` | ECONNREFUSED | Yes (service may be restarting) |
| `network_error` | ECONNRESET, EPIPE, EHOSTUNREACH | Yes |
| `client_error` | 4xx HTTP response | **No** (permanent) |
| `dns_error` | DNS resolution failure | **No** (configuration problem) |
| `tls_error` | TLS/SSL handshake failure | **No** (certificate/config problem) |

The classifier unwraps layered Go errors through: `*url.Error` -> `net.Error` (timeout) -> TLS type assertions -> `*net.DNSError` -> `*net.OpError` -> `*os.SyscallError` -> `syscall.Errno` -> string pattern fallback.

Non-retryable errors return `nil` to River (stopping retries); retryable errors return an `error` to trigger River's built-in retry mechanism.

---

## Health State Machine

Event-sourced health calculation with a 24-hour lookback window:

| State | Condition |
|-------|-----------|
| `unknown` | No recent delivery events |
| `healthy` | Success rate >= 90% with 3+ events |
| `degraded` | Success rate < 90% with 5+ events |
| `unhealthy` | 5+ consecutive failures, OR success rate < 80% with 10+ events |

**How it works:**
1. Each delivery outcome is recorded as a `webhook_health_events` row
2. `webhook_health_state` is atomically upserted (tracks consecutive failures, last success/failure timestamps)
3. Health status is recalculated and written to `webhook_registrations.health`
4. Hourly aggregation computes per-webhook summaries (p95 response time, error category breakdown) via bulk SQL

**Real-time reactivity:** PostgreSQL `LISTEN/NOTIFY` on the `webhook_health_event` channel pushes health events to a `NotificationHandler` interface for async processing.

---

## HTTP Client Design

A centralized, OTel-instrumented HTTP client (`internal/webhooks/client/`):

- **Connection pooling**: 100 max idle connections, 10 per host, 90s idle timeout
- **HMAC signing**: `X-Sparrow-Signature-256` header using `HMAC-SHA256(timestamp + "." + payload, secret)` (Stripe/GitHub pattern)
- **Template engine**: Go `text/template` with LRU cache (100 entries, SHA-256 keyed), ~20 built-in helper functions (json, base64, urlencode, string manipulation, etc.)
- **Object pooling**: `sync.Pool` for `bytes.Buffer`, `[]byte` slices, and header maps to reduce GC pressure
- **Header merging**: Subscription-level headers override webhook-level defaults
- **In-process metrics**: Lock-free atomic counters for request totals, error categories, cache hit rates, and response time statistics

---

## Observability

Full OpenTelemetry stack with OTLP export:

**Three pillars:**
- **Traces**: OTLP HTTP exporter, configurable sampler (ratio/always/never), batch processor
- **Metrics**: OTLP HTTP periodic reader (default 30s export interval)
- **Logs**: OTLP HTTP exporter with batch processor, bridged to Go `slog`

**Custom application metrics:**

| Metric | Type |
|--------|------|
| `sparrow_webhook_registrations_total` | Counter |
| `sparrow_events_pushed_total` | Counter |
| `sparrow_webhook_deliveries_total` | Counter |
| `sparrow_webhook_delivery_duration_seconds` | Histogram |
| `sparrow_queue_depth` | UpDownCounter |
| `sparrow_active_webhooks` | UpDownCounter |

**What's instrumented:** HTTP transport (`otelhttp`), gRPC server (`otelgrpc`), Connect-RPC interceptors (`otelconnect`), job inserter (code-generated tracing wrapper via `gowrap`), webhook delivery worker (manual spans), and all repository calls (generated tracing wrapper).

---

## Codebase Structure

```
cmd/
  server/          - Main entry point (dual-protocol server with graceful shutdown)
  migrate/         - Database migration runner
  benchmark/       - Performance benchmarks
  generate-docs/   - Template function doc generator
internal/
  grpc/            - gRPC service handlers (5 services)
  connect/         - Connect-RPC adapter (wraps gRPC handlers for HTTP/JSON)
  webhooks/
    webhook_service.go  - Central service layer (~25 methods)
    queue/              - River queue manager, workers, job types
    client/             - HTTP client, HMAC signing, template engine, object pools
    store/              - Repository pattern (~40 methods), split by domain
    health/             - Health calculator, aggregation, LISTEN/NOTIFY listener
  observability/   - OTel setup (traces, metrics, logs, custom metrics)
pkg/
  errors/          - Error classification, retryability, stack traces
  storage/         - DB interface + OTel-instrumented Postgres driver
  types/           - Generic utilities (Map, Set, Slice, Pointer, Secret)
proto/             - Protobuf service definitions (buf.build)
db/migrations/     - PostgreSQL migration files
web/               - SvelteKit 5 UI
```

---

## Server Boot Sequence

1. **OTel** - Initialize tracing, metrics, logging (non-fatal if fails)
2. **Database** - Create `pgxpool` (for River) + `sqlx` (for app queries), ping to verify
3. **Repository** - `store.NewRepository` wrapped with auto-generated OTel tracing decorator
4. **Queue** - Create River client with 3 queues, register workers, start processing
5. **gRPC server** (`:50051`) - Register 5 services with `otelgrpc` stats handler, enable reflection
6. **HTTP server** (`:8080`) - Connect-RPC adapter with CORS, `otelconnect` interceptor, `/health` and `/ready` endpoints
7. **Signal handling** - Wait for SIGINT/SIGTERM, graceful shutdown (HTTP 10s timeout, gRPC graceful stop, queue drain)

---

## Database Schema

Eight tables across three migrations:

**Core tables:**
- `event_registrations` - Event type registry (unique name, optional JSON schema)
- `webhook_registrations` - Webhook endpoints (unique namespace+url, health status, HTTP config with constraints: retries 0-10, backoff 1-3600s, timeout 1-300s)
- `event_subscriptions` - Webhook-to-event mappings with per-subscription overrides (CASCADE on webhook delete)
- `event_records` - Immutable event instances (payload as TEXT, metadata as JSONB)
- `webhook_deliveries` - Delivery tracking with status, attempts, request/response data (CASCADE on webhook/event delete, SET NULL on subscription delete)

**Health tables:**
- `webhook_health_events` - Individual health data points (event-sourced)
- `webhook_health_summaries` - Hourly rollup (p95/min/max/avg response times, error category breakdown)
- `webhook_health_state` - Current state per webhook (consecutive failures, timestamps)

**Indexing:** Composite indexes on namespace, active status, health status, timestamps (DESC), webhook+category+timestamp for aggregation, GIN full-text index on delivery request bodies.

---

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_URL` | `postgres://localhost/riverqueue?sslmode=disable` | PostgreSQL connection |
| `HTTP_PORT` | `8080` | Connect-RPC HTTP port |
| `GRPC_PORT` | `50051` | gRPC port |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OpenTelemetry collector |
| `ENVIRONMENT` | `development` | Environment name for OTel resource |
| `PUBLIC_API_URL` | `http://localhost:8080` | API URL for web UI |

## Quick Start

```bash
make docker-dev     # PostgreSQL + River UI + OpenTelemetry Collector
make migrate        # Run database migrations
make run            # Go server (gRPC :50051, HTTP :8080)
make run-web        # SvelteKit UI (:5173)
make example        # Run example gRPC client
```
