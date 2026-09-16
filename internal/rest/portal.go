package rest

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/middleware"
)

type createPortalTokenInput struct {
	Consumer   string `path:"consumer"`
	TTLSeconds int64  `query:"ttl_seconds" minimum:"1" maximum:"2592000" doc:"Token lifetime in seconds. Defaults to 7 days, capped at 30 days."`
}

type createPortalTokenOutput struct {
	Body struct {
		Token     string    `json:"token" doc:"Bearer token for the consumer portal. Send as 'Authorization: Bearer <token>'."`
		ExpiresAt time.Time `json:"expires_at" doc:"When the token stops working. Tokens are stateless — the only revocation is expiry."`
		Path      string    `json:"path" doc:"Server-relative portal URL with the token in the fragment (never sent to the server or logged). Prepend your Sparrow base URL and hand it to the end consumer."`
	}
}

// registerPortalRoutes registers the admin endpoint that mints consumer-scoped
// portal tokens. portal may be nil (e.g. spec export); minting then fails 503.
func registerPortalRoutes(api huma.API, portal *middleware.PortalTokens) {
	huma.Register(api, huma.Operation{
		OperationID: "createPortalToken",
		Method:      http.MethodPost,
		Path:        "/v1/consumers/{consumer}/portal-token",
		Summary:     "Mint a consumer-scoped portal access token",
		Description: "Creates a signed, expiring bearer token that lets an end consumer use the " +
			"embedded portal UI (or the API directly) scoped to this consumer only: manage their " +
			"webhooks and subscriptions, and inspect/retry their deliveries. Portal tokens cannot " +
			"push events or mint further tokens. Requires the admin API key; hand the returned " +
			"path (with the token in the URL fragment) to the end consumer.",
		Errors:        []int{400, 503},
		Tags:          []string{"Webhooks"},
		DefaultStatus: http.StatusCreated,
	}, func(_ context.Context, in *createPortalTokenInput) (*createPortalTokenOutput, error) {
		token, expiresAt, err := portal.Mint(in.Consumer, time.Duration(in.TTLSeconds)*time.Second)
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("portal tokens unavailable", err)
		}
		out := &createPortalTokenOutput{}
		out.Body.Token = token
		out.Body.ExpiresAt = expiresAt
		out.Body.Path = "/portal#token=" + token
		return out, nil
	})
}
