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
	URL                string             `json:"url" required:"true" format:"uri" doc:"http or https endpoint deliveries are POSTed to. The host must resolve: localhost, *.internal, *.local, private, loopback and cloud metadata addresses are rejected (SSRF protection) unless allowed by SPARROW_ALLOW_PRIVATE_NETWORKS or SPARROW_ALLOWED_NETWORKS."`
	Headers            map[string]any     `json:"headers,omitempty" doc:"Static HTTP headers sent with every delivery to this webhook. Only string values are sent. Names must be valid HTTP tokens, values at most 8 KiB with no CR/LF; framing headers (Host, Content-Length, Transfer-Encoding, Connection, Keep-Alive, Proxy-Connection, TE, Trailer, Upgrade) are rejected. Stored and returned in plaintext: put credentials in secret_headers."`
	SecretHeaders      map[string]string  `json:"secret_headers,omitempty" doc:"HTTP headers sent with every delivery whose values are envelope-encrypted at rest and masked in every API response (e.g. an upstream auth token). Same name and value rules as headers."`
	Active             *bool              `json:"active,omitempty" doc:"Whether the webhook is sent deliveries. Defaults to true. An inactive (paused) webhook still gets deliveries recorded for new events, held with status paused and sent only when retried."`
	Description        string             `json:"description,omitempty" doc:"Free-text note for humans, e.g. which system or team owns this endpoint."`
	HTTPConfig         *webhookHTTPConfig `json:"http_config,omitempty" doc:"Per-webhook HTTP delivery tuning (retries, timeouts, rate limit). Falls back to server defaults for any field left unset."`
	RateLimitRPS       *float64           `json:"rate_limit_rps,omitempty" deprecated:"true" doc:"Deprecated: use http_config.rate_limit_rps. Shorthand for it, applied only when http_config.rate_limit_rps is not set (http_config wins when both are sent). Requests per second, must be positive."`
	SignatureType      string             `json:"signature_type,omitempty" enum:"hmac,ed25519" doc:"Signing scheme. Every delivery carries an HMAC-SHA256 signature (v1,) in the Standard Webhooks webhook-signature header; ed25519 also generates a signing keypair and adds an Ed25519 signature (v1a,) next to it. Defaults to hmac."`
	RequiresTransform  bool               `json:"requires_transform,omitempty" doc:"Set when the receiver only accepts a transformed payload (Slack, SendGrid, ...). Every subscription must then have transform_enabled and a transform_template: transform_template is required here when events is non-empty, and later subscription writes without one are rejected. Recipes set this. Defaults to false."`
	TransformTemplate  string             `json:"transform_template,omitempty" doc:"Transform template given to every subscription created from events, with transform_enabled set. A template that does not parse is rejected with 400. Omit to create them untransformed (not allowed with requires_transform)."`
	OnTransformError   string             `json:"on_transform_error,omitempty" enum:"fail,fallback" doc:"What happens when transform_template fails to render for one of the created subscriptions. fail (default) or fallback; see the subscription's on_transform_error."`
	TemplateSource     string             `json:"template_source,omitempty" enum:"manual,ai_draft" doc:"How transform_template was produced, recorded as the first version in each created subscription's template history. Defaults to manual."`
	TemplateNotes      string             `json:"template_notes,omitempty" maxLength:"2000" doc:"Optional note stored with that first template version, e.g. the AI drafter's summary."`
	TemplateMissingKey string             `json:"template_missing_key,omitempty" enum:"error,zero" doc:"How transform_template reads a key the payload does not have, for the created subscriptions. error (default) or zero; see the subscription's template_missing_key."`
}

