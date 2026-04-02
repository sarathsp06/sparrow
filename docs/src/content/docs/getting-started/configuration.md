---
title: Configuration
description: Environment variables and configuration options.
sidebar:
  order: 3
---

All configuration is done via environment variables. No config files needed.

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | -- | PostgreSQL connection string |
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded web dashboard on `:8080` |
| `ENVIRONMENT` | No | -- | `development` or `production` (affects logging/OTel) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP HTTP endpoint for traces, metrics, and logs |
| `CORS_ALLOWED_ORIGINS` | No | -- | Allowed CORS origins for Connect-RPC |
| `PUBLIC_API_URL` | No | `/` | API base URL for the frontend (dev only) |

## Database Pools

Sparrow uses two connection pools:

| Pool | Library | Config | Purpose |
|------|---------|--------|---------|
| sqlx | `jmoiron/sqlx` | MaxOpen=25 | All application queries |
| pgxpool | `jackc/pgx/v5` | MaxConns=50, MinConns=10, 30min lifetime | River job queue only |

## Observability

Sparrow exports traces, metrics, and logs via OpenTelemetry (OTLP). Set `OTEL_EXPORTER_OTLP_ENDPOINT` to enable.

The included `docker-compose.yml` ships with an OTel Collector sidecar. To use your own collector or observability stack, point the env var to your OTLP endpoint:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4318
```

### Exported Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `webhook_registrations` | Counter | Total webhook registrations |
| `events_pushed` | Counter | Total events pushed |
| `webhook_deliveries` | Counter | Deliveries by status |
| `delivery_duration` | Histogram | Delivery response time |
| `queue_depth` | Gauge | Pending jobs per queue |
| `active_webhooks` | Gauge | Currently active webhooks |

## Default Tenant

There is no authentication. A default tenant (`00000000-0000-0000-0000-000000000001`) is auto-created on startup. All operations use this tenant. The tenant infrastructure is retained for future multi-tenant support.
