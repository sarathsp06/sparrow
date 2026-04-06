import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import proto2astroConfig from './src/data/proto2astro-config.json';

export default defineConfig({
  site: proto2astroConfig.site,
  base: proto2astroConfig.base,
  markdown: {
    syntaxHighlight: {
      type: 'shiki',
      excludeLangs: ['mermaid'],
    },
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
  theme: 'dark',
  themeVariables: {
    primaryColor: '#4f46e5',
    primaryTextColor: '#e0e7ff',
    primaryBorderColor: '#818cf8',
    lineColor: '#818cf8',
    secondaryColor: '#1e1b4b',
    tertiaryColor: '#1a1a2e',
    fontFamily: 'Fira Code, ui-monospace, monospace',
    fontSize: '14px',
  },
});
async function renderMermaid() {
  const blocks = document.querySelectorAll('pre[data-language="mermaid"]');
  for (const pre of blocks) {
    const wrapper = pre.closest('.expressive-code') || pre.parentElement;
    const copyBtn = wrapper.querySelector('[data-code]');
    const raw = copyBtn ? copyBtn.getAttribute('data-code') : pre.textContent;
    const container = document.createElement('div');
    container.classList.add('mermaid');
    container.textContent = raw;
    wrapper.replaceWith(container);
  }
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
