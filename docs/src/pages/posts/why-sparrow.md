---
layout: ../../layouts/BlogArticleLayout.astro
title: Why Sparrow is self-hosted first
author: Sparrow team
description: Reliable webhook delivery belongs close to the systems that own the events.
pubDate: 2026-09-30
tags: [architecture, self-hosting]
---

Webhook delivery is infrastructure. When it runs beside the systems producing events, operators keep control of the queue, the retry policy, and the delivery history.

Sparrow keeps that control deliberately small: one Go server, one PostgreSQL database, and no Redis dependency. Events are persisted before delivery begins, so a receiver outage does not erase work or hide what happened.

That makes Sparrow useful for teams that need reliable outbound webhooks without handing event payloads to another service.
