# Selective Subscription
Tags: subscription, filtering

Only matching subscribers receive deliveries. Stripe subscribes to order.created,
FedEx to order.shipped, Datadog to both. When order.created fires, FedEx gets nothing.

## Only Matching Subscribers Receive Deliveries
* Create namespace "selective"
* Start target "stripe"
* Start target "fedex"
* Start target "datadog"
* Register event type "order.created"
* Register event type "order.shipped"
* Register webhook "stripe" in current namespace subscribed to "order.created"
* Register webhook "fedex" in current namespace subscribed to "order.shipped"
* Register webhook "datadog" in current namespace subscribed to "order.created, order.shipped"
* Push event "order.created" with payload "{\"order_id\": \"ord-501\", \"total\": 129.99}"
* Wait for "stripe" to receive "1" deliveries
* Wait for "datadog" to receive "1" deliveries
* Target "fedex" should have received no deliveries
* Target "stripe" should have received "1" deliveries
* Target "datadog" should have received "1" deliveries
* API should show "2" deliveries in current namespace
