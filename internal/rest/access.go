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
	// PortalRetry is true when portal tokens may retry deliveries
	// (SPARROW_PORTAL_RETRY); reported by GET /v1/capabilities.
	PortalRetry bool
}

// --- Output shapes ---

// TokenOut describes an access token. The secret is never included.
type TokenOut struct {
	ID         string     `json:"id" doc:"Token id (not the secret)."`
	Name       string     `json:"name" doc:"Who or what the token is for."`
	Consumer   *string    `json:"consumer" doc:"Consumer the token is limited to (portal API only). Null: tenant-wide, same power as SPARROW_API_KEY."`
	Status     string     `json:"status" enum:"active,revoked,expired" doc:"active: usable. revoked: revoked by DELETE /v1/tokens/{token_id}. expired: past expires_at."`
	CreatedBy  string     `json:"created_by" doc:"Name of the credential that created it (the master key, or another token)."`
	CreatedAt  time.Time  `json:"created_at" doc:"When the token was created, RFC3339."`
	ExpiresAt  *time.Time `json:"expires_at" doc:"When the token stops working, RFC3339. Null: never expires."`
	RevokedAt  *time.Time `json:"revoked_at" doc:"When the token was revoked, RFC3339. Null: not revoked."`
	LastUsedAt *time.Time `json:"last_used_at" doc:"When the token was last used, RFC3339; approximate (updated at most once a minute). Null: never used."`
}

// InviteOut describes an invite. The secret is never included.
type InviteOut struct {
	ID              string     `json:"id" doc:"Invite id (not the secret)."`
	Name            string     `json:"name" doc:"Name given to the token the invite creates."`
	Consumer        *string    `json:"consumer" doc:"Consumer the resulting token is limited to. Null: tenant-wide."`
	Status          string     `json:"status" enum:"pending,redeemed,cancelled,expired" doc:"pending: can still be redeemed. redeemed: used once, see token_id. cancelled: cancelled by DELETE /v1/invites/{invite_id}. expired: past expires_at unused."`
	TokenTTLSeconds *int64     `json:"token_ttl_seconds" doc:"Lifetime of the token it creates, in seconds. Null: the token never expires."`
	CreatedBy       string     `json:"created_by" doc:"Name of the credential that created the invite."`
	CreatedAt       time.Time  `json:"created_at" doc:"When the invite was created, RFC3339."`
	ExpiresAt       time.Time  `json:"expires_at" doc:"When the invite stops working if unused, RFC3339."`
	RedeemedAt      *time.Time `json:"redeemed_at" doc:"When the invite was redeemed, RFC3339. Null: not redeemed."`
	CancelledAt     *time.Time `json:"cancelled_at" doc:"When the invite was cancelled, RFC3339. Null: not cancelled."`
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
		Name        string  `json:"name" doc:"Name of the credential used for this request: the master key's name, the access token's name, or a placeholder when auth is off."`
		MasterKey   bool    `json:"master_key" doc:"True when the request used SPARROW_API_KEY itself. Always false when auth is off."`
		TokenID     *string `json:"token_id" doc:"Id of the access token used; null for the master key or when auth is off."`
	}
}

type createTokenBody struct {
	Name         string `json:"name" required:"true" minLength:"1" maxLength:"200" doc:"Who or what the token is for, e.g. \"alice\" or \"ci-deploy\"."`
	Consumer     string `json:"consumer,omitempty" doc:"Limit the token to one consumer: it then works only through the portal API (/portal/api/). Omit for a tenant-wide token with the same power as SPARROW_API_KEY. At most 255 characters, no /, \\, ?, #, % or dot segments; _sparrow is not allowed."`
	TTLSeconds   int64  `json:"ttl_seconds,omitempty" minimum:"0" doc:"Lifetime in seconds. 0 or omitted takes the default: SPARROW_TOKEN_DEFAULT_TTL (90 days unless changed) for tenant-wide tokens, 7 days for consumer tokens. Consumer tokens allow at most 30 days."`
	NeverExpires bool   `json:"never_expires,omitempty" doc:"Create a tenant-wide token that never expires (revoke it when no longer needed). Not allowed with ttl_seconds or for consumer tokens."`
	// ExternalID needs a consumer: a tenant-wide secret is never stored in a
	// recoverable form.
	ExternalID string `json:"external_id,omitempty" maxLength:"200" doc:"Consumer tokens only. Your id for who or what the token is for, e.g. your user id when embedding the portal. There is at most one active token per consumer and external_id: while it is valid, asking again returns it, secret included (reused: true; name and ttl_seconds are ignored), instead of creating another; once it expires or is revoked, a new one is created. This makes minting a portal link on every page view safe. Not a secret: it is fine to log."`
}

type createTokenOutput struct {
	Body struct {
		Token      TokenOut `json:"token" doc:"The created (or reused) token."`
		Secret     string   `json:"secret" doc:"The credential. Shown only in this response (and again for the same external_id); send it as X-API-Key or Authorization: Bearer."`
		Reused     bool     `json:"reused" doc:"True when external_id matched a still-valid token, which is returned instead of a new one."`
		PortalPath string   `json:"portal_path,omitempty" doc:"Consumer tokens only: server-relative portal URL with the token in the fragment (never sent to the server or logged). Prepend the base URL the UI is served from (the Sparrow server with SPARROW_SERVE_UI=true, or your separately hosted UI) and hand it to the end consumer."`
	}
}

