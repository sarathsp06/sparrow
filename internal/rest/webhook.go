package rest

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/pkg/access/httpauth"
)

// --- Register ---

type registerWebhookBody struct {
	Events             []string           `json:"events,omitempty" doc:"Event type names this webhook should receive; auto-creates one subscription per entry. Use \"*\" as the sole entry to subscribe to every event in the consumer. Omit or leave empty to register the webhook with no subscriptions, then attach them individually via POST .../subscriptions (e.g. to set a per-subscription transform_template)."`
	URL                string             `json:"url" required:"true" format:"uri" doc:"HTTPS/HTTP endpoint to POST deliveries to. Private, loopback, and cloud metadata addresses are rejected (SSRF protection)."`
	Headers            map[string]any     `json:"headers,omitempty" doc:"Static HTTP headers sent with every delivery to this webhook."`
	SecretHeaders      map[string]string  `json:"secret_headers,omitempty" doc:"HTTP headers whose values are envelope-encrypted at rest and masked in every API response (e.g. an upstream auth token)."`
	Active             *bool              `json:"active,omitempty" doc:"Whether the webhook receives deliveries. Defaults to true. Inactive webhooks accept no new deliveries but keep their history."`
	Description        string             `json:"description,omitempty" doc:"Free-text note for humans, e.g. which system or team owns this endpoint."`
	HTTPConfig         *webhookHTTPConfig `json:"http_config,omitempty" doc:"Per-webhook HTTP delivery tuning (retries, timeouts, rate limit). Falls back to server defaults for any field left unset."`
	RateLimitRPS       *float64           `json:"rate_limit_rps,omitempty" doc:"Maximum sustained delivery rate to this webhook, in requests per second. Excess deliveries queue and are sent once the leaky bucket has capacity."`
	SignatureType      string             `json:"signature_type,omitempty" enum:"hmac,ed25519," doc:"Which signature algorithm to require verification against. Deliveries are HMAC-SHA256 signed (v1,) by default; once signature_type is set to ed25519 (which generates a signing keypair), later deliveries are dual-signed (v1, and v1a,, Standard Webhooks format). Defaults to hmac."`
	RequiresTransform  bool               `json:"requires_transform,omitempty" doc:"Set when the receiver only accepts a transformed payload (Slack, SendGrid, ...). Every subscription must then have transform_enabled and a transform_template: transform_template is required here when events is non-empty, and later subscription writes without one are rejected. Recipes set this. Defaults to false."`
	TransformTemplate  string             `json:"transform_template,omitempty" doc:"Transform template given to every subscription created from events, with transform_enabled set. Omit to create them untransformed (not allowed with requires_transform)."`
	OnTransformError   string             `json:"on_transform_error,omitempty" enum:"fail,fallback," doc:"What happens when transform_template fails to render for one of the created subscriptions. fail (default) or fallback; see the subscription's on_transform_error."`
	TemplateSource     string             `json:"template_source,omitempty" enum:"manual,ai_draft," doc:"How transform_template was produced, recorded as the first version in each created subscription's template history. Defaults to manual."`
	TemplateNotes      string             `json:"template_notes,omitempty" maxLength:"2000" doc:"Optional note stored with that first template version, e.g. the AI drafter's summary."`
	TemplateMissingKey string             `json:"template_missing_key,omitempty" enum:"error,zero," doc:"How transform_template reads a key the payload does not have, for the created subscriptions. error (default) or zero; see the subscription's template_missing_key."`
}

// webhookHTTPConfig tunes how deliveries to a single webhook are made and
// retried; every field is optional and falls back to a server default.
type webhookHTTPConfig struct {
	MaxRetries            *int     `json:"max_retries,omitempty" doc:"Maximum delivery attempts before a delivery is marked failed. 0 means no retries. Between 0 and 10; defaults to 3."`
	RetryBackoffSeconds   int      `json:"retry_backoff_seconds,omitempty" doc:"Base delay between retry attempts, in seconds. Backoff grows exponentially from this value. Between 1 and 3600; defaults to 60."`
	CaptureResponseBody   *bool    `json:"capture_response_body,omitempty" doc:"Whether to store the endpoint's response body alongside each delivery attempt, for debugging."`
	FollowRedirects       *bool    `json:"follow_redirects,omitempty" doc:"Whether to follow HTTP redirects returned by the endpoint."`
	VerifySSL             *bool    `json:"verify_ssl,omitempty" doc:"Whether to verify the endpoint's TLS certificate. Defaults to true. Set false only for trusted internal endpoints with self-signed certificates; SSRF and redirect checks still apply."`
	RequestTimeoutSeconds int      `json:"request_timeout_seconds,omitempty" doc:"How long to wait for the endpoint to respond before treating the attempt as a timeout, in seconds. Between 1 and 300; defaults to 30."`
	ExpectedStatusCodes   []int    `json:"expected_status_codes,omitempty" doc:"HTTP status codes treated as a successful delivery, each between 100 and 599. Defaults to 200, 201, 202 and 204."`
	UserAgent             string   `json:"user_agent,omitempty" doc:"Custom User-Agent header sent with deliveries."`
	ContentType           string   `json:"content_type,omitempty" doc:"Content-Type header sent with deliveries. Defaults to application/json."`
	RateLimitRPS          *float64 `json:"rate_limit_rps,omitempty" doc:"Per-webhook delivery rate limit override, in requests per second. Must be positive; unset means no limit."`
}

