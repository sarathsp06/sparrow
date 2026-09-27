import { dev } from "$app/environment";
import { replaceState } from "$app/navigation";
import { env } from "$env/dynamic/public";
import createClient from "openapi-fetch";
import type { paths } from "./api-types";
import { apiLogMiddleware } from "./apiConsole.svelte";
import { auth } from "./auth.svelte";
import { parsePortalToken, type PortalSession } from "./portal-token";
import { apiHref, portalGatewayURL, resolveApiBase, type SparrowConfig } from "./runtime-config";

// Runtime config: injected inline by the Go server (embedded UI) or set in the
// static /config.js (standalone UI). See runtime-config.ts for precedence.
const runtimeConfig: SparrowConfig =
  (typeof window !== "undefined" && window.__SPARROW_CONFIG__) || {};

class SameOriginRequest extends Request {
  constructor(input: RequestInfo | URL, init?: RequestInit) {
    super(typeof input === "string" ? new URL(input, location.href).href : input, init);
  }
}

/** API base URL without trailing slash; "" means same origin as the UI. */
export const apiBase = resolveApiBase(runtimeConfig.apiUrl, env.PUBLIC_API_URL, dev);

/** Absolute link to a server-side page such as "/docs" (on the API host, which may differ from the UI's). */
export const serverHref = (path: string) => apiHref(apiBase, path);

// Portal mode: pages under /portal authenticate with a consumer-scoped
// bearer token instead of the admin API key (which is never injected into
// portal HTML). The token arrives in the URL fragment (#token=...) so it
// never hits server logs, and is kept in sessionStorage to survive reloads.
// Supported token format:
// - spt_v2.<key-id>.<b64url(consumer)>.<unix-expiry>.<b64url(signature)>
function initPortal(): PortalSession | null {
  if (typeof window === "undefined" || !window.location.pathname.startsWith("/portal")) return null;
  const fromHash = window.location.hash.match(/(?:^#|[#&])token=([^&]+)/)?.[1] ?? "";
  if (fromHash) sessionStorage.setItem("sparrow_portal_token", fromHash);
  const token = fromHash || sessionStorage.getItem("sparrow_portal_token") || "";
  return parsePortalToken(token);
}

export const portal = initPortal();


// Portal path gateway: in portal mode every API call goes out under the single
// /portal/api/ prefix instead of /v1/..., so an operator embedding the portal
// allowlists just one API path (plus /portal and /_app) with no per-consumer
// rules. The consumer is carried by the bearer token, so we drop it from the
// URL here; the server's PortalGateway maps /portal/api/<rest> back to the real
// /v1 path. This covers every call site (including shared components) with no
// per-call changes.
async function rewritePortalURL(request: Request, session: PortalSession) {
  const target = portalGatewayURL(request.url, apiBase, session.consumer, window.location.href);
  if (!target) return request;

  const body = request.method === "GET" || request.method === "HEAD" ? undefined : await request.clone().arrayBuffer();
  return new SameOriginRequest(target, { method: request.method, headers: request.headers, body, signal: request.signal });
}

// Single typed REST client for the whole app. Sparrow's interface is
// REST/OpenAPI only (Connect-RPC and gRPC have been removed).
export const api = createClient<paths>({
  baseUrl: apiBase,
  Request: SameOriginRequest,
  headers: portal ? { Authorization: `Bearer ${portal.token}` } : undefined,
});
if (portal) {
  const session = portal;
  api.use({ onRequest: ({ request }) => rewritePortalURL(request, session) });
} else {
  // Admin auth: attach the current key on every request (it can change at
  // runtime via the sign-in prompt) and open the prompt when the server says 401.
  api.use({
    onRequest({ request }) {
      if (auth.key) request.headers.set("X-API-Key", auth.key);
      return request;
    },
    onResponse({ response }) {
      if (response.status === 401) auth.markRequired();
      return response;
    },
  });
}
api.use(apiLogMiddleware);

// One-time access link: /#access=<token> (minted with POST /v1/access-links or
// `sparrow access-link create`). Exchange the token for the API key once,
// remember the key like one typed into the prompt, and continue on the same
// page without the token. The token lives in the fragment, so it never
// reaches server logs.
//
// SvelteKit's router re-records the initial URL (fragment included) when it
// starts, so the fragment is dropped with a real navigation on success and
// with the router's own replaceState on failure — plain history.replaceState
// would be undone and a reload would redeem the link a second time.
async function redeemAccessLink(token: string) {
  auth.beginRedeem();
  const clean = location.pathname + location.search;
  try {
    const res = await fetch(`${apiBase}/access-link/redeem`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ token }),
    });
    const body = (await res.json().catch(() => ({}))) as { api_key?: string };
    if (!res.ok || !body.api_key) throw new Error(`status ${res.status}`);
    auth.save(body.api_key);
    location.replace(clean); // full load without the fragment
  } catch {
    try {
      replaceState(clean, {});
    } catch {
      history.replaceState(history.state, "", clean);
    }
    auth.failRedeem("This access link is invalid, expired, or has already been used. Ask for a new one, or enter the API key.");
  }
}

if (!portal && typeof window !== "undefined") {
  const token = window.location.hash.match(/(?:^#|[#&])access=([^&]+)/)?.[1];
  if (token) void redeemAccessLink(decodeURIComponent(token));
}


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
  if (result.data === undefined && result.response.status !== 204) {
    throw new Error(`Request failed (${result.response.status})`);
  }
  return result.data as T;
}
