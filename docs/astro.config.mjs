import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import config from './src/data/proto2astro-config.json';

// This file is scaffold-only: proto2astro writes it once and never overwrites it.
// All proto-derived and YAML-derived values are read from proto2astro-config.json
// (which IS regenerated on every `proto2astro generate` run).
//
// Feel free to customize this file — add integrations, Vite config, i18n, etc.

export default defineConfig({
  ...(config.site && { site: config.site }),
  ...(config.base && { base: config.base }),
  integrations: [
    starlight({
      title: config.title,
      description: config.description,
      ...(config.logo && { logo: { src: config.logo } }),
      ...(config.social?.length && { social: config.social }),
      ...(config.editLink && { editLink: { baseUrl: config.editLink } }),
      ...(Object.keys(config.components ?? {}).length && { components: config.components }),
      sidebar: config.sidebar,
      customCss: ['./src/styles/custom.css', ...(config.customCss ?? [])],
    }),
  ],
});
