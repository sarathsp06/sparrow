import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: 'https://sarathsp06.github.io',
  base: '/sparrow',
  integrations: [
    starlight({
      title: 'Sparrow',
      description: 'Self-hosted webhook delivery platform',
      components: {
        Footer: './src/components/Footer.astro',
      },
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/sarathsp06/sparrow',
        },
      ],
      editLink: {
        baseUrl: 'https://github.com/sarathsp06/sparrow/edit/main/docs/',
      },
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { slug: 'getting-started/installation' },
            { slug: 'getting-started/quickstart' },
            { slug: 'getting-started/configuration' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { slug: 'guides/webhooks' },
            { slug: 'guides/events' },
            { slug: 'guides/subscriptions' },
            { slug: 'guides/deliveries' },
            { slug: 'guides/health' },
            { slug: 'guides/namespaces' },
            { slug: 'guides/client-libraries' },
          ],
        },
        {
          label: 'API Reference',
          items: [
            { slug: 'reference/api' },
            { slug: 'reference/api/webhook-service' },
            { slug: 'reference/api/event-service' },
            { slug: 'reference/api/subscription-service' },
            { slug: 'reference/api/delivery-service' },
            { slug: 'reference/api/health-service' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { slug: 'reference/template-functions' },
            { slug: 'reference/error-classification' },
            { slug: 'reference/architecture' },
          ],
        },
        {
          label: 'Deployment',
          items: [
            { slug: 'deployment/docker-compose' },
          ],
        },
      ],
      customCss: ['./src/styles/custom.css'],
    }),
  ],
});
