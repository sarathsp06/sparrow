-- Synthetic data for the SQL benchmark (scripts/sqlbench/run.sh).
--
-- Generates a large, skewed dataset on a freshly migrated database; no data is
-- committed to the repo. Volumes scale with the psql variable `scale`
-- (default 1 = 2M events, 6M deliveries, 8M health events, ~9 GB):
--
--   psql "$DATABASE_URL" -v scale=0.1 -f scripts/sqlbench/seed.sql
--
-- Bounded tables (webhooks, subscriptions, event types, tokens) keep fixed
-- sizes; only the tables that grow with traffic scale. Skew is deliberate:
-- the first webhook (by created_at) carries ~10% of deliveries and
-- consumer_0001 ~25-30% of traffic, consumer_0150 is mid-sized and
-- consumer_0252 owns webhooks but no deliveries. Those are the shapes
-- internal/webhooks/store/sqlbench_test.go benchmarks.

\set ON_ERROR_STOP on
\if :{?scale}
\else
  \set scale 1
\endif
SELECT set_config('sqlbench.scale', :'scale', false);

SET synchronous_commit = off;
SET client_min_messages = warning;

\timing on

----------------------------------------------------------------------
-- 0. CLEANUP existing data (idempotent re-runs)
----------------------------------------------------------------------
TRUNCATE tenants CASCADE;
TRUNCATE access_tokens CASCADE;
TRUNCATE access_invites CASCADE;
TRUNCATE river_job CASCADE;
TRUNCATE webhook_health_evaluator;

----------------------------------------------------------------------
-- 1. TENANTS (5 total: default + 4 others)
----------------------------------------------------------------------
INSERT INTO tenants (id, name, slug, status, settings) VALUES
  ('00000000-0000-0000-0000-000000000001', 'Default Tenant', 'default', 'active', '{"max_events_per_day": 100000}'),
  ('00000000-0000-0000-0000-000000000002', 'Acme Corp', 'acme', 'active', '{}'),
  ('00000000-0000-0000-0000-000000000003', 'Beta Labs', 'beta-labs', 'active', '{}'),
  ('00000000-0000-0000-0000-000000000004', 'Gamma Health', 'gamma-health', 'suspended', '{}'),
  ('00000000-0000-0000-0000-000000000005', 'Delta Payments', 'delta-payments', 'active', '{}');

----------------------------------------------------------------------
-- 2. HELPER: consumer names (~300 + _sparrow)
-- Zipf-ish: consumer_0001 through consumer_0300 plus _sparrow
-- The "hot" consumer will be consumer_0001 (~25% of traffic)
----------------------------------------------------------------------
-- We'll use a temporary table for consumer names with weights
CREATE TEMP TABLE consumer_pool (idx int, name varchar(255), weight real);
INSERT INTO consumer_pool
SELECT i, '_sparrow', 0.005 FROM generate_series(0,0) i
UNION ALL
SELECT i,
  'consumer_' || lpad(i::text, 4, '0'),
  CASE
    WHEN i = 1 THEN 25.0    -- hot consumer
    WHEN i <= 5 THEN 8.0    -- top-5
    WHEN i <= 20 THEN 2.5   -- top-20
    WHEN i <= 50 THEN 1.0   -- medium
    ELSE 0.15               -- long tail
  END
FROM generate_series(1, 300) i;

-- Normalize weights (used later via random())
-- Total weight ~ 25 + 4*8 + 15*2.5 + 30*1 + 250*0.15 = 25+32+37.5+30+37.5 = 162
-- consumer_0001 share: 25/162 ~ 15.4%. Close enough to ~25% after Zipf effect on events.

----------------------------------------------------------------------
-- 3. EVENT REGISTRATIONS (~400 event types, 3 versions each)
----------------------------------------------------------------------
-- Regular event types for the default tenant
INSERT INTO event_registrations (tenant_id, name, description, schema, sample_payload, metadata, active, version, created_at, updated_at)
SELECT
  '00000000-0000-0000-0000-000000000001'::uuid,
  CASE
    WHEN i <= 10 THEN 'sparrow.' || (ARRAY['webhook.created','webhook.updated','webhook.deleted','webhook.disabled','webhook.enabled','delivery.failed','delivery.succeeded','event.received','health_changed','subscription.created'])[i]
    ELSE (ARRAY['order','payment','user','invoice','shipment','notification','ticket','alert','report','sync','inventory','customer','product','billing','subscription','account','session','transaction','message','task'])[((i-11) % 20) + 1]
      || '.'
      || (ARRAY['created','updated','deleted','completed','failed','cancelled','processed','pending','approved','rejected'])[((i-11) / 20 % 10) + 1]
      || CASE WHEN i > 210 THEN '.v' || ((i-210) % 5 + 2)::text ELSE '' END
  END,
  'Auto-registered event type #' || i,
  CASE WHEN i % 3 = 0 THEN '{"type":"object","properties":{"id":{"type":"string"},"timestamp":{"type":"string","format":"date-time"}}}' ELSE NULL END,
  jsonb_build_object('id', 'evt_sample_' || i, 'timestamp', '2026-09-15T12:00:00Z', 'data', jsonb_build_object('key', 'value')),
  '{}'::jsonb,
  CASE WHEN i % 50 = 0 THEN false ELSE true END,
  3, -- current version
  now() - interval '90 days' + (i * interval '4 hours'),
  now() - interval '30 days' + (i * interval '1 hour')
