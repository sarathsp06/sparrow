package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

type createSubscriptionBody struct {
	WebhookID          string            `json:"webhook_id" required:"true" doc:"Webhook to deliver matching events to."`
	EventName          string            `json:"event_name" required:"true" doc:"Event type name to subscribe to, or \"*\" to receive every event in the consumer (catch-all)."`
	Headers            map[string]string `json:"headers,omitempty" doc:"Extra HTTP headers to send with deliveries created by this subscription, merged with the webhook's own headers."`
	Method             string            `json:"method,omitempty" doc:"HTTP method used for deliveries from this subscription. Defaults to POST."`
	Timeout            int               `json:"timeout,omitempty" doc:"Per-delivery request timeout in seconds, overriding the webhook's default."`
	TransformEnabled   bool              `json:"transform_enabled,omitempty" doc:"Whether to render transform_template into the delivered payload instead of sending the raw event payload."`
	TransformTemplate  string            `json:"transform_template,omitempty" doc:"Go template rendered against the event to produce the delivered body. See GET /v1/template-functions for available helpers; test it with POST /v1/subscriptions:testTemplate."`
	LabelFilters       map[string]string `json:"label_filters,omitempty" doc:"Key/value pairs that must ALL be present in an event's labels for this subscription to receive it. Empty means match every event of event_name."`
	OnTransformError   string            `json:"on_transform_error,omitempty" enum:"fail,fallback" doc:"What happens when transform_template fails to render. fail (default): the delivery fails with error category template_error, is not retried automatically, and can be retried once the template is fixed; it does not affect the webhook's health. fallback: the default envelope payload is sent instead, and the error is still recorded on the delivery."`
	TemplateMissingKey string            `json:"template_missing_key,omitempty" enum:"error,zero" doc:"How transform_template reads a key the payload does not have. error (default): the render fails, so a field removed from the event schema cannot silently turn into \"<no value>\" in the body; read optional fields with index, dig or default. zero: the key renders as \"<no value>\"."`
}

type createSubscriptionInput struct {
	Consumer string `path:"consumer"`
	Body     createSubscriptionBody
}

type subscriptionIDInput struct {
	Consumer       string `path:"consumer"`
	SubscriptionID string `path:"subscription_id"`
}

type SubscriptionItem struct {
	SubscriptionID     string            `json:"subscription_id" doc:"Subscription id (UUID)."`
	Consumer           string            `json:"consumer" doc:"Tenant consumer this subscription belongs to."`
	WebhookID          string            `json:"webhook_id" doc:"Webhook this subscription delivers to."`
	EventName          string            `json:"event_name" doc:"Event type name this subscription matches, or \"*\" for catch-all."`
	Headers            map[string]string `json:"headers,omitempty" doc:"Extra HTTP headers sent with deliveries from this subscription."`
	Method             string            `json:"method,omitempty" enum:"GET,POST,PUT,PATCH,DELETE" doc:"HTTP method used for deliveries from this subscription."`
	Timeout            int               `json:"timeout,omitempty" doc:"Per-delivery request timeout in seconds, overriding the webhook's default."`
	TransformEnabled   bool              `json:"transform_enabled" doc:"Whether transform_template is rendered into the delivered payload."`
	TransformTemplate  string            `json:"transform_template,omitempty" doc:"Go template rendered against the event to produce the delivered body."`
	LabelFilters       map[string]string `json:"label_filters,omitempty" doc:"Key/value pairs that must all be present in an event's labels for this subscription to receive it."`
	OnTransformError   string            `json:"on_transform_error" enum:"fail,fallback" doc:"What happens when transform_template fails to render. fail (default): the delivery fails with error category template_error, is not retried automatically, and can be retried once the template is fixed; it does not affect the webhook's health. fallback: the default envelope payload is sent instead, and the error is still recorded on the delivery."`
	TemplateMissingKey string            `json:"template_missing_key" enum:"error,zero" doc:"How transform_template reads a key the payload does not have. error (default): the render fails, so a field removed from the event schema cannot silently turn into \"<no value>\" in the body; read optional fields with index, dig or default. zero: the key renders as \"<no value>\"."`
	Paused             bool              `json:"paused" doc:"Whether the subscription is paused. While paused, events still fan out to it, but each delivery is recorded with status paused and not attempted until retried. A pause never affects the webhook's health."`
	PausedAt           *string           `json:"paused_at,omitempty" doc:"When the current pause began, RFC3339."`
	PausedReason       string            `json:"paused_reason,omitempty" doc:"Why the subscription was paused."`
	CreatedAt          string            `json:"created_at" doc:"Creation timestamp, RFC3339."`
	UpdatedAt          string            `json:"updated_at" doc:"Last-modified timestamp, RFC3339."`
}

