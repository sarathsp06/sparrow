package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/sarathsp06/sparrow/pkg/template"
)

// recipeReport is the outcome of validating a recipe. Errors mean applying it
// would fail or deliver something the destination cannot read; warnings are
// what the built-in catalog additionally requires before a recipe can ship.
type recipeReport struct {
	Valid    bool     `json:"valid"`
	Recipe   string   `json:"recipe,omitempty"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
	// URL and Rendered show the request the recipe would make for the sample
	// event, with params substituted (secret params masked in URL).
	URL      string `json:"url,omitempty"`
	Rendered string `json:"rendered,omitempty"`
}

// sampleRecipePayload is what a template is rendered against when the caller
// gives no payload: a few typed fields plus characters that break naive JSON.
var sampleRecipePayload = map[string]any{
	"id":     "obj_123",
	"amount": 4200,
	"status": "active",
	"note":   "quotes \" and\nnewlines must survive JSON encoding",
}

// validateRecipeYAML decodes recipe YAML and validates it. Keys outside the
// schema are reported as warnings: they are ignored on apply, which usually
// means a typo (requires_tranform) that silently changes behaviour.
func validateRecipeYAML(data []byte, values map[string]string, eventName string, payload map[string]any) recipeReport {
	rep := recipeReport{Errors: []string{}, Warnings: []string{}}
	var r recipe
	if err := yaml.Unmarshal(data, &r); err != nil {
		rep.Errors = append(rep.Errors, "parse: "+err.Error())
		return rep
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	var strict recipe
	if err := dec.Decode(&strict); err != nil {
		var te *yaml.TypeError
		if errors.As(err, &te) {
			for _, e := range te.Errors {
				rep.Warnings = append(rep.Warnings, "ignored on apply: "+e)
			}
		}
	}
	other := validateRecipe(r, values, eventName, payload)
	other.Warnings = append(rep.Warnings, other.Warnings...)
	return other
}

// validateRecipe runs every check a recipe goes through: the structural rules
// applying it enforces, the catalog's shipping rules, and a strict render of
// the transform template against a sample event with params substituted.
func validateRecipe(r recipe, values map[string]string, eventName string, payload map[string]any) recipeReport {
	rep := recipeReport{Recipe: r.Name, Errors: []string{}, Warnings: []string{}}
	fail := func(f string, a ...any) { rep.Errors = append(rep.Errors, fmt.Sprintf(f, a...)) }
	warn := func(f string, a ...any) { rep.Warnings = append(rep.Warnings, fmt.Sprintf(f, a...)) }

	// What applying needs.
	if r.Version != 1 {
		fail("version is %d, want 1", r.Version)
	}
	if r.Name == "" {
		fail("name is empty")
	}
	if r.Webhook.URL == "" {
		fail("webhook.url is empty")
	}
	if err := r.Validate(); err != nil {
		fail("%v", err)
	}

	// What the built-in catalog needs (satellites/recipes/recipes_test.go).
	if r.Description == "" {
		warn("description is empty")
	}
	if strings.TrimSpace(r.Subscription.TransformTemplate) == "" {
		warn("subscription.transform_template is empty: the destination gets Sparrow's default envelope")
	} else if !r.Webhook.RequiresTransform {
		warn("webhook.requires_transform is false: set it when the destination cannot read Sparrow's default envelope (true for every shipped recipe)")
	}
	if strings.TrimSpace(r.Guidance) == "" {
		warn("guidance is empty: AI template drafting uses it to aim for a good message")
	}
	for _, p := range r.Params {
		if p.Prompt == "" || p.Help == "" {
			warn("param %s needs both prompt and help", p.Name)
		}
		if len(p.Prompt) > 40 {
			warn("param %s prompt is %d chars: keep it a short label, details go in help", p.Name, len(p.Prompt))
		}
	}
	if len(rep.Errors) > 0 {
		return rep
	}

	// Substitute params: given values, then defaults, then a placeholder, so
	// a recipe can be checked before anyone has real credentials.
	filled := make(map[string]string, len(r.Params))
	secret := map[string]bool{}
	placeholder := map[string]bool{}
	for _, p := range r.Params {
		switch v, ok := values[p.Name]; {
		case ok:
			filled[p.Name] = v
		case p.Default != "":
			filled[p.Name] = p.Default
		default:
			filled[p.Name] = "example-" + p.Name
			placeholder[p.Name] = true
		}
		secret[p.Name] = p.Secret
	}
	rr, err := substituteRecipe(r, filled)
	if err != nil {
		fail("%v", err)
		return rep
	}
	// A placeholder for a param that holds the whole URL (Slack's
	// webhook_url) can't look like one, so only check a URL built from real
	// values. Apply always has real values.
	urlIsReal := true
	for _, m := range paramToken.FindAllStringSubmatch(r.Webhook.URL, -1) {
		urlIsReal = urlIsReal && !placeholder[m[1]]
	}
	if u, err := url.Parse(rr.URL); urlIsReal && (err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "") {
		fail("webhook.url %q is not an http(s) URL after substituting params", rr.URL)
	}
	masked := make(map[string]string, len(filled))
	for k, v := range filled {
		masked[k] = v
		if secret[k] {
			masked[k] = "****"
		}
	}
	rep.URL, _ = substituteParams(r.Webhook.URL, masked)

	if rr.Template != "" {
		if eventName == "" {
			eventName = "example.event"
		}
		if payload == nil {
			payload = sampleRecipePayload
		}
		ctx := template.NewWebhookContext("evt_sample", eventName, time.Now().UTC().Format(time.RFC3339), 1, payload)
		out, err := template.New().ExecuteWith(rr.Template, ctx, template.ExecOptions{StrictMissingKeys: true})
		switch {
		case err != nil:
			fail("transform_template does not render against the sample event (missing keys are errors, as on a subscription; read optional fields with index or dig): %v", err)
		case declaresJSON(rr.Headers) && !json.Valid(out):
			fail("transform_template output is not valid JSON but Content-Type is JSON; quote interpolated values with the json helper, e.g. {{.event_name | json}}")
		}
		rep.Rendered = string(out)
		for k, v := range filled {
			if secret[k] && v != "" {
				rep.Rendered = strings.ReplaceAll(rep.Rendered, v, "****")
			}
		}
	}
	rep.Valid = len(rep.Errors) == 0
	return rep
}

// declaresJSON reports whether the delivered body is meant to be JSON: an
// explicit JSON Content-Type, or none (Sparrow then sends application/json).
func declaresJSON(headers map[string]string) bool {
	for k, v := range headers {
		if strings.EqualFold(k, "Content-Type") {
			return strings.Contains(strings.ToLower(v), "json")
		}
	}
	return true
}
