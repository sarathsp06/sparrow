---
title: Error Classification
description: How Sparrow classifies delivery errors and determines retry behavior
---

Sparrow classifies every delivery error into one of 10 categories. The classification determines whether the delivery is retried or permanently failed.

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
| `unknown` | No | Unclassified error |

## Retry Behavior

When a delivery attempt fails with a **retryable** error category, Sparrow re-enqueues the delivery with exponential backoff:

```
backoff = retry_backoff_seconds * 2^(attempt - 1) + random_jitter
```

Retries continue until:
- The delivery succeeds
- `max_retries` attempts are exhausted (terminal `FAILED` status)
- The event's `ttl_seconds` expires (terminal `EXPIRED` status)

**Non-retryable** errors immediately mark the delivery as `FAILED` regardless of remaining retry budget.

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

Use the [HealthService](/sparrow/reference/api/health-service/) to monitor error patterns:

- `GetWebhookHealth` returns error category breakdown (client_errors, server_errors, timeout_errors, network_errors) for the last 24 hours
- `ListWebhooksByHealth` finds all webhooks with `UNHEALTHY` or `DEGRADED` status
- `GetDeliveryAttempts` shows per-attempt error details for debugging
