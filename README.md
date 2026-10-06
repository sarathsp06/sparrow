<div align="center">

<img src="./web/src/lib/assets/favicon.svg" width="88" alt="Sparrow logo" />

# Sparrow

**Self-hosted webhook delivery with retries, signing, health tracking, and one infrastructure dependency: PostgreSQL.**

[![Release](https://img.shields.io/github/v/release/sarathsp06/sparrow?sort=semver&style=flat-square)](https://github.com/sarathsp06/sparrow/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/sarathsp06/sparrow/ci.yml?branch=main&style=flat-square&label=CI)](https://github.com/sarathsp06/sparrow/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Docker](https://img.shields.io/badge/Docker-ghcr.io%2Fsarathsp06%2Fsparrow%3Alatest-2496ED?style=flat-square&logo=docker&logoColor=white)](https://github.com/sarathsp06/sparrow/pkgs/container/sparrow?tag=latest)
[![Docs](https://img.shields.io/badge/docs-Starlight-F97316?style=flat-square)](https://sarathsp06.github.io/sparrow/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

[Quick Start](#quick-start) · [Local Dev](#local-development) · [CLI](#quick-start-with-the-cli) · [Why Sparrow](#why-sparrow) · [Security](#security) · [Architecture](#architecture) · [Satellites](#satellites) · [Docs](#docs) · [Contributing](#contributing)

</div>

Sparrow is a self-hosted webhook delivery platform for teams that want dependable outbound webhooks without routing payloads through a third-party service. Register event types, attach webhooks, push events, and let Sparrow handle async fan-out, delivery retries, signing, health tracking, and delivery history.

It ships as one Go server with an embedded Svelte dashboard. No Redis, no broker, no extra queue to babysit. PostgreSQL is the only required dependency; [River](https://riverqueue.com) runs the job queue in that same database.

```text
producers / curl / UI / SDKs
            ↓
   Sparrow REST API + embedded dashboard
            ↓
     PostgreSQL + River job queue
            ↓
 signed HTTP deliveries to subscriber endpoints
```

## Features

- **Async fan-out** — push one event occurrence, match subscriptions, and deliver to every webhook that should receive it.
- **At-least-once delivery** — retryable failures are retried automatically with delivery state persisted in PostgreSQL.
- **Deterministic bulk actions** — snapshot-based batch re-push and retry, so “what you filtered” is exactly “what gets acted on”.
- **Per-webhook health tracking** — healthy / degraded / unhealthy state with rolling summaries and delivery history.
- **Cryptographic signing** — HMAC-SHA256 on every delivery, plus Ed25519 when a webhook uses `signature_type: ed25519`, both in [Standard Webhooks](https://www.standardwebhooks.com/) format.
- **Encryption at rest** — webhook secrets and sensitive headers are envelope-encrypted with AES-256-GCM.
- **Payload transforms** — per-subscription Go templates let you reshape payloads per consumer.
- **Adapter recipes** — event → Slack/Discord/PagerDuty/ntfy/ClickHouse/Twilio/SendGrid/CloudEvents as pure subscription config via `sparrow use`; templates render server-side per delivery.
- **Soft schema validation** — invalid payloads produce warnings instead of being dropped.
- **Versioned event types** — every schema change is a kept version, events record the version they were accepted under, and a schema change that could break a subscription's transform needs an explicit opt-in.
- **Promote definitions between environments** — export event types to one JSON file and import it elsewhere, with a dry-run preview, from the UI, the CLI, or the API.
- **Visible template errors** — a transform that cannot render fails its delivery instead of sending the wrong body, and never counts against the receiver's health.
- **Pause a subscription** — hold its deliveries as paused rows, then resume and retry what was held.
- **Embedded admin UI** — webhooks, events, deliveries, health, and event-instance inspection in one dashboard.
- **Consumer self-service portal** — hand each customer a scoped, expiring link to register their own endpoints and inspect and retry their own deliveries, isolated to their consumer.
- **OpenAPI-first API** — REST on `:8080`, interactive docs at `/docs`, committed spec at [`api/openapi.yaml`](api/openapi.yaml).
- **Operationally boring** — one database, one binary, and a Docker image/Compose file.
- **OpenTelemetry built in** — traces, metrics, and logs, including propagation through async jobs.

## Why Sparrow

**Use Sparrow when you want:**

- self-hosted webhook infrastructure for internal systems or product backends
- delivery guarantees, retries, replay, and inspection without building it yourself
- real security controls for secrets, signatures, and outbound network access
- a Postgres-only operational model instead of Postgres + Redis + separate workers
- an embeddable, token-scoped consumer portal so customers self-serve their own webhooks

**Reach for something else when you need:**

- a fully managed webhook SaaS
- a multi-tenant SaaS control plane instead of a self-hosted operator tool

## Quick Start

The fastest path is Docker Compose -- for local evaluation, no clone or secrets needed:

```bash
curl -O https://raw.githubusercontent.com/sarathsp06/sparrow/main/deploy/docker-compose.yml
docker compose up -d
```

Open <http://localhost:8080> for the UI. The REST API is on the same address, and interactive API docs are at <http://localhost:8080/docs>. Clean up with `docker compose down -v`.

> [!NOTE]
> The Compose file is deliberately a **local evaluation** setup: open API (no `SPARROW_API_KEY`), a well-known encryption key, and the port bound to `127.0.0.1` only. For a real deployment, see [Production Deployment](https://sarathsp06.github.io/sparrow/deployment/production/).

For per-person credentials and one-time invite links instead of sharing the master key, see [Access: Tokens and Invites](https://sarathsp06.github.io/sparrow/deployment/access/). To host the UI on its own domain instead, see [Hosting the UI separately](https://sarathsp06.github.io/sparrow/deployment/separate-ui/).

## Local Development

Running from source needs Go 1.26+, Node (for the UI), and a PostgreSQL instance.

**Fastest**: start a throwaway Postgres, then run the server with `make`:

```bash
make dev-db    # Postgres 15 on localhost:5432 (user/password/db: sparrow)
make run       # go run ./cmd/server with SPARROW_SERVE_UI=true; migrations run on startup
```

`make run` uses `DATABASE_URL` and the encryption keyring from your shell or `.env`. When neither sets them, it falls back to `postgres://sparrow:sparrow@localhost:5432/sparrow?sslmode=disable` (the `make dev-db` database) and a dev-only all-zeros keyring. The server comes up on <http://localhost:8080>. The UI is only there after `make build-ui`; otherwise use `make run-web` below.

**Full container stack** (server + Postgres, rebuilt from source on every change):

```bash
make docker-dev     # docker compose -f docker-compose.dev.yml up -d --build
make docker-purge   # tear it down, including volumes
```

**UI with hot reload** — run the Go server as above, then in a second terminal:

```bash
make run-web    # cd web && npm run dev, served at localhost:5173
```

Other useful targets: `make build` / `make build-with-ui` (binary, optionally with embedded UI), `make test` (all modules), `make lint`, `make fmt`. Full list: `make help`.

### Quick start with the CLI

Prefer a terminal over curl? The `sparrow` CLI covers the same loop in 90 seconds.

Install it — grab a prebuilt binary (no Go toolchain needed) or use Go:

```bash
# No Go? Download the `sparrow-cli` archive for your OS/arch from the latest
# release, extract it, and move the `sparrow` binary onto your PATH:
#   https://github.com/sarathsp06/sparrow/releases/latest

# Have Go? Install straight from source:
go install github.com/sarathsp06/sparrow/satellites/sparrow@latest
```

Then run the loop:

```bash
sparrow init --url http://localhost:8080        # point the CLI at your server
sparrow listen --event order.created            # receive deliveries locally (Ctrl-C cleans up)
sparrow push order.created -d '{"order_id":"ord_1","amount":42}'   # in another terminal
```

And route an event to Slack in one command:

```bash
sparrow use slack --param webhook_url=https://hooks.slack.com/services/T00/B00/xxx --event order.created
```

Full reference: [Sparrow CLI docs](https://sarathsp06.github.io/sparrow/satellites/cli/). The curl equivalent is below.

### Push your first event

```bash
# 1) Register an event type
curl -X POST http://localhost:8080/v1/event-types \
  -H "Content-Type: application/json" \
  -d '{
    "name": "order.created",
    "description": "Fires when a new order is placed",
    "active": true
  }'

# 2) Register a webhook; Sparrow auto-creates the subscription
curl -X POST http://localhost:8080/v1/consumers/default/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://httpbin.org/post",
    "events": ["order.created"],
    "active": true
  }'

# 3) Push an event occurrence; delivery happens asynchronously
curl -X POST "http://localhost:8080/v1/consumers/default/events?event=order.created" \
  -H "Content-Type: application/json" \
  -d '{
    "payload": {
      "order_id": "ord_123",
      "customer": "alice@example.com",
      "total": 49.99
    },
    "idempotency_key": "idem_ord_123",
    "ttl_seconds": 3600
  }'

# 4) Inspect delivery state
curl "http://localhost:8080/v1/consumers/default/deliveries?limit=10"
```

What happens next: Sparrow stores the event, enqueues async fan-out work in River, creates deliveries for matching subscriptions, signs outbound requests, retries retryable failures, and records the full result history.

## Security

Sparrow assumes you run it inside a network you control, then adds application-level protections around secrets and delivery behavior:

- **Envelope encryption** — secrets and sensitive headers are encrypted with per-record data keys wrapped by the configured `SPARROW_ENCRYPTION_KEYS` keyring.
- **Signed deliveries** — every request carries `webhook-id`, `webhook-timestamp`, and `webhook-signature` headers in Standard Webhooks format.
- **SSRF protection** — private, loopback, link-local, and cloud-metadata IPs are blocked by default, and redirects are re-validated.
- **TLS verification** — deliveries verify the receiver's certificate. A webhook can opt out with `http_config.verify_ssl: false` for self-signed internal endpoints; SSRF checks still apply.
- **Optional shared-secret auth** — set `SPARROW_API_KEY` to require a key on API requests. Per-person [access tokens and one-time invites](https://sarathsp06.github.io/sparrow/deployment/access/) let you stop sharing the master key, and each one can be revoked on its own.
- **No secrets in logs or traces** — webhook URLs are reduced to scheme and host (`https://hooks.slack.com/…`) in logs, spans, and delivery errors, because URLs often carry tokens in the path or query.
- **No default telemetry egress** — OpenTelemetry export is off unless you set `OTEL_EXPORTER_OTLP_ENDPOINT`.
- **Proxy-friendly** — terminate TLS, SSO, and rate limiting at the reverse proxy; Sparrow stays a small HTTP service.

> [!WARNING]
> With `SPARROW_API_KEY` unset, anyone who can reach the port can use the API and dashboard — treat it as trusted-network-only. When `SPARROW_API_KEY` is set, the embedded dashboard shows a sign-in prompt: paste the master key (exchanged for a named browser token, never stored) or use an access token or invite link. On shared or internet-facing networks, set an API key and put Sparrow behind an authenticating proxy.

Full details — trust model, SSO via an identity-aware proxy (Authentik, oauth2-proxy, Keycloak), and a hardening checklist: [Securing Sparrow](https://sarathsp06.github.io/sparrow/deployment/security/).

### Verifying webhook signatures

Each delivery includes:

- `webhook-id` — stable message identifier.
- `webhook-timestamp` — Unix timestamp, used for replay protection.
- `webhook-signature` — one or more space-delimited signatures.

Sparrow signs the exact message:

```text
{webhook-id}.{webhook-timestamp}.{raw request body}
```

Signature formats:

- `v1,` — HMAC-SHA256 using the webhook secret (`whsec_<base64>` format)
- `v1a,` — Ed25519 using the webhook's `signing_public_key` when `signature_type=ed25519`

The Go module is the authoritative verifier (tested against the server's signer); the other
languages are copy-in samples that follow the same rules and should work. Review a sample
before relying on it. Framework examples for each are in the
[verification guide](https://sarathsp06.github.io/sparrow/guides/verify-signatures/).

- **Go**: [`pkg/signature`](pkg/signature/signature.go) (`go get`)
- **Python**: [`sparrow_verify.py`](client/verify/python/sparrow_verify.py)
- **TypeScript / JavaScript**: [`sparrow-verify.ts`](client/verify/js/sparrow-verify.ts)
- **Java (15+)**: [`SparrowVerify.java`](client/verify/java/SparrowVerify.java)
- **Kotlin**: [`SparrowVerify.kt`](client/verify/kotlin/SparrowVerify.kt)
- **Ruby**: [`sparrow_verify.rb`](client/verify/ruby/sparrow_verify.rb)
- **PHP (8.1+)**: [`SparrowVerify.php`](client/verify/php/SparrowVerify.php)
- **Rust**: [`sparrow_verify.rs`](client/verify/rust/sparrow_verify.rs)
- **Elixir**: [`sparrow_verify.ex`](client/verify/elixir/sparrow_verify.ex)

## Architecture

The API surface is a single HTTP server on `:8080`:

- **Router** — [chi](https://github.com/go-chi/chi) for routing and middleware
- **API layer** — [Huma](https://github.com/danielgtaylor/huma) generates the OpenAPI 3.1 contract from the Go handler structs
- **Service layer** — `internal/webhooks` owns event fan-out, subscriptions, retries, rate limiting, and health logic
- **Storage layer** — PostgreSQL via `sqlx`; transactions use the `WithConn` repository pattern
- **Queue layer** — River workers process event fan-out, webhook delivery, and batch re-push / retry jobs
- **UI** — Svelte 5 static app embedded into the Go binary with `go:embed`

Read the docs walk-through: [Architecture Overview](https://sarathsp06.github.io/sparrow/reference/architecture/) or open the [interactive layered diagram](https://sarathsp06.github.io/sparrow/diagrams/sparrow-layered-architecture.html).

<p align="center">
  <a href="https://sarathsp06.github.io/sparrow/diagrams/sparrow-layered-architecture.html">
    <picture>
      <source media="(prefers-color-scheme: dark)" srcset="./docs/src/assets/diagrams/system-overview-dark.svg">
      <source media="(prefers-color-scheme: light)" srcset="./docs/src/assets/diagrams/system-overview-light.svg">
      <img src="./docs/src/assets/diagrams/system-overview-light.svg" alt="Sparrow system overview diagram" />
    </picture>
  </a>
</p>

Open the full interactive diagram for pan/zoom, search, focus, and export.

## Deployment

### Docker image

Docker images are published to GitHub Container Registry: [github.com/sarathsp06/sparrow/pkgs/container/sparrow](https://github.com/sarathsp06/sparrow/pkgs/container/sparrow).

Latest Docker release: [`ghcr.io/sarathsp06/sparrow:latest`](https://github.com/sarathsp06/sparrow/pkgs/container/sparrow?tag=latest).

```bash
docker pull ghcr.io/sarathsp06/sparrow:latest
```

Use the Compose file in [`deploy/docker-compose.yml`](deploy/docker-compose.yml) for local evaluation. For a real deployment (Kubernetes or any container platform), see [Production Deployment](https://sarathsp06.github.io/sparrow/deployment/production/).

## Configuration

Everything is configured through environment variables.

**Required**

1. `DATABASE_URL` — PostgreSQL connection string.
   The built-in fallback, `postgres://localhost/riverqueue?sslmode=disable`, is for local development only.
2. `SPARROW_ENCRYPTION_KEYS` — the encryption keyring, as comma-separated `<key-id>=<64-char-hex-key>` entries.
   Each key is a cryptographically random 32-byte value, hex-encoded (`openssl rand -hex 32`). Key IDs may contain only `A-Z`, `a-z`, `0-9`, `_`, and `-`.
3. `SPARROW_ENCRYPTION_PRIMARY_KEY_ID` — which key ID in the keyring encrypts new data.

**Authentication and environment**

4. `SPARROW_API_KEY` — the master API key. Optional, but required when `ENVIRONMENT=production`, where it must be at least 32 characters (`openssl rand -hex 32`).
   When set, API requests need it, or an [access token](https://sarathsp06.github.io/sparrow/deployment/access/), in `X-API-Key` or `Authorization: Bearer`.
5. `SPARROW_TOKEN_DEFAULT_TTL` — optional. Default: `2160h` (90 days).
   Lifetime of tenant-wide access tokens created without an explicit TTL, including browser sign-ins and invites. `--ttl never` / `never_expires` still creates a token that never expires; `0` restores never-expiring defaults.
6. `ENVIRONMENT` — optional. Set to `production` to require `SPARROW_API_KEY` at startup and to block cross-origin requests unless `CORS_ALLOWED_ORIGINS` lists them.
7. `CORS_ALLOWED_ORIGINS` — optional. Comma-separated browser origins allowed to call the API.
   Required when the UI is [hosted separately](https://sarathsp06.github.io/sparrow/deployment/separate-ui/). When unset, every origin is allowed in development and none in production.

**Server and UI**

8. `SPARROW_HTTP_PORT` — optional. Default: `8080`. HTTP listen port.
9. `SPARROW_SERVE_UI` — optional. Default: `false`. Serves the embedded dashboard on the same port.

**Network and limits**

10. `SPARROW_ALLOWED_NETWORKS` — optional. Comma-separated CIDRs or bare IPs (e.g. `10.20.0.0/16,fd12::/48`).
    Webhook deliveries may reach these networks in addition to public addresses; loopback, cloud metadata, and other private space stay blocked. Recommended for internal services on a VPN. Invalid entries fail startup.
11. `SPARROW_ALLOW_PRIVATE_NETWORKS` — optional. Default: `false`.
    Allows every loopback and private-network webhook target (for local development and tests). Cloud metadata endpoints stay blocked; prefer `SPARROW_ALLOWED_NETWORKS` in production.
12. `SPARROW_MAX_BODY_BYTES` — optional. Default: `5242880` (5 MiB).
    Maximum request body size, at least 1 MiB; larger bodies get `413`.
13. `SPARROW_MAX_CAPTURED_RESPONSE_BYTES` — optional. Default: `1048576` (1 MiB).
    Stored response body cap for webhooks with `capture_response_body` (others store 1 KiB).
14. `SPARROW_EVENT_RETENTION_DAYS` — optional. Default: `0` (keep forever).
    Hourly purge of events, and their deliveries, older than this many days.
15. `SPARROW_AUTO_REGISTER_EVENTS` — optional. Default: `false`.
    Pushing an unregistered event name returns `404`; set `true` (as `make run` does) to create a schema-less event type on first push instead. Development only: event types are never deleted. Then use **Infer schema** in the UI to generate the type's schema from the pushed events, review it, and export it to other environments.

**Observability**

16. `OTEL_EXPORTER_OTLP_ENDPOINT` — optional. OTLP collector URL for traces, metrics, and logs (`https://` for TLS). Export is off when unset.
17. `OTEL_EXPORTER_OTLP_PROTOCOL` — optional. `http/protobuf` (default) or `grpc`.
18. `SPARROW_OTLP_SIGNALS` — optional. Default: `traces,metrics,logs`.
    Which signals go to the OTLP endpoint. Set `traces` for a traces-only backend such as Jaeger or Tempo, which otherwise reject the metric and log exporters with `unknown service`. `/metrics` is unaffected.
19. `SPARROW_METRICS_ENABLED` — optional. Default: `true`.
    Serves every OTel metric in Prometheus format at `GET /metrics` (no API key, like `/health`): delivery attempts and latency, queue backlog, webhooks by health and status.

**Failing receivers**

19. `SPARROW_AUTO_DISABLE_AFTER` — optional. Default: `120h` (5 days); `0` turns it off.
    Pauses a webhook whose receiver has failed every attempt for this long, records why on the webhook, and emits `sparrow.webhook.disabled`. Like a manual pause, deliveries are held as `paused` (not dropped); resume it, then retry what was held.
20. `SPARROW_AUTO_DISABLE_MIN_FAILURES` — optional. Default: `10`.
    Minimum run of consecutive failed attempts before a webhook can be auto-disabled.

**Template history**: every save that changes a subscription's `transform_template` records a version (`GET /v1/consumers/{c}/subscriptions/{id}/templateVersions`, last 20, with `manual`/`ai_draft` source, notes, and who saved it). The template editor lists them and can load one back; saving records it again.

**AI-assisted template drafting** (optional)

Without any of these set, the subscription editor still offers **Copy prompt for AI**: `POST /v1/subscriptions:draftTemplatePrompt` builds the same grounded prompt (data model, helper catalog, the event's sample payload, recipe or receiver details, your instructions) for pasting into any chat assistant; paste the returned template back and check it with Run Preview. Configure a provider to draft and verify in place instead.

16. `SPARROW_AI_PROVIDER` — optional. Default: `anthropic`. Set `openai` to use any OpenAI-compatible `/v1/chat/completions` server: a local Ollama, vLLM, LM Studio or llama.cpp, a router like OpenRouter, or OpenAI itself. Drafts stay small and every draft is render-verified and repaired, so a light local model is enough for most templates.
17. `SPARROW_AI_API_KEY` — optional. The provider API key. With the `anthropic` provider, setting it is what turns the feature on; with `openai` it is sent as a Bearer token when present (local servers usually need none). When drafting is on, the subscription editor gains a "Draft with AI" panel (and `POST /v1/subscriptions:draftTemplate`) that writes a `transform_template` from a plain-language description, grounded in the event type's schema and sample payload and verified by rendering before it is shown. Grounding can also use a sample payload you paste, a description or example of what the receiver expects, or a documentation URL that Sparrow fetches under the same network policy as deliveries. Only the schema, that sample payload, and the request's own text are sent to the model, never stored events, headers, or secrets. Off when unset.
18. `SPARROW_AI_MODEL` — optional for `anthropic` (default `claude-haiku-4-5`; raise it if drafts need many repair rounds), required for `openai` (e.g. `llama3.2`, `qwen2.5-coder`, `gpt-4o-mini`).
19. `SPARROW_AI_BASE_URL` — required for `openai` (e.g. `http://localhost:11434/v1` for Ollama); optional for `anthropic` (an internal gateway or proxy). Setting it with `openai` is what turns the feature on.

For a single-key deployment, still use the keyring format: for example `SPARROW_ENCRYPTION_KEYS=main=<64-char-hex-key>` with `SPARROW_ENCRYPTION_PRIMARY_KEY_ID=main`.

Full reference: [`okf/config/env-vars.md`](okf/config/env-vars.md).

Under Docker Compose, set these in a `.env` file next to `docker-compose.yml` (Compose loads it automatically for every command, including `down`/`ps`) rather than exporting them inline — an inline `VAR=value docker compose up` only lasts for that one command.

## Satellites

Satellites are companion tools that orbit the core — they use Sparrow, never reach inside it. Everything below is built entirely on the public REST API, lives in [`satellites/`](satellites/), and is optional: delete them all and the server still works. Full guide: [Satellites docs](https://sarathsp06.github.io/sparrow/satellites/).

### Recipes

Adapters as config: a recipe is a YAML file pairing a destination URL with a transform template that Sparrow renders server-side per delivery. All shipped recipes are **embedded in the `sparrow` CLI binary** — `sparrow recipes` lists them, `sparrow use <name> --param ... --event ...` applies one from anywhere (use `--file` or `SPARROW_RECIPES_DIR` for your own) — and you get retries, signing, and delivery tracking for free, no glue service to run.

1. [`slack`](satellites/recipes/slack.yaml) — Slack incoming webhook (Block Kit message).
   Params: `webhook_url`.
2. [`discord`](satellites/recipes/discord.yaml) — Discord channel webhook (embed).
   Params: `webhook_url`.
3. [`ntfy`](satellites/recipes/ntfy.yaml) — ntfy topic (push notification).
   Params: `topic_url`.
4. [`pagerduty`](satellites/recipes/pagerduty.yaml) — PagerDuty Events API v2 (deduplicated alerts).
   Params: `routing_key`.
5. [`clickhouse`](satellites/recipes/clickhouse.yaml) — ClickHouse HTTP insert (one row per delivery).
   Params: `base_url`, `table`, `user`, `password`.
6. [`twilio`](satellites/recipes/twilio.yaml) — Twilio SMS (Messages API).
   Params: `account_sid`, `basic_auth`, `from_number`, `to_number`.
7. [`sendgrid`](satellites/recipes/sendgrid.yaml) — SendGrid Mail Send API (transactional email).
   Params: `api_key`, `from_email`, `from_name`.
8. [`cloudevents`](satellites/recipes/cloudevents.yaml) — CloudEvents 1.0 structured mode (Knative, Argo Events, Dapr, …).
   Params: `target_url`, `source`.

Recipe credentials that map to HTTP headers (Twilio `basic_auth`, ClickHouse `password`, SendGrid `api_key`) are stored as **envelope-encrypted secret headers** and masked in every API response. PagerDuty's `routing_key` is part of the request body the API requires, so it lives in the transform template.

### Sources

[`satellites/sparrow-sources`](satellites/sparrow-sources/) pushes events **into** Sparrow from the outside world: a cron emitter for scheduled events, plus Stripe and GitHub webhook receivers that verify provider signatures and re-publish them as Sparrow events (`stripe.payment_intent.succeeded`, `github.pull_request.opened`). Once inside, everything applies — fan-out, retries, label filtering, transforms.

```bash
go install github.com/sarathsp06/sparrow/satellites/sparrow-sources@latest
sparrow-sources --config sources.yaml
```

### Sinks

[`satellites/sparrow-sinks`](satellites/sparrow-sinks/) forwards signed Sparrow deliveries **out** of HTTP land: an SMTP email sink, an S3 (or MinIO/R2) archiver that writes one JSON object per delivery, and an OTLP exporter that turns events into OpenTelemetry log records. It verifies Standard Webhooks signatures, holds no state, and leans on Sparrow's retries for anything downstream that fails.

```bash
go install github.com/sarathsp06/sparrow/satellites/sparrow-sinks@latest
sparrow-sinks --config sinks.yaml
```

Docs: [CLI](https://sarathsp06.github.io/sparrow/satellites/cli/) · [Recipes](https://sarathsp06.github.io/sparrow/satellites/recipes/) · [Sources](https://sarathsp06.github.io/sparrow/satellites/sources/) · [Sinks](https://sarathsp06.github.io/sparrow/satellites/sinks/)

## Docs

- **Product docs** — <https://sarathsp06.github.io/sparrow/>
- **Runtime API docs** — serve Sparrow, then open `/docs`
- **OpenAPI spec** — [`api/openapi.yaml`](api/openapi.yaml) and [`api/openapi.json`](api/openapi.json)
- **Knowledge bundle** — [`okf/`](okf/) for architecture, concepts, database, frontend, and ops details

## Contributing

```bash
git clone https://github.com/sarathsp06/sparrow.git
cd sparrow
make build-with-ui
make test
make lint
```

Useful commands:

- `make run` — run the server locally
- `make run-web` — run the UI dev server
- `make generate` — regenerate OpenAPI-derived artifacts
- `make test-integration` — integration tests with testcontainers
- `make test-e2e` — Gauge + Python end-to-end tests
- `make fmt` — format Go code with `goimports`

More detail: [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
