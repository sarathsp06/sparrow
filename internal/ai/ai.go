// Package ai drafts subscription transform templates from a plain-language
// description. It grounds the model in exactly what the server already
// knows — the event type's JSON Schema and sample payload, the template
// helper catalog, and (optionally) a recipe or an example of the body the
// receiver expects — then renders every draft through the same dry-run
// path the editor's "Run Preview" uses and feeds render errors back until
// the template renders cleanly.
//
// The feature is off unless SPARROW_AI_API_KEY is set. Only the registered
// sample payload is ever sent to the model, never stored events or secrets.
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go/option"

	svcerrors "github.com/sarathsp06/sparrow/pkg/errors"
)

// Renderer dry-runs a template against payload as a delivery of eventName
// and returns the rendered body. Implemented by webhooks.RenderTemplatePreview.
type Renderer func(ctx context.Context, eventName, tmpl string, payload map[string]any) (string, error)

// DocFetcher fetches a receiver's documentation page and returns its text.
// Implemented by NewDocFetcher, which enforces the delivery network policy.
type DocFetcher func(ctx context.Context, rawURL string) (string, error)

// HelperFunc is one template helper, as listed by GET /v1/template-functions.
type HelperFunc struct {
	Name        string
	Description string
}

// Provider selects which chat API the drafter talks to.
type Provider string

const (
	// ProviderAnthropic uses the Anthropic Messages API via the official SDK.
	ProviderAnthropic Provider = "anthropic"
	// ProviderOpenAI uses any OpenAI-compatible /v1/chat/completions endpoint:
	// OpenAI itself, or self-hosted servers such as Ollama, vLLM, LM Studio
	// and llama.cpp, or routers such as OpenRouter.
	ProviderOpenAI Provider = "openai"
)

// Config configures the drafter.
type Config struct {
	Provider Provider // empty means ProviderAnthropic
	APIKey   string   // optional for local OpenAI-compatible servers
	Model    string
	BaseURL  string // required for ProviderOpenAI (e.g. http://localhost:11434/v1)
	// MaxAttempts bounds the draft → render → fix loop. 0 means 3.
	MaxAttempts int
}

// Request is everything the drafter needs for one draft.
type Request struct {
	EventName     string
	Schema        map[string]any
	SamplePayload map[string]any
	// Instructions is the user's plain-language description of the payload
	// the receiver should get.
	Instructions string
	// Recipe, when set, is the shipped destination format (e.g. "slack") the
	// draft must follow.
	Recipe *Recipe
	// TargetExample is what the receiver expects: an example body, or a
	// plain-language description of its shape.
	TargetExample string
	// DocsURL is an optional page documenting the receiver's payload. It is
	// fetched server-side (see DocFetcher) and an excerpt is given to the model.
	DocsURL string
	// CurrentTemplate, when set, is refined rather than replaced.
	CurrentTemplate string
}

// Result is a draft that rendered successfully against the sample payload.
type Result struct {
	Template string
	Rendered string
	Notes    string
	Attempts int
	Model    string
}

// turn is one message in the provider-neutral conversation the drafter keeps.
type turn struct {
	role string // "user" or "assistant"
	text string
}

// completion is a provider's answer to one request.
type completion struct {
	text    string
	refused bool
}

// completer is the narrow seam between the drafter and a chat API. Each
// provider turns (system, turns, json schema) into one structured reply.
type completer interface {
	complete(ctx context.Context, system string, turns []turn, schema map[string]any) (completion, error)
}

// ResolveDocs fetches req.DocsURL through fetch when set. A nil fetch with a
// docs URL is an InvalidArgument so the caller can tell the user.
func ResolveDocs(ctx context.Context, fetch DocFetcher, docsURL string) (string, error) {
	u := strings.TrimSpace(docsURL)
	if u == "" {
		return "", nil
	}
	if fetch == nil {
		return "", svcerrors.Error(svcerrors.InvalidArgument, "docs_url is not supported on this server")
	}
	text, err := fetch(ctx, u)
	if err != nil {
		return "", svcerrors.Wrapf(err, svcerrors.InvalidArgument, "could not read docs_url: %v", err)
	}
	return text, nil
}

