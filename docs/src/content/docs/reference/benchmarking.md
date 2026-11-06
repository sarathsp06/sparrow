---
title: Benchmarking
description: "How to measure Sparrow's throughput and latency with cmd/benchmark so that the numbers can be compared: open-loop load, repeated runs with confidence intervals, TLS, and a separate host."
---

`cmd/benchmark` load-tests Sparrow in two modes:

- **`-mode client`** drives the outbound delivery client against an in-process
  HTTP server. No database and no queue: it isolates the sending path.
- **`-mode e2e`** publishes events through the REST API of a running Sparrow and
  receives the deliveries on its own receiver. It measures the whole pipeline:
  API ingest, Postgres, River, the delivery worker, and HTTP.

A single run on a laptop tells you whether something works, not how fast it is.
Run-to-run noise on one machine has been 25-45% for the same build, which is
larger than many real changes. The tool is built so that a number it prints can
be compared, and so that it says when one cannot be.

## How it measures

**Open-loop load.** Requests (or events) are due on a fixed schedule, `-rps`
per second, whether or not earlier ones have finished. A worker that falls
behind takes the next slot at once, so a stall keeps the offered load where it
was instead of quietly lowering it.

**Two latencies.**

- *Response time* is measured from the moment a request was **due**. It includes
  any time it waited because the system was behind. Use it for p99 and p99.9.
- *Service time* is measured from the moment a request was actually **sent**.

When the two drift apart, requests are queueing in front of the system. Service
time alone hides that ("coordinated omission"): a 2-second stall leaves only one
slow sample instead of every request that should have gone out during it.

**Repeated runs.** `-runs N` repeats the whole measurement, with a fresh event
type and receiver each time and `-pause` idle time in between. The headline
metrics then get:

- a mean with a 95% confidence interval (Student's t);
- the median, min and max;
- the coefficient of variation (CV).

Any metric with a CV above 10% is listed as noisy. One run prints no interval
and says so.

**Environment.** Every report starts with the host, OS and architecture, CPU
count, `GOMAXPROCS`, Go version and the git revision of the benchmark binary.
Build it with `go build`, because `go run` does not record the revision. The
`-json` report holds the same environment, every run, and the summary.

## Running it

Start Postgres and a server with private networks allowed, so it may deliver to
the benchmark's local receiver:

```bash
make dev-db
```

```bash
SPARROW_ALLOW_PRIVATE_NETWORKS=true make run
```

Then build and run the benchmark, with five runs over HTTPS:

```bash
go build -o bin/benchmark ./cmd/benchmark
```

```bash
bin/benchmark -mode e2e -sparrow-url http://localhost:8080 -receiver-tls -runs 5 -duration 30s -rps 500 -json e2e.json
```

The client mode needs no server:

```bash
bin/benchmark -mode client -tls -runs 5 -duration 30s -rps 2000 -concurrency 20
```

Other e2e workloads:

| Flag | What it changes |
|---|---|
| `-webhooks N` | Spread events over N event types, each with its own webhook |
| `-subscribers N` | N webhooks per event type (fan-out) |
| `-receiver-delay 50ms` | Simulated processing time in the receiver (slow receivers) |
| `-burst N` | Publish N events as fast as possible, then measure the drain. There is no schedule, so response time equals service time |

## TLS

- `-tls` (client mode) serves the in-process server over HTTPS.
- `-receiver-tls` (e2e mode) gives the receiver a throwaway self-signed
  certificate and registers the webhooks with `verify_ssl: false`.

Both measure the handshake and encryption cost of real deliveries. They do not
measure certificate-chain verification or [revocation
checks](/sparrow/reference/security/#tls-verification), which `verify_ssl:
false` skips. With keep-alive connections those run once per connection, not per
delivery.

## A separate host

On one machine the load generator, the receiver, Sparrow and Postgres compete
for the same CPUs. The receiver also sits on loopback, so the network costs
nothing.

The e2e receiver runs inside the benchmark process so that publish and receive
times come from one clock. To separate the two sides, run the benchmark on one
host and Sparrow plus Postgres on another:

```bash
bin/benchmark -mode e2e -sparrow-url http://sparrow-host:8080 -receiver-addr 0.0.0.0:9443 -receiver-url https://bench-host:9443 -receiver-tls -runs 5
```

`-receiver-addr` is where the receiver listens. `-receiver-url` is the address
Sparrow uses to reach it. The receiver's self-signed certificate does not name
`bench-host`, which does not matter: the webhooks are registered with
`verify_ssl: false`.

We have run both modes on a single machine. The flags above are what a
two-host setup needs, but we have not yet published results from one.

## Comparing two builds

1. Use the same machine, settings and flags for both, and record them. The JSON
   report does this.
2. Give each build a fresh database. A backlog left by an earlier run is
   delivered to the next run's receiver; the report counts those deliveries as
   `Foreign`.
3. Run each build at least 5 times. Where you can, alternate the builds (A, B,
   A, B ...) so that slow drift on the machine affects both equally.
4. Treat a difference as real only when the confidence intervals do not
   overlap. Do not trust anything the tool lists as noisy until it holds up
   with more runs.

For the cost of individual SQL queries on a large database, use
`make bench-sql` instead (see `scripts/sqlbench/run.sh`).
