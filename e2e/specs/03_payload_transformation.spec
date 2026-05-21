# Payload Transformation via Template
Tags: template, transformation

Slack expects a specific JSON format. A Go template transforms the alert payload
into Slack's format before delivery.

## Template Transforms Payload Before Delivery
* Create namespace "transform"
* Start target "slack"
* Register event type "alert.fired"
* Register webhook "slack" in current namespace with no subscriptions
* Subscribe webhook "slack" to "alert.fired" with template "{\"text\": \"Alert: {{.payload.title}} - severity {{.payload.severity}}\"}"
* Push event "alert.fired" with payload "{\"title\": \"CPU High on prod-api-3\", \"severity\": \"critical\", \"host\": \"prod-api-3\"}"
* Wait for "slack" to receive "1" deliveries
* Latest delivery to "slack" has body field "text" equal to "Alert: CPU High on prod-api-3 - severity critical"
* Latest delivery to "slack" body does not contain key "version"
* Latest delivery to "slack" body does not contain key "event_name"
