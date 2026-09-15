# CRUD Lifecycle -- Webhooks, Subscriptions, and Event Types
Tags: crud, lifecycle

An operator wires up a webhook, repoints it at a new endpoint, tunes its
subscription, and finally tears everything down.

## Webhook And Subscription Round-Trip From Register To Delete
* Use consumer "crud"
* Start target "primary"
* Start target "secondary"
* Register event type "crud.ticket.created"
* Register webhook "primary" in current consumer subscribed to "crud.ticket.created"
* GET webhook "primary" should match target "primary" and event "crud.ticket.created"
* Webhook "primary" should appear in the webhook list
* PATCH webhook "primary" url to target "secondary"
* Push event "crud.ticket.created" with payload "{\"ticket_id\": \"t-1\"}"
* Wait for "secondary" to receive "1" deliveries
* Target "primary" should have received "0" deliveries
* List subscriptions for webhook "primary" and save the first
* GET saved subscription should return status "200"
* PATCH saved subscription with template "{\"note\": \"ticket {{.payload.ticket_id}}\"}"
* Push event "crud.ticket.created" with payload "{\"ticket_id\": \"t-2\"}"
* Wait for "secondary" to receive "2" deliveries
* Latest delivery to "secondary" has body field "note" equal to "ticket t-2"
* DELETE saved subscription
* GET saved subscription should return status "404"
* Push event "crud.ticket.created" with payload "{\"ticket_id\": \"t-3\"}"
* Wait "3" seconds
* Target "secondary" should have received "2" deliveries
* DELETE webhook "primary"
* GET webhook "primary" should return status "404"

## Event Type Round-Trip From Register To Delete
* Use consumer "crud-events"
* Register event type "crud.report.ready"
* GET event type "crud.report.ready" should return status "200"
* PATCH event type "crud.report.ready" description to "Nightly report finished rendering"
* DELETE event type "crud.report.ready"
* GET event type "crud.report.ready" should return status "404"
