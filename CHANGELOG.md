# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.9.0] - 2026-04-06

### Added

- Search filters, batch re-push, and batch retry controls in web UI (event reports, deliveries, webhook detail pages)
- New deliveries page with full filter bar and bulk retry
- BatchProgress component for tracking batch job status with polling
- Compact visual flow diagram in webhooks empty state showing the full delivery pipeline
- Lead with "The Full Flow" section in how-it-works docs page

### Changed

- Migrate release pipeline to GoReleaser
- Docs link in webhooks empty state styled as a proper button

## [0.8.1] - 2026-04-06

### Fixed

- Use pre-built GHCR image for Railway deployment instead of Dockerfile build

## [0.8.0] - 2026-04-06

### Added

- Rich search filters for event reports (schema_valid, labels, date range) and deliveries (status, error_category, subscription, date range)
- Deterministic batch re-push and retry with snapshot-based pattern
- New RPCs: RePushEvents, GetRepushStatus, CancelRepush, RetryDeliveries, GetRetryStatus, CancelRetry
- Soft schema validation with warnings (events always accepted, tagged with schema_valid)
- Template transform graceful fallback to envelope payload on failure

## [0.7.1] - 2026-04-06

### Fixed

- Rewrite webhooks empty state with correct getting-started flow

## [0.7.0] - 2026-04-05

### Added

- Getting-started empty state for webhooks page

### Changed

- Remove unused Inter font and OTel collector config

## [0.6.0] - 2026-04-05

### Fixed

- Replace stdlib mux with chi router to fix Connect-RPC routing bug where API paths were unreachable

## [0.5.4] - 2026-04-05

### Changed

- Split API and UI into separate muxes for cleaner auth boundary

## [0.5.3] - 2026-04-05

### Fixed

- Allow UI routes through API key auth when embedded UI is served

## [0.5.2] - 2026-04-05

### Changed

- Migrate docs to proto2astro v0.4.1 JSON-based config pattern

### Fixed

- Enable encryption in integration tests and add dev docker-compose
- Restore missing sidebar sections in docs (Getting Started, Deployment, Reference)

## [0.5.1] - 2026-04-05

### Documentation

- Add Railway one-click deploy button to README, docs, and landing page

## [0.5.0] - 2026-04-05

### Added

- Envelope encryption for webhook secrets and sensitive headers
- Railway deployment template and docs

### Documentation

- Consolidate root markdown files into docs site
- Fix inaccuracies in architecture, config, and technical docs

### Fixed

- Make OTLP export opt-in via OTEL_EXPORTER_OTLP_ENDPOINT

## [0.4.0] - 2026-04-05

### Added

- Migrate docs generation from custom gendocs to proto2astro

## [0.3.2] - 2026-04-03

### Fixed

- Multi-arch Docker build for Apple Silicon support

## [0.3.1] - 2026-04-03

### Added

- Unify Fira Code as default font across web app, docs, and landing page

### Documentation

- Refactor README to follow open-source conventions

## [0.3.0] - 2026-04-03

### Added

- Standalone docker-compose for quick deployment
- Helm chart with PDB, PostgreSQL security context, and initContainer resource limits

## [0.2.0] - 2026-04-03

### Added

- Namespace UI with chooser and persistence (#11)
- Proto-to-docs generator with hostname switcher
- Marketing landing page (moved from web app to docs site)
- Release workflow with git-cliff and changelog generation
- Optional API key authentication via SPARROW_API_KEY
- Semver injection into web UI, docs site, and Docker builds
- Helm chart, Kubernetes Makefile targets, and CI/CD for releases
- Event subscription template dry-run API and UI (#10)
- Go template transforms for subscription payloads
- Kubernetes deployment docs

### Fixed

- CI workflow reliability (lint, go:embed placeholder)
- Docs links, broken references, and landing page sync
- Integration test failures (SSRF config, webhook ID mismatch)

## [0.1.0] - 2025-11-14

### Added

- Core webhook delivery platform: register webhooks, define events, create subscriptions
- Async HTTP delivery via River job queue with retries and exponential backoff
- gRPC and Connect-RPC dual-protocol API
- SvelteKit embedded web UI
- HMAC webhook signing
- Error classification (9 categories with retryability)
- Webhook health tracking (events, summaries, state)
- OpenTelemetry observability (traces, metrics, structured logs)
- JSON schema validation for event payloads
- Extensive test coverage (#4)
- Benchmarking tool with reservoir sampling
