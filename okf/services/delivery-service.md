---
type: REST Resource
title: Deliveries
description: Delivery status, listing, retry, and batch retry — 11 endpoints
tags: [rest, deliveries]
timestamp: 2026-08-29T00:00:00Z
---

# Deliveries

Registered under the `Deliveries` tag in the Huma-generated OpenAPI spec. Implemented in `internal/rest/delivery.go`.

## Endpoints

| Method | Path | OperationID | Description |
|--------|------|-------------|-------------|
| GET | `/v1/namespaces/{namespace}/deliveries/{delivery_id}` | `getDelivery` | Get a single delivery status |
| GET | `/v1/namespaces/{namespace}/deliveries` | `listDeliveries` | Filterable delivery listing (supports `prepare_retry`) |
| POST | `/v1/namespaces/{namespace}/deliveries/{delivery_id}:retry` | `retryDelivery` | Retry a single failed delivery |
| POST | `/v1/namespaces/{namespace}/deliveries:retry` | `retryDeliveriesByWebhook` | Retry every eligible delivery for one webhook |
| GET | `/v1/namespaces/{namespace}/deliveries/{delivery_id}/attempts` | `getDeliveryAttempts` | List HTTP attempt records for a delivery |
| POST | `/v1/namespaces/{namespace}/deliveries:retryBatch` | `startDeliveryRetryJob` | Batch retry via snapshot — uses batch_jobs |
| GET | `/v1/namespaces/{namespace}/retry-jobs/{job_id}` | `getDeliveryRetryJob` | Poll batch retry progress |
| POST | `/v1/namespaces/{namespace}/retry-jobs/{job_id}:cancel` | `cancelDeliveryRetryJob` | Cancel a batch retry operation |
| GET | `/v1/deliveries/{delivery_id}` | `getDeliveryGlobal` | Get a delivery by id (any namespace) |
| GET | `/v1/deliveries/{delivery_id}/attempts` | `getDeliveryAttemptsGlobal` | Get a delivery's attempt history (any namespace) |
| POST | `/v1/deliveries/{delivery_id}:retry` | `retryDeliveryGlobal` | Retry a single delivery (any namespace) |

## Citations

- `internal/rest/delivery.go` — endpoint registration + handlers
