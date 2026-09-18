import mdxServer from '@astrojs/mdx/server.js';
import type { APIContext } from 'astro';
import { experimental_AstroContainer } from 'astro/container';
import { render, type CollectionEntry } from 'astro:content';
import type { RootContent } from 'hast';
import { matches, select, selectAll } from 'hast-util-select';
import rehypeParse from 'rehype-parse';
import rehypeRemark from 'rehype-remark';
import remarkGfm from 'remark-gfm';
import remarkStringify from 'remark-stringify';
import { unified } from 'unified';
import { remove } from 'unist-util-remove';
import { starlightLllmsTxtContext } from 'virtual:starlight-llms-txt/context';

const minifyDefaults = {
  note: true,
  tip: true,
  caution: false,
  danger: false,
  details: true,
  whitespace: true,
  customSelectors: [],
};
const minify = { ...minifyDefaults, ...starlightLllmsTxtContext.minify };
const selectors = [...minify.customSelectors];
if (minify.details) selectors.unshift('details');

const astroContainer = await experimental_AstroContainer.create({
  renderers: [{ name: 'astro:jsx', ssr: mdxServer }],
});

const htmlToMarkdownPipeline = unified()
  .use(rehypeParse, { fragment: true })
  .use(function minifyLlmsTxt() {
    return (tree, file) => {
      if (!file.data.starlightLlmsTxt.minify) return;
      remove(tree, (_node) => {
        const node = _node as RootContent;
        for (const selector of selectors) {
          if (matches(selector, node)) return true;
        }
        if (matches('.starlight-aside', node)) {
          for (const variant of ['note', 'tip', 'caution', 'danger'] as const) {
            if (minify[variant] && matches(`.starlight-aside--${variant}`, node)) return true;
          }
        }
        return false;
      });
      return tree;
    };
  })
  .use(function improveExpressiveCodeHandling() {
    return (tree) => {
      const ecInstances = selectAll('.expressive-code', tree as Parameters<typeof selectAll>[1]);
      for (const instance of ecInstances) {
        const figcaption = select('figcaption', instance);
        if (figcaption) {
          const terminalWindowTextIndex = figcaption.children.findIndex((child) =>
            matches('span.sr-only', child)
          );
          if (terminalWindowTextIndex > -1) figcaption.children.splice(terminalWindowTextIndex, 1);
        }
        const pre = select('pre', instance);
        const code = select('code', instance);
        if (pre?.properties.dataLanguage && code) {
          if (!Array.isArray(code.properties.className)) code.properties.className = [];
          const diffLines =
            pre.properties.dataLanguage === 'diff'
              ? []
              : code.children.filter((child) => matches('div.ec-line.ins, div.ec-line.del', child));
          if (diffLines.length === 0) {
            code.properties.className.push(`language-${pre.properties.dataLanguage}`);
          } else {
            code.properties.className.push('language-diff');
            for (const line of diffLines) {
              if (line.type !== 'element') continue;
              const classes = line.properties?.className;
              if (typeof classes !== 'string' && !Array.isArray(classes)) continue;
              const marker = classes.includes('ins') ? '+' : '-';
              const span = select('span:not(.indent)', line);
              const firstChild = span?.children[0];
              if (firstChild?.type === 'text') firstChild.value = `${marker}${firstChild.value}`;
            }
          }
        }
      }
    };
  })
  .use(function improveTabsHandling() {
    return (tree) => {
      const tabInstances = selectAll('starlight-tabs', tree as Parameters<typeof selectAll>[1]);
      for (const instance of tabInstances) {
        const tabs = selectAll('[role="tab"]', instance);
        const panels = selectAll('[role="tabpanel"]', instance);
        instance.tagName = 'ul';
        instance.properties = {};
        instance.children = [];
        for (let i = 0; i < Math.min(tabs.length, panels.length); i++) {
          const tab = tabs[i];
          const panel = panels[i];
          if (!tab || !panel) continue;
          const tabLabel = tab.children
            .filter((child) => child.type === 'text' && child.value.trim())
            .map((child) => child.type === 'text' && child.value.trim())
            .join('');
          instance.children.push({
            type: 'element',
            tagName: 'li',
            properties: {},
            children: [
              { type: 'element', tagName: 'p', children: [{ type: 'text', value: tabLabel }], properties: {} },
              panel,
            ],
          });
        }
      }
    };
  })
  .use(function improveFileTreeHandling() {
    return (tree) => {
      const trees = selectAll('starlight-file-tree', tree as Parameters<typeof selectAll>[1]);
      for (const treeNode of trees) {
        remove(treeNode, (_node) => {
          const node = _node as RootContent;
          return matches('.sr-only', node);
        });
      }
    };
  })
  .use(rehypeRemark)
  .use(remarkGfm)
  .use(remarkStringify);

export async function entryToSimpleMarkdown(
  entry: CollectionEntry<'docs'>,
  context: APIContext,
  shouldMinify: boolean = false
) {
  const { rawContent } = starlightLllmsTxtContext;
  if (rawContent) return entry.body || '';

  const { Content } = await render(entry);
  const html = await astroContainer.renderToString(Content, context);
  const file = await htmlToMarkdownPipeline.process({
    value: html,
    data: { starlightLlmsTxt: { minify: shouldMinify } },
  });
  let markdown = String(file).trim();
  if (shouldMinify && minify.whitespace) markdown = markdown.replace(/\s+/g, ' ');
  return markdown;
}
