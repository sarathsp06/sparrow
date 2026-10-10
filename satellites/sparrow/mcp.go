package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"
)

const mcpHelp = `Serve Sparrow to AI agents over the Model Context Protocol (stdio).

Read-only by default: agents can list webhooks, subscriptions, deliveries,
attempts, event types and stats, and render a template against an event
type's sample payload. With --allow-write they can also push events, retry
deliveries, register and delete webhooks, create subscriptions, and run a
temporary local receiver (start_listen / list_received / stop_listen) to see
what a subscription actually delivers.

Register it with an MCP client, for example Claude Code:

  claude mcp add sparrow -- sparrow mcp --url http://localhost:8080 --consumer default

Connection flags, SPARROW_* env vars and ~/.config/sparrow apply as for every
other command.
`

func newMCPCmd() *cobra.Command {
	var allowWrite bool
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Serve Sparrow tools to AI agents over MCP (stdio)",
		Long:  mcpHelp,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			srv, listener := newMCPServer(client, cfg.Consumer, allowWrite)
			runErr := srv.Run(cmd.Context(), &mcp.StdioTransport{})
			// The client went away: never leave a temporary webhook behind.
			if _, err := listener.stop(context.Background()); err != nil {
				return err
			}
			return runErr
		},
	}
	cmd.Flags().BoolVar(&allowWrite, "allow-write", false, "expose tools that change state: push, retry, register/delete webhooks, subscriptions, listen")
	return cmd
}

// Tool inputs. Field descriptions become the JSON Schema the agent sees.

type listWebhooksIn struct {
	ActiveOnly bool `json:"active_only,omitempty" jsonschema:"only webhooks that are active"`
}

type webhookIDIn struct {
	WebhookID string `json:"webhook_id" jsonschema:"webhook id"`
}

type listDeliveriesIn struct {
	Status    string `json:"status,omitempty" jsonschema:"filter by status: pending, success, failed, retrying, paused"`
	WebhookID string `json:"webhook_id,omitempty" jsonschema:"only deliveries to this webhook"`
	EventName string `json:"event_name,omitempty" jsonschema:"only deliveries of this event type, e.g. order.created"`
	Limit     int    `json:"limit,omitempty" jsonschema:"max rows, default 50, max 1000"`
	Cursor    string `json:"cursor,omitempty" jsonschema:"pagination.next_cursor from the previous call, to fetch the next (older) page"`
}

type deliveryIDIn struct {
	DeliveryID string `json:"delivery_id" jsonschema:"delivery id (UUID)"`
}

type testTemplateIn struct {
	EventName string `json:"event_name" jsonschema:"registered event type whose sample payload the template is rendered against"`
	Template  string `json:"template" jsonschema:"Go transform template, same syntax as transform_template"`
}

type pushEventIn struct {
	Event          string            `json:"event" jsonschema:"event type name, e.g. order.created; auto-created when missing"`
	Payload        map[string]any    `json:"payload" jsonschema:"event payload object"`
	Labels         map[string]string `json:"labels,omitempty" jsonschema:"label key/values used by subscription filters"`
	IdempotencyKey string            `json:"idempotency_key,omitempty" jsonschema:"client-supplied dedup key"`
}

type registerWebhookIn struct {
	URL               string            `json:"url" jsonschema:"endpoint deliveries are POSTed to"`
	Events            []string          `json:"events,omitempty" jsonschema:"event type names to subscribe to right away (a subscription each)"`
	Description       string            `json:"description,omitempty" jsonschema:"free-text note, e.g. owning team"`
	Headers           map[string]string `json:"headers,omitempty" jsonschema:"static headers sent with every delivery"`
	TransformTemplate string            `json:"transform_template,omitempty" jsonschema:"Go template applied to every subscription created from events"`
	RequiresTransform bool              `json:"requires_transform,omitempty" jsonschema:"receiver only accepts transformed payloads (needs transform_template)"`
}

type createSubscriptionIn struct {
	WebhookID         string            `json:"webhook_id" jsonschema:"webhook to deliver matching events to"`
	EventName         string            `json:"event_name" jsonschema:"event type name, or * for every event"`
	LabelFilters      map[string]string `json:"label_filters,omitempty" jsonschema:"key/values an event's labels must all contain"`
	TransformTemplate string            `json:"transform_template,omitempty" jsonschema:"Go template producing the delivered body; enables the transform when set"`
}

