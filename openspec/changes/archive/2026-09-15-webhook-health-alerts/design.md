## Context

Full design discussion and rationale: `docs/design/webhook-health-alerts.pseudocode.md`. This file
records only the load-bearing technical decisions.

## Decisions

- **Emission points** (the only two hook points, both already exist):
  - `sparrow.webhook.health_changed`: `UpdateWebhookHealthState` in
    `internal/webhooks/store/health_repository.go` — the single place old vs. new computed health
    is compared. Skip `unknown → healthy`.
  - `sparrow.webhook.delivery_failed`: the terminal/retry-exhaustion determination in
    `internal/webhooks/queue/webhook_worker.go`.
- **Event consumer is `_sparrow`, not the affected webhook's consumer.** Subscription matching is
  exact on the event's consumer; the interested party for these events is Sparrow itself. Tenants
  get a feature, not an event.
- **Fan-out at push time.** Templates render from `payload` only (context:
  `event_id`/`event_name`/`timestamp`/`attempt`/`payload`; no DB access) — so matching
  `webhook_alert_configs` rows are resolved where the event is emitted and embedded as
  `payload.alert_recipients`. One event per occurrence; the SendGrid transform template ranges
  recipients into SendGrid v3 `personalizations` (one API call, N emails).
- **Payload contract** (convention for all future system events): flat event facts
  (`webhook_id`, `consumer`, `url`, plus event-specific fields) + derived metadata
  (`alert_recipients`) — derived data is anything pre-fetched because templates can't fetch it.
- **Subject direction via template branching** on `old_health`/`new_health` — no second health
  event type, no extra payload field.
- **`webhook_alert_configs` table**: `id, tenant_id, consumer, webhook_id (nullable →
  consumer-wide), email, event_types[], created_at`. Own REST handler file under `internal/rest/`
  (it's a feature resource, not a subscription variant).
- **Operator setup is the existing recipe flow**: new `satellites/recipes/sendgrid.yaml`
  (SendGrid v3 Mail Send, bearer auth; params `api_key`, `from_email`, `from_name`), applied once
  under `_sparrow`. No provisioning system, no env-var bootstrap.
- **Loop guard**: skip emission when the affected webhook's consumer is `_sparrow`.

## Alternatives rejected

- Per-tenant self-service *event subscriptions* (cross-consumer subscription to a shared webhook):
  exposes internal mechanism as tenant surface; rejected in review.
- `owner_email` column on `webhook_registrations`: replaced by `webhook_alert_configs` (multiple
  recipients, per-event-type opt-in, consumer-wide scope).
- One event per recipient: N deliveries for N recipients; SendGrid `personalizations` does the same
  in one call.
- Reconciling ops alerting into the same table: ops has OTel; different content, different
  triggers. Out of scope.
