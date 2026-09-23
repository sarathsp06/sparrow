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
export const RECIPE_META: Record<
  string,
  {
    title: string;
    demoParams: Record<string, string>;
    presets: EventPreset[];
    setup: { docsUrl: string; docsLabel: string; steps: string[] };
  }
> = {
  sendgrid: {
    title: 'SendGrid Email',
    demoParams: {
      api_key: 'SG.demo_key_123',
      from_email: 'alerts@example.com',
      from_name: 'Sparrow Webhooks',
      default_recipient: 'team@acme.com',
    },
    setup: {
      docsUrl: 'https://www.twilio.com/docs/sendgrid/ui/account-and-settings/api-keys',
      docsLabel: 'SendGrid API keys & sender verification',
      steps: [
        'Settings → API Keys → Create API Key; pick Restricted Access and enable Mail Send.',
        'Copy the key immediately (shown only once) into api_key.',
        'Settings → Sender Authentication → verify a Single Sender (or authenticate your domain).',
        'Use that verified address as from_email; from_name is the display name.',
      ],
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
    setup: {
      docsUrl: 'https://api.slack.com/messaging/webhooks',
      docsLabel: 'Slack incoming webhooks',
      steps: [
        'Create an app at api.slack.com/apps → From scratch, pick your workspace.',
        'Enable Incoming Webhooks, then Add New Webhook to Workspace and choose a channel.',
        'Copy the hooks.slack.com/services/… URL into webhook_url — the URL is the secret.',
        'The channel is fixed by the webhook; add one webhook per channel.',
      ],
    },
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
    setup: {
      docsUrl: 'https://www.twilio.com/docs/messaging/quickstart',
      docsLabel: 'Twilio SMS quickstart',
      steps: [
        'In the Twilio Console copy your Account SID (AC…) and Auth Token.',
        'Phone Numbers → Buy a number; pick an SMS-capable number for from_number (E.164 digits).',
        "Build basic_auth: printf '%s:%s' <AccountSID> <AuthToken> | base64",
        'Set to_number to the recipient in E.164 (digits only).',
      ],
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
    setup: {
      docsUrl: 'https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks',
      docsLabel: 'Discord webhooks',
      steps: [
        'Open Server Settings → Integrations → Webhooks (or a channel’s Edit → Integrations).',
        'Click New Webhook and select the target channel.',
        'Copy Webhook URL into webhook_url.',
      ],
    },
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
    setup: {
      docsUrl: 'https://docs.ntfy.sh/publish/',
      docsLabel: 'ntfy publish & subscribe',
      steps: [
        'Pick an unguessable topic name — anyone who knows it can read/write.',
        'Subscribe to it in the ntfy app (iOS/Android) or ntfy.sh/app — no signup needed.',
        'Set topic to that name; keep server_url as https://ntfy.sh unless self-hosting.',
        'Self-hosted with access control: add auth per the ntfy docs.',
      ],
    },
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
    setup: {
      docsUrl: 'https://support.pagerduty.com/main/docs/services-and-integrations',
      docsLabel: 'PagerDuty Events API v2',
      steps: [
        'Services → Service Directory; create or pick a service.',
        'Integrations tab → Add integration → Events API V2 → Add.',
        'Expand the integration and copy its Integration Key into routing_key.',
      ],
    },
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
    setup: {
      docsUrl: 'https://clickhouse.com/docs/interfaces/http',
      docsLabel: 'ClickHouse HTTP interface',
      steps: [
        'Confirm the HTTP endpoint (port 8123 HTTP / 8443 HTTPS) and set base_url.',
        'Create the table: CREATE TABLE <t> (event_id String, event_name String, timestamp String, payload String) ENGINE = MergeTree ORDER BY event_id;',
        'Set user/password; the recipe authenticates via X-ClickHouse-User / X-ClickHouse-Key headers.',
      ],
    },
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
