---
title: Configuration
description: "Every Sparrow environment variable: the minimum to start, access, outbound network, deliveries, throughput, encryption keys, OpenTelemetry and AI drafting."
sidebar:
  order: 5
---

Sparrow is configured entirely with environment variables. There's no config
file.

**Empty and `-` mean unset.** A variable set to an empty value, or to a lone
`-`, is treated as if it were not set at all, so its default applies. Use `-`
on platforms that cannot store an empty value or cannot delete a variable once
it exists (some secret managers and PaaS dashboards). The rule covers every
variable on this page, `OTEL_*` included: Sparrow unsets a blank one at
startup, so the OpenTelemetry exporters see it as unset too. No Sparrow
setting gives an empty value a meaning of its own.

## Minimum to start

Three variables are required. The server refuses to start without the two
encryption ones:

```bash
DATABASE_URL="postgres://sparrow:secret@db:5432/sparrow?sslmode=require"
SPARROW_ENCRYPTION_KEYS="main=$(openssl rand -hex 32)"   # generate once, keep in a secret manager
SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main
```

| Variable | What to set |
|----------|-------------|
| `DATABASE_URL` | PostgreSQL 15+ connection string. Migrations run on startup. (If unset, it falls back to `postgres://localhost/riverqueue?sslmode=disable`, which only suits a local experiment.) |
| `SPARROW_ENCRYPTION_KEYS` | The keyring: comma-separated `<key-id>=<64 hex chars>` pairs, each a random 32-byte key. Key IDs use `A-Z a-z 0-9 _ -`. See [Encryption](#encryption). |
| `SPARROW_ENCRYPTION_PRIMARY_KEY_ID` | Which key ID encrypts new data. Must be one of the IDs in the keyring. |

For anything reachable by more than you, also set `SPARROW_API_KEY` and
`SPARROW_ENVIRONMENT=production`. The [production guide](/sparrow/deployment/production/)
lists the rest. To serve the web UI from the same server, set
`SPARROW_SERVE_UI=true`.

:::caution[Without `SPARROW_API_KEY` the API is open]
With `SPARROW_API_KEY` unset, every `/v1` endpoint answers anyone who can
reach the port, and unless `SPARROW_ENVIRONMENT` is `production` CORS also allows
every origin, so any web page a user opens can call the API from their
browser. Only run like this on your own machine or a fully trusted network.

`SPARROW_ENVIRONMENT=production` turns the production checks on (case and
surrounding spaces don't matter). Any other value counts as development: no
key is required and CORS stays open. A value that looks like production but
isn't, such as `prod`, logs a warning at startup.
:::

## All variables

Grouped by what they control. Durations use Go syntax: `90s`, `20ms`, `1m`,
`120h`.

### Server and UI

| Variable | Default | Description |
|----------|---------|-------------|
| `SPARROW_HTTP_PORT` | `8080` | Port for the REST API, health endpoints, `/metrics` and (if enabled) the UI. Falls back to `PORT`. |
| `SPARROW_SERVE_UI` | `false` | Serve the embedded web dashboard on the same port. |
| `SPARROW_LOG_LEVEL` | `info` | Lowest level the server logs: `debug`, `info`, `warn` or `error`. `info` writes a line for every event and delivery; `warn` keeps only problems and saves that work on a busy instance. Logs are JSON on stdout (and exported through OpenTelemetry when it is enabled). An unknown value stops startup. |
| `SPARROW_ENVIRONMENT` | unset | Set `production` to reject cross-origin requests unless `SPARROW_CORS_ALLOWED_ORIGINS` allows them, to require a 32+ character `SPARROW_API_KEY`, to refuse an all-zeros encryption key, and to tag logs and telemetry. Case doesn't matter; any other value behaves as development, and one starting with `prod` logs a warning. |
| `SPARROW_CORS_ALLOWED_ORIGINS` | unset | Comma-separated browser origins allowed to call the API, e.g. `https://ui.example.com`. Needed when the UI is [hosted separately](/sparrow/deployment/separate-ui/). Unset: none is allowed with `SPARROW_ENVIRONMENT=production`; otherwise every origin is allowed when `SPARROW_API_KEY` is set, and only `localhost` and loopback origins when it is not. |
| `SPARROW_SHUTDOWN_TIMEOUT` | `10s` | Budget for a graceful shutdown on `SIGTERM`/`SIGINT`, HTTP drain and queue stop together. `/ready` turns `503` at once, but the listener closes right after (on Kubernetes add a `preStop` sleep; see [Graceful shutdown](/sparrow/deployment/production/#graceful-shutdown)); in-flight HTTP requests finish, then the queue stops taking jobs and waits for running ones; deliveries still running near the end are cancelled and retried later (a cancelled attempt is not recorded as a failure). Keep it below the orchestrator's kill deadline minus any `preStop` sleep (Kubernetes: `terminationGracePeriodSeconds`, 30s by default). |
| `SPARROW_MAX_BODY_BYTES` | `5242880` (5 MiB) | Largest accepted request body; larger ones get `413`. Minimum 1 MiB. API requests are also limited to 1 MiB each (event-type import: 5 MiB), whatever this is set to. |

### Access

| Variable | Default | Description |
|----------|---------|-------------|
| `SPARROW_API_KEY` | unset (API open) | The master key. When set, every `/v1` request needs it or an access token, in `X-API-Key` or `Authorization: Bearer`. Required, and at least 32 characters, with `SPARROW_ENVIRONMENT=production` (the server won't start otherwise). Generate with `openssl rand -hex 32`. See [Access tokens and invites](/sparrow/deployment/access/). |
| `SPARROW_TOKEN_DEFAULT_TTL` | `2160h` (90 days) | Lifetime of a tenant-wide access token created without `ttl_seconds` or `never_expires`, including browser sign-ins and invites. `0` means such tokens never expire. |
| `SPARROW_PORTAL_RETRY` | `true` | Whether consumer (portal) tokens may retry their own deliveries. `false`: the retry calls under `/portal/api/` answer `403`, `GET /v1/capabilities` reports `portal.retry: false`, and the portal shows its Retry buttons disabled. Consumers still see every delivery; the master key and tenant-wide tokens can still retry. See [Embedding the Consumer Portal](/sparrow/guides/portal-embedding/#letting-consumers-retry). |

### Outbound network

Sparrow refuses to deliver to loopback, private, link-local and cloud metadata
addresses unless you allow them.

| Variable | Default | Description |
|----------|---------|-------------|
| `SPARROW_ALLOWED_NETWORKS` | unset | Comma-separated CIDRs or IPs, e.g. `10.20.0.0/16,fd12::/48`, that deliveries may reach in addition to public addresses. Loopback, cloud metadata and the rest of private space stay blocked. The recommended way to deliver to internal services. Invalid entries stop startup. When set, `.internal` and `.local` host names are checked by their resolved address instead of being blocked by name. |
| `SPARROW_ALLOW_PRIVATE_NETWORKS` | `false` | Allow every private and loopback address. For local development and tests. Cloud metadata stays blocked. |
| `SPARROW_TLS_REVOCATION` | `soft` | Check the intermediates in each receiver's TLS chain against their issuers' CRLs, which Go doesn't do on its own. `soft` allows a connection whose CRL can't be downloaded or verified, `hard` refuses it (`tls_error`), `off` skips the check. A revoked intermediate is refused in both `soft` and `hard`. See [Certificate revocation](/sparrow/reference/security/#certificate-revocation). |
| `SPARROW_CRL_MAX_AGE` | `6h` | How old a cached CRL may get before the next delivery that uses it refreshes it in the background. Minimum `5m`. There is no scheduled refresh: `POST /v1/tls/crls:refresh` refreshes every instance at once. |

### Events

| Variable | Default | Description |
|----------|---------|-------------|
| `SPARROW_AUTO_REGISTER_EVENTS` | `false` | Pushing an unregistered event type creates a schema-less type instead of returning `404`. For local development (`make run` turns it on): event types are never deleted, so in production a typo becomes a permanent name. See [Unregistered event names](/sparrow/guides/event-type-versioning/#unregistered-event-names). |
| `SPARROW_EVENT_RETENTION_DAYS` | `0` (keep forever) | Delete events older than this many days, and their deliveries with them. Runs hourly. |

### Deliveries and webhook health

| Variable | Default | Description |
|----------|---------|-------------|
| `SPARROW_MAX_CAPTURED_RESPONSE_BYTES` | `1048576` (1 MiB) | How much of a receiver's response body is stored for webhooks with `capture_response_body` on (others store 1 KiB). Minimum 1024. |
| `SPARROW_HEALTH_EVAL_INTERVAL` | `1m` | How often delivery outcomes are folded into each webhook's health. The health label, `sparrow.webhook.health_changed` alerts and automatic disabling lag by at most this much. Each run only reads outcomes since the last one, so busy webhooks don't make it slower. |
| `SPARROW_AUTO_DISABLE_AFTER` | `120h` (5 days) | Pause a webhook once its receiver has failed every attempt for this long (and at least `SPARROW_AUTO_DISABLE_MIN_FAILURES` in a row). Its deliveries are then held as `paused`. `0` turns it off. See [Automatic disabling](/sparrow/guides/webhook-health-alerts/#automatic-disabling). |
| `SPARROW_AUTO_DISABLE_MIN_FAILURES` | `10` | Minimum run of consecutive failed attempts before a webhook can be auto-disabled. |

### Throughput

| Variable | Default | Description |
|----------|---------|-------------|
| `SPARROW_EVENT_WORKERS` | `20` | Concurrent fan-out jobs. Each pushed event is one job that finds its subscriptions and creates the deliveries. |
| `SPARROW_WEBHOOK_WORKERS` | `20` | Concurrent deliveries; also sizes the connection pool per receiver host. Raise it when receivers are slow: 20 workers against receivers taking 100 ms top out around 200 deliveries/s. |
| `SPARROW_QUEUE_DB_MAX_CONNS` | `50` | Postgres connections for the queue workers. Keep it at or above the total worker count. |
| `SPARROW_DB_MAX_CONNS` | `25` | Postgres connections for API requests and other application queries. |
| `SPARROW_QUEUE_FETCH_COOLDOWN` | `20ms` | Minimum gap between two job fetches on a queue. `workers / cooldown` caps each queue: 20 workers at `20ms` is about 1,000 jobs/s. Lower values query Postgres more often, under load only. |
| `SPARROW_QUEUE_RESCUE_AFTER` | `7m` | How long a job may stay running before Sparrow treats its instance as crashed and retries it. After a crash, deliveries that were in flight wait this long, and the receiver may get them twice (same `X-Sparrow-Delivery-ID`). Must be longer than `5m`, the longest delivery request. |

### Observability

| Variable | Default | Description |
|----------|---------|-------------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | unset (export off) | OTLP collector URL for traces, metrics and logs, e.g. `http://collector:4318`. See [Observability](#observability). |
| `OTEL_EXPORTER_OTLP_PROTOCOL` | `http/protobuf` | `http/protobuf` or `grpc`. With an endpoint set, any other value (or an unknown `SPARROW_OTLP_SIGNALS` entry) makes OpenTelemetry setup fail with a warning: the server starts, but without export, tracing or the `/metrics` endpoint. |
| `SPARROW_OTLP_SIGNALS` | `traces,metrics,logs` | Which signals are exported. `traces` for a traces-only backend such as Jaeger or Tempo. |
| `SPARROW_METRICS_ENABLED` | `true` | Serve all metrics in Prometheus format at `GET /metrics`. The endpoint needs no API key, and its labels include consumer names and webhook IDs: set `false`, or block `/metrics` at your proxy, if that is a concern. |

### AI drafting

| Variable | Default | Description |
|----------|---------|-------------|
| `SPARROW_AI_ENABLED` | `false` | The on/off switch for **Draft with AI**. `true` turns it on; the provider variables below are then required, and a missing one makes the draft request fail with a message naming it (the server still starts). Unset, empty, `-` or `false`: off. **Copy prompt** works either way. See [Turning drafting on and off](#turning-drafting-on-and-off). |
| `SPARROW_AI_PROVIDER` | `anthropic` | `anthropic` or `openai` (any OpenAI-compatible server). |
| `SPARROW_AI_API_KEY` | unset | Provider API key. Required for `anthropic`; optional for a local `openai` server. Read only when `SPARROW_AI_ENABLED=true`. |
| `SPARROW_AI_MODEL` | `claude-haiku-4-5` for `anthropic` | Model name. Required for `openai`. |
| `SPARROW_AI_BASE_URL` | unset | API root. Required for `openai`; optional proxy override for `anthropic`. |
| `SPARROW_AI_TIMEOUT` | `3m` | Longest one draft may take, repair rounds included; past it the request answers 504. A reverse proxy in front of Sparrow needs a read timeout at least this long. |

See [AI template drafting](#ai-template-drafting) below for what each setup
does.

## Web UI settings

These configure the web UI, not the Go server. They matter only when the UI is **not** served by Sparrow itself, i.e. under `npm run dev` or when you [host the UI separately](/sparrow/deployment/separate-ui/). With `SPARROW_SERVE_UI=true` the embedded UI uses the same origin; when `SPARROW_API_KEY` is set it shows a sign-in prompt on the first `401`.

| Setting | Where | Default | Description |
|---------|-------|---------|-------------|
| `apiUrl` | `window.__SPARROW_CONFIG__` in the UI's `/config.js`, set on the static host at deploy time | -- | Absolute URL of the Sparrow server. Overrides `PUBLIC_API_URL`. |
| `apiKey` | `window.__SPARROW_CONFIG__` in `/config.js` | -- | The master key (`SPARROW_API_KEY`) or a tenant-wide access token. Optional: without it, the UI shows a sign-in prompt on the first `401` and remembers the credential in the browser. Anyone who can load the UI can read a key placed here. |
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

New writes use the primary key ID while decryption accepts all configured keys in the ring. That lets you rotate by adding a new key and switching the primary. Keep the old key in the ring: existing secrets stay encrypted under it, and nothing rewraps them yet. See [Upgrading to 1.0](/sparrow/deployment/upgrading-to-1-0/#encryption-keyring-never-remove-an-old-key).

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

Jaeger and Grafana Tempo accept OTLP traces but not metrics or logs. Pointing Sparrow straight at one of them logs `failed to upload metrics: ... unknown service ...MetricsService` every 30 seconds. Send only traces:

```bash
SPARROW_OTLP_SIGNALS=traces
```

Metrics stay available for scraping at `/metrics` whenever `SPARROW_METRICS_ENABLED` is on, whatever `SPARROW_OTLP_SIGNALS` says.

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

## AI template drafting

The subscription template editor can help write a `transform_template`. It
works at two levels:

- **Always**, the editor offers **Copy prompt**
  (`POST /v1/subscriptions:draftTemplatePrompt`): a prompt grounded in the event
  type's schema and sample payload and in Sparrow's template helpers, to paste
  into any chat assistant.
- **With `SPARROW_AI_ENABLED=true`** and a provider configured, it adds **Draft with AI**
  (`POST /v1/subscriptions:draftTemplate`). You describe the body the receiver
  should get (optionally with an example body, a sample payload, a shipped
  recipe's format, or a documentation URL), and Sparrow drafts the template,
  renders it against the sample payload, and asks the model to repair it, up to
  three rounds, until it renders. Drafts are short, so a small model is usually
  enough.

`GET /v1/capabilities` tells clients which of the two is available.

**What leaves your server:** the event type's schema, the sample payload, and
the text of the request. Stored events, headers and secrets are never sent.
A documentation URL is fetched by Sparrow under the same network rules as
deliveries, so private and cloud metadata addresses are refused unless allowed.

### Anthropic

```bash
SPARROW_AI_ENABLED=true
SPARROW_AI_API_KEY=sk-ant-...
# SPARROW_AI_MODEL=claude-haiku-4-5   # the default; raise it if drafts often need repairs
# SPARROW_AI_BASE_URL=https://llm-gateway.internal   # only for a gateway or proxy
```

### An OpenAI-compatible server

Ollama, vLLM, LM Studio, llama.cpp, OpenRouter, OpenAI and others:

```bash
SPARROW_AI_ENABLED=true
SPARROW_AI_PROVIDER=openai
SPARROW_AI_BASE_URL=http://localhost:11434/v1   # Ollama; e.g. http://vllm:8000/v1, https://openrouter.ai/api/v1
SPARROW_AI_MODEL=qwen2.5-coder:7b               # the name the server serves
# SPARROW_AI_API_KEY=...                        # sent as a Bearer token when set
```


### Turning drafting on and off

`SPARROW_AI_ENABLED` is the only switch. The provider variables configure
drafting but never turn it on by themselves.

| `SPARROW_AI_ENABLED` | Result |
|----------------------|--------|
| unset, empty, `-` or `false` | **Off.** The editor shows **Copy prompt** only. Provider variables are ignored, so you can leave a key in place. |
| `true` | **On.** The editor shows **Draft with AI** next to **Copy prompt**. The provider variables below must be set. |

With `SPARROW_AI_ENABLED=true`, these must be set, or drafting fails:

| Provider | Must be set | Optional |
|----------|-------------|----------|
| `anthropic` (default) | `SPARROW_AI_API_KEY` | `SPARROW_AI_MODEL` (default `claude-haiku-4-5`), `SPARROW_AI_BASE_URL` (gateway or proxy) |
| `openai` | `SPARROW_AI_PROVIDER=openai`, `SPARROW_AI_BASE_URL`, `SPARROW_AI_MODEL` | `SPARROW_AI_API_KEY` (sent as a Bearer token) |

A missing or invalid provider setting does not stop the server. Instead:

- `POST /v1/subscriptions:draftTemplate` answers `503` with a message naming
  the setting, e.g. `AI drafting is enabled but not usable:
  SPARROW_AI_ENABLED=true but SPARROW_AI_API_KEY is not set`.
- `GET /v1/capabilities` reports `ai_drafting.enabled: true` with the same text
  in `ai_drafting.problem`, and the editor shows it in the AI panel.
- The startup banner and a startup warning say the same.
- **Copy prompt** keeps working.

The same happens for an unknown `SPARROW_AI_PROVIDER` or a
`SPARROW_AI_BASE_URL` that is not an absolute URL. A value of
`SPARROW_AI_ENABLED` that is not a boolean (`true`/`false`, `1`/`0`) stops
startup, like any other malformed variable.

If a key or base URL is set while `SPARROW_AI_ENABLED` is not `true`, startup
logs a warning that **Draft with AI** is off.

## Default Tenant

A default tenant (`00000000-0000-0000-0000-000000000001`) is auto-created on startup. All operations use this tenant. The tenant infrastructure is retained for future multi-tenant support.

Authentication is optional -- set `SPARROW_API_KEY` to require a shared secret on all API requests. When unset, all endpoints are open (designed for internal deployments behind a VPN). When `SPARROW_API_KEY` is set, the embedded dashboard shows a sign-in prompt on the first `401` -- see [Securing Sparrow](/sparrow/deployment/security/) for the trust model, SSO via an identity-aware proxy, and a hardening checklist.
