import type { ApiService } from "./types";

const service: ApiService = {
  service: "EventService",
  description:
    "The EventService manages event type definitions and event pushing. Event types must be registered before they can be pushed. Each event type has a unique name and an optional JSON schema for payload validation.",
  rpcs: [
    {
      name: "RegisterEvent",
      description:
        "Creates a new event type definition. If a JSON schema is provided, all future PushEvent payloads are validated against it.",
      request: [
        { name: "name", type: "string", required: true, description: "Unique event name (e.g., 'order.created'). Convention: dot-separated lowercase.", example: "order.created" },
        { name: "description", type: "string", description: "Human-readable description.", example: "Fired when a new order is placed" },
        { name: "schema", type: "Struct (JSON)", description: "JSON Schema for payload validation.", example: { type: "object", properties: { order_id: { type: "string" }, amount: { type: "number" } }, required: ["order_id", "amount"] } },
        { name: "metadata", type: "map<string, string>", description: "Arbitrary key-value metadata." },
        { name: "active", type: "bool", description: "Whether the event type is active. Default: true.", example: true },
      ],
      response: [
        { name: "created_at", type: "Timestamp", description: "When the event type was created.", example: "2025-01-15T10:30:00Z" },
      ],
      errors: [
        { code: "ALREADY_EXISTS", description: "An event with the same name already exists." },
      ],
    },
    {
      name: "ListEvents",
      description:
        "Returns all registered event types, optionally filtered to active-only. Paginated.",
      request: [
        { name: "active_only", type: "bool", description: "Only return active event types.", example: true },
        { name: "pagination", type: "PaginationRequest", description: "Pagination parameters." },
      ],
      response: [
        { name: "events", type: "RegisteredEvent[]", description: "Event type definitions." },
        { name: "pagination", type: "PaginationResponse", description: "Pagination metadata." },
      ],
    },
    {
      name: "GetEvent",
      description:
        "Returns a single event type by name, including its schema and auto-generated sample payload.",
      request: [
        { name: "name", type: "string", required: true, description: "Event name to look up.", example: "order.created" },
      ],
      response: [
        { name: "event", type: "RegisteredEvent", description: "Event type definition with schema and sample_payload." },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Event name does not exist." },
      ],
    },
    {
      name: "UpdateEvent",
      description:
        "Modifies an existing event type's description, schema, metadata, or active flag. Only non-zero fields are applied.",
      request: [
        { name: "name", type: "string", required: true, description: "Event name to update.", example: "order.created" },
        { name: "description", type: "string", description: "Updated description.", example: "Updated: Fired when a new order is placed" },
        { name: "schema", type: "Struct (JSON)", description: "Updated JSON Schema." },
        { name: "metadata", type: "map<string, string>", description: "Updated metadata." },
        { name: "active", type: "bool", description: "Updated active flag." },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Event name does not exist." },
      ],
    },
    {
      name: "DeleteEvent",
      description:
        "Permanently removes an event type definition. Existing subscriptions referencing this event are not automatically deleted.",
      request: [
        { name: "name", type: "string", required: true, description: "Event name to delete.", example: "order.created" },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Event name does not exist." },
      ],
    },
    {
      name: "PushEvent",
      description:
        "Emits an event instance. The primary ingestion endpoint. On success, the event is persisted and deliveries are fanned out asynchronously to all matching subscriptions.",
      request: [
        { name: "namespace", type: "string", required: true, description: "Namespace to push the event into.", example: "production" },
        { name: "event", type: "string", required: true, description: "Event type name (must match a registered event).", example: "order.created" },
        { name: "payload", type: "Struct (JSON)", required: true, description: "Event payload. Validated against schema if one exists.", example: { order_id: "ord-123", amount: 99.99, currency: "USD" } },
        { name: "ttl_seconds", type: "int64", description: "TTL for delivery retries. 0 = no expiration." },
        { name: "metadata", type: "map<string, string>", description: "Metadata stored with event (not sent to webhooks)." },
        { name: "id", type: "string", description: "Client-provided event ID for idempotency." },
        { name: "labels", type: "map<string, string>", description: "Labels for subscription label-filter matching.", example: { region: "us-east", priority: "high" } },
      ],
      response: [
        { name: "event_id", type: "string", description: "Server-generated UUID for the event instance.", example: "e-550e8400-e29b-41d4-a716-446655440000" },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Event name is not registered." },
        { code: "INVALID_ARGUMENT", description: "Payload fails schema validation." },
      ],
    },
    {
      name: "ListEventReports",
      description:
        "Returns pushed event instances with aggregated delivery statistics (webhook_count, successful/failed/pending). Newest first.",
      request: [
        { name: "namespace", type: "string", required: true, description: "Namespace to list events from.", example: "production" },
        { name: "event_name", type: "string", description: "Filter to events of this type.", example: "order.created" },
        { name: "pagination", type: "PaginationRequest", description: "Pagination parameters. Max limit: 1000.", example: { limit: 25 } },
      ],
      response: [
        { name: "events", type: "EventReport[]", description: "Event instances with delivery statistics." },
        { name: "pagination", type: "PaginationResponse", description: "Pagination metadata." },
      ],
    },
  ],
};

export default service;
