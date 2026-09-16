---
type: UI Route
title: SvelteKit Routes
description: SvelteKit dashboard plus an embeddable consumer portal for webhook, event, delivery, and health management
tags: [sveltekit, ui, routes]
timestamp: 2026-08-29T00:00:00Z
---

# SvelteKit Routes

Built with SvelteKit 2, Svelte 5, Tailwind CSS v4, adapter-static (SPA mode).

The dashboard covers the full webhook lifecycle plus an embeddable, token-scoped consumer portal. Route groups:

- Marketing landing page (`/`)
- Webhooks — list, register, detail
- Events — type list, register, push, edit, delivery reports, instance detail
- Deliveries — list and detail
- Health dashboard (`/dashboard/health`)
- Consumer portal (`/portal`) — token-scoped, embeddable self-service for a single consumer

Route files live under `web/src/routes/`; browse there for the current, authoritative set.

## Stack

| Layer | Tech |
|-------|------|
| Framework | SvelteKit 2 + Svelte 5 (runes) |
| CSS | Tailwind CSS v4 |
| API Client | `openapi-fetch`, typed from `api-types.d.ts` (generated from `api/openapi.yaml`) |
| Font | Fira Code |

## Citations

- `web/src/routes/` — all page files
- `web/src/lib/services.ts` — REST client setup with API key injection
