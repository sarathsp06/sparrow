import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import rehypeMermaidLite from 'rehype-mermaid-lite';
import proto2astroConfig from './src/data/proto2astro-config.json';

export default defineConfig({
  site: proto2astroConfig.site,
  base: proto2astroConfig.base,
  markdown: {
    syntaxHighlight: {
      type: 'shiki',
      excludeLangs: ['mermaid'],
    },
    rehypePlugins: [rehypeMermaidLite],
  },
  integrations: [
    starlight({
      title: proto2astroConfig.title,
      description: proto2astroConfig.description,
      social: proto2astroConfig.social,
      editLink: { baseUrl: proto2astroConfig.editLink },
      sidebar: proto2astroConfig.sidebar,
      components: proto2astroConfig.components,
      customCss: ['./src/styles/custom.css'],
      head: [
        {
          tag: 'script',
          attrs: {
            type: 'module',
          },
          content: `
import mermaid from 'https://cdn.jsdelivr.net/npm/mermaid@11/dist/mermaid.esm.min.mjs';
mermaid.initialize({
  startOnLoad: false,
  theme: 'default',
  fontFamily: 'Fira Code, ui-monospace, monospace',
  fontSize: 14,
});
async function renderMermaid() {
  if (document.querySelectorAll('.mermaid').length > 0) {
    await mermaid.run({ querySelector: '.mermaid' });
  }
}
renderMermaid();
document.addEventListener('astro:page-load', renderMermaid);
`,
        },
      ],
    }),
  ],
});
