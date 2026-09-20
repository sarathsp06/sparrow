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
  toMode: 'alert_recipients' | 'single';
  toAddress: string; // used when toMode === 'single'; may contain {{.payload.x}}
  subject: string;
  bodyMode: 'plain' | 'html';
  bodyText: string;
  bodyHtml: string;
}

export function generateSendgridTemplate(c: EmailCompose): string {
  // Mirrors sendgrid.yaml: alert_recipients when present, default_recipient param otherwise.
  const personalizations =
    c.toMode === 'alert_recipients'
      ? `[{{if .payload.alert_recipients}}{{range $i, $r := .payload.alert_recipients}}{{if $i}},{{end}}{"to": [{"email": {{$r.email | json}}}]}{{end}}{{else}}{"to": [{"email": "{{param "default_recipient"}}"}]}{{end}}]`
      : `[{"to": [{"email": ${toJsonExpr(c.toAddress)}}]}]`;
  const body = c.bodyMode === 'html' ? c.bodyHtml : c.bodyText;
  return `{
  "personalizations": ${personalizations},
  "from": {"email": "{{param "from_email"}}", "name": "{{param "from_name"}}"},
  "subject": ${toJsonExpr(c.subject)},
  "content": [{"type": "text/${c.bodyMode === 'html' ? 'html' : 'plain'}", "value": ${toJsonExpr(body)}}]
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