FROM generate_series(1, 400) i;

-- A handful for other tenants
INSERT INTO event_registrations (tenant_id, name, description, active, version)
SELECT t.id, 'order.created', 'Order creation', true, 1
FROM tenants t WHERE t.id != '00000000-0000-0000-0000-000000000001'::uuid;

-- Event registration versions (~3 per type = ~1200 rows)
INSERT INTO event_registration_versions (tenant_id, name, version, description, schema, sample_payload, created_at)
SELECT
  er.tenant_id, er.name, v,
  er.description || ' v' || v,
  CASE WHEN v = er.version THEN er.schema ELSE NULL END,
  er.sample_payload,
  er.created_at + ((v-1) * interval '10 days')
FROM event_registrations er, generate_series(1, 3) v
WHERE er.tenant_id = '00000000-0000-0000-0000-000000000001'::uuid;

-- Single version for other tenants
INSERT INTO event_registration_versions (tenant_id, name, version, description, created_at)
SELECT er.tenant_id, er.name, 1, er.description, er.created_at
FROM event_registrations er
WHERE er.tenant_id != '00000000-0000-0000-0000-000000000001'::uuid;

\echo '>>> Event registrations done'

----------------------------------------------------------------------
-- 4. WEBHOOK REGISTRATIONS (~5000)
-- Mix: 80% active-healthy, 5% active-degraded, 5% active-unhealthy,
--      5% paused (active=false), 5% auto_disabled
-- "Hot" webhook = first one for consumer_0001 (~10% of deliveries)
----------------------------------------------------------------------
INSERT INTO webhook_registrations (
  id, consumer, url, headers, timeout, active, description, health,
  max_retries, retry_backoff_seconds, rate_limit_rps,
  signature_type, requires_transform, content_type,
  auto_disabled_at, auto_disabled_reason,
  tenant_id, created_at, updated_at
)
SELECT
  gen_random_uuid(),
  -- Zipf-skewed consumer assignment
  CASE
    WHEN i = 1 THEN 'consumer_0001'  -- THE hot webhook
    WHEN i <= 50 THEN 'consumer_0001'
    WHEN i <= 150 THEN 'consumer_' || lpad(((i % 4) + 2)::text, 4, '0')  -- top-5 consumers
    WHEN i <= 500 THEN 'consumer_' || lpad(((i % 15) + 6)::text, 4, '0')  -- top-20
    WHEN i <= 1500 THEN 'consumer_' || lpad(((i % 30) + 21)::text, 4, '0')  -- top-50
    WHEN i <= 2500 THEN 'consumer_' || lpad(((i % 100) + 51)::text, 4, '0')
    WHEN i <= 4900 THEN 'consumer_' || lpad(((i % 150) + 151)::text, 4, '0')
    ELSE '_sparrow'
  END,
  'https://hooks.example.com/webhook/' || i || '/receive',
  jsonb_build_object('X-Custom-Header', 'value-' || i),
  CASE WHEN i % 10 = 0 THEN 60 ELSE 30 END,
  -- active: 90% true, 10% false (5% paused + 5% auto_disabled)
  CASE WHEN i % 20 IN (18, 19) THEN false ELSE true END,
  'Webhook endpoint #' || i || ' for integration testing',
  CASE
    WHEN i % 20 IN (18, 19) THEN 'unknown'
    WHEN i % 20 = 16 THEN 'degraded'
    WHEN i % 20 = 17 THEN 'unhealthy'
    ELSE 'healthy'
  END,
  CASE WHEN i % 7 = 0 THEN 5 ELSE 3 END,
  CASE WHEN i % 5 = 0 THEN 120 ELSE 60 END,
  CASE WHEN i % 10 = 0 THEN (i % 50 + 1)::real ELSE NULL END,
  CASE WHEN i % 15 = 0 THEN 'ed25519' ELSE 'hmac' END,
  CASE WHEN i % 8 = 0 THEN true ELSE false END,
  'application/json',
  CASE WHEN i % 20 = 19 THEN now() - interval '2 days' ELSE NULL END,
  CASE WHEN i % 20 = 19 THEN 'consecutive failures exceeded threshold (50 over 48h)' ELSE NULL END,
  '00000000-0000-0000-0000-000000000001'::uuid,
  now() - interval '120 days' + (i * interval '30 minutes'),
  now() - interval '1 day' + (i * interval '10 seconds')
FROM generate_series(1, 5000) i;

\echo '>>> Webhook registrations done'

-- Materialize webhook IDs for FK references
CREATE TEMP TABLE wh_ids AS
SELECT id AS webhook_id, consumer, row_number() OVER (ORDER BY created_at) AS rn
FROM webhook_registrations
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'::uuid;

CREATE INDEX ON wh_ids(rn);

-- The "hot" webhook is rn=1
-- Grab it for later reference
CREATE TEMP TABLE hot_webhook AS SELECT webhook_id FROM wh_ids WHERE rn = 1;

