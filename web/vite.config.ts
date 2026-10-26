import staticAdapter from "@sveltejs/adapter-static";
import { sveltekit } from "@sveltejs/kit/vite";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";
import tailwindcss from "@tailwindcss/vite";
import { defineConfig } from "vite";
import devtoolsJson from "vite-plugin-devtools-json";

export default defineConfig({
  plugins: [
    tailwindcss(),
    sveltekit({
      preprocess: vitePreprocess(),
      adapter: staticAdapter({
        pages: "../internal/ui/dist",
        assets: "../internal/ui/dist",
        fallback: "index.html",
        strict: false,
      }),
    }),
    devtoolsJson(),
  ],
  server: {
    allowedHosts: true,
    fs: {
      // recipes are bundled from ../satellites/recipes/*.yaml
      allow: [".."],
    },
  },
});
