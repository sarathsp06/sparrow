// Turns human-authored text with {{.payload.x}} placeholders into the Go
// template expressions Sparrow renders server-side. Composers generate the
// full transform template, so the in-browser preview engine only ever sees
// shapes produced here.

const VAR_RE = /\{\{\s*(\.[A-Za-z_][\w.]*)\s*\}\}/g;

function splitVars(text: string): { fmt: string; args: string[] } {
  let fmt = '';
  const args: string[] = [];
  let last = 0;
  for (const m of text.matchAll(VAR_RE)) {
    fmt += text.slice(last, m.index).replaceAll('%', '%%') + '%v';
    args.push(m[1]);
    last = m.index! + m[0].length;
  }
  fmt += text.slice(last).replaceAll('%', '%%');
  return { fmt, args };
}

/** Go template expression that renders `text` as a JSON string value. */
export function toJsonExpr(text: string): string {
  const { fmt, args } = splitVars(text);
  if (args.length === 0) return JSON.stringify(text);
  return `{{printf ${JSON.stringify(fmt)} ${args.join(' ')} | json}}`;
}

/** Go template expression that renders `text` URL-encoded (for form bodies). */
export function toUrlEncodedExpr(text: string): string {
  const { fmt, args } = splitVars(text);
  if (args.length === 0) return encodeURIComponent(text);
  return `{{printf ${JSON.stringify(fmt)} ${args.join(' ')} | urlencode}}`;
}

/** Leaf paths of an object as Go template accessors, e.g. `.payload.host`. */
export function payloadPaths(payload: unknown): string[] {
  const out: string[] = [];
  const walk = (val: unknown, path: string) => {
    if (Array.isArray(val)) return; // ranges need real template code; skip as chips
    if (val && typeof val === 'object') {
      for (const [k, v] of Object.entries(val)) {
        if (/^[A-Za-z_]\w*$/.test(k)) walk(v, `${path}.${k}`);
      }
      return;
    }
    out.push(path);
  };
  walk(payload, '.payload');
  return out;
}

export const ENVELOPE_PATHS = ['.event_name', '.event_id', '.timestamp', '.attempt'];

// ---------- SendGrid ----------

export interface EmailCompose {
  toMode: 'list' | 'addresses';
  listPath: string; // payload array accessor, e.g. '.payload.alert_recipients'
  listField: string; // address field on each list item; '' = items are plain strings
  to: string; // comma-separated; entries may contain {{.payload.x}}
  cc: string; // comma-separated, optional (addresses mode only)
  bcc: string; // comma-separated, optional (addresses mode only)
  subject: string;
  bodyText: string;
}

/** `"a@b.c, {{.payload.x}}"` → SendGrid address array `[{"email": ...}, …]`, or null when empty. */
function emailArray(csv: string): string | null {
  const entries = csv.split(',').map((s) => s.trim()).filter(Boolean);
  if (entries.length === 0) return null;
  return `[${entries.map((e) => `{"email": ${toJsonExpr(e)}}`).join(', ')}]`;
}

export function generateSendgridTemplate(c: EmailCompose): string {
  let personalizations: string;
  if (c.toMode === 'list') {
    // One personalization per list entry so recipients never see each other,
    // falling back to the default_recipient param — mirrors sendgrid.yaml.
    const addr = c.listField ? `$r.${c.listField}` : '$r';
    personalizations = `[{{if ${c.listPath}}}{{range $i, $r := ${c.listPath}}}{{if $i}},{{end}}{"to": [{"email": {{${addr} | json}}}]}{{end}}{{else}}{"to": [{"email": "{{param "default_recipient"}}"}]}{{end}}]`;
  } else {
    const cc = emailArray(c.cc);
    const bcc = emailArray(c.bcc);
    personalizations = `[{"to": ${emailArray(c.to) ?? '[]'}${cc ? `, "cc": ${cc}` : ''}${bcc ? `, "bcc": ${bcc}` : ''}}]`;
  }
  const body = c.bodyText;
  return `{
  "personalizations": ${personalizations},
  "from": {"email": "{{param "from_email"}}", "name": "{{param "from_name"}}"},
  "subject": ${toJsonExpr(c.subject)},
  "content": [{"type": "text/plain", "value": ${toJsonExpr(body)}}]
}`;
}

// ---------- Twilio SMS ----------

