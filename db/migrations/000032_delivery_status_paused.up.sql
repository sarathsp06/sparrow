-- A delivery created while its subscription is paused. It is never attempted
-- until it is retried. Kept in its own migration: Postgres cannot use an enum
-- value in the transaction that adds it.
ALTER TYPE webhook_delivery_status ADD VALUE IF NOT EXISTS 'paused';
