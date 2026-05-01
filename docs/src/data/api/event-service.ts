import type { ApiService } from "./types";

const service: ApiService = {
  "service": "EventService",
  "package": "webhook",
  "description": "EventService manages event type definitions and event pushing. Event types must be registered before they can be pushed. Each event type has a unique name (scoped to the default tenant) and an optional JSON schema for payload validation.",
  "rpcs": [
    {
      "name": "RegisterEvent",
      "description": "RegisterEvent creates a new event type definition. If a JSON schema is provided, all future PushEvent payloads for this event are validated against it. The event name is the primary identifier (not a UUID). Errors: ALREADY_EXISTS if an event with the same name already exists.",
      "request": [
        {
          "name": "name",
          "type": "string",
          "description": "Unique event name (e.g., \"order.created\", \"payment.completed\"). Required. Convention: use dot-separated lowercase names.",
          "example": "order.created"
        },
        {
          "name": "description",
          "type": "string",
          "description": "Human-readable description of what this event represents. Optional.",
          "example": "Fired when a new order is placed"
        },
        {
          "name": "schema",
          "type": "Struct (JSON)",
          "description": "JSON Schema for validating PushEvent payloads. Optional. When set, all future PushEvent calls for this event type must conform to this schema. Passed as a google.protobuf.Struct (JSON object).",
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
          "type": "map<string, string>",
          "description": "Arbitrary key-value metadata attached to the event type definition. Optional."
        },
        {
          "name": "active",
          "type": "bool",
          "description": "Whether the event type is active. Default: true. Inactive event types cannot receive new PushEvent calls.",
          "example": true
        }
      ],
      "response": [
        {
          "name": "event_id",
          "type": "string",
          "description": "The event type name."
        },
        {
          "name": "created_at",
          "type": "Timestamp",
          "description": "When the event type was created."
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
          "description": "When true, only return active event types. Default: false (return all).",
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
      "description": "UpdateEvent modifies an existing event type's description, schema, metadata, or active flag. Updating the schema does not retroactively validate previously pushed events. Errors: NOT_FOUND if the event name does not exist.",
      "request": [
        {
          "name": "name",
          "type": "string",
          "description": "Event name to update. Required. This is the lookup key (not a UUID).",
          "example": "order.created"
        },
        {
          "name": "description",
          "type": "string",
          "description": "Updated description. Omit to leave unchanged.",
          "example": "Updated: Fired when a new order is placed"
        },
        {
          "name": "schema",
          "type": "Struct (JSON)",
          "description": "Updated JSON Schema for payload validation. Omit to leave unchanged. Note: changing the schema does not retroactively validate previously pushed events."
        },
        {
          "name": "metadata",
          "type": "map<string, string>",
          "description": "Updated metadata. Omit to leave unchanged."
        },
        {
          "name": "active",
          "type": "bool",
          "description": "Updated active flag. Omit to leave unchanged."
        }
      ]
    },
    {
      "name": "DeleteEvent",
      "description": "DeleteEvent permanently removes an event type definition. Existing subscriptions referencing this event name are not automatically deleted. Errors: NOT_FOUND if the event name does not exist.",
      "request": [
        {
          "name": "name",
          "type": "string",
          "description": "Event name to delete. Required.",
          "example": "order.created"
        }
      ]
    },
    {
      "name": "GetEvent",
      "description": "GetEvent returns a single event type by name, including its schema and auto-generated sample payload. Errors: NOT_FOUND if the event name does not exist.",
      "request": [
        {
          "name": "name",
          "type": "string",
          "description": "Event name to look up. Required.",
          "example": "order.created"
        }
      ],
      "response": [
        {
          "name": "event",
          "type": "RegisteredEvent",
          "description": "The event type definition, including schema and sample_payload."
        }
      ]
    },
    {
      "name": "PushEvent",
      "description": "PushEvent emits an event instance. This is the primary ingestion endpoint. On success, the event is persisted and a background job is enqueued to fan out deliveries to all matching subscriptions (by namespace + event_name + label_filters). The response returns immediately with the event_id; delivery happens asynchronously. If the event type has a JSON schema, the payload is validated before acceptance. Errors: INVALID_ARGUMENT if the payload fails schema validation. Errors: NOT_FOUND if the event name is not registered.",
      "request": [
        {
          "name": "namespace",
          "type": "string",
          "description": "Namespace to push the event into. Required. Only subscriptions in this namespace are matched.",
          "example": "production"
        },
        {
          "name": "event",
          "type": "string",
          "description": "Event type name (must match a registered event type). Required. Subscriptions with matching event_name in this namespace will receive deliveries.",
          "example": "order.created"
        },
        {
          "name": "payload",
          "type": "Struct (JSON)",
          "description": "Event payload as a JSON object. Required. If the event type has a schema, this payload is validated against it before acceptance. This payload (or its template-transformed version) becomes the HTTP request body sent to each matching webhook.",
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
          "type": "map<string, string>",
          "description": "Arbitrary key-value metadata attached to this event instance. Optional. Metadata is stored with the event record and available in event reports, but is NOT included in the delivery payload sent to webhooks."
        },
        {
          "name": "id",
          "type": "string",
          "description": "Client-provided event ID for idempotency. Optional. If provided and an event with this ID already exists, the existing event_id is returned without creating a duplicate."
        },
        {
          "name": "labels",
          "type": "map<string, string>",
          "description": "Labels for label-based subscription matching. Optional. When set, only subscriptions whose label_filters are a subset of these labels will receive deliveries. Labels use AND logic: all filter keys must match. Subscriptions with no label_filters match all events regardless of labels.",
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
          "description": "Server-generated UUID for the event instance. Use this ID to query delivery status via DeliveryService.ListDeliveries.",
          "example": "e-550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "warnings",
          "type": "string[]",
          "description": "Schema validation warnings. Populated when the event payload does not match the registered JSON schema. The event is still accepted and stored, but schema_valid is set to false on the event record. Each string describes a specific validation failure (e.g., \"field 'amount': expected number, got string\"). Empty when the payload passes validation or no schema is registered."
        },
        {
          "name": "duplicate",
          "type": "bool",
          "description": "True when the request was deduplicated by idempotency key. The returned event_id belongs to the previously created event. No new event record or deliveries are created."
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
          "description": "Namespace to list events from. Required.",
          "example": "production"
        },
        {
          "name": "event_name",
          "type": "string",
          "description": "Filter to events of this type name. Optional.",
          "example": "order.created"
        },
        {
          "name": "pagination",
          "type": "PaginationRequest",
          "description": "Pagination parameters. Default: limit=50, offset=0. Max limit: 1000.",
          "example": {
            "limit": 25
          }
        },
        {
          "name": "schema_valid",
          "type": "bool",
          "description": "Filter by schema validation status. When set, only events matching the specified schema_valid value are returned."
        },
        {
          "name": "labels",
          "type": "map<string, string>",
          "description": "Filter by labels using JSONB containment. Only events whose labels contain all specified key-value pairs are returned.",
          "example": {
            "region": "us-east"
          }
        },
        {
          "name": "created_after",
          "type": "Timestamp",
          "description": "Filter to events created at or after this timestamp."
        },
        {
          "name": "created_before",
          "type": "Timestamp",
          "description": "Filter to events created at or before this timestamp."
        },
        {
          "name": "prepare_repush",
          "type": "bool",
          "description": "When true, snapshot all matching event IDs (up to 10,000) into a batch job and return a repush_id in the response. Pass that ID to RePushEvents to re-push the exact set of events that matched this query."
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
        },
        {
          "name": "repush_id",
          "type": "string",
          "description": "Batch ID for deterministic re-push. Only populated when prepare_repush=true was set in the request. Pass to RePushEvents."
        }
      ]
    },
    {
      "name": "GetEventRecord",
      "description": "GetEventRecord retrieves a single pushed event instance by its UUID. Returns the event record with its payload, metadata, labels, and aggregated delivery statistics (webhook_count, successful/failed/pending counts). This is different from GetEvent which returns an event type definition by name. Errors: NOT_FOUND if the event_id does not exist. Errors: INVALID_ARGUMENT if the event_id is not a valid UUID.",
      "request": [
        {
          "name": "event_id",
          "type": "string",
          "description": "UUID of the event instance. Required.",
          "example": "e-550e8400-e29b-41d4-a716-446655440000"
        }
      ],
      "response": [
        {
          "name": "event",
          "type": "EventReport",
          "description": "The event instance with aggregated delivery statistics."
        },
        {
          "name": "labels",
          "type": "map<string, string>",
          "description": "Labels attached when the event was pushed."
        },
        {
          "name": "expires_at",
          "type": "Timestamp",
          "description": "When the event expires (based on TTL). Zero value if no TTL was set."
        }
      ]
    },
    {
      "name": "RePushEvent",
      "description": "RePushEvent replays a single previously pushed event as if it were pushed fresh. Loads the original event record and re-pushes through the standard PushEvent pipeline. Errors: NOT_FOUND if the event_id does not exist. Errors: INVALID_ARGUMENT if the event_id is not a valid UUID.",
      "request": [
        {
          "name": "event_id",
          "type": "string",
          "description": "UUID of the original event to replay. Required. The event must exist in the event_records table.",
          "example": "e-550e8400-e29b-41d4-a716-446655440000"
        }
      ],
      "response": [
        {
          "name": "event_id",
          "type": "string",
          "description": "Server-generated UUID for the new event instance. This is a brand-new event; the original event is not modified.",
          "example": "e-660e8400-e29b-41d4-a716-446655440001"
        },
        {
          "name": "warnings",
          "type": "string[]",
          "description": "Schema validation warnings for the re-pushed payload. The original payload is validated against the CURRENT event type schema. Empty when the payload passes validation or no schema is registered."
        }
      ]
    },
    {
      "name": "RePushEvents",
      "description": "RePushEvents executes a deterministic batch re-push of events whose IDs were previously snapshotted via ListEventReports with prepare_repush=true. Each event is re-pushed as if it were pushed fresh: new event_id, current schema validation. The batch is processed asynchronously via a River job; poll GetRepushStatus for progress. Errors: NOT_FOUND if the repush_id does not exist or has expired. Errors: FAILED_PRECONDITION if the batch is not in 'pending' status.",
      "request": [
        {
          "name": "repush_id",
          "type": "string",
          "required": true,
          "description": "Batch ID returned by ListEventReports when prepare_repush=true."
        }
      ],
      "response": [
        {
          "name": "repush_id",
          "type": "string",
          "description": "Batch ID for polling status."
        },
        {
          "name": "total",
          "type": "int32",
          "description": "Total number of events that will be re-pushed."
        },
        {
          "name": "status",
          "type": "string",
          "description": "Current status (will be \"processing\" on success)."
        }
      ]
    },
    {
      "name": "GetRepushStatus",
      "description": "GetRepushStatus returns the current progress of a batch re-push operation. Errors: NOT_FOUND if the repush_id does not exist or has expired.",
      "request": [
        {
          "name": "repush_id",
          "type": "string",
          "required": true,
          "description": "Batch ID returned by RePushEvents or ListEventReports."
        }
      ],
      "response": [
        {
          "name": "batch.status",
          "type": "string",
          "description": "Current status of the batch job.",
          "example": "processing"
        },
        {
          "name": "batch.total",
          "type": "int32",
          "description": "Total number of items in the batch."
        },
        {
          "name": "batch.processed",
          "type": "int32",
          "description": "Number of items successfully processed so far."
        },
        {
          "name": "batch.failed",
          "type": "int32",
          "description": "Number of items that failed processing."
        },
        {
          "name": "batch.created_at",
          "type": "Timestamp",
          "description": "When the batch job was created."
        },
        {
          "name": "batch.expires_at",
          "type": "Timestamp",
          "description": "When the batch job expires (created_at + ttl_seconds)."
        }
      ]
    },
    {
      "name": "CancelRepush",
      "description": "CancelRepush aborts a batch re-push that is pending or in progress. Items already processed are not rolled back. Errors: NOT_FOUND if the repush_id does not exist. Errors: FAILED_PRECONDITION if the batch is already completed or cancelled.",
      "request": [
        {
          "name": "repush_id",
          "type": "string",
          "required": true,
          "description": "Batch ID to cancel."
        }
      ],
      "response": [
        {
          "name": "status",
          "type": "string",
          "description": "Current status after cancellation (will be \"cancelled\")."
        }
      ]
    }
  ]
};

export default service;
