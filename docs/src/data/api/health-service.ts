import type { ApiService } from "./types";

const service: ApiService = {
  service: "HealthService",
  description:
    "HealthService provides webhook health metrics derived from delivery outcomes. Health is computed per-webhook based on success rate and consecutive failures: HEALTHY (>90% success, <3 consecutive failures), DEGRADED (50-90% or 3-9 consecutive), UNHEALTHY (<50% or 10+ consecutive failures).",
  rpcs: [
    {
      name: "GetWebhookHealth",
      description:
        "GetWebhookHealth returns health status and detailed metrics for a single webhook: total/successful/failed deliveries, consecutive failures, success rate, avg response time, and error category breakdown (client/server/timeout/network errors). Returns HEALTH_UNSPECIFIED with no metrics if the webhook has no delivery history.",
      request: [
        { name: "webhook_id", type: "string", required: true, description: "UUID of the webhook. Required.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "namespace", type: "string", required: true, description: "Namespace the webhook belongs to. Required.", example: "production" },
      ],
      response: [
        { name: "webhook_id", type: "string", description: "UUID of the webhook.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "health", type: "WebhookHealth", description: "Computed health status based on success_rate and consecutive_failures. Returns HEALTH_UNSPECIFIED if the webhook has no delivery history.", example: "HEALTHY" },
        { name: "metrics.webhook_id", type: "string", description: "UUID of the webhook these metrics belong to." },
        { name: "metrics.total_deliveries", type: "int32", description: "Total number of deliveries ever made to this webhook (all time).", example: 1520 },
        { name: "metrics.successful_deliveries", type: "int32", description: "Number of deliveries that succeeded (terminal SUCCESS status).", example: 1480 },
        { name: "metrics.failed_deliveries", type: "int32", description: "Number of deliveries that failed permanently (terminal FAILED status).", example: 40 },
        { name: "metrics.consecutive_failures", type: "int32", description: "Current streak of consecutive failed deliveries. Resets to 0 on any success. Used for health status computation: 3-9 = DEGRADED, 10+ = UNHEALTHY.", example: 0 },
        { name: "metrics.last_success_at", type: "Timestamp", description: "Timestamp of the most recent successful delivery. Null if the webhook has never succeeded." },
        { name: "metrics.last_failure_at", type: "Timestamp", description: "Timestamp of the most recent failed delivery. Null if the webhook has never failed." },
        { name: "metrics.success_rate", type: "double", description: "Success rate as a decimal between 0.0 and 1.0. Computed as successful_deliveries / total_deliveries. Used for health thresholds: >0.9 = HEALTHY, 0.5-0.9 = DEGRADED, <0.5 = UNHEALTHY.", example: 0.974 },
        { name: "metrics.avg_response_time", type: "int32", description: "Average response time in milliseconds across all delivery attempts.", example: 245 },
        { name: "metrics.created_at", type: "Timestamp", description: "When the health metrics record was first created." },
        { name: "metrics.updated_at", type: "Timestamp", description: "When the health metrics were last updated." },
        { name: "metrics.client_errors", type: "int32", description: "Count of client errors (HTTP 4xx) in the last 24 hours. Client errors are never retried -- typically indicates a misconfigured endpoint (wrong URL, missing auth, payload format mismatch).", example: 2 },
        { name: "metrics.server_errors", type: "int32", description: "Count of server errors (HTTP 5xx) in the last 24 hours. Server errors are retried according to the webhook's retry configuration.", example: 5 },
        { name: "metrics.timeout_errors", type: "int32", description: "Count of timeout errors in the last 24 hours. Timeouts are retried. May indicate the endpoint is slow or overloaded.", example: 1 },
        { name: "metrics.network_errors", type: "int32", description: "Count of network-level errors in the last 24 hours. Includes DNS failures, TLS errors, connection refused, and other transport errors.", example: 0 },
      ],
    },
    {
      name: "ListWebhooksByHealth",
      description:
        "ListWebhooksByHealth returns all webhooks matching a given health status. Useful for finding degraded or unhealthy endpoints. Paginated.",
      request: [
        { name: "health", type: "WebhookHealth", required: true, description: "Health status to filter by. Required. Use HEALTH_UNHEALTHY to find problematic endpoints, HEALTH_DEGRADED for early warnings.", example: "HEALTH_UNHEALTHY" },
        { name: "pagination", type: "PaginationRequest", description: "Pagination parameters. Default: limit=50, offset=0." },
      ],
      response: [
        { name: "webhooks", type: "RegisteredWebhook[]", description: "Webhooks with the requested health status." },
        { name: "pagination", type: "PaginationResponse", description: "Pagination metadata." },
      ],
    },
    {
      name: "GetHealthSummary",
      description:
        "GetHealthSummary returns aggregate counts of webhooks by health status (healthy, degraded, unhealthy, unknown) across all namespaces.",
      request: [
      ],
      response: [
        { name: "summary.healthy_count", type: "int32", description: "Number of webhooks in HEALTHY state (>90% success rate, <3 consecutive failures).", example: 45 },
        { name: "summary.degraded_count", type: "int32", description: "Number of webhooks in DEGRADED state (50-90% success rate or 3-9 consecutive failures).", example: 3 },
        { name: "summary.unhealthy_count", type: "int32", description: "Number of webhooks in UNHEALTHY state (<50% success rate or 10+ consecutive failures).", example: 1 },
        { name: "summary.unknown_count", type: "int32", description: "Number of webhooks with no delivery history (HEALTH_UNSPECIFIED).", example: 2 },
        { name: "summary.total_count", type: "int32", description: "Total number of webhooks (sum of all categories above).", example: 51 },
      ],
    },
  ],
};

export default service;
