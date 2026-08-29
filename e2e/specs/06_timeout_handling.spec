# Timeout Handling
Tags: timeout, retry

Shopify's endpoint takes 10s to respond but timeout is 2s. Both attempts time out.

## Slow Endpoint Is Classified As Timeout
* Create namespace "timeout"
* Start target "shopify" with behavior "slow_10s"
* Register event type "cart.abandoned"
* Register webhook "shopify" in current namespace subscribed to "cart.abandoned" with max_retries "1" and timeout "2"
* Push event "cart.abandoned" with payload "{\"cart_id\": \"cart-9921\", \"items\": 3}"
* Wait for all deliveries in current namespace to be terminal with count "1" within "60" seconds
* Delivery "0" should have status "failed"
* Delivery "0" should have error category "timeout"
* Target "shopify" should have received at least "1" deliveries
