BEGIN;

DROP INDEX IF EXISTS idx_tenants_external_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS external_id;

COMMIT;