func (c *webhookHTTPConfig) toDomain() *webhooks.WebhookHTTPConfig {
	if c == nil {
		return nil
	}
	return &webhooks.WebhookHTTPConfig{
		MaxRetries:            c.MaxRetries,
		RetryBackoffSeconds:   c.RetryBackoffSeconds,
		CaptureResponseBody:   c.CaptureResponseBody,
		FollowRedirects:       c.FollowRedirects,
		VerifySSL:             c.VerifySSL,
		RequestTimeoutSeconds: c.RequestTimeoutSeconds,
		ExpectedStatusCodes:   webhooks.IntArray(c.ExpectedStatusCodes),
		UserAgent:             c.UserAgent,
		ContentType:           c.ContentType,
		RateLimitRPS:          c.RateLimitRPS,
	}
}

type registerWebhookInput struct {
	Consumer string `path:"consumer"`
	Body     registerWebhookBody
}

type webhookOutput struct {
	Body WebhookOut
}

type consumerOnlyInput struct {
	Consumer string `path:"consumer"`
}

type webhookIDInput struct {
	Consumer  string `path:"consumer" doc:"Tenant consumer the webhook belongs to."`
	WebhookID string `path:"webhook_id" doc:"Webhook id (UUID)."`
}

// --- List ---

type listWebhooksInput struct {
	Consumer  string `path:"consumer" doc:"Tenant consumer to list webhooks in."`
	WebhookID string `query:"webhook_id,omitempty" doc:"Filter to a single webhook by id."`
	Event     string `query:"event,omitempty" doc:"Filter to webhooks subscribed to this event type name."`
	Active    bool   `query:"active" default:"false" doc:"Only return active webhooks."`
	Health    string `query:"health,omitempty" enum:"healthy,degraded,unhealthy,unknown," doc:"Filter by computed health status."`
	Limit     int32  `query:"limit" default:"50" minimum:"1" maximum:"1000" doc:"Maximum items to return."`
	Offset    int32  `query:"offset" default:"0" doc:"Number of items to skip, for pagination."`
}

type listWebhooksOutput struct {
	Body struct {
		Items      []WebhookOut     `json:"items"`
		Pagination PaginationOutput `json:"pagination"`
	}
}

// --- Update (PATCH) ---

// patchWebhookBody applies a partial update: only fields present in the
// request JSON are changed, everything else is left untouched.
type patchWebhookBody struct {
	Events            *[]string          `json:"events,omitempty" doc:"Replace the full set of subscribed event type names."`
	URL               *string            `json:"url,omitempty" doc:"Replace the delivery endpoint URL."`
	Headers           *map[string]string `json:"headers,omitempty" doc:"Replace the static headers sent with every delivery."`
	Timeout           *int               `json:"timeout,omitempty" doc:"Replace the request timeout in seconds (equivalent to http_config.request_timeout_seconds)."`
	Active            *bool              `json:"active,omitempty" doc:"Enable or disable the webhook."`
	Description       *string            `json:"description,omitempty" doc:"Replace the human-readable description."`
	SecretHeaders     *map[string]string `json:"secret_headers,omitempty" doc:"Merge-patch encrypted secret headers by name. Send a new value to replace one header, omit a key to leave it untouched, or send an empty string to remove it."`
	SignatureType     *string            `json:"signature_type,omitempty" doc:"Replace the authoritative signature algorithm (hmac or ed25519)."`
	HTTPConfig        *webhookHTTPConfig `json:"http_config,omitempty" doc:"Update HTTP delivery settings. Only the fields you send change; the rest keep their current values. The merged result must satisfy the same limits as on create, or the request is rejected with 400."`
	RequiresTransform *bool              `json:"requires_transform,omitempty" doc:"Require (or stop requiring) a payload transform on every subscription. Turning it on fails with 409 while any subscription has no enabled transform_template. While on, events cannot be replaced in bulk (409): add or delete subscriptions individually."`
}

