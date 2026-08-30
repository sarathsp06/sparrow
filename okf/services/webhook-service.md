---
type: REST Resource
title: Webhooks
description: Webhook registration and configuration management — 10 endpoints
tags: [rest, webhooks]
timestamp: 2026-08-29T00:00:00Z
---

# Webhooks

Registered under the `Webhooks` tag in the Huma-generated OpenAPI spec. Implemented in `internal/rest/webhook.go`.

## Endpoints

| Method | Path | OperationID | Description |
|--------|------|-------------|-------------|
| POST | `/v1/namespaces/{namespace}/webhooks` | `registerWebhook` | Create webhook + auto-create subscriptions |
| GET | `/v1/namespaces/{namespace}/webhooks` | `listWebhooks` | Paginated listing with filters |
| GET | `/v1/namespaces/{namespace}/webhooks/{webhook_id}` | `getWebhook` | Get a webhook by id |
| PATCH | `/v1/namespaces/{namespace}/webhooks/{webhook_id}` | `updateWebhook` | Partial update with field mask |
| DELETE | `/v1/namespaces/{namespace}/webhooks/{webhook_id}` | `deleteWebhook` | Delete webhook (cascades subscriptions, deliveries) |
| POST | `/v1/namespaces/{namespace}/webhooks/{webhook_id}:pause` | `pauseWebhook` | Set webhook inactive |
| POST | `/v1/namespaces/{namespace}/webhooks/{webhook_id}:resume` | `resumeWebhook` | Set webhook active |
| GET | `/v1/namespaces/{namespace}/stats` | `getNamespaceStats` | Aggregate per-namespace delivery stats |
| GET | `/v1/template-functions` | `getTemplateFunctions` | List available Go template functions |
| GET | `/v1/stats` | `getGlobalStats` | Aggregate delivery stats across all namespaces |

## Citations

- `internal/rest/webhook.go` — endpoint registration + handlers
- `internal/rest/conversions.go` — `WebhookOut` REST representation
