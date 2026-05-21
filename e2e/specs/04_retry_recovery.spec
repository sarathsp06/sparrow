# Retry Recovery
Tags: retry, recovery

Google's endpoint fails once then recovers. Sparrow retries and delivers successfully.

## Retries Succeed After Target Recovers
* Create namespace "retry"
* Start target "google" with behavior "fail_then_succeed_1"
* Register event type "user.signup"
* Register webhook "google" in current namespace subscribed to "user.signup" with max_retries "3"
* Push event "user.signup" with payload "{\"user_id\": \"usr-8821\", \"email\": \"jane@example.com\"}"
* Wait for all deliveries in current namespace to be terminal with count "1" within "90" seconds
* Delivery "0" should have status "DELIVERY_SUCCESS"
* Delivery "0" should have attempt count "2"
* Target "google" should have received "2" deliveries
* Get delivery attempts for delivery "0" and verify count is "2"
