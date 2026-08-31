// Global namespace context — the app's primary scoping axis.
// One switcher in the shell; every list/filter page reads this instead of
// re-declaring a local free-text namespace input.
import { browser } from "$app/environment";

const KEY = "sparrow.namespace";
const DEFAULT = "default";

let current = $state(browser ? localStorage.getItem(KEY) || DEFAULT : DEFAULT);
// Namespaces discovered at runtime (from loaded webhooks/events) to power the switcher list.
let known = $state<string[]>([DEFAULT]);

export const namespaceStore = {
  get value(): string {
    return current;
  },
  set value(v: string) {
    const next = v.trim() || DEFAULT;
    current = next;
    if (browser) localStorage.setItem(KEY, next);
    this.remember(next);
  },
  get options(): string[] {
    return known;
  },
  /** Merge freshly seen namespaces into the switcher list. */
  remember(...names: (string | undefined | null)[]) {
    const set = new Set(known);
    for (const n of names) if (n && n.trim()) set.add(n.trim());
    set.add(DEFAULT);
    const next = [...set].sort();
    if (next.length !== known.length || next.some((n, i) => n !== known[i])) known = next;
  },
};
