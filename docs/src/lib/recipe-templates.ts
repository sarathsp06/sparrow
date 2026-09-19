export interface RecipeDef {
  id: string;
  name: string;
  tagline: string;
  params: { name: string; prompt: string; default?: string; secret?: boolean }[];
  sampleEvent: { event_name: string; payload: any };
  webhook: { url: string; headers?: Record<string, string> };
  transform_template: string;
}

export const RECIPES: RecipeDef[] = [
  {
    id: 'sendgrid',
    name: 'SendGrid Email',
    tagline: 'Send email notifications with per-recipient personalizations',
    params: [
      { name: 'api_key', prompt: 'SendGrid API key', default: 'SG.demo_key_123', secret: true },
      { name: 'from_email', prompt: 'Verified sender email', default: 'alerts@example.com' },
      { name: 'from_name', prompt: 'Sender display name', default: 'Sparrow Webhooks' }
    ],
    sampleEvent: {
      event_name: 'sparrow.webhook.health_changed',
      payload: {
        alert_recipients: [{ email: 'devops@acme.com' }, { email: 'admin@acme.com' }],
        consumer: 'payments-service',
        webhook_id: 'wh_99a',
        url: 'https://api.acme.com/v1/payments',
        old_health: 'healthy',
        new_health: 'degraded'
      }
    },
    webhook: {
      url: 'https://api.sendgrid.com/v3/mail/send',
      headers: { 'Content-Type': 'application/json' }
    },
    transform_template: `{\n  "personalizations": [{{range $i, $r := .payload.alert_recipients}}{{if $i}},{{end}}{"to": [{"email": {{$r.email | json}}}]}{{end}}],\n  "from": {"email": "{{param \\"from_email\\"}}", "name": "{{param \\"from_name\\"}}"},\n  "subject": {{if eq .event_name "sparrow.webhook.health_changed"}}{{if eq .payload.new_health "healthy"}}{{printf "Sparrow: webhook for %v recovered" .payload.consumer | json}}{{else}}{{printf "Sparrow: webhook for %v is now %v" .payload.consumer .payload.new_health | json}}{{end}}{{else}}{{printf "Sparrow: delivery to %v failed permanently" .payload.consumer | json}}{{end}},\n  "content": [{"type": "text/plain", "value": {{if eq .event_name "sparrow.webhook.health_changed"}}{{printf "Webhook %v (%v) health changed: %v -> %v" .payload.webhook_id .payload.url .payload.old_health .payload.new_health | json}}{{else}}{{printf "Webhook %v (%v) delivery %v failed permanently after %v attempt(s): %v (%v)" .payload.webhook_id .payload.url .payload.delivery_id .payload.attempt .payload.error_message .payload.error_category | json}}{{end}}}]\n}`
  },
  {
    id: 'slack',
    name: 'Slack Incoming Webhook',
    tagline: 'Post Block Kit messages to Slack channels',
    params: [
      { name: 'webhook_url', prompt: 'Slack incoming webhook URL', default: 'https://hooks.slack.com/services/T00/B00/X123' }
    ],
    sampleEvent: {
      event_name: 'order.created',
      payload: { order_id: 'ord_9901', amount: 1999, currency: 'USD', customer: 'Alice Doe' }
    },
    webhook: {
      url: '{{param "webhook_url"}}',
      headers: { 'Content-Type': 'application/json' }
    },
    transform_template: `{\n  "blocks": [\n    {"type": "header", "text": {"type": "plain_text", "text": {{.event_name | json}}, "emoji": true}},\n    {"type": "section", "fields": [\n      {"type": "mrkdwn", "text": {{printf "*Event ID:*\\n%v" .event_id | json}}},\n      {"type": "mrkdwn", "text": {{printf "*Timestamp:*\\n%v" .timestamp | json}}},\n      {"type": "mrkdwn", "text": {{printf "*Attempt:*\\n%v" .attempt | json}}}\n    ]},\n    {"type": "section", "text": {"type": "mrkdwn", "text": {{printf "\`\`\`%v\`\`\`" (.payload | json) | json}}}}\n  ]\n}`
  },
  {
    id: 'discord',
    name: 'Discord Channel Embed',
    tagline: 'Post rich embed cards to Discord channels',
    params: [
      { name: 'webhook_url', prompt: 'Discord channel webhook URL', default: 'https://discord.com/api/webhooks/123/abc' }
    ],
    sampleEvent: {
      event_name: 'deployment.completed',
      payload: { env: 'production', commit: '7f3a910', service: 'auth-api', duration_s: 42 }
    },
    webhook: {
      url: '{{param "webhook_url"}}',
      headers: { 'Content-Type': 'application/json' }
    },
    transform_template: `{\n  "embeds": [\n    {\n      "title": {{.event_name | json}},\n      "description": {{printf "\`\`\`json\\n%v\\n\`\`\`" (.payload | json) | json}},\n      "timestamp": {{.timestamp | json}}\n    }\n  ]\n}`
  },
  {
    id: 'ntfy',
    name: 'ntfy Push Notification',
    tagline: 'Push plain-text alerts to ntfy topics',
    params: [
      { name: 'topic_url', prompt: 'ntfy topic URL', default: 'https://ntfy.sh/sparrow_alerts' }
    ],
    sampleEvent: {
      event_name: 'server.high_cpu',
      payload: { host: 'node-01.us-east', usage_percent: 94.2 }
    },
    webhook: {
      url: '{{param "topic_url"}}',
      headers: { 'Content-Type': 'text/plain' }
    },
    transform_template: `{{.event_name}} (event {{.event_id}}, attempt {{.attempt}}) at {{.timestamp}}\n\n{{.payload | json}}`
  },
  {
    id: 'pagerduty',
    name: 'PagerDuty Incident',
    tagline: 'Trigger PagerDuty incidents with automatic deduplication',
    params: [
      { name: 'routing_key', prompt: 'PagerDuty Events v2 routing key', default: 'pd-key-prod-9988' }
    ],
    sampleEvent: {
      event_name: 'database.connection_pool_exhausted',
      payload: { pool: 'main-pg', active: 50, max: 50, waiting: 14 }
    },
    webhook: {
      url: 'https://events.pagerduty.com/v2/enqueue',
      headers: { 'Content-Type': 'application/json' }
    },
    transform_template: `{\n  "routing_key": "{{param "routing_key"}}",\n  "event_action": "trigger",\n  "dedup_key": {{.event_id | json}},\n  "payload": {\n    "summary": {{printf "Sparrow event: %v" .event_name | json}},\n    "source": "sparrow",\n    "severity": "error",\n    "timestamp": {{.timestamp | json}},\n    "custom_details": {{.payload | json}}\n  }\n}`
  },
  {
    id: 'clickhouse',
    name: 'ClickHouse Ingestion',
    tagline: 'Insert events as rows via ClickHouse JSONEachRow HTTP API',
    params: [
      { name: 'base_url', prompt: 'ClickHouse HTTP base URL', default: 'http://clickhouse.internal:8123' },
      { name: 'table', prompt: 'Target table name', default: 'webhook_events' },
      { name: 'user', prompt: 'Database username', default: 'default' },
      { name: 'password', prompt: 'Database password', default: 'secret123', secret: true }
    ],
    sampleEvent: {
      event_name: 'user.signup',
      payload: { user_id: 'usr_8812', plan: 'pro', referrer: 'google' }
    },
    webhook: {
      url: '{{param "base_url"}}/?query=INSERT+INTO+{{param "table"}}+FORMAT+JSONEachRow',
      headers: { 'Content-Type': 'application/json' }
    },
    transform_template: `{"event_id": {{.event_id | json}}, "event_name": {{.event_name | json}}, "timestamp": {{.timestamp | json}}, "payload": {{.payload | json | json}}}`
  },
  {
    id: 'twilio',
    name: 'Twilio SMS',
    tagline: 'Send SMS alerts via Twilio Messages API',
    params: [
      { name: 'account_sid', prompt: 'Twilio Account SID', default: 'AC1234567890' },
      { name: 'basic_auth', prompt: 'Base64 SID:AuthToken', default: 'QUMxMjM0NTY3ODkwOnNlY3JldA==', secret: true },
      { name: 'from_number', prompt: 'Twilio phone number', default: '+15551230000' },
      { name: 'to_number', prompt: 'Recipient phone number', default: '+15559876543' }
    ],
    sampleEvent: {
      event_name: 'pager.critical_system_down',
      payload: { datacenter: 'us-west-2', active_incidents: 3 }
    },
    webhook: {
      url: 'https://api.twilio.com/2010-04-01/Accounts/{{param "account_sid"}}/Messages.json',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
    },
    transform_template: `To={{param "to_number" | urlencode}}&From={{param "from_number" | urlencode}}&Body={{printf "Sparrow %v (event %v, attempt %v) at %v" .event_name .event_id .attempt .timestamp | urlencode}}`
  }
];