// webhookHTTPConfig tunes how deliveries to a single webhook are made and
// retried; every field is optional and falls back to a server default.
type webhookHTTPConfig struct {
	MaxRetries            *int     `json:"max_retries,omitempty" doc:"Retries after the first attempt before a delivery is marked failed, so a delivery gets up to max_retries + 1 attempts. 0 means no retries. Between 0 and 10; defaults to 3."`
	RetryBackoffSeconds   int      `json:"retry_backoff_seconds,omitempty" doc:"Base retry delay, in seconds. The delay after attempt N is retry_backoff_seconds * 2^(N-1), capped at 24 hours. Between 1 and 3600; defaults to 60."`
	CaptureResponseBody   *bool    `json:"capture_response_body,omitempty" doc:"How much of the endpoint's response body is stored with each attempt: false (default) keeps up to 1 KiB, true up to SPARROW_MAX_CAPTURED_RESPONSE_BYTES (default 1 MiB)."`
	FollowRedirects       *bool    `json:"follow_redirects,omitempty" doc:"Whether to follow HTTP redirects returned by the endpoint. Each redirect target passes the same SSRF checks. Defaults to true."`
	VerifySSL             *bool    `json:"verify_ssl,omitempty" doc:"Whether to verify the endpoint's TLS certificate. Defaults to true. Set false only for trusted internal endpoints with self-signed certificates; SSRF and redirect checks still apply."`
	RequestTimeoutSeconds int      `json:"request_timeout_seconds,omitempty" doc:"How long to wait for the endpoint to respond before treating the attempt as a timeout, in seconds. Between 1 and 300; defaults to 30."`
	ExpectedStatusCodes   []int    `json:"expected_status_codes,omitempty" doc:"HTTP status codes treated as a successful delivery, each between 100 and 599. Defaults to 200, 201, 202 and 204."`
	UserAgent             string   `json:"user_agent,omitempty" doc:"Stored and returned, but deliveries do not use it yet: they always send User-Agent: Sparrow-Webhook/<server version>. To send another User-Agent, set it in headers. Defaults to Sparrow-Webhook/1.0."`
	ContentType           string   `json:"content_type,omitempty" doc:"Stored and returned, but deliveries do not use it yet: they send Content-Type: application/json. To send another Content-Type, set it in headers (or the subscription's headers). Must not be empty; defaults to application/json."`
	RateLimitRPS          *float64 `json:"rate_limit_rps,omitempty" doc:"Maximum delivery rate to this webhook, in requests per second (fractions allowed, e.g. 0.5). Must be positive; unset means no limit. Deliveries over the limit wait in the queue."`
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
	Consumer string `path:"consumer" doc:"Consumer to register the webhook under. Created implicitly on first use."`
	Body     registerWebhookBody
}

type webhookOutput struct {
	Body WebhookOut
}

type rotateWebhookSecretOutput struct {
	Body struct {
		WebhookSecret string `json:"webhook_secret" doc:"The new HMAC signing secret, in plaintext. Shown only in this response."`
	}
}

type listConsumersInput struct {
	Q     string `query:"q" maxLength:"255" doc:"Only consumers whose name contains this text, case-insensitive."`
	Limit int32  `query:"limit" default:"50" minimum:"1" maximum:"1000" doc:"Maximum names to return."`
}

// ConsumerItem is one consumer in a consumer search.
type ConsumerItem struct {
	Name string `json:"name" doc:"Consumer name."`
}

type listConsumersOutput struct {
	Body struct {
		Items []ConsumerItem `json:"items" doc:"Matching consumer names, sorted by name."`
	}
}

type consumerOnlyInput struct {
	Consumer string `path:"consumer" doc:"Consumer name."`
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
	Active    bool   `query:"active" default:"false" doc:"true returns only active (not paused) webhooks; false returns all."`
	Health    string `query:"health,omitempty" enum:"healthy,degraded,unhealthy,unknown" doc:"Filter by computed health status."`
	Limit     int32  `query:"limit" default:"50" minimum:"1" maximum:"1000" doc:"Maximum items to return, 1 to 1000."`
	Offset    int32  `query:"offset" default:"0" doc:"Number of items to skip, for pagination."`
}

type listWebhooksOutput struct {
	Body struct {
		Items      []WebhookOut     `json:"items" doc:"Webhooks, newest first."`
		Pagination PaginationOutput `json:"pagination" doc:"Offset pagination with a total count."`
	}
}

// --- Update (PATCH) ---

