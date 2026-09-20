import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import sitemap from '@astrojs/sitemap';
import svelte from '@astrojs/svelte';
import react from '@astrojs/react';
import starlightLlmsTxt from './src/lib/llms/starlight-llms-txt/index.mjs';
import { llmsBasePrompt, llmsRootDetails } from './src/lib/llms/prompts.mjs';

export default defineConfig({
  site: 'https://sarathsp06.github.io',
  base: '/sparrow',
  redirects: {
    '/tools': '/sparrow/tools/recipe-workbench/',
    '/tools/email-studio': '/sparrow/tools/recipe-workbench/',
  },
  vite: {
    // The workbench imports the recipe catalog straight from satellites/recipes/*.yaml.
    server: { fs: { allow: ['..'] } },
  },
  integrations: [
    starlight({
      title: 'Sparrow',
      logo: {
        src: './src/assets/favicon.svg',
      },
      description: 'Self-hosted webhook delivery platform',
      plugins: [starlightLlmsTxt({
        projectName: 'Sparrow',
        description: llmsBasePrompt,
        details: llmsRootDetails,
      })],
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/sarathsp06/sparrow',
        },
      ],
      editLink: { baseUrl: 'https://github.com/sarathsp06/sparrow/edit/main/docs/' },

      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { slug: 'getting-started/why-sparrow' },
            { slug: 'getting-started/installation' },
            { slug: 'getting-started/quickstart' },
            { slug: 'getting-started/how-it-works' },
            { slug: 'getting-started/configuration' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { slug: 'guides/payload-transformation' },
            { slug: 'guides/webhook-health-alerts' },
            { slug: 'guides/portal-embedding' },
          ],
        },
        {
          label: 'Satellites',
          items: [
            { slug: 'satellites' },
            { slug: 'satellites/cli' },
            { slug: 'satellites/recipes' },
            { slug: 'satellites/sources' },
            { slug: 'satellites/sinks' },
          ],
        },
        {
          label: 'Interactive Apps',
          items: [
            { label: 'Email Recipe Studio', link: '/tools/email-studio/' },
            { label: 'Recipe Workbench', link: '/tools/recipe-workbench/' },
          ],
        },
        {
          label: 'API Reference',
          link: '/reference/api',
          badge: { text: 'OpenAPI', variant: 'note' },
        },
        {
          label: 'Reference',
          items: [
            { slug: 'reference/client-libraries' },
            { slug: 'reference/template-functions' },
            { slug: 'reference/error-classification' },
            { slug: 'reference/architecture' },
            { slug: 'reference/security' },
            { slug: 'reference/layered-architecture' },
          ],
        },
        {
          label: 'Deployment',
          items: [
            { slug: 'deployment/docker-compose' },
            { slug: 'deployment/security' },
          ],
        },
      ],
      components: {
        Footer: './src/components/Footer.astro',
      },
      head: [
        {
          tag: 'link',
          attrs: {
            rel: 'alternate',
            type: 'text/markdown',
            href: '/sparrow/llms.txt',
            title: 'llms.txt',
          },
        },
      ],
      customCss: ['./src/styles/custom.css'],
      expressiveCode: {
        themes: ['github-light', 'github-dark'],
        useStarlightDarkModeSwitch: true,
        styleOverrides: {
          borderColor: 'var(--sp-outline-variant, #e4e2dd)',
          borderRadius: '0.5rem',
          codeBackground: 'var(--sp-code-bg, #f3f1ec)',
          frames: {
            terminalTitlebarBackground: 'var(--sp-surface-container-high, #f3f1ec)',
            terminalTitlebarBorderBottomColor: 'var(--sp-outline-variant, #e4e2dd)',
            terminalTitlebarDotsForeground: 'var(--sp-border-hover, #cfccc4)',
            terminalBackground: 'var(--sp-code-bg, #f3f1ec)',
            editorTabBarBackground: 'var(--sp-surface-container-high, #f3f1ec)',
            editorActiveTabBackground: 'var(--sp-code-bg, #f3f1ec)',
            editorActiveTabIndicatorTopColor: 'var(--sp-brand-primary-text, #b06a10)',
            editorTabBarBorderBottomColor: 'var(--sp-outline-variant, #e4e2dd)',
            frameBoxShadowCssValue: 'none',
          },
        },
      },
    }),
    sitemap(),
    svelte(),
    react(),
  ],
});
