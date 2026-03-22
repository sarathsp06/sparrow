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

## Configuration

Sparrow is configured entirely via environment variables. No config files needed.

### Core

- `DATABASE_URL` (required) -- PostgreSQL connection string
- `SPARROW_SERVE_UI` -- Serve the embedded web dashboard on `:8080`. Default: `false`
- `ENVIRONMENT` -- `development` or `production` (affects logging/OTel)

### Authentication

- `SPARROW_AUTH_ENABLED` -- Enable authentication. When disabled, all requests use `tenant:admin` on the default tenant. Default: `false`
- `SPARROW_ROOT_API_KEY` -- Pre-configured root API key for bootstrap. Default: auto-generated on first boot

When auth is enabled, Sparrow supports both API keys and JWT. See [CONFIGURATION.md](CONFIGURATION.md) for the full list of JWT, OIDC, and identity provider settings.

### Deployment Modes

- **Zero-auth** -- Set only `DATABASE_URL`. All requests get `tenant:admin`. Best for single-user/dev.
- **API-key only** -- Add `SPARROW_AUTH_ENABLED=true`. Root key created on boot. Create more via API.
- **Any OIDC provider** -- Add `SPARROW_JWKS_URL=...`. JWT auth with any provider. Namespace roles resolved from DB.
- **Full Clerk** -- Add `CLERK_SECRET_KEY=...`. Syncs namespace roles to JWT claims for faster auth.

See [CONFIGURATION.md](CONFIGURATION.md) for detailed setup guides for each mode, including Clerk, Keycloak, Auth0, and other OIDC providers.

---

## Documentation

- [CONFIGURATION.md](CONFIGURATION.md) -- All environment variables, auth setup, OIDC provider examples, deployment guides
- [TECHNICAL.md](TECHNICAL.md) -- Architecture deep dive, RBAC internals, queue design, health state machine, HTTP client details
- [ARCHITECTURE.md](ARCHITECTURE.md) -- Package structure, dependency graph, design principles
- [TEMPLATE_FUNCTIONS.md](TEMPLATE_FUNCTIONS.md) -- Payload transformation reference (22 template functions)
- [client/README.md](client/README.md) -- gRPC client libraries (Go, JS, Python)
- [web/README.md](web/README.md) -- Web dashboard development guide
- [webhook.proto](webhook.proto) -- Service and message definitions (9 services, 40+ RPCs)

---

## Built With

This project was built with the help of [OpenCode](https://opencode.ai) -- an AI-powered coding agent for the terminal.

---

## License

Business Source License 1.1 -- see [LICENSE](LICENSE) for details.
