package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
	"github.com/sarathsp06/sparrow/pkg/access/httpauth"
)

type createSubscriptionBody struct {
	WebhookID          string            `json:"webhook_id" required:"true" doc:"Webhook id (UUID) to deliver matching events to. Must belong to this consumer."`
	EventName          string            `json:"event_name" required:"true" doc:"Event type name to subscribe to, or \"*\" to receive every event in the consumer (catch-all). sparrow.* names only under the _sparrow consumer. One subscription per webhook, event name and consumer (409 on a duplicate)."`
	Headers            map[string]string `json:"headers,omitempty" doc:"Extra HTTP headers sent with deliveries from this subscription. They override the webhook's headers of the same name; the webhook's secret_headers override both. Same name and value rules as webhook headers."`
	Method             string            `json:"method,omitempty" doc:"HTTP method used for deliveries from this subscription, e.g. PUT. Defaults to POST."`
	Timeout            int               `json:"timeout,omitempty" doc:"Request timeout for this subscription's deliveries, in seconds, overriding the webhook's request_timeout_seconds. 0 to 300; 0 (default) uses the webhook's."`
	TransformEnabled   bool              `json:"transform_enabled,omitempty" doc:"Whether to render transform_template into the delivered body instead of sending the default envelope. Defaults to false."`
	TransformTemplate  string            `json:"transform_template,omitempty" doc:"Go text/template rendered against the event to produce the delivered body, used when transform_enabled is true. See GET /v1/template-functions for available helpers; test it with POST /v1/subscriptions:testTemplate. A template that does not parse is rejected with 400; it is not rendered on save. Saving a new template records a version in the template history."`
	LabelFilters       map[string]string `json:"label_filters,omitempty" doc:"Key/value pairs that must ALL be present in an event's labels for this subscription to receive it. Empty means match every event of event_name. Same limits as event labels (20 pairs, 64-character keys, 256-character values)."`
	OnTransformError   string            `json:"on_transform_error,omitempty" enum:"fail,fallback" doc:"What happens when transform_template fails to render. fail (default): the delivery fails with error category template_error, is not retried automatically, and can be retried once the template is fixed; it does not affect the webhook's health. fallback: the default envelope payload is sent instead, and the error is still recorded on the delivery."`
	TemplateMissingKey string            `json:"template_missing_key,omitempty" enum:"error,zero" doc:"How transform_template reads a key the payload does not have. error (default): the render fails, so a field removed from the event schema cannot silently turn into \"<no value>\" in the body; read optional fields with index or dig (default only fills a key that is present but empty). zero: the key renders as \"<no value>\"."`
}

type createSubscriptionInput struct {
	Consumer string `path:"consumer" doc:"Consumer the webhook belongs to."`
	Body     createSubscriptionBody
}

type subscriptionIDInput struct {
	Consumer       string `path:"consumer" doc:"Consumer the subscription belongs to."`
	SubscriptionID string `path:"subscription_id" doc:"Subscription id (UUID)."`
}

type SubscriptionItem struct {
	SubscriptionID     string            `json:"subscription_id" doc:"Subscription id (UUID)."`
	Consumer           string            `json:"consumer" doc:"Consumer this subscription belongs to."`
	WebhookID          string            `json:"webhook_id" doc:"Webhook this subscription delivers to."`
	EventName          string            `json:"event_name" doc:"Event type name this subscription matches, or \"*\" for catch-all."`
	Headers            map[string]string `json:"headers,omitempty" doc:"Extra HTTP headers sent with deliveries from this subscription."`
	Method             string            `json:"method,omitempty" enum:"GET,POST,PUT,PATCH,DELETE" doc:"HTTP method used for deliveries from this subscription; absent means POST."`
	Timeout            int               `json:"timeout,omitempty" doc:"Request timeout override for this subscription's deliveries, in seconds; absent means the webhook's request_timeout_seconds."`
	TransformEnabled   bool              `json:"transform_enabled" doc:"Whether transform_template is rendered into the delivered payload."`
	TransformTemplate  string            `json:"transform_template,omitempty" doc:"Go text/template rendered against the event to produce the delivered body when transform_enabled is true."`
	LabelFilters       map[string]string `json:"label_filters,omitempty" doc:"Key/value pairs that must all be present in an event's labels for this subscription to receive it."`
	OnTransformError   string            `json:"on_transform_error" enum:"fail,fallback" doc:"What happens when transform_template fails to render. fail (default): the delivery fails with error category template_error, is not retried automatically, and can be retried once the template is fixed; it does not affect the webhook's health. fallback: the default envelope payload is sent instead, and the error is still recorded on the delivery."`
	TemplateMissingKey string            `json:"template_missing_key" enum:"error,zero" doc:"How transform_template reads a key the payload does not have. error (default): the render fails, so a field removed from the event schema cannot silently turn into \"<no value>\" in the body; read optional fields with index or dig (default only fills a key that is present but empty). zero: the key renders as \"<no value>\"."`
	Paused             bool              `json:"paused" doc:"Whether the subscription is paused. While paused, events still fan out to it, but each delivery is recorded with status paused and not attempted until retried. A pause never affects the webhook's health."`
	PausedAt           *string           `json:"paused_at,omitempty" format:"date-time" doc:"When the current pause began, RFC3339. Absent when not paused."`
	PausedReason       string            `json:"paused_reason,omitempty" doc:"Why the subscription was paused."`
	CreatedAt          string            `json:"created_at" format:"date-time" doc:"Creation timestamp, RFC3339."`
	UpdatedAt          string            `json:"updated_at" format:"date-time" doc:"Last-modified timestamp, RFC3339."`
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
	Consumer       string `path:"consumer" doc:"Consumer the subscription belongs to."`
	SubscriptionID string `path:"subscription_id" doc:"Subscription id (UUID)."`
	Body           struct {
		Reason string `json:"reason,omitempty" maxLength:"500" doc:"Why the subscription is paused, shown next to it, for example: receiver maintenance until Friday."`
	}
}

