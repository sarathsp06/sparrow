---
title: The benchmark that measured nothing
kicker: On load-testing a webhook pipeline
author: Sparrow team
description: We dusted off Sparrow's load-test tool, found it had been reporting zeros, and then let it find four throughput walls, three of them in code we thought was fine. The numbers, the fixes, and how to run it yourself.
pubDate: 2026-10-07
tags: [performance, benchmark, architecture, river]
---

Every project has a benchmark somebody wrote with great care and nobody has run since. Ours lives in `cmd/benchmark`. It has a token-bucket pacer, reservoir sampling for latencies, a resource monitor, and a section at the end that extrapolates how many CPU cores you would need at 50,000 requests per second.

We ran it last week. It printed this:

```text
Total Requests:  1010
Successful:      0 (0.00%)
Failed:          1010 (100.00%)
P50:             0s
P99:             0s
```

and then, without missing a beat, estimated production resource needs for "Peak Load (50000 RPS)".

This post is about what happened after we made it work. The short version: the tool found four throughput ceilings, and three of them were real bugs in Sparrow that had nothing to do with the benchmark being broken.

## Wall zero: a benchmark that cannot fail loudly

The hundred-percent failure was easy. Sparrow's delivery client refuses to connect to loopback and private addresses unless you opt in (that is the SSRF guard, added in October). The benchmark spins up an in-process HTTP server on 127.0.0.1 and never opted in. Every `Send` failed before a packet left the process.

The interesting part is not the bug. It is that the tool swallowed the error, skipped the latency sample on failure, and reported `0s` percentiles as if that were a measurement. A benchmark that can print zeros and a capacity plan in the same breath is worse than no benchmark. The first change was the dumbest one: log the first failure, and count TCP connections opened at the test server. That second number turned out to matter.

## Wall one: every delivery opened a new TCP connection

With the SSRF opt-in fixed, the client benchmark ran. At 100 requests per second it looked fine. At 1,000 per second, 22% of requests failed with:

```text
dial tcp 127.0.0.1:64727: connect: can't assign requested address
```

That is ephemeral port exhaustion. macOS gives you about 16,000 ephemeral ports and holds a closed socket in `TIME_WAIT` for 30 seconds, so you only see this if you open roughly 500 new connections a second for half a minute. A keep-alive HTTP client talking to one host should open about one connection per worker and reuse it forever.

We ran a control with two workers. After 20 seconds the machine had 16,359 sockets in `TIME_WAIT`. The run had 16,359 successful requests. Every single request had opened its own connection.

The transport was configured correctly. Keep-alive was on. The cause was five lines in `Send`:

```go
if req.Timeout > 0 {
    var cancel context.CancelFunc
    ctx, cancel = context.WithTimeout(ctx, req.Timeout)
    defer cancel()
    httpReq = httpReq.WithContext(ctx)
}
```

`Send` returns the response and lets the caller read the body. The deferred `cancel()` runs when `Send` returns, which is before the body has been read. Go's `net/http` transport watches the request context while the body is still open, and if the context is cancelled before the body reaches EOF it marks the connection dead and closes it. It has to: it cannot know whether the rest of the body is still coming.

