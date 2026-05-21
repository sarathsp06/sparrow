# Manual Retry and Event Replay
Tags: retry, manual, replay

## Retry Failed Delivery After Fixing Endpoint
Facebook fails with 404, gets fixed, manual retry succeeds.

* Create namespace "retry-replay"
* Start target "facebook" with behavior "status_404"
* Start target "github"
* Register event type "invoice.sent"
* Register webhook "facebook" in current namespace subscribed to "invoice.sent" with max_retries "0"
* Register webhook "github" in current namespace subscribed to "invoice.sent"
* Push event "invoice.sent" with payload "{\"invoice_id\": \"inv-2024-099\", \"customer\": \"Acme Corp\", \"total\": 15000.00}"
* Wait for all deliveries in current namespace to be terminal with count "2"
* Switch target "facebook" to behavior "ok"
* Retry the failed delivery
* Retried delivery should have status "DELIVERY_SUCCESS"
* Target "facebook" should have received "2" deliveries

## Replay Event Reaches New Subscribers
* Create namespace "replay"
* Start target "github2"
* Register event type "invoice.sent.v2"
* Register webhook "github2" in current namespace subscribed to "invoice.sent.v2"
* Push event "invoice.sent.v2" with payload "{\"invoice_id\": \"inv-2024-100\", \"customer\": \"Beta Corp\"}"
* Wait for "github2" to receive "1" deliveries
* Start target "zendesk"
* Register webhook "zendesk" in current namespace subscribed to "invoice.sent.v2"
* Replay the last pushed event
* Wait for "github2" to receive "2" deliveries
* Wait for "zendesk" to receive "1" deliveries
* Target "zendesk" should have received "1" deliveries