type subscriptionOutput struct {
	Body SubscriptionItem
}

func toSubscriptionItem(s *store.EventSubscription) SubscriptionItem {
	item := SubscriptionItem{
		SubscriptionID:     s.ID.String(),
		Consumer:           s.Consumer,
		WebhookID:          s.WebhookID.String(),
		EventName:          s.EventName,
		Headers:            s.Headers,
		Method:             s.Method,
		Timeout:            s.Timeout,
		TransformEnabled:   s.TransformEnabled,
		TransformTemplate:  s.TransformTemplate,
		LabelFilters:       s.LabelFilters,
		OnTransformError:   s.OnTransformError,
		TemplateMissingKey: s.TemplateMissingKey,
		Paused:             s.Paused(),
		PausedReason:       s.PausedReason,
		CreatedAt:          s.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:          s.UpdatedAt.Format(time.RFC3339Nano),
	}
	if s.PausedAt != nil {
		t := s.PausedAt.Format(time.RFC3339Nano)
		item.PausedAt = &t
	}
	return item
}

type pauseSubscriptionInput struct {
	Consumer       string `path:"consumer"`
	SubscriptionID string `path:"subscription_id"`
	Body           struct {
		Reason string `json:"reason,omitempty" maxLength:"500" doc:"Why the subscription is paused, shown next to it, for example: receiver maintenance until Friday."`
	}
}

type resumeSubscriptionOutput struct {
	Body struct {
		SubscriptionItem
		PausedSince      *string `json:"paused_since,omitempty" doc:"When the pause that just ended began, RFC3339. List or retry what was held with status=paused&subscription_id=...&created_after=<paused_since>."`
		PausedDeliveries int     `json:"paused_deliveries" doc:"Deliveries recorded while paused. They are not sent automatically; retry them to deliver."`
	}
}

func toSubscriptionOutput(s *store.EventSubscription) *subscriptionOutput {
	return &subscriptionOutput{Body: toSubscriptionItem(s)}
}

// SubscriptionListParams are the query filters shared by the consumer-scoped
// and global subscription list routes.
type SubscriptionListParams struct {
	WebhookID string `query:"webhook_id,omitempty" doc:"Filter to subscriptions for one webhook."`
	EventName string `query:"event_name,omitempty" doc:"Filter to subscriptions for one event type name."`
	Limit     int32  `query:"limit" default:"50" minimum:"1" maximum:"1000" doc:"Maximum items to return."`
	Offset    int32  `query:"offset" default:"0" doc:"Number of items to skip, for pagination."`
}

type listSubscriptionsInput struct {
	Consumer string `path:"consumer" doc:"Tenant consumer to list subscriptions in."`
	SubscriptionListParams
}

type listSubscriptionsGlobalInput struct {
	Consumer string `query:"consumer,omitempty" doc:"Filter to one consumer; omit to list subscriptions across all consumers."`
	SubscriptionListParams
}

type listSubscriptionsOutput struct {
	Body struct {
		Items      []SubscriptionItem `json:"items"`
		Pagination PaginationOutput   `json:"pagination"`
	}
}

// patchSubscriptionBody applies a partial update: only fields present in the
// request JSON are changed. webhook_id and event_name are immutable —
// delete and recreate the subscription to change either.
type patchSubscriptionBody struct {
	Headers            *map[string]string `json:"headers,omitempty" doc:"Replace the extra HTTP headers."`
	Method             *string            `json:"method,omitempty" doc:"Replace the HTTP method."`
	Timeout            *int               `json:"timeout,omitempty" doc:"Replace the per-delivery timeout override, in seconds."`
	TransformEnabled   *bool              `json:"transform_enabled,omitempty" doc:"Enable or disable payload transformation."`
	TransformTemplate  *string            `json:"transform_template,omitempty" doc:"Replace the transform template."`
	LabelFilters       *map[string]string `json:"label_filters,omitempty" doc:"Replace the label filters."`
	OnTransformError   *string            `json:"on_transform_error,omitempty" enum:"fail,fallback" doc:"Replace what happens when the template fails to render."`
	TemplateMissingKey *string            `json:"template_missing_key,omitempty" enum:"error,zero" doc:"Replace how the template reads a missing key."`
}

