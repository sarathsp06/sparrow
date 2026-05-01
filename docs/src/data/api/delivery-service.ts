import type { ApiService } from "./types";

const service: ApiService = {
  "service": "DeliveryService",
  "package": "webhook",
  "description": "DeliveryService provides read access to delivery history and manual retry control. Deliveries are created automatically when an event is pushed. Each delivery tracks its status (pending -> sending -> success/failed/retrying/expired), HTTP response code, response body (up to 1 KB by default, 1 MB if capture_response_body is enabled), error classification, and per-attempt history.",
  "rpcs": [
    {
      "name": "GetDeliveryStatus",
      "description": "GetDeliveryStatus returns full details of a single delivery, including request body, response body, response code, error category, and retry state. Errors: NOT_FOUND if the delivery_id does not exist in the given namespace.",
      "request": [
        {
          "name": "delivery_id",
          "type": "string",
          "description": "UUID of the delivery to retrieve. Required.",
          "example": "d-550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "namespace",
          "type": "string",
          "description": "Namespace the delivery belongs to. Required.",
          "example": "production"
        }
      ],
      "response": [
        {
          "name": "delivery",
          "type": "WebhookDelivery",
          "description": "Full delivery details including request/response bodies and error info."
        }
      ]
    },
    {
      "name": "ListDeliveries",
      "description": "ListDeliveries returns delivery records for a namespace, optionally filtered by webhook_id or event_id. Ordered by created_at descending. Paginated.",
      "request": [
        {
          "name": "namespace",
          "type": "string",
          "description": "Namespace to list deliveries from. Required.",
          "example": "production"
        },
        {
          "name": "webhook_id",
          "type": "string",
          "description": "Filter by webhook UUID. Optional.",
          "example": "550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "event_id",
          "type": "string",
          "description": "Filter by event UUID. Optional."
        },
        {
          "name": "pagination",
          "type": "PaginationRequest",
          "description": "Pagination parameters. Default: limit=50, offset=0.",
          "example": {
            "limit": 20
          }
        },
        {
          "name": "status",
          "type": "string",
          "description": "Filter by delivery status. Optional.",
          "example": "failed"
        },
        {
          "name": "error_category",
          "type": "string",
          "description": "Filter by error classification. Optional.",
          "example": "server_error"
        },
        {
          "name": "subscription_id",
          "type": "string",
          "description": "Filter by subscription UUID. Optional."
        },
        {
          "name": "created_after",
          "type": "Timestamp",
          "description": "Filter to deliveries created at or after this timestamp."
        },
        {
          "name": "created_before",
          "type": "Timestamp",
          "description": "Filter to deliveries created at or before this timestamp."
        },
        {
          "name": "prepare_retry",
          "type": "bool",
          "description": "When true, snapshot all matching delivery IDs (up to 10,000) into a batch job and return a retry_id in the response. Pass that ID to RetryDeliveries to retry the exact set that matched this query."
        }
      ],
      "response": [
        {
          "name": "deliveries",
          "type": "WebhookDelivery[]",
          "description": "Deliveries matching the filter criteria, newest first."
        },
        {
          "name": "pagination",
          "type": "PaginationResponse",
          "description": "Pagination metadata."
        },
        {
          "name": "retry_id",
          "type": "string",
          "description": "Batch ID for deterministic retry. Only populated when prepare_retry=true was set in the request. Pass to RetryDeliveries."
        }
      ]
    },
    {
      "name": "RetryDelivery",
      "description": "RetryDelivery re-enqueues deliveries for processing. Can target a specific delivery_id, all failed/pending deliveries for a webhook_id, or both. Set force=true to retry even successful deliveries. Returns the count and IDs of deliveries that were re-enqueued.",
      "request": [
        {
          "name": "namespace",
          "type": "string",
          "description": "Namespace. Required.",
          "example": "production"
        },
        {
          "name": "delivery_id",
          "type": "string",
          "description": "UUID of a specific delivery to retry. Optional.",
          "example": "d-550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "webhook_id",
          "type": "string",
          "description": "UUID of a webhook -- retry all its failed/pending deliveries. Optional."
        },
        {
          "name": "force",
          "type": "bool",
          "description": "When true, retry even successful deliveries. Default: false. Use with caution: this causes duplicate delivery to the target endpoint."
        }
      ],
      "response": [
        {
          "name": "retried_count",
          "type": "int32",
          "description": "Number of deliveries that were re-enqueued.",
          "example": 1
        },
        {
          "name": "delivery_ids",
          "type": "string[]",
          "description": "UUIDs of the deliveries that were re-enqueued.",
          "example": [
            "d-550e8400-e29b-41d4-a716-446655440000"
          ]
        }
      ]
    },
    {
      "name": "GetDeliveryAttempts",
      "description": "GetDeliveryAttempts returns the per-attempt history for a delivery, ordered by timestamp. Each attempt includes response_time, response_code, error_message, and error_category. Errors: INVALID_ARGUMENT if delivery_id is empty.",
      "request": [
        {
          "name": "delivery_id",
          "type": "string",
          "description": "UUID of the delivery to get attempts for. Required.",
          "example": "d-550e8400-e29b-41d4-a716-446655440000"
        }
      ],
      "response": [
        {
          "name": "attempts[].attempt_id",
          "type": "string",
          "description": "Server-generated UUID for this attempt.",
          "example": "a-110e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "attempts[].delivery_id",
          "type": "string",
          "description": "UUID of the parent delivery this attempt belongs to."
        },
        {
          "name": "attempts[].webhook_id",
          "type": "string",
          "description": "UUID of the target webhook."
        },
        {
          "name": "attempts[].success",
          "type": "bool",
          "description": "Whether this attempt resulted in a successful delivery (response code matched expected_status_codes).",
          "example": true
        },
        {
          "name": "attempts[].response_time",
          "type": "int32",
          "description": "Round-trip time in milliseconds from sending the request to receiving the full response (or timeout/error). 0 if no connection was established.",
          "example": 245
        },
        {
          "name": "attempts[].response_code",
          "type": "int32",
          "description": "HTTP response status code. 0 if no response was received (timeout, connection refused, DNS error, etc.).",
          "example": 200
        },
        {
          "name": "attempts[].error_message",
          "type": "string",
          "description": "Error message describing the failure. Empty string on success. Contains the raw error from the HTTP client (e.g., \"context deadline exceeded\", \"connection refused\", \"no such host\")."
        },
        {
          "name": "attempts[].error_category",
          "type": "string",
          "description": "Classified error category for this attempt. Values: \"success\", \"client_error\", \"server_error\", \"timeout\", \"connection_refused\", \"network_error\", \"dns_error\", \"tls_error\", \"unexpected_status\", \"unknown\".",
          "example": "success"
        },
        {
          "name": "attempts[].timestamp",
          "type": "Timestamp",
          "description": "When this attempt was made.",
          "example": "2025-01-15T10:30:05Z"
        }
      ]
    },
    {
      "name": "RetryDeliveries",
      "description": "RetryDeliveries executes a deterministic batch retry of deliveries whose IDs were previously snapshotted via ListDeliveries with prepare_retry=true. Each delivery is reset to pending and re-enqueued for HTTP delivery. The batch is processed asynchronously via a River job; poll GetRetryStatus for progress. Errors: NOT_FOUND if the retry_id does not exist or has expired. Errors: FAILED_PRECONDITION if the batch is not in 'pending' status.",
      "request": [
        {
          "name": "retry_id",
          "type": "string",
          "required": true,
          "description": "Batch ID returned by ListDeliveries when prepare_retry=true."
        }
      ],
      "response": [
        {
          "name": "retry_id",
          "type": "string",
          "description": "Batch ID for polling status."
        },
        {
          "name": "total",
          "type": "int32",
          "description": "Total number of deliveries that will be retried."
        },
        {
          "name": "status",
          "type": "string",
          "description": "Current status (will be \"processing\" on success)."
        }
      ]
    },
    {
      "name": "GetRetryStatus",
      "description": "GetRetryStatus returns the current progress of a batch retry operation. Errors: NOT_FOUND if the retry_id does not exist or has expired.",
      "request": [
        {
          "name": "retry_id",
          "type": "string",
          "required": true,
          "description": "Batch ID returned by RetryDeliveries or ListDeliveries."
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
      "name": "CancelRetry",
      "description": "CancelRetry aborts a batch retry that is pending or in progress. Items already processed are not rolled back. Errors: NOT_FOUND if the retry_id does not exist. Errors: FAILED_PRECONDITION if the batch is already completed or cancelled.",
      "request": [
        {
          "name": "retry_id",
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
