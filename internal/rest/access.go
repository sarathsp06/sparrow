package rest

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/accessauth"
	"github.com/sarathsp06/sparrow/pkg/access"
	"github.com/sarathsp06/sparrow/pkg/access/httpauth"
)

// AccessDeps wires the access-token endpoints. Service may be nil (spec
// export); the endpoints then answer 503.
type AccessDeps struct {
	Service *access.Service
	// AuthEnabled is true when SPARROW_API_KEY is set.
	AuthEnabled bool
	// TokenDefaultTTL is the lifetime of a tenant-wide token created without
	// ttl_seconds or never_expires (SPARROW_TOKEN_DEFAULT_TTL). 0 = never.
	TokenDefaultTTL time.Duration
}

// --- Output shapes ---

// TokenOut describes an access token. The secret is never included.
type TokenOut struct {
	ID         string     `json:"id" doc:"Token id (not the secret)."`
	Name       string     `json:"name" doc:"Who or what the token is for."`
	Consumer   *string    `json:"consumer" doc:"Consumer the token is limited to (portal API only). Null: tenant-wide, same power as SPARROW_API_KEY."`
	Status     string     `json:"status" enum:"active,revoked,expired"`
	CreatedBy  string     `json:"created_by" doc:"Name of the credential that created it (the master key, or another token)."`
	CreatedAt  time.Time  `json:"created_at"`
	ExpiresAt  *time.Time `json:"expires_at" doc:"Null: never expires."`
	RevokedAt  *time.Time `json:"revoked_at"`
	LastUsedAt *time.Time `json:"last_used_at" doc:"Approximate (updated at most once a minute)."`
}

// InviteOut describes an invite. The secret is never included.
type InviteOut struct {
	ID              string     `json:"id"`
	Name            string     `json:"name" doc:"Name given to the token the invite creates."`
	Consumer        *string    `json:"consumer" doc:"Consumer the resulting token is limited to. Null: tenant-wide."`
	Status          string     `json:"status" enum:"pending,redeemed,cancelled,expired"`
	TokenTTLSeconds *int64     `json:"token_ttl_seconds" doc:"Lifetime of the token it creates. Null: never expires."`
	CreatedBy       string     `json:"created_by"`
	CreatedAt       time.Time  `json:"created_at"`
	ExpiresAt       time.Time  `json:"expires_at" doc:"When the invite stops working if unused."`
	RedeemedAt      *time.Time `json:"redeemed_at"`
	CancelledAt     *time.Time `json:"cancelled_at"`
	TokenID         *string    `json:"token_id" doc:"Token created when the invite was redeemed."`
}

func toTokenOut(t access.Token, now time.Time) TokenOut {
	return TokenOut{
		ID: t.ID, Name: t.Name, Consumer: t.Scope, Status: string(t.Status(now)), CreatedBy: t.CreatedBy,
		CreatedAt: t.CreatedAt, ExpiresAt: t.ExpiresAt, RevokedAt: t.RevokedAt, LastUsedAt: t.LastUsedAt,
	}
}

func toInviteOut(inv access.Invite, now time.Time) InviteOut {
	out := InviteOut{
		ID: inv.ID, Name: inv.Name, Consumer: inv.Scope, Status: string(inv.Status(now)), CreatedBy: inv.CreatedBy,
		CreatedAt: inv.CreatedAt, ExpiresAt: inv.ExpiresAt, RedeemedAt: inv.RedeemedAt, CancelledAt: inv.CancelledAt, TokenID: inv.TokenID,
	}
	if inv.TokenTTL != nil {
		secs := int64(*inv.TokenTTL / time.Second)
		out.TokenTTLSeconds = &secs
	}
	return out
}

// --- Inputs / outputs ---

type whoAmIOutput struct {
	Body struct {
		AuthEnabled bool    `json:"auth_enabled" doc:"False when SPARROW_API_KEY is unset: every endpoint is open."`
		Name        string  `json:"name" doc:"Name of the credential used for this request."`
		MasterKey   bool    `json:"master_key" doc:"True when the request used SPARROW_API_KEY itself."`
		TokenID     *string `json:"token_id" doc:"Id of the access token used, if any."`
	}
}

