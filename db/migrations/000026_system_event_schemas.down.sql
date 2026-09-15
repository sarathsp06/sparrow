UPDATE event_registrations
SET schema = NULL, sample_payload = NULL
WHERE name IN ('sparrow.webhook.health_changed', 'sparrow.webhook.delivery_failed');
