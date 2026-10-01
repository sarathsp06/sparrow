---
layout: ../../layouts/BlogArticleLayout.astro
title: Why Sparrow Uses Go Templates for Subscription Transformations
author: Sparrow team
description: Subscription-local payload transforms without embedding a JavaScript runtime in the delivery path.
pubDate: 2026-10-01
tags: [Webhooks, Go, performance, security]
---

A webhook event rarely fits every receiver.

Slack wants a small `{ "text": "..." }` object. A billing system may want cents instead of dollars. A partner may need three fields from a record that contains thirty. Sending the original event to every subscriber makes receivers do work they should not have to do.

Sparrow solves that with a `transform_template` on each subscription. The template runs immediately before delivery and prints the body that receiver gets. The same event can therefore become a Slack message for one subscription and a flat integration payload for another.

The obvious implementation is JavaScript. Most teams already have Node somewhere, and JavaScript template engines are familiar. Sparrow deliberately chose Go's `text/template` instead.

## The decision was about the delivery path

A template engine in a webhook product is not a page renderer. It runs inside a queue worker, for every subscription that receives an event. That changes the constraints:

- the process already is Go;
- the input is structured event data, not an arbitrary application object;
- the output is usually JSON or plain text;
- transformations must not make outbound delivery unreliable;
- many deliveries should share one small, predictable runtime.

The best engine is not the one with the most language features. It is the one that does the required transformation while adding the fewest moving parts to the hottest path.

## What “Go templates” means here

Sparrow does not expose Go source code to subscribers. It exposes a constrained template language with a deliberately selected function map:

```go
{{ dict "text" (printf "%s paid %s" .payload.customer.name .payload.currency) | json }}
```

The data context contains the event ID, event name, timestamp, attempt number, and payload. Helpers cover the useful operations: building maps and lists, reading optional fields, formatting values, and serializing the result as JSON.

A template is parsed once, then the parsed representation is reused. Sparrow keeps a bounded LRU cache of parsed templates, so repeated deliveries do not repeatedly pay the parsing cost. The cache is keyed by the template source, with strict and lenient missing-key modes kept separate.

That is a small implementation with a useful property: the runtime is already in the server binary. There is no second interpreter, module loader, package tree, or language bridge in the worker.

## Why not run JavaScript?

JavaScript is not a bad choice. It is often the right choice when templates need a rich ecosystem, browser compatibility, or genuinely programmable application logic. Handlebars, for example, compiles templates and provides helpers, blocks, partials, and HTML escaping. Nunjucks and EJS offer different points on the expressiveness-versus-simplicity spectrum.

But “use JavaScript” can mean two very different designs:

1. use a JavaScript-shaped template language such as Handlebars; or
2. execute user-provided JavaScript in the server.

The second design is much heavier. A server must embed or call a JavaScript runtime, define what globals exist, restrict access to the filesystem and network, control CPU and memory, and keep the runtime's security model aligned with the host application. A language-level sandbox is not automatically a security boundary.

A JavaScript-shaped template engine reduces some of that risk, but it still introduces another runtime and another dependency surface. It also does not make output limits, timeout policy, data minimization, or error handling disappear. Those are properties of the host application, not magic properties of the template syntax.

## Memory footprint: fewer runtimes, fewer surprises

Sparrow is a self-hosted webhook server. Some installations run it beside a database on a small VM, inside a container with a fixed memory limit, or as one of several internal services. Adding a JavaScript runtime solely to reshape JSON would make the baseline process larger before the first template runs.

Go templates let the existing process do the work it is already equipped to do. Parsed templates live in a bounded cache. Execution uses a pooled buffer, and output is capped at 1 MiB. A bad or unexpectedly large transform cannot grow a response without bound.

This is not a claim that every Go template is smaller than every JavaScript template. Real memory usage depends on the runtime, engine, workload, and data. The narrower claim is the useful one: for Sparrow's existing Go process, a native standard-library template engine avoids a second general-purpose runtime and keeps the cache and output budget explicit.

## Speed: parse once, execute many times

The useful optimization is not a language benchmark. It is avoiding repeated work in the delivery loop.

Sparrow parses a template on a cache miss and executes the parsed template for later deliveries. The execution path writes directly into a bounded buffer and returns the transformed bytes. A JavaScript implementation could also precompile and cache templates; that is a valid design. It would still need the runtime boundary and its operational costs.

