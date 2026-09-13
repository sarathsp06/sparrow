# 1. Payload transform engine: Go `text/template`, not embedded JavaScript

- Status: accepted
- Date: 2026-09-13

## Context

Subscriptions reshape webhook payloads before delivery via a per-subscription
`transform_template`. Domain peers (Convoy, Svix, Hookdeck) use an embedded
JavaScript runtime (goja) for this. A recurring architecture suggestion is to
adopt goja for expressiveness (structural JSON reshaping, native conditionals,
type coercion).

Sparrow already ships a hardened engine in `pkg/template`:

- Go `text/template` with 37 built-in helper functions.
- LRU(100) parsed-template cache, SHA-256 keyed.
- DoS bounds: 1MB output cap (`limitedWriter`) + 5s execution timeout.
- Surfaced via REST (`GET /v1/template-functions`, `POST /v1/subscriptions:testTemplate`),
  CLI, and every `recipes/*.yaml`. Trust model is tenant-supplied templates,
  which is why the caps exist.

The only real capability gap versus JS was **structural payload→payload
reshaping** (building objects/arrays instead of emitting JSON as text with
holes, which invites quoting/escaping bugs).

## Decision

Keep Go `text/template` as the sole transform engine. Close the reshaping gap
with helper functions instead of a new runtime: `dict`, `list`, `append`,
`merge`, `dig`, arithmetic (`add`/`sub`/`mul`/`div`/`mod`), and type coercion
(`toString`/`toInt`/`toFloat`). Combined with the existing `json` helper,
these build structured objects/arrays natively — no hand-written JSON text.

Do **not** add goja, and do **not** run two mapping languages.

## Consequences

- **Performance / footprint (the load-bearing reason):** templates parse once
  and are cached; each delivery runs with near-zero overhead. No per-worker JS
  VM to pool, no extra dependency, server stays a single distroless ~15MB
  binary with a small, predictable memory footprint.
- **Concurrency:** parsed templates are reused safely across all delivery
  workers; throughput scales with workers instead of contending on interpreter
  instances (goja allows one `Runtime` per goroutine).
- **Safety:** stdlib templates have no I/O/network/eval; the 1MB + 5s bounds
  remain the guardrail.
- **Tradeoff accepted:** Go template syntax is less familiar than JavaScript,
  and truly arbitrary program logic is out of scope. That is intentional.

## Trigger to revisit

Reopen only if a concrete requirement needs logic templates genuinely cannot
express (not mere familiarity). If flipped, **replace** — migrate all
transforms to one engine; never run templates and JS side by side.
