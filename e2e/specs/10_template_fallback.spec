# Template Errors
Tags: template, fallback, template_error

A template that cannot render is visible. By default the delivery fails with
error category template_error and nothing is sent; the subscription can opt
into sending the envelope payload instead with on_transform_error=fallback.

## Broken Template Fails The Delivery By Default
* Use consumer "template-fail"
* Start target "pagerduty"
* Register event type "deploy.failed"
* Register webhook "pagerduty" in current consumer with no subscriptions
* Subscribe webhook "pagerduty" to "deploy.failed" with broken template "{{index .payload.nonexistent \"key\"}}"
* Push event "deploy.failed" with payload "{\"service\": \"api-gateway\", \"version\": \"v2.3.1\", \"error\": \"health check timeout\"}"
* Wait for all deliveries in current consumer to be terminal with count "1"
* Delivery "0" should have status "failed"
* Delivery "0" should have error category "template_error"
* Target "pagerduty" should have received "0" deliveries

## Broken Template With Fallback Sends The Envelope Payload
* Use consumer "fallback"
* Start target "pagerduty"
* Register event type "deploy.failed"
* Register webhook "pagerduty" in current consumer with no subscriptions
* Subscribe webhook "pagerduty" to "deploy.failed" with broken template "{{index .payload.nonexistent \"key\"}}" and on_transform_error "fallback"
* Push event "deploy.failed" with payload "{\"service\": \"api-gateway\", \"version\": \"v2.3.1\", \"error\": \"health check timeout\"}"
* Wait for "pagerduty" to receive "1" deliveries
* Latest delivery to "pagerduty" body contains key "version"
* Latest delivery to "pagerduty" body contains key "event_name"
* Latest delivery to "pagerduty" body contains key "payload"
* Wait for all deliveries in current consumer to be terminal with count "1"
* Delivery "0" should have status "success"
