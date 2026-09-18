import type { CollectionEntry } from 'astro:content';
import { starlightLllmsTxtContext } from 'virtual:starlight-llms-txt/context';

const { defaultLocale, locales, title } = starlightLllmsTxtContext;
export const defaultLang = (defaultLocale === 'root' ? locales?.root?.lang : defaultLocale) || 'en';

export function getSiteTitle(): string {
  return typeof title === 'string' ? title : (title[defaultLang] as string);
}

const localeKeys = Object.keys(locales || {}).filter((key) => key !== 'root' && key !== defaultLang);
const startsWithLocaleRE = new RegExp(`^(${localeKeys.join('|')})/`);

export function isDefaultLocale(doc: CollectionEntry<'docs'>): boolean {
  return !(localeKeys.includes(doc.id) || startsWithLocaleRE.test(doc.id));
}

export function ensureTrailingSlash(path: string) {
  return path.at(-1) === '/' ? path : path + '/';
}
