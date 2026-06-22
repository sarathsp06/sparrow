---
type: Concept
title: Subscription
description: Binding between a registered webhook and an event type with optional Go template transform
tags: [subscription, template, transform]
timestamp: 2026-06-22T00:00:00Z
---

# Subscription

A subscription binds a [webhook](/concepts/webhook-registration.md) to an [event](/concepts/event.md) under a namespace. When an event is pushed, all matching subscriptions receive deliveries.

## Key Fields

- `webhook_id` — FK to webhook_registrations
- `event_name` — event to subscribe to (or `*` for catch-all)
- `transform_template` — optional Go template string
- `label_filters` — label-based filtering

## Template Transforms

When `transform_template` is set, the event payload is run through the Go template before delivery. On failure, falls back to the [envelope payload](/packages/internal-webhooks-client.md).

## Citations

- `db/migrations/000001.up.sql` — initial schema
- `db/migrations/000012.up.sql` — labels
