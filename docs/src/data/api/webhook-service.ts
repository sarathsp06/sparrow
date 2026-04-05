import type { ApiService } from "./types";

const service: ApiService = {
  "service": "WebhookService",
  "package": "webhook",
  "description": "WebhookService manages webhook registration, configuration, and lifecycle. All RPCs require a namespace. Webhooks are scoped to a single namespace.",
  "rpcs": [
    {
      "name": "RegisterWebhook",
      "description": "RegisterWebhook creates a new webhook registration. Accepts a target URL, a list of event names, and optional HTTP configuration. Subscriptions are created automatically for each event in the events list. Returns the generated webhook_id and created_at timestamp.",
      "request": [
        {
          "name": "namespace",
          "type": "string",
          "required": true,
          "description": "Namespace to register the webhook in. Required. The namespace must already exist (created via the NamespaceService).",
          "example": "production"
        },
        {
          "name": "events",
          "type": "string[]",
          "description": "Event names this webhook should receive. At least one is required. A subscription is automatically created for each event name listed here.",
          "example": [
            "order.created",
            "order.updated"
          ]
        },
        {
          "name": "url",
          "type": "string",
          "required": true,
          "description": "Target URL that will receive HTTP POST requests for each delivery. Must be a valid HTTP or HTTPS URL. Required.",
          "example": "https://example.com/hooks"
        },
        {
          "name": "headers",
          "type": "map<string, string>",
          "description": "Custom HTTP headers included in every delivery request to this webhook. These are visible in API responses. For sensitive values (API keys, tokens), use secret_headers instead.",
          "example": {
            "X-Source": "sparrow"
          }
        },
        {
          "name": "active",
          "type": "bool",
          "description": "Whether the webhook starts in active state.  Inactive webhooks are skipped during event fan-out.",
          "example": true
        },
        {
          "name": "description",
          "type": "string",
          "description": "Human-readable description of the webhook's purpose. Optional.",
          "example": "Order notifications"
        },
        {
          "name": "http_config",
          "type": "WebhookHTTPConfig",
          "description": "HTTP delivery configuration (retries, timeouts, TLS, HMAC, etc.). Optional -- all fields have sensible defaults if omitted.",
          "example": {
            "max_retries": 5,
            "webhook_secret": "whsec_your_secret_key"
          }
        },
        {
          "name": "secret_headers",
          "type": "map<string, string>",
          "description": "Sensitive HTTP headers stored encrypted at rest (e.g., Authorization, API keys). Values are masked as \"******\" in all API responses -- only keys are visible. Included in delivery requests alongside regular headers."
        }
      ],
      "response": [
        {
          "name": "webhook_id",
          "type": "string",
          "description": "Server-generated UUID for the new webhook. Use this ID for all subsequent operations (update, pause, resume, unregister).",
          "example": "550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "created_at",
          "type": "Timestamp",
          "description": "Timestamp when the webhook was created.",
          "example": "2025-01-15T10:30:00Z"
        }
      ],
      "errors": [
        {
          "code": "ALREADY_EXISTS",
          "description": "The URL is already registered in the same namespace."
        }
      ]
    },
    {
      "name": "UnregisterWebhook",
      "description": "UnregisterWebhook permanently deletes a webhook and all its subscriptions. Associated delivery records are also cascade-deleted.",
      "request": [
        {
          "name": "webhook_id",
          "type": "string",
          "required": true,
          "description": "UUID of the webhook to delete. Required.",
          "example": "550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "namespace",
          "type": "string",
          "required": true,
          "description": "Namespace the webhook belongs to. Required.",
          "example": "production"
        }
      ],
      "errors": [
        {
          "code": "NOT_FOUND",
          "description": "The webhook_id does not exist in the given namespace."
        }
      ]
    },
    {
      "name": "ListWebhooks",
      "description": "ListWebhooks returns webhooks for a namespace with optional filters. Supports filtering by event name, active status, or specific webhook_id. Results are paginated (default limit: 50). Each webhook includes its current health status.",
      "request": [
        {
          "name": "namespace",
          "type": "string",
          "required": true,
          "description": "Namespace to list webhooks from. Required.",
          "example": "production"
        },
        {
          "name": "event",
          "type": "string",
          "description": "Filter to only webhooks subscribed to this event name. Optional."
        },
        {
          "name": "active_only",
          "type": "bool",
          "description": "When true, only return active (non-paused) webhooks.  (return all).",
          "example": true
        },
        {
          "name": "pagination",
          "type": "PaginationRequest",
          "description": "Pagination parameters.  offset=0.",
          "example": {
            "limit": 20
          }
        },
        {
          "name": "webhook_id",
          "type": "string",
          "description": "Filter to a specific webhook by ID. Optional. When set, returns at most one result."
        }
      ],
      "response": [
        {
          "name": "webhooks",
          "type": "RegisteredWebhook[]",
          "description": "Webhooks matching the filter criteria."
        },
        {
          "name": "pagination",
          "type": "PaginationResponse",
          "description": "Pagination metadata (total_count, limit, offset)."
        }
      ]
    },
    {
      "name": "UpdateWebhookConfig",
      "description": "UpdateWebhookConfig patches one or more fields on an existing webhook. Only non-zero fields in WebhookUpdateFields are applied. Updating events replaces the full subscription set: removed events have their subscriptions deleted, new events get subscriptions created.",
      "request": [
        {
          "name": "webhook_id",
          "type": "string",
          "required": true,
          "description": "UUID of the webhook to update. Required.",
          "example": "550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "namespace",
          "type": "string",
          "required": true,
          "description": "Namespace the webhook belongs to. Required.",
          "example": "production"
        },
        {
          "name": "updates.events",
          "type": "string[]",
          "description": "Replace the full list of subscribed events. Omit to leave unchanged. Passing a non-empty list replaces all existing subscriptions."
        },
        {
          "name": "updates.url",
          "type": "string",
          "description": "New target URL. Omit to leave unchanged."
        },
        {
          "name": "updates.headers",
          "type": "map<string, string>",
          "description": "Replace all custom headers. Omit to leave unchanged. Pass an empty map to clear all headers."
        },
        {
          "name": "updates.active",
          "type": "bool",
          "description": "Set active/inactive status. Note: prefer PauseWebhook/ResumeWebhook RPCs for explicit lifecycle control with reason tracking."
        },
        {
          "name": "updates.description",
          "type": "string",
          "description": "Updated human-readable description. Omit to leave unchanged.",
          "example": "Updated order webhook"
        },
        {
          "name": "updates.http_config",
          "type": "WebhookHTTPConfig",
          "description": "Updated HTTP delivery configuration. Omit to leave unchanged. When set, the entire http_config is replaced (not merged field-by-field).",
          "example": {
            "max_retries": 5,
            "request_timeout_seconds": 60
          }
        },
        {
          "name": "updates.secret_headers",
          "type": "map<string, string>",
          "description": "Replace all secret headers. Omit to leave unchanged. Pass an empty map to clear all secret headers."
        }
      ],
      "errors": [
        {
          "code": "NOT_FOUND",
          "description": "The webhook does not exist."
        }
      ]
    },
    {
      "name": "PauseWebhook",
      "description": "PauseWebhook sets a webhook to inactive. Paused webhooks are skipped during event fan-out -- no new deliveries are created for them. Existing in-flight deliveries are not cancelled.",
      "request": [
        {
          "name": "webhook_id",
          "type": "string",
          "required": true,
          "description": "UUID of the webhook to pause. Required.",
          "example": "550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "namespace",
          "type": "string",
          "required": true,
          "description": "Namespace the webhook belongs to. Required.",
          "example": "production"
        },
        {
          "name": "reason",
          "type": "string",
          "description": "Human-readable reason for pausing (stored for audit purposes). Optional.",
          "example": "Endpoint maintenance"
        }
      ],
      "errors": [
        {
          "code": "NOT_FOUND",
          "description": "The webhook does not exist."
        }
      ]
    },
    {
      "name": "ResumeWebhook",
      "description": "ResumeWebhook re-activates a paused webhook. Future events will again create deliveries for this webhook. Events pushed while the webhook was paused are not retroactively delivered.",
      "request": [
        {
          "name": "webhook_id",
          "type": "string",
          "required": true,
          "description": "UUID of the webhook to resume. Required.",
          "example": "550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "namespace",
          "type": "string",
          "required": true,
          "description": "Namespace the webhook belongs to. Required.",
          "example": "production"
        },
        {
          "name": "reason",
          "type": "string",
          "description": "Human-readable reason for resuming. Optional."
        }
      ],
      "errors": [
        {
          "code": "NOT_FOUND",
          "description": "The webhook does not exist."
        }
      ]
    },
    {
      "name": "GetNamespaceStats",
      "description": "GetNamespaceStats returns aggregate delivery statistics for a namespace: total/active webhooks, total/successful/failed/pending deliveries, and success rate.",
      "request": [
        {
          "name": "namespace",
          "type": "string",
          "required": true,
          "description": "Namespace to get statistics for. Required.",
          "example": "production"
        }
      ],
      "response": [
        {
          "name": "namespace",
          "type": "string",
          "description": "The namespace these stats apply to."
        },
        {
          "name": "stats.total_webhooks",
          "type": "int32",
          "description": "Total number of registered webhooks in this namespace (active + inactive).",
          "example": 12
        },
        {
          "name": "stats.active_webhooks",
          "type": "int32",
          "description": "Number of currently active (non-paused) webhooks.",
          "example": 10
        },
        {
          "name": "stats.total_deliveries",
          "type": "int32",
          "description": "Total number of deliveries ever created in this namespace.",
          "example": 5420
        },
        {
          "name": "stats.successful_deliveries",
          "type": "int32",
          "description": "Number of deliveries in terminal SUCCESS state.",
          "example": 5200
        },
        {
          "name": "stats.failed_deliveries",
          "type": "int32",
          "description": "Number of deliveries in terminal FAILED state.",
          "example": 120
        },
        {
          "name": "stats.pending_deliveries",
          "type": "int32",
          "description": "Number of deliveries in PENDING, SENDING, or RETRYING state.",
          "example": 100
        },
        {
          "name": "stats.success_rate",
          "type": "double",
          "description": "Overall delivery success rate as a decimal between 0.0 and 1.0. Computed as successful_deliveries / total_deliveries. 0.0 if no deliveries.",
          "example": 0.96
        }
      ]
    },
    {
      "name": "GetTemplateFunctions",
      "description": "GetTemplateFunctions returns the list of Go template functions available for payload transformation in subscriptions. Each entry includes the function name and a description of its behavior.",
      "request": [],
      "response": [
        {
          "name": "functions",
          "type": "TemplateFunction[]",
          "description": "Available template functions with names and descriptions."
        },
        {
          "name": "total_count",
          "type": "int32",
          "description": "Total number of available functions.",
          "example": 42
        }
      ]
    }
  ]
};

export default service;
