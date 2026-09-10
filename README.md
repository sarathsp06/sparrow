<div align="center">

<img src="./web/src/lib/assets/favicon.svg" width="88" alt="Sparrow logo" />

# Sparrow

**Self-hosted webhook delivery with retries, signing, health tracking, and one infrastructure dependency: PostgreSQL.**

[![Release](https://img.shields.io/github/v/release/sarathsp06/sparrow?sort=semver&style=flat-square)](https://github.com/sarathsp06/sparrow/releases)
[![CI](https://img.shields.io/github/actions/workflow/status/sarathsp06/sparrow/ci.yml?branch=main&style=flat-square&label=CI)](https://github.com/sarathsp06/sparrow/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Docker](https://img.shields.io/badge/ghcr.io-sarathsp06%2Fsparrow-2496ED?style=flat-square&logo=docker&logoColor=white)](https://github.com/sarathsp06/sparrow/pkgs/container/sparrow)
[![Docs](https://img.shields.io/badge/docs-Starlight-F97316?style=flat-square)](https://sarathsp06.github.io/sparrow/)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue?style=flat-square)](LICENSE)

[Quick Start](#quick-start) · [Why Sparrow](#why-sparrow) · [Security](#security) · [Architecture](#architecture) · [Docs](#docs) · [Contributing](#contributing)

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
- **Soft schema validation** — invalid payloads produce warnings instead of being dropped.
- **Embedded admin UI** — webhooks, events, deliveries, health, and event-instance inspection in one dashboard.
- **OpenAPI-first API** — REST on `:8080`, interactive docs at `/docs`, committed spec at [`api/openapi.yaml`](api/openapi.yaml).
- **Operationally boring** — one database, one binary, Docker image, Railway deploy, and a hardened Helm chart.
- **OpenTelemetry built in** — traces, metrics, and logs, including propagation through async jobs.

## Why Sparrow

**Use Sparrow when you want:**

- self-hosted webhook infrastructure for internal systems or product backends
- delivery guarantees, retries, replay, and inspection without building it yourself
- real security controls for secrets, signatures, and outbound network access
- a Postgres-only operational model instead of Postgres + Redis + separate workers

**Reach for something else when you need:**

- a fully managed webhook SaaS
- an end-user / consumer portal where your customers manage their own endpoints
- a multi-tenant SaaS control plane instead of a self-hosted operator tool

## Quick Start

The fastest path is Docker Compose. No repo clone needed:

```bash
curl -O https://raw.githubusercontent.com/sarathsp06/sparrow/main/deploy/docker-compose.yml
SPARROW_ENCRYPTION_KEY=$(openssl rand -hex 32) docker compose up -d
```

Open <http://localhost:8080> for the UI. The REST API is on the same address, and interactive API docs are at <http://localhost:8080/docs>.

> [!IMPORTANT]
> `SPARROW_ENCRYPTION_KEY` is the master key for data encrypted at rest. Generate it once, store it in your secret manager, and back it up. Lose it and encrypted webhook secrets are unrecoverable.

If you set `SPARROW_API_KEY`, add `X-API-Key: <your-key>` to every API request.

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
curl -X POST http://localhost:8080/v1/namespaces/default/webhooks \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://httpbin.org/post",
    "events": ["order.created"],
    "active": true
  }'

# 3) Push an event occurrence; delivery happens asynchronously
curl -X POST "http://localhost:8080/v1/namespaces/default/events?event=order.created" \
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
curl "http://localhost:8080/v1/namespaces/default/deliveries?limit=10"
```

What happens next: Sparrow stores the event, enqueues async fan-out work in River, creates deliveries for matching subscriptions, signs outbound requests, retries retryable failures, and records the full result history.

## Security

Sparrow assumes you run it inside a network you control, then adds application-level protections around secrets and delivery behavior:

- **Envelope encryption** — secrets and sensitive headers are encrypted with per-record data keys wrapped by `SPARROW_ENCRYPTION_KEY`.
- **Signed deliveries** — every request carries `webhook-id`, `webhook-timestamp`, and `webhook-signature` headers in Standard Webhooks format.
- **SSRF protection** — private, loopback, link-local, and cloud-metadata IPs are blocked by default, and redirects are re-validated.
- **Optional shared-secret auth** — set `SPARROW_API_KEY` to require `X-API-Key` on API requests.
- **No default telemetry egress** — OpenTelemetry export is off unless you set `OTEL_EXPORTER_OTLP_ENDPOINT`.
- **Hardened Helm defaults** — non-root, read-only root filesystem, dropped Linux capabilities, seccomp, and NetworkPolicy isolation.

> [!WARNING]
> With `SPARROW_API_KEY` unset, anyone who can reach the port can use the API and dashboard. On shared or internet-facing networks, set an API key and put Sparrow behind normal network controls.

### Verifying webhook signatures

Each delivery includes:

| Header | Purpose |
|---|---|
| `webhook-id` | Stable message identifier |
| `webhook-timestamp` | Unix timestamp used for replay protection |
| `webhook-signature` | One or more space-delimited signatures |

Sparrow signs the exact message:

```text
{webhook-id}.{webhook-timestamp}.{raw request body}
```

Signature formats:

- `v1,` — HMAC-SHA256 using the webhook secret (`whsec_<base64>` format)
- `v1a,` — Ed25519 using the webhook's `signing_public_key` when `signature_type=ed25519`

Helpers already live in the repo:

| Language | Helper |
|---|---|
| Go | [`pkg/signature`](pkg/signature/signature.go) |
| Python | [`client/verify/python/sparrow_verify.py`](client/verify/python/sparrow_verify.py) |
| TypeScript / JavaScript | [`client/verify/js/sparrow-verify.ts`](client/verify/js/sparrow-verify.ts) |

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

### Railway

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/KzfSI3?referralCode=otXr-t&utm_medium=integration&utm_source=template&utm_campaign=generic)

### Docker image

```bash
docker pull ghcr.io/sarathsp06/sparrow:latest
```

### Kubernetes

The Helm chart lives at [`charts/sparrow/`](charts/sparrow/) and includes bundled or external PostgreSQL options, secrets wiring, ingress, HPA, PDB, and NetworkPolicy.

```bash
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://user:pass@your-db:5432/sparrow?sslmode=require" \
  --set secrets.existingSecret="sparrow-secrets"
```

## Configuration

Everything is configured through environment variables.

| Variable | Required | Default | Purpose |
|---|---|---|---|
| `DATABASE_URL` | Yes | `postgres://localhost/riverqueue?sslmode=disable` | PostgreSQL connection string |
| `SPARROW_ENCRYPTION_KEY` | Yes | — | 64-char hex master key for envelope encryption |
| `SPARROW_API_KEY` | No | — | Require `X-API-Key` on API requests |
| `SPARROW_HTTP_PORT` | No | `8080` | HTTP listen port |
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded dashboard |
| `SPARROW_ALLOW_PRIVATE_NETWORKS` | No | `false` | Disable the private-network SSRF guard |
| `CORS_ALLOWED_ORIGINS` | No | — | Comma-separated browser allowlist |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | — | OTLP endpoint for traces / metrics / logs |

Full reference: [`okf/config/env-vars.md`](okf/config/env-vars.md).

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
