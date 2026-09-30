-- Postgres cannot drop an enum value. Move rows off it so the value is unused.
UPDATE webhook_deliveries SET status = 'failed', error_message = 'subscription was paused' WHERE status = 'paused';
