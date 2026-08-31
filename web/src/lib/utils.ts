import { type Content } from "svelte-jsoneditor";

/**
 * Returns a JSON string representation of the given content.
 * If the content has a "text" property, it returns that string.
 * If the content has a "json" property, it stringifies that JSON object with indentation.
 * If neither of the above conditions are met, it returns an empty JSON object as a fallback.
 * @param {Content} content - The content to be converted to a JSON string.
 * @returns {string} - The JSON string representation of the content.
 */
function stringifyContent(content: Content): string {
  if ("text" in content && content.text) {
    return content.text;
  }
  if ("json" in content && content.json !== undefined) {
    return JSON.stringify(content.json, null, 2);
  }
  return "{}"; // fallback
}

/**
 * JSON Schema Meta-Schema for validating JSON Schemas themselves.
 * This is intentionally permissive — it accepts any valid JSON object as a schema.
 * JSON Schema is flexible: `{}` (match anything), `{"type": "string"}`,
 * `{"type": "array", "items": {...}}`, etc. are all valid schemas.
 * We only enforce that the schema is a JSON object (not a primitive or array).
 */
const JSONSchemaMetaSchema = {
  type: "object",
  additionalProperties: true,
};

/**
 * Infer a JSON Schema type string from a JavaScript value.
 */
function inferType(value: any): string {
  if (value === null) return "null";
  if (Array.isArray(value)) return "array";
  return typeof value; // "string", "number", "boolean", "object"
}

/**
 * Generate a JSON Schema from a sample JSON value.
 *
 * This is a best-effort heuristic: a single JSON value cannot definitively
 * determine the full schema (e.g., optional fields, union types, enums).
 * The generated schema should be treated as a starting point that users
 * can refine — adding `required`, `enum`, `minLength`, `pattern`, etc.
 *
 * @param value - Any JSON-compatible value (object, array, string, number, etc.)
 * @returns A JSON Schema object describing the structure of the input.
 */
function jsonToJsonSchema(value: any): Record<string, any> {
  if (value === null || value === undefined) {
    return { type: "null" };
  }

  if (Array.isArray(value)) {
    const schema: Record<string, any> = { type: "array" };
    if (value.length > 0) {
      // Use the first element to infer items schema.
      // For mixed-type arrays, the user will need to adjust manually.
      schema.items = jsonToJsonSchema(value[0]);
    }
    return schema;
  }

  if (typeof value === "object") {
    const properties: Record<string, any> = {};
    const required: string[] = [];

    for (const [key, val] of Object.entries(value)) {
      properties[key] = jsonToJsonSchema(val);
      required.push(key);
    }

    const schema: Record<string, any> = {
      type: "object",
      properties,
    };
    if (required.length > 0) {
      schema.required = required;
    }
    return schema;
  }

  // Primitive types: string, number, boolean
  const type = inferType(value);

  // Distinguish integer vs number
  if (type === "number" && Number.isInteger(value)) {
    return { type: "integer" };
  }

  return { type };
}

/**
 * Returns a JSON object representation of the given content.
 * If the content has a "text" property, it parses and returns that string as an object.
 * If the content has a "json" property, it returns that JSON object.
 * If neither of the above conditions are met, it returns an empty object as a fallback.
 * @param {Content} content - The content to be converted to a JSON object.
 * @returns {any} - The JSON object representation of the content.
 */
function toJSONObject(content: Content): any {
  if ("text" in content && content.text) {
    try {
      return JSON.parse(content.text);
    } catch (e) {
      console.error("Failed to parse JSON text", e);
      return {};
    }
  }
  if ("json" in content && content.json !== undefined) {
    return content.json;
  }
  return {};
}

// -- Error category utilities --

/**
 * All known error categories from pkg/errors/category.go.
 * Used to populate filter dropdowns consistently.
 */
const ERROR_CATEGORIES = [
  { value: "client_error", label: "Client (4xx)" },
  { value: "server_error", label: "Server (5xx)" },
  { value: "timeout", label: "Timeout" },
  { value: "dns_error", label: "DNS" },
  { value: "tls_error", label: "TLS" },
  { value: "connection_refused", label: "Conn Refused" },
  { value: "network_error", label: "Network" },
  { value: "unexpected_status", label: "Unexpected Status" },
] as const;

/**
 * Compact badge for error categories (used in table rows).
 * Returns a short label and Tailwind classes for inline badges.
 */
