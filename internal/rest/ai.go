package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"

	"github.com/sarathsp06/sparrow/internal/ai"
	"github.com/sarathsp06/sparrow/internal/webhooks"
	"github.com/sarathsp06/sparrow/satellites/recipes"
)

// TemplateDrafter is what the REST layer needs from internal/ai. It is an
// interface so tests can stub the model and so a nil value means "off".
type TemplateDrafter interface {
	DraftTemplate(ctx context.Context, req ai.Request) (*ai.Result, error)
	Model() string
	Provider() string
}

// AIDeps wires optional AI features into Mount. A nil Drafter disables them:
// the capabilities endpoint reports enabled=false and the draft endpoint
// answers 503.
type AIDeps struct {
	Drafter TemplateDrafter
	// Fetch reads a docs_url for the prompt endpoint when no drafter is
	// configured (the drafter carries its own). Nil refuses docs_url.
	Fetch ai.DocFetcher
	// Timeout bounds one draft, repair rounds included. 0 means
	// defaultDraftTimeout.
	Timeout time.Duration
}

const defaultDraftTimeout = 3 * time.Minute

func (d AIDeps) enabled() bool { return d.Drafter != nil }

func (d AIDeps) draftTimeout() time.Duration {
	if d.Timeout > 0 {
		return d.Timeout
	}
	return defaultDraftTimeout
}

// extendWriteDeadline lifts the server's WriteTimeout for one slow
// operation. Without it the connection is dropped at 30s while the model is
// still answering, and the caller (or a proxy in front) sees a bare 5xx.
func extendWriteDeadline(d time.Duration) func(huma.Context, func(huma.Context)) {
	return func(hctx huma.Context, next func(huma.Context)) {
		_, w := humachi.Unwrap(hctx)
		// Best effort: a writer that cannot extend keeps the server default.
		_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(d))
		next(hctx)
	}
}

type capabilitiesOutput struct {
	Body struct {
		AIDrafting struct {
			Enabled    bool   `json:"enabled" doc:"True when the server is started with SPARROW_AI_API_KEY and POST /v1/subscriptions:draftTemplate is usable."`
			Model      string `json:"model,omitempty" doc:"Model id used for drafting, when enabled."`
			PromptOnly bool   `json:"prompt_only" doc:"True when no provider is configured: POST /v1/subscriptions:draftTemplatePrompt still builds the prompt for pasting into any chat assistant, while :draftTemplate answers 503."`
			Provider   string `json:"provider,omitempty" enum:"anthropic,openai" doc:"Chat API behind drafting: anthropic, or openai for any OpenAI-compatible server (Ollama, vLLM, OpenRouter, OpenAI)."`
		} `json:"ai_drafting" doc:"AI-assisted transform template drafting."`
		AlertDelivery struct {
			Configured bool `json:"configured" doc:"True when an active webhook under the _sparrow consumer has an unpaused subscription to Sparrow's system events, so alert configs actually send email. When false, alert configs are stored but nothing is sent."`
		} `json:"alert_delivery" doc:"Delivery of Sparrow's own alert emails (see the alert-configs endpoints)."`
	}
}

type draftTemplateBody struct {
	EventName       string         `json:"event_name" required:"true" doc:"Registered event type whose schema and sample payload ground the draft, and against which it is verified."`
	Instructions    string         `json:"instructions" required:"true" minLength:"1" maxLength:"4000" doc:"Plain-language description of the body the receiver should get, e.g. \"a short Slack message with the order id and total, flagged when refunded\"."`
	Recipe          string         `json:"recipe,omitempty" doc:"Name of a shipped recipe (see GET /v1/recipes) whose destination format the draft should follow, e.g. \"slack\". Omit for a custom receiver."`
	TargetExample   string         `json:"target_example,omitempty" maxLength:"20000" doc:"Optional: what the receiver expects, either an example body (the draft maps event fields onto that exact structure) or a plain-language description of its shape."`
	DocsURL         string         `json:"docs_url,omitempty" format:"uri" maxLength:"2048" doc:"Optional http(s) page documenting the receiver's payload. Sparrow fetches it (subject to the same network policy as webhook deliveries, so private and cloud-metadata addresses are refused unless allowed) and gives the model an excerpt of its text."`
	SamplePayload   map[string]any `json:"sample_payload,omitempty" doc:"Optional payload to ground and verify the draft against instead of the event type's registered sample. Does not change the event type."`
	CurrentTemplate string         `json:"current_template,omitempty" maxLength:"20000" doc:"Optional existing template to refine instead of drafting from scratch."`
}

type draftTemplateInput struct {
	Body draftTemplateBody
}