type patchSubscriptionInput struct {
	Consumer       string `path:"consumer"`
	SubscriptionID string `path:"subscription_id"`
	Body           patchSubscriptionBody
}

type testTemplateBody struct {
	EventName          string `json:"event_name" required:"true" doc:"Registered event type whose sample payload the template is rendered against."`
	Template           string `json:"template" required:"true" doc:"Go template to render, in the same syntax used by transform_template."`
	TemplateMissingKey string `json:"template_missing_key,omitempty" enum:"error,zero" required:"false" doc:"Render as a subscription with this template_missing_key would. Defaults to error, the subscription default."`
}

type testTemplateInput struct {
	Body testTemplateBody
}

type testTemplateOutput struct {
	Body struct {
		Rendered string `json:"rendered" doc:"The template rendered against the event type's sample payload."`
	}
}

func registerSubscriptionRoutes(api huma.API, svc webhooks.SubscriptionManager) {
	huma.Register(api, huma.Operation{
		OperationID:   "createSubscription",
		Method:        http.MethodPost,
		Path:          "/v1/consumers/{consumer}/subscriptions",
		Summary:       "Create a subscription linking a webhook to an event",
		Description:   "Subscribes a webhook to an event type within a consumer, with an optional payload transform and label filters. Registering a webhook already auto-creates one subscription per listed event — use this endpoint for additional or catch-all (\"*\") subscriptions.",
		Errors:        []int{400, 404},
		Tags:          []string{"Subscriptions"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *createSubscriptionInput) (*subscriptionOutput, error) {
		id, _, err := svc.CreateSubscription(ctx, in.Body.WebhookID, in.Body.EventName, in.Consumer, in.Body.Headers, in.Body.Method, in.Body.Timeout, in.Body.TransformEnabled, in.Body.TransformTemplate, in.Body.LabelFilters, webhooks.SubscriptionTemplateSettings{
			OnTransformError:   in.Body.OnTransformError,
			TemplateMissingKey: in.Body.TemplateMissingKey,
		})
		if err != nil {
			return nil, mapError(ctx, err, "failed to create subscription")
		}
		sub, err := svc.GetSubscription(ctx, id, in.Consumer)
		if err != nil {
			return nil, mapError(ctx, err, "failed to reload subscription")
		}
		return toSubscriptionOutput(sub), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getSubscription",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/subscriptions/{subscription_id}",
		Summary:     "Get a subscription by id",
		Description: "Fetches one subscription's webhook link, transform template, and label filters.",
		Errors:      []int{404},
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *subscriptionIDInput) (*subscriptionOutput, error) {
		sub, err := svc.GetSubscription(ctx, in.SubscriptionID, in.Consumer)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get subscription")
		}
		return toSubscriptionOutput(sub), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listSubscriptions",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/subscriptions",
		Summary:     "List subscriptions",
		Description: "Lists subscriptions in a consumer, optionally filtered by webhook or event type name.",
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *listSubscriptionsInput) (*listSubscriptionsOutput, error) {
		return listSubscriptionsImpl(ctx, svc, in.Consumer, in.SubscriptionListParams)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listSubscriptionsGlobal",
		Method:      http.MethodGet,
		Path:        "/v1/subscriptions",
		Summary:     "List subscriptions across all consumers",
		Description: "Cross-consumer subscription listing with the same filters as the per-consumer route. Pass consumer to scope to one consumer.",
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *listSubscriptionsGlobalInput) (*listSubscriptionsOutput, error) {
		return listSubscriptionsImpl(ctx, svc, in.Consumer, in.SubscriptionListParams)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateSubscription",
		Method:      http.MethodPatch,
		Path:        "/v1/consumers/{consumer}/subscriptions/{subscription_id}",
		Summary:     "Partially update a subscription",
		Description: "Merge-patches a subscription's headers, method, timeout, transform, or label filters. The linked webhook_id and event_name cannot be changed — delete and recreate instead.",
		Errors:      []int{400, 404},
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *patchSubscriptionInput) (*subscriptionOutput, error) {
		existing, err := svc.GetSubscription(ctx, in.SubscriptionID, in.Consumer)
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
		settings := webhooks.SubscriptionTemplateSettings{
			OnTransformError:   existing.OnTransformError,
			TemplateMissingKey: existing.TemplateMissingKey,
		}
		if in.Body.OnTransformError != nil {
			settings.OnTransformError = *in.Body.OnTransformError
		}
		if in.Body.TemplateMissingKey != nil {
			settings.TemplateMissingKey = *in.Body.TemplateMissingKey
		}
		if err := svc.UpdateSubscription(ctx, in.SubscriptionID, in.Consumer, headers, method, timeout, transformEnabled, transformTemplate, labelFilters, settings); err != nil {
			return nil, mapError(ctx, err, "failed to update subscription")
		}
		updated, err := svc.GetSubscription(ctx, in.SubscriptionID, in.Consumer)
		if err != nil {
			return nil, mapError(ctx, err, "failed to reload subscription")
		}
		return toSubscriptionOutput(updated), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "pauseSubscription",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/subscriptions/{subscription_id}:pause",
		Summary:     "Pause a subscription",
		Description: "Stops the subscription's deliveries from being attempted, for any reason (receiver maintenance, a template being fixed, an investigation). Events keep fanning out to it: each delivery is recorded with status paused and nothing is queued, so nothing is lost. Deliveries already queued finish. A pause never affects the webhook's health. Pausing again updates the reason and keeps the original pause time.",
		Errors:      []int{400, 404},
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *pauseSubscriptionInput) (*subscriptionOutput, error) {
		sub, err := svc.PauseSubscription(ctx, in.SubscriptionID, in.Consumer, in.Body.Reason)
		if err != nil {
			return nil, mapError(ctx, err, "failed to pause subscription")
		}
		return toSubscriptionOutput(sub), nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "resumeSubscription",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/subscriptions/{subscription_id}:resume",
		Summary:     "Resume a paused subscription",
		Description: "New deliveries are attempted again. Deliveries recorded while paused stay paused and are never sent automatically: the response says how many there are and when the pause began, so you can retry all of them or only those since a time with the delivery retry endpoints.",
		Errors:      []int{404},
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *subscriptionIDInput) (*resumeSubscriptionOutput, error) {
		res, err := svc.ResumeSubscription(ctx, in.SubscriptionID, in.Consumer)
		if err != nil {
			return nil, mapError(ctx, err, "failed to resume subscription")
		}
		out := &resumeSubscriptionOutput{}
		out.Body.SubscriptionItem = toSubscriptionItem(res.Subscription)
		out.Body.PausedDeliveries = res.PausedDeliveries
		if res.PausedSince != nil {
			t := res.PausedSince.Format(time.RFC3339Nano)
			out.Body.PausedSince = &t
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteSubscription",
		Method:        http.MethodDelete,
		Path:          "/v1/consumers/{consumer}/subscriptions/{subscription_id}",
		Summary:       "Delete a subscription",
		Description:   "Removes the link between a webhook and an event type. The webhook stops receiving that event's occurrences; its delivery history is unaffected.",
		Errors:        []int{404},
		Tags:          []string{"Subscriptions"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *subscriptionIDInput) (*emptyOutput, error) {
		if err := svc.DeleteSubscription(ctx, in.SubscriptionID, in.Consumer); err != nil {
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
		rendered, err := svc.TestSubscriptionTemplate(ctx, in.Body.EventName, in.Body.Template, "", in.Body.TemplateMissingKey != store.TemplateMissingKeyZero)
		if err != nil {
			return nil, mapError(ctx, err, "failed to test template")
		}
		out := &testTemplateOutput{}
		out.Body.Rendered = rendered
		return out, nil
	})
}

// listSubscriptionsImpl is the shared body of the consumer-scoped and global
// subscription list routes. An empty consumer lists across all consumers.
func listSubscriptionsImpl(ctx context.Context, svc webhooks.SubscriptionManager, consumer string, p SubscriptionListParams) (*listSubscriptionsOutput, error) {
	subs, total, err := svc.ListSubscriptions(ctx, consumer, p.WebhookID, p.EventName, p.Limit, p.Offset)
	if err != nil {
		return nil, mapError(ctx, err, "failed to list subscriptions")
	}
	out := &listSubscriptionsOutput{}
	out.Body.Items = make([]SubscriptionItem, 0, len(subs))
	for _, s := range subs {
		out.Body.Items = append(out.Body.Items, toSubscriptionItem(s))
	}
	out.Body.Pagination = newPagination(p.Limit, p.Offset, total)
	return out, nil
}