type createTokenBody struct {
	Name         string `json:"name" required:"true" minLength:"1" maxLength:"200" doc:"Who or what the token is for, e.g. \"alice\" or \"ci-deploy\"."`
	Consumer     string `json:"consumer,omitempty" doc:"Limit the token to one consumer (portal API only). Omit for a tenant-wide token with the same power as SPARROW_API_KEY."`
	TTLSeconds   int64  `json:"ttl_seconds,omitempty" minimum:"0" doc:"Lifetime in seconds. Tenant-wide tokens default to the server's SPARROW_TOKEN_DEFAULT_TTL (90 days unless changed); consumer tokens default to 7 days and allow at most 30."`
	NeverExpires bool   `json:"never_expires,omitempty" doc:"Create a tenant-wide token that never expires (revoke it when no longer needed). Not allowed with ttl_seconds or for consumer tokens."`
	// IdempotencyKey needs a consumer: a tenant-wide secret is never stored
	// in a recoverable form.
	IdempotencyKey string `json:"idempotency_key,omitempty" maxLength:"200" doc:"Consumer tokens only. Your id for who the token is for (e.g. your user id when embedding the portal). While a token created for this consumer with the same key is still valid, it is returned again, secret included, instead of a new one (reused: true; name and ttl_seconds are ignored); once it expires or is revoked, a new one is created. Makes minting a portal link on every page view safe."`
}

type createTokenOutput struct {
	Body struct {
		Token      TokenOut `json:"token"`
		Secret     string   `json:"secret" doc:"The credential. Shown only in this response (and again for the same idempotency_key); send it as X-API-Key or Authorization: Bearer."`
		Reused     bool     `json:"reused" doc:"True when idempotency_key matched a still-valid token, which is returned instead of a new one."`
		PortalPath string   `json:"portal_path,omitempty" doc:"Consumer tokens only: server-relative portal URL with the token in the fragment (never sent to the server or logged). Prepend the base URL the UI is served from (the Sparrow server with SPARROW_SERVE_UI=true, or your separately hosted UI) and hand it to the end consumer."`
	}
}

type listTokensInput struct {
	Consumer        string `query:"consumer" doc:"Only tokens limited to this consumer."`
	IncludeInactive bool   `query:"include_inactive" doc:"Also return revoked and expired tokens."`
}

type listTokensOutput struct {
	Body struct {
		Items []TokenOut `json:"items"`
	}
}

type tokenIDInput struct {
	TokenID string `path:"token_id"`
}

type createInviteBody struct {
	Name              string `json:"name" required:"true" minLength:"1" maxLength:"200" doc:"Name for the token the invite creates, e.g. the invitee's name."`
	Consumer          string `json:"consumer,omitempty" doc:"Invite into one consumer's portal. Omit for tenant-wide (operator console) access."`
	TTLSeconds        int64  `json:"ttl_seconds,omitempty" minimum:"0" doc:"How long the invite can be used. Defaults to 24 hours, at most 7 days."`
	TokenTTLSeconds   int64  `json:"token_ttl_seconds,omitempty" minimum:"0" doc:"Lifetime of the token it creates; same defaults and limits as POST /v1/tokens."`
	TokenNeverExpires bool   `json:"token_never_expires,omitempty" doc:"The token it creates never expires; same rules as never_expires on POST /v1/tokens."`
}

type createInviteOutput struct {
	Body struct {
		Invite InviteOut `json:"invite"`
		Secret string    `json:"secret" doc:"The invite secret. Shown only in this response."`
		Path   string    `json:"path" doc:"UI-relative link with the secret in the fragment (never sent to servers or logs). Prepend the base URL the UI is served from."`
	}
}

type listInvitesInput struct {
	IncludeInactive bool `query:"include_inactive" doc:"Also return redeemed, cancelled, and expired invites."`
}

type listInvitesOutput struct {
	Body struct {
		Items []InviteOut `json:"items"`
	}
}

type inviteIDInput struct {
	InviteID string `path:"invite_id"`
}

// principalName is recorded as created_by. Requests reach these handlers
// only through the /v1 auth middleware, which always sets a principal.
func principalName(ctx context.Context) string {
	if p, ok := httpauth.FromContext(ctx); ok {
		return p.Name
	}
	return "unknown"
}

func mapAccessError(ctx context.Context, err error) error {
	switch {
	case errors.Is(err, access.ErrNotFound):
		return huma.Error404NotFound("not found")
	case errors.Is(err, access.ErrInvalidInvite):
		return huma.Error409Conflict("invite is no longer pending")
	case errors.Is(err, access.ErrInvalidRequest):
		return huma.Error400BadRequest(err.Error())
	default:
		// Never echo store errors: they can carry driver or connection details.
		slog.ErrorContext(ctx, "access store error", "error", err)
		return huma.Error500InternalServerError("access store error")
	}
}

