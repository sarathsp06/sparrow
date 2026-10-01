---
layout: ../../layouts/BlogArticleLayout.astro
title: Describe the payload, get the template
kicker: On AI-assisted transforms
author: Sparrow team
tags: [templates, ai, subscriptions]
description: Draft a subscription's transform template from a plain-language description, verified by rendering and repaired until it works, with Anthropic, any OpenAI-compatible server, or a copy-and-paste prompt.
pubDate: 2026-10-01
---
Sparrow delivers webhooks. Between the event your system produces and the body a receiver wants to see sits a transform template: a Go `text/template` that runs once per delivery and shapes the payload into whatever Slack, PagerDuty, or your internal order service expects. Templates are the most useful knob in Sparrow and, until now, the most annoying one to turn.

This post is about what we built to fix that, why it took the shape it did, and how it works under the hood.

## The problem

Writing a transform template means holding three things in your head at once:

1. **The event's shape.** Which fields exist, what they are called, whether `amount` is a number or a string, whether `customer.email` is always there.
2. **The receiver's shape.** Slack wants Block Kit. PagerDuty wants an Events API v2 envelope. Your own service wants whatever your colleague wrote in a wiki two years ago.
3. **Go template syntax.** Pipes, `printf`, `index`, quoting rules, and the helper functions Sparrow ships for JSON, dates, and strings.

Most people get one of those three wrong on the first try. The classic failure is quiet: you write `{{ .Payload.id }}` instead of `{{ .payload.id }}`, the engine renders `<no value>`, and the receiver gets a body with a hole in it. Nothing errors. You find out when someone asks why the Slack message says "Order  for 42.5".

We had already built a dry-run endpoint and a "Run Preview" button so people could see the rendered body before saving. That helped with syntax errors. It did not help with the blank-page problem: staring at an empty textarea knowing roughly what you want and not knowing how to say it in template syntax.

## The solution, in one sentence

Describe the body the receiver should get in plain language, and Sparrow writes the template, renders it against your event's sample payload, fixes it if it does not render, and hands you something that works.

The feature lives inside the subscription editor as a panel called **Draft with AI**. You pick the event type, you see its sample payload in an editable box, you type something like:

> a short Slack message with the order id and total, prefixed with a warning when status is refunded

and press Draft. A few seconds later the template is in the editor and the rendered output is in the preview.

## The thought process

### Sparrow already had everything a model needs

The first realisation was that the grounding was already there. Every event type in Sparrow can carry a JSON Schema and a sample payload. The helper functions are published by the server at `GET /v1/template-functions`, each with its documentation. The dry-run endpoint renders any template against any event's sample. A model given the schema, the sample, the helper catalog, and a description has everything a careful human would use.

That shaped the design more than anything else. We did not want a chat box. We wanted a function: inputs in, verified template out.

### The model is advisory; the renderer is the authority

A language model will confidently invent a helper function that does not exist, or reach for a field that is not in the payload. We did not want to trust it. So the loop is:

1. Build the prompt from the schema, sample, helper catalog, and the user's instructions.
2. Ask the model for a structured JSON reply: `{ template, notes }`.
3. Render the template against the sample payload, through the exact same code path a real delivery uses.
4. If rendering fails, send the error message back to the model and ask it to fix the template. Up to three rounds.
5. Only a template that rendered reaches the editor.

The model never sees a stored event, a header, or a secret. It sees the registered sample payload, or one the user edited.

### The silent-failure bug, and the fix that made the loop honest

During testing the loop let a bad draft through. The model wrote `.Payload.id`, the engine rendered `%!s(<nil>)` into the output, and because the render did not *error*, the loop called it a success.

The first fix was a scan of the rendered output for missing-value markers. It worked, until we noticed the `json` helper escapes `<` as `<`, which hid the marker a second time.

The real fix came from a review comment: a missing key should fail. Go's `text/template` has a `missingkey=error` option, so we added a strict engine that uses it. Draft verification renders with the strict engine, which means the model gets back a precise error such as `map has no entry for key "Payload"` instead of a scan result. Deliveries stay lenient, because a payload missing an optional field should still produce a body rather than fail. The preview endpoint gained an optional `strict` flag so you can get the same rigour on your own hand-written templates.

One consequence worth knowing: under strict mode, `{{ if .payload.optional }}` is itself an error when the key is absent. The prompt tells the model to read optional fields with `(index .payload "optional")`, and the docs say the same to humans.

### The sample payload is the grounding, so show it and let people edit it

Schemas say "string". They do not say "this string is an email address" or "this one is an ISO date". A model drafting from the schema alone will treat `customer.email` as any old text. So the panel opens with the registered sample payload pre-filled in an editable box, with a hint to replace placeholder values with real-looking ones. Whatever is in that box is what the draft is written and verified against. The event type itself is never changed.

### Recipes for known destinations, examples and docs for everything else

Sparrow ships recipes for Slack, Discord, PagerDuty, Twilio, SendGrid, ClickHouse, and ntfy. Each recipe carries a reference template with the destination's required shape. Pick one from the **Destination format** dropdown and the model is told to keep that shape and adapt only the content.

