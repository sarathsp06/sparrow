/**
 * Detect the active auth provider from environment variables.
 *
 * Selection logic:
 *  1. If PUBLIC_AUTH_PROVIDER is explicitly set, use that value.
 *  2. Otherwise, infer from provider-specific keys:
 *     - PUBLIC_CLERK_PUBLISHABLE_KEY set → "clerk"
 *  3. Fall back to "none" (no authentication).
 *
 * This keeps backwards compatibility: existing deployments that only set
 * PUBLIC_CLERK_PUBLISHABLE_KEY will automatically use Clerk without needing
 * to add PUBLIC_AUTH_PROVIDER.
 */

import {
  PUBLIC_AUTH_PROVIDER,
  PUBLIC_CLERK_PUBLISHABLE_KEY,
} from "$env/static/public";
import type { AuthProviderConfig, AuthProviderType } from "./types.js";

function detectProvider(): AuthProviderType {
  // Explicit override
  const explicit = (PUBLIC_AUTH_PROVIDER || "").trim().toLowerCase();
  if (explicit === "clerk") return "clerk";
  if (explicit === "none") return "none";

  // Infer from provider-specific keys
  if (PUBLIC_CLERK_PUBLISHABLE_KEY) return "clerk";

  return "none";
}

function buildConfig(): AuthProviderConfig {
  const type = detectProvider();

  switch (type) {
    case "clerk":
      return {
        type: "clerk",
        options: {
          publishableKey: PUBLIC_CLERK_PUBLISHABLE_KEY || "",
        },
      };
    case "none":
    default:
      return { type: "none", options: {} };
  }
}

/** The resolved auth provider config for this build. */
export const authConfig: AuthProviderConfig = buildConfig();
