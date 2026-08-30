import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';


export default defineConfig({
  site: 'https://sarathsp06.github.io',
  base: '/sparrow',
  integrations: [
    starlight({
      title: 'Sparrow',
      logo: {
        src: './src/assets/favicon.svg',
      },
      description: 'Self-hosted webhook delivery platform',
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
          ],
        },
        {
          label: 'Deployment',
          items: [
            { slug: 'deployment/docker-compose' },
            { slug: 'deployment/railway' },
            { slug: 'deployment/kubernetes' },
          ],
        },
      ],
      components: {
        Footer: './src/components/Footer.astro',
      },
      customCss: ['./src/styles/custom.css'],
      expressiveCode: {
        themes: ['github-light', 'github-dark'],
        useStarlightDarkModeSwitch: true,
      },
    }),
  ],
});