type resumeSubscriptionOutput struct {
	Body struct {
		SubscriptionItem
		PausedSince      *string `json:"paused_since,omitempty" format:"date-time" doc:"When the pause that just ended began, RFC3339; absent when the subscription was not paused. List or retry what was held with status=paused&subscription_id=...&created_after=<paused_since>."`
		PausedDeliveries int     `json:"paused_deliveries" doc:"Deliveries of this subscription currently in status paused. They are not sent automatically; retry them to deliver."`
	}
}

func toSubscriptionOutput(s *store.EventSubscription) *subscriptionOutput {
	return &subscriptionOutput{Body: toSubscriptionItem(s)}
}

// SubscriptionListParams are the query filters shared by the consumer-scoped
// and global subscription list routes.
type SubscriptionListParams struct {
	WebhookID string `query:"webhook_id,omitempty" doc:"Filter to subscriptions for one webhook (UUID). Takes precedence over event_name: when both are given, event_name is ignored."`
	EventName string `query:"event_name,omitempty" doc:"Filter to subscriptions with exactly this event_name (\"*\" matches only catch-all subscriptions)."`
	Limit     int32  `query:"limit" default:"50" minimum:"1" maximum:"1000" doc:"Maximum items to return, 1 to 1000."`
	Offset    int32  `query:"offset" default:"0" doc:"Number of items to skip, for pagination."`
}

type listSubscriptionsInput struct {
	Consumer string `path:"consumer" doc:"Consumer to list subscriptions in."`
	SubscriptionListParams
}

type listSubscriptionsGlobalInput struct {
	Consumer string `query:"consumer,omitempty" doc:"Filter to one consumer; omit to list subscriptions across all consumers."`
	SubscriptionListParams
}

type listSubscriptionsOutput struct {
	Body struct {
		Items      []SubscriptionItem `json:"items" doc:"Subscriptions, newest first."`
		Pagination PaginationOutput   `json:"pagination" doc:"Offset pagination with a total count."`
	}
}

// patchSubscriptionBody applies a partial update: only fields present in the
// request JSON are changed. webhook_id and event_name are immutable —
// delete and recreate the subscription to change either.
type patchSubscriptionBody struct {
	Headers            *map[string]string `json:"headers,omitempty" doc:"Replace all extra HTTP headers."`
	Method             *string            `json:"method,omitempty" doc:"Replace the HTTP method."`
	Timeout            *int               `json:"timeout,omitempty" doc:"Replace the request timeout override, in seconds (0 to 300; 0 uses the webhook's)."`
	TransformEnabled   *bool              `json:"transform_enabled,omitempty" doc:"Enable or disable payload transformation."`
	TransformTemplate  *string            `json:"transform_template,omitempty" doc:"Replace the transform template. A template that does not parse is rejected with 400. A change is recorded as a new version in the template history (the last 20 are kept)."`
	LabelFilters       *map[string]string `json:"label_filters,omitempty" doc:"Replace all label filters; an empty object matches every event."`
	OnTransformError   *string            `json:"on_transform_error,omitempty" enum:"fail,fallback" doc:"Replace what happens when the template fails to render."`
	TemplateMissingKey *string            `json:"template_missing_key,omitempty" enum:"error,zero" doc:"Replace how the template reads a missing key."`
	TemplateSource     string             `json:"template_source,omitempty" enum:"manual,ai_draft" doc:"How a changed transform_template was produced, recorded in the subscription's template history. Defaults to manual."`
	TemplateNotes      string             `json:"template_notes,omitempty" maxLength:"2000" doc:"Optional note stored with the template version, e.g. the AI drafter's summary."`
}

