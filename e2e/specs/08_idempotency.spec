# Idempotency
Tags: idempotency, dedup

Duplicate pushes with the same idempotency key deliver only once.

## Duplicate Push With Idempotency Key Delivers Once
* Create namespace "idemp"
* Start target "stripe"
* Register event type "charge.completed"
* Register webhook "stripe" in current namespace subscribed to "charge.completed"
* Push event "charge.completed" with payload "{\"amount\": 99.99, \"currency\": \"USD\"}" and idempotency key "charge-xyz-001"
* Save event ID as first
* Last push response should not be duplicate
* Push event "charge.completed" with payload "{\"amount\": 99.99, \"currency\": \"USD\"}" and idempotency key "charge-xyz-001"
* Last push response should be duplicate
* Last push response event ID should match previous
* Wait for "stripe" to receive "1" deliveries
* Wait "3" seconds
* Target "stripe" should have received "1" deliveries
* API should show "1" deliveries in current namespace

## Push Without Idempotency Key Creates Separate Events
* Create namespace "no-idemp"
* Start target "stripe2"
* Register event type "charge.completed.v2"
* Register webhook "stripe2" in current namespace subscribed to "charge.completed.v2"
* Push event "charge.completed.v2" with payload "{\"amount\": 50.00}"
* Push event "charge.completed.v2" with payload "{\"amount\": 50.00}"
* Wait for "stripe2" to receive "2" deliveries
* Target "stripe2" should have received "2" deliveries
