-- Rollback initial schema
DROP TRIGGER IF EXISTS update_webhook_registrations_updated_at ON webhook_registrations;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE IF EXISTS webhook_deliveries;
DROP TYPE IF EXISTS webhook_delivery_status;
DROP TABLE IF EXISTS event_records;
DROP TABLE IF EXISTS webhook_registrations;