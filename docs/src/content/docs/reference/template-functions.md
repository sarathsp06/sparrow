---
title: Template Functions
description: What a subscription's transform template can read, the 37 helper functions it can call, and an example of each that renders against a sample event.
---

A subscription's `transform_template` is a Go
[`text/template`](https://pkg.go.dev/text/template) that Sparrow renders on
every delivery. Its output replaces the default event envelope as the HTTP
body. This page lists what the template can read and every helper function it
can call. For how to turn a transform on, see
[Payload transformation](/sparrow/guides/payload-transformation/).

## The context

A template sees exactly five top-level keys. Everything your producer pushed is
under `.payload`, so a field called `email` is `.payload.email`, never `.email`.

| Key | Type | Value |
|-----|------|-------|
| `.event_id` | string | The event's ID. The same on every retry. |
| `.event_name` | string | The event type, e.g. `order.created`. |
| `.timestamp` | string | When **this delivery attempt** was rendered, as RFC 3339 in UTC (`2026-01-02T03:04:05Z`). A retry gets a later time. It is not when the event was pushed: put that in the payload if you need it. |
| `.attempt` | number | The attempt number, `1` on the first try. |
| `.payload` | object | The event's JSON payload, as pushed. |

The examples on this page render against this context:

```json
{
  "event_id": "evt_sample",
  "event_name": "order.created",
  "timestamp": "2026-01-02T03:04:05Z",
  "attempt": 1,
  "payload": {
    "order_id": "ord_42",
    "amount": 1999,
    "currency": "usd",
    "status": "paid",
    "created_at": "2026-01-02T03:04:05Z",
    "note": "  Leave at the door\n",
    "customer": {"name": "ada lovelace", "email": "ada+orders@example.com", "tier": "gold"},
    "items": [{"sku": "SKU-1", "qty": 2, "price": 4.5}, {"sku": "SKU-2", "qty": 1, "price": 10.99}],
    "tags": "gift,priority",
    "image": "receipt.pdf",
    "encoded_note": "aGVsbG8gd29ybGQ="
  }
}
```

### Try an example

The CLI renders a template with the same engine the server uses, without
calling the server. Save the `payload` object above as `payload.json`, put a
template in a file, and run:

```sh
sparrow template test my.tmpl --event-name order.created --payload @payload.json
```

It sets `event_id` to `evt_sample`, `attempt` to `1` and `timestamp` to the
current time, so outputs that show the time differ from the ones here.

## Rules that save you a failed delivery

1. **A missing key fails the render.** Subscriptions render strictly by default
   (`template_missing_key: error`): reading `.payload.coupon` when the payload
   has no `coupon` is an error, the delivery fails with `template_error`, and
   nothing is sent. `default` doesn't help, because the lookup fails before
   `default` runs. Read optional fields with [`dig`](#dig) or `index`:

   ```go
   {{ dig "coupon" "none" .payload }}               → none
   {{ default "none" (index .payload "coupon") }}   → none
   ```

   Setting `template_missing_key: zero` on the subscription renders a missing
   key as `<no value>` instead, which usually produces a wrong body rather than
   a failed one.

2. **Put strings through `json` when you build JSON.** `{{ .payload.order_id | json }}`
   emits `"ord_42"` with the quotes and escapes any quote or newline inside the
   value. Writing `"{{ .payload.order_id }}"` by hand breaks the body the day a
   value contains a `"`. `json` also escapes `<`, `>` and `&` as `<`,
   `>` and `&`, which every JSON parser reads back correctly.

3. **Numbers are floats.** JSON numbers arrive as floating point. Small ones
   print as you'd expect (`{{ .payload.amount }}` → `1999`), but a large one
   printed directly comes out in exponent form: `{{ mul .payload.amount 1000 }}`
   → `1.999e+06`. Use `json` (`1999000`), `toInt` (`1999000`), or
   `printf "%.2f"` to control the format.

4. **No output escaping.** Unlike `html/template`, nothing is escaped for you.
   Use `json` for JSON bodies and `urlencode` for form-encoded ones.

Rendering is also capped: output over 1 MB or a render taking more than 5
seconds fails the delivery.

## A complete template

A JSON body that uses several helpers. Each line is safe against quotes in the
data and against the optional `coupon` field being absent:

```go
{
  "text": {{ printf "New order %s from %s" .payload.order_id (title .payload.customer.name) | json }},
  "total": {{ printf "%.2f %s" (div .payload.amount 100) (upper .payload.currency) | json }},
  "tier": {{ dig "customer" "tier" "standard" .payload | json }},
  "coupon": {{ dig "coupon" "none" .payload | json }},
  "skus": {{ $skus := list }}{{ range .payload.items }}{{ $skus = append $skus .sku }}{{ end }}{{ $skus | json }},
  "tags": {{ split "," .payload.tags | json }},
  "attempt": {{ .attempt }},
  "sent_at": {{ .timestamp | json }}
}
```

Renders:

```json
{
  "text": "New order ord_42 from Ada Lovelace",
  "total": "19.99 USD",
  "tier": "gold",
  "coupon": "none",
  "skus": ["SKU-1","SKU-2"],
  "tags": ["gift","priority"],
  "attempt": 1,
  "sent_at": "2026-01-02T03:04:05Z"
}
```

## Quick reference

Go's built-in template functions are available too: `printf`, `index`, `eq`,
`ne`, `lt`, `gt`, `and`, `or`, `not`, plus `if`, `range` and `with`. Sparrow
replaces the built-in `len` and `slice` with its own versions (below).

| Group | Functions |
|-------|-----------|
| [Encoding](#encoding) | `json`, `urlencode`, `base64`, `base64decode` |
| [Time](#time) | `now`, `parseTime`, `formatTime` |
| [Strings](#strings) | `upper`, `lower`, `title`, `trim`, `trimSpace`, `replace`, `repeat`, `slice`, `truncate`, `ellipsis` |
| [Tests](#tests) | `contains`, `hasPrefix`, `hasSuffix` |
| [Lists and splitting](#lists-and-splitting) | `split`, `join`, `len` |
| [Missing values](#missing-values) | `dig`, `default` |
| [Building JSON](#building-json) | `dict`, `list`, `append`, `merge` |
| [Numbers](#numbers) | `add`, `sub`, `mul`, `div`, `mod`, `toInt`, `toFloat`, `toString` |

Arguments come first and the value last, so every helper works at the end of a
pipe: `{{ .payload.order_id | replace "_" "-" }}` is `replace "_" "-" .payload.order_id`.

In the tables below, each example renders against the sample context.

## Encoding

| Function | Does | Example | Output |
|----------|------|---------|--------|
| `json` | Encodes any value as JSON: strings get quotes, objects and arrays are serialized. | `{{ .payload.customer \| json }}` | `{"email":"ada+orders@example.com","name":"ada lovelace","tier":"gold"}` |
| | | `{{ json .payload.order_id }}` | `"ord_42"` |
| `urlencode` | Escapes a string for a URL query or a form-encoded body (spaces become `+`). | `{{ .payload.customer.email \| urlencode }}` | `ada%2Borders%40example.com` |
| `base64` | Standard base64 of a string. | `{{ .payload.order_id \| base64 }}` | `b3JkXzQy` |
| `base64decode` | Decodes standard base64. Invalid input fails the render. | `{{ .payload.encoded_note \| base64decode }}` | `hello world` |

## Time

`.timestamp` and time fields in a payload are **strings**. `formatTime` needs a
time value, so parse the string first with `parseTime`. Layouts use Go's
reference time, `Mon Jan 2 15:04:05 MST 2006`: RFC 3339 is
`2006-01-02T15:04:05Z07:00`.

| Function | Does | Example | Output |
|----------|------|---------|--------|
| `parseTime` | `parseTime layout string` parses a string into a time. A string that doesn't match the layout fails the render. | `{{ parseTime "2006-01-02" "2026-01-02" }}` | `2026-01-02 00:00:00 +0000 UTC` |
| `formatTime` | `formatTime layout time` formats a time. | `{{ parseTime "2006-01-02T15:04:05Z07:00" .payload.created_at \| formatTime "January 2, 2006 at 3:04 PM" }}` | `January 2, 2026 at 3:04 AM` |
| | | `{{ parseTime "2006-01-02T15:04:05Z07:00" .timestamp \| formatTime "15:04 MST" }}` | `03:04 UTC` |
| `now` | The server's current time. Prefer `.timestamp`, which is the same moment as a string. | `{{ now \| formatTime "2006-01-02" }}` | today's date |

`{{ .timestamp | formatTime "15:04" }}` does **not** work: it fails with
`expected time.Time; got string`.

## Strings

`slice`, `truncate` and `ellipsis` count bytes, not characters, so they can cut
a multi-byte character (an accented letter or an emoji) in half.

| Function | Does | Example | Output |
|----------|------|---------|--------|
| `upper` / `lower` | Changes case. | `{{ .payload.currency \| upper }}` | `USD` |
| `title` | Capitalizes the first letter of each word, leaves the rest. | `{{ .payload.customer.name \| title }}` | `Ada Lovelace` |
| `trim` | `trim chars string` removes any of `chars` from both ends. | `{{ trim "_" "__ord_42__" }}` | `ord_42` |
| `trimSpace` | Removes spaces, tabs and newlines from both ends. | `{{ .payload.note \| trimSpace \| json }}` | `"Leave at the door"` |
| `replace` | `replace old new string` replaces every occurrence. | `{{ replace "_" "-" .payload.order_id }}` | `ord-42` |
| `repeat` | `repeat n string`, at most 1,000 times. | `{{ repeat 3 "*" }}` | `***` |
| `slice` | `slice start end string`, end exclusive; `-1` means to the end. | `{{ slice 4 -1 .payload.order_id }}` | `42` |
| `truncate` | `truncate n string` cuts to at most `n` bytes. | `{{ truncate 6 .payload.customer.name }}` | `ada lo` |
| `ellipsis` | Like `truncate`, but ends a cut string with `...` (within the `n`). Strings that fit are unchanged; with `n` ≤ 3 it just cuts. | `{{ ellipsis 8 .payload.customer.name }}` | `ada l...` |

## Tests

Case-sensitive, and the string being tested comes last. Use them with `if`.

| Function | Example | Output |
|----------|---------|--------|
| `contains` | `{{ if contains "priority" .payload.tags }}rush{{ else }}normal{{ end }}` | `rush` |
| `hasPrefix` | `{{ if hasPrefix "ord_" .payload.order_id }}order{{ end }}` | `order` |
| `hasSuffix` | `{{ if hasSuffix ".pdf" .payload.image }}pdf{{ end }}` | `pdf` |

## Lists and splitting

| Function | Does | Example | Output |
|----------|------|---------|--------|
| `split` | `split separator string` turns a string into a list of strings. | `{{ split "," .payload.tags \| json }}` | `["gift","priority"]` |
| | | `{{ range split "," .payload.tags }}[{{ . }}]{{ end }}` | `[gift][priority]` |
| `join` | `join separator list` joins a list of strings. | `{{ join ", " (split "," .payload.tags) }}` | `gift, priority` |
| `len` | Length of a string (in bytes), a JSON array or a JSON object. | `{{ len .payload.items }}` | `2` |

Two limits to know:

- `join` only accepts the result of `split`. A JSON array from the payload
  fails with `expected []string`. To join payload values, `range` over them:
  `{{ range $i, $it := .payload.items }}{{ if $i }}, {{ end }}{{ $it.sku }}{{ end }}` → `SKU-1, SKU-2`.
- `len` returns `0` for the result of `split`. It counts strings, JSON arrays
  and JSON objects only.

To read one element of an array, use `index`: `{{ index .payload.items 0 "sku" }}` → `SKU-1`.

## Missing values

### dig

`dig key... default object` walks a path of keys and returns the default if
any key along the way is missing (or isn't an object). It never fails on a
missing key, so it's the way to read optional fields.

| Example | Output |
|---------|--------|
| `{{ dig "customer" "tier" "standard" .payload }}` | `gold` |
| `{{ dig "customer" "address" "city" "unknown" .payload }}` | `unknown` |
| `{{ dig "coupon" "none" .payload }}` | `none` |

`dig` can't step into arrays. Use `index` for those.

### default

`default fallback value` returns `fallback` when `value` is missing (`nil`) or
an empty string. `0` and `false` are kept.

| Example | Output |
|---------|--------|
| `{{ default "n/a" .payload.status }}` | `paid` |
| `{{ default "n/a" "" }}` | `n/a` |
| `{{ default "none" (index .payload "coupon") }}` | `none` |

`{{ .payload.coupon | default "none" }}` fails under strict rendering: see
[rule 1](#rules-that-save-you-a-failed-delivery).

## Building JSON

Build objects and arrays, then pipe them through `json`. It's safer than
writing braces and quotes by hand.

| Function | Does | Example | Output |
|----------|------|---------|--------|
| `dict` | Makes an object from key/value pairs. Keys must be strings. | `{{ dict "id" .payload.order_id "total" .payload.amount \| json }}` | `{"id":"ord_42","total":1999}` |
| `list` | Makes an array from its arguments. | `{{ list .payload.order_id .payload.status \| json }}` | `["ord_42","paid"]` |
| `append` | Returns the array with items added. Reassign with `=` inside `range` to collect values. | `{{ $skus := list }}{{ range .payload.items }}{{ $skus = append $skus .sku }}{{ end }}{{ $skus \| json }}` | `["SKU-1","SKU-2"]` |
| `merge` | Combines objects into a new one; later keys win. | `{{ merge .payload.customer (dict "source" "sparrow") \| json }}` | `{"email":"ada+orders@example.com","name":"ada lovelace","source":"sparrow","tier":"gold"}` |

Object keys come out sorted alphabetically.

## Numbers

Arguments can be JSON numbers or numeric strings. `add`, `sub`, `mul` and `div`
return floats, `mod` and `toInt` return integers. Dividing by zero fails the
render.

| Function | Example | Output |
|----------|---------|--------|
| `add` | `{{ add 1 2 }}` | `3` |
| `sub` | `{{ sub .payload.amount 999 }}` | `1000` |
| `mul` | `{{ mul .payload.amount 1000 \| toInt }}` | `1999000` |
| `div` | `{{ div .payload.amount 100 }}` | `19.99` |
| `mod` | `{{ mod 17 5 }}` | `2` |
| `toInt` | `{{ toInt "42" }}`, `{{ toInt 4.9 }}` | `42`, `4` (truncates) |
| `toFloat` | `{{ toFloat "4.5" }}` | `4.5` |
| `toString` | `{{ toString .payload.amount }}` | `1999` |

For money and other fixed decimals, use `printf`:
`{{ printf "%.2f" (div .payload.amount 100) }}` → `19.99`.
`toString` prints large floats in exponent form (`1e+06`), so prefer `toInt`
or `printf` for those.
