---
type: REST Resource
title: Subscriptions
description: Subscription CRUD with Go template transforms — 6 endpoints
tags: [rest, subscriptions]
timestamp: 2026-08-29T00:00:00Z
---

# Subscriptions

Registered under the `Subscriptions` tag in the Huma-generated OpenAPI spec. Implemented in `internal/rest/subscription.go`.

## Endpoints

| Method | Path | OperationID | Description |
|--------|------|-------------|-------------|
| POST | `/v1/namespaces/{namespace}/subscriptions` | `createSubscription` | Subscribe a webhook to an event |
| GET | `/v1/namespaces/{namespace}/subscriptions/{subscription_id}` | `getSubscription` | Get subscription details |
| GET | `/v1/namespaces/{namespace}/subscriptions` | `listSubscriptions` | List subscriptions for a webhook or event |
| PATCH | `/v1/namespaces/{namespace}/subscriptions/{subscription_id}` | `updateSubscription` | Update subscription config, transform template |
| DELETE | `/v1/namespaces/{namespace}/subscriptions/{subscription_id}` | `deleteSubscription` | Delete subscription |
| POST | `/v1/subscriptions:testTemplate` | `testSubscriptionTemplate` | Preview Go template transform output |

## Citations

- `internal/rest/subscription.go` — endpoint registration + handlers
