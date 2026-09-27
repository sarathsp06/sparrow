// Operator-console credential state.
//
// The credential is one of:
//   - window.__SPARROW_CONFIG__.apiKey: injected by the Go server when it
//     serves the UI (SPARROW_UI_INJECT_KEY, on by default), or set in the
//     standalone UI's /config.js.
//   - A stored credential (localStorage, remembered across restarts): an
//     access token from an invite link, a token created when someone pasted
//     the master key into the sign-in prompt, or whatever was pasted if the
//     server cannot mint tokens.
// A stored credential wins over the injected one: it is the more recent,
// explicit choice.
//
// Tokens created for this browser (invite or master-key exchange) are
// "owned": signing out revokes them on the server, not just locally.
import type { SparrowConfig } from "../runtime-config";
import { rejectMessage, type RejectReason } from "./client";

declare global {
  interface Window {
    __SPARROW_CONFIG__?: SparrowConfig;
  }
}

/** localStorage keys. */
export const KEY_STORAGE = "sparrow_api_key";
export const OWNED_TOKEN_STORAGE = "sparrow_api_key_token_id";

function read(key: string): string {
  try {
    return localStorage.getItem(key) ?? "";
  } catch {
    return "";
  }
}

function write(key: string, value: string) {
  try {
    if (value) localStorage.setItem(key, value);
    else localStorage.removeItem(key);
  } catch {
    // Private mode / storage disabled: the credential still works for this page load.
  }
}

const browser = typeof window !== "undefined";
const injected = (browser && window.__SPARROW_CONFIG__?.apiKey) || "";
let stored = $state(browser ? read(KEY_STORAGE) : "");
let ownedTokenId = $state(browser ? read(OWNED_TOKEN_STORAGE) : "");
let required = $state(false);
let message = $state("");
let redeeming = $state(false);
let inviteFailed = false;

export const auth = {
  /** The credential to send, or "" for none. */
  get key() {
    return stored || injected;
  },
  /** True when a credential is stored in this browser (so it can be signed out). */
  get hasStoredKey() {
    return stored !== "";
  },
  /** Id of the token created for this browser, if any (revoked on sign-out). */
  get ownedTokenId() {
    return ownedTokenId;
  },
  /** True once the API rejected a request with 401: the UI shows the sign-in prompt. */
  get required() {
    return required;
  },
  /** Why sign-in is needed, for the prompt. */
  get message() {
    return message;
  },
  /** The server rejected the current credential. */
  reject(reason?: RejectReason) {
    // Requests racing an invite redemption fail with 401; the page reloads
    // with the new token once redemption succeeds, so don't flash the prompt.
    if (redeeming) return;
    // Keep an invite failure's explanation: later 401s from the same page
    // load would otherwise replace it with a generic "key required".
    if (!inviteFailed) message = rejectMessage(reason, auth.key !== "");
    required = true;
  },
  beginRedeem() {
    redeeming = true;
  },
  /** An invite link could not be used. */
  failRedeem(text: string) {
    redeeming = false;
    inviteFailed = true;
    message = text;
    required = true;
  },
  dismiss() {
    required = false;
  },
  /** Stores a credential. Pass tokenId when the token was created for this browser. */
  save(key: string, tokenId = "") {
    stored = key.trim();
    ownedTokenId = tokenId;
    required = false;
    write(KEY_STORAGE, stored);
    write(OWNED_TOKEN_STORAGE, ownedTokenId);
  },
  /** Forgets the stored credential locally. */
  forget() {
    stored = "";
    ownedTokenId = "";
    write(KEY_STORAGE, "");
    write(OWNED_TOKEN_STORAGE, "");
  },
};
