package rest

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/sarathsp06/sparrow/internal/webhooks"
)

// APIVersion is the served OpenAPI document's `info.version`.
const APIVersion = "1.0.0"

// Deps bundles the business-layer dependencies REST handlers call into.
// It is transport-agnostic: the same webhooks.WebhookServiceInterface used
// by the (removed) gRPC/Connect transports.
type Deps struct {
	Svc webhooks.WebhookServiceInterface
}

// Mount registers Huma on the given chi router: every /v1 REST operation,
// the OpenAPI document at /openapi.{json,yaml} (+ 3.0 variants), and the
// Scalar interactive reference at /docs. Returns the huma.API so callers can
// export the spec (see cmd/openapi-export).
func Mount(r chi.Router, svc webhooks.WebhookServiceInterface) huma.API {
	config := huma.DefaultConfig("Sparrow", APIVersion)
	config.Info.Description = "Sparrow is a self-hosted webhook delivery platform: register the " +
		"events your system produces, register the webhooks that should receive them, then push " +
		"event occurrences and Sparrow fans them out with retries and health tracking.\n\n" +
		"## Concepts\n\n" +
		"- **Event type** (`Event Types`) — a named, versioned schema for something that happened " +
		"in your system, e.g. `order.created`. Optional JSON Schema validation is *soft*: invalid " +
		"payloads are stored with warnings, never rejected.\n" +
		"- **Event** (`Events`) — one occurrence of an event type, pushed with a payload. Pushing " +
		"an event asynchronously fans it out to every matching subscription.\n" +
		"- **Webhook** (`Webhooks`) — a registered HTTP endpoint plus its delivery configuration " +
		"(retries, timeouts, signing). Every delivery is signed with HMAC-SHA256 and Ed25519.\n" +
		"- **Subscription** (`Subscriptions`) — links one webhook to one event type within a " +
		"namespace, optionally with a Go-template payload transform and label filters. Registering a " +
		"webhook auto-creates matching subscriptions.\n" +
		"- **Delivery** (`Deliveries`) — one attempt (and its retry history) to send an event to a " +
		"webhook. Failures are classified as retryable (5xx, timeout, network) or not (4xx, DNS, TLS).\n" +
		"- **Health** (`Health`) — a rolling per-webhook status (healthy/degraded/unhealthy) " +
		"computed from recent delivery outcomes.\n\n" +
		"## Namespaces\n\n" +
		"Webhooks, events, subscriptions, and deliveries are scoped to a `namespace` path segment " +
		"(defaults to `default` if you don't need multi-tenancy). A handful of read endpoints are " +
		"deliberately global (no namespace) for cross-tenant dashboards and id-only lookups where the " +
		"resource id is already globally unique.\n\n" +
		"## Authentication\n\n" +
		"Optional. When the server is started with `SPARROW_API_KEY` set, every `/v1/*` request " +
		"must include it in the `X-API-Key` header. When unset, all endpoints are open."
	config.Info.Contact = &huma.Contact{
		Name: "Sparrow",
		URL:  "https://github.com/sarathsp06/sparrow",
	}
	config.Info.License = &huma.License{
		Name:       "MIT",
		Identifier: "MIT",
	}
	config.Tags = []*huma.Tag{
		{Name: "Webhooks", Description: "Registered HTTP endpoints and their delivery configuration " +
			"(retries, timeouts, signing, rate limits). Secrets are always masked in responses except " +
			"once, at creation time."},
		{Name: "Event Types", Description: "Named, versioned schemas for the events your system can " +
			"produce. Schema validation is soft — invalid payloads are stored with warnings, not rejected."},
		{Name: "Events", Description: "Pushed occurrences of an event type. Pushing an event " +
			"asynchronously fans it out to every subscription that matches its event name and label filters."},
		{Name: "Subscriptions", Description: "The link between one webhook and one event type within a " +
			"namespace. Carries the optional payload-transform template and label filters that decide " +
			"whether a given event occurrence produces a delivery."},
		{Name: "Deliveries", Description: "Individual delivery attempts and their per-attempt retry " +
			"history. Failed deliveries can be retried singly or in bulk via a prepared batch snapshot."},
		{Name: "Health", Description: "Rolling per-webhook health status computed from recent delivery " +
			"outcomes, plus cross-namespace aggregate counts for dashboards."},
	}
	config.DocsPath = "/docs"
	config.DocsRenderer = huma.DocsRendererScalar
	config.OpenAPIPath = "/openapi"
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"ApiKeyAuth": {
			Type:        "apiKey",
			In:          "header",
			Name:        "X-API-Key",
			Description: "Required only when the server is started with SPARROW_API_KEY set; otherwise all endpoints are open.",
		},
	}
	// Optional auth: try the API key, but an empty requirement ({}) means
	// "or no credentials" is also acceptable.
	config.Security = []map[string][]string{
		{"ApiKeyAuth": {}},
		{},
	}

	api := humachi.New(r, config)
	d := &Deps{Svc: svc}

	registerWebhookRoutes(api, d)
	registerEventRoutes(api, d)
	registerSubscriptionRoutes(api, d)
	registerDeliveryRoutes(api, d)
	registerHealthRoutes(api, d)

	return api
}