// newMCPServer builds the MCP server and the local receiver the listen tools
// drive. Tools return the Sparrow API response as-is (JSON), so what the
// agent sees matches the REST docs.
func newMCPServer(client *apiClient, consumer string, allowWrite bool) (*mcp.Server, *mcpListener) {
	srv := mcp.NewServer(&mcp.Implementation{Name: "sparrow", Version: version}, nil)
	listener := &mcpListener{client: client, consumer: consumer}
	ro := &mcp.ToolAnnotations{ReadOnlyHint: true}
	c := url.PathEscape(consumer)

	get := func(ctx context.Context, path string) (any, error) {
		var out any
		err := client.do(ctx, http.MethodGet, path, nil, &out)
		return out, err
	}
	post := func(ctx context.Context, path string, body any) (any, error) {
		var out any
		err := client.do(ctx, http.MethodPost, path, body, &out)
		return out, err
	}

	mcp.AddTool(srv, &mcp.Tool{Name: "list_webhooks", Annotations: ro,
		Description: "List the consumer's registered webhooks with their url, events, active flag and health."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in listWebhooksIn) (*mcp.CallToolResult, any, error) {
			q := url.Values{"limit": {"200"}}
			if in.ActiveOnly {
				q.Set("active", "true")
			}
			out, err := get(ctx, "/v1/consumers/"+c+"/webhooks?"+q.Encode())
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "list_subscriptions", Annotations: ro,
		Description: "List one webhook's subscriptions: event name, label filters and transform template."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in webhookIDIn) (*mcp.CallToolResult, any, error) {
			out, err := get(ctx, "/v1/consumers/"+c+"/subscriptions?webhook_id="+url.QueryEscape(in.WebhookID))
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "list_deliveries", Annotations: ro,
		Description: "List deliveries newest first with status, response code, attempt count and error category. When pagination.has_more is true, pass pagination.next_cursor as cursor to get the next page."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in listDeliveriesIn) (*mcp.CallToolResult, any, error) {
			if in.Limit <= 0 {
				in.Limit = 50
			}
			q := url.Values{"limit": {fmt.Sprint(in.Limit)}}
			for k, v := range map[string]string{"status": in.Status, "webhook_id": in.WebhookID, "event_name": in.EventName, "cursor": in.Cursor} {
				if v != "" {
					q.Set(k, v)
				}
			}
			out, err := get(ctx, "/v1/consumers/"+c+"/deliveries?"+q.Encode())
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "get_delivery_attempts", Annotations: ro,
		Description: "Per-attempt history of one delivery: response code, timing, error category and message. Use it to explain why a delivery failed."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in deliveryIDIn) (*mcp.CallToolResult, any, error) {
			out, err := get(ctx, "/v1/consumers/"+c+"/deliveries/"+url.PathEscape(in.DeliveryID)+"/attempts")
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "list_event_types", Annotations: ro,
		Description: "List registered event types with description, JSON schema and sample payload."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			out, err := get(ctx, "/v1/event-types?limit=200")
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "get_stats", Annotations: ro,
		Description: "Aggregate delivery statistics for the consumer: webhook counts, delivery counts by outcome, success rate."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			out, err := get(ctx, "/v1/consumers/"+c+"/stats")
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "test_template", Annotations: ro,
		Description: "Render a transform template server-side against an event type's sample payload. Nothing is delivered; use it to iterate on a template before create_subscription."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in testTemplateIn) (*mcp.CallToolResult, any, error) {
			out, err := post(ctx, "/v1/subscriptions:testTemplate", map[string]any{"event_name": in.EventName, "template": in.Template})
			return nil, out, err
		})

	if !allowWrite {
		return srv, listener
	}

	mcp.AddTool(srv, &mcp.Tool{Name: "push_event",
		Description: "Push one event occurrence (a test push). An unknown event type is auto-created. Returns the event id and whether it was a duplicate."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in pushEventIn) (*mcp.CallToolResult, any, error) {
			if in.Payload == nil {
				in.Payload = map[string]any{}
			}
			res, created, err := pushWithAutoCreate(ctx, client, consumer, in.Event, pushBody{Payload: in.Payload, Labels: in.Labels, IdempotencyKey: in.IdempotencyKey})
			if err != nil {
				return nil, nil, err
			}
			return nil, map[string]any{"event_id": res.EventID, "duplicate": res.Duplicate, "event_type_created": created}, nil
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "retry_delivery",
		Description: "Immediately re-attempt one delivery regardless of its status or remaining attempt budget."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in deliveryIDIn) (*mcp.CallToolResult, any, error) {
			out, err := post(ctx, "/v1/consumers/"+c+"/deliveries/"+url.PathEscape(in.DeliveryID)+":retry", nil)
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "register_webhook",
		Description: "Register a webhook endpoint, optionally subscribing it to events in the same call. Returns the webhook including its signing secret."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in registerWebhookIn) (*mcp.CallToolResult, any, error) {
			out, err := post(ctx, "/v1/consumers/"+c+"/webhooks", webhookRequest{
				URL: in.URL, Events: in.Events, Active: true, Description: in.Description, Headers: in.Headers,
				TransformTemplate: in.TransformTemplate, RequiresTransform: in.RequiresTransform,
			})
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "delete_webhook", Annotations: &mcp.ToolAnnotations{DestructiveHint: ptr(true)},
		Description: "Delete a webhook and all of its subscriptions."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in webhookIDIn) (*mcp.CallToolResult, any, error) {
			err := client.deleteWebhook(ctx, consumer, in.WebhookID)
			return nil, map[string]any{"deleted": in.WebhookID}, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "create_subscription",
		Description: "Subscribe an existing webhook to an event type, optionally with label filters and a transform template."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in createSubscriptionIn) (*mcp.CallToolResult, any, error) {
			body := map[string]any{"webhook_id": in.WebhookID, "event_name": in.EventName}
			if len(in.LabelFilters) > 0 {
				body["label_filters"] = in.LabelFilters
			}
			if in.TransformTemplate != "" {
				body["transform_template"] = in.TransformTemplate
				body["transform_enabled"] = true
			}
			out, err := post(ctx, "/v1/consumers/"+c+"/subscriptions", body)
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "start_listen",
		Description: "Start a temporary local receiver and register it as a webhook for the given events (like `sparrow listen`). Deliveries are verified and kept in memory for list_received. One listener at a time; stop_listen deletes the webhook."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in startListenIn) (*mcp.CallToolResult, any, error) {
			out, err := listener.start(ctx, in)
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "list_received", Annotations: ro,
		Description: "Deliveries the temporary receiver has accepted since start_listen, oldest first, with headers and body."},
		func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			out, err := listener.current()
			return nil, out, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "stop_listen",
		Description: "Stop the temporary receiver and delete its webhook. Returns what it received."},
		func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			out, err := listener.stop(ctx)
			return nil, out, err
		})

	return srv, listener
}

func ptr[T any](v T) *T { return &v }
