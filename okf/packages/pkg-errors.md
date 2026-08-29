---
type: Go Package
title: pkg/errors
description: Error classification with 10 categories for webhook delivery outcomes, plus client-safe service errors
tags: [errors, classification, leaf]
timestamp: 2026-08-29T00:00:00Z
---

# pkg/errors

Classifies webhook delivery errors into 10 categories (`category.go`, `ErrorCategory`) and provides a client-safe `ServiceError` (`service.go`) tagged with a local `Status` enum (`status.go`) for REST status-code mapping — replaces the gRPC status codes used before the REST/OpenAPI migration.

## Error Categories (delivery outcomes)

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
- `Error(status Status, msg string) *ServiceError`, `Errorf`, `Wrap`, `Wrapf` — construct client-safe `ServiceError`, translated to HTTP status by `internal/rest/errors.go`

## Citations

- `pkg/errors/` — 5 files (`category.go`, `category_test.go`, `service.go`, `service_test.go`, `status.go`)
