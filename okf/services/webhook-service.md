---
type: REST Resource
title: Webhooks
description: Register, configure, pause/resume, and inspect consumer webhook endpoints; delivery stats and portal-token minting
tags: [rest, webhooks]
timestamp: 2026-09-16T00:00:00Z
---

# Webhooks

Consumer-scoped webhook endpoints. Registration auto-creates subscriptions; endpoints can be updated with a field mask, paused/resumed, and deleted (cascading their subscriptions and deliveries). The resource also exposes per-consumer and global delivery statistics, the available payload-transformation template functions, and minting of consumer-scoped portal access tokens. Registered under the `Webhooks` tag in the Huma-generated OpenAPI spec.

The authoritative endpoint list — paths, methods, and schemas — is the OpenAPI spec at `api/openapi.yaml` (browse at `/docs`).

## Citations

- `internal/rest/webhook.go` — endpoint registration + handlers
- `api/openapi.yaml` — canonical endpoint definitions