// Drafter drafts transform templates with a chat model.
type Drafter struct {
	client      completer
	provider    Provider
	model       string
	render      Renderer
	prompts     *PromptBuilder
	fetch       DocFetcher
	maxAttempts int
}

// New builds a Drafter. helpers is the template helper catalog the model is
// allowed to use; render verifies each draft.
func New(cfg Config, render Renderer, helpers []HelperFunc, fetch DocFetcher, opts ...option.RequestOption) (*Drafter, error) {
	if cfg.Model == "" {
		return nil, errors.New("ai: model is required")
	}
	if render == nil {
		return nil, errors.New("ai: renderer is required")
	}
	provider := cfg.Provider
	if provider == "" {
		provider = ProviderAnthropic
	}
	var client completer
	switch provider {
	case ProviderAnthropic:
		if cfg.APIKey == "" {
			return nil, errors.New("ai: api key is required for the anthropic provider")
		}
		client = newAnthropicClient(cfg, opts...)
	case ProviderOpenAI:
		if cfg.BaseURL == "" {
			return nil, errors.New("ai: base URL is required for the openai provider (e.g. http://localhost:11434/v1)")
		}
		client = newOpenAIClient(cfg)
	default:
		return nil, fmt.Errorf("ai: unknown provider %q (want anthropic or openai)", provider)
	}
	max := cfg.MaxAttempts
	if max <= 0 {
		max = 3
	}
	return &Drafter{
		client:      client,
		provider:    provider,
		model:       cfg.Model,
		render:      render,
		prompts:     NewPromptBuilder(helpers),
		fetch:       fetch,
		maxAttempts: max,
	}, nil
}

// Model returns the configured model id.
func (d *Drafter) Model() string { return d.model }

// Provider returns which chat API the drafter uses.
func (d *Drafter) Provider() string { return string(d.provider) }

// draft is the structured output the model must produce.
type draft struct {
	Template string `json:"template"`
	Notes    string `json:"notes"`
}

var draftSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"template": map[string]any{
			"type":        "string",
			"description": "The complete Go text/template. This exact string is saved as transform_template.",
		},
		"notes": map[string]any{
			"type":        "string",
			"description": "One or two sentences for the user: which payload fields were used and any assumption made.",
		},
	},
	"required":             []string{"template", "notes"},
	"additionalProperties": false,
}

// DraftTemplate produces a template for req, verifying it renders against the
// event's sample payload. Render errors are returned to the model for repair
// up to MaxAttempts times; if it never renders, the last error is returned as
// an InvalidArgument so the caller can show it.
func (d *Drafter) DraftTemplate(ctx context.Context, req Request) (*Result, error) {
	if strings.TrimSpace(req.Instructions) == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "instructions are required")
	}
	if req.EventName == "" {
		return nil, svcerrors.Error(svcerrors.InvalidArgument, "event name is required")
	}

	docsText, err := ResolveDocs(ctx, d.fetch, req.DocsURL)
	if err != nil {
		return nil, err
	}

	turns := []turn{{role: "user", text: d.prompts.userPrompt(req, docsText)}}
	system := d.prompts.systemPrompt(false)

	var lastErr error
	for attempt := 1; attempt <= d.maxAttempts; attempt++ {
		resp, err := d.client.complete(ctx, system, turns, draftSchema)
		if err != nil {
			return nil, err
		}
		if resp.refused {
			return nil, svcerrors.Error(svcerrors.FailedPrecondition, "the model declined this request; rephrase the instructions")
		}
		dr, err := parseDraft(resp.text)
		if err != nil {
			return nil, svcerrors.Wrapf(err, svcerrors.Internal, "ai: unusable model response")
		}
		turns = append(turns, turn{role: "assistant", text: resp.text})

		rendered, renderErr := d.render(ctx, req.EventName, dr.Template, req.SamplePayload)
		if renderErr == nil {
			renderErr = checkRendered(rendered)
		}
		if renderErr == nil {
			return &Result{
				Template: dr.Template,
				Rendered: rendered,
				Notes:    dr.Notes,
				Attempts: attempt,
				Model:    d.model,
			}, nil
		}
		lastErr = renderErr
		turns = append(turns, turn{role: "user", text: "Rendering that template against the sample payload failed with:\n\n" +
			renderErr.Error() +
			"\n\nFix the template so it renders. Only use helpers from the catalog and fields that exist in the payload: reading a missing key is an error, so read optional fields with dig or index; compare payload numbers with float literals (3.0, not 3); never leave a {{param}} token. Return the full corrected template."})
	}
	return nil, svcerrors.Wrapf(lastErr, svcerrors.InvalidArgument,
		"drafted template did not render after %d attempts: %v", d.maxAttempts, lastErr)
}

