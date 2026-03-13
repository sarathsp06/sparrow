-- Rollback consolidated schema

-- Drop triggers
DROP TRIGGER IF EXISTS webhook_registration_change_notification_trigger ON webhook_registrations;

DROP TRIGGER IF EXISTS update_event_registrations_updated_at ON event_registrations;
DROP TRIGGER IF EXISTS update_webhook_registrations_updated_at ON webhook_registrations;

-- Drop functions
DROP FUNCTION IF EXISTS notify_webhook_registration_change();

DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in dependency order
DROP TABLE IF EXISTS webhook_health_summaries;
DROP TABLE IF EXISTS webhook_health_state;
DROP TABLE IF EXISTS webhook_health_events;
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS event_subscriptions;
DROP TABLE IF EXISTS event_records;
DROP TABLE IF EXISTS webhook_registrations;
DROP TABLE IF EXISTS event_registrations;

-- Drop types
DROP TYPE IF EXISTS webhook_delivery_status;