package rest

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/sarathsp06/sparrow/internal/tenant"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/internal/webhooks/store"
)

// maskSecret shows the first 4 characters of a plaintext secret, masking the rest.
func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 4 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:4] + strings.Repeat("*", len(secret)-4)
}

// maskEncryptedSecret decrypts an encrypted webhook secret and masks it for
// safe display in API responses.
func maskEncryptedSecret(encrypted []byte, svc webhooks.WebhookServiceInterface) string {
	if len(encrypted) == 0 {
		return ""
	}
	plaintext, err := svc.DecryptWebhookSecret(encrypted)
	if err != nil || plaintext == "" {
		return "••••••"
	}
	return maskSecret(plaintext)
}

// maskSecretHeaders decrypts encrypted secret headers and masks every value.
func maskSecretHeaders(encrypted []byte, svc webhooks.WebhookServiceInterface) map[string]string {
	if len(encrypted) == 0 {
		return nil
	}
	decrypted, err := svc.DecryptSecretHeaders(encrypted)
	if err != nil || len(decrypted) == 0 {
		return nil
	}
	masked := make(map[string]string, len(decrypted))
	for k := range decrypted {
		masked[k] = "••••••"
	}
	return masked
}

// getWebhookEventsMap batch-fetches subscribed event names for multiple
// webhooks in a single query, keyed by webhook id string.
func getWebhookEventsMap(ctx context.Context, svc webhooks.WebhookServiceInterface, regs []*store.WebhookRegistration) map[string][]string {
	if len(regs) == 0 {
		return map[string][]string{}
	}
	ids := make([]uuid.UUID, len(regs))
	for i, r := range regs {
		ids[i] = r.ID
	}
	subs, err := svc.GetWebhookRepo().ListSubscriptionsByWebhookIDs(ctx, tenant.DefaultTenantID, ids)
	if err != nil {
		slog.ErrorContext(ctx, "failed to batch-fetch subscriptions", "error", err)
		return map[string][]string{}
	}
	result := make(map[string][]string, len(ids))
	for _, sub := range subs {
		key := sub.WebhookID.String()
		result[key] = append(result[key], sub.EventName)
	}
	return result
}

// WebhookHTTPConfigOut is the REST representation of per-webhook HTTP
// delivery configuration.
type WebhookHTTPConfigOut struct {
	MaxRetries            int      `json:"max_retries" doc:"Maximum delivery attempts before a delivery is marked failed."`
	RetryBackoffSeconds   int      `json:"retry_backoff_seconds" doc:"Base delay between retry attempts, in seconds."`
	CaptureResponseBody   bool     `json:"capture_response_body" doc:"Whether the endpoint's response body is stored with each delivery attempt."`
	FollowRedirects       bool     `json:"follow_redirects" doc:"Whether HTTP redirects returned by the endpoint are followed."`
	VerifySSL             bool     `json:"verify_ssl" doc:"Whether the endpoint's TLS certificate is verified."`
	RequestTimeoutSeconds int      `json:"request_timeout_seconds" doc:"Per-attempt request timeout, in seconds."`
	ExpectedStatusCodes   []int32  `json:"expected_status_codes,omitempty" doc:"HTTP status codes treated as a successful delivery; 2xx if empty."`
	WebhookSecret         string   `json:"webhook_secret,omitempty" doc:"HMAC signing secret. Shown in full only once, in the response to webhook registration; masked on every later read."`
	UserAgent             string   `json:"user_agent,omitempty" doc:"User-Agent header sent with deliveries."`
	ContentType           string   `json:"content_type,omitempty" doc:"Content-Type header sent with deliveries."`
	RateLimitRPS          *float64 `json:"rate_limit_rps,omitempty" doc:"Delivery rate limit for this webhook, in requests per second."`
}

