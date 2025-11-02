
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

  export { stringifyContent };