export function generateTwilioTemplate(message: string): string {
  return `To=%2B{{param "to_number"}}&From=%2B{{param "from_number"}}&Body=${toUrlEncodedExpr(message)}`;
}

/** GSM-7 vs UCS-2 segment estimate for the message with vars substituted out. */
export function smsSegments(rendered: string): { chars: number; segments: number; encoding: string } {
  // ponytail: simplified GSM-7 check (basic latin only); extend the charset if it matters
  const gsm = /^[\x20-\x7E\n\r]*$/.test(rendered);
  const chars = rendered.length;
  const single = gsm ? 160 : 70;
  const multi = gsm ? 153 : 67;
  const segments = chars === 0 ? 0 : chars <= single ? 1 : Math.ceil(chars / multi);
  return { chars, segments, encoding: gsm ? 'GSM-7' : 'UCS-2' };
}

// ---------- Slack ----------

export interface SlackCompose {
  header: string;
  body: string;
  includeContext: boolean; // event id / timestamp / attempt fields
}

export function generateSlackTemplate(c: SlackCompose): string {
  const blocks: string[] = [
    `    {"type": "header", "text": {"type": "plain_text", "text": ${toJsonExpr(c.header)}, "emoji": true}}`,
  ];
  if (c.includeContext) {
    blocks.push(
      `    {"type": "section", "fields": [
      {"type": "mrkdwn", "text": {{printf "*Event ID:*\\n%v" .event_id | json}}},
      {"type": "mrkdwn", "text": {{printf "*Timestamp:*\\n%v" .timestamp | json}}},
      {"type": "mrkdwn", "text": {{printf "*Attempt:*\\n%v" .attempt | json}}}
    ]}`,
    );
  }
  blocks.push(`    {"type": "section", "text": {"type": "mrkdwn", "text": ${toJsonExpr(c.body)}}}`);
  return `{
  "blocks": [
${blocks.join(',\n')}
  ]
}`;
}

// ---------- Payload dump helper ----------

/** Appends the full event payload to `text` as a trailing printf arg. */
function withPayload(text: string, fenced: boolean): string {
  const { fmt, args } = splitVars(text);
  const dump = fenced ? '```json\n%s\n```' : '%s';
  const combinedFmt = text ? `${fmt}\n\n${dump}` : dump;
  return `{{printf ${JSON.stringify(combinedFmt)} ${[...args, '(json .payload | ellipsis 4000)'].join(' ')} | json}}`;
}

// ---------- Discord ----------

export interface DiscordCompose {
  title: string;
  description: string;
  includePayload: boolean; // append the raw event payload as a json code block
}

export function generateDiscordTemplate(c: DiscordCompose): string {
  const description = c.includePayload ? withPayload(c.description, true) : toJsonExpr(c.description);
  return `{
  "embeds": [
    {
      "title": ${toJsonExpr(c.title)},
      "description": ${description},
      "timestamp": {{.timestamp | json}}
    }
  ]
}`;
}

// ---------- ntfy ----------

export interface NtfyCompose {
  title: string;
  message: string;
  tags: string; // comma-separated; ntfy renders known names as emoji
  priority: number; // 1 (min) .. 5 (max), 3 = default
  includePayload: boolean;
}

export function generateNtfyTemplate(c: NtfyCompose): string {
  const message = c.includePayload ? withPayload(c.message, false) : toJsonExpr(c.message);
  const tags = c.tags.split(',').map((t) => t.trim()).filter(Boolean);
  return `{
  "topic": "{{param "topic"}}",
  "title": ${toJsonExpr(c.title)},
  "message": ${message},
  "priority": ${c.priority},
  "tags": ${JSON.stringify(tags)}
}`;
}

// ---------- PagerDuty ----------

export interface PagerdutyCompose {
  summary: string;
  severity: string; // critical | error | warning | info
  source: string;
}

export function generatePagerdutyTemplate(c: PagerdutyCompose): string {
  return `{
  "routing_key": "{{param "routing_key"}}",
  "event_action": "trigger",
  "dedup_key": {{.event_id | json}},
  "payload": {
    "summary": ${toJsonExpr(c.summary)},
    "source": ${toJsonExpr(c.source)},
    "severity": ${toJsonExpr(c.severity)},
    "timestamp": {{.timestamp | json}},
    "custom_details": {{.payload | json}}
  }
}`;
}
