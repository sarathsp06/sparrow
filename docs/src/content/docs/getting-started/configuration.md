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
| `DATABASE_URL` | Yes | `postgres://localhost/riverqueue?sslmode=disable` (dev-only fallback) | PostgreSQL connection string |
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded web dashboard on the HTTP port |
| `SPARROW_API_KEY` | No | -- | Require this key in `X-API-Key` header for all API requests |
| `SPARROW_ENCRYPTION_KEYS` | Yes | -- | Keyring entries as comma-separated `<key-id>=<64-char-hex-key>` pairs where each value is a cryptographically random 32-byte (256-bit) key hex-encoded to 64 chars (`key-id` chars: `A-Z`, `a-z`, `0-9`, `_`, `-`) |
| `SPARROW_ENCRYPTION_PRIMARY_KEY_ID` | Yes | -- | Which configured key ID is primary for new encryption |
| `SPARROW_HTTP_PORT` | No | `8080` | HTTP listen port for the REST/OpenAPI API (also serves the web UI) |
| `SPARROW_ALLOW_PRIVATE_NETWORKS` | No | `false` | Allow localhost/private IP addresses as webhook URLs. Enable for local development and testing |
| `ENVIRONMENT` | No | -- | Deployment tag; any value is accepted. Set to `production` to block cross-origin requests by default (see `CORS_ALLOWED_ORIGINS`) and tag logs/OTel; any other value behaves as development. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP HTTP endpoint for traces, metrics, and logs |
| `CORS_ALLOWED_ORIGINS` | No | -- | Comma-separated list of exact browser origins allowed to call the API (e.g. `https://ui.example.com,https://admin.example.com`; trailing slashes are ignored). Required when the UI is [hosted separately](/sparrow/deployment/separate-ui/). When unset: with `ENVIRONMENT=production` every cross-origin request is rejected; otherwise every origin is allowed (local development only). |
| `SPARROW_MAX_BODY_BYTES` | No | `5242880` (5 MiB) | Maximum request body size in bytes. Minimum 1 MiB; larger bodies get `413`. |
| `SPARROW_EVENT_RETENTION_DAYS` | No | `0` (keep forever) | Purge events — and, via cascade, their deliveries — older than this many days. Runs hourly in the background. |

For a single-key deployment, still use the keyring format: `SPARROW_ENCRYPTION_KEYS=main=<64-char-hex-key>` with `SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main`.

### Web UI Variables

These configure the web UI, not the Go server. They matter only when the UI is **not** served by Sparrow itself, i.e. under `npm run dev` or when you [host the UI separately](/sparrow/deployment/separate-ui/). With `SPARROW_SERVE_UI=true` the UI uses the same origin and gets the API key from the server, so none of this applies.

| Setting | Where | Default | Description |
|---------|-------|---------|-------------|
| `apiUrl` | `window.__SPARROW_CONFIG__` in the UI's `/config.js`, set on the static host at deploy time | -- | Absolute URL of the Sparrow server. Overrides `PUBLIC_API_URL`. |
| `apiKey` | `window.__SPARROW_CONFIG__` in `/config.js` | -- | The server's `SPARROW_API_KEY`. Optional: without it, the UI asks for the key on the first `401` and remembers it in the browser. Anyone who can load the UI can read a key placed here. |
| `PUBLIC_API_URL` | Environment variable for `vite build` / `vite dev` | Same origin (`npm run build`), `http://localhost:8080` (`npm run dev`) | Sparrow server URL baked into the bundle at build time. Changing it later requires a rebuild. |

## Encryption

Sparrow encrypts webhook secrets and sensitive headers at rest using **envelope encryption** (AES-256-GCM). Each record gets its own random data encryption key (DEK), which is wrapped by a configured key encryption key (KEK).

### Key Management

Configure encryption with `SPARROW_ENCRYPTION_KEYS` and `SPARROW_ENCRYPTION_PRIMARY_KEY_ID`.

The server will not start without both of these variables.

The key material is **never** stored in the database. Storing the encryption key next to the data it protects defeats the purpose of encryption at rest. Each key value should be a cryptographically random 32-byte (256-bit) key, hex-encoded as 64 characters. `openssl rand -hex 32` is a suitable way to generate one. Use a secrets manager, Kubernetes Secret, or `.env` file to provide the key:

```bash
# Single-key deployment in keyring form
export SPARROW_ENCRYPTION_KEYS="main=$(openssl rand -hex 32)"
export SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main

# Rotation-friendly multi-key deployment
# key IDs must use only A-Z, a-z, 0-9, _ and -
export SPARROW_ENCRYPTION_KEYS="old=$(openssl rand -hex 32),new=$(openssl rand -hex 32)"
export SPARROW_ENCRYPTION_PRIMARY_KEY_ID=new
```

New writes use the primary key ID while decryption accepts all configured keys in the ring. That lets you rotate by adding a new key, switching the primary, and retiring the old key after data has been rewritten.

### What Gets Encrypted

| Field | Stored As | Encrypted |
|-------|-----------|-----------|
| `webhook_secret` | BYTEA | Yes (envelope) |
| `secret_headers` | BYTEA | Yes (envelope) |
| Event payloads | JSONB | No (plaintext) |
| Delivery responses | TEXT | No (plaintext) |

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
| `sparrow_webhook_registrations_total` | Counter | Total number of webhook registrations |
| `sparrow_events_pushed_total` | Counter | Total number of events pushed |
| `sparrow_active_webhooks` | UpDownCounter | Current number of active webhook registrations |

## Default Tenant

A default tenant (`00000000-0000-0000-0000-000000000001`) is auto-created on startup. All operations use this tenant. The tenant infrastructure is retained for future multi-tenant support.

Authentication is optional -- set `SPARROW_API_KEY` to require a shared secret on all API requests. When unset, all endpoints are open (designed for internal deployments behind a VPN). The embedded dashboard is served without authentication and exposes the API key to any browser that can load it -- see [Securing Sparrow](/sparrow/deployment/security/) for the trust model, SSO via an identity-aware proxy, and a hardening checklist.
