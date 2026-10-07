---
title: The benchmark that measured nothing
kicker: On finding four walls in a webhook pipeline
author: Sparrow team
description: We ran a load-test tool nobody had touched in months. It printed zeros and a capacity plan. Four ceilings, several embarrassing mistakes and one redesign later, a single busy webhook delivers four and a half times faster. The story, the numbers, and what we'd tell ourselves at the start.
pubDate: 2026-10-07
tags: [performance, benchmark, architecture, river, postgres]
---

Every project has a benchmark that someone wrote with great care and nobody has run since. Ours is `cmd/benchmark`. It has a token-bucket pacer, reservoir sampling for latencies, a resource monitor, and a closing section that extrapolates how many CPU cores you would need at 50,000 requests per second.

Last week we ran it. It printed this:

```text
Total Requests:  1010
Successful:      0 (0.00%)
Failed:          1010 (100.00%)
P50:             0s
P99:             0s
```

and then, without missing a beat, estimated the RAM and bandwidth for "Peak Load (50000 RPS)".

What follows is what happened after we made it work. It is a story about four throughput ceilings, but mostly it is a story about how easy it is to measure the wrong thing, trust the wrong signal, and feel good about it. The numbers at the end are real, and they are better than when we started, but the lessons are the part we want to keep.

## Day one: a tool that cannot fail loudly

The hundred-percent failure took ten minutes to find. Sparrow's delivery client refuses to connect to loopback and private addresses unless you opt in; that guard landed in October. The benchmark spins up an in-process HTTP server on 127.0.0.1 and never opted in. Every send failed before a packet left the process.

The bug was trivial. What stayed with us was that the tool swallowed the error, skipped the latency sample on failure, and reported `0s` percentiles as a measurement. Then it planned a data centre on top of them. A benchmark that can print zeros and a capacity plan in the same breath is worse than no benchmark, because it looks like evidence.

The first change we made was not a performance fix. It was to log the first failure, and to count TCP connections opened at the test server. That second number turned out to be the whole of day one.

## Wall one: every delivery opened a new connection

With the opt-in fixed, the client benchmark ran. At 100 requests per second it looked fine. At 1,000 per second, 22% of requests failed with:

```text
dial tcp 127.0.0.1:64727: connect: can't assign requested address
```

That is ephemeral port exhaustion. macOS hands out about 16,000 ephemeral ports and holds a closed socket in `TIME_WAIT` for 30 seconds, so you only see this if you open roughly 500 new connections a second for half a minute. A keep-alive HTTP client talking to one host should open about one connection per worker and reuse it forever.

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

`Send` returns the response and lets the caller read the body. The deferred `cancel()` runs when `Send` returns, which is before the body has been read. Go's transport watches the request context while the body is open, and if the context is cancelled before the body reaches EOF it marks the connection dead and closes it. It has to; it cannot know whether the rest of the body is still coming.

Sparrow always sets a per-request timeout, the webhook's `request_timeout_seconds`, 30 by default. So for as long as this code has existed, every delivery has paid a TCP handshake, and every HTTPS delivery a TLS handshake, for nothing. Nobody noticed because nothing failed. Deliveries were a few milliseconds slower than they needed to be, and receivers saw a new connection every time, and that is all.

The fix ties the cancel to `Body.Close()` instead of to `Send` returning. There is now a test that sends 20 requests through the client with a timeout set and asserts the server saw exactly one connection. It fails on the old code with 20.

| Client only, 20 s, 10 KB payload | Before: success / p50 | After: success / p50 |
|---|---|---|
| 1,000 rps, 50 workers | 77.8% / 424 µs | 100% / 216 µs |
| 5,000 rps, 200 workers | 35% / 122 µs | 100% / 55 µs |
| 20,000 rps, 500 workers | 44% / 239 µs | 100% / 48 µs |

