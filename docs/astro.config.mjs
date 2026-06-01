import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import proto2astroConfig from './src/data/proto2astro-config.json';

export default defineConfig({
  site: proto2astroConfig.site,
  base: proto2astroConfig.base,
  integrations: [
    starlight({
      title: proto2astroConfig.title,
      logo: {
        src: './src/assets/favicon.svg',
      },
      description: proto2astroConfig.description,
      social: proto2astroConfig.social,
      editLink: { baseUrl: proto2astroConfig.editLink },
      sidebar: proto2astroConfig.sidebar,
      components: proto2astroConfig.components,
      customCss: ['./src/styles/custom.css'],
      expressiveCode: {
        themes: ['github-light', 'github-dark'],
        useStarlightDarkModeSwitch: true,
      },
    }),
  ],
});
