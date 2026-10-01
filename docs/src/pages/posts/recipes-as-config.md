---
layout: ../../layouts/BlogArticleLayout.astro
title: Recipes turn webhooks into integrations
author: Sparrow team
description: Use a small YAML recipe to deliver Sparrow events to Slack, PagerDuty, ClickHouse, and more.
pubDate: 2026-10-01
tags: [recipes, integrations]
---

Start with the delivery you need. Suppose an `order.created` event should become a small JSON message for an internal service, with a custom header and a token that must never appear in plaintext:

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

The `transform_template` is rendered by Sparrow for every delivery. Custom headers travel with the webhook, while `secret_headers` are envelope-encrypted at rest. Parameters are supplied when the integration is applied, so the recipe contains the shape of the integration without containing the credential itself.

That first file solves one destination. Soon the same event needs to reach Slack, PagerDuty, or ClickHouse, each with a different payload shape and authentication scheme. Copying webhook setup into a new service would multiply the glue code. The recipe is the reusable boundary: one declarative adapter, applied wherever the event needs to go.

## Apply an integration without writing glue code

The built-in recipes cover Slack, Discord, ntfy, PagerDuty, SendGrid, ClickHouse, and Twilio. Choose one with the CLI and provide only the values that are specific to your environment:

```sh
sparrow use slack \
  --param webhook_url=https://hooks.slack.com/services/T000/B000/XXX \
  --event order.created \
  --event order.refunded \
  --label env=prod
```

The recipe supplies the URL, headers, and transform template. `--event` selects the event types that should be delivered, while `--label` narrows the subscription to matching event labels. Those choices stay outside the recipe, so the same adapter can be reused for development, staging, and production.

You can also start from a recipe in Sparrow's dashboard. The webhook registration flow prompts for its parameters and pre-fills the destination, headers, and transformation settings.

## Use recipes from the UI

There are two useful UI paths:

1. Open the [Recipe Workbench](/sparrow/tools/recipe-workbench/). Select Slack, Discord, PagerDuty, ClickHouse, or another recipe from the tabs at the top.
2. Choose an event preset, edit the sample payload, and fill in the destination parameters. The workbench updates the destination preview as you type.
3. Review the rendered request body under **What the destination receives**. This is the payload the recipe produces for the sample event.
4. Inspect the generated **Go transform template**, edit it if needed, and copy the template or the generated CLI command.
5. To register it on a running Sparrow instance, open the dashboard's webhook registration page, choose **Start from a recipe**, and provide the prompted values. Sparrow pre-fills the webhook URL, headers, and subscription transform; choose the event types and labels for that subscription before saving.

The workbench is a safe preview surface: it does not create a webhook or send a delivery. It lets you compare the formatted output with the template that Sparrow will store before applying the recipe.

## Rendering happens at delivery time

Applying a recipe is configuration, not a separate worker. Sparrow stores the resulting webhook and subscription through its API. On every delivery, the webhook worker renders the template against the event context:

```text
.event_id   .event_name   .timestamp   .attempt   .payload
```

The rendered body is then signed, delivered, retried, and audited like any other Sparrow webhook. A failed attempt does not bypass the normal delivery history just because a recipe transformed the payload.

This also makes retries predictable. A retry renders the same template again, with `.attempt` updated. PagerDuty's recipe uses the stable `.event_id` as its `dedup_key`, so a network retry does not create a second incident.

## Templates fail visibly

A missing field or invalid template should not quietly produce a malformed notification. If rendering fails, Sparrow records a `template_error` and sends nothing. Subscriptions can explicitly opt into the plain envelope with `on_transform_error: fallback`, but the original transform error remains recorded.

Before applying a recipe, render it locally with:

```sh
sparrow template test
```

The command uses the same template engine and a synthetic event context, without creating a webhook or making an API call. For optional payload fields, use helpers such as `dig` or `index`; quote interpolated values with `json` so embedded quotes and newlines cannot corrupt the output.

## From one transform to a reusable recipe

The opening YAML is the complete shape of a custom recipe: parameters are collected at apply time, ordinary headers are registered with the destination, secret headers are encrypted, and the Go template is rendered per delivery. Save it as `satellites/recipes/order-notifier.yaml`, or use the [Recipe Workbench](/sparrow/tools/recipe-workbench/) to compose and preview the transformation before applying it.

That is the transition from a one-off transform to a recipe: the integration stays declarative, while Sparrow continues to own persistence, signing, retries, and delivery history.
