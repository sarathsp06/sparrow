# Alert Configs -- Email Recipients For Sparrow Health Notifications
Tags: alert-configs, crud

Operators opt email recipients into Sparrow's self-generated health and
delivery-failure alerts, scoped either consumer-wide or to a single webhook.
This walks the full HTTP + DB + encryption round-trip: create, list back the
stored fields, filter by webhook, and delete.

## Register, List, Filter, And Delete Alert Recipients
* Use consumer "alerts"
* Start target "billing"
* Register event type "alerts.invoice.paid"
* Register webhook "billing" in current consumer subscribed to "alerts.invoice.paid"
* Create alert config "ops" with email "ops@example.com" for events "sparrow.webhook.health_changed"
* Create alert config "dev" for webhook "billing" with email "dev@example.com" for events "sparrow.webhook.delivery_failed"
* Alert configs list should have "2" items
* Alert configs list for webhook "billing" should have "1" items
* Delete alert config "ops"
* Alert configs list should have "1" items
* Delete alert config with id "00000000-0000-4000-8000-000000000009" should return status "404"

## Scoping An Alert To A Missing Webhook Is Rejected
* Use consumer "alerts-missing"
* Create alert config for webhook id "00000000-0000-4000-8000-00000000000a" with email "x@example.com" for events "sparrow.webhook.health_changed" expecting status "404"
