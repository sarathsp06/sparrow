---
type: Configuration
title: Environment Variables
description: All server configuration loaded from environment variables via kelseyhightower/envconfig
tags: [config, env-vars]
timestamp: 2026-09-14T00:00:00Z
---

# Environment Variables

All configuration via environment variables using `kelseyhightower/envconfig`.

## Core

| Variable | Purpose | Default |
|----------|---------|---------|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://localhost/riverqueue?sslmode=disable` |
| `SPARROW_ENCRYPTION_KEYS` | KEK keyring as `<key-id>=<64-char-hex-key>` entries (`key-id` chars: `A-Z`, `a-z`, `0-9`, `_`, `-`) | Required |
| `SPARROW_ENCRYPTION_PRIMARY_KEY_ID` | Primary key ID for new encryption and portal token signing | Required |

## Server

| Variable | Purpose | Default |
|----------|---------|---------|
| `SPARROW_HTTP_PORT` | HTTP/REST listen port | `8080` |
| `SPARROW_SERVE_UI` | Serve embedded SvelteKit UI | `false` |
| `SPARROW_MAX_BODY_BYTES` | Max request body size in bytes (min 1 MiB); oversized bodies return `413` | `5242880` (5 MiB) |

## Auth & Security

| Variable | Purpose | Default |
|----------|---------|---------|
| `SPARROW_API_KEY` | Master key for auth, accepted via `X-API-Key` or `Authorization: Bearer`. Also accepted: tenant-wide access tokens. **Required when `ENVIRONMENT=production`** (`Validate()` refuses to start without it); must be at least 32 characters in production. Otherwise optional and all endpoints are open | — (open access) |
| ~~`SPARROW_UI_INJECT_KEY`~~ | Removed. The server never writes the API key into the UI. If still set, a deprecation warning is logged and the value is ignored. | -- |
| `SPARROW_TOKEN_DEFAULT_TTL` | Lifetime of tenant-wide tokens created without `ttl_seconds`/`never_expires` (Go duration). `0` = never expire. | `2160h` |
| `SPARROW_MAX_CAPTURED_RESPONSE_BYTES` | Stored response body cap for `capture_response_body` webhooks (others 1 KiB). Min 1024. | `1048576` |
| `SPARROW_ALLOWED_NETWORKS` | Comma-separated CIDRs or bare IPs (e.g. `10.20.0.0/16,fd12::/48`). Deliveries may reach these networks in addition to public addresses; loopback, cloud metadata and rest of private space stay blocked. Invalid entries fail startup. With an allowlist, `.internal`/`.local` hostnames are allowed (resolved addresses are checked instead of names). Recommended for VPN deployments. | -- |
| `SPARROW_ALLOW_PRIVATE_NETWORKS` | Allow all private IPs as webhook URLs (for local dev/test). Cloud metadata endpoints are still blocked even when `true`. Prefer `SPARROW_ALLOWED_NETWORKS` in production. | `false` |

## Observability

| Variable | Purpose | Default |
|----------|---------|---------|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | OTLP HTTP export endpoint | — |
| `ENVIRONMENT` | `development` or `production`; `production` enforces `SPARROW_API_KEY` | — |

`Warnings()` also logs a non-fatal advisory if `DATABASE_URL` uses `sslmode=disable` against a non-local host.

## CORS

| Variable | Purpose | Default |
|----------|---------|---------|
| `CORS_ALLOWED_ORIGINS` | Comma-separated exact origins allowed cross-origin (trailing `/` ignored; headers `Authorization`, `Content-Type`, `X-API-Key`; no credentials). Needed for a separately hosted UI. Unset: block all when `ENVIRONMENT=production`, allow all otherwise. Implemented in `internal/middleware/cors.go` | — |

## Retention

| Variable | Purpose | Default |
|----------|---------|---------|
| `SPARROW_EVENT_RETENTION_DAYS` | Purge events (and cascaded deliveries) older than N days via an hourly background job; `0` disables retention (data kept forever) | `0` |
| `SPARROW_AUTO_REGISTER_EVENTS` | Push to an unregistered event name creates a schema-less event type instead of returning 404; development only | `false` |

## Template history

No configuration. `subscription_template_versions` keeps the last 20 saved templates per subscription (migration 000034); listed at `GET /v1/consumers/{c}/subscriptions/{id}/templateVersions`.

## AI-assisted template drafting

No variables set: prompt-only mode. `GET /v1/capabilities` reports `ai_drafting.prompt_only=true` and `POST /v1/subscriptions:draftTemplatePrompt` builds the grounded prompt for any chat assistant; `:draftTemplate` answers 503.

| Variable | Purpose | Default |
|----------|---------|---------|
| `SPARROW_AI_PROVIDER` | `anthropic` (Anthropic SDK) or `openai` (plain HTTP to any OpenAI-compatible `/v1/chat/completions`: Ollama, vLLM, LM Studio, llama.cpp, OpenRouter, OpenAI) | `anthropic` |
| `SPARROW_AI_API_KEY` | Provider key. Enables drafting for `anthropic`; optional Bearer token for `openai`. `GET /v1/capabilities` reports `ai_drafting.{enabled,provider,model}` | -- |
| `SPARROW_AI_MODEL` | Model id. Required for `openai` | `claude-haiku-4-5` (anthropic) |
| `SPARROW_AI_BASE_URL` | API base URL. Required for `openai` (enables drafting); optional gateway override for `anthropic` | -- |

## Database Pools

| Pool | Library | Config | Purpose |
|------|---------|--------|---------|
| sqlx | `jmoiron/sqlx` | MaxOpen=25 | App queries |
| pgxpool | `jackc/pgx/v5` | MaxConns=50, MinConns=10, 30min lifetime | River queue |

## Citations

- `internal/config/config.go`
