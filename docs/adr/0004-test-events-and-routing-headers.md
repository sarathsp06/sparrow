# 4. Routing headers and test events instead of a receiver code generator

- Status: accepted
- Date: 2026-10-20

## Context

Receivers get the same few things wrong: they parse JSON before verifying the
raw body, route on a body field a transform removed, and have no way to check
their wiring without waiting for a real event. We prototyped a receiver code
generator (Go, Python, Node projects with inferred models and tests) and
measured its cost before shipping it.

## Decision

- **Routing headers.** Every delivery carries `X-Sparrow-Event-Type`,
  `X-Sparrow-Event-Version` and `X-Sparrow-Subscription-ID`, set after custom
  headers like the other `X-Sparrow-*` IDs, so receivers route on headers
  whatever the transform does to the body.
- **Test events.** `POST …/subscriptions/{id}:sendTestEvent` sends an event
  (`event_name`, which the subscription must receive) to the subscription: one
  delivery of its sample, or of a given payload, to the subscription's webhook through the real request
  builder, signing, network policy and timeout, marked `X-Sparrow-Test: true`,
  and returns the receiver's status, latency and error class. It is
  synchronous, writes no delivery row and never touches health or
  auto-disable (Design Principle 3). It is limited per webhook in process
  (one every 2 seconds per server); Postgres stays the only shared store.
- **Verification stays with the existing helpers** in `client/verify/` and
  `pkg/signature`, linked from the Test tab and the signatures guide.

## Alternatives rejected

- **A receiver code generator** (`pkg/receivercode`). It worked, but cost about
  7k hand-written lines across three languages, a new versioned module, a
  separate contract-test module and 23k lines of golden snapshots, with an
  ongoing cost in framework idioms and pinned dependencies. What it added over
  the helpers was mostly inferred models and wiring, which coding agents
  already write well from a schema and the routing headers.
- **Test events as real, queued deliveries** (retries, a delivery row and a
  test flag). More useful in the deliveries list, but it needs a migration, a
  worker change to skip health bookkeeping and stats filtering. Revisit if
  people ask to see test sends in their delivery history.
