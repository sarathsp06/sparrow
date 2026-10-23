// Starlight route middleware: gives each docs page its own social preview image,
// rendered by src/pages/og/[...route].ts. Pages without one (the 404 page) get
// the site card. Starlight sets og:title/og:description and twitter:card itself.
import { getCollection } from 'astro:content';
import { defineRouteMiddleware } from '@astrojs/starlight/route-data';
import { socialCard } from './lib/social-card.mjs';

const ids = new Set((await getCollection('docs')).map((e) => e.id));

export const onRequest = defineRouteMiddleware((context) => {
  const { entry, head } = context.locals.starlightRoute;
  const base = import.meta.env.BASE_URL.replace(/\/?$/, '/');
  const own = ids.has(entry.id);
  const image = own ? new URL(`${base}og/${entry.id}.png`, context.site).href : socialCard.url;
  const alt = own ? `${entry.data.title} — Sparrow docs` : socialCard.alt;
  const meta = (key: 'property' | 'name', name: string, content: string) =>
    head.push({ tag: 'meta', attrs: { [key]: name, content } });
  meta('property', 'og:image', image);
  meta('property', 'og:image:width', socialCard.width);
  meta('property', 'og:image:height', socialCard.height);
  meta('property', 'og:image:alt', alt);
  meta('name', 'twitter:image', image);
});
