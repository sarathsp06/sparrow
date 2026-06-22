---
type: Go Package
title: pkg/errors
description: Error classification with 10 categories for webhook delivery outcomes and gRPC-compatible service errors
tags: [errors, classification, leaf]
timestamp: 2026-06-22T00:00:00Z
---

# pkg/errors

Classifies webhook delivery errors into 10 categories and provides client-safe `ServiceError` with gRPC status codes.

## Error Categories

| Category | Retryable |
|----------|-----------|
| `success` | No |
| `client_error` | No (4xx) |
| `server_error` | **Yes** (5xx) |
| `timeout` | **Yes** |
| `dns_error` | No |
| `tls_error` | No |
| `connection_refused` | **Yes** |
| `network_error` | **Yes** |
| `unexpected_status` | No (2xx/3xx outside expected) |
| `rate_limited` | **Yes** (HTTP 429) |
| `unknown` | No |

## Functions

- `ClassifyHTTPStatus(statusCode) ErrorCategory`
- `ClassifyError(err error) ErrorCategory`
- `IsRetryableCategory(cat) bool`
- `InvalidInput(msg) *ServiceError`, `NotFoundError(msg) *ServiceError`, etc.

## Citations

- `pkg/errors/` — 4 files
