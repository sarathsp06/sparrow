import { parse } from 'yaml';

// Source of truth: the real recipe catalog shipped with the satellite CLI.
// Imported raw at build time so the docs tools can never drift from the server.
const rawFiles = import.meta.glob('../../../satellites/recipes/*.yaml', {
  query: '?raw',
  import: 'default',
  eager: true,
}) as Record<string, string>;

export interface RecipeParam {
  name: string;
  prompt: string;
  required?: boolean;
  secret?: boolean;
  default?: string;
  activation_required?: boolean;
  must_override_default?: boolean;
}

export interface RecipeDef {
  name: string;
  description: string;
  params: RecipeParam[];
  webhook: { url: string; headers?: Record<string, string>; secret_headers?: Record<string, string> };
  transform_template: string;
}

export interface EventPreset {
  id: string;
  label: string;
  event_name: string;
  payload: Record<string, unknown>;
}

const parsed: RecipeDef[] = Object.values(rawFiles).map((raw) => {
  const doc = parse(raw);
  return {
    name: doc.name,
    description: doc.description,
    params: doc.params ?? [],
    webhook: doc.webhook,
    transform_template: (doc.subscription?.transform_template ?? '').trimEnd(),
  };
});

// Display metadata + sample events live here (docs-only concerns).
export const RECIPE_META: Record<string, { title: string; demoParams: Record<string, string>; presets: EventPreset[] }> = {
  sendgrid: {
    title: 'SendGrid Email',
    demoParams: {
      api_key: 'SG.demo_key_123',
      from_email: 'alerts@example.com',
      from_name: 'Sparrow Webhooks',
      default_recipient: 'team@acme.com',
    },
    presets: [
      {
        id: 'health_changed',
        label: 'Webhook health alert',
        event_name: 'sparrow.webhook.health_changed',
        payload: {
          alert_recipients: [{ email: 'devops-team@acme.com' }, { email: 'oncall@acme.com' }],
          consumer: 'billing-service',
          webhook_id: 'wh_01h8x9p3',
          url: 'https://api.acme.com/v1/webhooks/billing',
          old_health: 'healthy',
          new_health: 'degraded',
        },
      },
      {
        id: 'delivery_failed',
        label: 'Delivery failed alert',
        event_name: 'sparrow.delivery_failed',
        payload: {
          alert_recipients: [{ email: 'sre-alerts@acme.com' }],
          consumer: 'shipping-service',
          webhook_id: 'wh_01h8x9p8',
          url: 'https://shipping.acme.com/events',
          delivery_id: 'del_99f23a10',
          attempt: 5,
          error_message: 'HTTP 500 Internal Server Error (connection reset)',
          error_category: 'HTTP 5xx',
        },
      },
      {
        id: 'order_created',
        label: 'Order confirmation (custom event)',
        event_name: 'order.created',
        payload: {
          customer_email: 'jane@acme.com',
          customer_name: 'Jane Doe',
          order_id: 'ord_98124',
          total_amount: '$149.50',
        },
      },
    ],
  },
  slack: {
    title: 'Slack Message',
    demoParams: { webhook_url: 'https://hooks.slack.com/services/T00/B00/X123' },
    presets: [
      {
        id: 'deploy',
        label: 'Deployment finished',
        event_name: 'deploy.finished',
        payload: { service: 'billing-api', version: 'v2.14.0', environment: 'production', duration_seconds: 84 },
      },
    ],
  },
  twilio: {
    title: 'Twilio SMS',
    demoParams: {
      account_sid: 'AC1234567890',
      basic_auth: 'QUMxMjM0NTY3ODkwOnNlY3JldA==',
      from_number: '15551230000',
      to_number: '15559876543',
    },
    presets: [
      {
        id: 'incident',
        label: 'Critical incident page',
        event_name: 'pager.critical_system_down',
        payload: { datacenter: 'us-west-2', active_incidents: 3 },
      },
    ],
  },
  discord: {
    title: 'Discord Embed',
    demoParams: { webhook_url: 'https://discord.com/api/webhooks/123/abc' },
    presets: [
      {
        id: 'signup',
        label: 'New user signup',
        event_name: 'user.signup',
        payload: { user_id: 'usr_8812', plan: 'pro', referrer: 'google' },
      },
    ],
  },
  ntfy: {
    title: 'ntfy Push',
    demoParams: { server_url: 'https://ntfy.sh', topic: 'sparrow_alerts' },
    presets: [
      {
        id: 'cpu',
        label: 'High CPU alert',
        event_name: 'server.high_cpu',
        payload: { host: 'node-01.us-east', usage_percent: 94.2 },
      },
    ],
  },
  pagerduty: {
    title: 'PagerDuty Incident',
    demoParams: { routing_key: 'pd-key-prod-9988', severity: 'error' },
    presets: [
      {
        id: 'pool',
        label: 'Connection pool exhausted',
        event_name: 'database.connection_pool_exhausted',
        payload: { pool: 'main-pg', active: 50, max: 50, waiting: 14 },
      },
    ],
  },
  clickhouse: {
    title: 'ClickHouse Ingest',
    demoParams: { base_url: 'http://clickhouse.internal:8123', table: 'webhook_events', user: 'default', password: 'secret123' },
    presets: [
      {
        id: 'signup',
        label: 'User signup event',
        event_name: 'user.signup',
        payload: { user_id: 'usr_8812', plan: 'pro', referrer: 'google' },
      },
    ],
  },
};

const ORDER = ['sendgrid', 'slack', 'twilio', 'discord', 'ntfy', 'pagerduty', 'clickhouse'];

export const RECIPES: RecipeDef[] = ORDER.map((n) => parsed.find((r) => r.name === n)).filter(
  (r): r is RecipeDef => !!r,
);
