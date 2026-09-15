# Batch Jobs
Tags: batch, jobs

Bulk retry and async batch jobs: retry a webhook's failed deliveries in one
call, run a snapshot-based batch retry job, and re-push events in bulk.

## Bulk Retry All Failed Deliveries For A Webhook
Stripe's endpoint 404s three orders, gets fixed, one bulk retry call recovers all of them.

* Use consumer "bulk-retry"
* Start target "stripe" with behavior "status_404"
* Register event type "iso.bulk.order.created"
* Register webhook "stripe" in current consumer subscribed to "iso.bulk.order.created" with max_retries "0"
* Push event "iso.bulk.order.created" with payload "{\"order_id\": \"ord-101\"}"
* Push event "iso.bulk.order.created" with payload "{\"order_id\": \"ord-102\"}"
* Push event "iso.bulk.order.created" with payload "{\"order_id\": \"ord-103\"}"
* Wait for all deliveries in current consumer to be terminal with count "3"
* All terminal deliveries should have status "failed"
* Switch target "stripe" to behavior "ok"
* Bulk retry deliveries for webhook "stripe"
* Bulk retry should have retried "3" deliveries
* Wait for "stripe" to receive "6" deliveries
* Wait for all deliveries in current consumer to be terminal with count "3"
* All terminal deliveries should have status "success"

## Batch Retry Job Recovers A Prepared Snapshot Of Failures
Snapshot the failed deliveries, start an async retry job, and poll it to completion.

* Use consumer "batch-retry-job"
* Start target "shopify" with behavior "status_404"
* Register event type "iso.batchjob.invoice.paid"
* Register webhook "shopify" in current consumer subscribed to "iso.batchjob.invoice.paid" with max_retries "0"
* Push event "iso.batchjob.invoice.paid" with payload "{\"invoice_id\": \"inv-201\"}"
* Push event "iso.batchjob.invoice.paid" with payload "{\"invoice_id\": \"inv-202\"}"
* Push event "iso.batchjob.invoice.paid" with payload "{\"invoice_id\": \"inv-203\"}"
* Wait for all deliveries in current consumer to be terminal with count "3"
* All terminal deliveries should have status "failed"
* Switch target "shopify" to behavior "ok"
* Prepare retry snapshot of failed deliveries
* Start batch retry job from snapshot
* Wait for batch job to complete within "60" seconds
* Batch job should show total "3" processed "3" failed "0"
* Cancelling the completed batch job should return status "409"
* Wait for "shopify" to receive "6" deliveries

## Batch Repush Job Redelivers Pushed Events
Snapshot the pushed occurrences, start an async re-push job, and the target receives doubles.

* Use consumer "repush-job"
* Start target "slack"
* Register event type "iso.repushjob.user.invited"
* Register webhook "slack" in current consumer subscribed to "iso.repushjob.user.invited"
* Push event "iso.repushjob.user.invited" with payload "{\"user_id\": \"usr-301\"}"
* Push event "iso.repushjob.user.invited" with payload "{\"user_id\": \"usr-302\"}"
* Wait for "slack" to receive "2" deliveries
* Prepare repush snapshot of pushed events
* Start batch repush job from snapshot
* Wait for batch job to complete within "60" seconds
* Batch job should show total "2" processed "2" failed "0"
* Cancelling the completed batch job should return status "409"
* Wait for "slack" to receive "4" deliveries
