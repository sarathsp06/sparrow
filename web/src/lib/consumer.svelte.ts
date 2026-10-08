// Consumer scoping for list pages. A tenant can have hundreds of consumers,
// so there is no global switcher: each list page has its own consumer filter,
// kept in the page URL (?consumer=) so a filtered view can be linked and
// survives a reload. Empty means all consumers.
import { goto } from "$app/navigation";
import { page } from "$app/state";
import { SYSTEM_CONSUMER, SYSTEM_CONSUMER_LABEL } from "$lib/system";

const PARAM = "consumer";

export const consumerFilter = {
  /** The page's consumer filter; empty string means all consumers. */
  get value(): string {
    return page.url.searchParams.get(PARAM)?.trim() ?? "";
  },
  set value(v: string) {
    const next = v.trim();
    if (next === this.value) return;
    const url = new URL(page.url);
    if (next) url.searchParams.set(PARAM, next);
    else url.searchParams.delete(PARAM);
    goto(url, { replaceState: true, keepFocus: true, noScroll: true });
  },
  /** Human label for the current scope. */
  get label(): string {
    return this.value ? consumerLabel(this.value) : "all consumers";
  },
};

/** A link to path that keeps consumer as its consumer filter, when set. */
export function withConsumer(path: string, consumer: string): string {
  if (!consumer) return path;
  const sep = path.includes("?") ? "&" : "?";
  return `${path}${sep}${PARAM}=${encodeURIComponent(consumer)}`;
}

/** A consumer's name, marked when it is Sparrow's own alert-delivery consumer. */
export function consumerLabel(consumer: string): string {
  return consumer === SYSTEM_CONSUMER ? `${consumer} (${SYSTEM_CONSUMER_LABEL})` : consumer;
}
