package rest

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks"
)

// --- Register ---

type registerWebhookBody struct {
	Events        []string           `json:"events,omitempty" doc:"Event type names this webhook should receive; auto-creates one subscription per entry. Use \"*\" as the sole entry to subscribe to every event in the namespace. Omit or leave empty to register the webhook with no subscriptions, then attach them individually via POST .../subscriptions (e.g. to set a per-subscription transform_template)."`
	URL           string             `json:"url" required:"true" format:"uri" doc:"HTTPS/HTTP endpoint to POST deliveries to. Private, loopback, and cloud metadata addresses are rejected (SSRF protection)."`
	Headers       map[string]any     `json:"headers,omitempty" doc:"Static HTTP headers sent with every delivery to this webhook."`
	SecretHeaders map[string]string  `json:"secret_headers,omitempty" doc:"HTTP headers whose values are envelope-encrypted at rest and masked in every API response (e.g. an upstream auth token)."`
	Active        *bool              `json:"active,omitempty" doc:"Whether the webhook receives deliveries. Defaults to true. Inactive webhooks accept no new deliveries but keep their history."`
	Description   string             `json:"description,omitempty" doc:"Free-text note for humans, e.g. which system or team owns this endpoint."`
	HTTPConfig    *webhookHTTPConfig `json:"http_config,omitempty" doc:"Per-webhook HTTP delivery tuning (retries, timeouts, rate limit). Falls back to server defaults for any field left unset."`
	RateLimitRPS  *float64           `json:"rate_limit_rps,omitempty" doc:"Maximum sustained delivery rate to this webhook, in requests per second. Excess deliveries queue and are sent once the leaky bucket has capacity."`
	SignatureType string             `json:"signature_type,omitempty" enum:"hmac,ed25519," doc:"Which signature algorithm to require verification against. Every delivery is always dual-signed (HMAC-SHA256 and Ed25519, Standard Webhooks format); this only changes which one is treated as authoritative. Defaults to hmac."`
}

// webhookHTTPConfig tunes how deliveries to a single webhook are made and
// retried; every field is optional and falls back to a server default.
type webhookHTTPConfig struct {
	MaxRetries            int      `json:"max_retries,omitempty" doc:"Maximum delivery attempts before a delivery is marked failed."`
	RetryBackoffSeconds   int      `json:"retry_backoff_seconds,omitempty" doc:"Base delay between retry attempts, in seconds. Backoff grows exponentially from this value."`
	CaptureResponseBody   bool     `json:"capture_response_body,omitempty" doc:"Whether to store the endpoint's response body alongside each delivery attempt, for debugging."`
	FollowRedirects       bool     `json:"follow_redirects,omitempty" doc:"Whether to follow HTTP redirects returned by the endpoint."`
	VerifySSL             bool     `json:"verify_ssl,omitempty" doc:"Whether to verify the endpoint's TLS certificate. Disable only for trusted internal endpoints with self-signed certs."`
	RequestTimeoutSeconds int      `json:"request_timeout_seconds,omitempty" doc:"How long to wait for the endpoint to respond before treating the attempt as a timeout."`
	ExpectedStatusCodes   []int    `json:"expected_status_codes,omitempty" doc:"HTTP status codes treated as a successful delivery. Defaults to 2xx if left empty."`
	UserAgent             string   `json:"user_agent,omitempty" doc:"Custom User-Agent header sent with deliveries."`
	ContentType           string   `json:"content_type,omitempty" doc:"Content-Type header sent with deliveries. Defaults to application/json."`
	RateLimitRPS          *float64 `json:"rate_limit_rps,omitempty" doc:"Per-webhook delivery rate limit override, in requests per second."`
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
	Namespace string `path:"namespace"`
	Body      registerWebhookBody
}

type webhookOutput struct {
	Body WebhookOut
}

type namespaceOnlyInput struct {
	Namespace string `path:"namespace"`
}

type webhookIDInput struct {
	Namespace string `path:"namespace" doc:"Tenant namespace the webhook belongs to."`
	WebhookID string `path:"webhook_id" doc:"Webhook id (UUID)."`
}

// --- List ---

