# Receiver Code and Test Events
Tags: receiver-code

A consumer generates a receiver project for its own webhook and sends it a
test event. Generation details are unit-tested in pkg/receivercode; this
proves the wire path through the API and the portal gateway, and that a test
send reaches the receiver signed, marked as a test, and leaves no delivery
behind.

## Generate Receiver Code For A Webhook
* Use consumer "receiver-code"
* Start target "app"
* Register event type "order.paid"
* Register webhook "app" in current consumer subscribed to "order.paid"
* Generate "python" "fastapi" receiver code for webhook "app"
* Generated receiver code should contain file "receiver.py"
* Generated receiver code should contain file "sparrow_verify.py"
* Generated receiver spec should list "1" subscriptions
* Generated receiver code should not contain the webhook secret of "app"

## Generate Receiver Code Through The Portal
* Use consumer "receiver-portal"
* Start target "app"
* Register event type "order.shipped"
* Register webhook "app" in current consumer subscribed to "order.shipped"
* Mint portal token for current consumer
* Portal generate "go" "nethttp" receiver code for webhook "app" should return status "200"
* Portal generate "cobol" "cics" receiver code for webhook "app" should return status "400"

## Send A Test Event
* Use consumer "receiver-test"
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
