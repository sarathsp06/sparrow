package rest

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/sarathsp06/sparrow/internal/webhooks"
)

type capabilitiesOutput struct {
	Body struct {
		AIDrafting struct {
			Enabled    bool   `json:"enabled" doc:"True when the server is started with SPARROW_AI_ENABLED=true, whether or not the provider settings are valid. Drafting (POST /v1/subscriptions:draftTemplate) is usable only when enabled is true and problem is absent."`
			Problem    string `json:"problem,omitempty" doc:"Why drafting is enabled but not usable: a missing or invalid provider setting. POST /v1/subscriptions:draftTemplate answers 503 with this text."`
			Model      string `json:"model,omitempty" doc:"Model id used for drafting; absent when drafting is off or not usable."`
			PromptOnly bool   `json:"prompt_only" doc:"Always the opposite of enabled: true when SPARROW_AI_ENABLED is not true, so clients offer only the copy-and-paste prompt. POST /v1/subscriptions:draftTemplatePrompt works whatever this says."`
			Provider   string `json:"provider,omitempty" enum:"anthropic,openai" doc:"Chat API behind drafting: anthropic, or openai for any OpenAI-compatible server (Ollama, vLLM, OpenRouter, OpenAI). Absent when drafting is off or not usable."`
		} `json:"ai_drafting" doc:"AI-assisted transform template drafting."`
		AlertDelivery struct {
			Configured bool `json:"configured" doc:"True when an active webhook under the _sparrow consumer has an unpaused subscription to Sparrow's system events, so alert configs actually send email. When false, alert configs are stored but nothing is sent."`
		} `json:"alert_delivery" doc:"Delivery of Sparrow's own alert emails (see the alert-configs endpoints)."`
		Portal struct {
			Retry bool `json:"retry" doc:"True when portal tokens may retry their deliveries (SPARROW_PORTAL_RETRY, default true). When false, the retry calls under /portal/api/ answer 403; admins can still retry."`
		} `json:"portal" doc:"What the consumer portal is allowed to do on this server."`
	}
}

// registerCapabilitiesRoute serves GET /v1/capabilities: which optional,
// deployment-configured features this server offers, so clients show or
// hide the matching UI. Each feature's settings come from its own deps.
func registerCapabilitiesRoute(api huma.API, svc webhooks.WebhookServiceInterface, aiDeps AIDeps, portalRetry bool) {
	huma.Register(api, huma.Operation{
		OperationID: "getCapabilities",
		Method:      http.MethodGet,
		Path:        "/v1/capabilities",
		Summary:     "List optional server features",
		Description: "Reports which optional, deployment-configured features this server offers so clients can show or hide the matching UI: AI-assisted transform template drafting, with the provider and model in use, whether alert email delivery is set up, and whether portal tokens may retry deliveries. Read-only.\n\n" +
			"ai_drafting.enabled reflects SPARROW_AI_ENABLED=true; drafting is usable only when enabled is true and problem is absent; prompt_only is the opposite of enabled.\n\nPortal tokens: allowed, as GET /portal/api/capabilities.",
		Tags: []string{"Server"},
	}, func(ctx context.Context, _ *struct{}) (*capabilitiesOutput, error) {
		out := &capabilitiesOutput{}
		out.Body.AIDrafting.Enabled = aiDeps.enabled()
		out.Body.AIDrafting.PromptOnly = !aiDeps.enabled()
		if aiDeps.ConfigErr != nil {
			out.Body.AIDrafting.Problem = aiDeps.ConfigErr.Error()
		}
		if aiDeps.Drafter != nil {
			out.Body.AIDrafting.Model = aiDeps.Drafter.Model()
			out.Body.AIDrafting.Provider = aiDeps.Drafter.Provider()
		}
		configured, err := svc.AlertDeliveryConfigured(ctx)
		if err != nil {
			return nil, mapError(ctx, err, "failed to check alert delivery")
		}
		out.Body.AlertDelivery.Configured = configured
		out.Body.Portal.Retry = portalRetry
		return out, nil
	})
}
