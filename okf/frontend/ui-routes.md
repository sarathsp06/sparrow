---
type: UI Route
title: SvelteKit Routes
description: 13 SvelteKit pages for webhook management, event operations, delivery tracking, and health monitoring
tags: [sveltekit, ui, routes]
timestamp: 2026-08-29T00:00:00Z
---

# SvelteKit Routes

Built with SvelteKit 2, Svelte 5, Tailwind CSS v4, adapter-static (SPA mode).

| Route | Page |
|-------|------|
| `/` | Marketing landing page (hero, features, getting started, architecture, CTA) |
| `/webhooks` | Webhook list |
| `/webhooks/register` | Register new webhook |
| `/webhooks/[webhookId]` | Webhook detail + deliveries |
| `/events` | Event type list |
| `/events/register` | Register new event type |
| `/events/push` | Push event form |
| `/events/[eventName]/update` | Edit event type |
| `/events/[eventName]/reports` | Event delivery reports |
| `/events/instances/[eventId]` | Event instance detail |
| `/health` | Webhook health dashboard |
| `/deliveries` | Delivery list |
| `/deliveries/[deliveryId]` | Delivery detail |

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