// missingValueMarkers are what Go's text/template and fmt print when a
// template reads a field that is not in the data. The engine deliberately
// does not fail on missing keys (deliveries degrade gracefully), so a draft
// that names the wrong field renders "successfully" with garbage in it.
// Treating these markers as a render failure lets the repair loop catch the
// most common model mistake: a misspelled or mis-cased payload field.
var missingValueMarkers = []string{"<no value>", "%!s(<nil>)", "%!v(<nil>)", "%!d(<nil>)", "%!f(<nil>)", "%!q(<nil>)", "%!t(<nil>)"}

// checkRendered rejects output that rendered without error but is clearly
// wrong: it references a missing field, or it looks like JSON but is not
// valid JSON (the usual cause is hand-quoting instead of the json helper).
func checkRendered(rendered string) error {
	// The json helper escapes < and > as \u003c / \u003e, which would hide
	// "<no value>" inside a JSON string; undo that before scanning.
	scan := strings.NewReplacer(`\u003c`, "<", `\u003e`, ">").Replace(rendered)
	for _, m := range missingValueMarkers {
		if strings.Contains(scan, m) {
			return fmt.Errorf("the rendered output contains %q, so the template reads a field that does not exist in the data; check the field name and case against the sample payload (payload fields live under .payload)", m)
		}
	}
	trimmed := strings.TrimSpace(rendered)
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		if !json.Valid([]byte(trimmed)) {
			return errors.New("the rendered output starts like JSON but is not valid JSON; pipe string, object and array values through the json helper instead of quoting them by hand, and check commas between fields")
		}
	}
	return nil
}

// parseDraft decodes the model's structured reply. Providers that enforce a
// JSON schema return the bare object; smaller or older models sometimes wrap
// it in a code fence or lead with a sentence, so the first {...} object is
// extracted when a direct decode fails.
func parseDraft(raw string) (*draft, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, errors.New("empty response")
	}
	var dr draft
	if err := json.Unmarshal([]byte(raw), &dr); err != nil {
		start, end := strings.Index(raw, "{"), strings.LastIndex(raw, "}")
		if start < 0 || end <= start {
			return nil, fmt.Errorf("decode structured output: %w", err)
		}
		if err2 := json.Unmarshal([]byte(raw[start:end+1]), &dr); err2 != nil {
			return nil, fmt.Errorf("decode structured output: %w", err)
		}
	}
	if strings.TrimSpace(dr.Template) == "" {
		return nil, errors.New("empty template")
	}
	return &dr, nil
}

// mapStatus turns an HTTP status from any provider into a service error the
// REST layer can present. ok reports whether status was handled.
func mapStatus(status int) (error, bool) {
	switch {
	case status == 401 || status == 403:
		return svcerrors.Error(svcerrors.FailedPrecondition, "AI drafting is misconfigured: the provider rejected the API key"), true
	case status == 404:
		return svcerrors.Error(svcerrors.FailedPrecondition, "AI drafting is misconfigured: the provider reports the model or endpoint does not exist"), true
	case status == 429:
		return svcerrors.Error(svcerrors.ResourceExhausted, "AI drafting is rate limited; try again shortly"), true
	case status >= 500:
		return svcerrors.Error(svcerrors.Unavailable, "AI provider is unavailable; try again shortly"), true
	}
	return nil, false
}

func mapTransportError(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return svcerrors.Error(svcerrors.DeadlineExceeded, "AI drafting timed out")
	}
	return svcerrors.Wrapf(err, svcerrors.Unavailable, "ai: request failed")
}
