package rest_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

func contractRouter() http.Handler {
	svc := createOnlyService{
		reg: &store.WebhookRegistration{
			ID: uuid.New(), Consumer: "acme", URL: "https://example.com/hook",
			Active: true, Health: "unknown", SignatureType: store.SignatureTypeHMAC,
		},
		secret: "whsec_plaintext",
	}
	r := chi.NewRouter()
	rest.Mount(r, svc, rest.AccessDeps{}, rest.AIDeps{})
	return r
}

// Arrays are never null in a response: the stored registration has no
// events (a nil slice), and the response still carries []. A request that
// sends null for an optional array is treated as if it omitted the field.
func TestArraysAreNeverNull(t *testing.T) {
	h := contractRouter()
	for _, body := range []string{`{"url":"https://example.com/hook"}`, `{"url":"https://example.com/hook","events":null}`} {
		rec := do(h, http.MethodPost, "/v1/consumers/acme/webhooks", body)
		if rec.Code != http.StatusCreated {
			t.Fatalf("register %s = %d %s", body, rec.Code, rec.Body)
		}
		var out map[string]json.RawMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if got := string(out["events"]); got != "[]" {
			t.Fatalf("register %s: events = %s, want []", body, got)
		}
	}
}

// Request enums do not accept an empty string: omit the field for the default.
func TestRequestEnumsRejectEmptyString(t *testing.T) {
	h := contractRouter()
	for _, field := range []string{"on_transform_error", "template_missing_key", "signature_type", "template_source"} {
		rec := do(h, http.MethodPost, "/v1/consumers/acme/webhooks", `{"url":"https://example.com/hook","`+field+`":""}`)
		if rec.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s:\"\" = %d %s, want 422", field, rec.Code, rec.Body)
		}
	}
}

// The spec types arrays as plain arrays and timestamps as date-time.
func TestSpecTypesArraysAndTimestamps(t *testing.T) {
	api := rest.Mount(chi.NewRouter(), nil, rest.AccessDeps{}, rest.AIDeps{})
	spec, err := json.Marshal(api.OpenAPI())
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Type   any    `json:"type"`
					Format string `json:"format"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err := json.Unmarshal(spec, &doc); err != nil {
		t.Fatal(err)
	}
	for name, schema := range doc.Components.Schemas {
		for prop, p := range schema.Properties {
			types, _ := p.Type.([]any)
			isArray := p.Type == "array"
			for _, ty := range types {
				isArray = isArray || ty == "array"
			}
			if isArray && len(types) > 1 {
				t.Errorf("%s.%s is a nullable array %v", name, prop, types)
			}
			isString := p.Type == "string"
			for _, ty := range types {
				isString = isString || ty == "string"
			}
			if isString && (strings.HasSuffix(prop, "_at") || strings.HasSuffix(prop, "_since") || prop == "timestamp") && p.Format != "date-time" {
				t.Errorf("%s.%s is a timestamp without format date-time", name, prop)
			}
		}
	}
}
