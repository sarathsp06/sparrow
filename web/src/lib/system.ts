// Sparrow's own plumbing: the reserved _sparrow consumer its system events
// (sparrow.*) are pushed under, where operators register the alert mailer.
// It is listed and counted like any other consumer; these helpers only label
// it where it shows up.
import { api, unwrap } from "$lib/services";
import type { components } from "$lib/api-types";

type EventTypeItem = components["schemas"]["EventTypeItem"];
export type HealthRules = components["schemas"]["HealthRules"];

export const SYSTEM_CONSUMER = "_sparrow";
export const SYSTEM_CONSUMER_LABEL = "Sparrow alerts";

export const ALERTS_GUIDE_URL = "https://sarathsp06.github.io/sparrow/guides/webhook-health-alerts/";
export const VERIFY_SIGNATURES_GUIDE_URL = "https://sarathsp06.github.io/sparrow/guides/verify-signatures/";
/** Register page pre-filled with the SendGrid recipe under _sparrow. */
export const ALERT_SETUP_HREF = `/webhooks/register?recipe=sendgrid&consumer=${SYSTEM_CONSUMER}`;

export const ALERT_EVENT_TYPES = [
  "sparrow.webhook.health_changed",
  "sparrow.webhook.delivery_failed",
  "sparrow.webhook.disabled",
];

export function isSystemConsumer(consumer: string | undefined | null): boolean {
  return consumer === SYSTEM_CONSUMER;
}

export function isSystemEvent(name: string | undefined | null): boolean {
  return !!name && name.toLowerCase().startsWith("sparrow.");
}

/**
 * A plain local@domain check, run before anything is created. Loose on
 * purpose (ops@localhost passes, as it does on the server); the server's own
 * check has the final say.
 */
export function isEmail(value: string): boolean {
  return /^[^\s@<>()]+@[^\s@<>()]+$/.test(value.trim());
}

/**
 * Every event type matching query. The API pages them (50 by default, 1000
 * at most per page), so a picker that reads only the first page would
 * silently drop the rest.
 */
export async function listAllEventTypes(query: { active_only?: boolean; consumer?: string } = {}): Promise<EventTypeItem[]> {
  const limit = 500;
  const items: EventTypeItem[] = [];
  for (let offset = 0; ; offset += limit) {
    const page = unwrap(await api.GET("/v1/event-types", { params: { query: { ...query, limit, offset } } }));
    items.push(...(page.items ?? []));
    if (!page.pagination?.has_more || !page.items?.length) return items;
  }
}

const pct = (r: number) => `${Math.round(r * 100)}%`;

/** The server's health rules in words, so the copy cannot drift from them. */
export function describeHealthRules(r: HealthRules): { unhealthy: string; degraded: string; healthy: string } {
  return {
    unhealthy: `${r.unhealthy_consecutive_failures}+ failed attempts in a row, or under ${pct(r.unhealthy_success_rate)} success over ${r.unhealthy_min_attempts}+ attempts in the last ${r.window_hours}h`,
    degraded: `under ${pct(r.degraded_success_rate)} success over ${r.degraded_min_attempts}+ attempts in the last ${r.window_hours}h`,
    healthy: `${pct(r.degraded_success_rate)}+ success over ${r.healthy_min_attempts}+ attempts in the last ${r.window_hours}h`,
  };
}

/** Colour tone for a success rate, on the same cutoffs as the health labels. */
export function successRateTone(rate: number, r: HealthRules | undefined): "ok" | "warn" | "bad" {
  const degraded = r?.degraded_success_rate ?? 0.9;
  const unhealthy = r?.unhealthy_success_rate ?? 0.8;
  if (rate >= degraded) return "ok";
  if (rate >= unhealthy) return "warn";
  return "bad";
}
