# E2E Test Plan

Tests that verify Sparrow works correctly from a user's perspective. Each test tells a story: "I'm a developer setting up webhooks for my application, and I expect this to happen."

## Philosophy

- **Test user workflows, not API methods.** Each test is a scenario a real user would encounter.
- **Few tests, high confidence.** 10 well-chosen scenarios beat 85 granular tests that nobody maintains.
- **Fast feedback.** Single shared Postgres container, parallel-safe via unique namespaces per test.
- **Readable as documentation.** Someone new to the project should understand Sparrow's behavior by reading these tests.

## Technology: Robot Framework

**Why Robot Framework:** Keyword-driven syntax reads like natural language, built-in HTML reports with step-by-step visualization (no plugins needed), Python libraries for mock webhook servers, excellent for behavioral/acceptance testing.

**Stack:**
- Robot Framework 7.x
- RequestsLibrary (HTTP/Connect-RPC calls)
- Custom Python library for webhook target management (mock servers)
- Docker Compose to start Sparrow + Postgres (true black-box e2e)

## Infrastructure

```
e2e/
  requirements.txt          # robotframework, robotframework-requests, flask, etc.
  docker-compose.e2e.yml    # Sparrow + Postgres for test runs
  resources/
    sparrow_api.resource     # Keywords: Register Event, Push Event, List Deliveries, etc.
    webhook_targets.resource # Keywords: Start Target, Wait For Deliveries, Get Delivery Count
    common.resource          # Keywords: Setup Namespace, Teardown Namespace
  libraries/
    WebhookTargetServer.py   # Python library: Flask-based mock webhook targets
    SignatureVerifier.py     # Python library: HMAC + Ed25519 signature verification
  tests/
    01_happy_path.robot
    02_selective_subscription.robot
    03_payload_transformation.robot
    04_retry_recovery.robot
    05_permanent_failure.robot
    06_timeout_handling.robot
    07_pause_resume.robot
    08_idempotency.robot
    09_api_key_auth.robot
    10_template_fallback.robot
    11_manual_retry_and_replay.robot
  results/                   # Generated: HTML reports, logs (gitignored)
```

```bash
make test-e2e
# or: cd e2e && robot --outputdir results tests/
```

### Keyword Design

```robotframework
*** Keywords ***
Register Event
    [Arguments]    ${event_name}    ${namespace}
    # POST to Connect-RPC EventService/RegisterEvent

Push Event
    [Arguments]    ${event_name}    ${namespace}    ${payload}    ${idempotency_key}=${NONE}
    # POST to Connect-RPC EventService/PushEvent, returns event_id

Register Webhook
    [Arguments]    ${name}    ${namespace}    ${url}    ${max_retries}=3    ${timeout}=30s
    # POST to Connect-RPC WebhookService/RegisterWebhook, returns webhook_id + secret + public_key

Subscribe To Event
    [Arguments]    ${webhook_id}    ${event_name}    ${namespace}    ${template}=${NONE}
    # POST to Connect-RPC SubscriptionService/CreateSubscription

Start Webhook Target
    [Arguments]    ${name}    ${behavior}=ok
    # Starts a Flask mock server, returns URL. Behaviors: ok, status_404, status_500, fail_then_succeed_2, slow_10s

Wait For Deliveries
    [Arguments]    ${target_name}    ${count}    ${timeout}=30s
    # Blocks until target has received N deliveries or timeout

Get Delivery Count
    [Arguments]    ${target_name}
    # Returns number of deliveries received by target

Verify HMAC Signature
    [Arguments]    ${delivery}    ${secret}
    # Verifies webhook-signature header contains valid v1,<base64> HMAC

Verify Ed25519 Signature
    [Arguments]    ${delivery}    ${public_key}
    # Verifies webhook-signature header contains valid v1a,<base64> Ed25519 sig

Retry Delivery
    [Arguments]    ${delivery_id}
    # POST to Connect-RPC DeliveryService/RetryDelivery

Replay Event
    [Arguments]    ${event_id}
    # POST to Connect-RPC EventService/RePushEvent
```

### Mock Webhook Target Behaviors

| Behavior | Description |
|----------|-------------|
| `ok` | Always returns 200 |
| `status_404` | Always returns 404 |
| `status_500` | Always returns 500 |
| `fail_then_succeed_N` | Returns 500 for N requests, then 200 |
| `slow_Xs` | Sleeps X seconds before responding 200 |
| `rate_limited_N` | Returns 429 with Retry-After: N header |

---

## Scenarios

### 1. The Happy Path: One Event, Multiple Subscribers

> "I'm building an e-commerce platform. When an order is created, Stripe needs to know (for payment), Shippo needs to know (for shipping), and Slack needs to know (for team notifications). I push one event and all three receive it with valid signatures."

