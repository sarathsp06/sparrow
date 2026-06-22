---
type: UI Component
title: Frontend Components
description: Reusable Svelte components for the web UI — tables, badges, dialogs, batch progress
tags: [svelte, components, ui]
timestamp: 2026-06-22T00:00:00Z
---

# Frontend Components

## Shared Components

All under `web/src/lib/components/`.

| Component | Purpose |
|-----------|---------|
| `table.svelte` | Generic data table |
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

`web/src/lib/services.ts` — creates 5 Connect-RPC clients (Webhook, Event, Subscription, Delivery, Health). Reads `window.__SPARROW_CONFIG__` for dynamic API key injection (injected by the Go server at runtime).

## Citations

- `web/src/lib/`
