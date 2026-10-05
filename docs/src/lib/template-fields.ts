// Turns a Sparrow transform template (JSON or form body with Go template
// actions) into a list of editable fields, and writes edited field text back
// into the template. The template is the source of truth: fields are derived
// from it, and an edit rewrites only that field's span.
//
// Field text uses the same notation people type: literal text with
// {{.payload.x}} / {{param "x"}} placeholders. On write-back it is re-encoded
// so the value stays valid JSON (or URL-encoded) whatever the event contains:
//   "Hi {{.payload.name}}"  →  {{printf "Hi %v" .payload.name | json}}

export interface TemplateField {
  start: number; // span of the value in the template
  end: number;
  path: string; // e.g. blocks[2].text.text, or the form key
  key: string; // last object key ('' inside arrays / at the root)
  when: string; // enclosing {{if}}/{{range}} conditions, '' when unconditional
  text: string; // editable text
  kind: 'text' | 'raw' | 'number' | 'bool';
  encoding: 'json' | 'url';
  primary: boolean; // message content rather than structure / config
}

export interface ParsedTemplate {
  format: 'json' | 'form' | 'unknown';
  fields: TemplateField[];
}

// Keys that are usually wiring, not message content; shown under "Other fields".
const STRUCTURAL_KEYS = new Set([
  'type', 'emoji', 'event_action', 'routing_key', 'dedup_key', 'topic', 'timestamp', 'time', 'id',
  'specversion', 'datacontenttype', 'From', 'To', 'from', 'custom_details', 'priority', 'tags',
]);

/** Index just past the `}}` closing the action that opens at `i`. */
function actionEnd(t: string, i: number): number {
  let j = i + 2;
  while (j < t.length) {
    const c = t[j];
    if (c === '"' || c === '`') {
      // Go string / raw string inside the action — may contain }}
      const q = c;
      j++;
      while (j < t.length && t[j] !== q) j += t[j] === '\\' && q === '"' ? 2 : 1;
      j++;
      continue;
    }
    if (c === '}' && t[j + 1] === '}') return j + 2;
    j++;
  }
  return t.length;
}

function actionBody(raw: string): string {
  return raw.slice(2, -2).replace(/^-\s*/, '').replace(/\s*-$/, '').trim();
}

const CONTROL_RE = /^(if|else|end|range|with|define|template|block|break|continue)\b|^\/\*/;

/** Splits a pipeline / arg list on top-level separators, honouring quotes and parens. */
function splitTop(s: string, sep: (c: string) => boolean): string[] {
  const out: string[] = [];
  let cur = '';
  let depth = 0;
  for (let i = 0; i < s.length; i++) {
    const c = s[i];
    if (c === '"' || c === '`') {
      let j = i + 1;
      while (j < s.length && s[j] !== c) j += s[j] === '\\' && c === '"' ? 2 : 1;
      cur += s.slice(i, j + 1);
      i = j;
      continue;
    }
    if (c === '(') depth++;
    if (c === ')') depth--;
    if (depth === 0 && sep(c)) {
      if (cur.trim()) out.push(cur.trim());
      cur = '';
      continue;
    }
    cur += c;
  }
  if (cur.trim()) out.push(cur.trim());
  return out;
}

function unparen(a: string): string {
  if (!a.startsWith('(') || !a.endsWith(')')) return a;
  // only strip when the parens wrap the whole arg
  let depth = 0;
  for (let i = 0; i < a.length; i++) {
    if (a[i] === '(') depth++;
    if (a[i] === ')') depth--;
    if (depth === 0 && i < a.length - 1) return a;
  }
  return a.slice(1, -1).trim();
}

/**
 * Decodes the body of a value action (`printf "x %v" .a | json`) given the
 * filter that makes it safe (`json` / `urlencode`). Returns null when the
 * action isn't in that shape and must be edited verbatim.
 */
function decodeAction(body: string, filter: string): string | null {
  const stages = splitTop(body, (c) => c === '|');
  if (stages.length < 2 || stages[stages.length - 1] !== filter) return null;
  const inner = stages.slice(0, -1).join(' | ');
  const toks = splitTop(inner, (c) => /\s/.test(c));
  if (toks[0] !== 'printf' || stages.length > 2 || !toks[1]?.startsWith('"')) return `{{${inner}}}`;
  let fmt: string;
  try {
    fmt = JSON.parse(toks[1]);
  } catch {
    return null;
  }
  const args = toks.slice(2).map(unparen);
  let n = 0;
  let ok = true;
  const text = fmt.replace(/%(%|[vsdq])/g, (m, v) => {
    if (v === '%') return '%';
    if (v === 'q' || n >= args.length) ok = false;
    return `{{${args[n++]}}}`;
  });
  return ok && n === args.length ? text : null;
}

