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

## Annotations

All annotations use the `@` prefix for consistency, following conventions from Javadoc, JSDoc, and similar systems.

### @required

Mark a field as required:

```protobuf
// @required The URL to deliver webhook events to.
// @example "https://example.com/webhook"
string url = 1;
```

### @deprecated

Mark a field or message as deprecated. Deprecated fields are excluded from generated documentation:

```protobuf
// @deprecated Use new_field instead.
string old_field = 5;
```

Fields with the proto `deprecated` option are also detected:
```protobuf
string old_field = 5 [deprecated = true];
```

### @default

Document a field's default value:

```protobuf
// Maximum number of retry attempts. @default 5
int32 max_retries = 3;
```

### @range

Document valid value ranges:

```protobuf
// Number of items per page. @range 1-100 @default 50
int32 page_size = 2;
```

### @error

Document RPC error codes in RPC comments:

```protobuf
// RegisterWebhook creates a new webhook subscription.
// @error ALREADY_EXISTS if a webhook with the same URL already exists.
// @error INVALID_ARGUMENT if the URL is malformed.
rpc RegisterWebhook(RegisterWebhookRequest) returns (RegisterWebhookResponse);
```

### @example

Provide example values for fields. These are used to auto-generate curl commands and response JSON:

```protobuf
// @required The webhook endpoint URL.
// @example "https://example.com/webhook"
string url = 1;

// Maximum retry attempts. @example 5
int32 max_retries = 2;

// Whether the webhook is active. @example true
bool active = 3;
```

JSON values are parsed automatically. If parsing fails, the value is treated as a string.

#### Multi-line examples

For complex JSON values, use a fenced block with triple backticks:

```protobuf
// JSON metadata for the item.
// @example ```
// {"key": "value", "count": 1}
// ```
string metadata = 4;
```

The lines between the fences are joined into a single value and parsed as JSON.

## Complete Example

```protobuf
// Create a new user account.
// @error ALREADY_EXISTS if the email is taken.
rpc CreateUser(CreateUserRequest) returns (CreateUserResponse);

message CreateUserRequest {
  // @required The user's email address.
  // @example "alice@example.com"
  string email = 1;

  // Display name. @default "Anonymous"
  string display_name = 2;

  // Number of invites to pre-allocate. @range 0-100
  int32 invite_count = 3;

  // @deprecated Use display_name instead.
  string name = 4;
}
```

## Legacy Syntax

The following legacy patterns are still supported for backward compatibility:

| Legacy | Preferred |
|--------|-----------|
| `Required.` / `Required ` | `@required` |
| `Deprecated: reason` | `@deprecated reason` |
| `Default: VALUE.` | `@default VALUE` |
| `Range: MIN-MAX` | `@range MIN-MAX` |
| `Errors: CODE desc` | `@error CODE desc` |