type patchWebhookInput struct {
	Consumer  string `path:"consumer"`
	WebhookID string `path:"webhook_id"`
	Body      patchWebhookBody
}

type emptyOutput struct {
	Status int
}

type consumerStatsOutput struct {
	Body struct {
		TotalWebhooks        int     `json:"total_webhooks" doc:"Total webhooks registered."`
		ActiveWebhooks       int     `json:"active_webhooks" doc:"Webhooks currently active (not paused)."`
		TotalDeliveries      int     `json:"total_deliveries" doc:"Total delivery attempts recorded."`
		SuccessfulDeliveries int     `json:"successful_deliveries" doc:"Deliveries that succeeded."`
		FailedDeliveries     int     `json:"failed_deliveries" doc:"Deliveries that failed."`
		PendingDeliveries    int     `json:"pending_deliveries" doc:"Deliveries pending or retrying."`
		SuccessRate          float64 `json:"success_rate" doc:"Overall success rate, 0.0 to 1.0."`
	}
}

type templateFunctionItem struct {
	Name        string `json:"name" doc:"Function name, as used in a transform_template."`
	Description string `json:"description" doc:"Markdown documentation for the function, including usage and an example."`
}

type templateFunctionsOutput struct {
	Body struct {
		Items []templateFunctionItem `json:"items"`
	}
}

// webhookRouteService is the service slice webhook routes consume: webhook
// lifecycle, subscription lookup for event lists, and secret masking.
type webhookRouteService interface {
	webhooks.WebhookManager
	webhooks.SubscriptionManager
	webhooks.SecretRevealer
}

