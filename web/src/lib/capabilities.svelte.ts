// GET /v1/capabilities, the server's optional features, in one place.
// Callers that ask while a request is in flight share it, so a page that
// needs several flags makes one request. Each later call asks again: alert
// delivery can be set up while the UI is open.
import { api, portal, unwrap } from "./services";
import type { components } from "./api-types";

export type Capabilities = components["schemas"]["CapabilitiesOutputBody"];

let current = $state<Capabilities | null>(null);
let failed = $state(false);
let inflight: Promise<Capabilities> | null = null;

export function loadCapabilities(): Promise<Capabilities> {
  inflight ??= api
    .GET("/v1/capabilities")
    .then((res) => {
      current = unwrap(res);
      failed = false;
      return current;
    })
    .catch((e) => {
      failed = true;
      throw e;
    })
    .finally(() => {
      inflight = null;
    });
  return inflight;
}

// Whether this page may retry deliveries. The console always may (admin
// credentials). A portal may when the last capabilities answer says
// portal.retry (SPARROW_PORTAL_RETRY), and stays disabled until one arrives.
// If capabilities could not be read, retry is enabled: the server still
// decides, and a refused retry shows the server's 403 message.
export const retryPolicy = {
  get allowed() {
    if (!portal) return true;
    if (current) return current.portal.retry;
    return failed;
  },
  /** Shown on the disabled retry buttons. */
  reason: "Retrying is turned off for this portal. Ask the operator to retry these deliveries.",
};
