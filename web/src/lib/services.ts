import { env } from "$env/dynamic/public";
import createClient from "openapi-fetch";
import type { paths } from "./api-types";

// Runtime config injected by the Go server into window.__SPARROW_CONFIG__.
// The API key is always provided at runtime (never baked into the build).
interface SparrowConfig {
  apiKey?: string;
}

declare global {
  interface Window {
    __SPARROW_CONFIG__?: SparrowConfig;
  }
}

const runtimeConfig: SparrowConfig =
  (typeof window !== "undefined" && window.__SPARROW_CONFIG__) || {};

const apiKey: string = runtimeConfig.apiKey || "";

// Single typed REST client for the whole app. Sparrow's interface is
// REST/OpenAPI only (Connect-RPC and gRPC have been removed).
export const api = createClient<paths>({
  baseUrl: env.PUBLIC_API_URL || "/",
  headers: apiKey ? { "X-API-Key": apiKey } : undefined,
});

/**
 * Throws a readable Error when an openapi-fetch call returns `error`.
 *
 * Huma's error body (RFC 9457 Problem Details) puts a generic summary in
 * `detail` (e.g. "validation failed") and the actionable per-field messages
 * in `errors[]` (e.g. {location: "body.events", message: "expected array
 * length >= 1"}). Append them so the user sees what actually went wrong.
 */
export function unwrap<T>(result: { data?: T; error?: unknown; response: Response }): T {
  if (result.error !== undefined) {
    const err = result.error as
      | { detail?: string; title?: string; errors?: { location?: string; message?: string }[] }
      | undefined;
    let message = err?.detail || err?.title || `Request failed (${result.response.status})`;
    if (err?.errors?.length) {
      const details = err.errors
        .map((e) => (e.location ? `${e.location}: ${e.message}` : e.message))
        .filter(Boolean)
        .join("; ");
      if (details) message = `${message} (${details})`;
    }
    throw new Error(message);
  }
  if (result.data === undefined) {
    throw new Error(`Request failed (${result.response.status})`);
  }
  return result.data;
}
