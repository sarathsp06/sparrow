// Shared blog index: every post under src/blog (.md, or .mdx to use components
// from src/components/blog/content; served at /blog/<slug>/), sorted newest first,
// with reading time and tag slugs so the index, tag pages, RSS feed and
// article layout all agree on the same list.

import type { MarkdownHeading, MarkdownInstance } from 'astro';
import { experimental_AstroContainer as AstroContainer } from 'astro/container';
import { loadRenderers } from 'astro:container';
import { getContainerRenderer as mdxRenderer } from '@astrojs/mdx/container-renderer';
import { getContainerRenderer as svelteRenderer } from '@astrojs/svelte/container-renderer';

export interface PostFrontmatter {
  title: string;
  kicker?: string;
  description?: string;
  author?: string; // GitHub username
  pubDate: string | Date;
  /** Set when a post is materially revised; feeds dateModified and the sitemap. */
  updated?: string | Date;
  tags?: string[];
}

export interface Post {
  url: string;
  slug: string;
  frontmatter: PostFrontmatter;
  date: Date;
  /** `updated` if set, else the publish date. */
  modified: Date;
  minutes: number;
  tags: string[];
  Content: MarkdownInstance<PostFrontmatter>['Content'];
  /** Rendered HTML of the body, for the RSS feed. */
  html: string;
  headings: MarkdownHeading[];
}

// The parts .md and .mdx modules share. MDX has no rawContent()/compiledContent(),
// so reading time comes from the raw source and the RSS HTML from a render.
type PostModule = Pick<MarkdownInstance<PostFrontmatter>, 'file' | 'frontmatter' | 'Content' | 'getHeadings'>;

const WORDS_PER_MINUTE = 220;

export const base = import.meta.env.BASE_URL.replace(/\/?$/, '/');

export const tagSlug = (tag: string) => tag.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
export const tagUrl = (tag: string) => `${base}blog/tags/${tagSlug(tag)}/`;

const readingMinutes = (source: string) => {
  const prose = source
    .replace(/^---[\s\S]*?^---$/m, ' ') // frontmatter
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/^(import|export) .*$/gm, ' ');
  const words = prose.split(/\s+/).filter(Boolean).length;
  return Math.max(1, Math.round(words / WORDS_PER_MINUTE));
};

const loaders = import.meta.glob<PostModule>('../../blog/*.{md,mdx}');
const sources = import.meta.glob<string>('../../blog/*.{md,mdx}', { query: '?raw', import: 'default' });

let cache: Promise<Post[]> | undefined;

// Newest first; posts published the same day fall back to title order so the
// list is stable between builds.
export const getPosts = (): Promise<Post[]> =>
  (cache ??= (async () => {
    const container = await AstroContainer.create({ renderers: await loadRenderers([mdxRenderer(), svelteRenderer()]) });
    const posts = await Promise.all(
      Object.entries(loaders).map(async ([path, load]): Promise<Post> => {
        const [m, source] = await Promise.all([load(), sources[path]()]);
        const slug = m.file.split('/').pop()!.replace(/\.mdx?$/, '');
        const date = new Date(m.frontmatter.pubDate);
        return {
          url: `${base}blog/${slug}/`,
          slug,
          frontmatter: m.frontmatter,
          date,
          modified: m.frontmatter.updated ? new Date(m.frontmatter.updated) : date,
          minutes: readingMinutes(source),
          tags: (m.frontmatter.tags ?? []).map((t) => t.toLowerCase()),
          Content: m.Content,
          html: await container.renderToString(m.Content),
          headings: m.getHeadings(),
        };
      }),
    );
    return posts.sort((a, b) => b.date.valueOf() - a.date.valueOf() || a.frontmatter.title.localeCompare(b.frontmatter.title));
  })());

export const allTags = (posts: Post[]): { tag: string; count: number }[] => {
  const counts = new Map<string, number>();
  for (const p of posts) for (const t of p.tags) counts.set(t, (counts.get(t) ?? 0) + 1);
  return [...counts].map(([tag, count]) => ({ tag, count })).sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag));
};

export const formatDate = (d: Date, month: 'long' | 'short' = 'long') =>
  d.toLocaleDateString('en-US', { year: 'numeric', month, day: 'numeric', timeZone: 'UTC' });

// Up to `limit` other posts sharing the most tags with `post`, newest first on
// ties. Posts with no tag in common are left out.
export const relatedPosts = (post: Post, posts: Post[], limit = 2): Post[] =>
  posts
    .filter((p) => p.slug !== post.slug)
    .map((p) => ({ p, shared: p.tags.filter((t) => post.tags.includes(t)).length }))
    .filter(({ shared }) => shared > 0)
    .sort((a, b) => b.shared - a.shared || b.p.date.valueOf() - a.p.date.valueOf())
    .slice(0, limit)
    .map(({ p }) => p);
