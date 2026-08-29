package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

type createSubscriptionBody struct {
	WebhookID         string            `json:"webhook_id" required:"true" doc:"Webhook to deliver matching events to."`
	EventName         string            `json:"event_name" required:"true" doc:"Event type name to subscribe to, or \"*\" to receive every event in the namespace (catch-all)."`
	Headers           map[string]string `json:"headers,omitempty" doc:"Extra HTTP headers to send with deliveries created by this subscription, merged with the webhook's own headers."`
	Method            string            `json:"method,omitempty" doc:"HTTP method used for deliveries from this subscription. Defaults to POST."`
	Timeout           int               `json:"timeout,omitempty" doc:"Per-delivery request timeout in seconds, overriding the webhook's default."`
	TransformEnabled  bool              `json:"transform_enabled,omitempty" doc:"Whether to render transform_template into the delivered payload instead of sending the raw event payload."`
	TransformTemplate string            `json:"transform_template,omitempty" doc:"Go template rendered against the event to produce the delivered body. See GET /v1/template-functions for available helpers; test it with POST /v1/subscriptions:testTemplate."`
	LabelFilters      map[string]string `json:"label_filters,omitempty" doc:"Key/value pairs that must ALL be present in an event's labels for this subscription to receive it. Empty means match every event of event_name."`
}

type createSubscriptionInput struct {
	Namespace string `path:"namespace"`
	Body      createSubscriptionBody
}

type subscriptionIDInput struct {
	Namespace      string `path:"namespace"`
	SubscriptionID string `path:"subscription_id"`
}

type subscriptionItem struct {
	SubscriptionID    string            `json:"subscription_id" doc:"Subscription id (UUID)."`
	Namespace         string            `json:"namespace" doc:"Tenant namespace this subscription belongs to."`
	WebhookID         string            `json:"webhook_id" doc:"Webhook this subscription delivers to."`
	EventName         string            `json:"event_name" doc:"Event type name this subscription matches, or \"*\" for catch-all."`
	Headers           map[string]string `json:"headers,omitempty" doc:"Extra HTTP headers sent with deliveries from this subscription."`
	Method            string            `json:"method,omitempty" enum:"GET,POST,PUT,PATCH,DELETE" doc:"HTTP method used for deliveries from this subscription."`
	Timeout           int               `json:"timeout,omitempty" doc:"Per-delivery request timeout in seconds, overriding the webhook's default."`
	TransformEnabled  bool              `json:"transform_enabled" doc:"Whether transform_template is rendered into the delivered payload."`
	TransformTemplate string            `json:"transform_template,omitempty" doc:"Go template rendered against the event to produce the delivered body."`
	LabelFilters      map[string]string `json:"label_filters,omitempty" doc:"Key/value pairs that must all be present in an event's labels for this subscription to receive it."`
	CreatedAt         string            `json:"created_at" doc:"Creation timestamp, RFC3339."`
	UpdatedAt         string            `json:"updated_at" doc:"Last-modified timestamp, RFC3339."`
}

type subscriptionOutput struct {
	Body subscriptionItem
}

func toSubscriptionItem(s *store.EventSubscription) subscriptionItem {
	return subscriptionItem{
		SubscriptionID:    s.ID.String(),
		Namespace:         s.Namespace,
		WebhookID:         s.WebhookID.String(),
		EventName:         s.EventName,
		Headers:           s.Headers,
		Method:            s.Method,
		Timeout:           s.Timeout,
		TransformEnabled:  s.TransformEnabled,
		TransformTemplate: s.TransformTemplate,
		LabelFilters:      s.LabelFilters,
		CreatedAt:         s.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:         s.UpdatedAt.Format(time.RFC3339Nano),
	}
}

func toSubscriptionOutput(s *store.EventSubscription) *subscriptionOutput {
	return &subscriptionOutput{Body: toSubscriptionItem(s)}
}