type templateVersionItem struct {
	VersionID string    `json:"version_id" doc:"Version id (UUID)."`
	Template  string    `json:"template" doc:"The template text as saved."`
	Source    string    `json:"source" enum:"manual,ai_draft" doc:"How the template was produced: manual or ai_draft."`
	Notes     string    `json:"notes,omitempty" doc:"Note stored with the version."`
	SavedBy   string    `json:"saved_by,omitempty" doc:"Name of the master key or access token that saved it; absent when auth is off."`
	CreatedAt time.Time `json:"created_at" doc:"When it was saved, RFC3339."`
	Current   bool      `json:"current" doc:"True for the newest version, which is the subscription's current template."`
}

type listTemplateVersionsInput struct {
	Consumer       string `path:"consumer" doc:"Consumer the subscription belongs to."`
	SubscriptionID string `path:"subscription_id" doc:"Subscription id (UUID)."`
	Limit          int    `query:"limit" minimum:"1" maximum:"20" default:"20" doc:"Maximum versions to return, newest first, 1 to 20."`
}

type listTemplateVersionsOutput struct {
	Body struct {
		Items []templateVersionItem `json:"items" doc:"Saved versions, newest first."`
	}
}

type patchSubscriptionInput struct {
	Consumer       string `path:"consumer" doc:"Consumer the subscription belongs to."`
	SubscriptionID string `path:"subscription_id" doc:"Subscription id (UUID)."`
	Body           patchSubscriptionBody
}

type testTemplateBody struct {
	EventName          string `json:"event_name" required:"true" doc:"Registered event type whose sample_payload (generated from its schema; empty without one) the template is rendered against."`
	Template           string `json:"template" required:"true" doc:"Go template to render, in the same syntax used by transform_template."`
	TemplateMissingKey string `json:"template_missing_key,omitempty" enum:"error,zero" required:"false" doc:"Render as a subscription with this template_missing_key would. Defaults to error, the subscription default."`
}

type testTemplateInput struct {
	Body testTemplateBody
}

type testTemplateOutput struct {
	Body struct {
		Rendered string `json:"rendered" doc:"The template output, rendered against the event type's sample payload with a placeholder event id and the current time."`
	}
}

