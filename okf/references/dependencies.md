---
type: Reference
title: External Dependencies
description: Key external Go libraries, JavaScript packages, and infrastructure dependencies
tags: [dependencies, external]
timestamp: 2026-08-29T00:00:00Z
---

# External Dependencies

## Go Runtime Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/riverqueue/river` | Job queue (PostgreSQL-backed) |
| `github.com/jmoiron/sqlx` | SQL query builder and scanner |
| `github.com/jackc/pgx/v5` | PostgreSQL driver (pgxpool for River) |
| `github.com/go-chi/chi` | HTTP router |
| `github.com/danielgtaylor/huma/v2` | REST/OpenAPI framework — generates the spec from Go handler structs, serves `/docs` (Scalar) |
| `github.com/kelseyhightower/envconfig` | Env var config loading |
| `go.opentelemetry.io/otel` | OpenTelemetry tracing/metrics |
| `go.uber.org/mock` | Mock generation (golang/mock fork) |

## JavaScript Dependencies

| Package | Purpose |
|---------|---------|
| `@sveltejs/kit` + `@sveltejs/adapter-static` | SvelteKit framework + SPA adapter |
| `svelte` | Svelte 5 (runes) |
| `openapi-fetch` | Typed REST client, generated from the OpenAPI spec |
| `openapi-typescript` | Generates `api-types.d.ts` from `api/openapi.yaml` |
| `tailwindcss` + `@tailwindcss/vite` | CSS framework |
| `svelte-jsoneditor` | JSON editor component |

## Infrastructure

| Component | Purpose |
|-----------|---------|
| PostgreSQL 15+ | Single data store |
| River (PG-based) | Job queue |
| OpenTelemetry Collector | Observability pipeline |

## Citations

- `go.mod`
- `web/package.json`
