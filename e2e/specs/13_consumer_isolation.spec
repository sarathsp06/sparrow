# Consumer Isolation -- Tenants Never See Each Other's Traffic
Tags: isolation, multitenancy

Acme and Globex are separate consumers. An event pushed into Globex must
never reach Acme's webhooks, and Acme's resources are invisible from
Globex's consumer path.

## Event Pushed To Another Consumer Produces No Deliveries Here
* Use consumer "iso-acme"
* Also use consumer "iso-globex" as "globex"
* Start target "acme-billing"
* Register event type "iso.order.created"
* Register webhook "acme-billing" in current consumer subscribed to "iso.order.created"
* Push event "iso.order.created" with payload "{\"order_id\": \"ord-1\"}" to consumer "globex"
* Target "acme-billing" should have received no deliveries
* API should show "0" deliveries in current consumer

## Webhook Is Not Visible Through Another Consumer's Path
* Use consumer "iso-acme2"
* Also use consumer "iso-globex2" as "globex"
* Start target "acme-billing"
* Register event type "iso.invoice.paid"
* Register webhook "acme-billing" in current consumer subscribed to "iso.invoice.paid"
* GET webhook "acme-billing" via consumer "globex" should return status "404"
