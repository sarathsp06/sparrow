![Work in Progress](https://img.shields.io/badge/Status-Work%20in%20Progress-orange?style=for-the-badge)
<div align="center">
	<img src="./web/src/lib/assets/favicon.svg" alt="Sparrow" width="180" height="180" />
	<h1 style="font-family:monospace;font-weight:900;color:#222;">sparrow</h1>
	<p style="font-size:1.1em;color:#555;">Reliable webhook delivery with retries, health tracking, and full observability</p>
</div>

---

## Quick Start

### Prerequisites

- Go 1.25+
- Docker & Docker Compose
- Node.js + Yarn (for the web UI)

### 1. Start Infrastructure

```bash
make docker-dev   # PostgreSQL, River UI, OpenTelemetry Collector
make migrate      # Run database migrations
```

### 2. Start the Server

```bash
make run          # gRPC (:50051) + HTTP/JSON (:8080)
make run-web      # Web dashboard (:5173) — optional
```

### 3. End-to-End Walkthrough

Register an event, a webhook, subscribe, and push — all from curl.

**Register an event type:**

```bash
curl -s -X POST http://localhost:8080/webhook.EventService/RegisterEvent \
  -H "Content-Type: application/json" \
  -d '{
    "name": "user.created",
    "description": "New user signup",
    "schema": "{\"type\":\"object\",\"properties\":{\"user_id\":{\"type\":\"string\"},\"email\":{\"type\":\"string\"}}}",
    "active": true
  }'
```

**Register a webhook endpoint:**

```bash
curl -s -X POST http://localhost:8080/webhook.WebhookService/RegisterWebhook \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "default",
    "url": "https://httpbin.org/post",
    "active": true,
    "description": "Test endpoint"
  }'
# Note the webhook_id from the response — you need it next
```

**Subscribe the webhook to the event:**

```bash
curl -s -X POST http://localhost:8080/webhook.SubscriptionService/CreateSubscription \
  -H "Content-Type: application/json" \
  -d '{
    "webhook_id": "<webhook_id from above>",
    "event_name": "user.created",
    "namespace": "default",
    "method": "POST"
  }'
```

**Push an event:**

```bash
curl -s -X POST http://localhost:8080/webhook.EventService/PushEvent \
  -H "Content-Type: application/json" \
  -d '{
    "namespace": "default",
    "event": "user.created",
    "payload": {
      "user_id": "u_123",
      "email": "jane@example.com"
    }
  }'
```

Sparrow picks up the event, matches subscriptions, delivers the webhook with retries, and tracks health automatically.

**Verify delivery:**

```bash
# Check deliveries via the API
curl -s -X POST http://localhost:8080/webhook.DeliveryService/ListDeliveries \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default"}'
```

Or open the web dashboard at `http://localhost:5173` or the River queue UI at `http://localhost:8082`.

---

## How It Works

```
Push Event → Match Subscriptions → Queue Deliveries → Deliver with Retries
                                                        ↓
                                              Track Health + Log Everything
```

- Events are validated against their JSON Schema
- Subscriptions can transform payloads via Go templates
- Webhooks are signed with HMAC-SHA256
- Delivery attempts are logged with full request/response for debugging
- Endpoint health (healthy/degraded/unhealthy) is tracked automatically

## API

All endpoints are available via both gRPC (`:50051`) and Connect-RPC HTTP/JSON (`:8080`).

| Service | Key Methods |
|---|---|
| `EventService` | RegisterEvent, PushEvent, ListEvents |
| `WebhookService` | RegisterWebhook, ListWebhooks, PauseWebhook, ResumeWebhook |
| `SubscriptionService` | CreateSubscription, ListSubscriptions, TestSubscriptionTemplate |
| `DeliveryService` | ListDeliveries, GetDeliveryStatus, RetryDelivery |
| `HealthService` | GetWebhookHealth, GetHealthSummary |

HTTP endpoint pattern: `POST http://localhost:8080/webhook.<Service>/<Method>`

## Configuration

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://localhost/riverqueue?sslmode=disable` | PostgreSQL connection string |
| `HTTP_PORT` | `8080` | Connect-RPC HTTP server port |
| `GRPC_PORT` | `50051` | gRPC server port |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OpenTelemetry collector |

## Useful Commands

| Command | Description |
|---|---|
| `make docker-dev` | Start Postgres, River UI, OTel Collector |
| `make migrate` | Run database migrations |
| `make run` | Start the server |
| `make run-web` | Start the web dashboard |
| `make example` | Run the example gRPC client |
| `make test` | Run tests |
| `make lint` | Run linter |
| `make build` | Build server binary |
| `make docker-purge` | Tear down dev infrastructure |

## Further Reading

- [TECHNICAL.md](TECHNICAL.md) — Architecture, database schema, queue config
- [docs/TEMPLATE_FUNCTIONS.md](docs/TEMPLATE_FUNCTIONS.md) — Payload transformation reference
- [examples/grpc_client.go](examples/grpc_client.go) — Full gRPC client example
- [proto/webhook.proto](proto/webhook.proto) — Service and message definitions

## License

MIT
