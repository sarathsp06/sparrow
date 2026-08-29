# Permanent Failure
Tags: failure

## Client Error Is Not Retried
Facebook returns 404 -- no retries, immediate failure.

* Create namespace "perm-fail-a"
* Start target "facebook" with behavior "status_404"
* Register event type "payment.refunded"
* Register webhook "facebook" in current namespace subscribed to "payment.refunded" with max_retries "3"
* Push event "payment.refunded" with payload "{\"payment_id\": \"pay-112\", \"amount\": 25.00}"
* Wait for all deliveries in current namespace to be terminal with count "1"
* Delivery "0" should have status "failed"
* Delivery "0" should have error category "client_error"
* Delivery "0" should have attempt count "1"
* Target "facebook" should have received "1" deliveries

## Server Error Exhausts All Retries
Microsoft returns 500 forever -- retries exhaust then fails.

* Create namespace "perm-fail-b"
* Start target "microsoft" with behavior "status_500"
* Register event type "payment.refunded.v2"
* Register webhook "microsoft" in current namespace subscribed to "payment.refunded.v2" with max_retries "2"
* Push event "payment.refunded.v2" with payload "{\"payment_id\": \"pay-113\", \"amount\": 50.00}"
* Wait for all deliveries in current namespace to be terminal with count "1" within "60" seconds
* Delivery "0" should have status "failed"
* Delivery "0" should have error category "server_error"
* Delivery "0" should have attempt count "3"
* Target "microsoft" should have received "3" deliveries
