---
title: Proto Comment Guide
description: How to write proto comments for rich documentation
---

# Proto Comment Guide

This page documents the comment conventions supported by [proto2astro](https://github.com/sarathsp06/proto2astro). 
When you annotate your `.proto` files using these patterns, the documentation 
generator will automatically extract structured information and display it in the API reference.

## Leading Comments

Place comments directly above the message, field, RPC, or enum you want to document:

```protobuf
// RegisterWebhook creates a new webhook subscription.
// The webhook will start receiving events immediately after registration.
rpc RegisterWebhook(RegisterWebhookRequest) returns (RegisterWebhookResponse);
```

## Required Keyword

Include the word `Required` (capital R) anywhere in a field comment to mark it as required:

```protobuf
// Required. The URL to deliver webhook events to.
string url = 1;
```

## Deprecated Detection

Fields are marked as deprecated (and excluded from docs) in two ways:

1. The proto `deprecated` option:
```protobuf
string old_field = 5 [deprecated = true];
```

2. A comment starting with `Deprecated:`:
```protobuf
// Deprecated: Use new_field instead.
string old_field = 5;
```

## Default Values

Use the `Default: VALUE` pattern to document default values:

```protobuf
// Maximum number of retry attempts. Default: 5.
int32 max_retries = 3;
```

## Range Constraints

Use the `Range: MIN-MAX` pattern to document valid ranges:

```protobuf
// Number of items per page. Range: 1-100. Default: 50.
int32 page_size = 2;
```

## Error Codes

Document RPC error codes using the `Errors: CODE description` pattern in RPC comments:

```protobuf
// RegisterWebhook creates a new webhook subscription.
// Errors: ALREADY_EXISTS if a webhook with the same URL already exists.
// Errors: INVALID_ARGUMENT if the URL is malformed.
rpc RegisterWebhook(RegisterWebhookRequest) returns (RegisterWebhookResponse);
```

## @example Annotation

Provide example values for fields using `@example`. These are used to auto-generate curl commands and response JSON:

```protobuf
// The webhook endpoint URL. Required. @example "https://example.com/webhook"
string url = 1;

// Maximum retry attempts. @example 5
int32 max_retries = 2;

// Whether the webhook is active. @example true
bool active = 3;
```

JSON values are parsed automatically. If parsing fails, the value is treated as a string.
