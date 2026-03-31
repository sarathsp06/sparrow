![Work in Progress](https://img.shields.io/badge/Status-Work%20in%20Progress-orange?style=for-the-badge)
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
  -d '{"namespace": "default", "url": "https://httpbin.org/post", "events": ["order.created"], "active": true}'

# 3. Push an event
curl -s -X POST http://localhost:8080/webhook.EventService/PushEvent \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default", "event": "order.created", "payload": {"order_id": "ord_123", "amount": 99.99}}'
```

Sparrow matches subscriptions, delivers the webhook with retries, and tracks health automatically.

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

## Documentation

- [CONFIGURATION.md](CONFIGURATION.md) -- Deployment guides, all environment variables
- [TECHNICAL.md](TECHNICAL.md) -- Architecture deep dive, queue design, health state machine, HTTP client details
- [ARCHITECTURE.md](ARCHITECTURE.md) -- Package structure, dependency graph, design principles
- [docs/TEMPLATE_FUNCTIONS.md](docs/TEMPLATE_FUNCTIONS.md) -- Payload transformation reference (22 template functions)
- [client/README.md](client/README.md) -- gRPC client libraries (Go, JS, Python)
- [web/README.md](web/README.md) -- Web dashboard development and standalone deployment
- [webhook.proto](webhook.proto) -- Service and message definitions

---

## License

[MIT](LICENSE)
