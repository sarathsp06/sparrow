// Flattens a JSON Schema into one row per field, for showing an event's
// payload as a readable table. Anything the table cannot express ($ref,
// oneOf, open objects) degrades to a plain type instead of failing.

export type SchemaField = {
  path: string;
  depth: number;
  type: string;
  required: boolean;
  description?: string;
  enumValues?: unknown[];
};

type Schema = Record<string, unknown>;

const MAX_DEPTH = 8;

function isSchema(v: unknown): v is Schema {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

function typeOf(s: Schema): string {
  if (typeof s.$ref === "string") return s.$ref.split("/").pop() || "object";
  const t = s.type;
  if (Array.isArray(t)) return t.join(" | ");
  if (typeof t === "string") {
    if (t === "array") return `${isSchema(s.items) ? typeOf(s.items) : "any"}[]`;
    return typeof s.format === "string" ? `${t} (${s.format})` : t;
  }
  for (const k of ["oneOf", "anyOf"] as const) {
    const alts = s[k];
    if (Array.isArray(alts) && alts.length) return alts.filter(isSchema).map(typeOf).join(" | ");
  }
  if (isSchema(s.properties)) return "object";
  if (Array.isArray(s.enum)) return "enum";
  return "any";
}

function walk(s: Schema, prefix: string, depth: number, out: SchemaField[]) {
  if (depth > MAX_DEPTH) return;
  const props = isSchema(s.properties) ? s.properties : null;
  if (props) {
    const required = new Set(Array.isArray(s.required) ? s.required : []);
    for (const [name, child] of Object.entries(props)) {
      if (!isSchema(child)) continue;
      const path = prefix ? `${prefix}.${name}` : name;
      out.push({
        path,
        depth,
        type: typeOf(child),
        required: required.has(name),
        description: typeof child.description === "string" ? child.description : undefined,
        enumValues: Array.isArray(child.enum) ? child.enum : undefined,
      });
      walk(child, path, depth + 1, out);
    }
  }
  // Arrays of objects list the item's fields under `name[]`.
  if (isSchema(s.items) && prefix) walk(s.items, `${prefix}[]`, depth, out);
}

export function schemaFields(schema: unknown): SchemaField[] {
  if (!isSchema(schema)) return [];
  const out: SchemaField[] = [];
  walk(schema, "", 0, out);
  return out;
}

export function hasSchema(schema: unknown): boolean {
  return isSchema(schema) && Object.keys(schema).length > 0;
}

// Groups event names by their first dot-separated segment ("order.created"
// → "order"), in first-seen order; names without a dot share the "" group,
// which comes last.
export function groupByPrefix<T extends { name: string }>(items: T[]): { prefix: string; items: T[] }[] {
  const groups = new Map<string, T[]>();
  for (const it of items) {
    const i = it.name.indexOf(".");
    const p = i > 0 ? it.name.slice(0, i) : "";
    const g = groups.get(p);
    if (g) g.push(it);
    else groups.set(p, [it]);
  }
  const out = [...groups].map(([prefix, items]) => ({ prefix, items }));
  return [...out.filter((g) => g.prefix), ...out.filter((g) => !g.prefix)];
}