**What this catches:** Fan-out logic, subscription matching, delivery creation, HMAC+Ed25519 dual signing, envelope format, Sparrow headers.

```
Setup:
  - Register event "order.created"
  - Create 3 webhook targets: Stripe endpoint, Shippo endpoint, Slack endpoint
  - Register 3 webhooks (each pointing to a target), each subscribing to "order.created"

Action:
  - Push "order.created" with payload {order_id: "ord-7741", amount: 42.0, customer: "acme-corp"}

Expected:
  - Stripe, Shippo, and Slack each receive exactly 1 delivery
  - Each delivery body is the envelope format (version, event_id, event_name, timestamp, attempt, payload)
  - Each delivery has valid HMAC signature (verify with webhook secret)
  - Each delivery has valid Ed25519 signature (verify with public key from registration)
  - Each delivery has webhook-id, webhook-timestamp, webhook-signature headers
  - ListDeliveries shows 3 deliveries, all status=success
```

---

### 2. Selective Subscription: Only Matching Subscribers Get Hit

> "I'm running a marketplace. Stripe subscribes to order.created (for charging), FedEx subscribes to order.shipped (for logistics), and Datadog subscribes to both (for observability). When an order is created, only Stripe and Datadog should get the webhook -- FedEx has nothing to do with it."

**What this catches:** Subscription filtering by event name, ensuring non-matching subscriptions are NOT delivered to.

```
Setup:
  - Register events: "order.created", "order.shipped"
  - Stripe endpoint subscribes to "order.created" only
  - FedEx endpoint subscribes to "order.shipped" only
  - Datadog endpoint subscribes to both "order.created" and "order.shipped"

Action:
  - Push "order.created" with payload {order_id: "ord-501", total: 129.99}

Expected:
  - Stripe receives 1 delivery
  - FedEx receives 0 deliveries (wait a reasonable time, confirm nothing arrived)
  - Datadog receives 1 delivery
  - ListDeliveries shows exactly 2 deliveries total
```

---

### 3. Payload Transformation via Template

> "I want to forward alerts to Slack, but Slack expects a specific JSON format with a 'text' field. I set up a subscription with a Go template that transforms my alert payload into Slack's format."

**What this catches:** Template compilation, execution, data binding, and the full path from push through fan-out to transformed delivery.

```
Setup:
  - Register event "alert.fired"
  - Register webhook pointing to Slack's incoming webhook endpoint
  - Create subscription with template:
    {"text": "🚨 Alert: {{.payload.title}} - severity {{.payload.severity}}"}

Action:
  - Push "alert.fired" with payload {title: "CPU High on prod-api-3", severity: "critical", host: "prod-api-3"}

Expected:
  - Slack endpoint receives delivery with body: {"text": "🚨 Alert: CPU High on prod-api-3 - severity critical"}
  - Body is NOT the default envelope format (template overrides it)
```

---

### 4. Target Failures and Retry Recovery

> "Google's Cloud Functions endpoint is having a bad deployment -- it returns 500 for a few minutes, then recovers. Sparrow should keep retrying and eventually deliver successfully once Google's endpoint is back."

**What this catches:** Retry logic, exponential backoff actually re-enqueuing, status transitions (pending -> retrying -> success), attempt counting.

```
Setup:
  - Register event "user.signup" + webhook pointing to Google Cloud Function (max_retries=3, backoff=1s for fast test)
  - Google's endpoint is configured to return 500 for first 2 requests, then 200

Action:
  - Push "user.signup" with payload {user_id: "usr-8821", email: "jane@example.com"}

Expected:
  - Google's endpoint eventually receives the delivery (after retries)
  - Delivery status via API = success
  - Attempt count = 3 (2 failures + 1 success)
  - GetDeliveryAttempts shows 3 attempts with increasing timestamps
```

---

### 5. Permanent Failure: Target Always Broken

> "Facebook's webhook endpoint has been decommissioned and returns 404. Sparrow should NOT waste retries on a client error -- mark it failed immediately. Separately, if Microsoft's endpoint returns 500 forever, Sparrow should exhaust retries then give up."

**What this catches:** Error classification (4xx = client_error = not retryable), terminal failure state, no wasted retry attempts. Also: 5xx exhaustion path.