// patchWebhookBody applies a partial update: only fields present in the
// request JSON are changed, everything else is left untouched.
type patchWebhookBody struct {
	Events            *[]string          `json:"events,omitempty" doc:"Replace the full set of subscribed event type names. Every existing subscription of the webhook is deleted, with its transform and label filters, and one plain subscription is created per name. An empty list leaves subscriptions unchanged. Rejected with 409 while requires_transform is on; to keep transforms, add or delete subscriptions individually."`
	URL               *string            `json:"url,omitempty" doc:"Replace the delivery endpoint URL. Same rules as on register."`
	Headers           *map[string]string `json:"headers,omitempty" doc:"Replace all static headers sent with every delivery. Same rules as on register."`
	Timeout           *int               `json:"timeout,omitempty" deprecated:"true" doc:"Deprecated: use http_config.request_timeout_seconds. Shorthand for it, in seconds (1 to 300), applied only when http_config.request_timeout_seconds is not sent (http_config wins when both are sent)."`
	Active            *bool              `json:"active,omitempty" doc:"false pauses the webhook (like :pause), true resumes it and clears any auto-disable marker (like :resume, but without returning the held count)."`
	Description       *string            `json:"description,omitempty" doc:"Replace the human-readable description. An empty string leaves it unchanged."`
	SecretHeaders     *map[string]string `json:"secret_headers,omitempty" doc:"Merge-patch encrypted secret headers by name. Send a new value to replace one header, omit a key to leave it untouched, or send an empty string to remove it. An empty object removes them all."`
	SignatureType     *string            `json:"signature_type,omitempty" doc:"Change the signing scheme: hmac or ed25519. Switching to ed25519 generates a signing keypair; HMAC signing continues either way."`
	HTTPConfig        *webhookHTTPConfig `json:"http_config,omitempty" doc:"Update HTTP delivery settings. Only the fields you send change; the rest keep their current values. The merged result must satisfy the same limits as on create, or the request is rejected with 400."`
	RequiresTransform *bool              `json:"requires_transform,omitempty" doc:"Require (or stop requiring) a payload transform on every subscription. Turning it on fails with 409 while any subscription has no enabled transform_template. While on, events cannot be replaced in bulk (409): add or delete subscriptions individually."`
}

type patchWebhookInput struct {
	Consumer  string `path:"consumer" doc:"Consumer the webhook belongs to."`
	WebhookID string `path:"webhook_id" doc:"Webhook id (UUID)."`
	Body      patchWebhookBody
}

type emptyOutput struct {
	Status int
}

type consumerStatsOutput struct {
	Body struct {
		TotalWebhooks        int     `json:"total_webhooks" doc:"Total webhooks registered."`
		ActiveWebhooks       int     `json:"active_webhooks" doc:"Webhooks currently active (not paused)."`
		TotalDeliveries      int     `json:"total_deliveries" doc:"Delivery attempts recorded, all time. A retried delivery counts once per attempt. Updated by each health evaluation (SPARROW_HEALTH_EVAL_INTERVAL), so it can lag by that much."`
		SuccessfulDeliveries int     `json:"successful_deliveries" doc:"Attempts that succeeded, all time. Lags like total_deliveries."`
		FailedDeliveries     int     `json:"failed_deliveries" doc:"Attempts that failed, all time, including failures that were later retried successfully. Lags like total_deliveries."`
		PendingDeliveries    int     `json:"pending_deliveries" doc:"Deliveries waiting to be sent or retried right now (pending, sending or retrying)."`
		SuccessRate          float64 `json:"success_rate" doc:"successful_deliveries / total_deliveries, 0.0 to 1.0."`
	}
}

type templateFunctionItem struct {
	Name        string `json:"name" doc:"Function name, as used in a transform_template."`
	Description string `json:"description" doc:"Markdown documentation for the function, including usage and an example."`
}

