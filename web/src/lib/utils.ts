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
      return { label: "4xx", classes: "bg-orange-50 text-orange-700 border-orange-200" };
    case "server_error":
      return { label: "5xx", classes: "bg-red-50 text-red-700 border-red-200" };
    case "timeout":
      return { label: "Timeout", classes: "bg-yellow-50 text-yellow-700 border-yellow-200" };
    case "dns_error":
      return { label: "DNS", classes: "bg-purple-50 text-purple-700 border-purple-200" };
    case "tls_error":
      return { label: "TLS", classes: "bg-purple-50 text-purple-700 border-purple-200" };
    case "connection_refused":
      return { label: "Conn Refused", classes: "bg-purple-50 text-purple-700 border-purple-200" };
    case "network_error":
      return { label: "Network", classes: "bg-purple-50 text-purple-700 border-purple-200" };
    case "unexpected_status":
      return { label: "Unexpected Status", classes: "bg-amber-50 text-amber-700 border-amber-200" };
    case "success":
      return { label: "Success", classes: "bg-green-50 text-green-700 border-green-200" };
    default:
      return { label: category || "Unknown", classes: "bg-gray-50 text-gray-700 border-gray-200" };
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
      return { label: "4xx Client Error", color: "text-orange-700", bgColor: "bg-orange-50", borderColor: "border-orange-200" };
    case "server_error":
      return { label: "5xx Server Error", color: "text-red-700", bgColor: "bg-red-50", borderColor: "border-red-200" };
    case "timeout":
      return { label: "Timeout", color: "text-yellow-700", bgColor: "bg-yellow-50", borderColor: "border-yellow-200" };
    case "dns_error":
      return { label: "DNS Error", color: "text-purple-700", bgColor: "bg-purple-50", borderColor: "border-purple-200" };
    case "tls_error":
      return { label: "TLS Error", color: "text-purple-700", bgColor: "bg-purple-50", borderColor: "border-purple-200" };
    case "connection_refused":
      return { label: "Connection Refused", color: "text-purple-700", bgColor: "bg-purple-50", borderColor: "border-purple-200" };
    case "network_error":
      return { label: "Network Error", color: "text-purple-700", bgColor: "bg-purple-50", borderColor: "border-purple-200" };
    case "unexpected_status":
      return { label: "Unexpected Status Code", color: "text-amber-700", bgColor: "bg-amber-50", borderColor: "border-amber-200" };
    case "success":
      return { label: "Success", color: "text-green-700", bgColor: "bg-green-50", borderColor: "border-green-200" };
    default:
      return { label: category || "Unknown", color: "text-gray-700", bgColor: "bg-gray-50", borderColor: "border-gray-200" };
  }
}

export {
  ERROR_CATEGORIES,
  JSONSchemaMetaSchema,
  getCategoryBadge,
  getCategoryDisplay,
  jsonToJsonSchema,
  stringifyContent,
  toJSONObject,
};