func registerWebhookRoutes(api huma.API, svc webhookRouteService) {
	huma.Register(api, huma.Operation{
		OperationID:   "registerWebhook",
		Method:        http.MethodPost,
		Path:          "/v1/consumers/{consumer}/webhooks",
		Summary:       "Register a webhook",
		Description:   "Registers a new HTTP endpoint to receive deliveries and auto-creates a subscription for each listed event type. Returns the plaintext webhook secret once — it is masked on every subsequent read.",
		Errors:        []int{400, 409},
		Tags:          []string{"Webhooks"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *registerWebhookInput) (*webhookOutput, error) {
		headers := make(map[string]any, len(in.Body.Headers))
		for k, v := range in.Body.Headers {
			headers[k] = v
		}
		active := true
		if in.Body.Active != nil {
			active = *in.Body.Active
		}
		req := webhooks.WebhookRegistrationRequest{
			Consumer:      in.Consumer,
			Events:        in.Body.Events,
			URL:           in.Body.URL,
			Headers:       headers,
			SecretHeaders: in.Body.SecretHeaders,
			Active:        &active,
			Description:   in.Body.Description,
			HTTPConfig:    in.Body.HTTPConfig.toDomain(),
			RateLimitRPS:  in.Body.RateLimitRPS,
			SignatureType: in.Body.SignatureType,

			RequiresTransform: in.Body.RequiresTransform,
			TransformTemplate: in.Body.TransformTemplate,
			TemplateSettings: webhooks.SubscriptionTemplateSettings{
				OnTransformError:   in.Body.OnTransformError,
				TemplateMissingKey: in.Body.TemplateMissingKey,
			},
			TemplateMeta: webhooks.TemplateSaveMeta{Source: in.Body.TemplateSource, Notes: in.Body.TemplateNotes},
		}
		// Record who saved the template only for a real credential, as the
		// subscription endpoints do.
		if p, ok := httpauth.FromContext(ctx); ok && (p.Root || p.TokenID != "") {
			req.TemplateMeta.SavedBy = p.Name
		}
		reg, err := svc.CreateWebhook(ctx, req)
		if err != nil {
			return nil, mapError(ctx, err, "failed to register webhook")
		}
		return &webhookOutput{Body: toWebhookOutFromDomain(reg, svc)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listWebhooks",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/webhooks",
		Summary:     "List webhooks",
		Description: "Lists webhooks in a consumer, optionally filtered by id, subscribed event, active flag, or computed health status.",
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *listWebhooksInput) (*listWebhooksOutput, error) {
		limit, offset := in.Limit, in.Offset
		activeOnly := in.Active
		regs, total, err := svc.ListWebhooks(ctx, in.Consumer, in.WebhookID, in.Event, activeOnly, in.Health, limit, offset)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list webhooks")
		}
		eventsMap := getWebhookEventsMap(ctx, svc, regs)
		out := &listWebhooksOutput{}
		out.Body.Items = make([]WebhookOut, len(regs))
		for i, r := range regs {
			out.Body.Items[i] = toWebhookOut(r, eventsMap[r.ID.String()], svc)
		}
		out.Body.Pagination = newPagination(limit, offset, total)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getWebhook",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/webhooks/{webhook_id}",
		Summary:     "Get a webhook by id",
		Description: "Fetches a single webhook's configuration, masked secrets, and current health.",
		Errors:      []int{404},
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *webhookIDInput) (*webhookOutput, error) {
		regs, _, err := svc.ListWebhooks(ctx, in.Consumer, in.WebhookID, "", false, "", 1, 0)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get webhook")
		}
		if len(regs) == 0 {
			return nil, huma.Error404NotFound("webhook not found")
		}
		eventsMap := getWebhookEventsMap(ctx, svc, regs)
		return &webhookOutput{Body: toWebhookOut(regs[0], eventsMap[regs[0].ID.String()], svc)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateWebhook",
		Method:      http.MethodPatch,
		Path:        "/v1/consumers/{consumer}/webhooks/{webhook_id}",
		Summary:     "Partially update a webhook",
		Description: "Merge-patches a webhook: only fields present in the request body are changed. Omit a field to leave it untouched.",
		Errors:      []int{400, 404, 409},
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *patchWebhookInput) (*webhookOutput, error) {
		b := in.Body
		var mask []string
		var events []string
		var url string
		var headers map[string]string
		var active bool
		var description string
		var secretHeaders map[string]string
		var signatureType string
		var requiresTransform bool
		var httpCfg *webhooks.HTTPConfigUpdate

		if b.Events != nil {
			mask = append(mask, "events")
			events = *b.Events
		}
		if b.URL != nil {
			mask = append(mask, "url")
			url = *b.URL
		}
		if b.Headers != nil {
			mask = append(mask, "headers")
			headers = *b.Headers
		}
		if b.Active != nil {
			mask = append(mask, "active")
			active = *b.Active
		}
		if b.Description != nil {
			mask = append(mask, "description")
			description = *b.Description
		}
		if b.SecretHeaders != nil {
			mask = append(mask, "secret_headers")
			secretHeaders = *b.SecretHeaders
		}
		if b.SignatureType != nil {
			mask = append(mask, "signature_type")
			signatureType = *b.SignatureType
		}
		if b.RequiresTransform != nil {
			mask = append(mask, "requires_transform")
			requiresTransform = *b.RequiresTransform
		}
		if b.HTTPConfig != nil {
			mask = append(mask, "http_config")
			c := b.HTTPConfig
			httpCfg = &webhooks.HTTPConfigUpdate{
				MaxRetries:            c.MaxRetries,
				RetryBackoffSeconds:   c.RetryBackoffSeconds,
				CaptureResponseBody:   c.CaptureResponseBody,
				FollowRedirects:       c.FollowRedirects,
				VerifySSL:             c.VerifySSL,
				RequestTimeoutSeconds: c.RequestTimeoutSeconds,
				ExpectedStatusCodes:   c.ExpectedStatusCodes,
				UserAgent:             c.UserAgent,
				ContentType:           c.ContentType,
				RateLimitRPS:          c.RateLimitRPS,
			}
			if c.RateLimitRPS != nil {
				mask = append(mask, "http_config.rate_limit_rps")
			}
		}
		// Top-level timeout is shorthand for http_config.request_timeout_seconds.
		if b.Timeout != nil {
			if httpCfg == nil {
				mask = append(mask, "http_config")
				httpCfg = &webhooks.HTTPConfigUpdate{}
			}
			if httpCfg.RequestTimeoutSeconds == 0 {
				httpCfg.RequestTimeoutSeconds = *b.Timeout
			}
		}

		err := svc.UpdateWebhookConfig(ctx, in.WebhookID, in.Consumer, events, url, headers, active, description, httpCfg, secretHeaders, signatureType, requiresTransform, mask)
		if err != nil {
			return nil, mapError(ctx, err, "failed to update webhook")
		}
		regs, _, err := svc.ListWebhooks(ctx, in.Consumer, in.WebhookID, "", false, "", 1, 0)
		if err != nil {
			return nil, mapError(ctx, err, "failed to reload webhook")
		}
		if len(regs) == 0 {
			return nil, huma.Error404NotFound("webhook not found")
		}
		eventsMap := getWebhookEventsMap(ctx, svc, regs)
		return &webhookOutput{Body: toWebhookOut(regs[0], eventsMap[regs[0].ID.String()], svc)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteWebhook",
		Method:        http.MethodDelete,
		Path:          "/v1/consumers/{consumer}/webhooks/{webhook_id}",
		Summary:       "Delete a webhook",
		Description:   "Permanently unregisters a webhook and cascade-deletes its subscriptions and delivery history. This cannot be undone.",
		Errors:        []int{404},
		Tags:          []string{"Webhooks"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *webhookIDInput) (*emptyOutput, error) {
		if err := svc.UnregisterWebhook(ctx, in.WebhookID, in.Consumer); err != nil {
			return nil, mapError(ctx, err, "failed to unregister webhook")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "pauseWebhook",
		Method:        http.MethodPost,
		Path:          "/v1/consumers/{consumer}/webhooks/{webhook_id}:pause",
		Summary:       "Pause a webhook",
		Description:   "Stops new deliveries to this webhook without deleting it. Events matching its subscriptions are still recorded but not delivered until resumed.",
		Errors:        []int{404},
		Tags:          []string{"Webhooks"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *webhookIDInput) (*emptyOutput, error) {
		if err := svc.PauseWebhook(ctx, in.WebhookID, in.Consumer, ""); err != nil {
			return nil, mapError(ctx, err, "failed to pause webhook")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "resumeWebhook",
		Method:        http.MethodPost,
		Path:          "/v1/consumers/{consumer}/webhooks/{webhook_id}:resume",
		Summary:       "Resume a paused webhook",
		Description:   "Re-enables deliveries to a previously paused webhook.",
		Errors:        []int{404},
		Tags:          []string{"Webhooks"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *webhookIDInput) (*emptyOutput, error) {
		if err := svc.ResumeWebhook(ctx, in.WebhookID, in.Consumer); err != nil {
			return nil, mapError(ctx, err, "failed to resume webhook")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getConsumerStats",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/stats",
		Summary:     "Get aggregate delivery statistics for a consumer",
		Description: "Returns webhook and delivery counts (total, active, successful, failed, pending, success rate) scoped to one consumer.",
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *consumerOnlyInput) (*consumerStatsOutput, error) {
		stats, err := svc.GetConsumerStats(ctx, in.Consumer)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get consumer stats")
		}
		out := &consumerStatsOutput{}
		out.Body.TotalWebhooks = stats.TotalWebhooks
		out.Body.ActiveWebhooks = stats.ActiveWebhooks
		out.Body.TotalDeliveries = stats.TotalDeliveries
		out.Body.SuccessfulDeliveries = stats.SuccessfulDeliveries
		out.Body.FailedDeliveries = stats.FailedDeliveries
		out.Body.PendingDeliveries = stats.PendingDeliveries
		out.Body.SuccessRate = stats.SuccessRate
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getTemplateFunctions",
		Method:      http.MethodGet,
		Path:        "/v1/template-functions",
		Summary:     "List available payload-transformation template functions",
		Description: "Lists the Go template helper functions available to subscription transform templates (e.g. string/JSON helpers), each with its documentation.",
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *struct{}) (*templateFunctionsOutput, error) {
		fns := svc.GetTemplateFunctions()
		out := &templateFunctionsOutput{}
		out.Body.Items = make([]templateFunctionItem, 0, len(fns))
		for _, f := range fns {
			out.Body.Items = append(out.Body.Items, templateFunctionItem{Name: f.Name, Description: f.Description})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getGlobalStats",
		Method:      http.MethodGet,
		Path:        "/v1/stats",
		Summary:     "Get aggregate delivery statistics across all consumers",
		Description: "Returns the same counters as the per-consumer stats endpoint, aggregated across every consumer.",
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *struct{}) (*consumerStatsOutput, error) {
		stats, err := svc.GetConsumerStats(ctx, "")
		if err != nil {
			return nil, mapError(ctx, err, "failed to get stats")
		}
		out := &consumerStatsOutput{}
		out.Body.TotalWebhooks = stats.TotalWebhooks
		out.Body.ActiveWebhooks = stats.ActiveWebhooks
		out.Body.TotalDeliveries = stats.TotalDeliveries
		out.Body.SuccessfulDeliveries = stats.SuccessfulDeliveries
		out.Body.FailedDeliveries = stats.FailedDeliveries
		out.Body.PendingDeliveries = stats.PendingDeliveries
		out.Body.SuccessRate = stats.SuccessRate
		return out, nil
	})
}
