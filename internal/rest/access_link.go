package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/accesslink"
)

type createAccessLinkInput struct {
	TTLSeconds int64 `query:"ttl_seconds" minimum:"1" maximum:"86400" doc:"Link lifetime in seconds. Defaults to 15 minutes, capped at 24 hours."`
}

type createAccessLinkOutput struct {
	Body struct {
		Token     string    `json:"token" doc:"One-time token. The UI exchanges it for the API key at POST /access-link/redeem."`
		ExpiresAt time.Time `json:"expires_at" doc:"When the link stops working if it has not been used."`
		Path      string    `json:"path" doc:"UI-relative link with the token in the fragment (never sent to the server or logged). Prepend the base URL the UI is served from and send it to the person who needs access."`
	}
}

// registerAccessLinkRoutes registers the admin endpoint that mints one-time
// links handing the API key to a browser. links may be nil (spec export);
// minting then fails 503.
func registerAccessLinkRoutes(api huma.API, links *accesslink.Links) {
	huma.Register(api, huma.Operation{
		OperationID: "createAccessLink",
		Method:      http.MethodPost,
		Path:        "/v1/access-links",
		Summary:     "Mint a one-time link that signs a browser into the UI",
		Description: "Creates a signed, single-use, expiring link for sharing admin access to the web UI " +
			"without pasting the API key anywhere. Opening the link makes the UI exchange its token for " +
			"the API key once (POST /access-link/redeem) and remember it in that browser. Links are " +
			"invalidated by use, by expiry, and by rotating SPARROW_API_KEY. Whoever redeems a link " +
			"holds the real API key. Requires the admin API key; fails with 409 when the server has no " +
			"SPARROW_API_KEY.",
		Errors:        []int{409, 503},
		Tags:          []string{"Access"},
		DefaultStatus: http.StatusCreated,
	}, func(_ context.Context, in *createAccessLinkInput) (*createAccessLinkOutput, error) {
		token, expiresAt, err := links.Mint(time.Duration(in.TTLSeconds) * time.Second)
		if errors.Is(err, accesslink.ErrNoAPIKey) {
			return nil, huma.Error409Conflict(err.Error())
		}
		if err != nil {
			return nil, huma.Error503ServiceUnavailable("access links unavailable", err)
		}
		out := &createAccessLinkOutput{}
		out.Body.Token = token
		out.Body.ExpiresAt = expiresAt
		out.Body.Path = "/#access=" + token
		return out, nil
	})
}
