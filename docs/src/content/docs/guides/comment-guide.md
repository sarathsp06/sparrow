---
title: API Documentation Guide
description: How the API reference is generated from the OpenAPI spec, and how schema fields produce rich documentation
---

# API Documentation Guide

The [API reference](/sparrow/reference/api/) is generated automatically from
Sparrow's OpenAPI spec (`api/openapi.yaml`), which is produced from the Go REST
definitions in `internal/rest`. The richer the spec, the richer the generated
reference. This page describes the OpenAPI schema fields Sparrow uses to produce
descriptions, required-field markers, defaults, constraints, error codes, and
examples.

## Descriptions

Every schema and property carries a `description`, which becomes the prose shown
in the reference:

```yaml
RegisterWebhookRequest:
  type: object
  description: |
    Creates a new webhook subscription. The webhook starts receiving
    events immediately after registration.
```

## Required Fields

List required properties in the schema's `required` array. They are flagged as
required in the reference:

```yaml
RegisterWebhookRequest:
  type: object
  required: [url]
  properties:
    url:
      type: string
      description: The URL to deliver webhook events to.
```

## Deprecated Fields

Mark a property `deprecated: true` to flag it (and exclude it from the primary
reference):

```yaml
old_field:
  type: string
  deprecated: true
  description: Use new_field instead.
```

## Default Values

Document defaults with the `default` keyword:

```yaml
max_retries:
  type: integer
  default: 5
  description: Maximum number of retry attempts.
```

## Range Constraints

Use `minimum`/`maximum` to document valid ranges:

```yaml
page_size:
  type: integer
  minimum: 1
  maximum: 100
  default: 50
  description: Number of items per page.
```

## Error Responses

Document error responses per operation under `responses`, keyed by HTTP status
code:

```yaml
responses:
  '409':
    description: A webhook with the same URL already exists.
  '400':
    description: The URL is malformed.
```

## Examples

Provide example values with `example` (or `examples`). These are used to
auto-generate the sample requests and responses shown in the reference:

```yaml
url:
  type: string
  example: "https://example.com/webhook"
max_retries:
  type: integer
  example: 5
active:
  type: boolean
  example: true
```

Example values are parsed as JSON where applicable; otherwise they are treated
as strings.