type draftTemplatePromptOutput struct {
	Body struct {
		Prompt       string `json:"prompt" doc:"A self-contained prompt to paste into any chat assistant: the template data model, rules, helper catalog, the event's schema and sample payload, any recipe, receiver example, docs excerpt, current template, and the instructions. Asks for the template in a code block."`
		SampleSource string `json:"sample_source" enum:"registered,provided" doc:"Which sample payload the prompt carries."`
	}
}

type draftTemplateOutput struct {
	Body struct {
		Template     string `json:"template" doc:"The drafted Go template, ready to save as transform_template."`
		Rendered     string `json:"rendered" doc:"The template rendered against the sample payload used (provided or registered), proving it renders."`
		SampleSource string `json:"sample_source" enum:"registered,provided" doc:"Which sample payload grounded and verified the draft."`
		Notes        string `json:"notes,omitempty" doc:"Short explanation from the model of the fields used and assumptions made."`
		Attempts     int    `json:"attempts" doc:"Draft/render rounds it took; more than 1 means an earlier draft failed to render and was repaired."`
		Model        string `json:"model" doc:"Model id that produced the draft."`
	}
}

func registerAIRoutes(api huma.API, svc webhooks.WebhookServiceInterface, deps AIDeps) {
	registerPromptRoute(api, svc, deps)
	huma.Register(api, huma.Operation{
		OperationID: "getCapabilities",
		Method:      http.MethodGet,
		Path:        "/v1/capabilities",
		Summary:     "List optional server features",
		Description: "Reports which optional, deployment-configured features this server offers so clients can show or hide the matching UI: AI-assisted transform template drafting, with the provider and model in use, and whether alert email delivery is set up.",
		Tags:        []string{"Server"},
	}, func(ctx context.Context, _ *struct{}) (*capabilitiesOutput, error) {
		out := &capabilitiesOutput{}
		out.Body.AIDrafting.Enabled = deps.enabled()
		out.Body.AIDrafting.PromptOnly = !deps.enabled()
		if deps.enabled() {
			out.Body.AIDrafting.Model = deps.Drafter.Model()
			out.Body.AIDrafting.Provider = deps.Drafter.Provider()
		}
		configured, err := svc.AlertDeliveryConfigured(ctx)
		if err != nil {
			return nil, mapError(ctx, err, "failed to check alert delivery")
		}
		out.Body.AlertDelivery.Configured = configured
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "draftSubscriptionTemplate",
		Method:      http.MethodPost,
		Path:        "/v1/subscriptions:draftTemplate",
		Summary:     "Draft a transform template with AI",
		Description: "Drafts a transform_template from a plain-language description, grounded in the event type's JSON Schema and sample payload, the template helper catalog, and optionally a shipped recipe's destination format or an example body the receiver expects. Every draft is rendered against the sample payload (as POST /v1/subscriptions:testTemplate does) and repaired until it renders, so the returned template is known to work. Nothing is saved: put the template into a subscription's transform_template. Requires AI drafting to be configured on the server (SPARROW_AI_API_KEY for Anthropic, or SPARROW_AI_PROVIDER=openai with SPARROW_AI_BASE_URL for any OpenAI-compatible server); otherwise 503. Only the sample payload (registered or provided in the request), the schema, and the request's own text are sent to the model, never stored events, headers, or secrets.",
		Errors:      []int{400, 404, 429, 503, 504},
		Tags:        []string{"Subscriptions"},
		// Room to write the answer (or the 504) after the draft's own budget.
		Middlewares: huma.Middlewares{extendWriteDeadline(deps.draftTimeout() + 30*time.Second)},
	}, func(ctx context.Context, in *draftTemplateInput) (*draftTemplateOutput, error) {
		if !deps.enabled() {
			return nil, huma.Error503ServiceUnavailable("AI drafting is not configured on this server (set SPARROW_AI_API_KEY, or SPARROW_AI_PROVIDER=openai with SPARROW_AI_BASE_URL); POST /v1/subscriptions:draftTemplatePrompt still builds a prompt you can paste into any chat assistant")
		}
		req, sampleSource, err := buildDraftRequest(ctx, svc, in.Body)
		if err != nil {
			return nil, err
		}
		draftCtx, cancel := context.WithTimeout(ctx, deps.draftTimeout())
		defer cancel()
		res, err := deps.Drafter.DraftTemplate(draftCtx, req)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil {
				return nil, huma.Error504GatewayTimeout(fmt.Sprintf("the model did not finish a draft within %s; try again, use a faster model, or raise SPARROW_AI_TIMEOUT", deps.draftTimeout()))
			}
			return nil, mapError(ctx, err, "failed to draft template")
		}
		out := &draftTemplateOutput{}
		out.Body.Template = res.Template
		out.Body.Rendered = res.Rendered
		out.Body.Notes = res.Notes
		out.Body.Attempts = res.Attempts
		out.Body.Model = res.Model
		out.Body.SampleSource = sampleSource
		return out, nil
	})
}

