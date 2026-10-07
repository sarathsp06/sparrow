// Per-post social preview images at /blog/og/<slug>.png, rendered at build
// time by astro-og-canvas in the field notes' dark palette (sparrow-tokens.css).
import { OGImageRoute } from 'astro-og-canvas';

import { getPosts } from '../../../lib/blog/posts';

const posts = await getPosts();

// Keep the standfirst to roughly three lines at the rendered size.
const clamp = (text: string, max = 170) => {
  if (text.length <= max) return text;
  const cut = text.slice(0, max).replace(/\s+\S*$/, '');
  return `${cut}…`;
};

export const { getStaticPaths, GET } = await OGImageRoute({
  param: 'route',
  pages: Object.fromEntries(posts.map((p) => [p.slug, p.frontmatter])),
  getImageOptions: (_path, page) => ({
    title: page.title,
    description: clamp(page.description ?? 'Sparrow field notes'),
    logo: { path: './src/assets/og-logo.png', size: [96] },
    bgGradient: [[11, 18, 32]],
    border: { color: [242, 169, 59], width: 14, side: 'block-end' },
    padding: 72,
    // Same faces the site uses (see BlogLayout.astro), fetched once and cached.
    fonts: [
      'https://api.fontsource.org/v1/fonts/space-grotesk/latin-700-normal.ttf',
      'https://api.fontsource.org/v1/fonts/inter/latin-400-normal.ttf',
    ],
    font: {
      title: { color: [230, 236, 245], size: 66, lineHeight: 1.12, weight: 'Bold', families: ['Space Grotesk'] },
      description: { color: [132, 148, 176], size: 30, lineHeight: 1.4, weight: 'Normal', families: ['Inter'] },
    },
  }),
});
