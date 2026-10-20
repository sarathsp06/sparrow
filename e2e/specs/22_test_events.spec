# Test Events
Tags: test-event

A test event reaches the receiver signed like a real delivery, marked as a
test and with the routing headers, and leaves no delivery behind. The
service and handler are unit-tested; this proves the wire path.

## Send A Test Event
* Use consumer "test-event"
* Start target "app"
* Register event type "order.refunded"
* Register webhook "app" in current consumer subscribed to "order.refunded"
* Send a test event to webhook "app"
* Test event result should have status code "200"
* Target "app" should have received "1" deliveries
* Latest delivery to "app" has header "X-Sparrow-Test" equal to "true"
* Latest delivery to "app" has header "X-Sparrow-Event-Type" equal to "order.refunded"
* Latest delivery to "app" has a valid HMAC signature
* Webhook "app" should have no recorded deliveries
* Sending another test event to webhook "app" right away should return status "429"