/** Splits text into literal and `{{action}}` segments (quote-aware). */
function segments(text: string): { lit: string; expr: string | null }[] {
  const out: { lit: string; expr: string | null }[] = [];
  let i = 0;
  while (i < text.length) {
    const j = text.indexOf('{{', i);
    if (j === -1) {
      out.push({ lit: text.slice(i), expr: null });
      break;
    }
    if (j > i) out.push({ lit: text.slice(i, j), expr: null });
    const e = actionEnd(text, j);
    out.push({ lit: text.slice(j, e), expr: actionBody(text.slice(j, e)) });
    i = e;
  }
  return out;
}

/** Re-encodes field text as a template value that renders safely. */
export function encodeText(text: string, enc: 'json' | 'url'): string {
  const filter = enc === 'json' ? 'json' : 'urlencode';
  const segs = segments(text);
  const acts = segs.filter((s) => s.expr !== null);
  if (acts.length === 0) return enc === 'json' ? JSON.stringify(text) : encodeURIComponent(text);
  if (segs.length === 1) return `{{${acts[0].expr} | ${filter}}}`;
  let fmt = '';
  const args: string[] = [];
  for (const s of segs) {
    if (s.expr === null) {
      fmt += s.lit.replaceAll('%', '%%');
      continue;
    }
    fmt += '%v';
    args.push(/^[.$][\w.$]*$/.test(s.expr) ? s.expr : `(${s.expr})`);
  }
  return `{{printf ${JSON.stringify(fmt)} ${args.join(' ')} | ${filter}}}`;
}

/** True when every `{{` has its closing `}}` — partial edits aren't written back. */
export function balanced(text: string): boolean {
  return text.split('{{').length === text.split('}}').length && segments(text).every((s) => s.expr === null || s.lit.endsWith('}}'));
}

function decodeJsonString(lit: string): string {
  // lit includes the quotes; actions inside are kept verbatim
  let out = '';
  let i = 1;
  while (i < lit.length - 1) {
    if (lit.startsWith('{{', i)) {
      const e = actionEnd(lit, i);
      out += lit.slice(i, e);
      i = e;
      continue;
    }
    if (lit[i] === '\\') {
      const n = lit[i + 1];
      if (n === 'u') {
        out += String.fromCharCode(parseInt(lit.slice(i + 2, i + 6), 16));
        i += 6;
        continue;
      }
      out += ({ n: '\n', t: '\t', r: '\r', b: '\b', f: '\f' } as Record<string, string>)[n] ?? n;
      i += 2;
      continue;
    }
    out += lit[i++];
  }
  return out;
}

function decodeUrlValue(v: string): string {
  let out = '';
  let i = 0;
  while (i < v.length) {
    if (v.startsWith('{{', i)) {
      const e = actionEnd(v, i);
      const raw = v.slice(i, e);
      out += decodeAction(actionBody(raw), 'urlencode') ?? raw;
      i = e;
      continue;
    }
    let j = v.indexOf('{{', i);
    if (j === -1) j = v.length;
    try {
      out += decodeURIComponent(v.slice(i, j));
    } catch {
      out += v.slice(i, j);
    }
    i = j;
  }
  return out;
}

/** Skips leading whitespace and actions to find the first literal character. */
function firstLiteral(t: string): string {
  let i = 0;
  while (i < t.length) {
    if (/\s/.test(t[i])) i++;
    else if (t.startsWith('{{', i)) i = actionEnd(t, i);
    else return t[i];
  }
  return '';
}

function isPrimary(key: string, text: string, kind: TemplateField['kind']): boolean {
  if (kind !== 'text' || STRUCTURAL_KEYS.has(key)) return false;
  return segments(text).some((s) => (s.expr === null ? s.lit.trim() !== '' : !/^param\s/.test(s.expr)));
}

function updateConds(conds: string[], body: string) {
  const kw = body.split(/\s+/)[0];
  if (kw === 'if' || kw === 'range' || kw === 'with') conds.push(body);
  else if (kw === 'else') {
    const prev = conds.pop() ?? '';
    const rest = body.slice(4).trim();
    conds.push(rest ? `else ${rest}` : `else (not: ${prev.replace(/^(else )?(if )?/, '')})`);
  } else if (kw === 'end') conds.pop();
}

// condDepth: commas emitted inside a deeper {{if}}/{{range}} (e.g. `{{if $i}},{{end}}`)
// separate generated items, not template-level elements, so they don't advance the index.
type Frame = { arr: false; key: string | null; expectKey: boolean } | { arr: true; index: number; condDepth: number };

function framePath(frames: Frame[]): { path: string; key: string } {
  let path = '';
  let key = '';
  for (const f of frames) {
    if (f.arr) {
      path += `[${f.index}]`;
      key = '';
    } else if (f.key !== null) {
      path += /^[A-Za-z_]\w*$/.test(f.key) ? `${path ? '.' : ''}${f.key}` : `[${JSON.stringify(f.key)}]`;
      key = f.key;
    }
  }
  return { path: path || '(root)', key };
}

