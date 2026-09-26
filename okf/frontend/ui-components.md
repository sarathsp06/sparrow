---
type: UI Component
title: Frontend Components
description: Reusable Svelte components for the web UI — tables, badges, dialogs, batch progress
tags: [svelte, components, ui]
timestamp: 2026-08-29T00:00:00Z
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

## Service Layer

`web/src/lib/services.ts` — creates a single typed REST client (`openapi-fetch`) against `/v1/*`, typed from the generated `api-types.d.ts`. Base URL and key come from `web/src/lib/runtime-config.ts` + `auth.svelte.ts`: `window.__SPARROW_CONFIG__` (`apiUrl`/`apiKey`, injected inline by the Go server for the embedded UI, or set in the static `/config.js` for a separately hosted UI) > `PUBLIC_API_URL` (build time) > same origin (`http://localhost:8080` under `vite dev`). On a `401` the layout shows `ApiKeyPrompt.svelte`; the typed key is stored in `localStorage` (`sparrow_api_key`) and wins over the injected one. Portal pages use a bearer token and rewrite `/v1/...` to `<apiBase>/portal/api/...`.

## Citations

- `web/src/lib/`
