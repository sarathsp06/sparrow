package rest

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/sarathsp06/sparrow/internal/webhooks"
)

// APIVersion is the served OpenAPI document's `info.version`.
const APIVersion = "1.0.0"

// Mount registers Huma on the given chi router: every /v1 REST operation,
// the OpenAPI document at /openapi.{json,yaml} (+ 3.0 variants), and the
// Scalar interactive reference at /docs. Returns the huma.API so callers can
// export the spec (see cmd/openapi-export). aiDeps is optional: a zero
// value registers the AI routes in their disabled state.
func Mount(r chi.Router, svc webhooks.WebhookServiceInterface, accessDeps AccessDeps, aiDeps AIDeps) huma.API {
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
		"(retries, timeouts, signing). Every delivery is signed with HMAC-SHA256 (Standard Webhooks), plus Ed25519 when the webhook's signature_type is ed25519.\n" +
		"- **Subscription** (`Subscriptions`) — links one webhook to one event type within a " +
		"consumer, optionally with a Go-template payload transform and label filters. Registering a " +
		"webhook auto-creates matching subscriptions.\n" +
		"- **Delivery** (`Deliveries`) — one attempt (and its retry history) to send an event to a " +
		"webhook. Failures are classified as retryable (5xx, timeout, network) or not (4xx, DNS, TLS).\n" +
		"- **Health** (`Health`) — a per-webhook label (healthy/degraded/unhealthy/unknown) " +
		"derived from the last 24 hours of attempts, refreshed every `SPARROW_HEALTH_EVAL_INTERVAL`.\n\n" +
		"## Consumers\n\n" +
		"Webhooks, events, subscriptions, and deliveries are scoped to a `consumer` path segment " +
		"(defaults to `default` if you don't need multi-tenancy). A handful of read endpoints are " +
		"deliberately global (no consumer) for cross-tenant dashboards and id-only lookups where the " +
		"resource id is already globally unique.\n\n" +
		"## Authentication\n\n" +
		"Optional. When the server is started with `SPARROW_API_KEY` set, every `/v1/*` request " +
		"must carry that master key or a tenant-wide access token (`createToken` without a " +
		"`consumer`), in the `X-API-Key` header or as `Authorization: Bearer <credential>`. When " +
		"`SPARROW_API_KEY` is unset, every endpoint is open, and outside `ENVIRONMENT=production` " +
		"CORS also allows any origin: never expose such a server beyond a trusted network.\n\n" +
		"## Portal tokens\n\n" +
		"A consumer-scoped access token (`createToken` with a `consumer`) powers the embedded " +
		"portal at `/portal`. It is refused on `/v1` (403); its API calls go to the single " +
		"`/portal/api/` prefix with `Authorization: Bearer <token>`, and the consumer is carried " +
		"by the token, not the URL, so exposing the portal means allowlisting only `/portal`, " +
		"`/_app`, and `/portal/api`. `/portal/api/<rest>` is served by " +
		"`/v1/consumers/<token's consumer>/<rest>`, except `POST events` (refused), plus these " +
		"global helpers: `GET event-types[/...]` (listed for the token's consumer), " +
		"`GET template-functions`, `GET capabilities`, and `POST subscriptions:testTemplate`. " +
		"Each operation's description says whether portal tokens can call it.\n\n" +
		"## Errors\n\n" +
		"Errors from the API itself are RFC 9457 problem details " +
		"(`Content-Type: application/problem+json`): `{\"title\", \"status\", \"detail\", " +
		"\"errors\"}`, where `errors` lists per-field validation problems. Two layers in front of " +
		"the API answer with plain `application/json` instead:\n\n" +
		"- Authentication (`401`, `503`): `{\"error\": \"unauthorized\", \"reason\": " +
		"\"missing|invalid|expired|revoked\", \"message\": \"...\"}`; `reason` is absent for " +
		"`503 unavailable` (retry after the `Retry-After` header).\n" +
		"- Authorization and the portal gateway (`401`, `403`): `{\"error\": \"forbidden\" or " +
		"\"unauthorized\", \"message\": \"...\"}`, e.g. a consumer-scoped token on `/v1`.\n\n" +
		"Status codes: `400` invalid input, `404` not found, `409` conflict or wrong state, `413` " +
		"body over `SPARROW_MAX_BODY_BYTES`, `429` rate limited, `500` internal (details are " +
		"logged, not returned), `503` temporarily unavailable, `504` timed out."
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
			"consumer. Carries the optional payload-transform template and label filters that decide " +
			"whether a given event occurrence produces a delivery."},
		{Name: "Deliveries", Description: "Individual delivery attempts and their per-attempt retry " +
			"history. Failed deliveries can be retried singly or in bulk via a prepared batch snapshot."},
		{Name: "Health", Description: "Rolling per-webhook health status computed from recent delivery " +
			"outcomes, plus cross-consumer aggregate counts for dashboards."},
		{Name: "Alert Configs", Description: "Opt-in email recipients for Sparrow's own self-generated " +
			"webhook.health_changed, webhook.delivery_failed and webhook.disabled system events."},
		{Name: "Recipes", Description: "Shipped webhook adapter recipes that pre-fill destinations, headers, and payload transforms."},
		{Name: "Access", Description: "Access tokens and invites. Tenant-wide tokens work like SPARROW_API_KEY but can be revoked one by one; consumer-scoped tokens only reach the portal API. Secrets are shown once."},
		{Name: "Server", Description: "Deployment-level information about this server, such as which optional features are configured."},
	}
	config.DocsPath = "/docs"
	config.DocsRenderer = huma.DocsRendererScalar
	config.OpenAPIPath = "/openapi"
	config.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"ApiKeyAuth": {
			Type:        "apiKey",
			In:          "header",
			Name:        "X-API-Key",
			Description: "The master key (SPARROW_API_KEY) or a tenant-wide access token. Also accepted as Authorization: Bearer <credential>. Required only when the server is started with SPARROW_API_KEY set; otherwise all endpoints are open. The embedded UI never receives the key: it asks for a credential on the first 401.",
		},
	}
	// Optional auth: try the API key, but an empty requirement ({}) means
	// "or no credentials" is also acceptable.
	config.Security = []map[string][]string{
		{"ApiKeyAuth": {}},
		{},
	}

	api := humachi.New(r, config)

	registerWebhookRoutes(api, svc)
	registerEventRoutes(api, svc)
	registerEventBundleRoutes(api, svc)
	registerSubscriptionRoutes(api, svc)
	registerDeliveryRoutes(api, svc)
	registerHealthRoutes(api, svc)
	registerAlertConfigRoutes(api, svc)
	registerRecipeRoutes(api)
	registerAccessRoutes(api, accessDeps)
	registerAIRoutes(api, svc, aiDeps)

	return api
}
