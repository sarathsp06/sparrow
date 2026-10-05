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
| `SPARROW_API_KEY` | No | -- | Require this key in `X-API-Key` header for all API requests. Must be at least 32 characters when `ENVIRONMENT=production` (the server refuses to start otherwise); shorter keys log a warning in other environments. Generate with `openssl rand -hex 32`. |
| `SPARROW_ENCRYPTION_KEYS` | Yes | -- | Keyring entries as comma-separated `<key-id>=<64-char-hex-key>` pairs where each value is a cryptographically random 32-byte (256-bit) key hex-encoded to 64 chars (`key-id` chars: `A-Z`, `a-z`, `0-9`, `_`, `-`) |
| `SPARROW_ENCRYPTION_PRIMARY_KEY_ID` | Yes | -- | Which configured key ID is primary for new encryption |
| `SPARROW_HTTP_PORT` | No | `8080` | HTTP listen port for the REST/OpenAPI API (also serves the web UI) |
| `SPARROW_TOKEN_DEFAULT_TTL` | No | `2160h` (90 days) | Lifetime of a tenant-wide access token created without `ttl_seconds` or `never_expires` (including browser sign-ins and invites). Go duration. `0` means such tokens never expire (the pre-upgrade behaviour). |
| `SPARROW_MAX_CAPTURED_RESPONSE_BYTES` | No | `1048576` (1 MiB) | Stored response body limit for webhooks with `capture_response_body` enabled (others store 1 KiB). Minimum 1024. |
| `SPARROW_ALLOWED_NETWORKS` | No | -- | Comma-separated CIDRs or bare IPs (e.g. `10.20.0.0/16,fd12::/48`). Webhook deliveries may reach these networks in addition to public addresses; loopback, cloud metadata, and the rest of private space stay blocked. This is the recommended way to deliver to internal services on a VPN. Invalid entries fail startup. When an allowlist is set, `.internal` and `.local` hostnames are no longer blocked by name; their resolved addresses are checked instead. |
| `SPARROW_ALLOW_PRIVATE_NETWORKS` | No | `false` | Allow all private IP addresses as webhook URLs (for local development and testing). Cloud metadata endpoints are still blocked; use `SPARROW_ALLOWED_NETWORKS` for targeted access in production. |
| `ENVIRONMENT` | No | -- | Deployment tag; any value is accepted. Set to `production` to block cross-origin requests by default (see `CORS_ALLOWED_ORIGINS`) and tag logs/OTel; any other value behaves as development. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP collector URL for traces, metrics, and logs (e.g. `http://collector:4318`; `https://` for TLS). Export is off when unset. See [Observability](#observability). |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | No | `http/protobuf` | OTLP transport: `http/protobuf` or `grpc`. Any other value disables export and logs a warning at startup. |
| `CORS_ALLOWED_ORIGINS` | No | -- | Comma-separated list of exact browser origins allowed to call the API (e.g. `https://ui.example.com,https://admin.example.com`; trailing slashes are ignored). Required when the UI is [hosted separately](/sparrow/deployment/separate-ui/). When unset: with `ENVIRONMENT=production` every cross-origin request is rejected; otherwise every origin is allowed (local development only). |
| `SPARROW_MAX_BODY_BYTES` | No | `5242880` (5 MiB) | Maximum request body size in bytes. Minimum 1 MiB; larger bodies get `413`. |
| `SPARROW_EVENT_RETENTION_DAYS` | No | `0` (keep forever) | Purge events — and, via cascade, their deliveries — older than this many days. Runs hourly in the background. |
| `SPARROW_AUTO_DISABLE_AFTER` | No | `120h` (5 days) | Pause a webhook automatically once its receiver has failed every attempt for this long (and at least `SPARROW_AUTO_DISABLE_MIN_FAILURES` attempts in a row). Like a manual pause, deliveries are then held as `paused` until retried. Go duration; `0` turns automatic disabling off. See [Automatic disabling](/sparrow/guides/webhook-health-alerts/#automatic-disabling). |
| `SPARROW_AUTO_DISABLE_MIN_FAILURES` | No | `10` | Minimum run of consecutive failed attempts before a webhook can be auto-disabled, so a receiver whose only failure was long ago is not paused by the next one. |
| `SPARROW_METRICS_ENABLED` | No | `true` | Serve every OpenTelemetry metric in Prometheus format at `GET /metrics` (no API key, like `/health`), with or without OTLP export. |
| `SPARROW_AUTO_REGISTER_EVENTS` | No | `false` | When `true`, pushing an event whose type is not registered creates a schema-less event type instead of returning `404`. Meant for local development (`make run` turns it on); event types are never deleted, so in production a producer typo would become a permanent name. See [Event Type Versions](/sparrow/guides/event-type-versioning/#unregistered-event-names). |
| `SPARROW_AI_PROVIDER` | No | `anthropic` | Which chat API drafts templates: `anthropic` or `openai` (any OpenAI-compatible server). See [AI template drafting](#ai-template-drafting). |
| `SPARROW_AI_API_KEY` | No | -- | Provider API key. With `anthropic`, setting it turns AI drafting on. With `openai` it is optional and sent as a Bearer token (hosted services need it, local servers usually don't). |
| `SPARROW_AI_MODEL` | No | `claude-haiku-4-5` (`anthropic` only) | Model used for drafting. Must be set when using `openai`; there is no default. |
| `SPARROW_AI_BASE_URL` | No | -- | API root URL. Must be set when using `openai`, including the `/v1` path (e.g. `http://localhost:11434/v1`). With `anthropic`, only for a gateway or proxy. |

For a single-key deployment, still use the keyring format: `SPARROW_ENCRYPTION_KEYS=main=<64-char-hex-key>` with `SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main`.

### Web UI Variables

These configure the web UI, not the Go server. They matter only when the UI is **not** served by Sparrow itself, i.e. under `npm run dev` or when you [host the UI separately](/sparrow/deployment/separate-ui/). With `SPARROW_SERVE_UI=true` the embedded UI uses the same origin; when `SPARROW_API_KEY` is set it shows a sign-in prompt on the first `401`.

| Setting | Where | Default | Description |
|---------|-------|---------|-------------|
| `apiUrl` | `window.__SPARROW_CONFIG__` in the UI's `/config.js`, set on the static host at deploy time | -- | Absolute URL of the Sparrow server. Overrides `PUBLIC_API_URL`. |
| `apiKey` | `window.__SPARROW_CONFIG__` in `/config.js` | -- | The master key (`SPARROW_API_KEY`) or a tenant-wide access token. Optional: without it, the UI shows a sign-in prompt on the first `401` and remembers the credential in the browser. Anyone who can load the UI can read a key placed here. |
| `PUBLIC_API_URL` | Environment variable for `vite build` / `vite dev` | Same origin (`npm run build`), `http://localhost:8080` (`npm run dev`) | Sparrow server URL baked into the bundle at build time. Changing it later requires a rebuild. |

## AI template drafting

The subscription editor can draft a payload `transform_template` from a plain-language description. AI drafting is optional, and there are three ways to run it.

| Setup | What you get |
|-------|--------------|
| Nothing set | **Copy prompt for AI**: Sparrow builds the prompt (`POST /v1/subscriptions:draftTemplatePrompt`) and you paste it into any chat assistant. Sparrow sends nothing anywhere. |
| Anthropic | **Draft with AI** in the editor, using the Anthropic API. |
| OpenAI-compatible server | **Draft with AI** using Ollama, vLLM, LM Studio, llama.cpp, OpenRouter, OpenAI, or any other server that serves `/v1/chat/completions`. |

### Anthropic

```bash
SPARROW_AI_API_KEY=sk-ant-...
# Optional:
SPARROW_AI_MODEL=claude-sonnet-5-5      # default: claude-haiku-4-5
SPARROW_AI_BASE_URL=https://gateway.internal/anthropic   # only behind a gateway or proxy
```

`SPARROW_AI_PROVIDER` defaults to `anthropic`, so the API key alone turns drafting on.

### OpenAI-compatible server

```bash
SPARROW_AI_PROVIDER=openai
SPARROW_AI_BASE_URL=http://localhost:11434/v1   # Ollama; e.g. http://vllm:8000/v1, https://openrouter.ai/api/v1
SPARROW_AI_MODEL=qwen2.5-coder:7b               # a model that server serves
# Optional, for hosted services:
SPARROW_AI_API_KEY=...
```

`SPARROW_AI_BASE_URL` and `SPARROW_AI_MODEL` are both required. Local servers usually need no key.

### Startup checks

The server refuses to start when:

- `SPARROW_AI_PROVIDER` is anything other than `anthropic` or `openai`.
- `SPARROW_AI_PROVIDER=openai` with a key or base URL set, but `SPARROW_AI_BASE_URL` or `SPARROW_AI_MODEL` missing.
- `SPARROW_AI_BASE_URL` is not an absolute URL (scheme and host).

To confirm drafting is on, look for the `AI drafting` line in the startup banner, or call `GET /v1/capabilities`.

### Choosing a model

Templates are short. Sparrow renders each draft against the sample payload and asks the model to repair it, up to three rounds, so a light model is usually enough. If drafts often use up the repair rounds, switch to a larger model.

### What is sent to the model

Only the event type's JSON Schema, the sample payload (registered, or provided in the request), the template helper catalog, and the text in the request: your description, an example body, a recipe's destination format, and an excerpt of a documentation URL if you give one. Sparrow fetches that URL under the same network policy as deliveries, so private and cloud-metadata addresses are refused unless allowed. Sparrow never sends stored events, headers, or secrets.

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

Sparrow exports traces, metrics, and logs via OpenTelemetry (OTLP). Set `OTEL_EXPORTER_OTLP_ENDPOINT` to point to your collector. OTLP/HTTP is the default:

```bash
OTEL_EXPORTER_OTLP_ENDPOINT=http://your-otel-collector:4318
```

To export over OTLP/gRPC instead, set the protocol and use the collector's gRPC port:

```bash
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
OTEL_EXPORTER_OTLP_ENDPOINT=http://your-otel-collector:4317
```

The URL scheme controls TLS: `http://` sends plaintext, `https://` uses TLS. The other standard OpenTelemetry exporter variables work as well, for example `OTEL_EXPORTER_OTLP_HEADERS` for a hosted backend's API key, `OTEL_EXPORTER_OTLP_CERTIFICATE` for a private CA, or `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` to send one signal somewhere else. A bare `host:port` without a scheme is still accepted and is sent as plaintext.

### Prometheus

Every metric below is also served for scraping at `GET /metrics` in Prometheus
text format, whether or not OTLP export is on, together with the standard Go
runtime and process metrics. It needs no API key (like `/health`) and exposes
only aggregate counts; set `SPARROW_METRICS_ENABLED=false` to turn it off, or
keep `/metrics` off the public network at your reverse proxy.

```yaml
scrape_configs:
  - job_name: sparrow
    static_configs:
      - targets: ["sparrow:8080"]
```

### Exported Metrics

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `sparrow_delivery_attempts_total` | Counter | `result`, `error_category` | Delivery attempts that reached the receiver (or failed to). Success rate: `sum(rate(sparrow_delivery_attempts_total{result="success"}[5m])) / sum(rate(sparrow_delivery_attempts_total[5m]))` |
| `sparrow_delivery_attempt_duration_seconds` | Histogram | `result` | Time from sending an attempt to the receiver's response |
| `sparrow_queue_jobs` | Gauge | `queue`, `state` | River jobs waiting (`available`, `scheduled`, `retryable`) or `running`, per queue: the delivery backlog |
| `sparrow_webhooks` | Gauge | `health`, `status` | Webhooks by health and status (`active`, `paused`, `auto_disabled`) |
| `sparrow_webhooks_auto_disabled_total` | Counter | | Webhooks Sparrow paused automatically because their receiver kept failing |
| `sparrow_template_errors_total` | Counter | `on_transform_error` | Payload transform templates that failed to render |
| `sparrow_events_pushed_total` | Counter | | Events pushed |
| `sparrow_webhook_registrations_total` | Counter | | Webhook registrations |

HTTP server and client metrics (`http_server_*`, `http_client_*`) and database
pool metrics (`db_sql_*`) come from the OpenTelemetry instrumentation.

## Default Tenant

A default tenant (`00000000-0000-0000-0000-000000000001`) is auto-created on startup. All operations use this tenant. The tenant infrastructure is retained for future multi-tenant support.

Authentication is optional -- set `SPARROW_API_KEY` to require a shared secret on all API requests. When unset, all endpoints are open (designed for internal deployments behind a VPN). When `SPARROW_API_KEY` is set, the embedded dashboard shows a sign-in prompt on the first `401` -- see [Securing Sparrow](/sparrow/deployment/security/) for the trust model, SSO via an identity-aware proxy, and a hardening checklist.
