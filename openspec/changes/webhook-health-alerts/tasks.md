## 1. Foundation

- [x] 1.1 Migration: create `webhook_alert_configs` table and seed the `_sparrow` internal consumer
- [x] 1.2 Register `sparrow.webhook.health_changed` and `sparrow.webhook.delivery_failed` event
      types with JSON schemas, idempotently at server startup
- [x] 1.3 Store layer: repository for `webhook_alert_configs` (create, list by consumer, delete,
      resolve recipients for `{webhook_id, consumer, event_type}`)

## 2. Event emission

- [x] 2.1 Emit `health_changed` on health transitions in `UpdateWebhookHealthState`
      (skip `unknown → healthy`; skip `_sparrow`-owned webhooks; resolve and embed
      `alert_recipients` at push time)
- [x] 2.2 Emit `delivery_failed` at the retry-exhaustion point in `webhook_worker.go`
      (same guards, same payload contract, original `event_id`/`delivery_id`/`error_category`/
      `attempt_count` in facts)

## 3. Tenant-facing API

- [x] 3.1 REST handler (own file in `internal/rest/`): create/list/delete alert-email configs,
      webhook-level and consumer-level routes; reject empty `event_types`
- [x] 3.2 Regenerate OpenAPI spec + clients (`make generate`)

## 4. Delivery channel

- [x] 4.1 New recipe `satellites/recipes/sendgrid.yaml` (SendGrid v3 Mail Send, bearer auth,
      params `api_key`/`from_email`/`from_name`; transform template ranges
      `payload.alert_recipients` into `personalizations`, subject branches on
      `old_health`/`new_health`)
- [x] 4.2 Recipe fixture/test in `satellites/recipes` covering both event templates

## 5. Verification & docs

- [x] 5.1 Tests: transition-only emission (incl. no-event-without-transition), terminal-only
      `delivery_failed`, loop guard, recipient resolution (webhook-level + consumer-wide + empty)
- [x] 5.2 Smoke test end to end: degrade a webhook, observe one event, one SendGrid delivery
      attempt with all recipients
- [x] 5.3 Docs: feature page for alert emails + operator setup (recipe apply under `_sparrow`);
      `graphify update .`