----------------------------------------------------------------------
-- 4a. WEBHOOK HEALTH STATE (one per webhook)
----------------------------------------------------------------------
INSERT INTO webhook_health_state (webhook_id, consecutive_failures, last_success_at, last_failure_at, last_event_at, failing_since)
SELECT
  w.webhook_id,
  CASE
    WHEN wr.health = 'healthy' THEN 0
    WHEN wr.health = 'degraded' THEN (w.rn % 5) + 1
    WHEN wr.health = 'unhealthy' THEN (w.rn % 20) + 10
    ELSE 0
  END,
  CASE WHEN wr.health != 'unhealthy' THEN now() - interval '1 hour' * (w.rn % 24) ELSE now() - interval '3 days' END,
  CASE WHEN wr.health != 'healthy' THEN now() - interval '30 minutes' * (w.rn % 48) ELSE NULL END,
  now() - interval '1 minute' * (w.rn % 60),
  CASE WHEN wr.health = 'unhealthy' THEN now() - interval '2 days' ELSE NULL END
FROM wh_ids w
JOIN webhook_registrations wr ON wr.id = w.webhook_id;

\echo '>>> Webhook health state done'

----------------------------------------------------------------------
-- 4c. WEBHOOK RATE LIMIT STATE (webhooks with rate_limit_rps)
----------------------------------------------------------------------
INSERT INTO webhook_rate_limit_state (webhook_id, next_delivery_at)
SELECT w.webhook_id, now() - interval '1 second' * (w.rn % 60)
FROM wh_ids w
JOIN webhook_registrations wr ON wr.id = w.webhook_id
WHERE wr.rate_limit_rps IS NOT NULL;

----------------------------------------------------------------------
-- 4d. WEBHOOK ALERT CONFIGS (~500)
----------------------------------------------------------------------
INSERT INTO webhook_alert_configs (tenant_id, consumer, webhook_id, email, event_types)
SELECT
  '00000000-0000-0000-0000-000000000001'::uuid,
  wr.consumer,
  CASE WHEN w.rn % 3 = 0 THEN w.webhook_id ELSE NULL END,
  'alerts+' || w.rn || '@example.com',
  ARRAY['delivery_failed', 'health_changed']
FROM wh_ids w
JOIN webhook_registrations wr ON wr.id = w.webhook_id
WHERE w.rn <= 500;

\echo '>>> Webhook auxiliary tables done'

----------------------------------------------------------------------
-- 5. EVENT SUBSCRIPTIONS (~15,000)
-- ~3 subscriptions per webhook on average
----------------------------------------------------------------------
-- Gather event names for assignment
CREATE TEMP TABLE evt_names AS
SELECT name, row_number() OVER (ORDER BY name) AS rn, count(*) OVER () AS total
FROM event_registrations
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
  AND name NOT LIKE 'sparrow.%';

INSERT INTO event_subscriptions (
  id, webhook_id, event_name, consumer, headers, method,
  transform_enabled, transform_template,
  timeout, tenant_id, label_filters,
  on_transform_error, template_missing_key,
  paused_at, paused_reason,
  created_at, updated_at
)
SELECT
  gen_random_uuid(),
  w.webhook_id,
  en.name,
  wr.consumer,
  CASE WHEN s % 5 = 0 THEN '{"X-Sub-Header": "sub-val"}'::jsonb ELSE '{}'::jsonb END,
  'POST',
  CASE WHEN s % 4 = 0 THEN true ELSE false END,
  CASE WHEN s % 4 = 0 THEN '{{ .payload | toJson }}' ELSE '' END,
  CASE WHEN s % 7 = 0 THEN 60 ELSE 30 END,
  '00000000-0000-0000-0000-000000000001'::uuid,
  CASE
    WHEN s % 10 = 0 THEN jsonb_build_object('env', 'production')
    WHEN s % 15 = 0 THEN jsonb_build_object('region', 'us-east-1')
    ELSE '{}'::jsonb
  END,
  CASE WHEN s % 20 = 0 THEN 'fallback' ELSE 'fail' END,
  CASE WHEN s % 12 = 0 THEN 'zero' ELSE 'error' END,
  CASE WHEN s % 30 = 0 THEN now() - interval '5 days' ELSE NULL END,
  CASE WHEN s % 30 = 0 THEN 'Paused for maintenance' ELSE NULL END,
  now() - interval '100 days' + (s * interval '8 minutes'),
  now() - interval '2 days' + (s * interval '5 seconds')
FROM wh_ids w
JOIN webhook_registrations wr ON wr.id = w.webhook_id
CROSS JOIN generate_series(1, 3) s
JOIN evt_names en ON en.rn = ((w.rn * 3 + s) % en.total) + 1
WHERE wr.consumer != '_sparrow'
ON CONFLICT (tenant_id, webhook_id, event_name, consumer) DO NOTHING;

-- _sparrow subscriptions (sparrow.* events)
INSERT INTO event_subscriptions (
  webhook_id, event_name, consumer, tenant_id, method, transform_template, timeout, headers
)
SELECT w.webhook_id, er.name, '_sparrow', '00000000-0000-0000-0000-000000000001'::uuid, 'POST', '', 30, '{}'::jsonb
FROM wh_ids w
JOIN webhook_registrations wr ON wr.id = w.webhook_id
CROSS JOIN event_registrations er
WHERE wr.consumer = '_sparrow'
  AND er.name LIKE 'sparrow.%'
  AND er.tenant_id = '00000000-0000-0000-0000-000000000001'::uuid
ON CONFLICT (tenant_id, webhook_id, event_name, consumer) DO NOTHING;

\echo '>>> Event subscriptions done'

