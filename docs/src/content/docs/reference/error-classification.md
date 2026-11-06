---
title: Error Classification
description: How Sparrow classifies delivery errors into categories, and which failures it retries with backoff and which stop the delivery immediately.
---

Sparrow classifies every delivery error into categories. The classification determines whether the delivery is retried or permanently failed. (For the errors the API itself returns to your calls, see [API Errors](/sparrow/reference/api-errors/).)

## Error Categories

| Category | Retryable | Description |
|----------|-----------|-------------|
| `success` | n/a | Delivery succeeded |
| `client_error` | No | HTTP 4xx response (bad request, unauthorized, not found, etc.) |
| `server_error` | **Yes** | HTTP 5xx response (internal server error, bad gateway, etc.) |
| `timeout` | **Yes** | Request timed out before receiving a response |
| `connection_refused` | **Yes** | Target endpoint refused the TCP connection |
| `network_error` | **Yes** | Other network errors (ECONNRESET, EPIPE, EHOSTUNREACH) |
| `dns_error` | No | DNS resolution failed (no such host) |
| `tls_error` | No | TLS/SSL handshake failure (certificate errors) |
| `rate_limited` | **Yes** | HTTP 429 response. Retried after `Retry-After` delay (doesn't count as attempt) |
| `unexpected_status` | No | HTTP 2xx/3xx response that did not match `expected_status_codes` |
| `template_error` | No | The subscription's transform failed to render, so nothing was sent (`on_transform_error: fail`). Retryable by hand after fixing the template. Never counts toward webhook health. See [When a template fails](/sparrow/guides/payload-transformation/#when-a-template-fails). |
| `unknown` | No | Unclassified error |

## Retry Behavior

When a delivery attempt fails with a **retryable** error category, Sparrow re-enqueues the delivery with exponential backoff:

```
backoff = retry_backoff_seconds * 2^(attempt - 1)   # capped at 24 hours
```

Each delay is then shortened by a random 0-20%, so deliveries that failed together
(a receiver outage) spread their retries out instead of arriving all at once.
The formula is the longest a retry waits.

### Default schedule

By default (`max_retries: 13`, `retry_backoff_seconds: 60`) a failing delivery
keeps retrying for about three days, like the retry windows of hosted webhook
senders. A receiver that is down over a weekend or through a long deploy gets
its deliveries once it is back, with nobody retrying them by hand.

| Retry | Wait before it (at most) | Time since the first attempt (at most) |
|---|---|---|
| 1 | 1 min | 1 min |
| 2 | 2 min | 3 min |
| 3 | 4 min | 7 min |
| 4 | 8 min | 15 min |
| 5 | 16 min | 31 min |
| 6 | 32 min | 1 h 3 min |
| 7 | 64 min | 2 h 7 min |
| 8 | 128 min | 4 h 15 min |
| 9 | 256 min | 8 h 31 min |
| 10 | 512 min | 17 h 3 min |
| 11 | 1024 min | 34 h 7 min |
| 12 | 24 h | 2 d 10 h |
| 13 | 24 h | 3 d 10 h |

Jitter makes each wait up to 20% shorter, so the last retry lands between
about 2.7 and 3.4 days after the first attempt. The UI shows the window for a
webhook's settings next to its retry fields.

Shorter or longer windows are a per-webhook setting:

- **Fail fast.** For a receiver someone watches, where a failure should show up
  within minutes and be retried by hand, use few retries. For example,
  `max_retries: 3` retries for about 7 minutes. That was the default up to
  Sparrow 1.3.
- **Longer.** Up to `max_retries: 20` and `retry_backoff_seconds: 3600`. Every
  wait is capped at 24 hours, so with the default base each retry past the
  eleventh adds one day.

A delivery that is waiting for its next retry has status `retrying`. While a
receiver is down, `retrying` counts grow for as long as the window, and the
receiver's health turns `failing` long before the deliveries are marked
`failed`. [Auto-disable](/sparrow/guides/webhook-health-alerts/#automatic-disabling)
pauses a webhook that keeps failing for longer (5 days by default), and pausing
holds its deliveries instead of dropping them.

Retries continue until:
- The delivery succeeds
- `max_retries + 1` attempts in total are exhausted (terminal `failed` status)
- The event's `ttl_seconds` expires (terminal `expired` status)

**Non-retryable** errors immediately mark the delivery as `failed` regardless of remaining retry budget.

A delivery with status `paused` was held because its subscription or webhook
was paused (by hand, or because Sparrow
[auto-disabled](/sparrow/guides/webhook-health-alerts/#automatic-disabling) the
webhook): either it was created during the pause, or it was queued or retrying
when the pause began. It has no error category, is never attempted until
retried, and does not affect health. See
[Pausing a Subscription](/sparrow/guides/subscription-pause/).

## Classification Logic

The error classifier inspects the Go error chain to determine the category:

1. **HTTP response code** — If a response was received:
   - 2xx matching `expected_status_codes` -> `success`
   - 4xx -> `client_error`
   - 5xx -> `server_error`

2. **Error type inspection** — If no response:
   - `*net.DNSError` -> `dns_error`
   - TLS-related errors -> `tls_error`
   - `net.Error` with `Timeout()` -> `timeout`
   - `syscall.ECONNREFUSED` -> `connection_refused`
   - `syscall.ECONNRESET`, `EPIPE`, `EHOSTUNREACH` -> `network_error`
   - String pattern fallback for edge cases

## Monitoring Errors

Use the [health endpoints](/sparrow/reference/api/) to monitor error patterns:

- `GET /v1/consumers/{consumer}/webhooks/{webhook_id}/health` returns attempt counts and an error category breakdown (`client_errors`, `server_errors`, `timeout_errors`, `network_errors`, `unexpected_status_errors`) for the last 24 hours, plus `consecutive_failures` and the `rules` behind the health label
- `GET /v1/webhooks` (filtered by health) finds all webhooks with `unhealthy` or `degraded` status
- `GET /v1/consumers/{consumer}/deliveries/{delivery_id}/attempts` shows per-attempt error details for debugging
