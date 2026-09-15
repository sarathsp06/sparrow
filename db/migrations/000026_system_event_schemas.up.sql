-- Backfill JSON Schema for Sparrow's self-generated system event types,
-- which were previously auto-registered with an empty schema on first fire.
UPDATE event_registrations
SET schema = '{
  "type": "object",
  "required": ["webhook_id", "consumer", "url", "old_health", "new_health"],
  "properties": {
    "webhook_id": {"type": "string", "format": "uuid"},
    "consumer": {"type": "string"},
    "url": {"type": "string"},
    "old_health": {"type": "string"},
    "new_health": {"type": "string"},
    "alert_recipients": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {"email": {"type": "string", "format": "email"}}
      }
    }
  }
}'
WHERE name = 'sparrow.webhook.health_changed' AND (schema IS NULL OR schema = '{}'::jsonb);

UPDATE event_registrations
SET sample_payload = '{
  "webhook_id": "11111111-1111-1111-1111-111111111111",
  "consumer": "default",
  "url": "https://example.com/webhooks/inbound",
  "old_health": "healthy",
  "new_health": "unhealthy"
}'
WHERE name = 'sparrow.webhook.health_changed' AND (sample_payload IS NULL OR sample_payload = '{}'::jsonb);

UPDATE event_registrations
SET schema = '{
  "type": "object",
  "required": ["webhook_id", "consumer", "url", "delivery_id", "event_id", "attempt", "error_category", "error_message"],
  "properties": {
    "webhook_id": {"type": "string", "format": "uuid"},
    "consumer": {"type": "string"},
    "url": {"type": "string"},
    "delivery_id": {"type": "string", "format": "uuid"},
    "event_id": {"type": "string", "format": "uuid"},
    "attempt": {"type": "integer"},
    "error_category": {"type": "string"},
    "error_message": {"type": "string"},
    "alert_recipients": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {"email": {"type": "string", "format": "email"}}
      }
    }
  }
}'
WHERE name = 'sparrow.webhook.delivery_failed' AND (schema IS NULL OR schema = '{}'::jsonb);

UPDATE event_registrations
SET sample_payload = '{
  "webhook_id": "11111111-1111-1111-1111-111111111111",
  "consumer": "default",
  "url": "https://example.com/webhooks/inbound",
  "delivery_id": "22222222-2222-2222-2222-222222222222",
  "event_id": "33333333-3333-3333-3333-333333333333",
  "attempt": 5,
  "error_category": "timeout",
  "error_message": "request timed out after 30s"
}'
WHERE name = 'sparrow.webhook.delivery_failed' AND (sample_payload IS NULL OR sample_payload = '{}'::jsonb);