For an internal receiver with no recipe, the panel takes either a pasted example of the body it accepts, a plain description of the shape, or a documentation URL. The URL is fetched by the server, stripped to visible text, capped in size, and handed to the model as an excerpt. That fetch runs under the same network policy as webhook deliveries, so it cannot be pointed at loopback, private ranges, or cloud metadata unless the operator allowed them.

### Refine instead of replace

A draft is rarely the final word. The panel has a refine mode: with it on, the current template is sent along and the model is asked to change only what the instructions say. Refine is on by default whenever a template already exists. When it is off and the editor has content, pressing Draft asks first, and after any draft an **Undo draft** link restores what was there before.

### No AI configured? You still get the prompt

Many self-hosted installs will not configure a model. We did not want the feature to vanish for them. Since the server already assembles the full prompt, there is an endpoint that returns it as one pasteable text, and the same panel offers **Copy prompt for AI** instead of Draft. Paste it into any chat assistant, paste the template it returns into the editor, and check it with Run Preview. The prompt asks for the template in a code block rather than a JSON object, because that is what humans paste.

### Two providers, and that is enough

The drafter talks to a model through a very small interface. There are two implementations: the Anthropic API through the official SDK, and any OpenAI-compatible `/v1/chat/completions` server through a plain HTTP client. The second one is what makes local models work: Ollama, vLLM, LM Studio, and llama.cpp all speak that protocol, so a self-hoster can point Sparrow at a model on the same VPN and nothing leaves the network. A review suggested a multi-provider library; we decided against it. For one call, two small clients we control beat a dependency we would have to track, and the structured JSON output the repair loop relies on is easier to guarantee by hand.

The default Anthropic model is a light one. Drafts are short, and the render-and-repair loop, not model size, is what guarantees a working template. A small model with up to three repair rounds handles most templates. You can raise it with one environment variable.

### Light model, strict renderer

That combination is the design in miniature. Spend as little as possible on the model, because the expensive part of correctness is handled by something deterministic that already exists.

## How it works

### Configuration

```
SPARROW_AI_PROVIDER=anthropic   # or openai
SPARROW_AI_API_KEY=...          # Anthropic key; optional Bearer for openai
SPARROW_AI_MODEL=...            # default claude-haiku-4-5 for anthropic; required for openai
SPARROW_AI_BASE_URL=...         # required for openai, e.g. http://localhost:11434/v1
```

Set none of these and the editor offers the copy-prompt mode. `GET /v1/capabilities` tells the UI which mode the server is in.

### Endpoints

- `POST /v1/subscriptions:draftTemplate` takes an event name, instructions, and optionally a recipe name, a receiver example, a docs URL, a sample payload override, and the current template to refine. It returns the template, its rendered output, the model's notes, and how many repair rounds it took.
- `POST /v1/subscriptions:draftTemplatePrompt` takes the same body and returns the prompt as text, with no model involved.
- `POST /v1/subscriptions:testTemplate` is the existing dry-run, now with an optional `strict` flag.
- `GET /v1/consumers/{consumer}/subscriptions/{id}/templateVersions` lists the templates a subscription has been saved with, newest first, with how each was produced.

### The repair loop, concretely

A real exchange from testing, with a stub model that deliberately gets the first attempt wrong:

1. Model returns `{"text": {{ printf "Order %s" .Payload.id | json }}}`.
2. Strict render fails: `template: webhook:1:45: executing "webhook" at <.Payload.id>: map has no entry for key "Payload"`.
3. That message goes back as the next user turn, with the instruction to fix the template using only fields in the payload.
4. Model returns `{"text": {{ printf "Order %s for %v" .payload.id .payload.amount | json }}}`.
5. Render succeeds. The UI shows the template, the output, and "(repaired after 1 failed render)".

### What the model is told

The system prompt is stable across requests, so providers that cache prefixes benefit. It explains the data model (`.event_id`, `.event_name`, `.timestamp`, `.attempt`, `.payload`), the rules (use only catalog helpers, pipe values through `json`, read optional fields with `index`, keep a recipe's top-level shape), and then the full helper catalog with each function's documentation. The user turn carries the event name, schema, sample, any recipe reference, receiver example, docs excerpt, current template, and the instructions.

### Template history

Every save that changes a subscription's template records a version: the template text, whether it was manual or an AI draft, the drafter's notes, who saved it, and when. The last 20 are kept per subscription. There is no restore endpoint on purpose: you load a version into the editor and save, which runs the normal path and records a new version.

## What we would do next

- Infer a schema and sample payload for event types registered without one, from recent events. That would unlock drafting for existing installs.
- Repair from a failed delivery: feed a receiver's 4xx response body and the current template into refine mode from the delivery page.
- Explain a template in prose before saving, as a cheap review step for people who do not read Go templates.

## Try it

Set `SPARROW_AI_API_KEY`, or point `SPARROW_AI_PROVIDER=openai` at a local Ollama, open a subscription, enable payload transformation, and press **Draft with AI**. Or set nothing and press **Copy prompt for AI**.
