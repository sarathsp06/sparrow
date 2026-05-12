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
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded web dashboard on the HTTP port |
| `SPARROW_API_KEY` | No | -- | Require this key in `X-API-Key` header for all API requests |
| `SPARROW_ENCRYPTION_KEY` | Yes | -- | 64-char hex key (32 bytes) for envelope encryption of webhook secrets and headers. Generate with `openssl rand -hex 32` |
| `SPARROW_GRPC_PORT` | No | `50051` | gRPC listen port. macOS users: AirPlay Receiver uses 50051 — change this or disable AirPlay Receiver in System Settings > General > AirDrop & Handoff |
| `SPARROW_HTTP_PORT` | No | `8080` | HTTP / Connect-RPC listen port (also serves the web UI) |
| `SPARROW_ALLOW_PRIVATE_NETWORKS` | No | `false` | Allow localhost/private IP addresses as webhook URLs. Enable for local development and testing |
| `ENVIRONMENT` | No | -- | `development` or `production` (affects logging/OTel) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP HTTP endpoint for traces, metrics, and logs |
| `CORS_ALLOWED_ORIGINS` | No | -- | Comma-separated list of allowed CORS origins (e.g. `https://ui.example.com,https://admin.example.com`). Required when the UI is hosted separately from the API. In production (`ENVIRONMENT=production`), cross-origin requests are blocked by default; in development, all origins are allowed. |

### Frontend Development Variables

These variables are used only when running the SvelteKit web UI in development mode (`npm run dev`). They are **not** part of the Go server configuration.

| Variable | Default | Description |
|----------|---------|-------------|
| `PUBLIC_API_URL` | `/` | API base URL for the frontend dev server to proxy requests |

## Encryption

Sparrow encrypts webhook secrets and sensitive headers at rest using **envelope encryption** (AES-256-GCM). Each record gets its own random data encryption key (DEK), which is wrapped by the master key encryption key (KEK).

### Key Management

The encryption key is provided via the `SPARROW_ENCRYPTION_KEY` environment variable (64-char hex string = 32 bytes). The server will not start without it.

The key is **never** stored in the database. Storing the encryption key next to the data it protects defeats the purpose of encryption at rest. Use a secrets manager, Kubernetes Secret, or `.env` file to provide the key:

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
