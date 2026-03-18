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

On first boot a root API key is printed to the logs:

```bash
docker compose logs sparrow
```

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
Push Event -> Match Subscriptions -> Queue Deliveries -> Deliver with Retries
                                                          |
                                                Track Health + Log Everything
```

All endpoints are available via both gRPC (`:50051`) and Connect-RPC HTTP/JSON (`:8080`).

---

## Documentation

| Document | Description |
|---|---|
| [TECHNICAL.md](TECHNICAL.md) | Architecture, auth internals, RBAC, queue design, health state machine, deployment guide |
| [TEMPLATE_FUNCTIONS.md](TEMPLATE_FUNCTIONS.md) | Payload transformation reference (22 template functions) |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Package structure review, dependency graph, design philosophy |
| [web/README.md](web/README.md) | Web dashboard development guide |
| [webhook.proto](webhook.proto) | Service and message definitions (9 services, 40+ RPCs) |

---

## License

Business Source License 1.1 -- see [LICENSE](LICENSE) for details.
