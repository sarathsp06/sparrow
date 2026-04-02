import type { ApiService } from "./types";

const service: ApiService = {
  service: "WebhookService",
  description:
    "The WebhookService manages webhook registration, configuration, and lifecycle. All RPCs require a namespace. Webhooks are scoped to a single namespace.",
  rpcs: [
    {
      name: "RegisterWebhook",
      description:
        "Creates a new webhook registration. Subscriptions are created automatically for each event in the events list.",
      request: [
        { name: "namespace", type: "string", required: true, description: "Namespace to register the webhook in. Must already exist.", example: "production" },
        { name: "url", type: "string", required: true, description: "Target URL for webhook deliveries (HTTP or HTTPS).", example: "https://example.com/hooks" },
        { name: "events", type: "string[]", required: true, description: "Event names this webhook should receive. At least one required.", example: ["order.created", "order.updated"] },
        { name: "headers", type: "map<string, string>", description: "Custom HTTP headers included in every delivery request.", example: { "X-Source": "sparrow" } },
        { name: "secret_headers", type: "map<string, string>", description: "Sensitive headers stored encrypted, masked in API responses." },
        { name: "description", type: "string", description: "Human-readable description.", example: "Order notifications" },
        { name: "active", type: "bool", description: "Whether the webhook starts active. Default: true.", example: true },
        { name: "http_config", type: "WebhookHTTPConfig", description: "HTTP delivery configuration (retries, timeouts, TLS, HMAC).", example: { max_retries: 5, webhook_secret: "whsec_your_secret_key" } },
      ],
      response: [
        { name: "webhook_id", type: "string", description: "Server-generated UUID for the new webhook.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "created_at", type: "Timestamp", description: "When the webhook was created.", example: "2025-01-15T10:30:00Z" },
      ],
      errors: [
        { code: "ALREADY_EXISTS", description: "URL is already registered in the same namespace." },
      ],
    },
    {
      name: "UnregisterWebhook",
      description:
        "Permanently deletes a webhook and all its subscriptions. Associated delivery records are cascade-deleted.",
      request: [
        { name: "webhook_id", type: "string", required: true, description: "UUID of the webhook to delete.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "namespace", type: "string", required: true, description: "Namespace the webhook belongs to.", example: "production" },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Webhook does not exist in the given namespace." },
      ],
    },
    {
      name: "ListWebhooks",
      description:
        "Returns webhooks for a namespace with optional filters. Each webhook includes its current health status.",
      request: [
        { name: "namespace", type: "string", required: true, description: "Namespace to list webhooks from.", example: "production" },
        { name: "event", type: "string", description: "Filter to webhooks subscribed to this event." },
        { name: "active_only", type: "bool", description: "Only return active webhooks.", example: true },
        { name: "webhook_id", type: "string", description: "Filter to a specific webhook by ID." },
        { name: "pagination", type: "PaginationRequest", description: "Pagination parameters (limit, offset).", example: { limit: 20 } },
      ],
      response: [
        { name: "webhooks", type: "RegisteredWebhook[]", description: "List of webhooks matching the filters." },
        { name: "pagination", type: "PaginationResponse", description: "Pagination metadata (total_count, limit, offset)." },
      ],
    },
    {
      name: "UpdateWebhookConfig",
      description:
        "Patches one or more fields on an existing webhook. Only non-zero fields are applied. Updating events replaces the full subscription set.",
      request: [
        { name: "webhook_id", type: "string", required: true, description: "UUID of the webhook to update.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "namespace", type: "string", required: true, description: "Namespace the webhook belongs to.", example: "production" },
        { name: "updates.url", type: "string", description: "New target URL." },
        { name: "updates.events", type: "string[]", description: "Replace all subscribed events." },
        { name: "updates.headers", type: "map<string, string>", description: "Replace all custom headers." },
        { name: "updates.secret_headers", type: "map<string, string>", description: "Replace all secret headers." },
        { name: "updates.description", type: "string", description: "Updated description.", example: "Updated order webhook" },
        { name: "updates.http_config", type: "WebhookHTTPConfig", description: "Updated HTTP config (replaces entire config).", example: { max_retries: 5, request_timeout_seconds: 60 } },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Webhook does not exist." },
      ],
    },
    {
      name: "PauseWebhook",
      description:
        "Sets a webhook to inactive. Paused webhooks are skipped during event fan-out. Existing in-flight deliveries are not cancelled.",
      request: [
        { name: "webhook_id", type: "string", required: true, description: "UUID of the webhook to pause.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "namespace", type: "string", required: true, description: "Namespace the webhook belongs to.", example: "production" },
        { name: "reason", type: "string", description: "Human-readable reason for pausing.", example: "Endpoint maintenance" },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Webhook does not exist." },
      ],
    },
    {
      name: "ResumeWebhook",
      description:
        "Re-activates a paused webhook. Events pushed while paused are not retroactively delivered.",
      request: [
        { name: "webhook_id", type: "string", required: true, description: "UUID of the webhook to resume.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "namespace", type: "string", required: true, description: "Namespace the webhook belongs to.", example: "production" },
        { name: "reason", type: "string", description: "Human-readable reason for resuming." },
      ],
      errors: [
        { code: "NOT_FOUND", description: "Webhook does not exist." },
      ],
    },
    {
      name: "GetNamespaceStats",
      description:
        "Returns aggregate delivery statistics for a namespace: total/active webhooks, delivery counts, and success rate.",
      request: [
        { name: "namespace", type: "string", required: true, description: "Namespace to get statistics for.", example: "production" },
      ],
      response: [
        { name: "namespace", type: "string", description: "The namespace.", example: "production" },
        { name: "stats.total_webhooks", type: "int32", description: "Total registered webhooks (active + inactive).", example: 12 },
        { name: "stats.active_webhooks", type: "int32", description: "Currently active webhooks.", example: 10 },
        { name: "stats.total_deliveries", type: "int32", description: "Total deliveries ever created.", example: 5420 },
        { name: "stats.successful_deliveries", type: "int32", description: "Deliveries in SUCCESS state.", example: 5200 },
        { name: "stats.failed_deliveries", type: "int32", description: "Deliveries in FAILED state.", example: 120 },
        { name: "stats.pending_deliveries", type: "int32", description: "Deliveries in PENDING/SENDING/RETRYING state.", example: 100 },
        { name: "stats.success_rate", type: "double", description: "Success rate (0.0 to 1.0).", example: 0.96 },
      ],
    },
    {
      name: "GetTemplateFunctions",
      description:
        "Returns the list of Go template functions available for payload transformation in subscriptions.",
      request: [],
      response: [
        { name: "functions", type: "TemplateFunction[]", description: "Available template functions with names and descriptions." },
        { name: "total_count", type: "int32", description: "Total number of available functions.", example: 42 },
      ],
    },
  ],
  footer: `## WebhookHTTPConfig

Shared configuration object used in \`RegisterWebhook\` and \`UpdateWebhookConfig\`:

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| \`max_retries\` | \`int32\` | \`3\` | Retry attempts after failure (0-10). Only retryable errors trigger retries. |
| \`retry_backoff_seconds\` | \`int32\` | \`60\` | Base backoff between retries. Exponential with jitter. |
| \`capture_response_body\` | \`bool\` | \`false\` | Store up to 1 MB of response body (vs 1 KB default). |
| \`follow_redirects\` | \`bool\` | \`true\` | Follow HTTP 3xx redirects. |
| \`verify_ssl\` | \`bool\` | \`true\` | Verify TLS certificate chain. |
| \`request_timeout_seconds\` | \`int32\` | \`30\` | Per-request timeout (1-300 seconds). |
| \`expected_status_codes\` | \`int32[]\` | \`[200, 201, 202, 204]\` | Status codes treated as success. |
| \`webhook_secret\` | \`string\` | \`""\` | HMAC-SHA256 signing secret. |
| \`user_agent\` | \`string\` | \`"Sparrow-Webhook/1.0"\` | User-Agent header value. |
| \`content_type\` | \`string\` | \`"application/json"\` | Content-Type header value. |`,
};

export default service;