function parseJson(t: string): TemplateField[] {
  const fields: TemplateField[] = [];
  const frames: Frame[] = [];
  const conds: string[] = [];
  const top = () => frames[frames.length - 1];
  const inValue = () => {
    const f = top();
    return !f || f.arr || !f.expectKey;
  };
  const add = (start: number, end: number, text: string, kind: TemplateField['kind']) => {
    const { path, key } = framePath(frames);
    fields.push({ start, end, path, key, when: conds.join(' › '), text, kind, encoding: 'json', primary: isPrimary(key, text, kind) });
  };

  let i = 0;
  while (i < t.length) {
    const c = t[i];
    if (t.startsWith('{{', i)) {
      const e = actionEnd(t, i);
      const body = actionBody(t.slice(i, e));
      if (CONTROL_RE.test(body)) updateConds(conds, body);
      else if (inValue()) {
        const text = decodeAction(body, 'json');
        add(i, e, text ?? t.slice(i, e), text === null ? 'raw' : 'text');
      }
      i = e;
      continue;
    }
    if (c === '"') {
      let j = i + 1;
      while (j < t.length && t[j] !== '"') {
        if (t.startsWith('{{', j)) j = actionEnd(t, j);
        else j += t[j] === '\\' ? 2 : 1;
      }
      const lit = t.slice(i, j + 1);
      const f = top();
      if (f && !f.arr && f.expectKey) f.key = decodeJsonString(lit);
      else add(i, j + 1, decodeJsonString(lit), 'text');
      i = j + 1;
      continue;
    }
    if (c === '{') frames.push({ arr: false, key: null, expectKey: true });
    else if (c === '[') frames.push({ arr: true, index: 0, condDepth: conds.length });
    else if (c === '}' || c === ']') frames.pop();
    else if (c === ':') {
      const f = top();
      if (f && !f.arr) f.expectKey = false;
    } else if (c === ',') {
      const f = top();
      if (f?.arr) {
        if (conds.length === f.condDepth) f.index++;
      }
      else if (f) {
        f.expectKey = true;
        f.key = null;
      }
    } else if (inValue()) {
      const m = /^(-?\d+(\.\d+)?([eE][+-]?\d+)?|true|false)/.exec(t.slice(i));
      if (m) {
        const bool = m[0] === 'true' || m[0] === 'false';
        add(i, i + m[0].length, m[0], bool ? 'bool' : 'number');
        i += m[0].length;
        continue;
      }
    }
    i++;
  }
  return fields;
}

function parseForm(t: string): TemplateField[] {
  const fields: TemplateField[] = [];
  const conds: string[] = [];
  let i = 0;
  let partStart = 0;
  const flush = (end: number) => {
    const part = t.slice(partStart, end);
    const eq = part.indexOf('=');
    if (eq > 0 && !part.slice(0, eq).includes('{{')) {
      const key = decodeURIComponent(part.slice(0, eq).trim());
      // trim trailing whitespace/newline from the value span
      let vEnd = end;
      while (vEnd > partStart + eq + 1 && /\s/.test(t[vEnd - 1])) vEnd--;
      const text = decodeUrlValue(t.slice(partStart + eq + 1, vEnd));
      fields.push({
        start: partStart + eq + 1, end: vEnd, path: key, key, when: conds.join(' › '), text,
        kind: 'text', encoding: 'url', primary: isPrimary(key, text, 'text'),
      });
    }
  };
  while (i < t.length) {
    if (t.startsWith('{{', i)) {
      const e = actionEnd(t, i);
      const body = actionBody(t.slice(i, e));
      if (CONTROL_RE.test(body)) updateConds(conds, body);
      i = e;
      continue;
    }
    if (t[i] === '&') {
      flush(i);
      partStart = i + 1;
    }
    i++;
  }
  flush(t.length);
  return fields;
}

export function parseTemplate(t: string): ParsedTemplate {
  const first = firstLiteral(t);
  if (first === '{' || first === '[') return { format: 'json', fields: parseJson(t) };
  if (/^\s*[\w.-]+=/.test(t)) return { format: 'form', fields: parseForm(t) };
  return { format: 'unknown', fields: [] };
}

/** Returns the template with `field` set to `text`. */
export function writeField(template: string, field: TemplateField, text: string): string {
  let value: string;
  if (field.kind === 'raw' || field.kind === 'number' || field.kind === 'bool') value = text;
  else value = encodeText(text, field.encoding);
  return template.slice(0, field.start) + value + template.slice(field.end);
}

/** Best guess at which destination a template targets, from its shape. */
export function detectRecipe(t: string): string | null {
  if (/"personalizations"/.test(t)) return 'sendgrid';
  if (/"blocks"/.test(t)) return 'slack';
  if (/"embeds"/.test(t)) return 'discord';
  if (/"routing_key"/.test(t)) return 'pagerduty';
  if (/"specversion"/.test(t)) return 'cloudevents';
  if (/"topic"/.test(t) && /"message"/.test(t)) return 'ntfy';
  if (/(^|&)Body=/.test(t.trim())) return 'twilio';
  return null;
}
