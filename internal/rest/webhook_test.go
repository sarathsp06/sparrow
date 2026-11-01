package rest_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/rest"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// createOnlyService returns a fixed stored registration from CreateWebhook.
type createOnlyService struct {
	webhooks.WebhookServiceInterface
	reg    *store.WebhookRegistration
	secret string
}

func (s createOnlyService) CreateWebhook(context.Context, webhooks.WebhookRegistrationRequest) (*store.WebhookRegistration, string, error) {
	return s.reg, s.secret, nil
}

func (s createOnlyService) DecryptSecretHeaders([]byte) (map[string]string, error) {
	return map[string]string{"Authorization": "Bearer upstream"}, nil
}

func (s createOnlyService) DecryptWebhookSecret([]byte) (string, error) { return s.secret, nil }

func (s createOnlyService) WebhookSigningPublicKeyHex([]byte) string { return "" }

// The create response must describe the stored webhook like every later read
// does (headers, masked secret headers), plus the plaintext secret once.
func TestRegisterWebhook_ReturnsStoredRegistration(t *testing.T) {
	svc := createOnlyService{
		reg: &store.WebhookRegistration{
			ID:            uuid.New(),
			Consumer:      "acme",
			URL:           "https://example.com/hook",
			Headers:       map[string]string{"X-Team": "billing"},
			SecretHeaders: []byte("encrypted"),
			WebhookSecret: []byte("encrypted"),
			Active:        true,
			Health:        "unknown",
			SignatureType: store.SignatureTypeHMAC,
		},
		secret: "whsec_plaintext",
	}
	r := chi.NewRouter()
	rest.Mount(r, svc, rest.AccessDeps{}, rest.AIDeps{}, nil)

	rec := do(r, http.MethodPost, "/v1/consumers/acme/webhooks", `{"url":"https://example.com/hook","events":["order.created"]}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var out rest.WebhookOut
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.Headers["X-Team"] != "billing" {
		t.Errorf("headers = %v, want X-Team", out.Headers)
	}
	if out.SecretHeaders["Authorization"] != "••••••" {
		t.Errorf("secret_headers = %v, want Authorization masked", out.SecretHeaders)
	}
	if out.HTTPConfig.WebhookSecret != "whsec_plaintext" {
		t.Errorf("webhook_secret = %q, want the plaintext secret", out.HTTPConfig.WebhookSecret)
	}
	if len(out.Events) != 1 || out.Events[0] != "order.created" {
		t.Errorf("events = %v", out.Events)
	}
}
