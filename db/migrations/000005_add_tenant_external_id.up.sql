-- Add external_id column to tenants for mapping identity provider IDs
-- (e.g., Clerk org_id, Auth0 org_id) to internal tenant UUIDs.
ALTER TABLE tenants
    ADD COLUMN external_id TEXT;

CREATE UNIQUE INDEX idx_tenants_external_id ON tenants(external_id) WHERE external_id IS NOT NULL;

COMMENT ON COLUMN tenants.external_id IS 'External identity provider ID (e.g., Clerk org_id). Used to map OIDC tokens to tenants.';