type listTokensInput struct {
	Consumer        string `query:"consumer" doc:"Only tokens limited to this consumer. Omit to list every token, tenant-wide ones included."`
	IncludeInactive bool   `query:"include_inactive" doc:"Also return revoked and expired tokens."`
}

type listTokensOutput struct {
	Body struct {
		Items []TokenOut `json:"items" doc:"Tokens, newest first. Not paginated."`
	}
}

type tokenIDInput struct {
	TokenID string `path:"token_id" doc:"Token id (the id field, not the secret)."`
}

type createInviteBody struct {
	Name              string `json:"name" required:"true" minLength:"1" maxLength:"200" doc:"Name for the token the invite creates, e.g. the invitee's name."`
	Consumer          string `json:"consumer,omitempty" doc:"Invite into one consumer's portal: the token it creates is limited to that consumer. Omit for tenant-wide (operator console) access. Same naming rules as on POST /v1/tokens."`
	TTLSeconds        int64  `json:"ttl_seconds,omitempty" minimum:"0" doc:"How long the invite can be redeemed, in seconds. 0 or omitted means 24 hours; at most 7 days."`
	TokenTTLSeconds   int64  `json:"token_ttl_seconds,omitempty" minimum:"0" doc:"Lifetime in seconds of the token it creates, counted from redemption; same defaults and limits as ttl_seconds on POST /v1/tokens."`
	TokenNeverExpires bool   `json:"token_never_expires,omitempty" doc:"The token it creates never expires; same rules as never_expires on POST /v1/tokens."`
}

type createInviteOutput struct {
	Body struct {
		Invite InviteOut `json:"invite" doc:"The created invite."`
		Secret string    `json:"secret" doc:"The invite secret. Shown only in this response."`
		Path   string    `json:"path" doc:"UI-relative link with the secret in the fragment (never sent to servers or logs): /#invite=... for a tenant-wide invite, /portal#invite=... for a consumer invite. Prepend the base URL the UI is served from."`
	}
}

type listInvitesInput struct {
	IncludeInactive bool `query:"include_inactive" doc:"Also return redeemed, cancelled, and expired invites."`
}

type listInvitesOutput struct {
	Body struct {
		Items []InviteOut `json:"items" doc:"Invites, newest first. Not paginated."`
	}
}

type inviteIDInput struct {
	InviteID string `path:"invite_id" doc:"Invite id (the id field, not the secret)."`
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
		Description: "Reports whether auth is on and which credential (master key or access token) authenticated this request. No side effects. Useful to check a key or token before using it.\n\nPortal tokens: not allowed.",
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
			"for the same external_id while the token is valid. Without external_id every call creates a new token.\n\n" +
			"400 for an invalid consumer, both ttl_seconds and never_expires, never_expires or more than 30 days on a consumer token, or external_id without a consumer. 503 when the token store is unavailable, or for external_id when no keyring is configured.\n\nPortal tokens: not allowed.",
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
		externalID := in.Body.ExternalID
		if externalID != "" && consumer == nil {
			return nil, huma.Error400BadRequest("external_id needs a consumer")
		}
		req := access.CreateTokenRequest{Realm: realm, Scope: consumer, Name: in.Body.Name, TTL: ttl, CreatedBy: principalName(ctx)}
		var t access.Token
		var secret string
		created := true
		if externalID != "" {
			t, secret, created, err = svc.GetOrCreateToken(ctx, req, externalID)
		} else {
			t, secret, err = svc.CreateToken(ctx, req)
		}
		if errors.Is(err, access.ErrNoSealer) {
			return nil, huma.Error503ServiceUnavailable("tokens with an external_id are not configured")
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
		Description: "Lists access tokens, newest first. Secrets are never returned. Revoked and expired tokens are hidden unless include_inactive is set. Not paginated.\n\nPortal tokens: not allowed.",
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
		Description:   "Revokes a token: it stops working immediately on this server and within 30 seconds on other instances. Revoking is permanent and idempotent (204 again for an already revoked token). Returns 204; 404 for an unknown id.\n\nPortal tokens: not allowed.",
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
			"A consumer invite opens that consumer's portal instead of the operator console. The secret is returned only here. " +
			"Redeeming happens at POST /invite/redeem (outside /v1, no credential needed); an unknown, used, cancelled or expired invite gets 400 invalid_invite there.\n\nPortal tokens: not allowed.",
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
		Description: "Lists invites, newest first. Only pending invites unless include_inactive is set. Secrets are never returned. Not paginated.\n\nPortal tokens: not allowed.",
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
		Description:   "Cancels a pending invite so it can no longer be redeemed. Returns 204. 409 when the invite is no longer pending (redeemed, cancelled or expired); a token already created from it keeps working, so revoke that separately. 404 for an unknown id.\n\nPortal tokens: not allowed.",
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
