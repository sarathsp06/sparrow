---
type: Concept
title: Error Classification
description: 10-category error classification for webhook delivery outcomes with retryability semantics
tags: [errors, classification, retry]
timestamp: 2026-06-22T00:00:00Z
---

# Error Classification

All webhook delivery outcomes are classified into categories for retry decisions, health tracking, and observability.

## Categories

| Category | Retryable | Source |
|----------|-----------|--------|
| `success` | No | 2xx response |
| `client_error` | No | 4xx response |
| `server_error` | **Yes** | 5xx response |
| `timeout` | **Yes** | Request timeout |
| `dns_error` | No | DNS resolution failure |
| `tls_error` | No | TLS handshake failure |
| `connection_refused` | **Yes** | TCP connection refused |
| `network_error` | **Yes** | Generic network failure |
| `unexpected_status` | No | 2xx/3xx not in expected_status_codes |
| `rate_limited` | **Yes** | HTTP 429 with Retry-After |
| `unknown` | No | Unclassifiable |

## Citations

- `pkg/errors/` — implementation