So every delivery with a per-request timeout paid a fresh TCP handshake, and for HTTPS receivers a fresh TLS handshake too. Sparrow always sets a per-request timeout (the webhook's `request_timeout_seconds`, default 30). In production, this has been true for every delivery ever sent.

The fix ties the cancel to `Body.Close()` instead of to `Send` returning, plus one more line: `MaxIdleConnsPerHost` was never set, so Go's default of 2 applied, which would have closed every connection beyond the second one once we had many workers hitting one receiver. There is now a test that sends 20 requests through the client with a timeout set and asserts the server saw exactly one connection. It fails on the old code with 20.

| Client benchmark, 20 s, 10 KB payload | Before: success / p50 / p99 | After: success / p50 / p99 |
|---|---|---|
| 100 rps, 10 workers | 99.9% / 1.39 ms / 3.0 ms | 99.9% / 425 µs / 1.0 ms |
| 1,000 rps, 50 workers | 77.8% / 424 µs / 4.2 ms | 100% / 216 µs / 939 µs |
| 5,000 rps, 200 workers | 35% / 122 µs / 32 ms | 100% / 55 µs / 133 µs |
| 20,000 rps, 500 workers | 44% / 239 µs / 7.2 s | 100% / 48 µs / 274 µs |

Those are numbers for the sending client alone, against a loopback server that answers in microseconds. They say nothing about Sparrow's pipeline. Which brings us to the part the old tool never measured.

## Measuring the thing that matters

The question a webhook server has to answer is not "how fast can you POST?" It is: an event arrives at the API, how long until the receiver has it, and how many of those per second before the backlog starts growing?

So the benchmark got a second mode. `-mode e2e` starts a local HTTP receiver, registers a webhook pointing at it through the REST API, then publishes events at a paced rate. Every delivery carries an `X-Sparrow-Event-ID` header, so the receiver can match it back to the moment it was published. The run ends when every published event has arrived, or when a drain timeout gives up. It reports ingest latency (`POST /events` to `201`), end-to-end latency (publish to receipt), delivery throughput while publishing, peak backlog, duplicates, and a per-second timeline of accepted versus delivered.

The whole path is exercised: API handler, event insert, River fan-out job, delivery job, the HTTP client from the previous section, delivery status writes, health bookkeeping.

Setup for everything below: an Apple M3 Pro laptop, Postgres 15 in Docker on the same machine, 1 KB payloads, receiver answering in under a millisecond. These are not production numbers. They are the shape of the curve.

## Wall two: 200 deliveries per second, flat

| Publish rate | Delivered/s while publishing | p50 end-to-end | Peak backlog |
|---|---|---|---|
| 200/s | 195 | 322 ms | 153 |
| 500/s | 197 | 23.1 s | 9,129 |
| 1,000/s | 176 | 62 s | 24,815 |
| 2,000/s | 145 | 73 s | 55,821 |

Delivery throughput did not move. Server CPU sat between 20% and 50%, Postgres under 35%. Something was pacing the workers, and it was not the hardware.

Sparrow runs its queues on [River](https://riverqueue.com). Each queue has a worker pool (20 for event fan-out, 20 for webhook delivery) and a producer that fetches jobs for idle workers. The producer has a `FetchCooldown`: it will not query for new jobs more than once per cooldown. River's default is 100 ms. The doc comment on the field says, in so many words, "Throughput is limited by this value."

Twenty workers, one fetch per 100 ms, each fetch takes as many jobs as there are idle workers. If a job finishes in less than 100 ms, which a loopback delivery does, the worker then sits idle until the next fetch. Twenty jobs per 100 ms is 200 per second. Per queue. Both the fan-out queue and the delivery queue had the same cap, which is why the ceiling was so clean.

Sparrow never set `FetchCooldown` and had no knob for worker counts. Now it has three: `SPARROW_EVENT_WORKERS`, `SPARROW_WEBHOOK_WORKERS`, and `SPARROW_QUEUE_FETCH_COOLDOWN`, with the cooldown defaulting to 20 ms. The ceiling becomes `workers / cooldown`, about 1,000 jobs per second per queue with the default pools, and you can raise either number.

| Publish rate | Delivered/s while publishing (before → after) | p50 end-to-end (before → after) | Peak backlog (before → after) |
|---|---|---|---|
| 200/s | 195 → 199 | 322 ms → 41 ms | 153 → 38 |
| 500/s | 197 → 458 | 23.1 s → 1.1 s | 9,129 → 1,009 |
| 1,000/s | 176 → 438 | 62 s → 19 s | 24,815 → 16,936 |

Better, obviously. But look at the second column again. It stopped at 440.

## Wall three: what a delivery costs

Four hundred and forty per second, with CPU under 20% and Postgres nearly idle. The slow-receiver run gave the first clue. With the receiver sleeping 50 ms per request, throughput was 181 per second. Twenty workers divided by 181 is 110 ms per delivery; take away the 50 ms sleep and about 60 ms is Sparrow's own work. Twenty workers divided by 440 per second with a fast receiver is 45 ms. Each delivery job spends tens of milliseconds doing something, and it is not CPU.

Sampling `pg_stat_activity` during a run showed where:

```text
Lock:transactionid | 10 | INSERT INTO webhook_health_state (webhook_id, consecutive_failures, ...
LWLock:WALWrite    |  2 | INSERT INTO webhook_health_events (webhook_id, delivery_id, ...
IO:WALSync         |  1 | UPDATE webhook_registrations SET health = $1 ... WHERE id = $2
CPU                |  2 | SELECT COUNT(DISTINCT delivery_id), ... FROM webhook_health_events
                   |    |   WHERE webhook_id = $1 AND timestamp >= NOW() - INTERVAL '1 hour' * $2
```

Two things are going on.

**Every delivery is several small transactions.** The worker loads the webhook, the event and the subscription, takes a rate-limit slot, stores the request body, writes the delivery status, inserts a health event, upserts the health state, recomputes the health label, and writes it back to the registration. None of this is wrapped in a transaction: each statement autocommits on its own, and each autocommit is a WAL fsync. On a laptop with Postgres in Docker an fsync is a few milliseconds, so half a dozen of them per delivery is most of the 30 to 45 ms. Faster disks shrink it; the number of round trips does not change.

**One busy receiver serializes its workers.** Two of those writes go to rows that belong to the webhook: the upsert of its `webhook_health_state` row and the `UPDATE webhook_registrations SET health = ...` on its registration row, which runs on every delivery whether or not the label changed. A row lock in Postgres is held until the transaction commits, and with `synchronous_commit = on` the commit waits for the fsync. So with one hot webhook, twenty workers take turns holding a row through an fsync, and throughput becomes roughly one over commit latency: about 440 per second here, no matter how many workers you add. The `Lock:transactionid` waits in the sample above are exactly that queue.

**The health label gets more expensive as the webhook gets busier.** The label is recomputed on every delivery with a `COUNT(DISTINCT delivery_id)` over that webhook's health events from the last 24 hours. It does not hold the row lock (it is its own statement), but it is paid on every delivery and it grows with the number of recent deliveries. That is why a 10-second run at 500 per second kept up and a 30-second run did not: the first 5,000 rows are cheap to count, the next 15,000 less so, and after an hour of real traffic to a busy receiver it is a scan of every delivery that hour, per delivery.

Spreading the same traffic across receivers separates the two effects:

| Publish rate | Webhooks | Delivered/s while publishing | p50 end-to-end | Peak backlog |
|---|---|---|---|---|
| 1,000/s | 1 | 438 | 19 s | 16,936 |
| 1,000/s | 10 | 601 | 9.2 s | 12,056 |
| 2,000/s | 50 | 464 (661 over the whole run) | 42 s | 45,579 |
| 500/s, receiver sleeps 50 ms | 1 | 181 | 25.9 s | 9,627 |
| 500/s, receiver sleeps 50 ms | 10 | 196 | 23.5 s | 9,159 |

Ten receivers instead of one buys 40% more throughput, which is the hot-row share. The rest is the per-delivery cost, and with a slow receiver the receiver count makes no difference at all: twenty workers times about 100 ms per delivery is 200 per second, full stop. That one is not a bug, it is arithmetic, and it is what the new `SPARROW_WEBHOOK_WORKERS` knob is for. Same slow receiver, 100 workers instead of 20:

| Publish rate | Webhooks | Workers | Delivered/s while publishing | p50 end-to-end | Peak backlog |
|---|---|---|---|---|---|
| 500/s, receiver sleeps 50 ms | 10 | 20 | 196 | 23.5 s | 9,159 |
| 500/s, receiver sleeps 50 ms | 10 | 100 | 198 (see below) | 23.3 s | 9,113 |
| 1,000/s | 10 | 100 | 938 | 222 ms | 1,936 |
| 1,000/s | 1 | 100 | 434 | 19.4 s | 16,780 |

Except that the slow-receiver row did not move. Five times the workers, the same 198 per second. The ten webhooks in that run all point at the same host, and the delivery client's HTTP transport had `MaxConnsPerHost` at the library default of 10. Ten connections to one host, 50 ms each, 200 per second. A fourth wall, hiding behind the third, in a value nobody had looked at since the client was written. It now scales with the worker count; the per-webhook `rate_limit_rps` is the intended way to protect a receiver, not an accidental connection cap.

| Publish rate | Webhooks | Workers | Per-host connections | Delivered/s while publishing | p50 end-to-end |
|---|---|---|---|---|---|
| 500/s, receiver sleeps 50 ms | 10 | 100 | 10 | 198 | 23.3 s |
| 500/s, receiver sleeps 50 ms | 10 | 100 | 100 | 500 | 34 ms |

So we changed the bookkeeping. One repository call now records a delivery outcome in a single transaction: the health event is inserted first (append-only, nothing to contend on), then the state row is upserted, and the registration is written only when the label actually changed. The 24-hour recomputation runs at most every five seconds per webhook while outcomes stay the same, and immediately when they flip (a failure after successes, a success after failures, or the fifth consecutive failure), so the label still moves when it matters. Same benchmark, same defaults, before and after:

| Shape, 1,000 events/s for 30 s, 20 workers | Delivered/s while publishing | p50 publish → delivered | Drain |
|---|---|---|---|
| 1 webhook | 438 → 535 | 19 s → 13 s | 44 s → 19 s |
| 10 webhooks | 601 → 862 | 9.2 s → 1.5 s | 22 s → 5.9 s |
| 300 webhooks | 457 → 861 | 17.8 s → 1.7 s | 19.6 s → 4.7 s |

The many-webhook shapes now run at about 860 deliveries per second on 20 workers, where before they needed 100 workers to get near that. The single hot webhook still has one webhook-owned row commit per delivery (the state upsert), so it stays bounded by commit latency; spreading a very busy integration over two webhooks, or batching that upsert, would be the next step if anyone needs more than about 500 per second to one receiver.

## The realistic shape

Ten subscribers on one event type is a stress shape, not a typical one. In practice an event type has one subscriber, sometimes two, and what a webhook server actually faces is many webhooks registered and bursts of events across many types. So the last runs look like that, at the shipped defaults (20 fan-out workers, 20 delivery workers, 20 ms cooldown): 300 event types, each with its own receiver URL, and one or two webhooks per type.

| Shape | Events in | Delivered/s (events × subscribers) | p50 publish → fully delivered | Drain after publishing stopped |
|---|---|---|---|---|
| 300 types × 1 subscriber, paced 1,000/s for 30 s | 1,003/s | 457 → 607/s whole run | 17.8 s | 19.6 s |
| 300 types × 2 subscribers, paced 500/s for 30 s | 502/s | 425 events/s = 850 deliveries/s | 3.2 s | 5.1 s |
| 300 types × 1, burst of 20,000 events | 3,616/s accepted in 5.5 s | 749/s | 13.6 s | 21.2 s |
| 300 types × 2, burst of 20,000 events | 4,013/s accepted in 5.0 s | 406 events/s = 811 deliveries/s | 24.2 s | 44.3 s |

Three things to read off that table. Ingest is fast: a burst of 20,000 events is accepted in five seconds, about 4,000 per second with a 20 ms median, and nothing is rejected. Delivery at the defaults runs at 600 to 850 per second once the hot-row problem is spread across 300 webhooks, and the fan-out to two subscribers costs almost nothing extra per event. And while a burst is being accepted, delivery slows (148 per second during the five-second burst, 749 after), because ingest and delivery share the same Postgres; the backlog then drains at the delivery rate. Those numbers were taken before the bookkeeping change in the previous section; with it, the same 300-webhook shape delivers about 860 per second on the default 20 workers (table above), and `SPARROW_WEBHOOK_WORKERS` scales it from there until the remaining per-delivery commits are the limit.

## What the numbers mean for you

- **Upgrade for the client fix** even if you never push 200 events a second. Every HTTPS delivery you have ever sent paid a TLS handshake it did not need.
- **If deliveries lag under load with idle CPU**, check `SPARROW_QUEUE_FETCH_COOLDOWN` and the worker counts. The ceiling is `workers / cooldown` per queue; the new defaults give about 1,000 per second.
- **If your receivers are slow**, raise `SPARROW_WEBHOOK_WORKERS`. Throughput to a slow host is `workers / receiver latency`, now that the connection pool follows the worker count. Use the webhook's `rate_limit_rps` to protect a receiver, not the worker count.
- **If one receiver takes most of your traffic**, expect around 500 deliveries per second to that one webhook on commodity disks: its health state row is still committed once per delivery. Deliveries to other webhooks are unaffected, and splitting a very busy integration across two webhooks doubles its ceiling.
- **Ingest is not the bottleneck.** `POST /events` answered in 5 to 8 ms at p50 across every run, up to 2,000 per second, and never rejected an event. Sparrow accepts first and delivers from a durable queue, so a delivery backlog never turns into lost events; it turns into latency you can see on the dashboard.

## Run it yourself

The client microbenchmark needs nothing:

```bash
go run ./cmd/benchmark -mode client -duration 30s -rps 1000 -concurrency 50 -payload 10
```

Watch the "TCP connections" line. It should be close to the worker count, not the request count.

The pipeline benchmark needs a running Sparrow that is allowed to deliver to loopback:

```bash
SPARROW_ALLOW_PRIVATE_NETWORKS=true SPARROW_AUTO_REGISTER_EVENTS=true make run
```

```bash
go run ./cmd/benchmark -mode e2e -sparrow-url http://localhost:8080 \
  -duration 30s -rps 500 -concurrency 50 -webhooks 300 -subscribers 2 -json results.json
```

`-webhooks` is the number of event types (each with its own receiver URL), `-subscribers` the webhooks per event type, and `-burst 20000` publishes that many events as fast as possible instead of pacing, then times the drain.

Each run registers its own event types and webhooks and removes them afterwards. If a run times out with a backlog, the leftover jobs will still be processed; the next run reports them as "foreign" deliveries rather than counting them. The JSON report includes the per-second backlog timeline if you want to plot it.

A benchmark earns its keep by failing in interesting ways. Ours had been failing in the least interesting way possible, silently, for months. Four ceilings later, we are glad we ran it.
