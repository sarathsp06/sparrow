-- Add secret_headers column for encrypted header storage (AES-256-GCM ciphertext)
ALTER TABLE webhook_registrations ADD COLUMN secret_headers BYTEA;
