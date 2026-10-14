## Purpose

Sparrow emits events about its own operation (webhook health transitions, terminal delivery
failures) through its own event pipeline, under an internal `_sparrow` consumer, forming a
growable catalog of system events that internal features can consume.

## ADDED Requirements

### Requirement: Internal `_sparrow` consumer

The system SHALL reserve the consumer name `_sparrow` for Sparrow's own internal use, seeded at
migration time. Events emitted by Sparrow about its own operation SHALL be pushed under this
consumer. `_sparrow` and its webhooks/subscriptions MUST NOT appear in any tenant-facing feature;
tenants interact only with the alert-email feature, never with these events directly.

#### Scenario: System events are invisible to tenants

- **WHEN** a consumer lists their events, webhooks, or subscriptions
- **THEN** nothing owned by `_sparrow` is included, because all listings are already scoped by
  consumer name

### Requirement: Registered system event types

The system SHALL register the event types `sparrow.webhook.health_changed` and
`sparrow.webhook.delivery_failed` with explicit JSON schemas, idempotently at startup. The catalog
is expected to grow; each new system event MUST follow the same payload contract: event facts
(what happened) plus derived metadata (data pre-fetched at push time because delivery templates
cannot query the database).

#### Scenario: Idempotent registration

- **WHEN** the server starts and the event types already exist
- **THEN** startup succeeds without duplicating or erroring on the registrations

### Requirement: Health transition event

The system SHALL push one `sparrow.webhook.health_changed` event when a webhook's computed health
state transitions between `healthy`, `degraded`, and `unhealthy` — in either direction. The
transition from `unknown` to `healthy` (first ever delivery) SHALL NOT emit an event. The payload
facts SHALL include `webhook_id`, `consumer`, `url`, `old_health`, and `new_health`.

#### Scenario: Degradation emits one event

- **WHEN** a webhook's health transitions from `healthy` to `degraded`
- **THEN** exactly one `sparrow.webhook.health_changed` event is pushed under consumer `_sparrow`,
  carrying `old_health=healthy`, `new_health=degraded`

#### Scenario: Recovery emits one event

- **WHEN** a webhook's health transitions from `unhealthy` to `healthy`
- **THEN** exactly one `sparrow.webhook.health_changed` event is pushed, carrying
  `old_health=unhealthy`, `new_health=healthy`

#### Scenario: No event without a transition

- **WHEN** deliveries succeed or fail without changing the computed health state
- **THEN** no `sparrow.webhook.health_changed` event is pushed

### Requirement: Terminal delivery failure event

The system SHALL push one `sparrow.webhook.delivery_failed` event when all delivery attempts for
one event to one webhook are exhausted and the delivery will not be retried again. The payload
facts SHALL include `webhook_id`, `consumer`, `url`, the original `event_id`, `delivery_id`,
`error_category`, and `attempt_count`.

#### Scenario: Fired only on terminal failure

- **WHEN** a delivery attempt fails but retries remain
- **THEN** no `sparrow.webhook.delivery_failed` event is pushed

#### Scenario: Fired once when retries are exhausted

- **WHEN** the final permitted delivery attempt for an event to a webhook fails
- **THEN** exactly one `sparrow.webhook.delivery_failed` event is pushed under consumer `_sparrow`

### Requirement: Feedback-loop guard

The system SHALL NOT emit system events for webhooks whose own consumer is `_sparrow`.

#### Scenario: Internal channel failure does not recurse

- **WHEN** the internal alert-delivery webhook itself degrades or exhausts delivery retries
- **THEN** no `sparrow.webhook.health_changed` or `sparrow.webhook.delivery_failed` event is
  emitted for it
