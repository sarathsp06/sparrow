package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/tlsrevocation"
	"github.com/sarathsp06/sparrow/pkg/safetransport"
)

type fakeRevocation struct {
	status   []safetransport.CRLStatus
	err      error
	requests int
}

func (f *fakeRevocation) Mode() string                      { return "soft" }
func (f *fakeRevocation) Status() []safetransport.CRLStatus { return f.status }
func (f *fakeRevocation) RequestRefresh(context.Context) error {
	f.requests++
	return f.err
}

func TestListCRLs(t *testing.T) {
	fetched := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	rev := &fakeRevocation{status: []safetransport.CRLStatus{
		{URL: "http://crl.example/root.crl", Issuer: "Example Root", FetchedAt: fetched, Entries: 3},
		{URL: "http://crl.example/down.crl", Issuer: "Example Root", Error: "CRL download: HTTP 500"},
	}}
	r := chi.NewRouter()
	rest.Mount(r, nil, rest.AccessDeps{}, rest.AIDeps{}, rev)

	rec := do(r, http.MethodGet, "/v1/tls/crls", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var out struct {
		Mode string `json:"mode"`
		CRLs []struct {
			URL       string     `json:"url"`
			FetchedAt *time.Time `json:"fetched_at"`
			Entries   int        `json:"entries"`
			Error     string     `json:"error"`
		} `json:"crls"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Mode != "soft" || len(out.CRLs) != 2 {
		t.Fatalf("got %+v", out)
	}
	if c := out.CRLs[0]; c.FetchedAt == nil || !c.FetchedAt.Equal(fetched) || c.Entries != 3 {
		t.Errorf("first CRL = %+v", c)
	}
	if c := out.CRLs[1]; c.FetchedAt != nil || c.Error == "" {
		t.Errorf("failed CRL = %+v", c)
	}
}

func TestRefreshCRLs(t *testing.T) {
	rev := &fakeRevocation{}
	r := chi.NewRouter()
	rest.Mount(r, nil, rest.AccessDeps{}, rest.AIDeps{}, rev)

	if rec := do(r, http.MethodPost, "/v1/tls/crls:refresh", ""); rec.Code != http.StatusAccepted || rev.requests != 1 {
		t.Fatalf("status = %d, requests %d, body %s", rec.Code, rev.requests, rec.Body)
	}
	rev.err = tlsrevocation.ErrTooSoon
	if rec := do(r, http.MethodPost, "/v1/tls/crls:refresh", ""); rec.Code != http.StatusTooManyRequests {
		t.Errorf("too soon: status = %d, want 429", rec.Code)
	}
}

func TestCRLsWhenOff(t *testing.T) {
	r := chi.NewRouter()
	rest.Mount(r, nil, rest.AccessDeps{}, rest.AIDeps{}, nil)

	rec := do(r, http.MethodGet, "/v1/tls/crls", "")
	if rec.Code != http.StatusOK || !json.Valid(rec.Body.Bytes()) {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var out struct {
		Mode string `json:"mode"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.Mode != "off" {
		t.Errorf("mode = %q, want off", out.Mode)
	}
	if rec := do(r, http.MethodPost, "/v1/tls/crls:refresh", ""); rec.Code != http.StatusConflict {
		t.Errorf("refresh when off: status = %d, want 409", rec.Code)
	}
}
