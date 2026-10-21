package rest

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/tlsrevocation"
	"github.com/sarathsp06/sparrow/pkg/safetransport"
)

// TLSRevocation is the server's CRL checker as the TLS endpoints see it
// (*tlsrevocation.Refresher; nil when SPARROW_TLS_REVOCATION=off).
type TLSRevocation interface {
	Mode() string
	Status() []safetransport.CRLStatus
	RequestRefresh(ctx context.Context) error
}

type crlOut struct {
	URL        string     `json:"url" doc:"CRL distribution point, from an intermediate in a receiver's chain."`
	Issuer     string     `json:"issuer" doc:"Common name of the CA that signs this CRL."`
	FetchedAt  *time.Time `json:"fetched_at,omitempty" doc:"When this instance last downloaded it successfully."`
	NextUpdate *time.Time `json:"next_update,omitempty" doc:"When the CA promises a newer CRL. Past it, the CRL is downloaded again before use."`
	Entries    int        `json:"entries" doc:"Revoked certificates it lists."`
	Error      string     `json:"error,omitempty" doc:"Why the last download or check failed. In soft mode such a CRL gives no answer and connections are allowed."`
}

type listCrlsOutput struct {
	Body struct {
		Mode string   `json:"mode" enum:"soft,hard,off" doc:"SPARROW_TLS_REVOCATION: soft allows a connection whose CRL is unavailable, hard refuses it, off skips the check."`
		CRLs []crlOut `json:"crls" doc:"CRLs this instance has cached, by URL. Each instance keeps its own cache."`
	}
}

type refreshCrlsOutput struct{}

// registerTLSRoutes registers the CRL status and refresh endpoints. rev may
// be nil (revocation off).
func registerTLSRoutes(api huma.API, rev TLSRevocation) {
	if rev == nil {
		rev = (*tlsrevocation.Refresher)(nil)
	}
	huma.Register(api, huma.Operation{
		OperationID: "listCrls",
		Method:      http.MethodGet,
		Path:        "/v1/tls/crls",
		Summary:     "List cached CRLs",
		Description: "Shows the CRLs the instance that answers has cached for checking intermediates in receivers' TLS chains, with when each was fetched and any error. " +
			"CRLs are downloaded the first time a chain needs them, refreshed in the background by the next handshake once older than SPARROW_CRL_MAX_AGE, and on demand with refreshCrls. " +
			"Each instance keeps its own cache, so behind a load balancer this shows one instance.",
		Tags: []string{"Server"},
	}, func(_ context.Context, _ *struct{}) (*listCrlsOutput, error) {
		out := &listCrlsOutput{}
		out.Body.Mode = rev.Mode()
		out.Body.CRLs = []crlOut{}
		for _, s := range rev.Status() {
			c := crlOut{URL: s.URL, Issuer: s.Issuer, Entries: s.Entries, Error: s.Error}
			if !s.FetchedAt.IsZero() {
				c.FetchedAt = &s.FetchedAt
			}
			if !s.NextUpdate.IsZero() {
				c.NextUpdate = &s.NextUpdate
			}
			out.Body.CRLs = append(out.Body.CRLs, c)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "refreshCrls",
		Method:        http.MethodPost,
		Path:          "/v1/tls/crls:refresh",
		Summary:       "Refresh cached CRLs on every instance",
		Description:   "Asks every instance (through Postgres NOTIFY) to download its cached CRLs now, for when a CA revokes an intermediate and deliveries should stop trusting it before SPARROW_CRL_MAX_AGE passes. Returns 202 once the request is sent; check listCrls for the result. One request a minute on each server: more answer 429. 409 when SPARROW_TLS_REVOCATION=off.",
		DefaultStatus: http.StatusAccepted,
		Errors:        []int{409, 429},
		Tags:          []string{"Server"},
	}, func(ctx context.Context, _ *struct{}) (*refreshCrlsOutput, error) {
		err := rev.RequestRefresh(ctx)
		switch {
		case errors.Is(err, tlsrevocation.ErrDisabled):
			return nil, huma.Error409Conflict(err.Error())
		case errors.Is(err, tlsrevocation.ErrTooSoon):
			return nil, huma.Error429TooManyRequests(err.Error())
		case err != nil:
			return nil, mapError(ctx, err, "failed to request a CRL refresh")
		}
		return &refreshCrlsOutput{}, nil
	})
}
