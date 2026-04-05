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
| `SPARROW_API_KEY` | No | -- | Require this key in `X-API-Key` header for all API requests |
| `SPARROW_ENCRYPTION_KEY` | No | auto-generated | 64-char hex key (32 bytes) for envelope encryption of webhook secrets and headers |
| `ENVIRONMENT` | No | -- | `development` or `production` (affects logging/OTel) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP HTTP endpoint for traces, metrics, and logs |
| `CORS_ALLOWED_ORIGINS` | No | -- | Allowed CORS origins for Connect-RPC |
| `PUBLIC_API_URL` | No | `/` | API base URL for the frontend (dev only) |

## Encryption

Sparrow encrypts webhook secrets and sensitive headers at rest using **envelope encryption** (AES-256-GCM). Each record gets its own random data encryption key (DEK), which is wrapped by the master key encryption key (KEK).

### Key Management

The encryption key is resolved in priority order:

1. **Environment variable** -- `SPARROW_ENCRYPTION_KEY` (64-char hex string = 32 bytes)
2. **Database** -- If no env var is set, Sparrow checks the `system_settings` table for a previously generated key
3. **Auto-generate** -- If neither exists, a random key is generated and persisted to `system_settings`

Encryption works out of the box with zero configuration. For production deployments where you want explicit control over the key, set the environment variable:

```bash
# Generate a key
openssl rand -hex 32

# Set it
export SPARROW_ENCRYPTION_KEY=your-64-char-hex-key
```

### What Gets Encrypted

| Field | Stored As | Encrypted |
|-------|-----------|-----------|
| `webhook_secret` | BYTEA | Yes (envelope) |
| `secret_headers` | BYTEA | Yes (envelope) |
| Event payloads | JSONB | No (plaintext) |
| Delivery responses | TEXT | No (plaintext) |

### Backward Compatibility

Existing data encrypted with the previous direct AES-256-GCM format is automatically detected and decrypted. New writes always use envelope encryption.

## Database Pools

Sparrow uses two connection pools:

| Pool | Library | Config | Purpose |
|------|---------|--------|---------|
| sqlx | `jmoiron/sqlx` | MaxOpen=25 | All application queries |
| pgxpool | `jackc/pgx/v5` | MaxConns=50, MinConns=10, 30min lifetime | River job queue only |

## Observability

Sparrow exports traces, metrics, and logs via OpenTelemetry (OTLP). Set `OTEL_EXPORTER_OTLP_ENDPOINT` to point to your collector:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://your-otel-collector:4318
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

A default tenant (`00000000-0000-0000-0000-000000000001`) is auto-created on startup. All operations use this tenant. The tenant infrastructure is retained for future multi-tenant support.

Authentication is optional -- set `SPARROW_API_KEY` to require a shared secret on all API requests. When unset, all endpoints are open (designed for internal deployments behind a VPN).