// portalLinkPath is the portal URL for a consumer token. The consumer and
// expiry ride along in the fragment for display only: the server pins every
// request to the token's own consumer.
func portalLinkPath(secret, consumer string, expiresAt *time.Time) string {
	path := "/portal#token=" + secret + "&consumer=" + fragmentEscape(consumer)
	if expiresAt != nil {
		path += "&expires=" + strconv.FormatInt(expiresAt.Unix(), 10)
	}
	return path
}

// fragmentEscape percent-encodes s so the UI's decodeURIComponent restores it
// (QueryEscape's "+" for space is not decoded there).
func fragmentEscape(s string) string {
	return strings.ReplaceAll(url.QueryEscape(s), "+", "%20")
}

// consumerScope validates an optional consumer from a request body.
func consumerScope(c string) (*string, error) {
	if c == "" {
		return nil, nil
	}
	if err := accessauth.ValidateConsumer(c); err != nil {
		return nil, huma.Error400BadRequest(err.Error())
	}
	return &c, nil
}

func registerAccessRoutes(api huma.API, deps AccessDeps) {
	svc := deps.Service
	unavailable := func() error { return huma.Error503ServiceUnavailable("access tokens are not configured") }
	realm := accessauth.Realm()

	huma.Register(api, huma.Operation{
		OperationID: "getWhoAmI",
		Method:      http.MethodGet,
		Path:        "/v1/whoami",
		Summary:     "Show which credential this request used",
		Tags:        []string{"Access"},
	}, func(ctx context.Context, _ *struct{}) (*whoAmIOutput, error) {
		out := &whoAmIOutput{}
		out.Body.AuthEnabled = deps.AuthEnabled
		p, _ := httpauth.FromContext(ctx)
		out.Body.Name = p.Name
		out.Body.MasterKey = p.Root && deps.AuthEnabled
		if p.TokenID != "" {
			id := p.TokenID
			out.Body.TokenID = &id
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createToken",
		Method:      http.MethodPost,
		Path:        "/v1/tokens",
		Summary:     "Create an access token",
		Description: "Creates a named token. A tenant-wide token (no consumer) works exactly like SPARROW_API_KEY " +
			"but can be revoked on its own and never needs the master key to be shared. A consumer token only " +
			"works through the portal API (/portal/api/), limited to that consumer, and comes with a ready-made " +
			"portal link (portal_path) to embed or hand to the end consumer. The secret is returned once, or again " +
			"for the same idempotency_key while the token is valid.",
		Errors:        []int{400, 503},
		Tags:          []string{"Access"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct{ Body createTokenBody }) (*createTokenOutput, error) {
		if svc == nil {
			return nil, unavailable()
		}
		consumer, err := consumerScope(in.Body.Consumer)
		if err != nil {
			return nil, err
		}
		ttl, err := accessauth.TokenTTL(consumer, time.Duration(in.Body.TTLSeconds)*time.Second, in.Body.NeverExpires, deps.TokenDefaultTTL)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		key := in.Body.IdempotencyKey
		if key != "" && consumer == nil {
			return nil, huma.Error400BadRequest("idempotency_key needs a consumer")
		}
		req := access.CreateTokenRequest{Realm: realm, Scope: consumer, Name: in.Body.Name, TTL: ttl, CreatedBy: principalName(ctx)}
		var t access.Token
		var secret string
		created := true
		if key != "" {
			t, secret, created, err = svc.CreateTokenIdempotent(ctx, req, key)
		} else {
			t, secret, err = svc.CreateToken(ctx, req)
		}
		if errors.Is(err, access.ErrNoSealer) {
			return nil, huma.Error503ServiceUnavailable("idempotent tokens are not configured")
		}
		if err != nil {
			return nil, mapAccessError(ctx, err)
		}
		out := &createTokenOutput{}
		out.Body.Token, out.Body.Secret, out.Body.Reused = toTokenOut(t, time.Now()), secret, !created
		if consumer != nil {
			out.Body.PortalPath = portalLinkPath(secret, *consumer, t.ExpiresAt)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listTokens",
		Method:      http.MethodGet,
		Path:        "/v1/tokens",
		Summary:     "List access tokens",
		Description: "Newest first. Secrets are never returned. Revoked and expired tokens are hidden unless include_inactive is set.",
		Errors:      []int{503},
		Tags:        []string{"Access"},
	}, func(ctx context.Context, in *listTokensInput) (*listTokensOutput, error) {
		if svc == nil {
			return nil, unavailable()
		}
		tokens, err := svc.ListTokens(ctx, realm)
		if err != nil {
			return nil, mapAccessError(ctx, err)
		}
		now := time.Now()
		out := &listTokensOutput{}
		out.Body.Items = []TokenOut{}
		for _, t := range tokens {
			if in.Consumer != "" && (t.Scope == nil || *t.Scope != in.Consumer) {
				continue
			}
			if !in.IncludeInactive && t.Status(now) != access.StatusActive {
				continue
			}
			out.Body.Items = append(out.Body.Items, toTokenOut(t, now))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "revokeToken",
		Method:        http.MethodDelete,
		Path:          "/v1/tokens/{token_id}",
		Summary:       "Revoke an access token",
		Description:   "The token stops working immediately on this server and within 30 seconds everywhere else. Revoking is permanent and idempotent.",
		Errors:        []int{404, 503},
		Tags:          []string{"Access"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *tokenIDInput) (*struct{}, error) {
		if svc == nil {
			return nil, unavailable()
		}
		if err := svc.RevokeToken(ctx, realm, in.TokenID); err != nil {
			return nil, mapAccessError(ctx, err)
		}
		return nil, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "createInvite",
		Method:      http.MethodPost,
		Path:        "/v1/invites",
		Summary:     "Invite someone with a one-time link",
		Description: "Creates a single-use, expiring invite. Opening the returned path in the UI exchanges it for a new " +
			"access token named after the invitee (POST /invite/redeem), so nobody pastes a key into chat. " +
			"A consumer invite opens that consumer's portal instead of the operator console.",
		Errors:        []int{400, 503},
		Tags:          []string{"Access"},
		DefaultStatus: http.StatusCreated,
	}, func(ctx context.Context, in *struct{ Body createInviteBody }) (*createInviteOutput, error) {
		if svc == nil {
			return nil, unavailable()
		}
		consumer, err := consumerScope(in.Body.Consumer)
		if err != nil {
			return nil, err
		}
		inviteTTL, err := accessauth.InviteTTL(time.Duration(in.Body.TTLSeconds) * time.Second)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		tokenTTL, err := accessauth.TokenTTL(consumer, time.Duration(in.Body.TokenTTLSeconds)*time.Second, in.Body.TokenNeverExpires, deps.TokenDefaultTTL)
		if err != nil {
			return nil, huma.Error400BadRequest(err.Error())
		}
		inv, secret, err := svc.CreateInvite(ctx, access.CreateInviteRequest{
			Realm: realm, Scope: consumer, Name: in.Body.Name, TTL: inviteTTL, TokenTTL: tokenTTL, CreatedBy: principalName(ctx),
		})
		if err != nil {
			return nil, mapAccessError(ctx, err)
		}
		out := &createInviteOutput{}
		out.Body.Invite, out.Body.Secret = toInviteOut(inv, time.Now()), secret
		out.Body.Path = "/#invite=" + secret
		if consumer != nil {
			out.Body.Path = "/portal#invite=" + secret
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "listInvites",
		Method:      http.MethodGet,
		Path:        "/v1/invites",
		Summary:     "List invites",
		Description: "Newest first. Only pending invites unless include_inactive is set. Secrets are never returned.",
		Errors:      []int{503},
		Tags:        []string{"Access"},
	}, func(ctx context.Context, in *listInvitesInput) (*listInvitesOutput, error) {
		if svc == nil {
			return nil, unavailable()
		}
		invites, err := svc.ListInvites(ctx, realm)
		if err != nil {
			return nil, mapAccessError(ctx, err)
		}
		now := time.Now()
		out := &listInvitesOutput{}
		out.Body.Items = []InviteOut{}
		for _, inv := range invites {
			if !in.IncludeInactive && inv.Status(now) != access.StatusPending {
				continue
			}
			out.Body.Items = append(out.Body.Items, toInviteOut(inv, now))
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "cancelInvite",
		Method:        http.MethodDelete,
		Path:          "/v1/invites/{invite_id}",
		Summary:       "Cancel a pending invite",
		Errors:        []int{404, 409, 503},
		Tags:          []string{"Access"},
		DefaultStatus: http.StatusNoContent,
	}, func(ctx context.Context, in *inviteIDInput) (*struct{}, error) {
		if svc == nil {
			return nil, unavailable()
		}
		if err := svc.CancelInvite(ctx, realm, in.InviteID); err != nil {
			return nil, mapAccessError(ctx, err)
		}
		return nil, nil
	})
}
