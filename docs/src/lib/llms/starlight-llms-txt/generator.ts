import type { APIContext } from 'astro';
import { getCollection } from 'astro:content';
import micromatch from 'micromatch';
import { starlightLllmsTxtContext } from 'virtual:starlight-llms-txt/context';
import { entryToSimpleMarkdown } from './entryToSimpleMarkdown';
import { defaultLang, isDefaultLocale } from './utils';

const collator = new Intl.Collator(defaultLang);

export async function generateLlmsTxt(
  context: APIContext,
  {
    minify,
    description,
    exclude,
    include,
  }: {
    minify: boolean;
    description: string | undefined;
    exclude?: string[] | undefined;
    include?: string[] | undefined;
  }
): Promise<string> {
  let docs = await getCollection('docs', (doc) => isDefaultLocale(doc) && !doc.data.draft);
  if (include) docs = docs.filter((doc) => micromatch.isMatch(doc.id, include));
  if (exclude) docs = docs.filter((doc) => !micromatch.isMatch(doc.id, exclude));
  const { promote, demote, pageSeparator } = starlightLllmsTxtContext;
  const prioritizePages = (id: string) => {
    const demoted = demote.findIndex((expr) => micromatch.isMatch(id, expr));
    const promoted = demoted > -1 ? -1 : promote.findIndex((expr) => micromatch.isMatch(id, expr));
    const prefixLength = (promoted > -1 ? promote.length - promoted : 0) + demote.length - demoted - 1;
    return '_'.repeat(prefixLength) + id;
  };
  docs.sort((a, b) => collator.compare(prioritizePages(a.id), prioritizePages(b.id)));
  const segments: string[] = [];
  for (const doc of docs) {
    const docSegments = [`# ${doc.data.hero?.title || doc.data.title}`];
    const docDescription = doc.data.hero?.tagline || doc.data.description;
    if (docDescription) docSegments.push(`> ${docDescription}`);
    docSegments.push(await entryToSimpleMarkdown(doc, context, minify));
    segments.push(docSegments.join('\n\n'));
  }
  if (description) segments.unshift(description);
  return segments.join(pageSeparator);
}
