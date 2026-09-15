# Stats and Health
Tags: stats, health

Delivery statistics per consumer and globally, plus per-webhook computed
health, the cross-consumer health summary, and health-filtered webhook listing.

## Consumer Stats Reflect A Known Delivery Mix
One healthy endpoint and one broken endpoint each get the same event; stats count one success and one failure.

* Use consumer "stats-mix"
* Start target "mailchimp"
* Start target "billing" with behavior "status_404"
* Register event type "iso.stats.metric.recorded"
* Register webhook "mailchimp" in current consumer subscribed to "iso.stats.metric.recorded"
* Register webhook "billing" in current consumer subscribed to "iso.stats.metric.recorded" with max_retries "0"
* Push event "iso.stats.metric.recorded" with payload "{\"metric\": \"signups\", \"value\": 42}"
* Wait for all deliveries in current consumer to be terminal with count "2"
* Consumer stats should show "1" successful and "1" failed deliveries
* Consumer stats should show "2" total webhooks
* Global stats should include at least the consumer's counts

## Failing Webhook Is Reported Unhealthy
Five consecutive failures push a webhook to unhealthy; health endpoints and the health-filtered listing agree.

* Use consumer "health-check"
* Start target "legacy" with behavior "status_404"
* Register event type "iso.health.ping.sent"
* Register webhook "legacy" in current consumer subscribed to "iso.health.ping.sent" with max_retries "0"
* Push event "iso.health.ping.sent" with payload "{\"seq\": 1}"
* Push event "iso.health.ping.sent" with payload "{\"seq\": 2}"
* Push event "iso.health.ping.sent" with payload "{\"seq\": 3}"
* Push event "iso.health.ping.sent" with payload "{\"seq\": 4}"
* Push event "iso.health.ping.sent" with payload "{\"seq\": 5}"
* Wait for all deliveries in current consumer to be terminal with count "5"
* All terminal deliveries should have status "failed"
* Wait for webhook "legacy" health to become "unhealthy" within "30" seconds
* Webhook health should report "5" failed deliveries
* Health summary should count at least "1" "unhealthy" webhooks
* Webhook "legacy" should be listed under health "unhealthy"
