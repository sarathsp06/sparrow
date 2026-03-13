DROP INDEX IF EXISTS idx_tenants_created_by;
ALTER TABLE tenants DROP COLUMN IF EXISTS created_by;
