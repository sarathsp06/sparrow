// Social preview images in the docs' paper palette (sparrow-tokens.css), rendered
// at build time by astro-og-canvas: /og/site.png, the site card (src/lib/
// social-card.mjs), and /og/<entry id>.png for each docs page, which
// src/route-data.ts points its og:image at. The field notes have their own,
// dark, route in src/pages/blog/og/.
import { getCollection } from 'astro:content';
import { OGImageRoute } from 'astro-og-canvas';

const entries = await getCollection('docs');

// Keep the description to roughly three lines at the rendered size.
const clamp = (text: string, max = 170) => {
  if (text.length <= max) return text;
  const cut = text.slice(0, max).replace(/\s+\S*$/, '');
  return `${cut}…`;
};

export const { getStaticPaths, GET } = await OGImageRoute({
  param: 'route',
  pages: {
    site: {
      title: 'Sparrow',
      description: 'Self-hosted webhook delivery: classified retries, HMAC and Ed25519 signing, OpenTelemetry. One Go binary and PostgreSQL.',
    },
    ...Object.fromEntries(entries.map((e) => [e.id, e.data])),
  },
  getImageOptions: (_path, page) => ({
    title: page.title,
    description: clamp(page.description ?? 'Self-hosted webhook delivery'),
    logo: { path: './src/assets/og-logo.png', size: [96] },
    bgGradient: [[250, 249, 247]],
    border: { color: [242, 169, 59], width: 14, side: 'block-start' },
    padding: 72,
    fonts: [
      'https://api.fontsource.org/v1/fonts/space-grotesk/latin-700-normal.ttf',
      'https://api.fontsource.org/v1/fonts/inter/latin-400-normal.ttf',
    ],
    font: {
      title: { color: [26, 26, 26], size: 66, lineHeight: 1.12, weight: 'Bold', families: ['Space Grotesk'] },
      description: { color: [85, 82, 76], size: 30, lineHeight: 1.4, weight: 'Normal', families: ['Inter'] },
    },
  }),
});