// WebhookOut is the REST representation of a registered webhook (list/get).
// Secrets are always masked; use the value returned at creation time to
// retrieve the real secret once.
type WebhookOut struct {
	WebhookID        string               `json:"webhook_id" doc:"Webhook id (UUID)."`
	Namespace        string               `json:"namespace" doc:"Tenant namespace this webhook belongs to."`
	Events           []string             `json:"events" doc:"Event type names this webhook is currently subscribed to, derived from its subscriptions."`
	URL              string               `json:"url" doc:"HTTP endpoint deliveries are POSTed to."`
	Headers          map[string]string    `json:"headers,omitempty" doc:"Static HTTP headers sent with every delivery."`
	Active           bool                 `json:"active" doc:"Whether the webhook currently receives deliveries."`
	Description      string               `json:"description,omitempty" doc:"Human-readable note about this webhook."`
	Health           string               `json:"health" enum:"healthy,degraded,unhealthy,unknown" doc:"Computed rolling health status."`
	SecretHeaders    map[string]string    `json:"secret_headers,omitempty" doc:"Encrypted secret header names, with values always masked."`
	SigningPublicKey string               `json:"signing_public_key,omitempty" doc:"Ed25519 public key for verifying the v1a, delivery signature. Safe to expose; there is no private-key equivalent to mask."`
	SignatureType    string               `json:"signature_type" enum:"hmac,ed25519" doc:"Which signature algorithm (hmac or ed25519) is treated as authoritative. Every delivery is always dual-signed with both."`
	HTTPConfig       WebhookHTTPConfigOut `json:"http_config" doc:"Per-webhook HTTP delivery configuration."`
	CreatedAt        string               `json:"created_at" doc:"Creation timestamp, RFC3339."`
	UpdatedAt        string               `json:"updated_at" doc:"Last-modified timestamp, RFC3339."`
}

func toWebhookOut(reg *store.WebhookRegistration, events []string, svc webhooks.WebhookServiceInterface) WebhookOut {
	codes := make([]int32, len(reg.ExpectedStatusCodes))
	for i, c := range reg.ExpectedStatusCodes {
		codes[i] = int32(c)
	}
	return WebhookOut{
		WebhookID:        reg.ID.String(),
		Namespace:        reg.Namespace,
		Events:           events,
		URL:              reg.URL,
		Headers:          reg.Headers,
		Active:           reg.Active,
		Description:      reg.Description,
		Health:           string(reg.Health),
		SecretHeaders:    maskSecretHeaders(reg.SecretHeaders, svc),
		SigningPublicKey: svc.WebhookSigningPublicKeyHex(reg.Ed25519PrivateKey),
		SignatureType:    string(reg.SignatureType),
		HTTPConfig: WebhookHTTPConfigOut{
			MaxRetries:            reg.MaxRetries,
			RetryBackoffSeconds:   reg.RetryBackoffSeconds,
			CaptureResponseBody:   reg.CaptureResponseBody,
			FollowRedirects:       reg.FollowRedirects,
			VerifySSL:             reg.VerifySSL,
			RequestTimeoutSeconds: reg.RequestTimeoutSeconds,
			ExpectedStatusCodes:   codes,
			WebhookSecret:         maskEncryptedSecret(reg.WebhookSecret, svc),
			UserAgent:             reg.UserAgent,
			ContentType:           reg.ContentType,
			RateLimitRPS:          reg.RateLimitRPS,
		},
		CreatedAt: reg.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: reg.UpdatedAt.Format(time.RFC3339Nano),
	}
}

// toWebhookOutFromDomain converts the webhooks-package WebhookRegistration
// (returned fresh at creation time, secret shown once) into the REST shape.
func toWebhookOutFromDomain(reg *webhooks.WebhookRegistration) WebhookOut {
	codes := make([]int32, len(reg.HTTPConfig.ExpectedStatusCodes))
	for i, c := range reg.HTTPConfig.ExpectedStatusCodes {
		codes[i] = int32(c)
	}
	return WebhookOut{
		WebhookID:     reg.ID,
		Namespace:     reg.Namespace,
		Events:        []string(reg.Events),
		URL:           reg.URL,
		Active:        reg.Active,
		Description:   reg.Description,
		Health:        reg.Health,
		SignatureType: reg.SignatureType,
		HTTPConfig: WebhookHTTPConfigOut{
			MaxRetries:            reg.HTTPConfig.MaxRetries,
			RetryBackoffSeconds:   reg.HTTPConfig.RetryBackoffSeconds,
			CaptureResponseBody:   reg.HTTPConfig.CaptureResponseBody,
			FollowRedirects:       reg.HTTPConfig.FollowRedirects,
			VerifySSL:             reg.HTTPConfig.VerifySSL,
			RequestTimeoutSeconds: reg.HTTPConfig.RequestTimeoutSeconds,
			ExpectedStatusCodes:   codes,
			WebhookSecret:         reg.HTTPConfig.WebhookSecret, // shown once, plaintext, at creation
			UserAgent:             reg.HTTPConfig.UserAgent,
			ContentType:           reg.HTTPConfig.ContentType,
			RateLimitRPS:          reg.HTTPConfig.RateLimitRPS,
		},
		CreatedAt: reg.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt: reg.UpdatedAt.Format(time.RFC3339Nano),
	}
}
