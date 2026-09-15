# Deliberately NOT covered by e2e

Scope decisions for the Gauge e2e suite. Each item is intentional, not an
oversight. Revisit when the stated condition changes.

## Out of e2e scope (belongs in unit/integration tests)

| Not testing | Reason |
|---|---|
| Retry backoff timing/jitter | Wall-clock assertions in containerized runs are inherently flaky. The retry *count* and error classification are asserted e2e; backoff math belongs in Go unit tests of the queue worker. |
| Health-status thresholds (healthy → degraded → failing boundaries) | Threshold math depends on windows/ratios that make e2e timing-sensitive. E2E asserts deterministic fields (failure counts, presence in health listing); boundary math belongs in unit tests of the health computation. |
| Delivery ordering guarantees | Sparrow makes no ordering promise (River workers are concurrent). Testing an unpromised property would enshrine an accident. |
| TLS / DNS error classification (`tls_error`, `dns_error`) | Requires a broken-cert or broken-DNS fixture inside Docker networking — high setup cost for classification logic that is pure Go and unit-testable. |
| SSRF / private-network target blocking | The e2e env sets `SPARROW_ALLOW_PRIVATE_NETWORKS=true` by necessity (targets are local containers), so the guard is unobservable here. Unit-test the validator instead. |
| Payload size limit (`SPARROW_MAX_BODY_BYTES`) | Pushing >1 MiB bodies through Gauge/requests is slow and proves only that chi's body limiter works. Unit/integration territory. |
| Load, concurrency, throughput | Not a correctness suite. Benchmarks/soak tests are a separate concern. |
| Encryption-at-rest of webhook secrets | Not observable from the API surface (secrets never round-trip). Covered by `pkg/crypto` unit tests. |
| Migrations / advisory-lock concurrency | Every e2e run exercises startup migration implicitly; concurrent-instance locking needs a dedicated integration test, not Gauge. |

## API surface deliberately not e2e-tested (removal / design candidates)

These endpoints are questioned rather than covered. Writing e2e tests for
them would enshrine surface we may delete.

| Endpoint(s) | Concern |
|---|---|
| `GET/POST /v1/deliveries/{id}`, `…/attempts`, `…:retry` (global, consumer-less trio) | Duplicates the consumer-scoped trio and bypasses tenant scoping — a security smell, not just bloat. Decide: delete, or gate behind an admin key. Test only after the decision. |
| `POST /v1/events/{id}:repush` vs `POST …/events:rePush` (single vs batch job) and `retryDelivery` vs `retryDeliveriesByWebhook` vs `retryBatch` | Three retry entry points / two repush entry points. The single + batch-job pair is canonical; the middle bulk-sync variant is a candidate for removal. Single retry and the batch jobs are e2e-covered; the overlap itself is the open question. |
| `POST /v1/subscriptions:testTemplate`, `GET /v1/template-functions` | UI-support endpoints; a smoke unit test on the handler suffices. E2E already proves templates render in real deliveries. |

## Known product-contract questions (asserted as-is, flagged)

- **Silent template fallback**: a broken transform template falls back to the
  envelope payload and the delivery reports `success`
  (10_template_fallback.spec). This hides misconfiguration from users; the
  delivery should probably carry a template-error flag. The spec asserts
  current behavior; change the spec when the contract changes.
- **Pause drops rather than queues**: events pushed while a webhook is paused
  are never delivered after resume (07_pause_resume.spec asserts drop
  semantics). If queue-on-pause is ever the intended behavior, that spec is
  the one to flip.
- **Dual signing**: an Ed25519-configured webhook also carries a valid HMAC
  signature (01_happy_path.spec). Assumed intentional (migration-friendly);
  confirm before relying on it.
