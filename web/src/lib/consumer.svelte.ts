// Global consumer context — an optional scoping filter.
// One switcher in the shell; every list page reads this. Empty value means
// "all consumers" (the default) and routes to the global list endpoints.
import { browser } from "$app/environment";

const KEY = "sparrow.consumer";
const DEFAULT = "default";

let current = $state(browser ? (localStorage.getItem(KEY) ?? "") : "");
// Consumers discovered at runtime (from loaded webhooks/events) to power the switcher list.
let known = $state<string[]>([DEFAULT]);

export const consumerStore = {
  /** Active consumer; empty string means all consumers. */
  get value(): string {
    return current;
  },
  set value(v: string) {
    const next = v.trim();
    current = next;
    if (browser) localStorage.setItem(KEY, next);
    this.remember(next);
  },
  /** Human label for the current scope. */
  get label(): string {
    return current || "all consumers";
  },
  get options(): string[] {
    return known;
  },
  /** Merge freshly seen consumers into the switcher list. */
  remember(...names: (string | undefined | null)[]) {
    const set = new Set(known);
    for (const n of names) if (n && n.trim()) set.add(n.trim());
    set.add(DEFAULT);
    const next = [...set].sort();
    if (next.length !== known.length || next.some((n, i) => n !== known[i])) known = next;
  },
};
