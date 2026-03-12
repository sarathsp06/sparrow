/**
 * Auth provider contract.
 *
 * Each auth provider (Clerk, Auth0, custom, etc.) must expose:
 *  - A wrapper Svelte component that handles sign-in/sign-out UI
 *  - A way to inject session tokens into API requests via registerTokenProvider()
 *
 * The provider is selected at build time by the PUBLIC_AUTH_PROVIDER env var.
 * When no provider is configured, the app runs without authentication.
 */

/** Supported auth provider identifiers. */
export type AuthProviderType = "clerk" | "none";

/** Resolved auth provider configuration. */
export interface AuthProviderConfig {
  type: AuthProviderType;
  /** Provider-specific options (e.g., publishable key). */
  options: Record<string, string>;
}
