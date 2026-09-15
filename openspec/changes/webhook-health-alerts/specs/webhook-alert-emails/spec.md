## Purpose

Tenants opt a webhook — or all their webhooks — into email alerts for webhook health degradation
and terminal delivery failures. To the tenant this is a plain configuration ("email X when Y
happens"), not an event subscription; delivery happens internally through Sparrow's own pipeline.

## ADDED Requirements

### Requirement: Alert-email configuration resource

The system SHALL provide a REST resource for alert-email configs, scoped to a consumer. A config
consists of an email address, a scope (one specific `webhook_id`, or consumer-wide when omitted),
and an explicit non-empty list of system event types the email applies to
(`sparrow.webhook.health_changed`, `sparrow.webhook.delivery_failed`). Configs SHALL be creatable,
listable, and deletable. Multiple configs MAY exist for the same webhook and event type; each is an
independent recipient.

#### Scenario: Webhook-level opt-in

- **WHEN** a consumer creates a config with a `webhook_id`, an email, and event types
- **THEN** alerts for those event types on that webhook are emailed to that address

#### Scenario: Consumer-level opt-in

- **WHEN** a consumer creates a config without a `webhook_id`
- **THEN** alerts for the chosen event types on ALL of that consumer's webhooks are emailed to
  that address

#### Scenario: Explicit event types required

- **WHEN** a consumer creates a config with no event types
- **THEN** the request is rejected with a validation error

#### Scenario: Deleting a config stops its emails

- **WHEN** a consumer deletes an alert-email config
- **THEN** subsequent matching events no longer send email to that address

### Requirement: Recipient resolution at push time

When a system event is emitted for a webhook, the system SHALL resolve matching alert-email
configs (configs for that `webhook_id` plus consumer-wide configs of the owning consumer, filtered
by event type) at push time, and attach the resolved recipients to the event payload as derived
metadata (`alert_recipients`). Exactly one event SHALL be pushed per health transition or terminal
delivery failure, regardless of recipient count.

#### Scenario: Multiple recipients, one event

- **WHEN** three configs match a webhook's health transition
- **THEN** one event is pushed carrying all three recipients, and each address receives the email

#### Scenario: No matching configs

- **WHEN** a system event fires for a webhook with no matching alert-email configs
- **THEN** the event is still pushed (with an empty recipient list) and no email is sent

### Requirement: Email delivery through the core pipeline

Alert emails SHALL be delivered via the existing event/subscription/delivery pipeline: an internal
`_sparrow`-owned webhook targeting the SendGrid v3 Mail Send API (configured once by the operator
with Sparrow's own SendGrid credentials, via a `sendgrid` recipe), with subscriptions to the two
system event types. One event SHALL result in one SendGrid API call addressing all recipients.
Tenants SHALL NOT need or supply email/SendGrid credentials.

#### Scenario: Feature inert until operator setup

- **WHEN** no `_sparrow` SendGrid webhook has been configured by the operator
- **THEN** alert-email configs can still be managed, but no emails are sent and no existing
  behavior changes

#### Scenario: Delivery failures are observable like any delivery

- **WHEN** the SendGrid API call fails
- **THEN** the failure is retried and recorded exactly as any other webhook delivery

### Requirement: Human-readable alert content

Alert emails SHALL make the nature of the event unambiguous. For health transitions, the subject
SHALL distinguish degradation from recovery based on the old and new health values. Email content
is defined entirely in the delivery template; tenants do not configure or see it.

#### Scenario: Degradation subject

- **WHEN** a `health_changed` alert is sent for `healthy → degraded`
- **THEN** the subject clearly reads as a degradation, naming the webhook

#### Scenario: Recovery subject

- **WHEN** a `health_changed` alert is sent for `unhealthy → healthy`
- **THEN** the subject clearly reads as a recovery, naming the webhook
