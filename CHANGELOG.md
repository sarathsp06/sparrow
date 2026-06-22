# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.4.0] - 2026-06-22

### Added

- OKF knowledge bundle at `okf/` covering architecture, packages, services, database schema, frontend, and DevOps — auto-regeneratable from source
- Dual-signature webhook signing key auto-generation on registration (Ed25519 keypair created automatically)
- Contributor guide (`CONTRIBUTING.md`)

### Changed

- Docs hardened: fragile counts (RPCs, migrations, tables, error categories) removed — authoritative data lives in OKF
- Landing page capabilities updated to reflect encryption and signing
- Documentation adopted paper-style theme with hardened comparison claims
- Landing page comparison table improved spacing and removed "choose/alternatives" sections

## [1.3.5] - 2026-06-10

### Added

- End-to-end retry integration test with Gauge test suite
- Comparison table expanded to include Convoy, Hookdeck, AWS SNS, and Zeplo

### Changed

- OSS positioning and fit guidance clarified
- Helm validation targets made self-contained
- README updated with accurate info and e2e test integration

## [1.3.4] - 2026-05-28

### Changed

- Landing page replaced with redirect to `/webhooks` for unified UX
- Docs landing page replaced with Starlight splash page
- Network policy and signing configurations updated

### Fixed

- Dark mode code block readability and inline code contrast in docs

## [1.3.3] - 2026-05-15

### Added

- Ed25519 asymmetric webhook signing (dual HMAC + Ed25519) — every delivery signed with both `v1,` and `v1a,` signatures
- Migration 000022: `ed25519_private_key` column on `webhook_registrations`
- Docker image build and push to GHCR in release workflow

## [1.3.2] - 2026-05-01

### Added

- Interactive Table of Contents with internal PDF links in Svelte 5 tutorial
- opencode.json for automated agent-based code review on every task

### Changed

- Tutorial PDF generation migrated from ReportLab to Typst
- Tutorial content hardened with improved accuracy and Svelte 5 best practices
- opencode.md synced with codebase state

### Fixed

- Orphaned heading prevention and table splitting across pages in Typst PDF
- Double code block borders in tutorial output

## [1.3.1] - 2026-04-20

### Added

- Svelte 5 tutorial PDF with FiraCode/Poppins fonts and generative cover design

## [1.3.0] - 2026-04-15

### Added

- Per-webhook rate limiting with leaky bucket algorithm and HTTP 429 Retry-After parsing
- Migration 000021: `rate_limit_rps` column on `webhook_registrations`, `webhook_rate_limit_state` table
- `rate_limited` error category (retryable)

### Fixed

- Client packaging files (Go/Python/TypeScript) preserved across `make generate`

## [1.2.0] - 2026-04-10

### Added

- SDK guides for Go, Python, and TypeScript in the documentation site with step-by-step integration examples, API key authentication, label filters, TLS configuration, and error handling
- Python client packaging (`pyproject.toml`, `__init__.py`, `py.typed`) -- now pip-installable with `pip install -e client/python`
- TypeScript Connect-RPC client packaging (`package.json`) with proper peer dependencies and exports
- "SDK Guides" section in docs sidebar linking to per-language guides
- "Multi-Language Client SDKs" row in landing page comparison table
- Client library callout in Getting Started section with links to Go, Python, and TypeScript guides
- "SDKs" link in landing page navigation bar

### Fixed

- N+1 query in ListWebhooks and ListWebhooksByHealth -- subscription events are now batch-fetched in a single query via `ListSubscriptionsByWebhookIDs`
- Batch worker terminal status decision used stale local counters after periodic flush; now re-reads cumulative totals from DB
- Double close of HTTP response body in webhook client `ReadBody`
- Non-atomic `UpdateWebhookConfig` -- webhook update and subscription replacement now run in a single transaction via `RunInTransaction`
- `GetWebhookByID` and `GetSubscription` returned `(nil, nil)` for not-found instead of `(nil, ErrNotFound)`
- `RegisterWebhook` and `RegisterWebhookWithSubscriptions` silently returned nil on duplicate URL instead of `ErrAlreadyExists`
- Readiness probe now pings the database and returns 503 if unreachable
- Helm chart now validates `secrets.encryptionKey` is set via `{{ required }}` template function
- Go client import paths in README and docs corrected from `client/go/proto` to `proto` (the actual module path)

### Changed

- Per-webhook request timeout applied via `context.WithTimeout` using the configured `request_timeout_seconds`
- Service-layer validation errors now use typed `svcerrors.ServiceError` with explicit gRPC codes, replacing the fragile string-matching fallback block in `toGRPCError`
- Landing page compliance tags changed from "HIPAA-Ready" / "SOC 2" to "Audit-Friendly" / "Compliance-Ready"
- Added `config.Validate()` with port range, encryption key, and DATABASE_URL checks on startup
- Simplified `GetWebhooksByHealthPaginated` implementation using `SelectContext` instead of manual row scanning
- Documented PushEvent cross-driver transaction gap (sqlx vs pgxpool)
- Documented `GenerateKey` as a test/development utility in crypto package
- Deduplicated code patterns across service, repository, gRPC handler, and worker layers (13 extracted helpers)
- Client libraries reference page now links to dedicated SDK guides and uses correct import paths
- `.gitignore` updated with negation rules to track packaging files within generated client directories

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
- Error classification with automatic retryability detection
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