```
Setup (Case A - client error):
  - Register event "payment.refunded" + webhook pointing to Facebook (max_retries=3)
  - Facebook's endpoint always returns 404

Action:
  - Push "payment.refunded" with payload {payment_id: "pay-112", amount: 25.00}

Expected:
  - Facebook receives exactly 1 request (no retries for 4xx)
  - Delivery status = failed
  - error_category = "client_error"
  - Attempt count = 1

Setup (Case B - server error exhaustion):
  - Register webhook pointing to Microsoft (max_retries=3)
  - Microsoft's endpoint always returns 500

Action:
  - Push same event type

Expected:
  - Microsoft receives 4 requests (initial + 3 retries)
  - Delivery status = failed after all attempts exhausted
  - error_category = "server_error"
  - Attempt count = 4
```
Setup:
  - Register event + webhook (max_retries=3)
  - Target always returns 404

Action:
  - Push event

Verify:
  - Target receives exactly 1 request (no retries for 4xx)
  - Delivery status = failed
  - error_category = "client_error"
  - Attempt count = 1
```

Also test separately with target returning 500 (retryable) and exhausting all retries:
  - Target always returns 500
  - Delivery status = failed after max_retries+1 attempts
  - error_category = "server_error"

---

### 6. Timeout Handling

> "Shopify's webhook receiver is overloaded and takes 30 seconds to respond. Sparrow should time out after the configured threshold, classify it as a timeout, and retry."

**What this catches:** Request timeout enforcement, timeout error classification, retry of timeouts.

```
Setup:
  - Register event "cart.abandoned" + webhook pointing to Shopify (request_timeout=2s, max_retries=1)
  - Shopify's endpoint sleeps 10 seconds before responding

Action:
  - Push "cart.abandoned" with payload {cart_id: "cart-9921", items: 3}

Expected:
  - Delivery error_category = "timeout"
  - Status = failed (after retry also times out)
  - Shopify received 2 requests (initial + 1 retry, both timed out client-side)
```

---

### 7. Paused Webhook Skips Delivery

> "I'm migrating Twilio's webhook endpoint to a new URL. I pause it, push events during the migration window, then resume. Events pushed while paused should not be delivered, but new events after resume should work fine."

**What this catches:** Pause/resume lifecycle, subscription matching respects active state, webhook state transitions.

```
Setup:
  - Register event "sms.received" + webhook pointing to Twilio + subscription

Action 1 (baseline):
  - Push "sms.received" with payload {from: "+1555000111", body: "Hello"}
  - Verify Twilio receives delivery (proves normal flow works)

Action 2 (paused):
  - Pause the Twilio webhook
  - Push "sms.received" with payload {from: "+1555000222", body: "During maintenance"}
  - Wait 5 seconds

Expected:
  - Twilio received nothing new (still only 1 delivery from baseline)

Action 3 (resumed):
  - Resume the Twilio webhook
  - Push "sms.received" with payload {from: "+1555000333", body: "After maintenance"}

Expected:
  - Twilio receives the new delivery (now 2 total: baseline + post-resume)
  - The event pushed during pause was NOT delivered (no catch-up)
  - ListDeliveries shows deliveries only for the non-paused pushes
```

---

### 8. Idempotency: Same Event Pushed Twice

> "My payment service has at-least-once semantics and might call PushEvent twice for the same charge. I use an idempotency key so Sparrow deduplicates and only delivers once -- Stripe shouldn't charge the customer twice."

**What this catches:** Idempotency key dedup at the DB level, duplicate flag in response, single fan-out.

```
Setup:
  - Register event "charge.completed" + webhook pointing to Stripe + subscription

Action:
  - Push "charge.completed" with id="charge-xyz-001" (idempotency key), payload {amount: 99.99}
  - Push same event again with same id="charge-xyz-001"

Expected:
  - First push: duplicate=false, returns event_id
  - Second push: duplicate=true, returns SAME event_id
  - Stripe receives exactly 1 delivery (not 2)
  - ListDeliveries shows 1 delivery total

Also verify: push "charge.completed" twice WITHOUT idempotency key -> 2 separate events, 2 deliveries to Stripe
```

---

### 9. API Key Authentication

> "Sparrow is deployed behind a VPN but we still want a shared secret so only our services can call the API. When SPARROW_API_KEY is set, unauthenticated requests should be rejected, but health checks must remain open for the load balancer."

**What this catches:** Auth middleware on Connect-RPC paths, correct error codes, that health/ready endpoints remain open.

```
Setup:
  - Start the test server WITH SPARROW_API_KEY="sk-sparrow-test-secret-2024"

Action & Expected:
  - ListEvents without any key -> HTTP 401 Unauthorized
  - ListEvents with wrong key "sk-wrong-key" -> HTTP 401 Unauthorized
  - ListEvents with correct X-API-Key header "sk-sparrow-test-secret-2024" -> success (200)
  - GET /health without any key -> success (health endpoints are exempt)
  - GET /ready without any key -> success (readiness endpoints are exempt)
