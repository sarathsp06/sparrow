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

export {
  JSONSchemaMetaSchema,
  jsonToJsonSchema,
  stringifyContent,
  toJSONObject,
};
