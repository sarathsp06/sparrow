import type { ApiEnum } from "./types";

const enumData: ApiEnum = {
  "name": "WebhookHealth",
  "package": "webhook",
  "description": "WebhookHealth represents the computed health status of a webhook endpoint. Health is derived from recent delivery outcomes using two signals: - Success rate: successful deliveries / total deliveries - Consecutive failures: unbroken streak of failed deliveries Thresholds: HEALTHY:   success_rate \u003e 90% AND consecutive_failures \u003c 3 DEGRADED:  success_rate 50-90% OR consecutive_failures 3-9 UNHEALTHY: success_rate \u003c 50% OR consecutive_failures \u003e= 10",
  "values": [
    {
      "name": "HEALTH_UNSPECIFIED",
      "number": 0,
      "description": "No delivery data available. The webhook has never received a delivery, or health has not been computed yet."
    },
    {
      "name": "HEALTH_HEALTHY",
      "number": 1,
      "description": "Webhook is healthy: \u003e90% success rate and fewer than 3 consecutive failures."
    },
    {
      "name": "HEALTH_DEGRADED",
      "number": 2,
      "description": "Webhook is degraded: 50-90% success rate or 3-9 consecutive failures. May indicate intermittent issues at the target endpoint."
    },
    {
      "name": "HEALTH_UNHEALTHY",
      "number": 3,
      "description": "Webhook is unhealthy: \u003c50% success rate or 10+ consecutive failures. Investigate the target endpoint -- deliveries are still attempted but mostly failing."
    }
  ],
  "usedBy": [
    "HealthService"
  ]
};

export default enumData;