```

Note: This test may need its own sub-environment since the global env runs without an API key for simplicity. Alternatively, test against the middleware directly via a separate HTTP handler wired with the auth middleware.

---

### 10. Invalid Template Graceful Degradation

> "An engineer on the team wrote a bad Go template for the PagerDuty subscription -- it references a field that doesn't exist. When an event fires, PagerDuty should still get the delivery (with the raw envelope payload as fallback) rather than silently dropping it."

**What this catches:** Template execution error handling, fallback to envelope format, delivery still succeeds despite transform failure.

```
Setup:
  - Register event "deploy.failed"
  - Register webhook pointing to PagerDuty
  - Create subscription with broken template: "{{.nonexistent.deep.field}}"

Action:
  - Push "deploy.failed" with payload {service: "api-gateway", version: "v2.3.1", error: "health check timeout"}

Expected:
  - PagerDuty receives delivery (not dropped!)
  - Delivery body IS the standard envelope format (fallback, since template failed)
  - Delivery status = success (the HTTP call succeeded; template failure is internal)
```

---

### 11. Manual Retry and Event Replay

> "We pushed an invoice event but Facebook's endpoint was down (404 -- decommissioned URL). The delivery failed permanently. After fixing the URL, we manually retry that specific delivery. Separately, we also want to replay the entire event to pick up a new subscriber (Zendesk) that was added after the original push."

**What this catches:** RetryDelivery re-enqueues a single failed delivery, RePushEvent replays the original event through current subscriptions (which may have changed since original push).

```
Setup:
  - Register event "invoice.sent"
  - Register webhook "Facebook" (target returns 404 -> permanent failure)
  - Register webhook "GitHub" (target returns 200)
  - Subscribe both to "invoice.sent"

Action 1 -- push and let Facebook fail:
  - Push "invoice.sent" with payload {invoice_id: "inv-2024-099", customer: "Acme Corp", total: 15000.00}
  - Wait for deliveries to complete

Expected after push:
  - GitHub's delivery = success
  - Facebook's delivery = failed (client_error, 1 attempt, no retries for 4xx)

Action 2 -- manually retry Facebook's failed delivery:
  - Fix Facebook's endpoint (switch handler to return 200)
  - Call RetryDelivery with Facebook's failed delivery ID
  - Wait for delivery to complete

Expected after retry:
  - Facebook's delivery status = success
  - Attempt count = 2 (original 404 failure + successful retry)
  - Facebook's endpoint received exactly 1 new request (the retry)

Action 3 -- replay the entire event (with a new subscriber):
  - Register a NEW webhook "Zendesk" (target returns 200), subscribe to "invoice.sent"
  - Call RePushEvent with the original event ID

Expected after replay:
  - Facebook receives a new delivery (separate from the retried one)
  - GitHub receives a new delivery
  - Zendesk receives a delivery (proves replay uses CURRENT subscriptions, not original)
  - ListDeliveries for the re-pushed event shows 3 new deliveries
  - The re-push response has duplicate=false (re-push bypasses idempotency)
```

---

## Implementation Order

1. Infrastructure: `docker-compose.e2e.yml`, `requirements.txt`, Python libraries (`WebhookTargetServer.py`, `SignatureVerifier.py`)
2. Resource files: `sparrow_api.resource`, `webhook_targets.resource`, `common.resource`
3. `01_happy_path.robot` -- proves the pipeline works end-to-end
4. `05_permanent_failure.robot` + `04_retry_recovery.robot` -- error handling
5. `02_selective_subscription.robot` -- subscription filtering
6. `08_idempotency.robot` -- dedup
7. `03_payload_transformation.robot` + `10_template_fallback.robot` -- templates
8. `07_pause_resume.robot` -- lifecycle
9. `06_timeout_handling.robot` -- timeout classification
10. `09_api_key_auth.robot` -- authentication
11. `11_manual_retry_and_replay.robot` -- retry + replay operations

## Running

```bash
make test-e2e                                         # all scenarios, generates HTML report
cd e2e && robot --outputdir results tests/             # same, manual
cd e2e && robot --outputdir results tests/01_happy_path.robot  # single scenario

# View report after run:
open e2e/results/report.html
```

### Generated Reports

Robot Framework produces three files automatically (no plugins needed):
- **`report.html`** -- High-level summary with pass/fail per test, execution stats, tag breakdown
- **`log.html`** -- Detailed step-by-step execution log, expandable keyword calls with arguments and return values
- **`output.xml`** -- Machine-readable results (for CI integration, trend analysis)

## Relationship to Existing Integration Tests

`internal/integration/e2e_test.go` covers a subset of Scenario 1 (single subscriber, HMAC only). After these Robot Framework e2e tests are implemented, the existing integration test can be removed and `make test-integration` retired in favor of `make test-e2e`.
