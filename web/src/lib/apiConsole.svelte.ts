// Captures every request the app's API client makes, so the UI can show a
// "console" of what it just did as a copyable curl command — handy for
// scripting the same action or filing a bug report with exact repro steps.
// Redacts the API key; it's a dev/debugging aid, not a secrets viewer.
import type { Middleware } from "openapi-fetch";

export interface ApiLogEntry {
  id: string;
  time: number;
  method: string;
  url: string;
  headers: [string, string][];
  body?: string;
  /** HTTP status once the response lands; undefined while in flight. */
  status?: number;
  durationMs?: number;
  /** Consecutive repeats of this exact request, coalesced into one row. */
  count?: number;
}

const MAX_ENTRIES = 50;
let entries = $state<ApiLogEntry[]>([]);
const started = new Map<string, number>();

export const apiConsole = {
  get entries() {
    return entries;
  },
  clear() {
    entries = [];
  },
};

export function toCurl(entry: ApiLogEntry): string {
  const parts = [`curl '${entry.url}'`];
  if (entry.method !== "GET") parts.push(`  --request ${entry.method}`);
  for (const [name, value] of entry.headers) {
    const shown = name.toLowerCase() === "x-api-key" ? "YOUR_API_KEY" : value;
    parts.push(`  --header '${name}: ${shown}'`);
  }
  if (entry.body) parts.push(`  --data '${entry.body}'`);
  return parts.join(" \\\n");
}

function settle(id: string, status?: number) {
  const startedAt = started.get(id);
  started.delete(id);
  entries = entries.map((e) =>
    e.id === id
      ? { ...e, status, durationMs: startedAt ? Math.round(performance.now() - startedAt) : undefined }
      : e,
  );
}

export const apiLogMiddleware: Middleware = {
  async onRequest({ request, id }) {
    let body: string | undefined;
    if (request.body) {
      try {
        body = await request.clone().text();
        // Pretty-print JSON bodies; stays valid inside the curl single quotes.
        body = JSON.stringify(JSON.parse(body), null, 2);
      } catch {
        // best-effort only; keep whatever we managed to read
      }
    }
    started.set(id, performance.now());
    const next: ApiLogEntry = {
      id,
      time: Date.now(),
      method: request.method,
      url: request.url,
      headers: [...request.headers.entries()],
      body: body || undefined,
    };
    // Coalesce a repeat of an existing entry (poll loops like /v1/stats and
    // /v1/health-summary, which alternate) into one row with a xN counter,
    // bumped in place, instead of flooding the log.
    const prev = entries.findIndex(
      (e) => e.method === next.method && e.url === next.url && e.body === next.body,
    );
    if (prev !== -1) {
      // Keep next.id so the newest response's settle() finds this row;
      // the superseded request's settle is intentionally dropped.
      started.delete(entries[prev].id);
      entries = entries.with(prev, { ...next, count: (entries[prev].count ?? 1) + 1 });
      return;
    }
    entries = [next, ...entries].slice(0, MAX_ENTRIES);
  },
  async onResponse({ response, id }) {
    settle(id, response.status);
  },
  async onError({ id }) {
    settle(id, undefined);
  },
};
