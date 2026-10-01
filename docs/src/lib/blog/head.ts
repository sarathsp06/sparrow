// Extra <head> entries for blog pages. Starlight already emits the canonical
// link, Open Graph and Twitter tags from the page title and description.

type HeadEntry = { tag: 'meta' | 'link' | 'script'; attrs?: Record<string, string>; content?: string };

export const blogHead = (site: URL, base: string, jsonLd: Record<string, unknown>, extra: HeadEntry[] = []): HeadEntry[] => [
  {
    tag: 'link',
    attrs: { rel: 'alternate', type: 'application/rss+xml', title: 'Sparrow field notes', href: new URL(`${base}blog/rss.xml`, site).href },
  },
  { tag: 'script', attrs: { type: 'application/ld+json' }, content: JSON.stringify(jsonLd) },
  ...extra,
];