type listWebhooksInput struct {
	Namespace string `path:"namespace" doc:"Tenant namespace to list webhooks in."`
	WebhookID string `query:"webhook_id,omitempty" doc:"Filter to a single webhook by id."`
	Event     string `query:"event,omitempty" doc:"Filter to webhooks subscribed to this event type name."`
	Active    bool   `query:"active" default:"false" doc:"Only return active webhooks."`
	Health    string `query:"health,omitempty" enum:"healthy,degraded,unhealthy,unknown," doc:"Filter by computed health status."`
	Limit     int32  `query:"limit" default:"50" doc:"Maximum items to return."`
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
	Events        *[]string          `json:"events,omitempty" doc:"Replace the full set of subscribed event type names."`
	URL           *string            `json:"url,omitempty" doc:"Replace the delivery endpoint URL."`
	Headers       *map[string]string `json:"headers,omitempty" doc:"Replace the static headers sent with every delivery."`
	Timeout       *int               `json:"timeout,omitempty" doc:"Replace the request timeout in seconds (equivalent to http_config.request_timeout_seconds)."`
	Active        *bool              `json:"active,omitempty" doc:"Enable or disable the webhook."`
	Description   *string            `json:"description,omitempty" doc:"Replace the human-readable description."`
	SecretHeaders *map[string]string `json:"secret_headers,omitempty" doc:"Replace the encrypted, masked-on-read secret headers."`
	SignatureType *string            `json:"signature_type,omitempty" doc:"Replace the authoritative signature algorithm (hmac or ed25519)."`
	HTTPConfig    *webhookHTTPConfig `json:"http_config,omitempty" doc:"Replace the HTTP delivery configuration."`
}

type patchWebhookInput struct {
	Namespace string `path:"namespace"`
	WebhookID string `path:"webhook_id"`
	Body      patchWebhookBody
}

type emptyOutput struct {
	Status int
}

