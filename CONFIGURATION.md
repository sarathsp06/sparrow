# Sparrow -- Configuration Reference

All configuration is done via environment variables. No config files needed.

---

## Table of Contents

- [Core](#core)
- [Authentication](#authentication)
- [JWT Claims](#jwt-claims)
- [Identity Provider](#identity-provider)
- [Observability](#observability)
- [Web UI (Frontend)](#web-ui-frontend)
- [Deployment Modes](#deployment-modes)
- [Deployment Guide](#deployment-guide)
  - [Docker Compose](#docker-compose)
  - [Binary Deployment](#binary-deployment)
- [Enabling Authentication](#enabling-authentication)
  - [API Keys Only](#api-keys-only)
  - [API Keys + JWT](#api-keys--jwt)
- [OIDC Provider Setup](#oidc-provider-setup)
  - [Clerk](#clerk)
  - [Keycloak](#keycloak)
  - [Auth0](#auth0)
  - [Authelia / Zitadel / Generic OIDC](#authelia--zitadel--generic-oidc)
- [Namespace Role Resolution](#namespace-role-resolution)
- [Self-Hosted Deployment](#self-hosted-deployment)

---

## Core

- `DATABASE_URL` (required) -- PostgreSQL connection string
- `SPARROW_SERVE_UI` -- Serve the embedded web dashboard on `:8080`. Default: `false`
- `ENVIRONMENT` -- `development` or `production` (affects logging/OTel)

---

## Authentication

- `SPARROW_AUTH_ENABLED` -- Enable authentication. When disabled, all requests use `tenant:admin` on the default tenant. Default: `false`
- `SPARROW_ROOT_API_KEY` -- Pre-configured root API key for bootstrap. Default: auto-generated on first boot

---

## JWT Claims

All JWT claim names are configurable so Sparrow works with any OIDC provider.

- `SPARROW_JWKS_URL` -- JWKS endpoint for JWT validation. Works with any OIDC provider (Clerk, Auth0, Keycloak, Authelia, Zitadel, etc.)
- `SPARROW_JWT_TENANT_CLAIM` -- JWT claim containing the tenant/org identifier. Default: `org_id`
- `SPARROW_JWT_ROLE_CLAIM` -- JWT claim containing the user's role. Default: `org_role`
- `SPARROW_JWT_SUBJECT_CLAIM` -- JWT claim containing the user identifier. Default: `sub`
- `SPARROW_JWT_ISSUER` -- Expected JWT issuer (`iss`). Leave empty to skip validation.
- `SPARROW_JWT_AUDIENCES` -- Comma-separated expected audience values (`aud`). Leave empty to skip.
- `SPARROW_JWT_NAMESPACE_ROLES_CLAIM` -- JWT claim for embedded namespace roles. Set to `""` to disable and always resolve from DB. Default: `namespace_roles`
- `SPARROW_JWT_ROLE_MAPPING` -- Comma-separated `provider_role=sparrow_role` pairs. Default: `org:admin=tenant:admin,org:member=tenant:member`

---

## Identity Provider

- `CLERK_SECRET_KEY` -- Clerk secret key. When set, namespace role changes are synced to Clerk org membership metadata so they appear in future JWTs.

---

## Observability

- `OTEL_EXPORTER_OTLP_ENDPOINT` -- OTLP HTTP endpoint for traces, metrics, and logs

---

## Web UI (Frontend)

- `PUBLIC_AUTH_PROVIDER` -- Auth provider: `clerk` or `none`. Auto-detected from provider-specific keys if unset.
- `PUBLIC_CLERK_PUBLISHABLE_KEY` -- Clerk publishable key. Auto-enables Clerk provider when set.
- `PUBLIC_API_URL` -- API base URL for the frontend
- `CORS_ALLOWED_ORIGINS` -- Allowed CORS origins for Connect-RPC

---

## Deployment Modes

Sparrow supports four deployment modes depending on your auth needs:

- **Zero-auth** -- Set only `DATABASE_URL`. All requests get `tenant:admin`. Best for single-user/dev.
- **API-key only** -- Add `SPARROW_AUTH_ENABLED=true`. Root key created on boot. Create more via API. No OIDC.
- **Any OIDC provider** -- Add `SPARROW_JWKS_URL=...`. JWT auth with any provider. Namespace roles resolved from DB (30s cache).
- **Full Clerk** -- Add `CLERK_SECRET_KEY=...`. Adds namespace role sync to JWT claims. Performance optimization, not required.

---

## Deployment Guide

### Docker Compose

The simplest way to run Sparrow. This starts PostgreSQL, runs migrations, and launches the server with the embedded web UI.

```bash
docker compose up -d
```

The server is available at:
- **Web UI:** http://localhost:8080
- **HTTP API (Connect-RPC):** http://localhost:8080
- **gRPC API:** localhost:50051

On first boot, a root API key is printed to the logs:

```bash
docker compose logs sparrow
```

To stop:

```bash
docker compose down        # stop containers
docker compose down -v     # stop and delete data
```

### Binary Deployment

Build from source and run directly:

```bash
# Build with embedded UI
make build-with-ui

# Start infrastructure (or provide your own Postgres)
export DATABASE_URL=postgres://user:pass@localhost:5432/sparrow?sslmode=disable

# Run migrations
./build/server-* migrate  # or: make migrate

# Start the server
SPARROW_SERVE_UI=true ./build/server-*
```

---

## Enabling Authentication

### API Keys Only

The simplest auth mode. No OIDC provider needed.

```bash
SPARROW_AUTH_ENABLED=true
```

On first boot, Sparrow prints a root API key. Use it to create additional keys:

```bash
curl -s -X POST http://localhost:8080/webhook.APIKeyService/CreateAPIKey \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk_default_<root_key>" \
  -d '{
    "tenant_id": "<tenant_id>",
    "name": "Production Key",
    "role": "tenant:admin"
  }'
```

### API Keys + JWT

For web UI with an identity provider. Both auth methods work simultaneously -- JWT for browser sessions, API keys for scripts.

```bash
SPARROW_AUTH_ENABLED=true
SPARROW_JWKS_URL=https://your-provider.example.com/.well-known/jwks.json
SPARROW_JWT_TENANT_CLAIM=org_id
SPARROW_JWT_ROLE_CLAIM=org_role
SPARROW_JWT_SUBJECT_CLAIM=sub
SPARROW_JWT_ISSUER=https://...        # optional
SPARROW_JWT_AUDIENCES=api,web         # optional
```

---

## OIDC Provider Setup

Sparrow's backend is provider-agnostic. It validates JWTs using standard JWKS and reads configurable claims. No provider SDK is linked into the Go binary.

Any OIDC provider works as long as it:
1. Publishes a JWKS endpoint with RS256 keys
2. Includes a tenant/org identifier claim in the JWT
3. Optionally includes a role claim

### Clerk

Clerk is a managed identity provider. Sparrow auto-provisions tenants when users create Clerk organizations -- no webhooks or manual setup needed.

**1. Create a Clerk application** at [clerk.com](https://clerk.com) and enable Organizations.

**2. Create a JWT template** in Clerk Dashboard > JWT Templates:

- Template name: `sparrow` (or any name)
- Claims (JSON):
  ```json
  {
    "org_id": "{{org.id}}",
    "org_role": "{{org_membership.role}}",
    "namespace_roles": "{{org_membership.public_metadata.namespace_roles}}"
  }
  ```

The `namespace_roles` claim is optional but recommended. When present, Sparrow reads namespace roles directly from the JWT instead of querying the database on each request.

**3. Configure the backend:**

```bash
SPARROW_AUTH_ENABLED=true
SPARROW_JWKS_URL=https://<your-instance>.clerk.accounts.dev/.well-known/jwks.json
SPARROW_JWT_TENANT_CLAIM=org_id
SPARROW_JWT_ROLE_CLAIM=org_role
SPARROW_JWT_ISSUER=https://<your-instance>.clerk.accounts.dev

# Optional: enable namespace role sync to Clerk metadata
CLERK_SECRET_KEY=sk_live_your-key-here
```

When `CLERK_SECRET_KEY` is set, Sparrow syncs namespace role changes to Clerk org membership `publicMetadata`. On the next JWT refresh (~60s), the roles appear in the session token, eliminating per-request DB lookups.

**4. Configure the frontend** (in `web/.env` or environment):

```bash
PUBLIC_CLERK_PUBLISHABLE_KEY=pk_test_your-key-here
PUBLIC_API_URL=http://localhost:8080
```

### Keycloak

```bash
SPARROW_AUTH_ENABLED=true
SPARROW_JWKS_URL=https://keycloak.example.com/realms/sparrow/protocol/openid-connect/certs
SPARROW_JWT_TENANT_CLAIM=organization_id
SPARROW_JWT_ROLE_CLAIM=realm_role
SPARROW_JWT_ISSUER=https://keycloak.example.com/realms/sparrow
SPARROW_JWT_ROLE_MAPPING=admin=tenant:admin,user=tenant:member
SPARROW_JWT_NAMESPACE_ROLES_CLAIM=""
```

### Auth0

```bash
SPARROW_AUTH_ENABLED=true
SPARROW_JWKS_URL=https://your-tenant.auth0.com/.well-known/jwks.json
SPARROW_JWT_TENANT_CLAIM=org_id
SPARROW_JWT_ROLE_CLAIM=https://sparrow.example.com/role
SPARROW_JWT_ISSUER=https://your-tenant.auth0.com/
SPARROW_JWT_AUDIENCES=https://api.sparrow.example.com
SPARROW_JWT_ROLE_MAPPING=Admin=tenant:admin,Member=tenant:member
SPARROW_JWT_NAMESPACE_ROLES_CLAIM=""
```

### Authelia / Zitadel / Generic OIDC

```bash
SPARROW_AUTH_ENABLED=true
SPARROW_JWKS_URL=https://auth.example.com/.well-known/jwks.json
SPARROW_JWT_TENANT_CLAIM=tenant_id
SPARROW_JWT_ROLE_CLAIM=role
SPARROW_JWT_SUBJECT_CLAIM=sub
SPARROW_JWT_ROLE_MAPPING=administrator=tenant:admin,member=tenant:member
SPARROW_JWT_NAMESPACE_ROLES_CLAIM=""
```

---

## Namespace Role Resolution

Namespace roles control fine-grained access within a tenant. There are two resolution paths:

1. **JWT claim (fast path)** -- When the JWT contains a `namespace_roles` claim (a JSON array like `["namespace:admin:customer-a", "namespace:viewer:customer-b"]`), roles are read directly from the token. Zero DB queries. Requires an identity provider that supports embedding custom metadata in JWTs.

2. **Database fallback (universal)** -- When the claim is absent, empty, or disabled (`SPARROW_JWT_NAMESPACE_ROLES_CLAIM=""`), memberships are resolved from the `namespace_memberships` table with a 30-second in-memory cache. Works with any OIDC provider.

For self-hosted deployments without Clerk, the DB fallback is the primary path. It is functionally identical -- the only difference is a cached DB query per JWT-authenticated request.

---

## Self-Hosted Deployment

Sparrow is designed for self-hosting. No mandatory external service dependencies beyond PostgreSQL.

**Minimal self-hosted setup (no auth):**

```yaml
# docker-compose.yml
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: sparrow
      POSTGRES_USER: sparrow
      POSTGRES_PASSWORD: sparrow
    ports: ["5432:5432"]

  sparrow:
    image: sparrow:latest
    environment:
      DATABASE_URL: postgres://sparrow:sparrow@postgres:5432/sparrow?sslmode=disable
      SPARROW_SERVE_UI: "true"
    ports: ["8080:8080", "50051:50051"]
    depends_on:
      postgres: { condition: service_healthy }
```

**Self-hosted with OIDC auth (e.g., Keycloak):**

```yaml
# Add to the sparrow service environment:
SPARROW_AUTH_ENABLED: "true"
SPARROW_JWKS_URL: "https://keycloak.internal/realms/myapp/protocol/openid-connect/certs"
SPARROW_JWT_TENANT_CLAIM: "organization_id"
SPARROW_JWT_ROLE_CLAIM: "role"
SPARROW_JWT_ISSUER: "https://keycloak.internal/realms/myapp"
SPARROW_JWT_ROLE_MAPPING: "admin=tenant:admin,user=tenant:member"
SPARROW_JWT_NAMESPACE_ROLES_CLAIM: ""
```

**Key points:**

- **Clerk is optional.** It is only used for JWT validation (but any OIDC JWKS works) and syncing namespace roles to JWT claims (a performance optimization). Without `CLERK_SECRET_KEY`, namespace roles come from the database.
- **No external API calls at runtime** (unless Clerk sync is enabled). All auth validation uses cached JWKS keys and local DB lookups.
- **API keys work standalone.** If you don't have an OIDC provider, set `SPARROW_AUTH_ENABLED=true` and use API keys for all access.
- **Auto-provisioning:** Unknown tenant/org IDs in JWTs are auto-provisioned as new tenants (limited to 2 per user). No manual tenant setup needed.