----------------------------------------------------------------------
-- 5a. SUBSCRIPTION TEMPLATE VERSIONS (~60k total)
-- Up to 20 versions for subscriptions with transform_enabled
----------------------------------------------------------------------
CREATE TEMP TABLE sub_ids AS
SELECT id AS subscription_id, tenant_id, transform_enabled,
       row_number() OVER (ORDER BY created_at) AS rn
FROM event_subscriptions
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'::uuid;

INSERT INTO subscription_template_versions (
  tenant_id, subscription_id, template, source, notes, saved_by, created_at
)
SELECT
  s.tenant_id, s.subscription_id,
  '{{ .payload | toJson }}  /* version ' || v || ' */',
  CASE WHEN v % 3 = 0 THEN 'ai_draft' ELSE 'manual' END,
  CASE WHEN v = vers THEN 'Latest save' ELSE 'Historical version ' || v END,
  CASE WHEN v % 2 = 0 THEN 'admin@example.com' ELSE 'dev@example.com' END,
  now() - interval '60 days' + (v * interval '3 days')
FROM sub_ids s
CROSS JOIN LATERAL (
  SELECT CASE WHEN s.transform_enabled THEN LEAST(20, 5 + (s.rn % 16)) ELSE 1 END AS vers
) calc
CROSS JOIN generate_series(1, calc.vers) v;

DROP TABLE sub_ids;

\echo '>>> Template versions done'

----------------------------------------------------------------------
-- 6. EVENT RECORDS (2,000,000 x scale over 30 days)
-- Zipf-skewed by consumer: consumer_0001 gets ~25% of events
----------------------------------------------------------------------
-- We'll insert in batches of 500k for speed
DO $$
DECLARE
  total int := (2000000 * current_setting('sqlbench.scale')::numeric)::int;
  batch_size int := LEAST(500000, GREATEST(total, 1));
  i int := 0;
BEGIN
  WHILE i < total LOOP
    INSERT INTO event_records (
      id, consumer, event, payload, ttl, metadata, labels,
      created_at, expires_at, tenant_id, schema_valid,
      idempotency_key, event_version
    )
    SELECT
      gen_random_uuid(),
      -- Zipf consumer: ~25% consumer_0001, rest spread
      CASE
        WHEN random() < 0.25 THEN 'consumer_0001'
        WHEN random() < 0.15 THEN 'consumer_' || lpad(((g % 4) + 2)::text, 4, '0')
        WHEN random() < 0.30 THEN 'consumer_' || lpad(((g % 15) + 6)::text, 4, '0')
        WHEN random() < 0.40 THEN 'consumer_' || lpad(((g % 30) + 21)::text, 4, '0')
        ELSE 'consumer_' || lpad(((g % 250) + 51)::text, 4, '0')
      END,
      -- Event name spread across registered types
      (ARRAY['order.created','payment.completed','user.updated','invoice.created',
             'shipment.pending','notification.created','ticket.processed','alert.failed',
             'report.completed','sync.updated','inventory.created','customer.updated',
             'product.deleted','billing.approved','subscription.created','account.rejected',
             'session.created','transaction.completed','message.processed','task.pending']
      )[(g % 20) + 1],
      -- Realistic JSON payload 0.5–2 KB
      jsonb_build_object(
        'id', 'evt_' || (i + g)::text,
        'type', 'webhook_event',
        'timestamp', (now() - interval '30 days' + ((i + g) * interval '1.3 seconds' / current_setting('sqlbench.scale')::numeric))::text,
        'data', jsonb_build_object(
          'entity_id', 'ent_' || ((i + g) % 100000)::text,
          'action', (ARRAY['create','update','delete','process','approve'])[((i+g) % 5) + 1],
          'attributes', jsonb_build_object(
            'name', 'Entity ' || ((i+g) % 10000)::text,
            'status', (ARRAY['active','pending','completed','failed','cancelled'])[((i+g) % 5) + 1],
            'amount', round((random() * 10000)::numeric, 2),
            'currency', 'USD',
            'metadata', jsonb_build_object('source', 'api', 'version', '2.1', 'region', (ARRAY['us-east-1','us-west-2','eu-west-1','ap-southeast-1'])[((i+g) % 4) + 1])
          )
        )
      )::text,
      86400 * (7 + (g % 24)), -- TTL: 7-30 days in seconds
      jsonb_build_object('source', 'api', 'ip', '10.0.' || (g % 256) || '.' || ((g/256) % 256)),
      CASE
        WHEN g % 10 = 0 THEN jsonb_build_object('env', 'staging', 'region', 'us-west-2')
        WHEN g % 7 = 0 THEN jsonb_build_object('env', 'production')
        ELSE '{}'::jsonb
      END,
      now() - interval '30 days' + ((i + g) * interval '1.3 seconds' / current_setting('sqlbench.scale')::numeric),
      now() + interval '30 days' + (g * interval '30 seconds'),
      '00000000-0000-0000-0000-000000000001'::uuid,
      CASE WHEN g % 100 = 0 THEN false ELSE true END,
      CASE WHEN g % 20 = 0 THEN 'idem_' || (i + g)::text ELSE NULL END,
      CASE WHEN g % 3 = 0 THEN 3 WHEN g % 3 = 1 THEN 2 ELSE 1 END
    FROM generate_series(1, batch_size) g;

    i := i + batch_size;
    RAISE NOTICE 'Inserted % / % event records', i, total;
  END LOOP;
END $$;

\echo '>>> Event records done'

