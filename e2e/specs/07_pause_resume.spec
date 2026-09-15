# Pause and Resume
Tags: pause, resume, lifecycle

Paused webhooks don't receive events. After resume, new events are delivered.

## Paused Webhook Does Not Receive Events
* Use consumer "pause"
* Start target "twilio"
* Register event type "sms.received"
* Register webhook "twilio" in current consumer subscribed to "sms.received"
* Push event "sms.received" with payload "{\"from\": \"+1555000111\", \"body\": \"Hello\"}"
* Wait for "twilio" to receive "1" deliveries
* Pause webhook "twilio"
* Push event "sms.received" with payload "{\"from\": \"+1555000222\", \"body\": \"During maintenance\"}"
* Wait "5" seconds
* Target "twilio" should have received "1" deliveries
* Resume webhook "twilio"
* Push event "sms.received" with payload "{\"from\": \"+1555000333\", \"body\": \"After maintenance\"}"
* Wait for "twilio" to receive "2" deliveries