type templateFunctionsOutput struct {
	Body struct {
		Items []templateFunctionItem `json:"items" doc:"Every helper function, in the engine's own order."`
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
		OperationID: "registerWebhook",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/webhooks",
		Summary:     "Register a webhook",
		Description: "Registers a new HTTP endpoint and creates one subscription per name in events (with transform_template, when given). Generates the HMAC signing secret and returns it in plaintext in http_config.webhook_secret; this response is the only time it is shown. Event names are not required to be registered event types yet.\n\n" +
			"Not idempotent: a second call with the same URL in the same consumer fails with 409 while the first webhook exists. 400 for an invalid URL (scheme, unresolvable or blocked host), invalid headers, http_config out of range, a sparrow.* event outside the _sparrow consumer, or requires_transform without transform_template.\n\n" +
			"Portal tokens: allowed, as POST /portal/api/webhooks.",
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
		reg, secret, err := svc.CreateWebhook(ctx, req)
		if err != nil {
			return nil, mapError(ctx, err, "failed to register webhook")
		}
		out := toWebhookOut(reg, in.Body.Events, svc)
		out.HTTPConfig.WebhookSecret = secret // shown once, plaintext, at creation
		return &webhookOutput{Body: out}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listWebhooks",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/webhooks",
		Summary:     "List webhooks",
		Description: "Lists the webhooks of one consumer, newest first, optionally filtered by id, subscribed event, active flag, or health status. Secrets are masked. Offset paginated with total_count. Deleted webhooks are never listed.\n\nPortal tokens: allowed, as GET /portal/api/webhooks.",
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
		Description: "Returns one webhook's configuration, subscribed events, masked secrets, and current health label. 404 when it does not exist in this consumer or was deleted.\n\nPortal tokens: allowed, as GET /portal/api/webhooks/{webhook_id}.",
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
		Description: "Partially updates a webhook: only fields present in the body change. Within http_config only the sent fields change, and the merged result is validated as on register (400 if out of range). Sending events replaces every subscription of the webhook (see events). Setting active toggles delivery like :pause/:resume but never fails when the state is already as requested. Returns the updated webhook. Idempotent: repeating the same body leaves the same state.\n\n" +
			"409 when turning requires_transform on while a subscription has no enabled transform, or replacing events while it is on. The deprecated top-level timeout is applied only when http_config.request_timeout_seconds is not sent.\n\n" +
			"Portal tokens: allowed, as PATCH /portal/api/webhooks/{webhook_id}.",
		Errors: []int{400, 404, 409},
		Tags:   []string{"Webhooks"},
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
		var httpCfg *webhooks.WebhookHTTPConfig

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
			httpCfg = &webhooks.WebhookHTTPConfig{
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
			if c.RateLimitRPS != nil {
				mask = append(mask, "http_config.rate_limit_rps")
			}
		}
		// Top-level timeout is shorthand for http_config.request_timeout_seconds.
		if b.Timeout != nil {
			if httpCfg == nil {
				mask = append(mask, "http_config")
				httpCfg = &webhooks.WebhookHTTPConfig{}
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
		OperationID: "deleteWebhook",
		Method:      http.MethodDelete,
		Path:        "/v1/consumers/{consumer}/webhooks/{webhook_id}",
		Summary:     "Delete a webhook",
		Description: "Deletes a webhook (a soft delete): it stops receiving events and no longer appears in webhook lists, and its URL can be registered again in the consumer. Its subscriptions and delivery history are kept, and its deliveries stay listed until retention removes them. Queued deliveries for it fail with \"Webhook was deleted\". This cannot be undone. A second delete returns 404.\n\n" +
			"Portal tokens: allowed, as DELETE /portal/api/webhooks/{webhook_id}.",
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
		OperationID: "rotateWebhookSecret",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/webhooks/{webhook_id}:rotateSecret",
		Summary:     "Rotate a webhook's signing secret",
		Description: "Replaces the webhook's HMAC signing secret with a newly generated one and returns it in plaintext. This response is the only time the new secret is shown; it is masked on every later read. There is no overlap window: every attempt made after the rotation, including retries of older deliveries, is signed with the new secret only, so update the receiver right away. Not idempotent: each call generates a new secret. The Ed25519 key, if any, is not rotated.\n\n" +
			"Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:rotateSecret.",
		Errors: []int{404},
		Tags:   []string{"Webhooks"},
	}, func(ctx context.Context, in *webhookIDInput) (*rotateWebhookSecretOutput, error) {
		secret, err := svc.RotateWebhookSecret(ctx, in.WebhookID, in.Consumer)
		if err != nil {
			return nil, mapError(ctx, err, "failed to rotate webhook secret")
		}
		out := &rotateWebhookSecretOutput{}
		out.Body.WebhookSecret = secret
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "pauseWebhook",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/webhooks/{webhook_id}:pause",
		Summary:     "Pause a webhook",
		Description: "Stops deliveries to this webhook without deleting it. Returns 204 with no body. Events keep fanning out to it: each new delivery, and any queued or retrying one, is held with status paused until it is retried after resuming. Holding never affects the webhook's health. 409 when the webhook is already paused.\n\n" +
			"Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:pause.",
		Errors:        []int{404, 409},
		Tags:          []string{"Webhooks"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *webhookIDInput) (*emptyOutput, error) {
		if err := svc.PauseWebhook(ctx, in.WebhookID, in.Consumer, ""); err != nil {
			return nil, mapError(ctx, err, "failed to pause webhook")
		}
		return &emptyOutput{Status: http.StatusNoContent}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "resumeWebhook",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/webhooks/{webhook_id}:resume",
		Summary:     "Resume a paused webhook",
		Description: "Re-enables deliveries to a webhook paused by an operator or auto-disabled, and clears auto_disabled_at/auto_disabled_reason. Deliveries held while it was paused stay paused and are never sent automatically: the response says how many there are, so you can retry them with status=paused&webhook_id=... and the delivery retry endpoints. 409 when the webhook is already active.\n\n" +
			"Portal tokens: allowed, as POST /portal/api/webhooks/{webhook_id}:resume.",
		Errors: []int{404, 409},
		Tags:   []string{"Webhooks"},
	}, func(ctx context.Context, in *webhookIDInput) (*resumeWebhookOutput, error) {
		res, err := svc.ResumeWebhook(ctx, in.WebhookID, in.Consumer)
		if err != nil {
			return nil, mapError(ctx, err, "failed to resume webhook")
		}
		out := &resumeWebhookOutput{}
		out.Body.PausedDeliveries = res.PausedDeliveries
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listConsumers",
		Method:      http.MethodGet,
		Path:        "/v1/consumers",
		Summary:     "Search consumer names",
		Description: "Lists the consumers that own a webhook or an event, sorted by name, optionally only those whose name contains q (case-insensitive). Consumers are created implicitly by registering a webhook or pushing an event under them; this is how clients find them, e.g. for a consumer picker. Not paginated: returns at most limit names.\n\nPortal tokens: not allowed.",
		Tags:        []string{"Webhooks"},
	}, func(ctx context.Context, in *listConsumersInput) (*listConsumersOutput, error) {
		names, err := svc.ListConsumers(ctx, in.Q, in.Limit)
		if err != nil {
			return nil, mapError(ctx, err, "failed to list consumers")
		}
		out := &listConsumersOutput{}
		out.Body.Items = make([]ConsumerItem, 0, len(names))
		for _, n := range names {
			out.Body.Items = append(out.Body.Items, ConsumerItem{Name: n})
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getConsumerStats",
		Method:      http.MethodGet,
		Path:        "/v1/consumers/{consumer}/stats",
		Summary:     "Get aggregate delivery statistics for a consumer",
		Description: "Returns webhook and delivery-attempt counts (total, active, successful, failed, pending, success rate) for one consumer. Attempt counts come from the health evaluator and lag by up to SPARROW_HEALTH_EVAL_INTERVAL; pending is live.\n\nPortal tokens: allowed, as GET /portal/api/stats.",
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
		Description: "Lists the helper functions available to subscription transform templates (on top of Go text/template built-ins), each with Markdown documentation. Static: the same for every caller.\n\nPortal tokens: allowed, as GET /portal/api/template-functions.",
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
		Description: "Returns the same counters as the per-consumer stats endpoint, summed across every consumer, with the same lag.\n\nPortal tokens: not allowed.",
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

type resumeWebhookOutput struct {
	Body struct {
		PausedDeliveries int `json:"paused_deliveries" doc:"Deliveries held as paused for this webhook. They are not sent automatically; retry them to deliver."`
	}
}