----------------------------------------------------------------------
-- 7. WEBHOOK DELIVERIES (6,000,000 x scale over 30 days)
-- Status: 88% success, 5% failed, 3% retrying, 2% queued/pending, 2% paused
-- Hot webhook rn=1 gets ~10%, hot consumer ~25%
----------------------------------------------------------------------
-- Materialize event IDs and subscription IDs for FK references
CREATE TEMP TABLE evt_ids AS
SELECT id AS event_id, consumer, row_number() OVER (ORDER BY created_at) AS rn
FROM event_records
WHERE tenant_id = '00000000-0000-0000-0000-000000000001'::uuid;

CREATE INDEX ON evt_ids(rn);
CREATE INDEX ON evt_ids(consumer);

-- Subscription lookup: subscription for a given webhook
CREATE TEMP TABLE sub_lookup AS
SELECT es.id AS subscription_id, es.webhook_id, es.consumer,
       row_number() OVER (PARTITION BY es.webhook_id ORDER BY es.created_at) AS sub_rn
FROM event_subscriptions es
WHERE es.tenant_id = '00000000-0000-0000-0000-000000000001'::uuid;

CREATE INDEX ON sub_lookup(webhook_id, sub_rn);

\echo '>>> Temp tables for deliveries ready'

-- Insert deliveries in batches of 1M
DO $$
DECLARE
  total int := (6000000 * current_setting('sqlbench.scale')::numeric)::int;
  batch_size int := LEAST(1000000, GREATEST(total, 1));
  i int := 0;
  hot_wh uuid;
  total_wh int;
  total_evt int;
BEGIN
  SELECT webhook_id INTO hot_wh FROM hot_webhook LIMIT 1;
  SELECT count(*) INTO total_wh FROM wh_ids;
  SELECT count(*) INTO total_evt FROM evt_ids;

  WHILE i < total LOOP
    INSERT INTO webhook_deliveries (
      id, webhook_id, event_id, subscription_id,
      status, attempt_count, max_attempts,
      request_body, created_at, last_attempted_at, next_retry_at,
      expires_at, response_code, response_body,
      error_message, error_category, template_error
    )
    SELECT
      gen_random_uuid(),
      w.webhook_id,
      e.event_id,
      sl.subscription_id,
      -- Status distribution
      CASE
        WHEN g % 100 < 88 THEN 'success'
        WHEN g % 100 < 93 THEN 'failed'
        WHEN g % 100 < 96 THEN 'retrying'
        WHEN g % 100 < 98 THEN 'pending'
        ELSE 'paused'
      END::webhook_delivery_status,
      -- attempt_count
      CASE
        WHEN g % 100 < 88 THEN 1
        WHEN g % 100 < 93 THEN 3
        WHEN g % 100 < 96 THEN (g % 3) + 1
        ELSE 0
      END,
      3, -- max_attempts
      -- request_body (~0.5-1 KB)
      '{"event_id":"' || e.event_id::text || '","webhook_id":"' || w.webhook_id::text || '","payload":{"data":{"id":"ent_' || (g % 100000)::text || '","action":"process","ts":"' || (now() - interval '30 days' + ((i + g) * interval '0.43 seconds' / current_setting('sqlbench.scale')::numeric))::text || '"}}}',
      -- created_at: spread over 30 days
      now() - interval '30 days' + ((i + g) * interval '0.43 seconds' / current_setting('sqlbench.scale')::numeric),
      -- last_attempted_at
      CASE WHEN g % 100 < 98 THEN now() - interval '30 days' + ((i + g) * interval '0.43 seconds' / current_setting('sqlbench.scale')::numeric) + interval '2 seconds' ELSE NULL END,
      -- next_retry_at
      CASE WHEN g % 100 BETWEEN 93 AND 95 THEN now() + interval '1 minute' * (g % 30) ELSE NULL END,
      -- expires_at
      now() + interval '7 days' + (g * interval '1 second'),
      -- response_code
      CASE
        WHEN g % 100 < 88 THEN (ARRAY[200, 201, 202, 204])[((g / 100) % 4) + 1]
        WHEN g % 100 < 91 THEN (ARRAY[500, 502, 503])[((g / 100) % 3) + 1]
        WHEN g % 100 < 93 THEN (ARRAY[400, 401, 403, 404, 422])[((g / 100) % 5) + 1]
        WHEN g % 100 < 96 THEN (ARRAY[500, 502, 503, 504])[((g / 100) % 4) + 1]
        ELSE 0
      END,
      -- response_body
      CASE WHEN g % 100 < 88 THEN '{"status":"ok"}' ELSE '' END,
      -- error_message
      CASE
        WHEN g % 100 < 88 THEN ''
        WHEN g % 100 < 91 THEN 'server returned 5xx'
        WHEN g % 100 < 93 THEN 'client error: invalid request'
        WHEN g % 100 < 96 THEN 'connection timeout after 30s'
        ELSE ''
      END,
      -- error_category
      CASE
        WHEN g % 100 < 88 THEN 'success'
        WHEN g % 100 < 91 THEN 'server_error'
        WHEN g % 100 < 93 THEN 'client_error'
        WHEN g % 100 < 95 THEN 'timeout'
        WHEN g % 100 < 96 THEN 'network_error'
        ELSE ''
      END,
      CASE WHEN g % 200 = 0 THEN 'template execution failed: missing key "id"' ELSE NULL END
    FROM generate_series(1, batch_size) g
    -- Zipf-skewed webhook assignment: hot webhook gets ~10%
    JOIN wh_ids w ON w.rn = CASE
      WHEN g % 10 = 0 THEN 1  -- hot webhook = 10%
      WHEN g % 100 < 25 THEN (g % 49) + 2  -- top-50 webhooks
      ELSE (g % total_wh) + 1
    END
    -- Event: pick from matching consumer or just any
    JOIN evt_ids e ON e.rn = ((i + g) % total_evt) + 1
    -- Subscription: first subscription of this webhook (nullable)
    LEFT JOIN sub_lookup sl ON sl.webhook_id = w.webhook_id AND sl.sub_rn = 1;

    i := i + batch_size;
    RAISE NOTICE 'Inserted % / % deliveries', i, total;
  END LOOP;