Those are numbers for the sending client alone, against a loopback server that answers in microseconds. They say nothing about Sparrow as a whole. Which was the next uncomfortable realisation: the old benchmark could not say anything about Sparrow as a whole. It never touched the API, the database or the queue.

## Measuring the thing that matters

The question a webhook server has to answer is not "how fast can you POST?" It is: an event arrives, how long until the receiver has it, and how many of those per second before the backlog starts growing?

So the benchmark got a second mode. It starts a local HTTP receiver, registers webhooks pointing at it through the REST API, and publishes events at a paced rate. Every delivery carries an `X-Sparrow-Event-ID` header, so the receiver can match it back to the moment it was published. The run ends when every published event has arrived, or when a drain timeout gives up. It reports ingest latency, publish-to-receipt latency, delivery throughput, peak backlog, and a per-second timeline of accepted versus delivered.

Two details of the harness cost us a confusing hour and are worth passing on. First, Sparrow fans events out to webhooks at processing time, not at publish time. Events left in the queue by a run that timed out were fanned out to whichever webhook was active when they finally got processed, which was the *next* run's receiver. The next run reported more deliveries than it had published and no matching latencies. Every run now uses its own event type names, and deliveries the run did not publish are counted separately as "foreign". Second, identical runs on this laptop, minutes apart, differed by up to 45%. Any comparison below between builds comes from running them back to back on fresh databases, never from numbers taken on different days.

## Wall two: 200 deliveries per second, flat

The first pipeline runs produced a number that did not move.

| Publish rate | Delivered/s while publishing | p50 end-to-end | Peak backlog |
|---|---|---|---|
| 200/s | 195 | 322 ms | 153 |
| 500/s | 197 | 23 s | 9,129 |
| 1,000/s | 176 | 62 s | 24,815 |
| 2,000/s | 145 | 73 s | 55,821 |

Server CPU sat between 20% and 50%. Postgres under 35%. Something was pacing the workers, and it was not the hardware.

