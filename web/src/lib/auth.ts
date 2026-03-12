/**
 * Auth token provider — a thin abstraction between identity providers
 * and the rest of the app.
 *
 * Any auth provider (Clerk, Auth0, Keycloak, custom) registers a token
 * getter via registerTokenProvider(). The services layer retrieves tokens
 * via getSessionToken() without knowing which provider is active.
 *
 * This module has zero provider-specific imports — it only deals with
 * the "get me a JWT string" contract.
 */

// Token getter — set by the active auth provider's bridge component.
// We store a getter function so the services module can retrieve the latest
// session token without importing any provider SDK directly.
let _getToken: (() => Promise<string | null>) | null = null;

/**
 * Register a session token getter. Called once by the active auth provider's
 * bridge component after it initializes (e.g., ClerkAuthBridge, Auth0Bridge).
 */
export function registerTokenProvider(getter: () => Promise<string | null>) {
  _getToken = getter;
}

/**
 * Get the current session token (JWT) for API requests.
 * Returns null if no auth provider is active or the user is not signed in.
 */
export async function getSessionToken(): Promise<string | null> {
  if (!_getToken) return null;
  try {
    return await _getToken();
  } catch {
    return null;
  }
}
