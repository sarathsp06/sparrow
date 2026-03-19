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

| Variable | Required | Default | Description |
|---|---|---|---|
| `DATABASE_URL` | Yes | -- | PostgreSQL connection string |
| `SPARROW_SERVE_UI` | No | `false` | Serve the embedded web dashboard on `:8080` |
| `ENVIRONMENT` | No | -- | `development` or `production` (affects logging/OTel) |

### Authentication

| Variable | Required | Default | Description |
|---|---|---|---|
| `SPARROW_AUTH_ENABLED` | No | `false` | Enable authentication. When disabled, all requests use `tenant:admin` on the default tenant. |
| `SPARROW_JWKS_URL` | No | -- | JWKS endpoint for JWT validation. Works with any OIDC provider (Clerk, Auth0, Keycloak, Authelia, Zitadel, etc.) |
| `SPARROW_JWT_TENANT_CLAIM` | No | `org_id` | JWT claim containing the tenant/org identifier |
| `SPARROW_JWT_ROLE_CLAIM` | No | `org_role` | JWT claim containing the user's role |
| `SPARROW_JWT_SUBJECT_CLAIM` | No | `sub` | JWT claim containing the user identifier |
| `SPARROW_JWT_ISSUER` | No | -- | Expected JWT issuer (`iss`). Leave empty to skip validation. |
| `SPARROW_JWT_AUDIENCES` | No | -- | Comma-separated expected audience values (`aud`). Leave empty to skip validation. |
| `SPARROW_JWT_NAMESPACE_ROLES_CLAIM` | No | `namespace_roles` | JWT claim for embedded namespace roles. Set to `""` to disable and always resolve from DB. |
| `SPARROW_JWT_ROLE_MAPPING` | No | `org:admin=tenant:admin,org:member=tenant:member` | Comma-separated `provider_role=sparrow_role` pairs for mapping provider roles to Sparrow roles |
| `SPARROW_ROOT_API_KEY` | No | (auto-generated) | Pre-configured root API key for bootstrap |

### Identity Provider (Optional)

| Variable | Required | Default | Description |
|---|---|---|---|
| `CLERK_SECRET_KEY` | No | -- | Clerk secret key. When set, namespace role changes are synced to Clerk org membership metadata so they appear in future JWTs. |

### Observability

| Variable | Required | Default | Description |
|---|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | No | -- | OTLP HTTP endpoint for traces, metrics, and logs |

### Web UI (Frontend)

| Variable | Required | Default | Description |
|---|---|---|---|
| `PUBLIC_AUTH_PROVIDER` | No | (auto-detected) | Auth provider: `clerk` or `none` |
| `PUBLIC_CLERK_PUBLISHABLE_KEY` | No | -- | Clerk publishable key (auto-enables Clerk provider) |
| `PUBLIC_API_URL` | No | -- | API base URL for the frontend |
| `CORS_ALLOWED_ORIGINS` | No | -- | Allowed CORS origins for Connect-RPC |

### Deployment Modes

Sparrow supports four deployment modes depending on your auth needs:

| Mode | Env Vars | Description |
|---|---|---|
| **Zero-auth** | `DATABASE_URL` only | All requests get `tenant:admin`. Best for single-user/dev. |
| **API-key only** | `+ SPARROW_AUTH_ENABLED=true` | Root key created on boot. Create more via API. No OIDC. |
| **Any OIDC provider** | `+ SPARROW_JWKS_URL=...` | JWT auth with any provider. Namespace roles resolved from DB (30s cache). |
| **Full Clerk** | `+ CLERK_SECRET_KEY=...` | Adds namespace role sync to JWT claims. Performance optimization, not required. |

See [TECHNICAL.md](TECHNICAL.md) for detailed deployment and auth configuration guides.

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

## Built With

This project was built with the help of [OpenCode](https://opencode.ai) -- an AI-powered coding agent for the terminal.

---

## License

Business Source License 1.1 -- see [LICENSE](LICENSE) for details.