type listSubscriptionsInput struct {
	Namespace string `path:"namespace" doc:"Tenant namespace to list subscriptions in."`
	WebhookID string `query:"webhook_id,omitempty" doc:"Filter to subscriptions for one webhook."`
	EventName string `query:"event_name,omitempty" doc:"Filter to subscriptions for one event type name."`
	Limit     int32  `query:"limit" default:"50" doc:"Maximum items to return."`
	Offset    int32  `query:"offset" default:"0" doc:"Number of items to skip, for pagination."`
}

type listSubscriptionsOutput struct {
	Body struct {
		Items      []subscriptionItem `json:"items"`
		Pagination PaginationOutput   `json:"pagination"`
	}
}

// patchSubscriptionBody applies a partial update: only fields present in the
// request JSON are changed. webhook_id and event_name are immutable —
// delete and recreate the subscription to change either.
type patchSubscriptionBody struct {
	Headers           *map[string]string `json:"headers,omitempty" doc:"Replace the extra HTTP headers."`
	Method            *string            `json:"method,omitempty" doc:"Replace the HTTP method."`
	Timeout           *int               `json:"timeout,omitempty" doc:"Replace the per-delivery timeout override, in seconds."`
	TransformEnabled  *bool              `json:"transform_enabled,omitempty" doc:"Enable or disable payload transformation."`
	TransformTemplate *string            `json:"transform_template,omitempty" doc:"Replace the transform template."`
	LabelFilters      *map[string]string `json:"label_filters,omitempty" doc:"Replace the label filters."`
}

type patchSubscriptionInput struct {
	Namespace      string `path:"namespace"`
	SubscriptionID string `path:"subscription_id"`
	Body           patchSubscriptionBody
}

type testTemplateBody struct {
	EventName string `json:"event_name" required:"true" doc:"Registered event type whose sample payload the template is rendered against."`
	Template  string `json:"template" required:"true" doc:"Go template to render, in the same syntax used by transform_template."`
}

type testTemplateInput struct {
	Body testTemplateBody
}

type testTemplateOutput struct {
	Body struct {
		Rendered string `json:"rendered" doc:"The template rendered against the event type's sample payload."`
	}
}

