import type { ApiService } from "./types";

const service: ApiService = {
  service: "HealthService",
  description:
    "The HealthService provides webhook health metrics derived from delivery outcomes. Health is computed per-webhook based on success rate and consecutive failures.",
  notes: `### Health Status Thresholds

| Status | Success Rate | Consecutive Failures |
|--------|-------------|---------------------|
| **HEALTHY** | > 90% | < 3 |
| **DEGRADED** | 50-90% | 3-9 |
| **UNHEALTHY** | < 50% | >= 10 |`,
  rpcs: [
    {
      name: "GetWebhookHealth",
      description:
        "Returns health status and detailed metrics for a single webhook: total/successful/failed deliveries, consecutive failures, success rate, avg response time, and error category breakdown.",
      request: [
        { name: "webhook_id", type: "string", required: true, description: "UUID of the webhook.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "namespace", type: "string", required: true, description: "Namespace the webhook belongs to.", example: "production" },
      ],
      response: [
        { name: "webhook_id", type: "string", description: "UUID of the webhook.", example: "550e8400-e29b-41d4-a716-446655440000" },
        { name: "health", type: "WebhookHealth", description: "Computed health status (HEALTHY, DEGRADED, UNHEALTHY, UNSPECIFIED).", example: "HEALTHY" },
        { name: "metrics.webhook_id", type: "string", description: "UUID of the webhook these metrics belong to." },
        { name: "metrics.total_deliveries", type: "int32", description: "Total deliveries (all time).", example: 1520 },
        { name: "metrics.successful_deliveries", type: "int32", description: "Successful deliveries.", example: 1480 },
        { name: "metrics.failed_deliveries", type: "int32", description: "Failed deliveries.", example: 40 },
        { name: "metrics.consecutive_failures", type: "int32", description: "Current consecutive failure streak.", example: 0 },
        { name: "metrics.last_success_at", type: "Timestamp", description: "Timestamp of the most recent successful delivery. Null if the webhook has never succeeded." },
        { name: "metrics.last_failure_at", type: "Timestamp", description: "Timestamp of the most recent failed delivery. Null if the webhook has never failed." },
        { name: "metrics.success_rate", type: "double", description: "Success rate (0.0 to 1.0).", example: 0.974 },
        { name: "metrics.avg_response_time", type: "int32", description: "Average response time in ms.", example: 245 },
        { name: "metrics.created_at", type: "Timestamp", description: "When the health metrics record was first created." },
        { name: "metrics.updated_at", type: "Timestamp", description: "When the health metrics were last updated." },
        { name: "metrics.client_errors", type: "int32", description: "HTTP 4xx errors (last 24h).", example: 2 },
        { name: "metrics.server_errors", type: "int32", description: "HTTP 5xx errors (last 24h).", example: 5 },
        { name: "metrics.timeout_errors", type: "int32", description: "Timeout errors (last 24h).", example: 1 },
        { name: "metrics.network_errors", type: "int32", description: "Network-level errors (last 24h).", example: 0 },
      ],
    },
    {
      name: "ListWebhooksByHealth",
      description:
        "Returns all webhooks matching a given health status. Useful for finding degraded or unhealthy endpoints.",
      request: [
        { name: "health", type: "WebhookHealth", required: true, description: "Health status to filter by (HEALTHY, DEGRADED, UNHEALTHY).", example: "HEALTH_UNHEALTHY" },
        { name: "pagination", type: "PaginationRequest", required: false, description: "Pagination parameters." },
      ],
      response: [
        { name: "webhooks", type: "RegisteredWebhook[]", description: "Webhooks with the requested health status." },
        { name: "pagination", type: "PaginationResponse", required: false, description: "Pagination parameters." },
      ],
    },
    {
      name: "GetHealthSummary",
      description:
        "Returns aggregate counts of webhooks by health status (healthy, degraded, unhealthy, unknown) across all namespaces.",
      request: [
      ],
      response: [
        { name: "summary.healthy_count", type: "int32", description: "Webhooks in HEALTHY state.", example: 45 },
        { name: "summary.degraded_count", type: "int32", description: "Webhooks in DEGRADED state.", example: 3 },
        { name: "summary.unhealthy_count", type: "int32", description: "Webhooks in UNHEALTHY state.", example: 1 },
        { name: "summary.unknown_count", type: "int32", description: "Webhooks with no delivery history.", example: 2 },
        { name: "summary.total_count", type: "int32", description: "Total webhooks.", example: 51 },
      ],
    },
  ],
};

export default service;