function getCategoryBadge(category: string): { label: string; classes: string } {
  switch (category) {
    case "client_error":
      return { label: "4xx", classes: "text-warn border-warn/40 bg-warn/10" };
    case "server_error":
      return { label: "5xx", classes: "text-bad border-bad/40 bg-bad/10" };
    case "timeout":
      return { label: "Timeout", classes: "text-warn border-warn/40 bg-warn/10" };
    case "dns_error":
      return { label: "DNS", classes: "text-idle border-idle/40 bg-idle/10" };
    case "tls_error":
      return { label: "TLS", classes: "text-idle border-idle/40 bg-idle/10" };
    case "connection_refused":
      return { label: "Conn Refused", classes: "text-idle border-idle/40 bg-idle/10" };
    case "network_error":
      return { label: "Network", classes: "text-idle border-idle/40 bg-idle/10" };
    case "unexpected_status":
      return { label: "Unexpected Status", classes: "text-warn border-warn/40 bg-warn/10" };
    case "success":
      return { label: "Success", classes: "text-ok border-ok/40 bg-ok/10" };
    default:
      return { label: category || "Unknown", classes: "text-muted border-line bg-panel-2" };
  }
}

/**
 * Detailed display for error categories (used in detail pages).
 * Returns a verbose label and separate color tokens.
 */
function getCategoryDisplay(category: string): {
  label: string;
  color: string;
  bgColor: string;
  borderColor: string;
} {
  switch (category) {
    case "client_error":
      return { label: "4xx Client Error", color: "text-warn", bgColor: "bg-warn/10", borderColor: "border-warn/40" };
    case "server_error":
      return { label: "5xx Server Error", color: "text-bad", bgColor: "bg-bad/10", borderColor: "border-bad/40" };
    case "timeout":
      return { label: "Timeout", color: "text-warn", bgColor: "bg-warn/10", borderColor: "border-warn/40" };
    case "dns_error":
      return { label: "DNS Error", color: "text-idle", bgColor: "bg-idle/10", borderColor: "border-idle/40" };
    case "tls_error":
      return { label: "TLS Error", color: "text-idle", bgColor: "bg-idle/10", borderColor: "border-idle/40" };
    case "connection_refused":
      return { label: "Connection Refused", color: "text-idle", bgColor: "bg-idle/10", borderColor: "border-idle/40" };
    case "network_error":
      return { label: "Network Error", color: "text-idle", bgColor: "bg-idle/10", borderColor: "border-idle/40" };
    case "unexpected_status":
      return { label: "Unexpected Status Code", color: "text-warn", bgColor: "bg-warn/10", borderColor: "border-warn/40" };
    case "success":
      return { label: "Success", color: "text-ok", bgColor: "bg-ok/10", borderColor: "border-ok/40" };
    default:
      return { label: category || "Unknown", color: "text-muted", bgColor: "bg-panel-2", borderColor: "border-line" };
  }
}

// -- API error formatting --

/**
 * Extracts a clean, user-friendly error message from a REST API error thrown
 * by {@link unwrap} (services.ts), which already unpacks Huma's RFC 9457
 * problem-details body (`detail` + per-field `errors[]`) into `err.message`.
 * This function only handles the optional context prefix:
 *   Prepends a context prefix (e.g., "Failed to register webhook"), unless
 *   the message already starts with something similar (avoids "Failed to
 *   register webhook: failed to register webhook").
 *
 * @param err - The caught error (typically an Error thrown by unwrap)
 * @param contextPrefix - Optional context like "Failed to register webhook"
 * @returns A clean, actionable error message string
 */
function formatAPIError(err: unknown, contextPrefix?: string): string {
  const msg = err instanceof Error ? err.message : String(err);

  if (!msg) {
    return contextPrefix ?? "An unexpected error occurred";
  }

  // If no prefix requested, return the raw message
  if (!contextPrefix) {
    return msg;
  }

  // Avoid double-prefix: if the message already starts with something similar
  // to the prefix (case-insensitive), skip the prefix.
  const prefixLower = contextPrefix.toLowerCase().replace(/^failed to\s+/i, "");
  const msgLower = msg.toLowerCase();
  if (msgLower.startsWith("failed to " + prefixLower) || msgLower.startsWith(prefixLower)) {
    return msg;
  }

  return `${contextPrefix}: ${msg}`;
}

/**
 * Relative "time ago" label for a timestamp (e.g. "2m ago", "3h ago").
 * Falls back to a locale date for anything older than ~30 days.
 */
function timeAgo(timestamp: string | null | undefined): string {
  if (!timestamp) return "—";
  const d = new Date(timestamp);
  const ms = d.getTime();
  if (isNaN(ms)) return "—";
  const diff = Date.now() - ms;
  const sec = Math.round(diff / 1000);
  if (sec < 0) return "just now";
  if (sec < 45) return `${sec}s ago`;
  const min = Math.round(sec / 60);
  if (min < 60) return `${min}m ago`;
  const hr = Math.round(min / 60);
  if (hr < 24) return `${hr}h ago`;
  const day = Math.round(hr / 24);
  if (day < 30) return `${day}d ago`;
  return d.toLocaleDateString();
}

export {
  ERROR_CATEGORIES,
  JSONSchemaMetaSchema,
  formatAPIError,
  getCategoryBadge,
  getCategoryDisplay,
  jsonToJsonSchema,
  stringifyContent,
  toJSONObject,
  timeAgo,
};