END $$;

\echo '>>> Deliveries done'

----------------------------------------------------------------------
-- 8. WEBHOOK HEALTH EVENTS (8,000,000 x scale over 7 days)
-- Append-only attempt rows linked to deliveries and webhooks
----------------------------------------------------------------------
-- We need delivery IDs. Rather than joining 6M deliveries, we'll
-- generate health events that reference real webhook_ids and
-- real delivery_ids from the last 7 days of deliveries.

CREATE TEMP TABLE recent_delivery_ids AS
SELECT id AS delivery_id, webhook_id, row_number() OVER (ORDER BY created_at DESC) AS rn
FROM webhook_deliveries
WHERE created_at >= now() - interval '7 days'
LIMIT (4000000 * :scale)::bigint;  -- ~2/3 of deliveries fall in the last 7 days

CREATE INDEX ON recent_delivery_ids(rn);

\echo '>>> Recent delivery IDs indexed'

DO $$
DECLARE
  total int := (8000000 * current_setting('sqlbench.scale')::numeric)::int;
  batch_size int := LEAST(2000000, GREATEST(total, 1));
  i int := 0;
  rd_count int;
BEGIN
  SELECT count(*) INTO rd_count FROM recent_delivery_ids;
  IF rd_count = 0 THEN rd_count := 1; END IF;

  WHILE i < total LOOP
    INSERT INTO webhook_health_events (
      webhook_id, delivery_id, success, response_time, response_code,
      error_message, error_category, "timestamp"
    )
    SELECT
      rd.webhook_id,
      rd.delivery_id,
      CASE WHEN g % 100 < 90 THEN true ELSE false END,
      50 + (g % 950), -- response time 50-999ms
      CASE
        WHEN g % 100 < 90 THEN 200
        WHEN g % 100 < 95 THEN (ARRAY[500, 502, 503])[((g/100) % 3) + 1]
        ELSE (ARRAY[400, 408, 429])[((g/100) % 3) + 1]
      END,
      CASE WHEN g % 100 >= 90 THEN 'delivery attempt failed' ELSE '' END,
      CASE
        WHEN g % 100 < 90 THEN 'success'
        WHEN g % 100 < 93 THEN 'server_error'
        WHEN g % 100 < 96 THEN 'timeout'
        WHEN g % 100 < 98 THEN 'client_error'
        ELSE 'network_error'
      END,
      now() - interval '7 days' + ((i + g) * interval '0.076 seconds' / current_setting('sqlbench.scale')::numeric)
    FROM generate_series(1, batch_size) g
    JOIN recent_delivery_ids rd ON rd.rn = ((i + g) % rd_count) + 1;

    i := i + batch_size;
    RAISE NOTICE 'Inserted % / % health events', i, total;
  END LOOP;
END $$;

\echo '>>> Health events done'

----------------------------------------------------------------------
-- 8a. WEBHOOK HEALTH EVALUATOR (watermark ~50k events from the end)
----------------------------------------------------------------------
INSERT INTO webhook_health_evaluator (singleton, watermark, updated_at)
SELECT true,
  (SELECT "timestamp" FROM webhook_health_events ORDER BY "timestamp" DESC OFFSET (50000 * :scale)::bigint LIMIT 1),
  now()
ON CONFLICT (singleton) DO UPDATE SET
  watermark = EXCLUDED.watermark,
  updated_at = now();

----------------------------------------------------------------------
-- 8b. WEBHOOK HEALTH BUCKETS (per-minute for last 24h, active webhooks)
-- ~2M+ rows
----------------------------------------------------------------------
INSERT INTO webhook_health_buckets (webhook_id, bucket_start, attempts, failures)
SELECT
  w.webhook_id,
  ts,
  8 + (w.rn % 12),  -- 8-19 attempts per minute
  CASE WHEN w.rn % 10 = 0 THEN (w.rn % 3) + 1 ELSE 0 END
FROM wh_ids w
CROSS JOIN generate_series(
  date_trunc('minute', now() - interval '24 hours'),
  date_trunc('minute', now()),
  interval '1 minute'
) ts
WHERE w.rn <= 1500  -- only active, healthy-ish webhooks
ON CONFLICT (webhook_id, bucket_start) DO NOTHING;

\echo '>>> Health buckets done'

----------------------------------------------------------------------
-- 8c. WEBHOOK METRICS (running counters, as migrations 038/040 seed them:
-- totals from health events up to the watermark, window from the buckets)
----------------------------------------------------------------------
INSERT INTO webhook_metrics (webhook_id, total_attempts, total_failures)
SELECT e.webhook_id, COUNT(*), COUNT(*) FILTER (WHERE NOT e.success)
FROM webhook_health_events e
WHERE e."timestamp" <= (SELECT watermark FROM webhook_health_evaluator WHERE singleton)
GROUP BY e.webhook_id
ON CONFLICT (webhook_id) DO NOTHING;

