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
- Node.js + npm (for the web UI, optional)

### 1. Start Infrastructure

```bash
make docker-dev   # PostgreSQL, River UI, OpenTelemetry Collector
make migrate      # Run database migrations
```

### 2. Start the Server

```bash
make run          # gRPC (:50051) + HTTP/JSON (:8080)
```

That's it. No authentication is required by default -- everything just works out of the box.

### 3. Open the Web Dashboard

Build and run with the embedded UI:

```bash
make build-with-ui    # Builds frontend + server with embedded UI
SPARROW_SERVE_UI=true ./build/server-*
```

Open `http://localhost:8080/`. For development with hot-reload, use `make run-web` (Vite dev server on `:5173`).

### 4. End-to-End Walkthrough

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

Or open the web dashboard, or the River queue UI at `http://localhost:8082`.

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
| `TenantService` | CreateTenant, GetTenant, ListTenants, UpdateTenant, DeleteTenant |
| `APIKeyService` | CreateAPIKey, ListAPIKeys, RevokeAPIKey |

HTTP endpoint pattern: `POST http://localhost:8080/webhook.<Service>/<Method>`

## Useful Commands

| Command | Description |
|---|---|
| `make docker-dev` | Start Postgres, River UI, OTel Collector |
| `make migrate` | Run database migrations |
| `make run` | Start the server |
| `make run-web` | Start the web dashboard dev server |
| `make build` | Build server binary |
| `make build-ui` | Build frontend for embedding |
| `make build-with-ui` | Build frontend + server binary with embedded UI |
| `make example` | Run the example gRPC client |
| `make test` | Run tests |
| `make lint` | Run linter |
| `make docker-purge` | Tear down dev infrastructure |

---

# Advanced Usage

Everything below is optional. Sparrow works without authentication, multi-tenancy configuration, or a separate UI process. These features are for production deployments and teams that need access control.

---

## Multi-Tenancy

Sparrow supports full multi-tenancy with data isolation at the database level. Every resource (webhooks, events, subscriptions, deliveries) is scoped to a tenant.

- On first boot, Sparrow auto-creates a **default tenant** and a **root API key** (printed to stdout)
- All data is scoped to a tenant ID -- queries never leak across tenants
- Tenants are identified by UUID and have a URL-safe slug derived from their name
- Tenant status can be `active`, `suspended`, or `archived`

### Managing Tenants

```bash
# Create a new tenant
curl -s -X POST http://localhost:8080/webhook.TenantService/CreateTenant \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{"name": "Acme Corp"}'

# List all tenants
curl -s -X POST http://localhost:8080/webhook.TenantService/ListTenants \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{}'
```

---

## Authentication & Authorization

Sparrow supports two authentication methods that can be used independently or together:

1. **API keys** -- for programmatic / machine-to-machine access
2. **JWT (OIDC)** -- for browser-based access via any identity provider (Clerk, Auth0, Keycloak, etc.)

Both are optional. When `SPARROW_AUTH_ENABLED=false` (the default), all requests use the default tenant with admin access.

### API Key Authentication

Set `SPARROW_AUTH_ENABLED=true` to require authentication. API keys use the format `sk_<tenant_slug>_<random>` and are passed via the `Authorization: Bearer` header.

```bash
# Create an API key for a tenant
curl -s -X POST http://localhost:8080/webhook.APIKeyService/CreateAPIKey \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{
    "tenant_id": "<tenant_id>",
    "name": "Production Key",
    "role": "tenant:admin"
  }'
# Save the raw_key from the response -- it is only shown once

# List API keys
curl -s -X POST http://localhost:8080/webhook.APIKeyService/ListAPIKeys \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{"tenant_id": "<tenant_id>"}'

# Revoke an API key
curl -s -X POST http://localhost:8080/webhook.APIKeyService/RevokeAPIKey \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{"id": "<api_key_id>"}'
```

### JWT / OIDC Authentication

Sparrow can verify RS256 JWTs from any OIDC-compliant identity provider. The backend is **provider-agnostic** -- it validates tokens using standard JWKS and reads configurable claims. No Clerk, Auth0, or Keycloak SDK is linked into the Go binary.

**Enable JWT auth** by setting `SPARROW_JWKS_URL` alongside `SPARROW_AUTH_ENABLED=true`:

```bash
export SPARROW_AUTH_ENABLED=true
export SPARROW_JWKS_URL=https://your-provider.example.com/.well-known/jwks.json

# Optional -- defaults are Clerk-compatible
export SPARROW_JWT_TENANT_CLAIM=org_id      # claim containing the tenant/org identifier
export SPARROW_JWT_ROLE_CLAIM=org_role       # claim containing the user's role
export SPARROW_JWT_ISSUER=https://your-provider.example.com  # reject tokens from other issuers
```

When both JWT and API key authentication are enabled, the interceptor tries each authenticator in order -- a request can authenticate with either method. This lets the web UI use JWTs while scripts and CI pipelines use API keys.

**Tenant mapping:** JWT tokens contain an external org identifier (e.g., Clerk `org_id`). Sparrow maps this to an internal tenant via the `external_id` column on the `tenants` table. You must create the tenant and set its `external_id` before users can authenticate:

