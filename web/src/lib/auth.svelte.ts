// Admin API key state for the operator console.
//
// The key comes from one of two places:
//   - window.__SPARROW_CONFIG__.apiKey — injected by the Go server when it
//     serves the UI (SPARROW_SERVE_UI=true), or set in /config.js for a
//     standalone deployment.
//   - localStorage — typed by the operator into the sign-in prompt, which
//     opens when the API answers 401. This is how a standalone UI talks to a
//     server with SPARROW_API_KEY set without baking the key into any file.
//
// A key the operator typed wins over the injected one: it is the more recent,
// explicit choice (e.g. after the server key was rotated).
import { API_KEY_STORAGE_KEY, type SparrowConfig } from "./runtime-config";

declare global {
  interface Window {
    __SPARROW_CONFIG__?: SparrowConfig;
  }
}

function readStored(): string {
  try {
    return localStorage.getItem(API_KEY_STORAGE_KEY) ?? "";
  } catch {
    return "";
  }
}

const injected = (typeof window !== "undefined" && window.__SPARROW_CONFIG__?.apiKey) || "";
let stored = $state(typeof window !== "undefined" ? readStored() : "");
let required = $state(false);

export const auth = {
  /** The key to send, or "" for none. */
  get key() {
    return stored || injected;
  },
  /** True when a key was saved from the sign-in prompt (so it can be forgotten). */
  get hasStoredKey() {
    return stored !== "";
  },
  /** True once the API rejected a request with 401 — the UI shows the sign-in prompt. */
  get required() {
    return required;
  },
  markRequired() {
    required = true;
  },
  dismiss() {
    required = false;
  },
  save(key: string) {
    stored = key.trim();
    required = false;
    try {
      localStorage.setItem(API_KEY_STORAGE_KEY, stored);
    } catch {
      // Private mode / storage disabled: the key still works for this page load.
    }
  },
  forget() {
    stored = "";
    try {
      localStorage.removeItem(API_KEY_STORAGE_KEY);
    } catch {
      // ignore
    }
  },
};
