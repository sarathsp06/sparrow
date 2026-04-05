import type { ApiEnum } from "./types";

const enumData: ApiEnum = {
  "name": "WebhookDeliveryStatus",
  "package": "webhook",
  "description": "WebhookDeliveryStatus tracks the lifecycle of a single delivery. State transitions: PENDING -\u003e SENDING -\u003e SUCCESS PENDING -\u003e SENDING -\u003e RETRYING -\u003e SENDING -\u003e ... -\u003e SUCCESS | FAILED | EXPIRED Terminal states: SUCCESS, FAILED, EXPIRED.",
  "values": [
    {
      "name": "DELIVERY_UNSPECIFIED",
      "number": 0,
      "description": "Default zero value. Never set by the system; indicates an uninitialized field."
    },
    {
      "name": "DELIVERY_PENDING",
      "number": 1,
      "description": "Delivery is queued but has not been attempted yet."
    },
    {
      "name": "DELIVERY_SENDING",
      "number": 2,
      "description": "Delivery is currently being sent (HTTP request in flight)."
    },
    {
      "name": "DELIVERY_SUCCESS",
      "number": 3,
      "description": "Delivery succeeded. The webhook endpoint returned an expected status code (default: 200, 201, 202, 204 -- configurable via expected_status_codes)."
    },
    {
      "name": "DELIVERY_FAILED",
      "number": 4,
      "description": "Delivery failed permanently. All retry attempts exhausted or the error is classified as non-retryable (client_error, dns_error, tls_error)."
    },
    {
      "name": "DELIVERY_RETRYING",
      "number": 5,
      "description": "Delivery failed but will be retried. Applies to retryable error categories: server_error (5xx), timeout, connection_refused, network_error. The next_retry_at field indicates when the next attempt is scheduled."
    },
    {
      "name": "DELIVERY_EXPIRED",
      "number": 6,
      "description": "Delivery TTL elapsed before it could succeed. The event's ttl_seconds determines the expiration window. No further retries will be attempted."
    }
  ]
};

export default enumData;
