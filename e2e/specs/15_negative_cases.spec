# Negative Cases -- Unknown Resources, Empty Fan-Out, Pagination
Tags: negative

What happens off the happy path: pushing types nobody registered or
subscribes to, fetching ids that do not exist, and paging through
delivery history.

## Pushing An Unregistered Event Type Auto-Registers It And Succeeds
* Use consumer "neg-unregistered"
* Push event "neg.never.registered" with payload "{\"k\": \"v\"}" expecting status "201"
* GET event type "neg.never.registered" should return status "200"

## Push With No Subscribers Succeeds But Creates No Deliveries
* Use consumer "neg-nosubs"
* Register event type "neg.audit.logged"
* Push event "neg.audit.logged" with payload "{\"actor\": \"root\"}" expecting status "201"
* Wait "3" seconds
* API should show "0" deliveries in current consumer

## Unknown Ids Return 404
* Use consumer "neg-unknown"
* GET consumer path "/webhooks/00000000-0000-4000-8000-000000000001" should return status "404"
* GET consumer path "/deliveries/00000000-0000-4000-8000-000000000002" should return status "404"
* GET "/v1/events/00000000-0000-4000-8000-000000000003" should return status "404"
* GET "/v1/event-types/neg.no.such.type" should return status "404"

## Delivery List Pages With Limit And Offset
* Use consumer "neg-paging"
* Start target "collector"
* Register event type "neg.metric.emitted"
* Register webhook "collector" in current consumer subscribed to "neg.metric.emitted"
* Push event "neg.metric.emitted" with payload "{\"n\": 1}"
* Push event "neg.metric.emitted" with payload "{\"n\": 2}"
* Push event "neg.metric.emitted" with payload "{\"n\": 3}"
* Wait for "collector" to receive "3" deliveries
* List deliveries with limit "2" and offset "0" should return "2" items
* Last delivery page should have total_count "3" and has_more "true"
* List deliveries with limit "2" and offset "2" should return "1" items
* Last delivery page should have total_count "3" and has_more "false"
* Paging through deliveries with limit "2" should yield "3" unique deliveries
