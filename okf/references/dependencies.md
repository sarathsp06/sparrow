---
type: Reference
title: External Dependencies
description: Key external Go libraries, JavaScript packages, and infrastructure dependencies
tags: [dependencies, external]
timestamp: 2026-06-22T00:00:00Z
---

# External Dependencies

## Go Runtime Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/riverqueue/river` | Job queue (PostgreSQL-backed) |
| `github.com/jmoiron/sqlx` | SQL query builder and scanner |
| `github.com/jackc/pgx/v5` | PostgreSQL driver (pgxpool for River) |
| `github.com/go-chi/chi` | HTTP router |
| `github.com/kelseyhightower/envconfig` | Env var config loading |
| `github.com/bufbuild/connect-go` | Connect-RPC framework |
| `google.golang.org/grpc` | gRPC framework |
| `google.golang.org/protobuf` | Protobuf runtime |
| `go.opentelemetry.io/otel` | OpenTelemetry tracing/metrics |
| `go.uber.org/mock` | Mock generation (golang/mock fork) |

## JavaScript Dependencies

| Package | Purpose |
|---------|---------|
| `@sveltejs/kit` + `@sveltejs/adapter-static` | SvelteKit framework + SPA adapter |
| `svelte` | Svelte 5 (runes) |
| `@connectrpc/connect-web` | Connect-RPC client |
| `@bufbuild/protobuf` | Protobuf runtime for JS/TS |
| `tailwindcss` + `@tailwindcss/vite` | CSS framework |
| `svelte-jsoneditor` | JSON editor component |
| `@zerodevx/svelte-json-view` | JSON viewer component |

## Infrastructure

| Component | Purpose |
|-----------|---------|
| PostgreSQL 15+ | Single data store |
| River (PG-based) | Job queue |
| OpenTelemetry Collector | Observability pipeline |

## Citations

- `go.mod`
- `web/package.json`
