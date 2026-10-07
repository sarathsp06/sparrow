---
title: Introducing Sparrow
kicker: Hello, world
author: Sarath Sadasivan Pillai
description: Sparrow is a self-hosted webhook delivery server. One Go binary, one PostgreSQL database, MIT licensed. Here is what it does and why it is built the way it is.
pubDate: 2026-10-07
tags: [announcement, self-hosting, open-source]
---

Sparrow is a webhook delivery server you run yourself. You post an event to it, and it takes care of the rest: fan-out to every subscriber, signing, retries with backoff, per-endpoint health, and a full record of every attempt. It is open source under the MIT license, and it has been quietly shipping for a while. This post is the proper introduction.

## The whole thing fits in your head

Most webhook infrastructure arrives as a set of boxes: an API, a queue, a broker, a cache, a fleet of workers, and a dashboard that talks to all of them. Each box is reasonable on its own. Together they are a system you have to learn before you can trust it.

Sparrow is two boxes.

- **One Go binary.** The API, the delivery workers, and the dashboard are compiled into a single executable. The Svelte UI is embedded in it. There is nothing else to deploy.
- **One PostgreSQL database.** Events, subscriptions, deliveries, attempts, health state, and the job queue itself ([River](https://riverqueue.com)) all live there. No Redis, no message broker, no object storage.

That is the entire operational surface. One thing to back up. One thing to monitor. One place to look at 2 a.m. when a partner asks what they missed.

```text
your services ──▶ Sparrow ──▶ PostgreSQL ──▶ signed deliveries to subscribers
```

The simplicity is not a stage I plan to grow out of. It is the design. A tool you can understand completely is a tool you can run with confidence, fix when it breaks, and fork if you ever need to.

## Simple to run does not mean simple-minded

Keeping the footprint small forced me to be careful about what goes in. What is in is the part that is hard to get right on your own:

- **At-least-once delivery** with retries, backoff, per-webhook rate limits, and 429-aware snoozing.
- **Signatures** in [Standard Webhooks](https://www.standardwebhooks.com/) format, HMAC-SHA256 and optional Ed25519, so receivers can verify with any existing library.
- **Health per endpoint.** Healthy, degraded, or unhealthy, with automatic pausing of receivers that keep failing. Paused deliveries are held, never dropped.
- **Full history.** Every request and response is recorded. Re-push and retry operate on a deterministic snapshot, so a bulk action does exactly what you saw when you clicked it.
- **Payload shaping.** Per-subscription Go templates turn your event into the shape the receiver wants. Shipped recipes do this for Slack, Discord, PagerDuty, ntfy, ClickHouse, Twilio, SendGrid, and CloudEvents with no glue service in between.
- **Encryption at rest** for secrets and sensitive headers, and SSRF protection on every outbound request.
- **A consumer portal** your customers can use to register endpoints and inspect their own deliveries, scoped by token.

And when a step cannot do what you asked, Sparrow fails visibly. A transform that errors fails that delivery with a permanent, recorded error. It never silently sends something else.

## Your data stays home

Webhook payloads carry customer names, amounts, and addresses. Routing them through a third-party relay means one more company holds that data, one more vendor review, one more place a breach can begin.

Sparrow runs inside the network that already owns the data. It is built for teams delivering webhooks to internal systems and partners from behind their own VPN. It is deliberately not a multi-tenant SaaS, and that narrow focus is a big part of why it can stay small.

## MIT, and meant that way

Sparrow is licensed under MIT. Not source-available, not open-core, not a license that changes when you get successful. You can run it, modify it, embed it, and ship it inside your own product. If the project stopped tomorrow, you would still have everything you need to keep going.

I chose MIT because infrastructure you depend on should not come with a catch. The whole point of self-hosting is that nobody can take the thing away from you. A restrictive license would undo that.

## Try it in five minutes

Run it locally with Docker Compose:

```bash
curl -O https://raw.githubusercontent.com/sarathsp06/sparrow/main/deploy/docker-compose.yml
docker compose up -d
```

Install the CLI and send your first event:

```bash
curl -fsSL https://raw.githubusercontent.com/sarathsp06/sparrow/main/scripts/install.sh | sh
sparrow init --url http://localhost:8080
sparrow listen --event order.created
```

In another terminal:

```bash
sparrow push order.created -d '{"order_id":"ord_1","amount":42}'
```

The delivery arrives in the first terminal with its signature verified. Routing that same event to Slack is one more command:

```bash
sparrow use slack --param webhook_url=https://hooks.slack.com/services/T00/B00/xxx --event order.created
```

The dashboard is at <http://localhost:8080>, the interactive API reference at <http://localhost:8080/docs>, and the full documentation at [sarathsp06.github.io/sparrow](https://sarathsp06.github.io/sparrow/). The code is on [GitHub](https://github.com/sarathsp06/sparrow). Issues, questions, and pull requests are welcome.

Sparrow is small on purpose. I hope that is exactly what you were looking for.
