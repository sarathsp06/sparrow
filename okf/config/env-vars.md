---
type: Configuration
title: Environment Variables
description: All server configuration loaded from environment variables via kelseyhightower/envconfig
tags: [config, env-vars]
timestamp: 2026-09-14T00:00:00Z
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
| `SPARROW_HTTP_PORT` | HTTP/REST listen port | `8080` |
| `SPARROW_SERVE_UI` | Serve embedded SvelteKit UI | `false` |
| `SPARROW_MAX_BODY_BYTES` | Max request body size in bytes (min 1 MiB); oversized bodies return `413` | `5242880` (5 MiB) |

## Auth & Security

| Variable | Purpose | Default |
|----------|---------|---------|
| `SPARROW_API_KEY` | API key for auth, accepted via `X-API-Key`. **Required when `ENVIRONMENT=production`** (`Validate()` refuses to start without it); otherwise optional and all endpoints are open | — (open access) |
| `SPARROW_ALLOW_PRIVATE_NETWORKS` | Allow localhost/private IPs as webhook URLs | `false` |

## Observability

| Variable | Purpose | Default |
|----------|---------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP HTTP export endpoint | — |
| `ENVIRONMENT` | `development` or `production`; `production` enforces `SPARROW_API_KEY` | — |

`Warnings()` also logs a non-fatal advisory if `DATABASE_URL` uses `sslmode=disable` against a non-local host.

## CORS

| Variable | Purpose | Default |
|----------|---------|---------|
| `CORS_ALLOWED_ORIGINS` | Comma-separated CORS origins | — |

## Alerts

| Variable | Purpose | Default |
|----------|---------|---------|
| `SPARROW_SENDGRID_API_KEY` | Activates the bootstrapped SendGrid alert webhook (system events -> email); unset = created inactive with a mock key | — |
| `SPARROW_ALERT_FROM_EMAIL` | Verified SendGrid sender address for alert emails | `alerts@example.com` |
| `SPARROW_ALERT_FROM_NAME` | Sender display name for alert emails | `Sparrow` |

## Retention

| Variable | Purpose | Default |
|----------|---------|---------|
| `SPARROW_EVENT_RETENTION_DAYS` | Purge events (and cascaded deliveries) older than N days via an hourly background job; `0` disables retention (data kept forever) | `0` |

## Database Pools

| Pool | Library | Config | Purpose |
|------|---------|--------|---------|
| sqlx | `jmoiron/sqlx` | MaxOpen=25 | App queries |
| pgxpool | `jackc/pgx/v5` | MaxConns=50, MinConns=10, 30min lifetime | River queue |

## Citations

- `internal/config/config.go`
