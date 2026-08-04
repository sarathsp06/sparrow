---
type: Configuration
title: Environment Variables
description: All server configuration loaded from environment variables via kelseyhightower/envconfig
tags: [config, env-vars]
timestamp: 2026-07-06T16:46:48Z
---

# Environment Variables

All configuration via environment variables using `kelseyhightower/envconfig`.

## Core

| Variable | Purpose | Default |
|----------|---------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://localhost/riverqueue?sslmode=disable` |
| `SPARROW_ENCRYPTION_KEY` | 64-char hex (32 bytes) KEK | Required |

## Server

| Variable | Purpose | Default |
|----------|---------|---------|
| `SPARROW_GRPC_PORT` | gRPC listen port | `50051` |
| `SPARROW_HTTP_PORT` | HTTP/Connect-RPC listen port | `8080` |
| `SPARROW_SERVE_UI` | Serve embedded SvelteKit UI | `false` |

## Auth & Security

| Variable | Purpose | Default |
|----------|---------|---------|
| `SPARROW_API_KEY` | Optional API key for auth; accepted via HTTP `X-API-Key` header and gRPC `x-api-key` metadata | — (open access) |
| `SPARROW_ALLOW_PRIVATE_NETWORKS` | Allow localhost/private IPs as webhook URLs | `false` |

## Observability

| Variable | Purpose | Default |
|----------|---------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP HTTP export endpoint | — |
| `ENVIRONMENT` | `development` or `production` | — |

## CORS

| Variable | Purpose | Default |
|----------|---------|---------|
| `CORS_ALLOWED_ORIGINS` | Comma-separated CORS origins | — |

## Database Pools

| Pool | Library | Config | Purpose |
|------|---------|--------|---------|
| sqlx | `jmoiron/sqlx` | MaxOpen=25 | App queries |
| pgxpool | `jackc/pgx/v5` | MaxConns=50, MinConns=10, 30min lifetime | River queue |

## Citations

- `internal/config/config.go`
