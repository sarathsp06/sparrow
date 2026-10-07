import type { APIRoute } from 'astro';
import { base, getPosts } from '../../lib/blog/posts';

const esc = (s: string) => s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');

export const GET: APIRoute = async ({ site }) => {
  const posts = await getPosts();
  const link = (path: string) => new URL(path, site).href;
  // Readers render the item away from our origin, so root-relative links must be absolute.
  const absolutize = (html: string) => html.replace(/(href|src)="\/(?!\/)/g, (_m, attr) => `${attr}="${new URL('/', site).href}`);
  const items = posts.map((p) => `
    <item>
      <title>${esc(p.frontmatter.title)}</title>
      <link>${link(p.url)}</link>
      <guid isPermaLink="true">${link(p.url)}</guid>
      <pubDate>${p.date.toUTCString()}</pubDate>
      <description>${esc(p.frontmatter.description ?? '')}</description>
      ${p.frontmatter.author ? `<dc:creator>${esc(p.frontmatter.author)}</dc:creator>` : ''}
      ${p.tags.map((t) => `<category>${esc(t)}</category>`).join('')}
      <enclosure url="${link(`${base}blog/og/${p.slug}.png`)}" type="image/png" length="0" />
      <content:encoded><![CDATA[${absolutize(p.html)}]]></content:encoded>
    </item>`).join('');
  const xml = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>Sparrow field notes</title>
    <link>${link(`${base}blog/`)}</link>
    <atom:link href="${link(`${base}blog/rss.xml`)}" rel="self" type="application/rss+xml" />
    <description>Stories and engineering notes from the people building Sparrow.</description>
    <language>en</language>
    <image>
      <url>${link(`${base}og.png`)}</url>
      <title>Sparrow field notes</title>
      <link>${link(`${base}blog/`)}</link>
    </image>${items}
  </channel>
</rss>`;
  return new Response(xml, { headers: { 'Content-Type': 'application/rss+xml; charset=utf-8' } });
};
