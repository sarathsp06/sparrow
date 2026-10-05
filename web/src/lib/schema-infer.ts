/**
 * JSON Schema inference from sample payloads.
 *
 * A best-effort heuristic: samples show what a payload looked like, not what
 * it may look like, so the result is a starting point for a human to refine.
 * With several samples, a property is required only when every object that
 * could hold it had it, types seen across samples are unioned, and a string
 * format is kept only when every string at that position matched it.
 * Only structure is emitted, never sample values.
 */

type Schema = Record<string, any>;

// Checked in order; the first match wins. Patterns are deliberately strict so
// a guessed format rarely rejects a real payload.
const DATE = String.raw`\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])`;
const OCTET = String.raw`(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)`;
const STRING_FORMATS: [string, RegExp][] = [
  ["uuid", /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i],
  ["date-time", new RegExp(String.raw`^${DATE}[Tt]([01]\d|2[0-3]):[0-5]\d:([0-5]\d|60)(\.\d+)?([Zz]|[+-]([01]\d|2[0-3]):[0-5]\d)$`)],
  ["date", new RegExp(`^${DATE}$`)],
  ["email", /^[^\s@"(),:;<>[\\\]]+@[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)+$/i],
  ["uri", /^[a-z][a-z0-9+.-]*:\/\/[^\s/?#]+[^\s]*$/i],
  ["ipv4", new RegExp(String.raw`^(${OCTET}\.){3}${OCTET}$`)],
];

/** The JSON Schema format a string matches, if any. */
export function detectStringFormat(value: string): string | undefined {
  for (const [format, re] of STRING_FORMATS) {
    if (re.test(value)) return format;
  }
  return undefined;
}

// Everything observed at one position in the payload, across all samples.
interface Node {
  kinds: Set<string>;
  // undefined: no string seen yet; null: strings disagree or match no format.
  format?: string | null;
  objects: number;
  props: Map<string, { node: Node; seen: number }>;
  items?: Node;
}

function newNode(): Node {
  return { kinds: new Set(), objects: 0, props: new Map() };
}

function observe(node: Node, value: unknown) {
  if (value === null || value === undefined) {
    node.kinds.add("null");
    return;
  }
  if (Array.isArray(value)) {
    node.kinds.add("array");
    for (const item of value) {
      node.items ??= newNode();
      observe(node.items, item);
    }
    return;
  }
  switch (typeof value) {
    case "object": {
      node.kinds.add("object");
      node.objects++;
      for (const [key, val] of Object.entries(value as object)) {
        let prop = node.props.get(key);
        if (!prop) {
          prop = { node: newNode(), seen: 0 };
          node.props.set(key, prop);
        }
        prop.seen++;
        observe(prop.node, val);
      }
      return;
    }
    case "string": {
      node.kinds.add("string");
      const format = detectStringFormat(value);
      if (node.format === undefined) node.format = format ?? null;
      else if (node.format !== format) node.format = null;
      return;
    }
    case "number":
      node.kinds.add(Number.isInteger(value) ? "integer" : "number");
      return;
    case "boolean":
      node.kinds.add("boolean");
      return;
  }
}

const KIND_ORDER = ["object", "array", "string", "number", "integer", "boolean", "null"];

function emit(node: Node): Schema {
  const kinds = new Set(node.kinds);
  if (kinds.has("number")) kinds.delete("integer");
  const types = KIND_ORDER.filter((k) => kinds.has(k));

  const schema: Schema = {};
  if (types.length === 1) schema.type = types[0];
  else if (types.length > 1) schema.type = types;

  if (kinds.has("object")) {
    const properties: Schema = {};
    const required: string[] = [];
    for (const [key, { node: child, seen }] of node.props) {
      properties[key] = emit(child);
      if (seen === node.objects) required.push(key);
    }
    schema.properties = properties;
    if (required.length > 0) schema.required = required;
  }
  if (kinds.has("array") && node.items) {
    schema.items = emit(node.items);
  }
  if (kinds.has("string") && node.format) {
    schema.format = node.format;
  }
  return schema;
}

/** How one property showed up across the samples. */
export interface FieldCoverage {
  /** Dotted path; `[]` marks array items, e.g. `items[].sku`. */
  path: string;
  /** JSON Schema type(s) emitted for the field. */
  types: string[];
  format?: string;
  /** Objects that had the field, out of `of` objects that could have. */
  seen: number;
  of: number;
  required: boolean;
}

function collect(node: Node, prefix: string, out: FieldCoverage[]) {
  for (const [key, { node: child, seen }] of node.props) {
    const path = prefix ? `${prefix}.${key}` : key;
    const emitted = emit(child);
    out.push({
      path,
      types: [emitted.type ?? []].flat(),
      format: emitted.format,
      seen,
      of: node.objects,
      required: seen === node.objects,
    });
    collect(child, path, out);
  }
  if (node.items) collect(node.items, `${prefix}[]`, out);
}

/**
 * Infer one JSON Schema that accepts every sample, plus a per-field account
 * of how often each property appeared, so a reviewer can see why a field
 * came out optional or nullable.
 */
export function analyzeSamples(samples: unknown[]): { schema: Schema; fields: FieldCoverage[] } {
  if (samples.length === 0) return { schema: {}, fields: [] };
  const root = newNode();
  for (const sample of samples) observe(root, sample);
  const fields: FieldCoverage[] = [];
  collect(root, "", fields);
  return { schema: emit(root), fields };
}

/**
 * Infer one JSON Schema that accepts every sample.
 * Returns `{}` (accept anything) when there are no samples.
 */
export function inferSchema(samples: unknown[]): Schema {
  return analyzeSamples(samples).schema;
}

/** Generate a JSON Schema from a single sample value. */
export function jsonToJsonSchema(value: unknown): Schema {
  return inferSchema([value]);
}
