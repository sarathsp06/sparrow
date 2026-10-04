---
type: Concept
title: Subscription
description: Binding between a registered webhook and an event type with optional Go template transform
tags: [subscription, template, transform]
timestamp: 2026-06-22T00:00:00Z
---

# Subscription

A subscription binds a [webhook](/concepts/webhook-registration.md) to an [event](/concepts/event.md) under a consumer. When an event is pushed, all matching subscriptions receive deliveries.

## Key Fields

- `webhook_id` — FK to webhook_registrations
- `event_name` — event to subscribe to (or `*` for catch-all)
- `transform_template` — optional Go template string
- `label_filters` — label-based filtering

## Template Transforms

When `transform_template` is set, the event payload is run through the Go template before delivery.

- `template_missing_key`: `error` (default) renders with `missingkey=error`, so a missing field fails the render; `zero` renders it as `<no value>`.
- `on_transform_error`: `fail` (default) marks the delivery failed with `template_error`, sends nothing, no automatic retry; `fallback` sends the [envelope payload](/packages/internal-webhooks-client.md). The error is stored in `webhook_deliveries.template_error` either way, and never recorded as a health event.

## Pause

`paused_at` / `paused_reason`. While paused, fan-out creates deliveries with status `paused` and no job; queued or retrying deliveries are held as `paused` by the worker too; resume does not send them (retry with `status=paused&subscription_id=…`). A paused webhook (manual or auto-disabled) behaves the same for all its subscriptions. A pause never affects webhook health.

## Citations

- `db/migrations/000001.up.sql` — initial schema
- `db/migrations/000012.up.sql` — labels
- `db/migrations/000031_template_error_handling.up.sql` — on_transform_error, template_missing_key, template_error
- `db/migrations/000033_subscription_pause.up.sql` — paused_at, paused_reason
