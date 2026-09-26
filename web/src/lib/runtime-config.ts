// Pure helpers for resolving where the UI talks to and how it authenticates.
// Kept free of SvelteKit imports so they can be unit-tested with node --test.
//
// Config sources, highest precedence first:
//   1. window.__SPARROW_CONFIG__ — set at runtime, either injected inline by
//      the Go server (embedded UI) or by the static /config.js file that ships
//      with the build (standalone UI; edit it at deploy time, no rebuild).
//   2. PUBLIC_API_URL — baked into the bundle at `vite build` time.
//   3. Defaults: http://localhost:8080 under `vite dev`, same origin otherwise.

export interface SparrowConfig {
  /** Absolute URL of the Sparrow API (e.g. https://sparrow-api.example.com). */
  apiUrl?: string;
  /** Admin API key sent as X-API-Key. */
  apiKey?: string;
}

/** localStorage key holding an API key the operator typed into the sign-in prompt. */
export const API_KEY_STORAGE_KEY = "sparrow_api_key";

/**
 * Resolves the API base URL with no trailing slash. An empty string means
 * "same origin as the UI" (openapi-fetch then issues root-relative paths).
 */
export function resolveApiBase(runtimeUrl: string | undefined, buildUrl: string | undefined, dev: boolean): string {
  const raw = (runtimeUrl || buildUrl || (dev ? "http://localhost:8080" : "")).trim();
  return raw.replace(/\/+$/, "");
}

/** Absolute URL of a server-side path (e.g. "/docs") on the API host. */
export function apiHref(apiBase: string, path: string): string {
  return apiBase + path;
}

/** Path prefix of the API base (e.g. "/sparrow" for https://host/sparrow), "" for none. */
function basePath(apiBase: string, origin: string): string {
  return new URL(apiBase || "/", origin).pathname.replace(/\/+$/, "");
}

/**
 * Maps an admin-style request URL (<apiBase>/v1/...) to the portal gateway
 * (<apiBase>/portal/api/...). Consumer-scoped paths drop the consumer segment
 * because the gateway takes it from the bearer token. Returns null when the
 * URL is not a /v1 API call and should be sent unchanged.
 */
export function portalGatewayURL(requestUrl: string, apiBase: string, consumer: string, origin: string): string | null {
  const url = new URL(requestUrl, origin);
  const prefix = basePath(apiBase, origin);
  if (!url.pathname.startsWith(prefix + "/v1/")) return null;

  const rest = url.pathname.slice(prefix.length);
  const scoped = [`/v1/consumers/${consumer}/`, `/v1/consumers/${encodeURIComponent(consumer)}/`].find((s) => rest.startsWith(s));
  const tail = scoped ? rest.slice(scoped.length) : rest.slice("/v1/".length);
  url.pathname = `${prefix}/portal/api/${tail}`;
  return url.href;
}
