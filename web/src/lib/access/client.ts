// Framework-free helpers for Sparrow access tokens and invites (server side:
// pkg/access + internal/accessauth). No Svelte or SvelteKit imports, so they
// can be unit-tested with node --test and reused outside this app.

/** Prefix of Sparrow access-token secrets. */
export const TOKEN_PREFIX = "sparrow_tk_";

/** Why the server rejected a credential (the `reason` field of a 401 body). */
export type RejectReason = "missing" | "invalid" | "expired" | "revoked";

/** Reads `name=value` from a URL fragment like "#invite=abc&x=1"; "" if absent. */
export function fragmentParam(hash: string, name: string): string {
  const m = hash.match(new RegExp(`(?:^#|[#&])${name}=([^&]+)`));
  if (!m) return "";
  try {
    return decodeURIComponent(m[1]);
  } catch {
    return m[1];
  }
}

/** Extracts the rejection reason from a 401 response body, if any. */
export function rejectReason(body: unknown): RejectReason | undefined {
  const r = (body as { reason?: unknown } | null)?.reason;
  return r === "missing" || r === "invalid" || r === "expired" || r === "revoked" ? r : undefined;
}

/** Sign-in prompt text for a rejection reason (and whether a key was sent at all). */
export function rejectMessage(reason: RejectReason | undefined, hadKey: boolean): string {
  switch (reason) {
    case "revoked":
      return "Your access to this Sparrow server was revoked. Ask for a new invite, or enter an API key.";
    case "expired":
      return "Your access to this Sparrow server has expired. Ask for a new invite, or enter an API key.";
    case "invalid":
      return "The Sparrow server rejected the current key. Enter its SPARROW_API_KEY or an access token.";
    default:
      return hadKey
        ? "The Sparrow server rejected the current key. Enter its SPARROW_API_KEY or an access token."
        : "This Sparrow server requires an API key. Enter its SPARROW_API_KEY or an access token.";
  }
}

/** A short, human label for a browser token, e.g. "Browser sign-in (Chrome on macOS)". */
export function browserTokenName(userAgent: string): string {
  const browser = /Edg\//.test(userAgent)
    ? "Edge"
    : /Firefox\//.test(userAgent)
      ? "Firefox"
      : /Chrome\//.test(userAgent)
        ? "Chrome"
        : /Safari\//.test(userAgent)
          ? "Safari"
          : "browser";
  const os = /Windows/.test(userAgent)
    ? "Windows"
    : /Mac OS X|Macintosh/.test(userAgent)
      ? "macOS"
      : /Android/.test(userAgent)
        ? "Android"
        : /iPhone|iPad/.test(userAgent)
          ? "iOS"
          : /Linux/.test(userAgent)
            ? "Linux"
            : "";
  return `Browser sign-in (${os ? `${browser} on ${os}` : browser})`;
}

/** Result of redeeming an invite: a new token, shown once. */
export interface RedeemedInvite {
  token: string;
  token_id: string;
  name: string;
  /** Consumer the token is limited to; null for tenant-wide. */
  scope: string | null;
  expires_at: string | null;
}

/** Thrown when an invite is invalid, expired, cancelled, or already used. */
export class InviteError extends Error {}

/** Exchanges an invite secret for its token (POST /invite/redeem). */
export async function redeemInvite(apiBase: string, invite: string, fetchFn: typeof fetch = fetch): Promise<RedeemedInvite> {
  const res = await fetchFn(`${apiBase}/invite/redeem`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ invite }),
  });
  const body = (await res.json().catch(() => ({}))) as Partial<RedeemedInvite> & { message?: string };
  if (res.status === 400 || res.status === 404) throw new InviteError(body.message || "This invite can't be used.");
  if (!res.ok || !body.token) throw new Error(body.message || `Could not redeem the invite (${res.status})`);
  return body as RedeemedInvite;
}

/** Outcome of swapping a pasted key for a browser token. */
export type ExchangeResult =
  | { kind: "token"; secret: string; tokenId: string }
  /** The key was wrong. */
  | { kind: "rejected"; reason?: RejectReason }
  /** Store the key as typed: it is already a token, or the server cannot mint tokens (older server). */
  | { kind: "keep" };

/**
 * Swaps a pasted master key for a named, revocable browser token
 * (POST /v1/tokens), so the master key itself is never stored in the browser.
 * Tokens are stored as they are.
 */
export async function exchangeKey(apiBase: string, key: string, name: string, fetchFn: typeof fetch = fetch): Promise<ExchangeResult> {
  if (key.startsWith(TOKEN_PREFIX)) {
    const res = await fetchFn(`${apiBase}/v1/whoami`, { headers: { "X-API-Key": key } });
    if (res.status === 401) return { kind: "rejected", reason: rejectReason(await res.json().catch(() => null)) };
    return { kind: "keep" };
  }
  const res = await fetchFn(`${apiBase}/v1/tokens`, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-API-Key": key },
    body: JSON.stringify({ name }),
  });
  if (res.status === 401) return { kind: "rejected", reason: rejectReason(await res.json().catch(() => null)) };
  if (res.status === 201) {
    const body = (await res.json()) as { secret: string; token: { id: string } };
    return { kind: "token", secret: body.secret, tokenId: body.token.id };
  }
  return { kind: "keep" };
}
