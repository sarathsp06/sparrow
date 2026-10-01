---
title: Speak the receiver's language
kicker: On Go templates
author: Sparrow team
description: Why Sparrow uses Go templates for subscription transformations instead of embedding JavaScript, and what a small, restrained language buys you on the delivery path.
pubDate: 2026-10-01
tags: [webhooks, go, performance, security]
---

A good interpreter is almost invisible. They listen to one person and speak to another, and the best ones add nothing of their own. No opinions, no flourishes, no clever improvements to what was said. Just the same meaning, in words the listener understands.

It turns out webhooks need an interpreter too.

## One event, three listeners

It started with one `payment.succeeded` event and three receivers that each wanted to hear about it differently.

The Slack channel wanted a single friendly line: `{ "text": "Acme paid USD" }`. The billing system wanted the amount in cents, not dollars. A partner wanted three fields out of a record with thirty, and nothing else.

Sending the raw event to everyone meant each receiver had to write its own little adapter, which is a lot like shouting in your own language and expecting the room to cope. Running a separate proxy to reshape payloads meant one more service to deploy and one more pager. What we wanted was simpler: let each subscription say how its receiver likes to be spoken to.

That became `transform_template`. It lives on the subscription, runs just before delivery, and prints the body that receiver gets. One event, three subscriptions, three ways of saying the same thing.

The harder question was what language the interpreter should think in.

## The obvious answer

JavaScript came up first, and it was a fair suggestion. Most teams have Node somewhere. Handlebars, Nunjucks, and EJS are familiar. Plenty of people would happily write a small function instead of a template.

But "use JavaScript" can mean two quite different things:

1. use a JavaScript-flavoured template language such as Handlebars; or
2. run user-provided JavaScript inside the server.

The second is much heavier than it sounds. The server has to embed or call a JavaScript runtime, decide which globals exist, block filesystem and network access, limit CPU and memory, and keep that runtime's security model in step with its own. A language-level sandbox is not automatically a security boundary.

The first is lighter, but it still adds a second runtime and a new tree of dependencies. And it does not make output limits, timeouts, or error handling go away. Those belong to the host, whatever the syntax looks like.

## Where the words are spoken

What settled it was looking at where this code would actually run. A template in a webhook server is not rendering a web page. It runs inside a queue worker, once for every subscription that receives an event. That is the busiest path in the whole system, and it comes with a few plain facts:

- the process is already Go;
- the input is structured event data, not arbitrary application objects;
- the output is almost always JSON or plain text;
- a transform must never make delivery unreliable;
- thousands of deliveries should share one small, predictable runtime.

On a path like that, the best tool is the one that does the job while adding the fewest moving parts. Go's standard `text/template` was already in the binary.

## A deliberately small vocabulary

Sparrow does not expose Go code. It exposes a small template language with a chosen set of helpers:

```go
{{ dict "text" (printf "%s paid %s" .payload.customer.name .payload.currency) | json }}
```

The template can see the event ID, event name, timestamp, attempt number, and payload. The helpers cover building maps and lists, reading optional fields, formatting, arithmetic, and turning the result into JSON. That is the whole vocabulary.

There is a kind of freedom in a small vocabulary. Poets who write sonnets know it: the form's limits push you toward clarity. A transform written in a narrow language is easy to read, easy to review, and hard to make dangerous.

## Learn once, speak many times

Parsing a template is the expensive part, so Sparrow does it once. Parsed templates live in a bounded LRU cache keyed by a hash of the template source, with strict and lenient missing-key modes cached separately. After warm-up, a delivery parses nothing. It runs the parsed template into a pooled buffer and gets bytes back.

The cache holds parsed templates, never rendered output. Output depends on the event, the attempt number, and the timestamp, so caching it would be wrong. Parsing is reusable; what you say each time is not.

Because the cache is bounded, a stream of unique templates cannot grow memory forever. A JavaScript engine could also precompile and cache; that is a valid design. It would still bring its own runtime, and someone would still need to decide bounds, invalidation, and concurrency.

## Room on a small machine

Plenty of Sparrow installations run on a small VM next to the database, or in a container with a tight memory limit. Adding a JavaScript runtime just to reshape JSON would raise the floor before a single template ran.

We are not claiming every Go template beats every JavaScript template. Real numbers depend on the engine and the workload. The narrower claim is the one that matters: for a server that is already Go, the standard-library engine avoids a second runtime and keeps the cache and output limits visible in one place.

## Taking power away first

There is a habit in security of adding a guard after you have handed out the keys. It is usually better not to hand out the keys.

Go templates can only call the functions and read the values the host gives them. Sparrow passes a plain map of event data and a curated set of functions. A template gets no database handle, no request object, no filesystem, no environment variables, and no arbitrary application objects. It can reshape data. It cannot turn a subscription into a program.

Other engines can be run safely too. Jinja has a sandbox, though its own documentation says a sandbox is not perfect security and recommends resource limits and passing only the data you need. Handlebars escapes HTML by default but warns that JavaScript strings and unsafe helpers need separate care. Safety comes from limiting what code can reach and how long it can run, whichever language you choose.

So Sparrow adds the limits the standard engine does not provide on its own:

- output is capped at 1 MiB;
- execution is capped at five seconds;
- strict missing-key mode can fail instead of printing `<no value>`;
- a failed transform is recorded as `template_error` and never counts against the receiver's health;
- a failed transform is not quietly replaced with the raw event unless the subscription opts into fallback.

A fast engine with no limits is still an invitation to a denial-of-service. The limits matter more than the language.

## Boring on purpose

An interpreter who starts making business decisions has stopped being an interpreter. The same goes for transforms. If a mapping needs database lookups, external calls, or state, it belongs in a real service. Sparrow's template language is meant to do five things: read the event, pick and reshape fields, apply small deterministic helpers, print text or JSON, and stop.

That restraint pays off in daily work. A template can be checked when the subscription changes, tested against a sample payload, cached by the worker, and fixed and retried without deploying any code.

## What we gave up

Go templates are not the most expressive option, and they feel less familiar to teams who live in JavaScript. `text/template` also does not auto-escape HTML. That is fine for webhook bodies, which are not web pages. For JSON, use the `json` helper rather than building strings by hand.

If a future need calls for a richer language, the right way is to add it on purpose: decide the data model, the helpers, the resource limits, the cache, and the failure behaviour first. Switching languages alone would solve none of those.

## Back to the three listeners

Here is what the billing system hears, amount in cents:

```go
{{ dict
  "id" .payload.id
  "amount_cents" (toInt (mul .payload.amount 100))
  "currency" .payload.currency
  "event" .event_name
  | json }}
```

Slack gets its one friendly line, the partner gets its three fields, and nobody runs a proxy. One event, said three ways, with nothing added that wasn't there.

Use `dig` or `index` for optional fields, and test the template against a sample event before you switch it on. The [payload transformation guide](/sparrow/guides/payload-transformation/) covers the helpers, error modes, and retries.

### Sources and further reading

- [Go `text/template` package](https://pkg.go.dev/text/template)
- [Jinja sandbox documentation](https://jinja.palletsprojects.com/en/stable/sandbox/)
- [Handlebars guide](https://handlebarsjs.com/guide/)
- [Sparrow payload transformations](/sparrow/guides/payload-transformation/)