type namespaceStatsOutput struct {
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

func registerWebhookRoutes(api huma.API, d *Deps) {
	huma.Register(api, huma.Operation{
		OperationID:   "registerWebhook",
		Method:        http.MethodPost,
		Path:          "/v1/namespaces/{namespace}/webhooks",
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
			Namespace:     in.Namespace,
			Events:        in.Body.Events,
			URL:           in.Body.URL,
			Headers:       headers,
			SecretHeaders: in.Body.SecretHeaders,
			Active:        &active,
			Description:   in.Body.Description,
			HTTPConfig:    in.Body.HTTPConfig.toDomain(),
			RateLimitRPS:  in.Body.RateLimitRPS,
			SignatureType: in.Body.SignatureType,
		}
		reg, err := d.Svc.CreateWebhook(ctx, req)
		if err != nil {
			return nil, mapError(ctx, err, "failed to register webhook")
		}
		return &webhookOutput{Body: toWebhookOutFromDomain(reg)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listWebhooks",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/webhooks",
		Summary:     "List webhooks",
		Description: "Lists webhooks in a namespace, optionally filtered by id, subscribed event, active flag, or computed health status.",
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *listWebhooksInput) (*listWebhooksOutput, error) {
		limit, offset := in.Limit, in.Offset
		activeOnly := in.Active
		regs, total, err := d.Svc.ListWebhooks(ctx, in.Namespace, in.WebhookID, in.Event, activeOnly, limit, offset)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list webhooks")
		}
		if in.Health != "" {
			filtered := regs[:0]
			for _, r := range regs {
				if string(r.Health) == in.Health {
					filtered = append(filtered, r)
				}
			}
			regs = filtered
		}
		eventsMap := getWebhookEventsMap(ctx, d.Svc, regs)
		out := &listWebhooksOutput{}
		out.Body.Items = make([]WebhookOut, len(regs))
		for i, r := range regs {
			out.Body.Items[i] = toWebhookOut(r, eventsMap[r.ID.String()], d.Svc)
		}
		out.Body.Pagination = newPagination(limit, offset, total)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getWebhook",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/webhooks/{webhook_id}",
		Summary:     "Get a webhook by id",
		Description: "Fetches a single webhook's configuration, masked secrets, and current health.",
		Errors:      []int{404},
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *webhookIDInput) (*webhookOutput, error) {
		regs, _, err := d.Svc.ListWebhooks(ctx, in.Namespace, in.WebhookID, "", false, 1, 0)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get webhook")
		}
		if len(regs) == 0 {
			return nil, huma.Error404NotFound("webhook not found")
		}
		eventsMap := getWebhookEventsMap(ctx, d.Svc, regs)
		return &webhookOutput{Body: toWebhookOut(regs[0], eventsMap[regs[0].ID.String()], d.Svc)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "updateWebhook",
		Method:      http.MethodPatch,
		Path:        "/v1/namespaces/{namespace}/webhooks/{webhook_id}",
		Summary:     "Partially update a webhook",
		Description: "Merge-patches a webhook: only fields present in the request body are changed. Omit a field to leave it untouched.",
		Errors:      []int{400, 404},
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
			signatureType = *b.SignatureType
		}
		timeout := 0
		if b.Timeout != nil {
			timeout = *b.Timeout
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
			if c.RequestTimeoutSeconds != 0 {
				timeout = c.RequestTimeoutSeconds
			}
		}

		err := d.Svc.UpdateWebhookConfig(ctx, in.WebhookID, in.Namespace, events, url, headers, timeout, active, description, httpCfg, secretHeaders, signatureType, mask)
		if err != nil {
			return nil, mapError(ctx, err, "failed to update webhook")
		}
		regs, _, err := d.Svc.ListWebhooks(ctx, in.Namespace, in.WebhookID, "", false, 1, 0)
		if err != nil || len(regs) == 0 {
			return nil, mapError(ctx, err, "failed to reload webhook")
		}
		eventsMap := getWebhookEventsMap(ctx, d.Svc, regs)
		return &webhookOutput{Body: toWebhookOut(regs[0], eventsMap[regs[0].ID.String()], d.Svc)}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "deleteWebhook",
		Method:        http.MethodDelete,
		Path:          "/v1/namespaces/{namespace}/webhooks/{webhook_id}",
		Summary:       "Delete a webhook",
		Description:   "Permanently unregisters a webhook and cascade-deletes its subscriptions and delivery history. This cannot be undone.",
		Errors:        []int{404},
		Tags:          []string{"Webhooks"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *webhookIDInput) (*emptyOutput, error) {
		if err := d.Svc.UnregisterWebhook(ctx, in.WebhookID, in.Namespace); err != nil {
			return nil, mapError(ctx, err, "failed to unregister webhook")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "pauseWebhook",
		Method:        http.MethodPost,
		Path:          "/v1/namespaces/{namespace}/webhooks/{webhook_id}:pause",
		Summary:       "Pause a webhook",
		Description:   "Stops new deliveries to this webhook without deleting it. Events matching its subscriptions are still recorded but not delivered until resumed.",
		Errors:        []int{404},
		Tags:          []string{"Webhooks"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *webhookIDInput) (*emptyOutput, error) {
		if err := d.Svc.PauseWebhook(ctx, in.WebhookID, in.Namespace, ""); err != nil {
			return nil, mapError(ctx, err, "failed to pause webhook")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "resumeWebhook",
		Method:        http.MethodPost,
		Path:          "/v1/namespaces/{namespace}/webhooks/{webhook_id}:resume",
		Summary:       "Resume a paused webhook",
		Description:   "Re-enables deliveries to a previously paused webhook.",
		Errors:        []int{404},
		Tags:          []string{"Webhooks"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *webhookIDInput) (*emptyOutput, error) {
		if err := d.Svc.ResumeWebhook(ctx, in.WebhookID, in.Namespace); err != nil {
			return nil, mapError(ctx, err, "failed to resume webhook")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getNamespaceStats",
		Method:      http.MethodGet,
		Path:        "/v1/namespaces/{namespace}/stats",
		Summary:     "Get aggregate delivery statistics for a namespace",
		Description: "Returns webhook and delivery counts (total, active, successful, failed, pending, success rate) scoped to one namespace.",
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *namespaceOnlyInput) (*namespaceStatsOutput, error) {
		stats, err := d.Svc.GetNamespaceStats(ctx, in.Namespace)
		if err != nil {
			return nil, mapError(ctx, err, "failed to get namespace stats")
		}
		out := &namespaceStatsOutput{}
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
		fns := d.Svc.GetTemplateFunctions()
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
		Summary:     "Get aggregate delivery statistics across all namespaces",
		Description: "Returns the same counters as the per-namespace stats endpoint, aggregated across every namespace.",
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *struct{}) (*namespaceStatsOutput, error) {
		stats, err := d.Svc.GetNamespaceStats(ctx, "")
		if err != nil {
			return nil, mapError(ctx, err, "failed to get stats")
		}
		out := &namespaceStatsOutput{}
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
