-- A webhook whose receiver only accepts a specific payload shape (Slack,
-- SendGrid, PagerDuty, ...) can require a payload transform. Every one of its
-- subscriptions must then carry an enabled transform_template; the API rejects
-- one without, and a delivery that somehow has none fails with template_error
-- instead of sending Sparrow's default envelope.
ALTER TABLE webhook_registrations
    ADD COLUMN requires_transform BOOLEAN NOT NULL DEFAULT false;
