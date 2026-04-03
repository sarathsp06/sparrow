# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2026-04-03

### Added

- Standalone `deploy/docker-compose.yml` using published `ghcr.io/sarathsp06/sparrow` image -- users can run Sparrow without cloning the repo
- Docker image badge in README linking to GitHub Container Registry

### Changed

- README quick start now shows inline docker-compose YAML (no git clone needed)
- Docs installation and Docker Compose pages updated to use published image as primary path
- Helm chart defaults `image.tag` to `.Chart.AppVersion` instead of `latest`
- Cleaned up stale `SPARROW_ENCRYPTION_KEY` from root docker-compose.yml

## [0.2.0] - 2026-04-03

### Added

- Optional API key authentication via `SPARROW_API_KEY` environment variable with HTTP middleware and gRPC interceptors
- Runtime config injection so the embedded SPA reads the API key without a rebuild
- Helm chart with PodDisruptionBudget, PostgreSQL security context, and initContainer resource limits
- Kubernetes deployment documentation and k8s Makefile targets
- Release workflow with git-cliff for automated changelog generation (`make release`)
- Clean semver injection into web UI, docs site, and Docker builds
- Event subscription template dry-run API and UI (#10)
- Namespace chooser in the web UI (#11)
- Documentation site with proto-to-docs generator and hostname switcher
- Marketing landing page on Astro/Starlight docs site

### Changed

- Refactored webhook service based on protobuf evaluation (#8)
- Optimized database interactions and fixed performance bottlenecks (#7)
- Propagated protobuf Timestamp and Struct types (#5)
- Used ghcr.io image references across docs and config

### Fixed

- OTLP export is now opt-in via `OTEL_EXPORTER_OTLP_ENDPOINT` (no more startup errors without collector)
- CI workflow lint issues and reliable builds
- Integration test failures (SSRF config and webhook ID mismatch)
- Embedded UI placeholder for `go:embed` in CI
- SvelteKit build failure from stale `PUBLIC_API_KEY` import
- Broken docs site links (missing trailing slash on base URL)

### Documentation

- Kubernetes deployment guide
- Docker Compose deployment guide
- Getting started and installation docs
- Project badges and web UI quick start section
- Protobuf evaluation and recommendations (#6)

## [0.1.0] - 2025-11-14

### Added

- Core webhook registration, event management, and subscription system
- Event fan-out with River job queue for async HTTP delivery
- Go template transforms for subscription payloads
- HMAC webhook signing
- Delivery retries with exponential backoff and error classification
- Health tracking with success rates and response time percentiles
- gRPC and Connect-RPC dual-protocol API
- SvelteKit embedded web UI
- OpenTelemetry tracing, metrics, and structured logging
- PostgreSQL storage with sqlx and migration tooling
- Extensive test coverage (#4)
- Docker multi-stage build (Node frontend + Go backend + distroless runtime)
- Benchmarking tool with reservoir sampling
- JSON schema validation for event payloads