For a small JSON reshaping operation, the native path gives us predictable overhead and fewer transitions between languages. It also keeps the worker's scheduling model simple: the same Go workers that fetch delivery data and make HTTP requests perform the transform.

The important performance property is therefore repeatability, not a universal “Go is faster” slogan:

- bounded cache size;
- no parse on every delivery after warm-up;
- bounded output;
- no network or filesystem work from a template;
- one execution model across all subscriptions.

## Caching is part of the design

Template caching is easy to describe and easy to get subtly wrong. Sparrow hashes the template source for its cache key and uses an LRU with a fixed default size. That gives us two practical guarantees:

- frequently used subscriptions reuse parsed templates;
- a stream of unique templates cannot grow memory forever.

The cache stores parsed syntax, not rendered output. Rendered output depends on the event, attempt number, and timestamp, so caching final bytes would be incorrect. Parsing is reusable; delivery bodies are not.

This distinction matters for any template engine. “It supports compilation” is not the same as “the application has a safe cache policy.” The host still owns invalidation, bounds, concurrency, and the data passed to execution.

## Security: reduce capability before adding a sandbox

Go's `text/template` evaluates template actions against the values and functions the host provides. Sparrow passes an explicit map context and a curated function map. It does not pass a database handle, request object, filesystem object, process environment, or arbitrary callable application objects.

That is a strong default: templates can transform data, but they cannot turn a subscription into a general-purpose program merely because the host process is powerful.

Other engines can be operated safely too. Jinja provides a sandbox that can intercept attribute access, method calls, operators, and mutations. Its own documentation still warns that a sandbox is not perfect security and recommends resource limits, exception handling, and passing only relevant data. Handlebars escapes HTML expressions by default, but its documentation warns that JavaScript strings and unsafe helpers need separate care.

The lesson is not “Go is secure and JavaScript is insecure.” The lesson is that security comes from capability control and resource limits. Go gives Sparrow a narrow starting point without requiring a general-purpose runtime sandbox for a data-mapping problem.

Sparrow adds the limits the standard engine does not provide by itself:

- output is limited to 1 MiB;
- execution is limited to five seconds;
- strict missing-key mode can fail instead of silently emitting `<no value>`;
- template failures are recorded as `template_error` and do not count against receiver health;
- a failed transform is not silently replaced unless the subscription explicitly opts into fallback behavior.

These controls are more important than the branding of the language. A fast template engine with no resource limits is still a denial-of-service risk.

## Go templates are intentionally not JavaScript

A subscription transform should not become a place to write business software. If a mapping needs database lookups, external API calls, complex branching, or state, it belongs in an application or integration service.

Sparrow's template language is intentionally boring:

- read the event context;
- select and reshape fields;
- apply small deterministic helpers;
- print text or JSON;
- stop.

That boundary is useful operationally. A transform can be validated when a subscription is changed, tested against a sample payload, parsed and cached by the worker, and retried after a correction without deploying code.

## The trade-off

Go templates are not the most expressive option. They are less familiar to teams that live in JavaScript, and `text/template` is not an HTML auto-escaping engine. That is acceptable because Sparrow transforms webhook bodies, not untrusted HTML pages. For JSON, the `json` helper is the right boundary; hand-built JSON strings are not.

If a future requirement needs a richer language, the safe path is to add that capability deliberately: define the data model, helper surface, resource budgets, cache behavior, and failure semantics first. Switching languages alone would not solve those problems.

For the current job, Go is the smaller and more honest tool. It keeps the transformation close to the queue worker, makes the memory and cache policy visible, limits what a subscription can do, and avoids paying for a second runtime on every Sparrow installation.

## Try a transform

A subscription can reshape an event without a proxy service:

```go
{{ dict
  "id" .payload.id
  "amount_cents" (toInt (mul .payload.amount 100))
  "currency" .payload.currency
  "event" .event_name
  | json }}
```

Use `dig` or `index` for optional fields, and test the result against a sample event before enabling it. The [payload transformation guide](/sparrow/guides/payload-transformation/) covers the available helpers, error modes, retries, and examples.

### Sources and further reading

- [Go `text/template` package](https://pkg.go.dev/text/template)
- [Jinja sandbox documentation](https://jinja.palletsprojects.com/en/stable/sandbox/)
- [Handlebars guide](https://handlebarsjs.com/guide/)
- [Sparrow payload transformations](/sparrow/guides/payload-transformation/)
