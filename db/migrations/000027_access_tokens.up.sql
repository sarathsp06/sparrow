-- One row per redeemed admin access link (internal/accesslink). The nonce is
-- the link's single-use identifier; rows are purged once the link has expired
-- (it could no longer verify), so the table stays tiny.
CREATE TABLE access_link_redemptions (
    nonce VARCHAR(64) PRIMARY KEY,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    redeemed_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_access_link_redemptions_expires_at ON access_link_redemptions(expires_at);

COMMENT ON TABLE access_link_redemptions IS 'Redeemed one-time admin access links; rows live only until the link expires.';
