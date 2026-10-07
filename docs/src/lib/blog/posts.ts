// Shared blog index: every post under src/blog (served at /blog/<slug>/), sorted newest first,
// with reading time and tag slugs so the index, tag pages, RSS feed and
// article layout all agree on the same list.

import type { MarkdownHeading, MarkdownInstance } from 'astro';

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

type PostModule = MarkdownInstance<PostFrontmatter>;

const WORDS_PER_MINUTE = 220;

export const base = import.meta.env.BASE_URL.replace(/\/?$/, '/');

export const tagSlug = (tag: string) => tag.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
export const tagUrl = (tag: string) => `${base}blog/tags/${tagSlug(tag)}/`;

const readingMinutes = (markdown: string) => {
  const prose = markdown.replace(/```[\s\S]*?```/g, ' ');
  const words = prose.split(/\s+/).filter(Boolean).length;
  return Math.max(1, Math.round(words / WORDS_PER_MINUTE));
};

const loaders = import.meta.glob<PostModule>('../../blog/*.md');

let cache: Promise<Post[]> | undefined;

// Newest first; posts published the same day fall back to title order so the
// list is stable between builds.
export const getPosts = (): Promise<Post[]> =>
  (cache ??= Promise.all(Object.values(loaders).map((load) => load())).then(async (modules) => {
    const posts = await Promise.all(
      modules.map(async (m): Promise<Post> => {
        const slug = m.file.split('/').pop()!.replace(/\.md$/, '');
        const date = new Date(m.frontmatter.pubDate);
        return {
          url: `${base}blog/${slug}/`,
          slug,
          frontmatter: m.frontmatter,
          date,
          modified: m.frontmatter.updated ? new Date(m.frontmatter.updated) : date,
          minutes: readingMinutes(m.rawContent()),
          tags: (m.frontmatter.tags ?? []).map((t) => t.toLowerCase()),
          Content: m.Content,
          html: await m.compiledContent(),
          headings: m.getHeadings(),
        };
      }),
    );
    return posts.sort((a, b) => b.date.valueOf() - a.date.valueOf() || a.frontmatter.title.localeCompare(b.frontmatter.title));
  }));

export const allTags = (posts: Post[]): { tag: string; count: number }[] => {
  const counts = new Map<string, number>();
  for (const p of posts) for (const t of p.tags) counts.set(t, (counts.get(t) ?? 0) + 1);
  return [...counts].map(([tag, count]) => ({ tag, count })).sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag));
};

export const formatDate = (d: Date, month: 'long' | 'short' = 'long') =>
  d.toLocaleDateString('en-US', { year: 'numeric', month, day: 'numeric', timeZone: 'UTC' });
