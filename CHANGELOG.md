# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.9] - 2026-09-18

### Changed

- README deployment docs now link directly to the GitHub Container Registry package and list configuration variables as an ordered list.

## [0.5.8] - 2026-09-18

### Changed

- SendGrid alert bootstrap now keeps the auto-created `_sparrow` webhook inactive until both `SPARROW_SENDGRID_API_KEY` and a non-placeholder `SPARROW_ALERT_FROM_EMAIL` are configured.
- Recipe parameters now expose defaults, secret-entry hints, and activation metadata to the API and dashboard.

## [0.5.6] - 2026-09-17

### Changed

- **BREAKING (encryption config):** Sparrow now requires `SPARROW_ENCRYPTION_KEYS` and `SPARROW_ENCRYPTION_PRIMARY_KEY_ID` for all deployments. The old single-key `SPARROW_ENCRYPTION_KEY` path is no longer supported.
- Envelope encryption is now keyring-only: newly encrypted values always carry a key ID, and keyed envelopes are the only supported ciphertext format.
- Portal tokens are now keyring-only `spt_v2` tokens with deterministic key-ID verification; legacy `spt_v1` portal tokens are no longer accepted.
- Helm chart, CI, Makefile, README, and deployment/docs examples now use the keyring-only configuration path.

## [0.5.3] - 2026-09-17

### Changed

- **BREAKING (portal embedding):** the consumer portal now serves its entire API under a single static prefix, `/portal/api/*`, via a new gateway that derives the consumer from the bearer token instead of the URL. Operators exposing the portal to external users allowlist just `/portal`, `/_app`, and `/portal/api` — no per-consumer forwarding rules and no hand-written deny rules for event injection or token minting (the gateway refuses those itself, and cross-consumer access is now structurally impossible). Portal tokens no longer authenticate against `/v1/consumers/{consumer}/*` directly; all portal traffic routes through `/portal/api`. Minting (`POST /v1/consumers/{consumer}/portal-token`, admin key) is unchanged.

### Fixed

- Event type Update/Register/Push pages rendered blank: `svelte-jsoneditor`'s Ajv-based schema validator compiles validators via `new Function(...)` at component setup, which the app's `script-src` CSP (missing `unsafe-eval`) throws on before `onMount` runs, crashing the whole page silently. Validator creation now degrades to no live validation instead of crashing the page.

## [0.4.0] - 2026-09-15

### Added

- Sparrow now self-generates two internal events under a reserved `_sparrow` consumer: `sparrow.webhook.health_changed` (on any health transition, skipping the initial `unknown` → `healthy`) and `sparrow.webhook.delivery_failed` (when a delivery exhausts all retries)
- Opt-in email alerts on those events via a new `webhook_alert_configs` resource (`/v1/consumers/{consumer}/alert-configs`), scoped per-webhook or consumer-wide
- `sendgrid` recipe: sends alert emails via SendGrid's v3 Mail Send API, fanning out to every opted-in recipient in one call
- New guide: [Webhook Health Alert Emails](docs/src/content/docs/guides/webhook-health-alerts.mdx)

### Fixed

- Corrected the satellites summary table in the docs, tutorial, and OKF architecture notes

## [0.3.0] - 2026-09-14

### Added

- `twilio` recipe: send events as SMS via the Twilio Messages API
- Recipes can declare `webhook.secret_headers`; values are envelope-encrypted at rest and masked in API responses (wired through the CLI `use` command)
- `SPARROW_MAX_BODY_BYTES` caps incoming request body size (default 5 MiB, min 1 MiB); oversized bodies return `413`
- Content-Security-Policy header on all responses
- `scripts/release-submodules.sh`, wired into CI, tags and publishes `pkg/signature`, `pkg/template`, and `satellites/sparrow` with working-tree `replace` directives stripped so `go install .../satellites/sparrow@latest` resolves (see `docs/adr/0002-cli-module-split.md`)

### Changed

- **BREAKING:** renamed the "namespace" concept to "consumer" throughout the API (`/v1/consumers/{consumer}/...`), JSON fields, DB schema, CLI (`--consumer` / `SPARROW_CONSUMER`), and UI
- **BREAKING:** with `ENVIRONMENT=production`, the server now refuses to start unless `SPARROW_API_KEY` is set
- `clickhouse` recipe now stores the ClickHouse key as an encrypted `X-ClickHouse-Key` secret header instead of a plaintext header
- `http_config`'s `max_retries`, `verify_ssl`, `follow_redirects`, and `capture_response_body` now distinguish "left unset" from "explicitly set to zero/false" — setting any one no longer silently resets the others to their zero value, and `max_retries: 0` is honored instead of falling back to the default
- Per-webhook `follow_redirects: false` is now honored (previously silently ignored)
- Pagination `limit` is capped at 1000 through a single shared helper; documented min/max in the OpenAPI spec
- Delivery `status` reports `retrying` while River attempts remain, reserving `failed` for the actual terminal state (previously flipped from `failed` back to `success` mid-retry-cycle)
- Webhook creation response now returns the real `created_at`/`updated_at` instead of the zero timestamp
- Dashboard's Health page moved to `/dashboard/health` so it no longer collides with the server's `/health` liveness endpoint on hard refresh/deep-link
- `maskSecret` no longer leaks plaintext secret characters (masks to the `whsec_` prefix only)

## [1.4.1] - 2026-06-23

### Changed

- Split monolithic `RepositoryInterface` (63 methods) into 7 per-domain narrow interfaces: `WebhookRepository`, `SubscriptionRepository`, `EventTypeRepository`, `EventRepository`, `HealthRepository`, `BatchRepository`, `RateLimitRepository`
- Workers now depend only on the interfaces they actually use (2-5 params instead of 63 methods each)
- Composite `RepositoryInterface` embeds all narrow interfaces + `Transactor` base interface

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
