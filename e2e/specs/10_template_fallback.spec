# Template Fallback
Tags: template, fallback, graceful_degradation

A broken Go template doesn't drop the delivery. PagerDuty receives the envelope
payload as fallback.

## Broken Template Falls Back To Envelope Payload
* Create namespace "fallback"
* Start target "pagerduty"
* Register event type "deploy.failed"
* Register webhook "pagerduty" in current namespace with no subscriptions
* Subscribe webhook "pagerduty" to "deploy.failed" with broken template "{{index .payload.nonexistent \"key\"}}"
* Push event "deploy.failed" with payload "{\"service\": \"api-gateway\", \"version\": \"v2.3.1\", \"error\": \"health check timeout\"}"
* Wait for "pagerduty" to receive "1" deliveries
* Latest delivery to "pagerduty" body contains key "version"
* Latest delivery to "pagerduty" body contains key "event_name"
* Latest delivery to "pagerduty" body contains key "payload"
* Wait for all deliveries in current namespace to be terminal with count "1"
* Delivery "0" should have status "success"
