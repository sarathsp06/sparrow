# Happy Path -- Fan Out to Multiple Subscribers
Tags: happy_path, fan_out, signatures

When an order is created, Stripe, Shippo, and Slack all need to know.
Push one event and all three receive it with valid signatures.

## One Event Fans Out To All Subscribers With Valid Signatures
* Create namespace "happy-path"
* Start target "stripe"
* Start target "shippo"
* Start target "slack"
* Register event type "order.created"
* Register webhook "stripe" in current namespace subscribed to "order.created"
* Register webhook "shippo" in current namespace subscribed to "order.created"
* Register webhook "slack" in current namespace subscribed to "order.created"
* Push event "order.created" with payload "{\"order_id\": \"ord-7741\", \"amount\": 42.0, \"customer\": \"acme-corp\"}"
* Wait for "stripe" to receive "1" deliveries
* Wait for "shippo" to receive "1" deliveries
* Wait for "slack" to receive "1" deliveries
* Target "stripe" should have received "1" deliveries
* Target "shippo" should have received "1" deliveries
* Target "slack" should have received "1" deliveries
* Latest delivery to "stripe" has signature headers
* API should show "3" deliveries in current namespace
