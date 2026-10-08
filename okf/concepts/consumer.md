---
type: Concept
title: Consumer
description: Implicit scoping name for webhooks, subscriptions and events within a tenant; _sparrow is reserved for Sparrow's own system events
tags: [consumer, scoping]
timestamp: 2026-10-08T00:00:00Z
---

# Consumer

A consumer is a name that scopes webhooks, subscriptions, event records and deliveries within a tenant (e.g. `billing`, `acme-prod`). Most API paths carry it: `/v1/consumers/{consumer}/...`.

## Implicit, no table

Consumers are not stored as rows of their own: the `consumers` table (migration 000008) was dropped in 000013. A consumer exists once a webhook is registered or an event is pushed under its name. `GET /v1/consumers?q=` searches the distinct names on `webhook_registrations` and `event_records` with a loose index scan over their `(tenant_id, consumer)` indexes, so it stays cheap for tenants with hundreds of consumers.

## Reserved: `_sparrow`

`_sparrow` (`tenant.SystemConsumer`) is where Sparrow pushes its own `sparrow.webhook.*` events and where operators register the alert-delivery webhook. The split is enforced:

- `sparrow.*` events can only be subscribed under `_sparrow`, and `_sparrow` can only subscribe to `sparrow.*` events or `*` (`checkEventForConsumer`).
- `GET /v1/event-types` lists `sparrow.*` types only with `consumer=_sparrow`.
- Consumer-scoped tokens and invites cannot be minted for `_sparrow`; only the master key or a tenant-wide token reaches it.
- Otherwise it is listed and counted like any other consumer.

## In the UI

There is no global consumer switcher. Each list page (Webhooks, Deliveries, Event reports, Health) has its own consumer filter, kept in the URL as `?consumer=`; empty means all consumers. Forms take the consumer from `?consumer=` or a searchable picker and never assume `default`.

## Citations

- `internal/webhooks/store/consumer_repository.go` — `ListConsumers`
- `internal/webhooks/event_type_save.go` — `checkEventForConsumer`, `eventTypesForConsumer`
- `internal/tenant/tenant.go` — `SystemConsumer`
- `internal/accessauth/accessauth.go` — `ValidateConsumer`
- `web/src/lib/consumer.svelte.ts`, `web/src/lib/components/ConsumerPicker.svelte`
- Migrations 000008 (create consumers), 000013 (drop consumer entities)
