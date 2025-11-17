
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
   * This schema defines the structure and rules that a valid JSON Schema must adhere to.
   * This is a simplified version and may not cover all aspects of the full JSON Schema specification
   */
	const JSONSchemaMetaSchema ={
  "type": "object",
  "properties": {
    "type": {
      "type": "string",
      "enum": ["object"]
    },
    "properties": {
      "type": "object",
      "additionalProperties": { "$ref": "#" }
    },
    "required": {
      "type": "array",
      "items": { "type": "string" }
    }
  },
  "required": ["properties"],
  "additionalProperties": true
}


export { JSONSchemaMetaSchema, stringifyContent };




