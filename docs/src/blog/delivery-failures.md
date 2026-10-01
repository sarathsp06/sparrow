---
title: Knowing when to knock again
kicker: On retries and failure
author: Sparrow team
description: How Sparrow classifies webhook delivery failures. A timeout, a 401 and a 503 look alike on a dashboard, but each one asks for a different response.
pubDate: 2026-09-24
tags: [webhooks, reliability]
---

Think about knocking on a friend's door.

If nobody answers, they might be in the shower. You wait a minute and knock again. If a voice says "not now, come back in an hour," you come back in an hour. If a stranger opens the door and says you have the wrong house, knocking harder will not help. You check the address.

Nobody has to teach us this. We read the silence, or the answer, and we adjust. Persistence is a virtue right up to the moment it becomes noise.

A webhook system has to learn the same manners, and most of them start out rude.

## Three red rows

You open the deliveries page on Monday morning and see three red rows. The first says **timeout**. The second says **401 Unauthorized**. The third says **503 Service Unavailable**.

A dashboard that only says "failed" treats these as the same problem. They are not. One will fix itself, one will never fix itself, and one is waiting for you. A retry policy that cannot tell them apart either keeps knocking on the wrong house or walks away from a friend who was only in the shower.

There is an old idea, from the Stoics, that peace starts with sorting the world into what you can change and what you cannot. Error classification is that idea written as code. Sparrow sorts each failure before it decides what to do next.

## The friend in the shower

The timeout means the receiver did not answer in time. Maybe it was deploying, maybe its database was slow. This kind of failure usually passes on its own.

Sparrow labels it `timeout`, which is retryable. The delivery goes back on the queue with exponential backoff, `retry_backoff_seconds * 2^(attempt - 1)`, capped at 24 hours. Each knock waits a little longer than the last. By the time you look, the second or third attempt has probably landed. Refused connections, dropped connections, and 5xx responses follow the same patient path.

You do nothing. The system already did the considerate thing.

## The wrong key

The 401 is different. The receiver answered, clearly, and said no. Someone rotated the receiver's credentials, and your webhook is still holding the old key.

Sparrow labels this `client_error`. Every 4xx response is permanent, so the delivery is marked `FAILED` at once, no matter how much retry budget is left. Knocking ten more times with the same wrong key would only fill the receiver's logs and teach them to ignore you.

This is the row that needs a person. Update the webhook's secret header, then retry the failed deliveries by ID. They go out with the new key and land.

A hostname that does not resolve (`dns_error`) or a certificate that does not verify (`tls_error`) is the wrong address. It will not fix itself between attempts, so Sparrow stops and tells you.

## "Come back later"

The 503 is a server error, so Sparrow retries it on backoff like the timeout. If the receiver stays down until the retry budget runs out, the delivery ends as `FAILED`. If the event's `ttl_seconds` passes first, it ends as `EXPIRED`, because some news is only worth delivering while it is still news. Either way the ending is explicit, and the last response is stored on the delivery.

A **429 Too Many Requests** is the receiver saying "slow down" out loud. Sparrow labels it `rate_limited`, waits for the `Retry-After` delay, and tries again, and that wait does not use up an attempt. Being asked to be patient should not cost you a chance.

## When the fault is ours

Sometimes the receiver did nothing wrong. If a subscription's transform template fails to render, nothing is sent, and the delivery is recorded as `template_error`. That failure never counts against the webhook's health, because blaming the receiver for our own mistake would be both wrong and unkind. Fix the template, retry the delivery, move on.

## What the red rows say now

Each delivery keeps its category and its last response together, so the dashboard can say more than "red":

- **timeout, 5xx, connection errors:** still knocking; usually no action needed.
- **4xx, DNS, TLS:** stopped on purpose; fix the configuration, then retry.
- **template_error:** our side; fix the template, then retry.

Three red rows become one item on your list. That is the whole point of classifying failure: not to make it disappear, but to make it say what it needs. The full table is in the [error classification reference](/sparrow/reference/error-classification/).
