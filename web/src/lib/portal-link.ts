export interface PortalSession {
  token: string;
  consumer: string;
  expiresAt: Date | null;
}

/** Access-token secret prefix (TOKEN_PREFIX in access/client.ts). */
const ACCESS_TOKEN_PREFIX = "sparrow_tk_";

/**
 * Reads a portal link fragment ("#token=...&consumer=...&expires=<unix>", the
 * portal_path of a consumer token from POST /v1/tokens). The consumer and
 * expiry are only for display and URL building, the server pins
 * every request to the token's own consumer, so a hand-built "#token=..."
 * link without them still works (API calls route by the token; only the
 * consumer name and expiry are missing from the page).
 */
export function parsePortalLink(fragment: string): PortalSession | null {
  const params = new URLSearchParams(fragment.replace(/^#/, ""));
  const token = params.get("token") ?? "";
  if (!token.startsWith(ACCESS_TOKEN_PREFIX)) return null;

  const consumer = params.get("consumer") ?? "";
  const expUnix = Number(params.get("expires") ?? NaN);
  return {
    token,
    consumer,
    expiresAt: Number.isFinite(expUnix) ? new Date(expUnix * 1000) : null,
  };
}
