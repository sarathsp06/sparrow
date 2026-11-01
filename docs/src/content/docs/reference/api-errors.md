---
title: API Errors
description: The error bodies the Sparrow API returns, which layer sends each one, and what the status codes mean.
---

A failed API call gets one of two body shapes, depending on which layer
refused it. Check the `Content-Type` header to tell them apart. (Delivery
errors, the outcome of sending a webhook, are covered in [Error
Classification](/sparrow/reference/error-classification/).)

## Errors from the API: problem details

Every `/v1` operation (and the same operations through `/portal/api/`)
answers errors as [RFC 9457](https://www.rfc-editor.org/rfc/rfc9457) problem
details, with `Content-Type: application/problem+json`:

```json
{
  "title": "Bad Request",
  "status": 400,
  "detail": "validation failed",
  "errors": [
    {
      "message": "expected number <= 1000",
      "location": "query.limit",
      "value": 5000
    }
  ]
}
```

- `title` is the HTTP status text and `status` the code.
- `detail` says what went wrong in words, e.g. `webhook not found` or
  `delivery already succeeded. Use force to resubmit anyway`.
- `errors` is present for request validation failures: one entry per problem,
  with `location` (`path.`, `query.`, `header.` or `body.` plus the field) and
  the offending `value`.

A `500` never carries internal details: the server logs them, and `detail`
only names the failed operation.

## Errors from authentication and the portal gateway

Requests refused before they reach an operation get plain
`Content-Type: application/json`, in one of two shapes.

Authentication failures (no, unknown, expired or revoked credential) and an
unavailable token store:

```json
{ "error": "unauthorized", "reason": "expired", "message": "this token has expired" }
```

| Status | `error` | `reason` |
|---|---|---|
| `401` | `unauthorized` | `missing`, `invalid`, `expired` or `revoked` |
| `503` | `unavailable` | absent; retry after the `Retry-After` header (5 seconds) |

Authorization failures, mostly from the portal gateway, carry no `reason`:

```json
{ "error": "forbidden", "message": "consumer-scoped tokens can only call the portal API under /portal/api/" }
```

| Status | When |
|---|---|
| `401` | A `/portal/api/` request without `Authorization: Bearer`, or with a token that is not valid |
| `403` | A consumer-scoped token on `/v1`, a tenant-wide token on `/portal/api/`, or a portal path the token may not reach (such as pushing events) |

Two smaller cases use the same plain shape: `POST /invite/redeem` answers an
unknown, used, cancelled or expired invite with
`400 {"error": "invalid_invite", "message": "..."}`, and, with the embedded
UI on, a non-`GET` request to an unknown path gets `404 {"error": "not found"}`.

A client that handles both shapes can read `detail` from problem details and
`message` from the plain JSON.

## Status codes

| Code | Meaning |
|---|---|
| `400` | Invalid input: a malformed id, an out-of-range value, an unknown enum value, a bad filter or cursor. |
| `401` | No valid credential (see above). |
| `403` | The credential cannot use this path (see above). |
| `404` | The resource does not exist in this consumer, or the event type is not registered. |
| `409` | Conflict or wrong state: a duplicate (webhook URL, subscription, event type name), an inactive event type, pausing a paused webhook, a batch job that is not pending, a breaking schema change without `allow_breaking`. |
| `413` | Request body over 1 MiB for an API request (5 MiB for event-type import), or over `SPARROW_MAX_BODY_BYTES` (default 5 MiB). |
| `429` | Rate limited (the AI provider, for template drafting). |
| `500` | Internal error; details are in the server log. |
| `503` | Temporarily unavailable: the token store, the template engine at capacity, or AI drafting not configured. Retry later. |
| `504` | Timed out: an AI draft longer than `SPARROW_AI_TIMEOUT`. |

Each operation in the [API reference](/sparrow/reference/api/) lists the codes
it can return and what they mean for it.
