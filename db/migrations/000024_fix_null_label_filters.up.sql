-- Subscriptions written with a nil label-filter map were stored as jsonb
-- null, which never matches the delivery lookup predicate
-- (label_filters = '{}' OR label_filters <@ labels), silently dropping
-- all deliveries for those subscriptions. Normalize to the empty object.
UPDATE event_subscriptions
SET label_filters = '{}'::jsonb
WHERE label_filters = 'null'::jsonb;
