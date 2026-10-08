---
type: UI Component
title: Frontend Components
description: Reusable Svelte components for the web UI — tables, badges, dialogs, batch progress, access/auth
tags: [svelte, components, ui, access, auth]
timestamp: 2026-10-08T00:00:00Z
---

# Frontend Components

## Shared Components

All under `web/src/lib/components/`.

| Component | Purpose |
|-----------|---------|
| `Pagination.svelte` | Pagination controls |
| `BatchProgress.svelte` | Batch job progress indicator |
| `EventReportsTable.svelte` | Filterable event reports |
| `SubscriptionManager.svelte` | Subscription CRUD manager |
| `StatusBadge.svelte` | Delivery status badge |
| `HealthBadge.svelte` | Health status badge |
| `EmptyState.svelte` | Empty state placeholder |
| `CopyableId.svelte` | Click-to-copy UUID |
| `ConfirmDialog.svelte` | Confirmation modal |
| `FloatingAction.svelte` | Floating action button |
| `ConsumerPicker.svelte` | Consumer input with server-side search (`GET /v1/consumers?q=`); list filters and forms |
| `SigningSecretReveal.svelte` | Shows a webhook's signing secret or Ed25519 public key once, after registration or rotation |
| `AlertConfigs.svelte` | Lists, adds and removes a webhook's alert email configs |

## Access Components

Under `web/src/lib/access/`:

| File | Purpose |
|------|---------|
| `auth.svelte.ts` | Reactive auth state; stores credential in `localStorage`; master-key-to-token exchange on sign-in |
| `SignInPrompt.svelte` | Sign-in dialog (replaces former `ApiKeyPrompt.svelte`); accepts master key or token; shows reason on revoked/expired/wrong key |
| `AccountBadge.svelte` | Sidebar badge: "Signed in as &lt;name&gt;" / "Authentication off", with sign-out button |
| `client.ts` | Access API client (tokens CRUD, invites CRUD, whoami, redeem) |
| `client.test.mjs` | Tests for access client |
| `portal-invite.svelte.ts` | Portal invite redemption (`/portal#invite=...` fragment) |

Under `web/src/routes/access/`:

| File | Purpose |
|------|---------|
| `+page.svelte` | Access page in nav: token list with Revoke, pending invites with Cancel, Invite/Create token dialogs |

Invite dialog: name, full access or one consumer's portal, link expiry (1h/24h/7d), access expiry. Create token: shows secret once. Sign-in prompt: on 401, exchanges a pasted master key for a named browser token so the key is never stored. `/#invite=<secret>` redeems an invite, stores the token, reloads without the fragment. Sign out: revokes the browser's own token and forgets it. `/portal#invite=<secret>` redeems a consumer invite and opens that consumer's portal.

## Service Layer

`web/src/lib/services.ts` — creates a single typed REST client (`openapi-fetch`) against `/v1/*`, typed from the generated `api-types.d.ts`. Base URL and key come from `web/src/lib/runtime-config.ts` + `web/src/lib/access/auth.svelte.ts`: `window.__SPARROW_CONFIG__` (`apiUrl`/`apiKey`, set in the static `/config.js` for a separately hosted UI) > `PUBLIC_API_URL` (build time) > same origin (`http://localhost:8080` under `vite dev`). On a `401` the layout shows `SignInPrompt.svelte`; the credential (master key or access token) is stored in `localStorage`. A pasted master key is exchanged for a browser token. Portal pages use a bearer token and rewrite `/v1/...` to `<apiBase>/portal/api/...`.

## Consumer scoping

`web/src/lib/consumer.svelte.ts` — `consumerFilter` reads and writes the page's `?consumer=` URL param (empty = all consumers); `withConsumer()` builds links that keep it. There is no global switcher.

## Citations

- `web/src/lib/`
