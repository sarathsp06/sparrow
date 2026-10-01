---
layout: ../../layouts/BlogArticleLayout.astro
title: Write it down once
kicker: On recipes
author: Sparrow team
description: How Sparrow recipes turn webhooks into integrations for Slack, PagerDuty, ClickHouse and more. The first integration is a favour; by the third, you want to write it down.
pubDate: 2026-10-01
tags: [recipes, integrations]
---

Most families have a recipe that lives on an index card. The handwriting is fading and there's a stain in one corner. It says "a cup of flour," not "120 grams." It leaves out the things the writer took for granted and spells out the one step they learned the hard way.

What makes that card valuable is not the dish. It is that someone decided a thing they knew how to do was worth writing down, so that somebody else could do it without them in the room.

Integrations go through the same change. The first one is a favour you do by hand. By the third, you wish you had written the first one down.

## The first request

It was a small ask. The operations team wanted every `order.created` event sent to their internal service, as a tidy JSON message, with a bearer token that must never appear in plain text anywhere.

You could set that up by hand: register a webhook, add the headers, attach a transform to a subscription. Or you could write it down once, as a recipe:

```yaml
version: 1
name: order-notifier
description: Send order events to the operations service
params:
  - name: api_url
    prompt: "Operations API URL"
    required: true
  - name: api_token
    prompt: "Operations API token"
    required: true
    secret: true
webhook:
  url: '{{param "api_url"}}'
  headers:
    Content-Type: application/json
    X-Sparrow-Source: webhook-events
  secret_headers:
    Authorization: 'Bearer {{param "api_token"}}'
subscription:
  transform_template: |
    {
      "type": {{.event_name | json}},
      "event_id": {{.event_id | json}},
      "order": {{.payload | json}}
    }
```

Read it top to bottom and it tells you the whole story. The `params` are the questions someone answers when they use it. The `webhook` block says where the request goes and what it carries. The token sits under `secret_headers`, so Sparrow envelope-encrypts it at rest. The `transform_template` shapes the message on every delivery.

Notice what the card leaves blank: the actual URL and the actual token. Like a good recipe, it writes down the method and leaves room for your own ingredients. The credentials only arrive when someone applies it.

## Then everyone else asked

A week later, support wants `order.created` and `order.refunded` in a Slack channel. Then on-call wants a PagerDuty incident when a payment fails. Then analytics wants raw events in ClickHouse.

Each destination expects its own payload shape and its own way of proving who you are. Writing a little glue service for each one would leave you owning a drawer full of code nobody remembers writing. This is where a recipe stops being a convenience and becomes the boundary: one written-down adapter per destination, used wherever the event needs to go.

Sparrow ships recipes for Slack, Discord, ntfy, PagerDuty, SendGrid, ClickHouse, and Twilio. For the support channel, you only supply what belongs to your kitchen:

```sh
sparrow use slack \
  --param webhook_url=https://hooks.slack.com/services/T000/B000/XXX \
  --event order.created \
  --event order.refunded \
  --label env=prod
```

The recipe brings the headers and the transform. `--event` picks which event types to deliver, and `--label` narrows the subscription to events with matching labels. Those choices stay outside the recipe, so the same Slack adapter works in development, staging, and production.

## Tasting before serving

Nobody serves a new dish without tasting it first. Before you point anything at a live channel, you want to see exactly what Slack will receive. The [Recipe Workbench](/sparrow/tools/recipe-workbench/) is the tasting spoon:

1. Pick Slack, PagerDuty, ClickHouse, or another recipe from the tabs at the top.
2. Choose an event preset, edit the sample payload, and fill in the parameters. The preview updates as you type.
3. Check **What the destination receives** to see the exact rendered body for that sample event.
4. Look at the generated **Go transform template**, adjust it if you need to, and copy either the template or the matching CLI command.

The workbench never creates a webhook or sends a delivery. It is a safe place to experiment and get things wrong.

When you are happy, register it on your running Sparrow: open the dashboard's webhook registration page, choose **Start from a recipe**, and answer the prompts. Sparrow pre-fills the URL, headers, and transform. You choose the event types and labels before saving.

## Same care, every time

Applying a recipe creates configuration, not a new worker. Sparrow stores an ordinary webhook and subscription. On each delivery, the webhook worker renders the template against the event:

```text
.event_id   .event_name   .timestamp   .attempt   .payload
```

The rendered body is then signed, delivered, retried, and recorded like any other Sparrow delivery. A recipe gets no shortcut around the history. It gets the same care as everything else.

That makes retries predictable. A retry renders the same template again with `.attempt` updated. The PagerDuty recipe uses the stable `.event_id` as its `dedup_key`, so a network retry at 3 a.m. does not wake someone up twice for the same failed payment. Small detail, big kindness.

## When the recipe goes wrong

Sooner or later someone edits a template and refers to a field that is not there. Sparrow will not quietly send a broken Slack message. If rendering fails, it records a `template_error` and sends nothing. A subscription can choose to send the plain event instead with `on_transform_error: fallback`, and even then the error is still recorded.

You can catch most of these before they ship:

```sh
sparrow template test
```

It renders the template locally with the same engine and a synthetic event, without creating a webhook or calling the API. For optional fields, use `dig` or `index`, and pass interpolated values through `json` so quotes and newlines cannot break the output.

## The card in the drawer

That first YAML file was already a complete recipe. Save it as `satellites/recipes/order-notifier.yaml`, or build it in the [Recipe Workbench](/sparrow/tools/recipe-workbench/) and taste the result before applying it.

The operations service got its tidy JSON. Support got Slack. On-call got PagerDuty without double pages. Each integration is a few lines anyone on the team can read, and Sparrow keeps doing the hard, unglamorous parts: storage, signing, retries, and history.

Writing something down is a small act of generosity toward whoever comes next. Most of the time, that person is you, six months from now, trying to remember how you did it.
