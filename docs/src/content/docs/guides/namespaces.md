---
title: Namespaces
description: Organize webhooks and events into logical groups.
sidebar:
  order: 6
---

Namespaces let you organize webhooks and events into logical groups. Every webhook and event belongs to a namespace.

A `default` namespace is available out of the box. You can create additional namespaces to separate concerns (e.g., `billing`, `notifications`, `integrations`).

## Create a Namespace

```bash
curl -X POST http://localhost:8080/webhook.NamespaceService/CreateNamespace \
  -H "Content-Type: application/json" \
  -d '{
    "name": "billing",
    "description": "Billing and payment webhooks"
  }'
```

## List Namespaces

```bash
curl -X POST http://localhost:8080/webhook.NamespaceService/ListNamespaces \
  -H "Content-Type: application/json" \
  -d '{}'
```

## Update a Namespace

```bash
curl -X POST http://localhost:8080/webhook.NamespaceService/UpdateNamespace \
  -H "Content-Type: application/json" \
  -d '{
    "id": "NAMESPACE_ID",
    "description": "Updated description"
  }'
```

## Delete a Namespace

:::caution
Deleting a namespace cascades to all webhooks, subscriptions, and deliveries within it.
:::

```bash
curl -X POST http://localhost:8080/webhook.NamespaceService/DeleteNamespace \
  -H "Content-Type: application/json" \
  -d '{"id": "NAMESPACE_ID"}'
```

## Namespace Scoping

All API operations that involve webhooks, events, or deliveries accept a `namespace` parameter. Queries are automatically scoped to the specified namespace.

For example, listing webhooks in the `billing` namespace:

```bash
curl -X POST http://localhost:8080/webhook.WebhookService/ListWebhooks \
  -H "Content-Type: application/json" \
  -d '{"namespace": "billing"}'
```

:::note
The NamespaceService is implemented directly in Go (no proto definition) and is only accessible via Connect-RPC HTTP/JSON on port 8080, not via gRPC on port 50051.
:::
