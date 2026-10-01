// Shared blog index: every post under src/pages/posts, sorted newest first,
// with reading time and tag slugs so the index, tag pages, RSS feed and
// article layout all agree on the same list.

export interface PostFrontmatter {
  title: string;
  kicker?: string;
  description?: string;
  author?: string;
  pubDate: string | Date;
  tags?: string[];
}

export interface Post {
  url: string;
  slug: string;
  frontmatter: PostFrontmatter;
  date: Date;
  minutes: number;
  tags: string[];
}

interface PostModule {
  url: string;
  file: string;
  frontmatter: PostFrontmatter;
  rawContent: () => string;
}

const WORDS_PER_MINUTE = 220;

export const base = import.meta.env.BASE_URL.replace(/\/?$/, '/');

export const tagSlug = (tag: string) => tag.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '');
export const tagUrl = (tag: string) => `${base}blog/tags/${tagSlug(tag)}/`;

const readingMinutes = (markdown: string) => {
  const prose = markdown.replace(/```[\s\S]*?```/g, ' ');
  const words = prose.split(/\s+/).filter(Boolean).length;
  return Math.max(1, Math.round(words / WORDS_PER_MINUTE));
};

// Lazy glob: the post pages import BlogArticleLayout, which imports this
// module, so an eager glob would be a circular import.
const loaders = import.meta.glob<PostModule>('../../pages/posts/*.md');

let cache: Promise<Post[]> | undefined;

// Newest first; posts published the same day fall back to title order so the
// list is stable between builds.
export const getPosts = (): Promise<Post[]> =>
  (cache ??= Promise.all(Object.values(loaders).map((load) => load())).then((modules) =>
    modules
      .map((m) => ({
        url: m.url.replace(/\/?$/, '/'),
        slug: m.file.split('/').pop()!.replace(/\.md$/, ''),
        frontmatter: m.frontmatter,
        date: new Date(m.frontmatter.pubDate),
        minutes: readingMinutes(m.rawContent()),
        tags: (m.frontmatter.tags ?? []).map((t) => t.toLowerCase()),
      }))
      .sort((a, b) => b.date.valueOf() - a.date.valueOf() || a.frontmatter.title.localeCompare(b.frontmatter.title)),
  ));

export const allTags = (posts: Post[]): { tag: string; count: number }[] => {
  const counts = new Map<string, number>();
  for (const p of posts) for (const t of p.tags) counts.set(t, (counts.get(t) ?? 0) + 1);
  return [...counts].map(([tag, count]) => ({ tag, count })).sort((a, b) => b.count - a.count || a.tag.localeCompare(b.tag));
};

export const formatDate = (d: Date, month: 'long' | 'short' = 'long') =>
  d.toLocaleDateString('en-US', { year: 'numeric', month, day: 'numeric', timeZone: 'UTC' });

export const findPost = (posts: Post[], url: string | undefined) => {
  const clean = (u: string) => u.replace(/\/$/, '');
  return posts.findIndex((p) => url && clean(p.url) === clean(url));
};
