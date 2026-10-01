---
layout: ../../layouts/BlogArticleLayout.astro
title: Make webhook failures actionable
author: Sparrow team
description: Retry policy starts with knowing which failures can recover.
pubDate: 2026-09-24
tags: [webhooks, reliability]
---

A failed webhook is not a single kind of failure. A timeout, a refused connection, and a permanent client error need different operator actions.

Sparrow classifies delivery failures before deciding whether to retry. Transient network failures and server errors stay on the retry path; permanent responses remain visible without wasting attempts. The delivery record keeps the classification and the final response together.

The result is a queue that is easier to operate: retries are purposeful, and a dashboard can explain what needs attention instead of showing only a red status.
