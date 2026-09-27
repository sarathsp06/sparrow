import { dev } from "$app/environment";
import { replaceState } from "$app/navigation";
import { env } from "$env/dynamic/public";
import createClient from "openapi-fetch";
import type { paths } from "./api-types";
import { apiLogMiddleware } from "./apiConsole.svelte";
import { auth } from "./access/auth.svelte";
import { browserTokenName, fragmentParam, InviteError, redeemInvite, rejectReason, TOKEN_PREFIX } from "./access/client";
import { portalInvite } from "./access/portal-invite.svelte";
import { parsePortalToken, type PortalSession } from "./portal-token";
import { apiHref, portalGatewayURL, resolveApiBase, type SparrowConfig } from "./runtime-config";

// Runtime config: set in the static /config.js. See runtime-config.ts for
// precedence.
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
// portal HTML). Tokens arrive in the URL fragment, so they never hit server
// logs:
//   - #token=spt_v2... : a stateless portal link (POST .../portal-token),
//     kept in sessionStorage to survive reloads;
//   - #invite=sparrow_inv_... : a one-time invite, exchanged below for a
//     consumer access token that is remembered in localStorage.
const PORTAL_TOKEN_STORAGE = "sparrow_portal_token";
const PORTAL_SESSION_STORAGE = "sparrow_portal_session";

function storedPortalSession(): PortalSession | null {
  try {
    const raw = localStorage.getItem(PORTAL_SESSION_STORAGE);
    if (!raw) return null;
    const s = JSON.parse(raw) as { token: string; consumer: string; expiresAt: string | null };
    if (!s.token?.startsWith(TOKEN_PREFIX) || !s.consumer) return null;
    return { token: s.token, consumer: s.consumer, expiresAt: s.expiresAt ? new Date(s.expiresAt) : null };
  } catch {
    return null;
  }
}

function initPortal(): PortalSession | null {
  if (typeof window === "undefined" || !window.location.pathname.startsWith("/portal")) return null;
  const fromHash = fragmentParam(window.location.hash, "token");
  if (fromHash) sessionStorage.setItem(PORTAL_TOKEN_STORAGE, fromHash);
  const link = parsePortalToken(fromHash || sessionStorage.getItem(PORTAL_TOKEN_STORAGE) || "");
  return link ?? storedPortalSession();
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
    async onResponse({ response }) {
      if (response.status === 401) auth.reject(rejectReason(await response.clone().json().catch(() => null)));
      return response;
    },
  });
}
api.use(apiLogMiddleware);

// Invite links: /#invite=<secret> (operator console) and
// /portal#invite=<secret> (a consumer's portal), minted with POST /v1/invites
// or `sparrow invite`. The invite is exchanged once for a new access token
// named after the invitee, which is remembered in this browser.
//
// SvelteKit's router re-records the initial URL (fragment included) when it
// starts, so the fragment is dropped with a real navigation on success and
// with the router's own replaceState on failure; plain history.replaceState
// would be undone and a reload would try the spent invite again.
function dropFragment() {
  const clean = location.pathname + location.search;
  try {
    replaceState(clean, {});
  } catch {
    history.replaceState(history.state, "", clean);
  }
}

const INVITE_FAILED = "This invite link is invalid, expired, cancelled, or has already been used. Ask for a new one.";

async function redeemConsoleInvite(secret: string) {
  auth.beginRedeem();
  try {
    const res = await redeemInvite(apiBase, secret);
    if (res.scope) {
      // A consumer invite opened on the console: continue in that portal.
      localStorage.setItem(PORTAL_SESSION_STORAGE, JSON.stringify({ token: res.token, consumer: res.scope, expiresAt: res.expires_at }));
      location.replace("/portal");
      return;
    }
    auth.save(res.token, res.token_id);
    location.replace(location.pathname + location.search);
  } catch (e) {
    dropFragment();
    auth.failRedeem(e instanceof InviteError ? INVITE_FAILED : "Could not reach the Sparrow server to use this invite. Reload to try again.");
  }
}

async function redeemPortalInvite(secret: string) {
  portalInvite.start();
  try {
    const res = await redeemInvite(apiBase, secret);
    if (!res.scope) throw new InviteError("This invite is for the operator console, not a portal.");
    localStorage.setItem(PORTAL_SESSION_STORAGE, JSON.stringify({ token: res.token, consumer: res.scope, expiresAt: res.expires_at }));
    sessionStorage.removeItem(PORTAL_TOKEN_STORAGE);
    location.replace("/portal");
  } catch (e) {
    dropFragment();
    portalInvite.fail(e instanceof InviteError ? INVITE_FAILED : "Could not reach the server to use this invite. Reload to try again.");
  }
}

if (typeof window !== "undefined") {
  const invite = fragmentParam(window.location.hash, "invite");
  if (invite && window.location.pathname.startsWith("/portal")) void redeemPortalInvite(invite);
  else if (invite && !portal) void redeemConsoleInvite(invite);
}

/** Label for a token created when someone pastes the master key into the prompt. */
export const browserSignInName = () => browserTokenName(typeof navigator === "undefined" ? "" : navigator.userAgent);

/**
 * Signs this browser out. A token created for this browser (invite or
 * sign-in) is revoked on the server too; a pasted token or key is only
 * forgotten locally, since it may be in use elsewhere.
 */
export async function signOut() {
  const owned = auth.ownedTokenId;
  if (owned) {
    await api.DELETE("/v1/tokens/{token_id}", { params: { path: { token_id: owned } } }).catch(() => {});
  }
  auth.forget();
  location.reload();
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
      | { detail?: string; message?: string; title?: string; errors?: { location?: string; message?: string }[] }
      | undefined;
    let message = err?.detail || err?.message || err?.title || `Request failed (${result.response.status})`;
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
