import { defineEnvVars } from "@sveltejs/kit/env";

export const variables = defineEnvVars({
  PUBLIC_API_URL: {
    public: true,
    static: true,
    // Optional: runtime-config.ts falls back to /config.js and same origin.
    schema: (value) => value,
    description: "API base URL baked into the bundle at build time. See src/lib/runtime-config.ts.",
  },
  PUBLIC_APP_VERSION: {
    public: true,
    static: true,
    schema: (value) => value,
    description: "UI version baked into the bundle at build time (the Makefile and Dockerfile pass the release semver). Empty in dev.",
  },
});
