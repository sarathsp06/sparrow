import { AstroError } from 'astro/errors';
import GithubSlugger from 'github-slugger';

export default function starlightLlmsTxt(opts = {}) {
  return {
    name: 'custom-starlight-llms-txt',
    hooks: {
      setup({ astroConfig, addIntegration, config }) {
        if (!astroConfig.site) {
          throw new AstroError(
            '`site` not set in Astro configuration',
            'The llms plugin requires setting `site` in your Astro configuration file.'
          );
        }
        addIntegration({
          name: 'custom-starlight-llms-txt',
          hooks: {
            'astro:config:setup'({ injectRoute, updateConfig }) {
              injectRoute({
                entrypoint: new URL('./llms.txt.ts', import.meta.url),
                pattern: '/llms.txt',
                prerender: true,
              });
              injectRoute({
                entrypoint: new URL('./llms-full.txt.ts', import.meta.url),
                pattern: '/llms-full.txt',
                prerender: true,
              });
              injectRoute({
                entrypoint: new URL('./llms-small.txt.ts', import.meta.url),
                pattern: '/llms-small.txt',
                prerender: true,
              });
              injectRoute({
                entrypoint: new URL('./llms-custom.txt.ts', import.meta.url),
                pattern: '/_llms-txt/[slug].txt',
                prerender: true,
              });

              const slugger = new GithubSlugger();
              const projectContext = {
                base: astroConfig.base,
                title: opts.projectName ?? config.title,
                description: opts.description ?? config.description,
                details: opts.details,
                optionalLinks: opts.optionalLinks ?? [],
                customSets: (opts.customSets ?? []).map((set) => ({
                  ...set,
                  slug: slugger.slug(set.label),
                })),
                minify: opts.minify ?? {},
                promote: opts.promote ?? ['index*'],
                demote: opts.demote ?? [],
                exclude: opts.exclude ?? [],
                defaultLocale: config.defaultLocale,
                locales: config.locales,
                pageSeparator: opts.pageSeparator ?? '\n\n',
                rawContent: opts.rawContent ?? false,
              };

              const modules = {
                'virtual:starlight-llms-txt/context': `export const starlightLllmsTxtContext = ${JSON.stringify(projectContext)}`,
              };
              const resolutionMap = Object.fromEntries(
                Object.keys(modules).map((key) => [resolveVirtualModuleId(key), key])
              );

              updateConfig({
                vite: {
                  plugins: [
                    {
                      name: 'vite-plugin-custom-starlight-llms-text',
                      resolveId(id) {
                        if (id in modules) return resolveVirtualModuleId(id);
                      },
                      load(id) {
                        const resolution = resolutionMap[id];
                        if (resolution) return modules[resolution];
                      },
                    },
                  ],
                },
              });
            },
          },
        });
      },
    },
  };
}

function resolveVirtualModuleId(id) {
  return `\0${id}`;
}
