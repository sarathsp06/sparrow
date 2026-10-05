---
title: Error Classification
description: How Sparrow classifies delivery errors and determines retry behavior
---

Sparrow classifies every delivery error into categories. The classification determines whether the delivery is retried or permanently failed.

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

Retries continue until:
- The delivery succeeds
- `max_retries` attempts are exhausted (terminal `failed` status)
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

- `GET /v1/consumers/{consumer}/webhooks/{webhook_id}/health` returns error category breakdown (client_errors, server_errors, timeout_errors, network_errors) for the last 24 hours
- `GET /v1/webhooks` (filtered by health) finds all webhooks with `unhealthy` or `degraded` status
- `GET /v1/consumers/{consumer}/deliveries/{delivery_id}/attempts` shows per-attempt error details for debugging