func registerSubscriptionRoutes(api huma.API, svc webhooks.SubscriptionManager) {
	huma.Register(api, huma.Operation{
		OperationID: "createSubscription",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/subscriptions",
		Summary:     "Create a subscription linking a webhook to an event",
		Description: "Subscribes a webhook to an event type within a consumer, with an optional payload transform and label filters. Registering a webhook with events already creates one subscription per name; use this for additional, catch-all (\"*\") or per-subscription-template subscriptions. A non-empty transform_template is recorded as the first template version.\n\n" +
			"Not idempotent: 409 when the webhook already has a subscription for event_name. 404 when the webhook is not in this consumer. 400 for invalid headers, timeout or label filters, a sparrow.* event outside _sparrow, or a missing transform when the webhook has requires_transform.\n\n" +
			"Portal tokens: allowed, as POST /portal/api/subscriptions.",
		Errors:        []int{400, 404, 409},
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
		Description: "Returns one subscription: its webhook link, transform template and settings, label filters and pause state. 404 when it is not in this consumer.\n\nPortal tokens: allowed, as GET /portal/api/subscriptions/{subscription_id}.",
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
		Description: "Lists subscriptions in a consumer, newest first, optionally filtered by webhook or event type name. Offset paginated with total_count. 404 when webhook_id is given and the webhook is not in this consumer.\n\nPortal tokens: allowed, as GET /portal/api/subscriptions.",
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *listSubscriptionsInput) (*listSubscriptionsOutput, error) {
		return listSubscriptionsImpl(ctx, svc, in.Consumer, in.SubscriptionListParams)
	})

	huma.Register(api, huma.Operation{
		OperationID: "listSubscriptionsGlobal",
		Method:      http.MethodGet,
		Path:        "/v1/subscriptions",
		Summary:     "List subscriptions across all consumers",
		Description: "Lists subscriptions across every consumer with the same filters and pagination as the per-consumer route. Pass consumer to scope to one consumer.\n\nPortal tokens: not allowed.",
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *listSubscriptionsGlobalInput) (*listSubscriptionsOutput, error) {
		return listSubscriptionsImpl(ctx, svc, in.Consumer, in.SubscriptionListParams)
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateSubscription",
		Method:      http.MethodPatch,
		Path:        "/v1/consumers/{consumer}/subscriptions/{subscription_id}",
		Summary:     "Partially update a subscription",
		Description: "Partially updates a subscription's headers, method, timeout, transform, or label filters: only fields present in the body change, and map fields replace the whole map. The linked webhook_id and event_name cannot be changed; delete and recreate instead. A changed transform_template is recorded as a new template version with template_source and template_notes. Returns the updated subscription. Idempotent: repeating the same body records no new version.\n\n" +
			"400 for invalid values, or when the webhook has requires_transform and the result has no enabled template. 404 when it is not in this consumer.\n\nPortal tokens: allowed, as PATCH /portal/api/subscriptions/{subscription_id}.",
		Errors: []int{400, 404},
		Tags:   []string{"Subscriptions"},
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
		meta := webhooks.TemplateSaveMeta{Source: in.Body.TemplateSource, Notes: in.Body.TemplateNotes}
		// Record who saved it only for a real credential; with auth off the
		// middleware supplies an anonymous placeholder that is not worth storing.
		if p, ok := httpauth.FromContext(ctx); ok && (p.Root || p.TokenID != "") {
			meta.SavedBy = p.Name
		}
		if err := svc.UpdateSubscription(ctx, in.SubscriptionID, in.Consumer, headers, method, timeout, transformEnabled, transformTemplate, labelFilters, settings, meta); err != nil {
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
		Description: "Stops the subscription's deliveries from being attempted, for any reason (receiver maintenance, a template being fixed, an investigation). Events keep fanning out to it: each delivery is recorded with status paused and nothing is queued, so nothing is lost. Queued or retrying deliveries of the subscription are held as paused when their turn comes. A pause never affects the webhook's health. Returns the subscription (200), unlike webhook pause, which returns 204.\n\n" +
			"Idempotent: pausing again updates the reason and keeps the original pause time. 400 for a reason over 500 characters, 404 when it is not in this consumer.\n\nPortal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:pause.",
		Errors: []int{400, 404},
		Tags:   []string{"Subscriptions"},
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
		Description: "Lets new deliveries be attempted again. Deliveries recorded while paused stay paused and are never sent automatically: the response says how many there are and when the pause began, so you can retry all of them or only those since a time with the delivery retry endpoints. Idempotent: resuming a subscription that is not paused succeeds with no paused_since.\n\n" +
			"Portal tokens: allowed, as POST /portal/api/subscriptions/{subscription_id}:resume.",
		Errors: []int{404},
		Tags:   []string{"Subscriptions"},
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
		OperationID: "listSubscriptionTemplateVersions",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/subscriptions/{subscription_id}/templateVersions",
		Summary:     "List a subscription's saved template history",
		Description: "Returns the transform templates this subscription has been saved with, newest first; the newest is the current one. A version is recorded whenever a save changes the template, with how it was produced (manual or ai_draft), an optional note, and who saved it. The last 20 are kept. There is no restore endpoint: load a version into the editor and save the subscription, which records it as a new version.\n\nPortal tokens: allowed, as GET /portal/api/subscriptions/{subscription_id}/templateVersions.",
		Errors:      []int{404},
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *listTemplateVersionsInput) (*listTemplateVersionsOutput, error) {
		versions, err := svc.ListSubscriptionTemplateVersions(ctx, in.SubscriptionID, in.Consumer, in.Limit)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list template versions")
		}
		out := &listTemplateVersionsOutput{}
		out.Body.Items = make([]templateVersionItem, 0, len(versions))
		for i, v := range versions {
			out.Body.Items = append(out.Body.Items, templateVersionItem{
				VersionID: v.ID.String(), Template: v.Template, Source: v.Source, Notes: v.Notes,
				SavedBy: v.SavedBy, CreatedAt: v.CreatedAt, Current: i == 0,
			})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteSubscription",
		Method:        http.MethodDelete,
		Path:          "/v1/consumers/{consumer}/subscriptions/{subscription_id}",
		Summary:       "Delete a subscription",
		Description:   "Deletes the subscription and its template history. The webhook stops receiving that event's occurrences; delivery history is kept. Cannot be undone. A second delete returns 404.\n\nPortal tokens: allowed, as DELETE /portal/api/subscriptions/{subscription_id}.",
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
		Description: "Dry-runs a transform template against the named event type's sample payload, without creating a subscription or any delivery. Use this to iterate on transform_template before saving it. Stateless and idempotent.\n\n" +
			"400 when the template does not parse or fails to render (with template_missing_key error, a key the sample lacks is an error), 404 when the event type is not registered, 503 when the template engine is at capacity (retry).\n\nPortal tokens: allowed, as POST /portal/api/subscriptions:testTemplate.",
		Errors: []int{400, 404, 503},
		Tags:   []string{"Subscriptions"},
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