INSERT INTO webhook_metrics (webhook_id, window_attempts, window_failures)
SELECT b.webhook_id, SUM(b.attempts), SUM(b.failures)
FROM webhook_health_buckets b
WHERE b.bucket_start >= date_trunc('minute', (SELECT watermark FROM webhook_health_evaluator WHERE singleton)) - INTERVAL '24 hours'
GROUP BY b.webhook_id
ON CONFLICT (webhook_id) DO UPDATE SET
  window_attempts = EXCLUDED.window_attempts,
  window_failures = EXCLUDED.window_failures;

\echo '>>> Webhook metrics done'

----------------------------------------------------------------------
-- 9. BATCH JOBS (~2000)
----------------------------------------------------------------------
INSERT INTO batch_jobs (
  tenant_id, consumer, job_type, status, data,
  total, processed, failed, ttl_seconds,
  created_at, expires_at, updated_at
)
SELECT
  '00000000-0000-0000-0000-000000000001'::uuid,
  'consumer_' || lpad(((g % 50) + 1)::text, 4, '0'),
  CASE WHEN g % 2 = 0 THEN 'event_repush' ELSE 'delivery_retry' END,
  CASE
    WHEN g % 10 < 7 THEN 'completed'
    WHEN g % 10 < 8 THEN 'pending'
    WHEN g % 10 < 9 THEN 'processing'
    ELSE 'cancelled'
  END,
  -- JSONB data with snapshotted ID arrays (5-50 UUIDs)
  jsonb_build_object(
    'item_ids', (
      SELECT jsonb_agg(gen_random_uuid())
      FROM generate_series(1, 5 + (g % 46))
    ),
    'filter', jsonb_build_object('status', 'failed', 'event', 'order.created')
  ),
  5 + (g % 46),
  CASE WHEN g % 10 < 7 THEN 5 + (g % 46) ELSE g % 30 END,
  CASE WHEN g % 10 < 7 THEN g % 3 ELSE 0 END,
  900,
  now() - interval '30 days' + (g * interval '20 minutes'),
  now() - interval '30 days' + (g * interval '20 minutes') + interval '15 minutes',
  now() - interval '30 days' + (g * interval '20 minutes') + interval '10 minutes'
FROM generate_series(1, 2000) g;

\echo '>>> Batch jobs done'

----------------------------------------------------------------------
-- 10. ACCESS TOKENS (~20,000) and ACCESS INVITES (~5,000)
----------------------------------------------------------------------
-- Realm = default tenant ID string
INSERT INTO access_tokens (
  id, realm, scope, name, secret_hash,
  created_by, created_at, expires_at, revoked_at, last_used_at
)
SELECT
  'tok_' || lpad(g::text, 6, '0'),
  '00000000-0000-0000-0000-000000000001',
  CASE
    WHEN g % 5 = 0 THEN 'consumer_' || lpad(((g % 100) + 1)::text, 4, '0')  -- consumer-scoped
    ELSE NULL  -- tenant-wide
  END,
  'Token ' || g,
  sha256(('secret_' || g)::bytea),
  CASE WHEN g % 3 = 0 THEN 'admin@example.com' ELSE 'system' END,
  now() - interval '90 days' + (g * interval '6 minutes'),
  CASE WHEN g % 4 = 0 THEN now() + interval '30 days' ELSE NULL END,
  CASE WHEN g % 20 = 0 THEN now() - interval '5 days' ELSE NULL END,
  CASE WHEN g % 2 = 0 THEN now() - interval '1 hour' * (g % 48) ELSE NULL END
FROM generate_series(1, 20000) g;

INSERT INTO access_invites (
  id, realm, scope, name, secret_hash,
  token_ttl_seconds, created_by,
  created_at, expires_at, redeemed_at, cancelled_at, token_id
)
SELECT
  'inv_' || lpad(g::text, 5, '0'),
  '00000000-0000-0000-0000-000000000001',
  CASE WHEN g % 3 = 0 THEN 'consumer_' || lpad(((g % 50) + 1)::text, 4, '0') ELSE NULL END,
  'Invite ' || g,
  sha256(('invite_secret_' || g)::bytea),
  86400 * 30,
  'admin@example.com',
  now() - interval '60 days' + (g * interval '15 minutes'),
  now() + interval '30 days',
  CASE WHEN g % 4 = 0 THEN now() - interval '30 days' + (g * interval '15 minutes') ELSE NULL END,
  CASE WHEN g % 10 = 0 THEN now() - interval '10 days' ELSE NULL END,
  CASE WHEN g % 4 = 0 THEN 'tok_' || lpad(((g * 4) % 20000 + 1)::text, 6, '0') ELSE NULL END
FROM generate_series(1, 5000) g;

\echo '>>> Access tokens and invites done'

----------------------------------------------------------------------
-- 11. RIVER JOBS (~300,000 x scale)
-- Mix: 90% completed, 5% cancelled, 5% discarded. All finished: a seeded job in
-- a live state (available, retryable, running) has made-up args, and a server
-- started on this database (run.sh --e2e) would spend its workers on them.
-- Kinds: event_processing, webhook_delivery, batch_processing,
--        health_evaluate, batch_cleanup, retention, token_purge
----------------------------------------------------------------------
-- Need to reset the sequence first
SELECT setval('river_job_id_seq', 1, false);

