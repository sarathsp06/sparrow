-- Add Ed25519 private key column for asymmetric webhook signing.
-- The private key is envelope-encrypted (AES-256-GCM) like webhook_secret.
-- The public key is derived at runtime from the private key.
ALTER TABLE webhook_registrations
    ADD COLUMN ed25519_private_key BYTEA;
