[![CI](https://github.com/sarathsp06/sparrow/actions/workflows/ci.yml/badge.svg)](https://github.com/sarathsp06/sparrow/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/ghcr.io-sarathsp06%2Fsparrow-blue?logo=docker)](https://github.com/sarathsp06/sparrow/pkgs/container/sparrow)
[![Docs](https://img.shields.io/badge/docs-GitHub%20Pages-blue)](https://sarathsp06.github.io/sparrow)

<p align="center">
  <img src="./web/src/lib/assets/favicon.svg" alt="Sparrow" width="120" height="120" />
</p>

# Sparrow

Self-hosted webhook delivery platform with async fan-out, retries, health tracking, and observability. Built for teams that need reliable outbound webhooks without depending on a third-party service.

## Features

- **Event-driven fan-out** -- push one event, deliver to all matching subscriptions
- **Reliable delivery** -- at-least-once semantics with configurable retries and exponential backoff
- **Payload transformation** -- Go templates per subscription to reshape payloads before delivery
- **Health tracking** -- per-webhook success rates, error classification, and automatic degradation detection
- **HMAC signing** -- every delivery is signed so receivers can verify authenticity
- **Dual-protocol API** -- gRPC on `:50051` and Connect-RPC (HTTP/JSON) on `:8080`
- **Web dashboard** -- embedded UI for managing webhooks, events, deliveries, and health
- **Observability** -- OpenTelemetry traces, metrics, and structured logs via OTLP

## Quick Start

Download [`deploy/docker-compose.yml`](deploy/docker-compose.yml) and start it:

```bash
curl -O https://raw.githubusercontent.com/sarathsp06/sparrow/main/deploy/docker-compose.yml
docker compose up -d
```

Open **http://localhost:8080** for the web UI.

### Send your first event

```bash
# Register an event type
curl -X POST http://localhost:8080/webhook.EventService/RegisterEvent \
  -H "Content-Type: application/json" \
  -d '{"name": "order.created", "description": "New order", "active": true}'

# Register a webhook (subscription is created automatically)
curl -X POST http://localhost:8080/webhook.WebhookService/RegisterWebhook \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default", "url": "https://httpbin.org/post", "events": ["order.created"], "active": true}'

# Push an event -- Sparrow fans out and delivers
curl -X POST http://localhost:8080/webhook.EventService/PushEvent \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default", "event": "order.created", "payload": {"order_id": "ord_123", "amount": 99.99}}'
```

Check delivery status in the web UI at **Deliveries**, or query the API:

```bash
curl -X POST http://localhost:8080/webhook.DeliveryService/ListDeliveries \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default", "limit": 5}'
```

## Use Cases

- **SaaS webhook notifications** -- notify customer endpoints when resources change
- **Internal event bus** -- fan out domain events to downstream services over HTTP
- **Reliability layer** -- add retries, health tracking, and observability to existing webhook flows
- **Development and testing** -- inspect deliveries, replay failed events, test payload transforms

## Architecture

```
PushEvent API
  -> persist event in PostgreSQL
  -> enqueue fan-out job (River)
     -> match subscriptions, apply transforms, create deliveries
     -> enqueue delivery jobs
        -> HTTP POST with HMAC signature
        -> retry on failure (server errors, timeouts, network errors)
        -> track health per webhook
```

Events are persisted before delivery. The [River](https://riverqueue.com) job queue provides at-least-once delivery with configurable retries (default: 3 attempts, 60s backoff). Failures are classified into retryable (5xx, timeout, connection refused, network error) and non-retryable (4xx, DNS, TLS) categories.

See [TECHNICAL.md](TECHNICAL.md) for the full pipeline design, queue configuration, error classification, and health state machine.

## Configuration

All configuration is via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_URL` | Yes | -- | PostgreSQL connection string |
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded web dashboard |
| `SPARROW_API_KEY` | No | -- | Require this key in `X-API-Key` header |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP endpoint for traces/metrics/logs |

See [CONFIGURATION.md](CONFIGURATION.md) for the full list.

## Deployment

### Docker

Pre-built multi-arch images (linux/amd64, linux/arm64) are published on every release:

```bash
docker pull ghcr.io/sarathsp06/sparrow:latest
```

See [Docker Compose deployment guide](https://sarathsp06.github.io/sparrow/deployment/docker-compose/) for details.

### Kubernetes

A Helm chart is included at [`charts/sparrow/`](charts/sparrow/):

```bash
helm install sparrow charts/sparrow/ \
  --set secrets.databaseURL="postgres://user:pass@your-db:5432/sparrow?sslmode=require"
```

See [Kubernetes deployment guide](https://sarathsp06.github.io/sparrow/deployment/kubernetes/) for all chart values and examples.

## Documentation

**[sarathsp06.github.io/sparrow](https://sarathsp06.github.io/sparrow)**

- [Getting Started](https://sarathsp06.github.io/sparrow/getting-started/installation/) -- installation and quickstart
- [API Reference](https://sarathsp06.github.io/sparrow/reference/api/) -- all RPCs and message types
- [Architecture](https://sarathsp06.github.io/sparrow/reference/architecture/) -- system design overview

In-repo docs: [CONFIGURATION.md](CONFIGURATION.md) | [KUBERNETES.md](KUBERNETES.md) | [TECHNICAL.md](TECHNICAL.md) | [ARCHITECTURE.md](ARCHITECTURE.md) | [webhook.proto](webhook.proto)

## Contributing

Contributions are welcome. Please open an issue to discuss larger changes before submitting a PR.

```bash
git clone https://github.com/sarathsp06/sparrow.git
cd sparrow
make build-with-ui   # build server + embedded UI
make test            # run tests
make lint            # run linters
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for the package structure and dependency graph.

## License

[MIT](LICENSE)
