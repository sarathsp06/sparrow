-- Add signature_type column to webhook_registrations.
-- Controls which signing scheme is used for deliveries: 'hmac' (default) or 'ed25519'.
ALTER TABLE webhook_registrations
    ADD COLUMN signature_type TEXT NOT NULL DEFAULT 'hmac'
    CONSTRAINT chk_signature_type CHECK (signature_type IN ('hmac', 'ed25519'));
