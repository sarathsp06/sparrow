-- Track which identity provider user created each tenant.
-- This is the JWT "sub" claim value, used to enforce per-user tenant limits.
ALTER TABLE tenants
    ADD COLUMN created_by TEXT;

CREATE INDEX idx_tenants_created_by ON tenants(created_by) WHERE created_by IS NOT NULL;

COMMENT ON COLUMN tenants.created_by IS 'Identity provider user ID (JWT sub claim) of the user who created this tenant. Used for per-user tenant limits.';