```bash
curl -s -X POST http://localhost:8080/webhook.TenantService/CreateTenant \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{"name": "Acme Corp", "external_id": "org_2xYz..."}'
```

Unknown external IDs are rejected (no auto-provisioning).

### RBAC Roles

| Role | Scope | Description |
|---|---|---|
| `tenant:admin` | Tenant-wide | Full access to all resources within the tenant |
| `tenant:member` | Tenant-wide | Read/write access to webhooks, events, subscriptions |
| `namespace:admin` | Single namespace | Full access within a specific namespace |
| `namespace:member` | Single namespace | Read/write access within a specific namespace |
| `namespace:viewer` | Single namespace | Read-only access within a specific namespace |

JWT role mapping (configurable): `org:admin` -> `tenant:admin`, `org:member` -> `tenant:member`.

Platform admin keys (`is_platform_admin: true`) have cross-tenant access for management operations.

---

## Web UI Authentication (Pluggable Providers)

The SvelteKit web dashboard has a pluggable auth provider system. The active provider is selected via environment variables -- Clerk is included out of the box, and adding new providers requires no changes to the layout or services layer.

**Provider selection** (via `PUBLIC_AUTH_PROVIDER` or auto-detected from provider-specific keys):

| `PUBLIC_AUTH_PROVIDER` | Provider-Specific Key | Result |
|---|---|---|
| *(unset)* | *(unset)* | No authentication (open access) |
| *(unset)* | `PUBLIC_CLERK_PUBLISHABLE_KEY=pk_...` | Clerk (auto-detected) |
| `clerk` | `PUBLIC_CLERK_PUBLISHABLE_KEY=pk_...` | Clerk (explicit) |
| `none` | *(any)* | No authentication (forced) |

**To enable Clerk:**

```bash
# In web/.env
PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_your-key-here
```

When no provider is active, the frontend runs without authentication (backwards compatible). When Clerk is active, users must sign in before accessing the dashboard, and session JWTs are automatically attached to all API requests.

**Adding a new auth provider** (e.g., Auth0):

1. Create `web/src/lib/auth/providers/auth0/Auth0AuthShell.svelte` with the same snippet contract (`header`, `children`)
2. Call `registerTokenProvider()` from `web/src/lib/auth.ts` with your provider's token getter
3. Add `"auth0"` to `AuthProviderType` in `web/src/lib/auth/types.ts`
4. Add detection logic in `web/src/lib/auth/provider.ts`
5. Add a case in `web/src/lib/auth/AuthShell.svelte`

The services layer (`services.ts`) and backend remain completely unchanged -- they only see JWTs.

---

## Configuration Reference

All environment variables in one place.

### Backend

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://localhost/riverqueue?sslmode=disable` | PostgreSQL connection string |
| `HTTP_PORT` | `8080` | Connect-RPC HTTP server port |
| `GRPC_PORT` | `50051` | gRPC server port |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `localhost:4317` | OpenTelemetry collector |
| `ENVIRONMENT` | `development` | Environment name for OTel resource |
| `SPARROW_AUTH_ENABLED` | `false` | Enable authentication (API keys and/or JWT) |
| `SPARROW_AUTO_BOOTSTRAP` | `true` | Auto-create default tenant and root API key on startup |
| `SPARROW_ROOT_API_KEY` | *(auto-generated)* | Pre-defined root API key (for deterministic deploys) |
| `SPARROW_SERVE_UI` | `false` | Serve embedded web UI on the HTTP port |
| `SPARROW_JWKS_URL` | *(unset)* | JWKS endpoint URL for JWT verification (enables JWT auth) |
| `SPARROW_JWT_TENANT_CLAIM` | `org_id` | JWT claim containing the tenant/org identifier |
| `SPARROW_JWT_ROLE_CLAIM` | `org_role` | JWT claim containing the user's role |
| `SPARROW_JWT_ISSUER` | *(unset)* | Expected JWT issuer (rejects tokens from other issuers) |

### Frontend

| Variable | Default | Description |
|---|---|---|
| `PUBLIC_API_URL` | `http://localhost:8080` | Sparrow backend API URL (Connect-RPC HTTP/JSON) |
| `PUBLIC_AUTH_PROVIDER` | *(auto-detect)* | Auth provider: `clerk`, `none`, or unset for auto-detect |
| `PUBLIC_CLERK_PUBLISHABLE_KEY` | *(unset)* | Clerk publishable key (enables Clerk auth when set) |

---

## Further Reading

- [TECHNICAL.md](TECHNICAL.md) -- Architecture deep-dive: database schema, queue design, error classification, health state machine, HTTP client internals, boot sequence
- [web/README.md](web/README.md) -- Web dashboard development guide
- [docs/TEMPLATE_FUNCTIONS.md](docs/TEMPLATE_FUNCTIONS.md) -- Payload transformation reference
- [examples/grpc_client.go](examples/grpc_client.go) -- Full gRPC client example
- [proto/webhook.proto](proto/webhook.proto) -- Service and message definitions

## License

MIT
