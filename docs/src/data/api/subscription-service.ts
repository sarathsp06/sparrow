import type { ApiService } from "./types";

const service: ApiService = {
  service: "SubscriptionService",
  description:
    "The SubscriptionService manages the mapping between webhooks and event types. A subscription links one webhook to one event name within a namespace. Subscriptions can optionally transform the payload using Go templates and filter events by labels.",
  rpcs: [
    {
      name: "CreateSubscription",
      description:
        "Links a webhook to an event name in a namespace. Optionally configure per-subscription headers, HTTP method override, timeout, payload transformation, and label filters.",
      request: [
        { name: "webhook_id", type: "string", required: true, description: "UUID of the webhook to subscribe.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "event_name", type: "string", required: true, description: "Event type name to subscribe to.", example: "order.created" },
        { name: "namespace", type: "string", required: true, description: "Namespace (must match webhook's namespace).", example: "production" },
        { name: "headers", type: "map<string, string>", description: "Per-subscription headers (override webhook-level)." },
        { name: "method", type: "string", description: "HTTP method override. Default: POST." },
        { name: "timeout", type: "int32", description: "Timeout override in seconds." },
        { name: "transform_enabled", type: "bool", description: "Enable Go template payload transformation." },
        { name: "transform_template", type: "string", description: "Go template string for transformation." },
        { name: "label_filters", type: "map<string, string>", description: "Label filters for selective event matching (AND logic).", example: { region: "us-east" } },
      ],
      response: [
        { name: "subscription_id", type: "string", description: "Server-generated UUID for the subscription.", example: "7c9e6679-7425-40de-944b-e07fc1f90ae7" },
        { name: "created_at", type: "Timestamp", description: "When the subscription was created.", example: "2025-01-15T10:30:00Z" },
      ],
      errors: [
        { code: "ALREADY_EXISTS", description: "Webhook is already subscribed to this event in this namespace." },
        { code: "NOT_FOUND", description: "Webhook does not exist." },
      ],
    },
    {
      name: "GetSubscription",
      description: "Returns a single subscription by ID.",
      request: [
        { name: "subscription_id", type: "string", required: true, description: "UUID of the subscription.", example: "7c9e6679-7425-40de-944b-e07fc1f90ae7" },
        { name: "namespace", type: "string", required: true, description: "Namespace the subscription belongs to.", example: "production" },
      ],
      response: [
        { name: "subscription", type: "EventSubscription", description: "Full subscription details." },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Subscription does not exist in the given namespace." },
      ],
    },
    {
      name: "ListSubscriptions",
      description:
        "Returns subscriptions in a namespace, optionally filtered by webhook_id or event_name. Paginated.",
      request: [
        { name: "namespace", type: "string", required: true, description: "Namespace to list from.", example: "production" },
        { name: "webhook_id", type: "string", description: "Filter by webhook UUID.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "event_name", type: "string", description: "Filter by event type name." },
        { name: "pagination", type: "PaginationRequest", description: "Pagination parameters." },
      ],
      response: [
        { name: "subscriptions", type: "EventSubscription[]", description: "Matching subscriptions." },
        { name: "pagination", type: "PaginationResponse", description: "Pagination metadata." },
      ],
    },
    {
      name: "UpdateSubscription",
      description:
        "Modifies a subscription's headers, method, timeout, transform settings, or label filters. Only non-zero fields are applied.",
      request: [
        { name: "subscription_id", type: "string", required: true, description: "UUID of the subscription to update.", example: "7c9e6679-7425-40de-944b-e07fc1f90ae7" },
        { name: "namespace", type: "string", required: true, description: "Namespace the subscription belongs to.", example: "production" },
        { name: "headers", type: "map<string, string>", description: "Updated per-subscription headers." },
        { name: "method", type: "string", description: "Updated HTTP method." },
        { name: "timeout", type: "int32", description: "Updated timeout in seconds." },
        { name: "transform_enabled", type: "bool", description: "Enable/disable transformation.", example: true },
        { name: "transform_template", type: "string", description: "Updated Go template." },
        { name: "label_filters", type: "map<string, string>", description: "Updated label filters." },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Subscription does not exist." },
      ],
    },
    {
      name: "DeleteSubscription",
      description:
        "Permanently removes a subscription. In-flight deliveries are not cancelled.",
      request: [
        { name: "subscription_id", type: "string", required: true, description: "UUID of the subscription to delete.", example: "7c9e6679-7425-40de-944b-e07fc1f90ae7" },
        { name: "namespace", type: "string", required: true, description: "Namespace the subscription belongs to.", example: "production" },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Subscription does not exist." },
      ],
    },
    {
      name: "TestSubscriptionTemplate",
      description:
        "Renders a Go template against the sample payload of the given event type. Use this to validate templates before saving.",
      request: [
        { name: "event_name", type: "string", required: true, description: "Event name to get the sample payload from.", example: "order.created" },
        { name: "transform_template", type: "string", required: true, description: "Go template to test.", example: "{\"id\": \"{{ .payload.order_id }}\", \"total\": \"{{ .payload.amount }}\"}" },
        { name: "namespace", type: "string", required: true, description: "Namespace.", example: "production" },
      ],
      response: [
        { name: "transformed_payload", type: "string", description: "The rendered template output.", example: "{\"id\": \"ord-123\", \"total\": \"99.99\"}" },
      ],
      errors: [
        { code: "INVALID_ARGUMENT", description: "Template fails to parse or execute." },
        { code: "NOT_FOUND", description: "Event name does not exist." },
      ],
    },
  ],
};

export default service;
