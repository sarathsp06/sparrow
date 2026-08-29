---
type: REST Resource
title: Health
description: Webhook health queries — status, filtering, summary — 3 endpoints
tags: [rest, health]
timestamp: 2026-08-29T00:00:00Z
---

# Health

Registered under the `Health` tag in the Huma-generated OpenAPI spec. Implemented in `internal/rest/health.go`.

## Endpoints

| Method | Path | OperationID | Description |
|--------|------|-------------|-------------|
| GET | `/v1/namespaces/{namespace}/webhooks/{webhook_id}/health` | `getWebhookHealth` | Health state for a single webhook |
| GET | `/v1/health-summary` | `getHealthSummary` | Aggregate health summary across webhooks |
| GET | `/v1/webhooks` | `listWebhooksByHealth` | List webhooks across all namespaces, filtered by health status |

## Citations

- `internal/rest/health.go` — endpoint registration + handlers
