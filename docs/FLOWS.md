# Sparrow — Detailed Flow Reference

> This document traces every major operation through the codebase, showing the exact function call chain, decision points, and database writes.

---

## Table of Contents

1. [Webhook Registration](#1-webhook-registration)
2. [Event Registration](#2-event-registration)
3. [Event Push (PushEvent)](#3-event-push-pushevent)
4. [Event Processing (EventProcessingWorker)](#4-event-processing-eventprocessingworker)
5. [Webhook Delivery (WebhookWorker)](#5-webhook-delivery-webhookworker)
6. [Subscription Creation](#6-subscription-creation)
7. [Webhook Update (UpdateWebhookConfig)](#7-webhook-update-updatewebhookconfig)

---

## 1. Webhook Registration

**Entry point:** `internal/grpc/webhook_handlers.go:11` — `WebhookServer.RegisterWebhook()`

### Flow

```
RegisterWebhook(proto request)
  │
  ├─ webhook_conversions.go:95 — CreateWebhookRegistrationRequest()
  │    Convert proto → internal request, apply HTTP config defaults
  │    (max_retries=3, backoff=60s, timeout=30s, status_codes=[200,201,202,204])
  │
  ├─ webhook_service.go:428 — CreateWebhook()
  │    ├─ :451 — ValidateWebhookURL() — SSRF protection (blocks private IPs, loopback, metadata)
  │    ├─ :456 — req.ToWebhookRegistration() — merge defaults via ApplyConfig() + ValidateConfig()
  │    ├─ :462 — Generate UUID if not provided
  │    ├─ :467-482 — Validate event names exist (warn only, never blocks)
  │    ├─ :490-509 — Build store.WebhookRegistration struct
  │    ├─ :512-516 — Set signature_type: defaults to "hmac" unless "ed25519" specified
  │    ├─ :520-527 — Encrypt webhook secret → crypto.EncryptString() → AES-256-GCM
  │    ├─ :538-548 — Ed25519 keygen (ONLY when signature_type="ed25519" AND crypto enabled):
  │    │    ed25519.GenerateKey(nil) → encrypt private key → store ciphertext
  │    ├─ :551-557 — Encrypt secret headers → crypto.EncryptJSON()
  │    ├─ :567-573 — Build subscriptions (one per event name)
  │    │
  │    ├─ webhook_repository.go:249 — RegisterWebhookWithSubscriptions()
  │    │    TRANSACTION {
  │    │      :251 — checkWebhookDuplicate() — SELECT by tenant+namespace+url
  │    │      :254 — INSERT INTO webhook_registrations (25 columns)
  │    │      :259 — For each event: INSERT INTO event_subscriptions (13 columns)
  │    │    }
  │    │
  │    ├─ :589-597 — If rate_limit_rps set: UPSERT INTO webhook_rate_limit_state
  │    └─ :600-603 — OTel metrics: WebhookRegistrations+1, ActiveWebhooks+1
  │
  └─ webhook_handlers.go:17-22 — Build response:
       webhook_id, created_at, signing_public_key (derived from encrypted privkey), signature_type
```

### DB Writes

| Table | Operation | Condition |
|-------|-----------|-----------|
| `webhook_registrations` | INSERT (1 row) | Always |
| `event_subscriptions` | INSERT (N rows, 1 per event) | Always |
| `webhook_rate_limit_state` | UPSERT (1 row) | Only if `rate_limit_rps` set |

---

## 2. Event Registration

**Entry point:** `internal/grpc/event_handlers.go:44` — `WebhookServer.RegisterEvent()`

### Flow

```
RegisterEvent(proto request)
  │
  ├─ :47-49 — Convert proto Schema Struct → map[string]any
  │
  └─ webhook_service.go:1405 — RegisterEvent()
       ├─ :1410 — Validate name not empty
       ├─ :1418 — GetEventByName() — check for duplicate
       ├─ :1423 — If exists → InvalidInput("event already exists")
       ├─ :1428 — generateSamplePayload(schema) — uses schemagen to create example payload
       ├─ :1434 — Build store.EventRegistration struct
       └─ :1442 — INSERT INTO event_registrations (9 columns)
```

### DB Writes

| Table | Operation | Condition |
|-------|-----------|-----------|
| `event_registrations` | INSERT (1 row) | Always |

---

## 3. Event Push (PushEvent)

**Entry point:** `internal/grpc/event_handlers.go:19` — `WebhookServer.PushEvent()`

### Flow

```
PushEvent(proto request)
  │
  ├─ :20-30 — Convert payload, extract optional idempotency key from req.Id
  │
  └─ webhook_service.go:728 — PushEvent()
       ├─ :745-761 — Validate namespace, event name, labels (max 20, key≤64, value≤256)
       │
       ├─ :768-785 — IDEMPOTENCY CHECK (when key provided):
       │    GetEventByIdempotencyKey() → SELECT by tenant+namespace+key
       │    If found → return (existingID, duplicate=true) ← SHORT CIRCUIT
       │
       ├─ :788-811 — Event lookup / auto-register:
       │    GetEventByName() → SELECT
       │    If nil → auto-register event (INSERT INTO event_registrations)
       │    If inactive → FailedPrecondition error
       │
       ├─ :825-846 — SOFT SCHEMA VALIDATION (when schema present):
       │    ValidateJSONSchema() → compile + validate
       │    On mismatch: schema_valid=false, extract per-field warnings
       │    EVENT IS STILL ACCEPTED (warnings only)
       │
       ├─ :854-869 — Build store.EventRecord (uuid, payload, ttl, labels, schema_valid, idempotency_key)
       ├─ :871 — INSERT INTO event_records (12 columns)
       │
       ├─ :896 — ENQUEUE River job:
       │    EventArgs{TenantID, EventID, Namespace, Event, TTL, Metadata, Labels}
       │    → river.Insert() → queue="events", kind="event_processing"
       │    OTel trace context injected into job.Metadata
       │
       ├─ :907-912 — COMPENSATION on enqueue failure:
       │    DeleteEventByID() — remove orphaned event record
       │
       └─ :917-919 — OTel: EventsPushed+1
```

### Key Decision Points

- **Idempotency**: If a key matches an existing event, the entire flow short-circuits — no new record, no new job. Response includes `duplicate=true`.
- **Schema validation**: Failures produce warnings but never reject the event. The `schema_valid` flag is set to `false` on the record.
- **Auto-registration**: If the event type doesn't exist yet, it's automatically created. This means `PushEvent` never fails due to a missing event type.
- **Compensation**: If River job insertion fails after the event record is written, the orphaned record is deleted to maintain consistency.

### DB Writes

| Table | Operation | Condition |
|-------|-----------|-----------|
| `event_records` | INSERT (1 row) | Always (unless idempotency hit) |
| River job table | INSERT (1 row) | Always (unless idempotency hit) |
| `event_registrations` | INSERT (1 row) | Only if event type auto-registered |

---

## 4. Event Processing (EventProcessingWorker)

**Entry point:** `internal/webhooks/queue/events_worker.go:38` — River picks up job from `"events"` queue.

### Flow

```
EventProcessingWorker.Work(job)
  │
  ├─ :43-48 — Restore OTel trace context from job.Metadata
  ├─ :55-59 — GetEventByID() — load event record (verify exists)
  │
  ├─ :67 — SUBSCRIPTION MATCHING:
  │    GetSubscriptionsWithWebhooksByEvent(tenant, namespace, event, labels)
  │    → JOIN event_subscriptions + webhook_registrations WHERE:
  │      - event_name matches OR event_name = '*' (catch-all)
  │      - webhook active = true
  │      - label_filters match via JSONB containment (es.label_filters <@ $4::jsonb)
  │
  ├─ :73-79 — No subscriptions? → return nil (no-op, no deliveries)
  │
  ├─ :88-93 — Calculate expiresAt: TTL≤0 → year 9999 (never expires); else now+TTL
  │
  ├─ :96-132 — BUILD DELIVERIES IN MEMORY:
  │    For each SubscriptionWithWebhook:
  │      deliveryID = uuid.New()
  │      maxAttempts = webhook.MaxRetries + 1 (min 3)
  │      → WebhookDelivery{status=pending, maxAttempts, expiresAt}
  │      → WebhookArgs{deliveryID, webhookID, subscriptionID, eventID, expiresAt, maxAttempts}
  │
  ├─ :135 — BatchCreateDeliveries() — single multi-row INSERT INTO webhook_deliveries
  │
  ├─ :141 — river.InsertMany() — batch INSERT into River jobs (queue="webhooks")
  │
  └─ :148-156 — COMPENSATION on batch insert failure: delete orphaned delivery records
```

### Key Decision Points

- **Catch-all subscriptions**: Subscriptions with `event_name = '*'` match every event in the namespace.
- **Label filtering**: Uses PostgreSQL JSONB containment (`<@`) — the subscription's label_filters must be a subset of the event's labels.
- **Batch efficiency**: All deliveries and River jobs are inserted in a single batch operation each, not one-by-one.
- **MaxAttempts**: Calculated as `webhook.MaxRetries + 1` with a floor of 3. This means even a webhook with `max_retries=0` gets at least 3 delivery attempts.

### DB Writes

| Table | Operation | Condition |
|-------|-----------|-----------|
| `webhook_deliveries` | Batch INSERT (N rows) | 1 per matching subscription |
| River job table | Batch INSERT (N rows) | 1 per matching subscription |

---

## 5. Webhook Delivery (WebhookWorker)

**Entry point:** `internal/webhooks/queue/webhook_worker.go:63` — River picks up job from `"webhooks"` queue.

### Flow

```
WebhookWorker.Work(job)
  │
  ├─ :67-71 — Restore OTel trace context
  ├─ :77 — GetWebhookByID() — load full webhook config (including signature_type, encrypted keys)
  ├─ :85 — GetEventByID() — load event record (payload)
  ├─ :94-103 — Load subscription if present (optional, continues without)
  │
  ├─ :120-129 — TTL CHECK:
  │    if time.Now().After(args.ExpiresAt) → update status=expired, return nil
  │
  ├─ :136-159 — RATE LIMITING (when webhook.RateLimitRPS > 0):
  │    AcquireDeliverySlot() → atomic UPDATE on webhook_rate_limit_state (leaky bucket)
  │    If slot is in the future → river.JobSnooze(delay) ← RE-ENQUEUE WITH DELAY
  │
  ├─ :161-207 — PAYLOAD CONSTRUCTION:
  │    IF subscription has transform_enabled + template:
  │      TransformPayload() — Go template execution
  │      ON TEMPLATE FAILURE → graceful degradation: BuildEnvelopePayload() (fallback)
  │    ELSE:
  │      BuildEnvelopePayload() → JSON envelope:
  │        {version, event_id, event_name, timestamp, attempt, payload}
  │
  ├─ :210 — PrepareDeliveryRequest() (client/request.go:150):
  │    ├─ Merge headers: webhook → subscription → decrypted secret headers (secret wins)
  │    ├─ Determine method (subscription override or POST)
  │    ├─ Determine timeout (subscription override or webhook config, default 30s)
  │    ├─ Decrypt webhook secret for HMAC signing
  │    ├─ Decrypt Ed25519 private key for asymmetric signing
  │    └─ Set SignatureType from webhook.SignatureType
  │
  ├─ :213 — UpdateDeliveryRequestBody() — store request body on delivery record
  │
  ├─ :218 — client.Send() → internally calls BuildRequest() (client/request.go:80):
  │    ├─ Set headers: Content-Type, User-Agent, X-Sparrow-Event-ID/Delivery-ID/Webhook-ID
  │    ├─ Set custom headers
  │    └─ STANDARD WEBHOOKS SIGNING (when secret present):
  │         webhook-id = "msg_" + deliveryID
  │         webhook-timestamp = unix seconds
  │         message = "{msgID}.{timestamp}.{payload}"
  │         ┌─ signature_type="hmac" (default):
  │         │    HMAC-SHA256(message, secret) → "v1," + base64
  │         └─ signature_type="ed25519":
  │              Ed25519.Sign(privKey, message) → "v1a," + base64
  │         → webhook-signature header (single signature)
  │
  │  ┌─────────────── RESPONSE HANDLING ───────────────┐
  │  │                                                  │
  ├─ :220-244 — TRANSPORT ERROR (no HTTP response):
   │    ClassifyError() → error category
  │    Update delivery → StatusFailed
  │    Record health event + update health state
  │    Non-retryable (dns_error, tls_error) → return nil (done)
  │    Retryable (timeout, connection_refused, network_error) → return error (River retries)
  │
  ├─ :271-288 — SUCCESS (status code in expected list):
  │    Update delivery → StatusSuccess + response code/body
  │    Record health event (success)
  │    Return nil
  │
  ├─ :292-314 — HTTP 429 (rate limited by target):
  │    Parse Retry-After header (seconds or HTTP-date, capped at 15min, default 60s)
  │    Record health event (rate_limited)
  │    river.JobSnooze(duration) ← DOES NOT COUNT AS RETRY ATTEMPT
  │
  └─ :319-357 — OTHER HTTP FAILURES:
       Classify: 4xx→client_error, 5xx→server_error, 2xx-not-expected→unexpected_status
       Update delivery → StatusFailed
       Record health event
       Non-retryable (client_error, unexpected_status) → return nil
       Retryable (server_error) → return error (River retries with backoff)
```

### Key Decision Points

- **TTL expiry**: Checked before any work. Expired deliveries are marked and abandoned.
- **Rate limiting**: Uses a leaky bucket stored in PostgreSQL. If no slot is available, the job is snoozed (re-enqueued) without counting as an attempt.
- **Template failure**: Never fails the delivery — falls back to the standard envelope payload.
- **Signing**: Only one scheme is used per webhook, determined by `signature_type` ("hmac" or "ed25519").
- **HTTP 429**: Uniquely handled — the job is snoozed for the `Retry-After` duration without counting as a retry attempt. This prevents exhausting retries against rate-limited endpoints.
- **Error classification**: Determines retryability. DNS and TLS errors are terminal (endpoint is misconfigured). Server errors, timeouts, and network errors trigger retries.

### Error Categories and Retryability

| Category | Retryable | Triggers |
|----------|-----------|----------|
| `success` | — | HTTP status in expected list |
| `client_error` | No | HTTP 4xx (except 429) |
| `server_error` | Yes | HTTP 5xx |
| `timeout` | Yes | Request timeout exceeded |
| `dns_error` | No | DNS resolution failed |
| `tls_error` | No | TLS handshake failed |
| `connection_refused` | Yes | TCP connection refused |
| `network_error` | Yes | Other network errors |
| `unexpected_status` | No | HTTP 2xx/3xx not in expected_status_codes |
| `rate_limited` | Yes | HTTP 429 |

### DB Writes (per attempt)

| Table | Operation | Condition |
|-------|-----------|-----------|
| `webhook_deliveries` | UPDATE (status, response_code, response_body, error_message, error_category) | Always |
| `webhook_health_events` | INSERT (1 row) | Always |
| `webhook_health_state` | UPDATE (consecutive_failures, last_success/failure) | Always |

---

## 6. Subscription Creation

**Entry point:** `internal/grpc/subscription_handlers.go:14` — `WebhookServer.CreateSubscription()`

### Flow

```
CreateSubscription(proto request)
  │
  └─ webhook_service.go:2174 — CreateSubscription()
       ├─ :2177 — Validate namespace not empty
       ├─ :2183 — Parse webhook UUID
       ├─ :2188 — validateLabels(labelFilters) — max 20, key/value constraints
       ├─ :2192 — Build store.EventSubscription struct
       └─ :2204 — insertSubscription() (webhook_repository.go:419)
            Generate UUID, marshal headers + label_filters to JSON
            INSERT INTO event_subscriptions (13 columns)
```

### Label Filter Semantics

- A subscription with empty `label_filters` (`{}`) matches **all** events of that type — it's a catch-all for that event name.
- A subscription with label filters only matches events whose labels are a **superset** of the filter (PostgreSQL `<@` containment).
- Example: filter `{"env": "prod"}` matches events with labels `{"env": "prod", "region": "us-east-1"}` but not `{"env": "staging"}`.

### DB Writes

| Table | Operation | Condition |
|-------|-----------|-----------|
| `event_subscriptions` | INSERT (1 row) | Always |

---

## 7. Webhook Update (UpdateWebhookConfig)

**Entry point:** `internal/grpc/webhook_handlers.go:61` — `WebhookServer.UpdateWebhookConfig()`

### Flow

```
UpdateWebhookConfig(proto request)
  │
  ├─ :62-94 — Extract all fields from proto: events, url, headers, secretHeaders,
  │           timeout, active, description, signatureType, httpConfig
  ├─ :96 — Extract field mask paths
  │
  └─ webhook_service.go:1809 — UpdateWebhookConfig()
       ├─ :1832 — GetWebhookByID() — load existing webhook
       ├─ :1839-1850 — Build mask set (O(1) lookup). No mask = legacy (apply all non-zero)
       │
       ├─ :1864-1873 — URL update: trim + ValidateWebhookURL() (SSRF check)
       ├─ :1874-1882 — Headers, active, description (conditional on mask)
       ├─ :1884-1937 — HTTP config updates:
       │    MaxRetries, BackoffSeconds, TimeoutSeconds, StatusCodes,
       │    WebhookSecret (mask-gated: "http_config.webhook_secret"),
       │    RateLimitRPS (mask-gated: "http_config.rate_limit_rps"),
       │    CaptureResponseBody, FollowRedirects, VerifySSL
       ├─ :1939-1945 — Secret headers encryption (mask-gated)
       │
       ├─ :1947-1972 — SIGNATURE TYPE SWITCHING (mask-gated: "signature_type"):
       │    Validate "hmac" or "ed25519"
       │    TO ed25519: ed25519.GenerateKey() → encrypt → store private key
       │    TO hmac:    webhook.Ed25519PrivateKey = nil (clear key)
       │
       ├─ :1975-1982 — ATOMIC TRANSACTION:
       │    IF events changed: ReplaceWebhookSubscriptions()
       │      → DELETE all existing + INSERT new subscriptions
       │    UpdateWebhook() → UPDATE webhook_registrations (22 columns)
       │
       └─ :1991-2006 — Rate limit state management:
            If RateLimitRPS set → UPSERT webhook_rate_limit_state
            If nil → DELETE webhook_rate_limit_state
```

### Field Mask Behavior

The `update_mask` controls which fields are applied. This prevents accidental overwrites:

| Mask Path | What it controls |
|-----------|-----------------|
| `"url"` | Target URL |
| `"active"` | Active/inactive state |
| `"description"` | Description text |
| `"events"` | Replace all subscriptions |
| `"headers"` | Replace all custom headers |
| `"secret_headers"` | Replace all encrypted headers |
| `"http_config"` | All HTTP config fields |
| `"http_config.webhook_secret"` | Only the HMAC secret within http_config |
| `"http_config.rate_limit_rps"` | Only the rate limit within http_config |
| `"signature_type"` | Signing scheme (triggers keygen/key removal) |

When `update_mask` is **empty or omitted**, all non-zero fields are applied (legacy behavior).

### DB Writes

| Table | Operation | Condition |
|-------|-----------|-----------|
| `webhook_registrations` | UPDATE (1 row) | Always |
| `event_subscriptions` | DELETE all + INSERT new | Only if events changed |
| `webhook_rate_limit_state` | UPSERT or DELETE | Only if rate_limit_rps changed |

---

## End-to-End Sequence Diagram

```
Client                    Sparrow Server              PostgreSQL              Target URL
  │                           │                          │                      │
  │── PushEvent ─────────────>│                          │                      │
  │                           │── INSERT event_records ─>│                      │
  │                           │── INSERT River job ─────>│                      │
  │<── {event_id} ────────────│                          │                      │
  │                           │                          │                      │
  │                    [EventProcessingWorker picks up job]                     │
  │                           │── SELECT subscriptions ─>│                      │
  │                           │<─ N matching subs ───────│                      │
  │                           │── BATCH INSERT deliveries>│                      │
  │                           │── BATCH INSERT River jobs>│                      │
  │                           │                          │                      │
  │                    [WebhookWorker picks up job (×N)]                        │
  │                           │── SELECT webhook config ─>│                      │
  │                           │── SELECT event record ──>│                      │
  │                           │                          │                      │
  │                           │── HTTP POST (signed) ───────────────────────────>│
  │                           │<── 200 OK ──────────────────────────────────────│
  │                           │                          │                      │
  │                           │── UPDATE delivery=success>│                      │
  │                           │── INSERT health_event ──>│                      │
  │                           │── UPDATE health_state ──>│                      │
```