Sparrow runs its queues on [River](https://riverqueue.com). Each queue has a worker pool, 20 for event fan-out and 20 for webhook delivery, and a producer that fetches jobs for idle workers. The producer has a `FetchCooldown`: it will not query for new jobs more than once per cooldown. River's default is 100 ms. The doc comment on the field says, in so many words, "Throughput is limited by this value."

Twenty workers, one fetch per 100 ms, each fetch taking as many jobs as there are idle workers. If a job finishes in less than 100 ms, which a loopback delivery does, the worker then sits idle until the next fetch. Twenty jobs per 100 ms is 200 per second. Per queue. Both the fan-out queue and the delivery queue had the same cap, which is why the ceiling was so clean.

We had never set it. Sparrow now defaults the cooldown to 20 ms and exposes it, together with both worker counts, as `SPARROW_QUEUE_FETCH_COOLDOWN`, `SPARROW_EVENT_WORKERS` and `SPARROW_WEBHOOK_WORKERS`. The ceiling becomes workers divided by cooldown.

## Wall three: ten connections per receiver

The next ceiling only showed up when we made the receiver slow. With the receiver sleeping 50 ms per request, throughput was 198 per second, and raising the worker count from 20 to 100 did nothing at all. Five times the workers, the same 198.

The ten webhooks in that run all pointed at the same host, and the delivery client's transport had `MaxConnsPerHost` at the library default of 10. Ten connections to one host, 50 ms each, 200 per second. The server builds that client from `client.DefaultConfig()`, and nobody had looked at the number since the file was written. The per-host pool now follows the worker count; the per-webhook `rate_limit_rps` is the intended way to protect a receiver, not an accidental connection cap.

## Wall four: one busy webhook

Here is where the story stops being about defaults and becomes about design.

After the first three fixes, 300 webhooks sharing 1,000 events a second delivered about 800 per second. One webhook taking all 1,000 delivered about 440, and adding workers did not help: 100 workers, 434 per second. CPU was under 20%. Postgres was nearly idle. Where did the time go?

Sampling `pg_stat_activity` during a run answered it:

```text
Lock:transactionid | 10 | INSERT INTO webhook_health_state (...) ON CONFLICT ...
LWLock:WALWrite    |  2 | INSERT INTO webhook_health_events (...)
Lock:tuple         |  1 | UPDATE webhook_registrations SET health = $1 ...
CPU                |  2 | SELECT COUNT(DISTINCT delivery_id), ... FROM webhook_health_events
                   |    |   WHERE webhook_id = $1 AND timestamp >= NOW() - INTERVAL '24 hours'
```

After every delivery the worker recorded the outcome: insert a health event, upsert the webhook's health-state row, recompute the webhook's health label from its last 24 hours of events, and write the label back to the registration whether or not it changed. Each of those was its own autocommit. Two of them wrote a row that belongs to the webhook. A row lock in Postgres is held until the transaction commits, and with synchronous commit the commit waits for the WAL fsync. So twenty workers delivering to one webhook took turns holding a row through an fsync. Throughput to that webhook became roughly one over commit latency, a couple of milliseconds on this disk, and the number of workers stopped mattering.

The 24-hour recount made it worse over time. A ten-second run at 500 per second kept up. A thirty-second run did not, because the first 5,000 rows are cheap to count and the next 15,000 are not. After an hour of real traffic to a busy receiver, every single delivery would have been scanning an hour of rows.

We tried the small fix first: do the bookkeeping in one transaction, skip the registration write when the label is unchanged, recompute the label at most every five seconds. It helped, 440 to 535 per second, and it was not enough, because one webhook-owned row was still being committed once per delivery.

So we asked a different question: why does a delivery write a webhook's health at all?

### Deriving health instead of maintaining it

A delivery attempt now writes exactly one transaction: the delivery row (status, response, the request body that was sent) plus one append-only row in the health events table. Nothing it writes is shared with any other delivery to the same webhook. There is nothing to queue on.

Health state, labels, alerts and auto-disable are derived from that event log by a periodic job, once a minute by default. Each pass reads only the events since its watermark, folds them into per-minute counters per webhook, derives the failure run (consecutive failures and when it started) from the ordered outcomes, sums the last 24 hours of counters for the success rate, and rewrites a registration's label only when it changed. Everything, including advancing the watermark, is one transaction behind a locked single-row table, so a crash mid-pass replays cleanly and two instances cannot both evaluate.

A webhook taking ten thousand events a minute costs that pass one aggregate over ten thousand rows and one write. In our runs a pass took 66 ms on average and 189 ms at worst. Metrics remain the real-time signal; the health label is the reflective one, now at most a minute behind, which is what a health label is for.

The one semantic shift: the 24-hour success rate now counts attempts rather than distinct deliveries, so a delivery that failed once and succeeded on retry counts as one failure and one success instead of a success. A flaky receiver reads as degraded a little sooner. We think that is the more honest number for "how is this receiver doing", and it is what makes the window incremental.

### Three things that went wrong on the way

We would be lying if we said this went in cleanly.

The first version stored the watermark in a `system_settings` table. That table had been dropped by migration 19, months ago. The evaluator failed on every run, River retried it politely, and the integration tests that should have caught it did not, because of the second thing.

`make test-integration` pipes `go test` through `tail -40`. The exit code of a pipeline is the exit code of its last command. The suite had been "passing" with two failing tests, and we had reported it as green twice. Run directly, the health tests failed immediately and told us exactly what was wrong.

The third was River again. Insert a periodic job with `UniqueOpts{ByArgs: true}` and nothing else, and River's default unique states include *completed* jobs, which it keeps for 24 hours. The evaluator ran once at startup and was then treated as a duplicate of itself until the next day. Uniqueness is now scoped to active states. While fixing it we noticed our hourly cleanup and retention jobs are configured the same way, which means they have been running daily, not hourly. That one is tracked separately.

## The numbers, side by side

Three builds of the server, five workloads, run back to back on the same machine with a fresh database before each run. Main is Sparrow as it was before any of this. The middle build has the first three fixes. The final build adds the one-transaction delivery and the health evaluator. All three use the default 20 fan-out and 20 delivery workers. Deliveries per second while publishing, median time from publish to receiver, and how long the backlog took to drain after publishing stopped:

| Workload | main | walls 1–3 fixed | final |
|---|---|---|---|
| One busy webhook, 1,000 events/s | 193/s · 49 s · never drained | 395/s · 23 s · 50 s | **881/s · 1.4 s · 4 s** |
| 300 webhooks, 1,000 events/s | 177/s · 50 s · never drained | 406/s · 21 s · 35 s | 530/s · 13 s · 29 s |
| 300 webhooks × 2 subscribers, 500 events/s (1,000 deliveries/s) | 187/s · 50 s · never drained | 872/s · 2.6 s · 4.6 s | 899/s · 2.5 s · 3.3 s |
| 10 slow receivers (50 ms), 500 events/s | 139/s · 38 s · 75 s | 228/s · 19 s · 36 s | 295/s · 10 s · 28 s |
| Burst of 20,000 events to 600 webhooks (40,000 deliveries) | 195/s · 46 s · never drained | 757/s · 29 s · 48 s | 817/s · 25 s · 42 s |

Rates are deliveries per second while publishing (over the whole run for the burst); "never drained" means the 90-second drain limit expired with events still queued, so that median is a lower bound.

Two things to read off that table beyond the obvious. Main is pinned near 195 in every shape: that is wall two, and nothing else matters until it is gone. And in the middle build the busy webhook sits far below the fleet, while in the final build it leads it. That gap was wall four, and it is the one that would have hurt most in production, because the receiver that takes most of your traffic is the integration you care about most.

Ingest never flinched. `POST /events` answered in 4 to 13 milliseconds at the median under paced load in every build, in the low tens of milliseconds during the burst, a burst of 20,000 events was accepted in about five seconds every time, and nothing was ever rejected. Sparrow accepts first and delivers from a durable queue, so a slow stage shows up as latency on the dashboard, never as loss.

## What we would tell ourselves at the start

- **A benchmark that cannot fail loudly is a liability.** Ours printed zeros and a capacity plan. The first thing to add to any load tool is the first error message and a count of something that should be constant, like connections.
- **`defer cancel()` after returning a body is a connection leak.** If a function returns an `*http.Response`, the context that governs it must live until `Body.Close()`.
- **Read the defaults of your queue library.** River's `FetchCooldown` is documented as a throughput limit, in those words. We had never set it. Twenty workers at 100 ms is 200 jobs a second, however fast the jobs are.
- **Read the defaults of your HTTP transport too.** `MaxConnsPerHost: 10` is a fine default for a browser and a ceiling for a webhook sender.
- **Idle CPU with a growing backlog means a lock or a pacer.** `pg_stat_activity` with `wait_event_type` tells you which in under a minute.
- **Per-delivery writes to a per-webhook row serialize on commit latency.** Not on CPU, not on workers. Append per event; derive per period.
- **Hot paths should not maintain state they can derive.** The health label is read a few times a minute and was being written a thousand times a second.
- **Periodic jobs need an explicit uniqueness scope.** With River, `ByArgs` alone means "once a day".
- **Never pipe a test command through `tail`.** The exit code you are looking at belongs to `tail`.
- **Compare builds back to back or not at all.** Our same-binary variance was 45%. A number from Tuesday means nothing against a number from Friday.

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

`-webhooks` is the number of event types, each with its own receiver URL; `-subscribers` is the webhooks per event type; `-burst 20000` publishes that many events as fast as possible and times the drain; `-receiver-delay 50ms` makes the receiver slow. The JSON report includes the per-second backlog timeline if you want to plot it.

A benchmark earns its keep by failing in interesting ways. Ours had been failing in the least interesting way possible, silently, for months. Four walls later, we are glad we ran it.
