## Why

When a registered webhook starts failing — health degrading to `degraded`/`unhealthy`, or an
event's delivery exhausting all retries — nobody is told; someone has to notice it in the UI.
Tenants need an email alert for their own webhooks' health, delivered by Sparrow itself, without
Sparrow's internal event/subscription mechanism becoming a tenant-facing concept.

## What Changes

- Sparrow emits its own events (a new, growable catalog under a seeded internal `_sparrow`
  consumer): `sparrow.webhook.health_changed` (any health transition, degrade or recover) and
  `sparrow.webhook.delivery_failed` (all delivery attempts for one event to one webhook exhausted).
- New tenant-facing feature — alert-email configuration: a consumer opts a single webhook or all
  their webhooks into email alerts for chosen event types, via a small dedicated REST resource
  (`webhook_alert_configs`). Tenants configure an email; they never see or subscribe to the
  underlying events.
- Internal delivery reuses the existing core pipeline end to end: `_sparrow` owns one
  SendGrid-recipe webhook + subscriptions to the two events; recipient fan-out happens at push
  time (matching alert configs resolved into the payload as `alert_recipients`), one event → one
  SendGrid call with N `personalizations`.
- New `sendgrid` recipe in `satellites/recipes/` (SendGrid v3 Mail Send, bearer auth).
- Feedback-loop guard: no self-events are emitted for webhooks owned by `_sparrow`.

## Capabilities

### New Capabilities

- `system-events`: Sparrow-authored event types — definition, registration with JSON schema,
  emission points (health transition, terminal delivery failure), payload contract (event facts +
  derived metadata), `_sparrow` internal consumer, feedback-loop guard.
- `webhook-alert-emails`: the tenant-facing alert-email feature — `webhook_alert_configs` resource
  (webhook-level and consumer-level scope, per-event-type opt-in), REST API, push-time recipient
  fan-out, SendGrid delivery through the core pipeline.

### Modified Capabilities

<!-- none: existing event/subscription/delivery behavior is unchanged; this change only adds new
     producers/consumers on top of it. No existing specs exist yet in this repo. -->

## Impact

- `internal/webhooks/store/health_repository.go` (`UpdateWebhookHealthState`) — emit
  `health_changed` on transitions.
- `internal/webhooks/queue/webhook_worker.go` — emit `delivery_failed` at the retry-exhaustion
  point.
- New migration: `webhook_alert_configs` table + `_sparrow` consumer seed.
- New REST handler (own file under `internal/rest/`): alert-email config CRUD.
- `satellites/recipes/sendgrid.yaml` — new recipe.
- Startup: idempotent `RegisterEvent` for the two event types (with JSON schemas).
- OpenAPI spec regeneration (`make generate`) for the new REST resource.
