![Work in Progress](https://img.shields.io/badge/Status-Work%20in%20Progress-orange?style=for-the-badge)
<div align="center">
	<img src="./web/src/lib/assets/favicon.svg" alt="Sparrow" width="180" height="180" />
	<h1 style="font-family:monospace;font-weight:900;color:#222;">sparrow</h1>
	<p style="font-size:1.1em;color:#555;">Reliable webhook delivery with retries, health tracking, and full observability</p>
</div>

---

## Quick Start

### Option A: Docker Compose (Recommended)

```bash
docker compose up -d
```

That's it. Postgres, migrations, and the server with the web UI all start automatically. Open `http://localhost:8080/`.

On first boot a root API key is printed to the logs:

```bash
docker compose logs sparrow
```

### Option B: From Source

Requires Go 1.25+, Node.js, Docker.

```bash
make docker-dev       # Start PostgreSQL + River UI + OTel Collector
make migrate          # Run database migrations
make run              # gRPC (:50051) + HTTP/JSON (:8080)
```

To include the web dashboard:

```bash
make build-with-ui    # Builds frontend + server binary
SPARROW_SERVE_UI=true ./build/server-*
```

Open `http://localhost:8080/`. For frontend development with hot-reload, use `make run-web`.

### End-to-End Walkthrough

Register an event, a webhook, subscribe, and push -- all from curl.

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
# Note the webhook_id from the response -- you need it next
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
curl -s -X POST http://localhost:8080/webhook.DeliveryService/ListDeliveries \
  -H "Content-Type: application/json" \
  -d '{"namespace": "default"}'
```

Or open the web dashboard at `http://localhost:8080`. If running the dev stack (`docker-compose.dev.yml`), the River queue UI is at `http://localhost:8082`.

---

## How It Works

```
Push Event -> Match Subscriptions -> Queue Deliveries -> Deliver with Retries
                                                          |
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

---

## Authentication

Sparrow supports two authentication methods that can be used independently or together:

1. **API keys** -- for programmatic / machine-to-machine access
2. **JWT (OIDC)** -- for browser-based access via any identity provider (Clerk, Auth0, Keycloak, etc.)

Both are optional. When `SPARROW_AUTH_ENABLED=false` (the default), all requests use the default tenant with admin access.

### Enabling Authentication

```bash
export SPARROW_AUTH_ENABLED=true
```

On first boot, Sparrow creates a default tenant and prints a root API key to stdout. Use this key to create additional tenants and keys.

### API Keys

API keys use the format `sk_<tenant_slug>_<random>` and are passed via the `Authorization: Bearer` header.

```bash
# Use the root key to create a new API key
curl -s -X POST http://localhost:8080/webhook.APIKeyService/CreateAPIKey \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{
    "tenant_id": "<tenant_id>",
    "name": "Production Key",
    "role": "tenant:admin"
  }'
# Save the raw_key from the response -- it is only shown once
```

### JWT / OIDC

Sparrow verifies RS256 JWTs from any OIDC-compliant identity provider. Set `SPARROW_JWKS_URL` to enable:

```bash
export SPARROW_AUTH_ENABLED=true
export SPARROW_JWKS_URL=https://your-provider.example.com/.well-known/jwks.json
```

When both JWT and API key auth are enabled, the interceptor tries each in order -- a request can authenticate with either method. See [TECHNICAL.md](TECHNICAL.md) for JWT claim configuration, RBAC role details, and provider-specific setup (Clerk, Auth0, etc.).

---

## Configuration

### Backend

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://localhost/riverqueue?sslmode=disable` | PostgreSQL connection string |
| `SPARROW_AUTH_ENABLED` | `false` | Enable authentication |
| `SPARROW_JWKS_URL` | *(unset)* | JWKS endpoint URL (enables JWT auth) |
| `SPARROW_JWT_TENANT_CLAIM` | `org_id` | JWT claim containing the tenant/org identifier |
| `SPARROW_JWT_ROLE_CLAIM` | `org_role` | JWT claim containing the user's role |
| `SPARROW_JWT_ISSUER` | *(unset)* | Expected JWT issuer |
| `SPARROW_SERVE_UI` | `false` | Serve embedded web UI on the HTTP port |
| `SPARROW_AUTO_BOOTSTRAP` | `true` | Auto-create default tenant and root API key |
| `SPARROW_ROOT_API_KEY` | *(auto-generated)* | Pre-defined root API key (for deterministic deploys) |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OpenTelemetry collector |
| `ENVIRONMENT` | `development` | Environment name for OTel resource |

### Frontend

| Variable | Default | Description |
|---|---|---|
| `PUBLIC_API_URL` | `http://localhost:8080` | Sparrow backend URL |
| `PUBLIC_AUTH_PROVIDER` | *(auto-detect)* | Auth provider: `clerk`, `none`, or unset for auto-detect |
| `PUBLIC_CLERK_PUBLISHABLE_KEY` | *(unset)* | Clerk publishable key (enables Clerk auth) |

---

## Docker Deployment

```bash
docker build -t sparrow .
docker run -p 8080:8080 -p 50051:50051 \
  -e DATABASE_URL=postgres://user:pass@host:5432/sparrow \
  -e SPARROW_AUTH_ENABLED=true \
  -e SPARROW_SERVE_UI=true \
  sparrow
```

The image is based on distroless (nonroot) and includes both the server and migration tool. Run migrations with:

```bash
docker run --entrypoint /app/tools/migrate \
  -e DATABASE_URL=postgres://user:pass@host:5432/sparrow \
  sparrow
```

---

## Useful Commands

| Command | Description |
|---|---|
| `make docker-dev` | Start Postgres, River UI, OTel Collector |
| `make migrate` | Run database migrations |
| `make run` | Start the server |
| `make run-web` | Start the web dashboard dev server |
| `make build` | Build server binary |
| `make build-with-ui` | Build frontend + server binary with embedded UI |
| `make test` | Run tests |
| `make lint` | Run linter |
| `make docker-purge` | Tear down dev infrastructure |

---

## Further Reading

- [TECHNICAL.md](TECHNICAL.md) -- Architecture deep-dive, auth internals, Clerk/OIDC deployment guide, RBAC details, queue design, health state machine
- [web/README.md](web/README.md) -- Web dashboard development guide
- [docs/TEMPLATE_FUNCTIONS.md](docs/TEMPLATE_FUNCTIONS.md) -- Payload transformation reference
- [proto/webhook.proto](proto/webhook.proto) -- Service and message definitions

## License

Business Source License 1.1 -- see [LICENSE](LICENSE) for details. Self-hosting is permitted; offering Sparrow as a commercial service requires a separate license.
