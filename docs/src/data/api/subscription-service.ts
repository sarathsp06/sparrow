import type { ApiService } from "./types";

const service: ApiService = {
  "service": "SubscriptionService",
  "package": "webhook",
  "description": "SubscriptionService manages the mapping between webhooks and event types. A subscription links one webhook to one event name within a namespace. When an event is pushed, all subscriptions matching (namespace, event_name, label_filters) generate a delivery. Subscriptions can optionally transform the payload using Go templates.",
  "rpcs": [
    {
      "name": "CreateSubscription",
      "description": "CreateSubscription links a webhook to an event name in a namespace. Optionally configure per-subscription headers, HTTP method override, timeout override, payload transformation (Go template), and label filters for selective matching. Errors: ALREADY_EXISTS if the webhook is already subscribed to this event in this namespace. Errors: NOT_FOUND if the webhook_id does not exist.",
      "request": [
        {
          "name": "webhook_id",
          "type": "string",
          "description": "UUID of the webhook to subscribe. Required. The webhook must exist in the specified namespace.",
          "example": "550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "event_name",
          "type": "string",
          "description": "Event type name to subscribe to. Required.",
          "example": "order.created"
        },
        {
          "name": "namespace",
          "type": "string",
          "description": "Namespace for the subscription. Required. Must match the webhook's namespace.",
          "example": "production"
        },
        {
          "name": "headers",
          "type": "map<string, string>",
          "description": "Per-subscription HTTP headers. Optional. Merged with (and overrides) webhook-level headers on delivery."
        },
        {
          "name": "method",
          "type": "string",
          "description": "HTTP method override. Optional. Default: \"POST\"."
        },
        {
          "name": "timeout",
          "type": "int32",
          "description": "Timeout override in seconds. Optional. Default: use webhook's timeout."
        },
        {
          "name": "transform_enabled",
          "type": "bool",
          "description": "Enable Go template payload transformation. Default: false."
        },
        {
          "name": "transform_template",
          "type": "string",
          "description": "Go template for payload transformation. Only used when transform_enabled is true. Use TestSubscriptionTemplate to validate before saving."
        },
        {
          "name": "label_filters",
          "type": "map<string, string>",
          "description": "Label filters for selective event matching. Optional. Only events with labels matching all key-value pairs will trigger this subscription.",
          "example": {
            "region": "us-east"
          }
        }
      ],
      "response": [
        {
          "name": "subscription_id",
          "type": "string",
          "description": "Server-generated UUID of the new subscription.",
          "example": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
        },
        {
          "name": "created_at",
          "type": "Timestamp",
          "description": "When the subscription was created.",
          "example": "2025-01-15T10:30:00Z"
        }
      ]
    },
    {
      "name": "GetSubscription",
      "description": "GetSubscription returns a single subscription by ID. Errors: NOT_FOUND if the subscription_id does not exist in the given namespace.",
      "request": [
        {
          "name": "subscription_id",
          "type": "string",
          "description": "UUID of the subscription to retrieve. Required.",
          "example": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
        },
        {
          "name": "namespace",
          "type": "string",
          "description": "Namespace the subscription belongs to. Required.",
          "example": "production"
        }
      ],
      "response": [
        {
          "name": "subscription",
          "type": "EventSubscription",
          "description": "Full subscription details."
        }
      ]
    },
    {
      "name": "ListSubscriptions",
      "description": "ListSubscriptions returns subscriptions in a namespace, optionally filtered by webhook_id or event_name. Results are paginated.",
      "request": [
        {
          "name": "webhook_id",
          "type": "string",
          "description": "Filter by webhook UUID. Optional.",
          "example": "550e8400-e29b-41d4-a716-446655440000"
        },
        {
          "name": "event_name",
          "type": "string",
          "description": "Filter by event type name. Optional."
        },
        {
          "name": "namespace",
          "type": "string",
          "description": "Namespace to list subscriptions from. Required.",
          "example": "production"
        },
        {
          "name": "pagination",
          "type": "PaginationRequest",
          "description": "Pagination parameters. Default: limit=50, offset=0."
        }
      ],
      "response": [
        {
          "name": "subscriptions",
          "type": "EventSubscription[]",
          "description": "Subscriptions matching the filter criteria."
        },
        {
          "name": "pagination",
          "type": "PaginationResponse",
          "description": "Pagination metadata."
        }
      ]
    },
    {
      "name": "UpdateSubscription",
      "description": "UpdateSubscription modifies a subscription's headers, method, timeout, transform settings, or label filters. Only non-zero fields are applied. Errors: NOT_FOUND if the subscription does not exist.",
      "request": [
        {
          "name": "subscription_id",
          "type": "string",
          "description": "UUID of the subscription to update. Required.",
          "example": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
        },
        {
          "name": "namespace",
          "type": "string",
          "description": "Namespace the subscription belongs to. Required.",
          "example": "production"
        },
        {
          "name": "headers",
          "type": "map<string, string>",
          "description": "Updated per-subscription headers. Replaces existing headers when set."
        },
        {
          "name": "method",
          "type": "string",
          "description": "Updated HTTP method override."
        },
        {
          "name": "timeout",
          "type": "int32",
          "description": "Updated timeout override in seconds."
        },
        {
          "name": "transform_enabled",
          "type": "bool",
          "description": "Updated transform enabled flag.",
          "example": true
        },
        {
          "name": "transform_template",
          "type": "string",
          "description": "Updated Go template. Use TestSubscriptionTemplate to validate before saving."
        },
        {
          "name": "label_filters",
          "type": "map<string, string>",
          "description": "Updated label filters. Replaces existing filters when set."
        }
      ]
    },
    {
      "name": "DeleteSubscription",
      "description": "DeleteSubscription permanently removes a subscription. Existing in-flight deliveries for this subscription are not cancelled. Errors: NOT_FOUND if the subscription does not exist.",
      "request": [
        {
          "name": "subscription_id",
          "type": "string",
          "description": "UUID of the subscription to delete. Required.",
          "example": "7c9e6679-7425-40de-944b-e07fc1f90ae7"
        },
        {
          "name": "namespace",
          "type": "string",
          "description": "Namespace the subscription belongs to. Required.",
          "example": "production"
        }
      ]
    },
    {
      "name": "TestSubscriptionTemplate",
      "description": "TestSubscriptionTemplate renders a Go template against the sample payload of the given event type. Returns the transformed output string. Use this to validate templates before saving them on a subscription. Errors: INVALID_ARGUMENT if the template fails to parse or execute. Errors: NOT_FOUND if the event_name does not exist.",
      "request": [
        {
          "name": "event_name",
          "type": "string",
          "description": "Event name to get the sample payload from. Required. The event type must have a schema defined for a sample payload to be generated.",
          "example": "order.created"
        },
        {
          "name": "transform_template",
          "type": "string",
          "description": "Go template string to test. Required. The template is executed with the event's sample_payload as its data context.",
          "example": "{\"id\": \"{{ .payload.order_id }}\", \"total\": \"{{ .payload.amount }}\"}"
        },
        {
          "name": "namespace",
          "type": "string",
          "description": "Namespace. Required.",
          "example": "production"
        }
      ],
      "response": [
        {
          "name": "transformed_payload",
          "type": "string",
          "description": "The template output after rendering against the sample payload. This is exactly what would be sent as the delivery request body if this template were applied to a subscription.",
          "example": "{\"id\": \"ord-123\", \"total\": \"99.99\"}"
        }
      ]
    }
  ]
};

export default service;
