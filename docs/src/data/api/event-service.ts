import type { ApiService } from "./types";

const service: ApiService = {
  "service": "EventService",
  "package": "webhook",
  "description": "EventService manages event type definitions and event pushing. Event types must be registered before they can be pushed. Each event type has a unique name (scoped to the default tenant) and an optional JSON schema for payload validation.",
  "rpcs": [
    {
      "name": "RegisterEvent",
      "description": "RegisterEvent creates a new event type definition. If a JSON schema is provided, all future PushEvent payloads for this event are validated against it. The event name is the primary identifier (not a UUID).",
      "request": [
        {
          "name": "name",
          "type": "string",
          "required": true,
          "description": "Unique event name (e.g., \"order.created\", \"payment.completed\"). Required. Convention: use dot-separated lowercase names. @example \"order.created\"",
          "example": "order.created"
        },
        {
          "name": "description",
          "type": "string",
          "description": "Human-readable description of what this event represents. Optional. @example \"Fired when a new order is placed\"",
          "example": "Fired when a new order is placed"
        },
        {
          "name": "schema",
          "type": "Struct (JSON)",
          "description": "JSON Schema for validating PushEvent payloads. Optional. When set, all future PushEvent calls for this event type must conform to this schema. Passed as a google.protobuf.Struct (JSON object). @example {\"type\": \"object\", \"properties\": {\"order_id\": {\"type\": \"string\"}, \"amount\": {\"type\": \"number\"}}, \"required\": [\"order_id\", \"amount\"]}",
          "example": {
            "properties": {
              "amount": {
                "type": "number"
              },
              "order_id": {
                "type": "string"
              }
            },
            "required": [
              "order_id",
              "amount"
            ],
            "type": "object"
          }
        },
        {
          "name": "metadata",
          "type": "map\u003cstring, string\u003e",
          "description": "Arbitrary key-value metadata attached to the event type definition. Optional."
        },
        {
          "name": "active",
          "type": "bool",
          "description": "Whether the event type is active. Default: true. Inactive event types cannot receive new PushEvent calls. @example true",
          "example": true
        }
      ],
      "response": [
        {
          "name": "created_at",
          "type": "Timestamp",
          "description": "Timestamp when the event type was created. @example \"2025-01-15T10:30:00Z\"",
          "example": "2025-01-15T10:30:00Z"
        }
      ],
      "errors": [
        {
          "code": "ALREADY_EXISTS",
          "description": "An event with the same name already exists."
        }
      ]
    },
    {
      "name": "ListEvents",
      "description": "ListEvents returns all registered event types, optionally filtered to active-only. Results are paginated.",
      "request": [
        {
          "name": "active_only",
          "type": "bool",
          "description": "When true, only return active event types. Default: false (return all). @example true",
          "example": true
        },
        {
          "name": "pagination",
          "type": "PaginationRequest",
          "description": "Pagination parameters. Default: limit=50, offset=0."
        }
      ],
      "response": [
        {
          "name": "events",
          "type": "RegisteredEvent[]",
          "description": "Event type definitions matching the filter criteria."
        },
        {
          "name": "pagination",
          "type": "PaginationResponse",
          "description": "Pagination metadata."
        }
      ]
    },
    {
      "name": "UpdateEvent",
      "description": "UpdateEvent modifies an existing event type's description, schema, metadata, or active flag. Updating the schema does not retroactively validate previously pushed events.",
      "request": [
        {
          "name": "name",
          "type": "string",
          "required": true,
          "description": "Event name to update. Required. This is the lookup key (not a UUID). @example \"order.created\"",
          "example": "order.created"
        },
        {
          "name": "description",
          "type": "string",
          "description": "Updated description. Omit to leave unchanged. @example \"Updated: Fired when a new order is placed\"",
          "example": "Updated: Fired when a new order is placed"
        },
        {
          "name": "schema",
          "type": "Struct (JSON)",
          "description": "Updated JSON Schema for payload validation. Omit to leave unchanged. Note: changing the schema does not retroactively validate previously pushed events."
        },
        {
          "name": "metadata",
          "type": "map\u003cstring, string\u003e",
          "description": "Updated metadata. Omit to leave unchanged."
        },
        {
          "name": "active",
          "type": "bool",
          "description": "Updated active flag. Omit to leave unchanged."
        }
      ],
      "errors": [
        {
          "code": "NOT_FOUND",
          "description": "The event name does not exist."
        }
      ]
    },
    {
      "name": "DeleteEvent",
      "description": "DeleteEvent permanently removes an event type definition. Existing subscriptions referencing this event name are not automatically deleted.",
      "request": [
        {
          "name": "name",
          "type": "string",
          "required": true,
          "description": "Event name to delete. Required. @example \"order.created\"",
          "example": "order.created"
        }
      ],
      "errors": [
        {
          "code": "NOT_FOUND",
          "description": "The event name does not exist."
        }
      ]
    },
    {
      "name": "GetEvent",
      "description": "GetEvent returns a single event type by name, including its schema and auto-generated sample payload.",
      "request": [
        {
          "name": "name",
          "type": "string",
          "required": true,
          "description": "Event name to look up. Required. @example \"order.created\"",
          "example": "order.created"
        }
      ],
      "response": [
        {
          "name": "event",
          "type": "RegisteredEvent",
          "description": "The event type definition, including schema and sample_payload."
        }
      ],
      "errors": [
        {
          "code": "NOT_FOUND",
          "description": "The event name does not exist."
        }
      ]
    },
    {
      "name": "PushEvent",
      "description": "PushEvent emits an event instance. This is the primary ingestion endpoint. On success, the event is persisted and a background job is enqueued to fan out deliveries to all matching subscriptions (by namespace + event_name + label_filters). The response returns immediately with the event_id; delivery happens asynchronously. If the event type has a JSON schema, the payload is validated before acceptance.",
      "request": [
        {
          "name": "namespace",
          "type": "string",
          "required": true,
          "description": "Namespace to push the event into. Required. Only subscriptions in this namespace are matched. @example \"production\"",
          "example": "production"
        },
        {
          "name": "event",
          "type": "string",
          "required": true,
          "description": "Event type name (must match a registered event type). Required. Subscriptions with matching event_name in this namespace will receive deliveries. @example \"order.created\"",
          "example": "order.created"
        },
        {
          "name": "payload",
          "type": "Struct (JSON)",
          "required": true,
          "description": "Event payload as a JSON object. Required. If the event type has a schema, this payload is validated against it before acceptance. This payload (or its template-transformed version) becomes the HTTP request body sent to each matching webhook. @example {\"order_id\": \"ord-123\", \"amount\": 99.99, \"currency\": \"USD\"}",
          "example": {
            "amount": 99.99,
            "currency": "USD",
            "order_id": "ord-123"
          }
        },
        {
          "name": "ttl_seconds",
          "type": "int64",
          "description": "Time-to-live in seconds for delivery retries. Optional. When set, deliveries that haven't succeeded within this window transition to EXPIRED. Default: 0 (no expiration -- retries continue until max_retries is exhausted)."
        },
        {
          "name": "metadata",
          "type": "map\u003cstring, string\u003e",
          "description": "Arbitrary key-value metadata attached to this event instance. Optional. Metadata is stored with the event record and available in event reports, but is NOT included in the delivery payload sent to webhooks."
        },
        {
          "name": "id",
          "type": "string",
          "description": "Client-provided event ID for idempotency. Optional. If provided and an event with this ID already exists, the existing event_id is returned without creating a duplicate."
        },
        {
          "name": "labels",
          "type": "map\u003cstring, string\u003e",
          "description": "Labels for label-based subscription matching. Optional. When set, only subscriptions whose label_filters are a subset of these labels will receive deliveries. Labels use AND logic: all filter keys must match. Subscriptions with no label_filters match all events regardless of labels. @example {\"region\": \"us-east\", \"priority\": \"high\"}",
          "example": {
            "priority": "high",
            "region": "us-east"
          }
        }
      ],
      "response": [
        {
          "name": "event_id",
          "type": "string",
          "description": "Server-generated UUID for the event instance. Use this ID to query delivery status via DeliveryService.ListDeliveries. @example \"e-550e8400-e29b-41d4-a716-446655440000\"",
          "example": "e-550e8400-e29b-41d4-a716-446655440000"
        }
      ],
      "errors": [
        {
          "code": "INVALID_ARGUMENT",
          "description": "The payload fails schema validation."
        },
        {
          "code": "NOT_FOUND",
          "description": "The event name is not registered."
        }
      ]
    },
    {
      "name": "ListEventReports",
      "description": "ListEventReports returns pushed event instances (not type definitions) for a namespace, ordered by created_at descending. Each report includes delivery stats (webhook_count, successful/failed/pending counts). Paginated, max 1000 per page.",
      "request": [
        {
          "name": "namespace",
          "type": "string",
          "required": true,
          "description": "Namespace to list events from. Required. @example \"production\"",
          "example": "production"
        },
        {
          "name": "event_name",
          "type": "string",
          "description": "Filter to events of this type name. Optional. @example \"order.created\"",
          "example": "order.created"
        },
        {
          "name": "pagination",
          "type": "PaginationRequest",
          "description": "Pagination parameters. Default: limit=50, offset=0. Max limit: 1000. @example {\"limit\": 25}",
          "example": {
            "limit": 25
          }
        }
      ],
      "response": [
        {
          "name": "events",
          "type": "EventReport[]",
          "description": "Event instances ordered by created_at descending (newest first)."
        },
        {
          "name": "pagination",
          "type": "PaginationResponse",
          "description": "Pagination metadata."
        }
      ]
    }
  ]
};

export default service;
