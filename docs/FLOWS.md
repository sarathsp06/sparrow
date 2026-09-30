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
  │    │      :251 — checkWebhookDuplicate() — SELECT by tenant+consumer+url
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

**Entry points:** `internal/rest/event.go` (`registerEventType`, `updateEventType`) and `internal/rest/event_bundle.go` (`importEventTypes`). All of them, plus auto-register on push, go through one save path in `internal/webhooks/event_type_save.go`.

### Flow

```
saveEventType(def, mode)                      mode: create-only (POST), must-exist (PATCH), upsert (import)
  │
  ├─ validateEventTypeName — non-empty, ≤255 chars, not "sparrow." (case-insensitive)
  │
  └─ RunInTransaction → saveEventTypeTx
       ├─ GetEventByNameForUpdate() — SELECT … FOR UPDATE on event_registrations
       ├─ mode check: exists + create-only → AlreadyExists (409); missing + must-exist → NotFound
       ├─ planEventTypeSave(current, def) — pure decision:
       │    missing                        → created (v1)
       │    no schema → schema             → updated (fill in v1, schema_defined_at)
       │    schema differs by value        → new_version (v+1)
       │    only description/metadata/active → updated (same version)
       │    identical                      → unchanged
       ├─ new_version only:
       │    ClassifySchemaChange(old, new) — subscriber-side compatibility (schema_compat.go)
       │    ListSubscriptionsTargetingEvent() — by name or catch-all "*"
       │    breaking + subscriptions + !allow_breaking → Blocked, nothing written (409)
       └─ writes:
            created     → RegisterEvent(): one statement inserts the head row and the v1 history row
            new_version → UpdateEvent() (head, version+1) + AddEventTypeVersion() (history)
            updated     → UpdateEvent(); fill-in also FillInEventTypeVersion()
```

On a unique violation from a concurrent create, an upsert runs once more.

### DB Writes

| Table | Operation | Condition |
|-------|-----------|-----------|
| `event_registrations` | INSERT or UPDATE (1 row) | created / new_version / updated |
| `event_registration_versions` | INSERT (1 row) | created / new_version |
| `event_registration_versions` | UPDATE (1 row) | fill-in only |

Nothing ever deletes from either table; a foreign key from the history to the head row blocks it.

---

## 3. Event Push (PushEvent)

**Entry point:** `internal/rest/event.go` (`pushEvent`) → `WebhookService.PushEvent` in `internal/webhooks/webhook_service_event.go`.

### Flow

```
PushEvent(consumer, event, payload, ttl, metadata, labels, idempotencyKey)
  │
  ├─ Validate consumer, event name (≤255 chars), labels
  ├─ Reject "sparrow.*" names: only Sparrow emits them (queue.pushSystemEvent)
  │
  ├─ IDEMPOTENCY CHECK (when key provided):
  │    GetEventByIdempotencyKey() → found → return (existingID, duplicate=true) ← SHORT CIRCUIT
  │
  ├─ Event type lookup: GetEventByName()
  │    missing + SPARROW_AUTO_REGISTER_EVENTS off (default) → NotFound (404)
  │    missing + on → autoRegisterEvent(): schema-less v1 (re-read on a concurrent create)
  │    inactive → FailedPrecondition (409)
  │
  ├─ SOFT SCHEMA VALIDATION against the current version's schema:
  │    mismatch → schema_valid=false + per-field warnings; EVENT IS STILL ACCEPTED
  │
  ├─ INSERT INTO event_records (… schema_valid, event_version = current version …)
  │
  ├─ ENQUEUE River job EventArgs → queue="events"
  │
  └─ COMPENSATION on enqueue failure: DeleteEventByID()
```

### Key Decision Points

