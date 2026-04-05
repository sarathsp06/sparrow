import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

export default defineConfig({
  site: "https://sarathsp06.github.io",
  base: "/sparrow",
  integrations: [
    starlight({
      title: "Sparrow",
      description: "Self-hosted webhook delivery platform",
      social: [
        { icon: "github", label: "GitHub", href: "https://github.com/sarathsp06/sparrow" },
      ],
      editLink: { baseUrl: "https://github.com/sarathsp06/sparrow/edit/main/docs/" },
      sidebar: [
        {
          label: 'Guides',
          items: [
            { slug: 'guides/comment-guide' },
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
            { slug: 'reference/api/enum-webhook-delivery-status' },
            { slug: 'reference/api/enum-webhook-health' },
          ],
        },
      ],
      customCss: ['./src/styles/custom.css'],
    }),
  ],
});
