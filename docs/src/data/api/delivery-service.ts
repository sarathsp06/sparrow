import type { ApiService } from "./types";

const service: ApiService = {
  service: "DeliveryService",
  description:
    "The DeliveryService provides read access to delivery history and manual retry control. Deliveries are created automatically when an event is pushed. Each delivery tracks its status, HTTP response, error classification, and per-attempt history.",
  rpcs: [
    {
      name: "GetDeliveryStatus",
      description:
        "Returns full details of a single delivery, including request body, response body, response code, error category, and retry state.",
      request: [
        { name: "delivery_id", type: "string", required: true, description: "UUID of the delivery to retrieve.", example: "d-550e8400-e29b-41d4-a716-446655440000" },
        { name: "namespace", type: "string", required: true, description: "Namespace the delivery belongs to.", example: "production" },
      ],
      response: [
        { name: "delivery", type: "WebhookDelivery", description: "Full delivery details including request/response bodies and error info." },
      ],
      errors: [
        { code: "NOT_FOUND", description: "The delivery_id does not exist in the given namespace." },
      ],
    },
    {
      name: "ListDeliveries",
      description:
        "Returns delivery records for a namespace, optionally filtered by webhook_id or event_id. Ordered by created_at descending.",
      request: [
        { name: "namespace", type: "string", required: true, description: "Namespace to list deliveries from.", example: "production" },
        { name: "webhook_id", type: "string", required: false, description: "Filter by webhook UUID.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "event_id", type: "string", required: false, description: "Filter by event UUID." },
        { name: "pagination", type: "PaginationRequest", required: false, description: "Pagination parameters.", example: { limit: 20 } },
      ],
      response: [
        { name: "deliveries", type: "WebhookDelivery[]", description: "Deliveries matching filters, newest first." },
        { name: "pagination", type: "PaginationResponse", required: false, description: "Pagination parameters.", example: { limit: 20 } },
      ],
    },
    {
      name: "RetryDelivery",
      description:
        "Re-enqueues deliveries for processing. Can target a specific delivery, all failed deliveries for a webhook, or both.",
      request: [
        { name: "namespace", type: "string", required: true, description: "Namespace.", example: "production" },
        { name: "delivery_id", type: "string", required: false, description: "Specific delivery to retry.", example: "d-550e8400-e29b-41d4-a716-446655440000" },
        { name: "webhook_id", type: "string", required: false, description: "Retry all failed/pending deliveries for this webhook." },
        { name: "force", type: "bool", required: false, description: "Retry even successful deliveries (causes duplicates). Default: false." },
      ],
      response: [
        { name: "retried_count", type: "int32", description: "Number of deliveries re-enqueued.", example: 1 },
        { name: "delivery_ids", type: "string[]", description: "UUIDs of re-enqueued deliveries.", example: ["d-550e8400-e29b-41d4-a716-446655440000"] },
      ],
    },
    {
      name: "GetDeliveryAttempts",
      description:
        "Returns the per-attempt history for a delivery, ordered by timestamp. Each attempt includes response_time, response_code, error_message, and error_category.",
      request: [
        { name: "delivery_id", type: "string", required: true, description: "UUID of the delivery.", example: "d-550e8400-e29b-41d4-a716-446655440000" },
      ],
      response: [
        { name: "attempts[].attempt_id", type: "string", description: "UUID of this attempt.", example: "a-110e8400-e29b-41d4-a716-446655440000" },
        { name: "attempts[].delivery_id", type: "string", description: "UUID of the parent delivery this attempt belongs to." },
        { name: "attempts[].webhook_id", type: "string", description: "UUID of the target webhook." },
        { name: "attempts[].success", type: "bool", description: "Whether the attempt succeeded.", example: true },
        { name: "attempts[].response_time", type: "int32", description: "Round-trip time in milliseconds.", example: 245 },
        { name: "attempts[].response_code", type: "int32", description: "HTTP status code (0 if no response).", example: 200 },
        { name: "attempts[].error_message", type: "string", description: "Error description (empty on success)." },
        { name: "attempts[].error_category", type: "string", description: "Classified error category.", example: "success" },
        { name: "attempts[].timestamp", type: "Timestamp", description: "When this attempt was made.", example: "2025-01-15T10:30:05Z" },
      ],
      errors: [
        { code: "INVALID_ARGUMENT", description: "Delivery_id is empty." },
      ],
    },
  ],
  footer: `## Delivery Status Lifecycle

\`\`\`
PENDING -> SENDING -> SUCCESS
PENDING -> SENDING -> RETRYING -> SENDING -> ... -> SUCCESS | FAILED | EXPIRED
\`\`\`

| Status | Description |
|--------|-------------|
| \`PENDING\` | Queued, not yet attempted |
| \`SENDING\` | HTTP request in flight |
| \`SUCCESS\` | Endpoint returned an expected status code |
| \`FAILED\` | All retries exhausted or non-retryable error |
| \`RETRYING\` | Failed but will retry (server_error, timeout, connection_refused, network_error) |
| \`EXPIRED\` | TTL elapsed before success |`,
};

export default service;