- **Idempotency**: If a key matches an existing event, the flow short-circuits — no new record, no new job. Response includes `duplicate=true`.
- **Unknown event types**: Rejected with 404 unless `SPARROW_AUTO_REGISTER_EVENTS=true`. The CLI and sparrow-sources register on 404 and retry.
- **Version pinning**: `event_version` records the version whose schema the payload was validated against. Rows from before versioning read back as 1.
- **Schema validation**: Failures produce warnings but never reject the event.
- **Compensation**: If River job insertion fails after the event record is written, the orphaned record is deleted.

### DB Writes

| Table | Operation | Condition |
|-------|-----------|-----------|
| `event_records` | INSERT (1 row) | Always (unless idempotency hit) |
| River job table | INSERT (1 row) | Always (unless idempotency hit) |
| `event_registrations` + `event_registration_versions` | INSERT (1 row each) | Only if auto-registered |

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
  │    GetSubscriptionsWithWebhooksByEvent(tenant, consumer, event, labels)
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
  │      subscription paused → WebhookDelivery{status=paused}, no job
  │      otherwise → WebhookDelivery{status=pending, maxAttempts, expiresAt}
  │                + WebhookArgs{deliveryID, webhookID, subscriptionID, eventID, expiresAt, maxAttempts}
  │
  ├─ :135 — BatchCreateDeliveries() — single multi-row INSERT INTO webhook_deliveries
  │
  ├─ :141 — river.InsertMany() — batch INSERT into River jobs (queue="webhooks")
  │
  └─ :148-156 — COMPENSATION on batch insert failure: delete orphaned delivery records
```

### Key Decision Points

- **Catch-all subscriptions**: Subscriptions with `event_name = '*'` match every event in the consumer.
- **Label filtering**: Uses PostgreSQL JSONB containment (`<@`) — the subscription's label_filters must be a subset of the event's labels.
- **Batch efficiency**: All deliveries and River jobs are inserted in a single batch operation each, not one-by-one.
- **Paused subscriptions**: Still get a delivery row, with status `paused` and no job, so held deliveries stay visible and retryable. If every match is paused, no River insert happens.
- **MaxAttempts**: Calculated as `webhook.MaxRetries + 1` with a floor of 3. This means even a webhook with `max_retries=0` gets at least 3 delivery attempts.

### DB Writes

| Table | Operation | Condition |
|-------|-----------|-----------|
| `webhook_deliveries` | Batch INSERT (N rows) | 1 per matching subscription |
| River job table | Batch INSERT (N rows) | 1 per matching subscription that is not paused |

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
  ├─ renderPayload() — runs BEFORE rate limiting, so a failing template never takes a slot:
  │    IF subscription has transform_enabled + template:
  │      TransformPayloadWith(strict = template_missing_key != "zero")
  │      ON TEMPLATE FAILURE → UpdateDeliveryTemplateError(); sparrow_template_errors_total+1
  │        on_transform_error=fail (default) → status=failed, category=template_error,
  │                                            no health event, return nil (no River retry)
  │        on_transform_error=fallback       → BuildEnvelopePayload()
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
- **Template failure**: Per subscription. `fail` (default) marks the delivery failed with `template_error`, sends nothing and is not retried automatically; `fallback` sends the envelope. The error is stored on the delivery either way, and it is never recorded as a health event.
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
| `template_error` | No | Subscription transform failed to render (not a health event) |

### DB Writes (per attempt)

| Table | Operation | Condition |
|-------|-----------|-----------|
| `webhook_deliveries` | UPDATE (status, response_code, response_body, error_message, error_category) | Always |
| `webhook_health_events` | INSERT (1 row) | Every receiver outcome; never for template_error |
| `webhook_health_state` | UPDATE (consecutive_failures, last_success/failure) | Every receiver outcome; never for template_error |

---

## 6. Subscription Creation

**Entry point:** `internal/grpc/subscription_handlers.go:14` — `WebhookServer.CreateSubscription()`

### Flow

```
CreateSubscription(proto request)
  │
  └─ webhook_service.go:2174 — CreateSubscription()
       ├─ :2177 — Validate consumer not empty
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