// buildDraftRequest grounds a draft or prompt request in the event type and
// optional recipe. Returns Huma errors ready to surface.
func buildDraftRequest(ctx context.Context, svc webhooks.WebhookServiceInterface, body draftTemplateBody) (ai.Request, string, error) {
	event, err := svc.GetEvent(ctx, body.EventName)
	if err != nil {
		return ai.Request{}, "", mapError(ctx, err, "failed to get event type")
	}
	if event == nil {
		return ai.Request{}, "", huma.Error404NotFound("event type not found")
	}
	sample, sampleSource := map[string]any(event.SamplePayload), "registered"
	if len(body.SamplePayload) > 0 {
		sample, sampleSource = body.SamplePayload, "provided"
	}
	req := ai.Request{
		EventName:       event.Name,
		Schema:          event.Schema,
		SamplePayload:   sample,
		Instructions:    body.Instructions,
		TargetExample:   body.TargetExample,
		DocsURL:         body.DocsURL,
		CurrentTemplate: body.CurrentTemplate,
	}
	if name := strings.TrimSpace(body.Recipe); name != "" {
		r, err := findRecipe(name)
		if err != nil {
			return ai.Request{}, "", mapError(ctx, err, "failed to load recipes")
		}
		if r == nil {
			return ai.Request{}, "", huma.Error404NotFound("recipe not found: " + name)
		}
		req.Recipe = toAIRecipe(r)
	}
	return req, sampleSource, nil
}

func registerPromptRoute(api huma.API, svc webhooks.WebhookServiceInterface, deps AIDeps) {
	huma.Register(api, huma.Operation{
		OperationID: "buildSubscriptionTemplatePrompt",
		Method:      http.MethodPost,
		Path:        "/v1/subscriptions:draftTemplatePrompt",
		Summary:     "Build the AI drafting prompt without calling a model",
		Description: "Returns the same prompt POST /v1/subscriptions:draftTemplate would send, as one self-contained text to paste into any chat assistant (ChatGPT, Claude, a local model). Works whether or not the server has an AI provider configured, so installs without SPARROW_AI_* still get a one-click prompt. The user pastes the assistant's template back into the subscription editor and checks it with Run Preview. Nothing is sent anywhere by Sparrow; a docs_url, if given, is fetched under the delivery network policy to include an excerpt.",
		Errors:      []int{400, 404},
		Tags:        []string{"Subscriptions"},
	}, func(ctx context.Context, in *draftTemplateInput) (*draftTemplatePromptOutput, error) {
		req, sampleSource, err := buildDraftRequest(ctx, svc, in.Body)
		if err != nil {
			return nil, err
		}
		docsText, err := ai.ResolveDocs(ctx, deps.Fetch, req.DocsURL)
		if err != nil {
			return nil, mapError(ctx, err, "failed to read docs_url")
		}
		helpers := make([]ai.HelperFunc, 0)
		for _, f := range svc.GetTemplateFunctions() {
			helpers = append(helpers, ai.HelperFunc{Name: f.Name, Description: f.Description})
		}
		out := &draftTemplatePromptOutput{}
		out.Body.Prompt = ai.NewPromptBuilder(helpers).ChatPrompt(req, docsText)
		out.Body.SampleSource = sampleSource
		return out, nil
	})
}

// toAIRecipe keeps what the model needs from a recipe. Secret headers are
// left out: their values are {{param}} tokens, but they say nothing about
// the body. A secret param's default is left out too.
func toAIRecipe(r *recipes.Recipe) *ai.Recipe {
	out := &ai.Recipe{
		Name:        r.Name,
		Description: r.Description,
		Guidance:    r.Guidance,
		URL:         r.Webhook.URL,
		Template:    r.Subscription.TransformTemplate,
	}
	for k, v := range r.Webhook.Headers {
		if strings.EqualFold(k, "Content-Type") {
			out.ContentType = v
		}
	}
	for _, p := range r.Params {
		prm := ai.RecipeParam{Name: p.Name, Prompt: p.Prompt, Help: p.Help, Example: p.Example, Enum: p.Enum}
		if !p.Secret {
			prm.Default = p.Default
		}
		out.Params = append(out.Params, prm)
	}
	return out
}

func findRecipe(name string) (*recipes.Recipe, error) {
	all, err := recipes.All()
	if err != nil {
		return nil, err
	}
	for i := range all {
		if strings.EqualFold(all[i].Name, name) {
			return &all[i], nil
		}
	}
	return nil, nil
}
