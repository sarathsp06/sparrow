# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.2] - 2026-04-10

### Changed

- `SPARROW_ENCRYPTION_KEY` is now required -- the server will not start without it. Previously, an ephemeral key was auto-generated on startup, which silently made encrypted data unreadable after restart. Generate a key with `openssl rand -hex 32`

## [1.1.1] - 2026-04-09

### Added

- Idempotency keys on PushEvent -- pass an optional `id` field to deduplicate events. Duplicate pushes return the existing event ID with a `duplicate` flag instead of creating a new event
- Migration 000020: `idempotency_key` column on `event_records` with partial unique index `(tenant_id, namespace, idempotency_key) WHERE idempotency_key IS NOT NULL`
- `GetEventByIdempotencyKey` repository method for deduplication lookups
- `bool duplicate = 3` field in `PushEventResponse` proto message

### Changed

- `PushEvent` service signature now accepts an optional `idempotencyKey *string` parameter
- RePushEvent and batch RePushEvents always pass nil for idempotency key, ensuring re-pushes are never deduplicated

## [1.0.0] - 2026-04-07

First stable release of Sparrow -- a self-hosted webhook delivery platform with
async fan-out, HMAC signing, health tracking, and a built-in management UI.

### Core Platform

- gRPC and Connect-RPC dual-protocol API with 5 proto-defined services
  (Webhook, Event, Subscription, Delivery, Health) and 1 Go-only service (Namespace)
- Async event fan-out via River job queue with configurable worker pools
- HTTP webhook delivery with HMAC-SHA256 signing, redirect following, and response capture
- Error classification into 9 categories with automatic retryability detection
- Go template payload transformation on subscriptions with graceful fallback
- Soft JSON Schema validation -- events are always accepted, invalid payloads tagged with warnings
- Envelope encryption for webhook secrets and sensitive headers

### Search, Filter & Batch Operations

- Search filters on event reports (schema validity, labels, time range) and deliveries
  (status, error category, subscription, time range)
- Deterministic batch re-push and retry via snapshot-based batch jobs
- Single event re-push (replay an event as if pushed fresh against current schema)

### Namespace & Multi-tenancy

- Namespace-scoped webhooks, events, and subscriptions
- Default tenant auto-provisioned on startup (designed for single-tenant self-hosting)

### Health & Observability

- Per-webhook health tracking with success rate, P95 response time, and consecutive failure count
- Health summary windows with automatic state management
- OpenTelemetry integration: traces, metrics, and structured logs via OTLP export
- Trace context propagation through River job queue

### Web UI

- Embedded SvelteKit SPA served from the Go binary (opt-in via `SPARROW_SERVE_UI`)
- Webhook management: register, pause/resume, view deliveries, bulk retry
- Event management: register schemas, push events, view reports, bulk re-push
- Subscription management with template dry-run testing
- Health dashboard and delivery explorer
- Namespace switcher with persistence
- Terminal aesthetic with Fira Code typography

### Security

- Optional API key authentication via `SPARROW_API_KEY` environment variable
- Constant-time key comparison, HTTP middleware + gRPC interceptors
- Runtime config injection for embedded UI (no rebuild needed to change key)
- Private network protection (configurable via `SPARROW_ALLOW_PRIVATE_NETWORKS`)

### Deployment

- Docker image on GHCR (`ghcr.io/sarathsp06/sparrow`)
- Helm chart with PostgreSQL subchart, PDB, security contexts, and init containers
- Docker Compose for local development
- Cross-platform binaries (Linux/macOS amd64+arm64, Windows amd64) via GoReleaser
- 11 database migrations with composite indexes for hot-path queries

### API & Routing

- Chi router with explicit route registration and middleware groups
- CORS support with configurable allowed origins
- JSON 404 responses for non-GET requests to unknown paths
- gRPC reflection enabled for development tooling