func registerSubscriptionRoutes(api huma.API, d *Deps) {
	huma.Register(api, huma.Operation{
		OperationID:   "createSubscription",
		Method:        http.MethodPost,
		Path:          "/v1/namespaces/{namespace}/subscriptions",
		Summary:       "Create a subscription linking a webhook to an event",
		Description:   "Subscribes a webhook to an event type within a namespace, with an optional payload transform and label filters. Registering a webhook already auto-creates one subscription per listed event — use this endpoint for additional or catch-all (\"*\") subscriptions.",
		Errors:        []int{400, 404},
		Tags:          []string{"Subscriptions"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *createSubscriptionInput) (*subscriptionOutput, error) {
		id, _, err := d.Svc.CreateSubscription(ctx, in.Body.WebhookID, in.Body.EventName, in.Namespace, in.Body.Headers, in.Body.Method, in.Body.Timeout, in.Body.TransformEnabled, in.Body.TransformTemplate, in.Body.LabelFilters)
		if err != nil {
			return nil, mapError(ctx, err, "failed to create subscription")
		}
		sub, err := d.Svc.GetSubscription(ctx, id, in.Namespace)
		if err != nil {
			return nil, mapError(ctx, err, "failed to reload subscription")
		}
		return toSubscriptionOutput(sub), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getSubscription",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/subscriptions/{subscription_id}",
		Summary:     "Get a subscription by id",
		Description: "Fetches one subscription's webhook link, transform template, and label filters.",
		Errors:      []int{404},
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *subscriptionIDInput) (*subscriptionOutput, error) {
		sub, err := d.Svc.GetSubscription(ctx, in.SubscriptionID, in.Namespace)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get subscription")
		}
		return toSubscriptionOutput(sub), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listSubscriptions",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/subscriptions",
		Summary:     "List subscriptions",
		Description: "Lists subscriptions in a namespace, optionally filtered by webhook or event type name.",
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *listSubscriptionsInput) (*listSubscriptionsOutput, error) {
		limit, offset := in.Limit, in.Offset
		subs, total, err := d.Svc.ListSubscriptions(ctx, in.Namespace, in.WebhookID, in.EventName, limit, offset)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list subscriptions")
		}
		out := &listSubscriptionsOutput{}
		for _, s := range subs {
			out.Body.Items = append(out.Body.Items, toSubscriptionItem(s))
		}
		out.Body.Pagination = newPagination(limit, offset, total)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateSubscription",
		Method:      http.MethodPatch,
		Path:        "/v1/namespaces/{namespace}/subscriptions/{subscription_id}",
		Summary:     "Partially update a subscription",
		Description: "Merge-patches a subscription's headers, method, timeout, transform, or label filters. The linked webhook_id and event_name cannot be changed — delete and recreate instead.",
		Errors:      []int{400, 404},
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *patchSubscriptionInput) (*subscriptionOutput, error) {
		existing, err := d.Svc.GetSubscription(ctx, in.SubscriptionID, in.Namespace)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get subscription")
		}
		headers := map[string]string(existing.Headers)
		if in.Body.Headers != nil {
			headers = *in.Body.Headers
		}
		method := existing.Method
		if in.Body.Method != nil {
			method = *in.Body.Method
		}
		timeout := existing.Timeout
		if in.Body.Timeout != nil {
			timeout = *in.Body.Timeout
		}
		transformEnabled := existing.TransformEnabled
		if in.Body.TransformEnabled != nil {
			transformEnabled = *in.Body.TransformEnabled
		}
		transformTemplate := existing.TransformTemplate
		if in.Body.TransformTemplate != nil {
			transformTemplate = *in.Body.TransformTemplate
		}
		labelFilters := map[string]string(existing.LabelFilters)
		if in.Body.LabelFilters != nil {
			labelFilters = *in.Body.LabelFilters
		}
		if err := d.Svc.UpdateSubscription(ctx, in.SubscriptionID, in.Namespace, headers, method, timeout, transformEnabled, transformTemplate, labelFilters); err != nil {
			return nil, mapError(ctx, err, "failed to update subscription")
		}
		updated, err := d.Svc.GetSubscription(ctx, in.SubscriptionID, in.Namespace)
		if err != nil {
			return nil, mapError(ctx, err, "failed to reload subscription")
		}
		return toSubscriptionOutput(updated), nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteSubscription",
		Method:        http.MethodDelete,
		Path:          "/v1/namespaces/{namespace}/subscriptions/{subscription_id}",
		Summary:       "Delete a subscription",
		Description:   "Removes the link between a webhook and an event type. The webhook stops receiving that event's occurrences; its delivery history is unaffected.",
		Errors:        []int{404},
		Tags:          []string{"Subscriptions"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *subscriptionIDInput) (*emptyOutput, error) {
		if err := d.Svc.DeleteSubscription(ctx, in.SubscriptionID, in.Namespace); err != nil {
			return nil, mapError(ctx, err, "failed to delete subscription")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "testSubscriptionTemplate",
		Method:      http.MethodPost,
		Path:        "/v1/subscriptions:testTemplate",
		Summary:     "Render a transform template against an event's sample payload",
		Description: "Dry-runs a transform template against the named event type's stored sample payload, without creating a subscription or any delivery. Use this to iterate on transform_template before saving it.",
		Errors:      []int{400, 404},
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *testTemplateInput) (*testTemplateOutput, error) {
		rendered, err := d.Svc.TestSubscriptionTemplate(ctx, in.Body.EventName, in.Body.Template, "")
		if err != nil {
			return nil, mapError(ctx, err, "failed to test template")
		}
		out := &testTemplateOutput{}
		out.Body.Rendered = rendered
		return out, nil
	})
}