INSERT INTO river_job (
  state, attempt, max_attempts, attempted_at,
  created_at, finalized_at, scheduled_at,
  priority, args, attempted_by, kind, queue, tags
)
SELECT
  CASE
    WHEN g % 100 < 90 THEN 'completed'
    WHEN g % 100 < 95 THEN 'cancelled'
    ELSE 'discarded'
  END::river_job_state,
  1, -- attempt
  CASE WHEN kind_name = 'webhook_delivery' THEN 3 ELSE 5 END,
  now() - interval '7 days' + (g * interval '2 seconds'),
  now() - interval '7 days' + (g * interval '2 seconds'),
  now() - interval '7 days' + (g * interval '2 seconds') + interval '1 second',
  now() - interval '7 days' + (g * interval '2 seconds'),
  1,
  -- args JSONB
  CASE kind_name
    WHEN 'event_processing' THEN jsonb_build_object(
      'tenant_id', '00000000-0000-0000-0000-000000000001',
      'event_id', gen_random_uuid()::text,
      'consumer', 'consumer_' || lpad(((g % 50) + 1)::text, 4, '0'),
      'event', 'order.created',
      'ttl_seconds', 604800,
      'metadata', '{}'::jsonb,
      'labels', '{}'::jsonb,
      'created_at', (now() - interval '7 days' + (g * interval '2 seconds'))::text
    )
    WHEN 'webhook_delivery' THEN jsonb_build_object(
      'tenant_id', '00000000-0000-0000-0000-000000000001',
      'delivery_id', gen_random_uuid()::text,
      'webhook_id', gen_random_uuid()::text,
      'subscription_id', gen_random_uuid()::text,
      'event_id', gen_random_uuid()::text,
      'expires_at', (now() + interval '7 days')::text,
      'consumer', 'consumer_' || lpad(((g % 50) + 1)::text, 4, '0'),
      'max_attempts', 3
    )
    WHEN 'batch_processing' THEN jsonb_build_object(
      'tenant_id', '00000000-0000-0000-0000-000000000001',
      'batch_id', gen_random_uuid()::text
    )
    ELSE '{}'::jsonb
  END,
  ARRAY['worker-1'],
  kind_name,
  CASE kind_name
    WHEN 'event_processing' THEN 'events'
    WHEN 'webhook_delivery' THEN 'webhooks'
    WHEN 'batch_processing' THEN 'batch_jobs'
    ELSE 'default'
  END,
  ARRAY[]::varchar[]
FROM generate_series(1, (300000 * :scale)::int) g
CROSS JOIN LATERAL (
  SELECT CASE
    WHEN g % 100 < 45 THEN 'event_processing'
    WHEN g % 100 < 85 THEN 'webhook_delivery'
    WHEN g % 100 < 90 THEN 'batch_processing'
    WHEN g % 100 < 93 THEN 'health_evaluate'
    WHEN g % 100 < 96 THEN 'batch_cleanup'
    WHEN g % 100 < 98 THEN 'retention'
    ELSE 'token_purge'
  END AS kind_name
) k;

\echo '>>> River jobs done'

----------------------------------------------------------------------
-- 12. RIVER INFRASTRUCTURE TABLES
----------------------------------------------------------------------
-- river_queue
INSERT INTO river_queue (name, updated_at) VALUES
  ('default', now()),
  ('events', now()),
  ('webhooks', now()),
  ('batch_jobs', now())
ON CONFLICT DO NOTHING;


----------------------------------------------------------------------
-- 13. CLEANUP TEMP TABLES
----------------------------------------------------------------------
DROP TABLE IF EXISTS consumer_pool;
DROP TABLE IF EXISTS evt_names;
DROP TABLE IF EXISTS wh_ids;
DROP TABLE IF EXISTS hot_webhook;
DROP TABLE IF EXISTS evt_ids;
DROP TABLE IF EXISTS sub_lookup;
DROP TABLE IF EXISTS recent_delivery_ids;

----------------------------------------------------------------------
-- 14. VACUUM ANALYZE
----------------------------------------------------------------------
\echo '>>> Running VACUUM ANALYZE on all tables...'
VACUUM ANALYZE tenants;
VACUUM ANALYZE event_registrations;
VACUUM ANALYZE event_registration_versions;
VACUUM ANALYZE webhook_registrations;
VACUUM ANALYZE webhook_health_state;
VACUUM ANALYZE webhook_rate_limit_state;
VACUUM ANALYZE webhook_alert_configs;
VACUUM ANALYZE event_subscriptions;
VACUUM ANALYZE subscription_template_versions;
VACUUM ANALYZE event_records;
VACUUM ANALYZE webhook_deliveries;
VACUUM ANALYZE webhook_health_events;
VACUUM ANALYZE webhook_health_buckets;
VACUUM ANALYZE webhook_health_evaluator;
VACUUM ANALYZE webhook_metrics;
VACUUM ANALYZE batch_jobs;
VACUUM ANALYZE access_tokens;
VACUUM ANALYZE access_invites;
VACUUM ANALYZE river_job;
VACUUM ANALYZE river_queue;
VACUUM ANALYZE river_migration;

\echo '>>> Seed complete!'

SET synchronous_commit = on;
