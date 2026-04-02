[![CI](https://github.com/sarathsp06/sparrow/actions/workflows/ci.yml/badge.svg)](https://github.com/sarathsp06/sparrow/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-15-4169E1?logo=postgresql&logoColor=white)](https://www.postgresql.org)
[![gRPC](https://img.shields.io/badge/gRPC-Connect--RPC-244c5a?logo=grpc)](https://connectrpc.com)
[![Docs](https://img.shields.io/badge/docs-GitHub%20Pages-blue)](https://sarathsp06.github.io/sparrow)
<div align="center">
	<img src="./web/src/lib/assets/favicon.svg" alt="Sparrow" width="180" height="180" />
	<h1 style="font-family:monospace;font-weight:900;color:#222;">sparrow</h1>
	<p style="font-size:1.1em;color:#555;">Reliable webhook delivery with retries, health tracking, and full observability</p>
</div>

---

## Quick Start

```bash
docker compose up -d
```

Postgres, migrations, and the server with the web UI all start automatically. Open `http://localhost:8080/`.

### Try It Out

```bash
# 1. Register an event type
curl -s -X POST http://localhost:8080/webhook.EventService/RegisterEvent \
  -H "Content-Type: application/json" \
  -d '{"name": "order.created", "description": "New order placed", "active": true}'

# 2. Register a webhook (subscriptions are created automatically)
curl -s -X POST http://localhost:8080/webhook.WebhookService/RegisterWebhook \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default", "url": "https://testhooks.sarathsadasivan.com/hooks", "events": ["order.created"], "active": true}'

# 3. Push an event
curl -s -X POST http://localhost:8080/webhook.EventService/PushEvent \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default", "event": "order.created", "payload": {"order_id": "ord_123", "amount": 99.99}}'
```

> **Need a test endpoint?** Use `https://testhooks.sarathsadasivan.com/hooks` -- it accepts any webhook payload and lets you inspect the requests.

Sparrow matches subscriptions, delivers the webhook with retries, and tracks health automatically.

### Or Use the Web UI

All of the above -- registering events, creating webhooks with subscriptions, pushing events, inspecting deliveries, and monitoring health -- can be done through the web dashboard at [localhost:8080](http://localhost:8080).

1. **Events** → Register event types, push test events with live schema validation
2. **Webhooks** → Register webhooks, manage subscriptions with payload transformation templates, pause/resume, edit configuration
3. **Deliveries** → Inspect delivery status, view request/response bodies, retry failed deliveries
4. **Health** → Monitor webhook health, view error category breakdowns, track success rates

---

## How It Works

```
PushEvent
  -> Validate payload against event schema (if defined)
  -> Persist event record
  -> Enqueue fan-out job
     -> Match subscriptions by (namespace, event_name, label_filters)
     -> Apply Go template transformation per subscription (if enabled)
     -> Create one delivery record per matching subscription
     -> Enqueue delivery jobs
        -> HTTP POST to webhook URL with HMAC signature
        -> Record attempt (response code, response time, error category)
        -> On success: mark delivered, update health metrics
        -> On failure: classify error, retry with exponential backoff (if retryable)
        -> Store response body (up to 1 KB by default, 1 MB if capture_response_body is on)
```

**Delivery guarantees**: Events are persisted in PostgreSQL before any delivery is attempted. The River job queue provides at-least-once delivery semantics with configurable retries (default: 3 attempts, 60s backoff).

**Error classification**: Failures are categorized as `client_error` (4xx, not retried), `server_error` (5xx, retried), `timeout` (retried), `connection_refused` (retried), `network_error` (retried), `dns_error` (not retried), or `tls_error` (not retried).

**Health tracking**: Per-webhook health is computed from delivery outcomes -- healthy (>90% success rate, <3 consecutive failures), degraded (50-90% or 3-9 consecutive), unhealthy (<50% or 10+ consecutive).

All endpoints are available via both gRPC (`:50051`) and Connect-RPC HTTP/JSON (`:8080`).

---

## Configuration

Sparrow is configured entirely via environment variables. No config files needed.

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | -- | PostgreSQL connection string |
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded web dashboard on `:8080` |
| `ENVIRONMENT` | No | -- | `development` or `production` (affects logging/OTel) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP HTTP endpoint for traces, metrics, and logs |
| `CORS_ALLOWED_ORIGINS` | No | -- | Allowed CORS origins for Connect-RPC |

See [CONFIGURATION.md](CONFIGURATION.md) for deployment guides and all options.

---

## Kubernetes

A Helm chart is included at [`charts/sparrow/`](charts/sparrow/). Bring your own PostgreSQL or enable the bundled one for evaluation:

```bash
# External database (recommended for production)
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://user:pass@your-db:5432/sparrow?sslmode=require"

# Bundled PostgreSQL (evaluation only)
helm install sparrow charts/sparrow/ --set postgresql.enabled=true
```

See [KUBERNETES.md](KUBERNETES.md) for the full deployment guide, all chart values, and examples.

---

## Documentation

**[sarathsp06.github.io/sparrow](https://sarathsp06.github.io/sparrow)** -- Full documentation site with guides, API reference, and deployment instructions.

Quick links:

- [Getting Started](https://sarathsp06.github.io/sparrow/getting-started/installation/)
- [API Reference](https://sarathsp06.github.io/sparrow/reference/api/)
- [Architecture](https://sarathsp06.github.io/sparrow/reference/architecture/)
- [Docker Compose](https://sarathsp06.github.io/sparrow/deployment/docker-compose/)
- [Kubernetes](https://sarathsp06.github.io/sparrow/deployment/kubernetes/)

In-repo references:

- [CONFIGURATION.md](CONFIGURATION.md) -- All environment variables
- [KUBERNETES.md](KUBERNETES.md) -- Helm chart deployment guide
- [TECHNICAL.md](TECHNICAL.md) -- Architecture deep dive, queue design, health state machine
- [ARCHITECTURE.md](ARCHITECTURE.md) -- Package structure, dependency graph
- [webhook.proto](webhook.proto) -- Service and message definitions

---

## License

[MIT](LICENSE)
