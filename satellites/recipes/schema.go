// Package recipes defines the schema for Sparrow adapter recipes:
// declarative YAML files that pair a webhook target with a server-side
// transform template, turning Sparrow deliveries into destination-native
// payloads (Slack, Discord, PagerDuty, ...). The sparrow CLI loads these
// files and registers them via the REST API.
package recipes

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
)

// Recipe is one adapter recipe, schema version 1.
//
// Occurrences of {{param "name"}} in Webhook.URL, Webhook.Headers values,
// Webhook.SecretHeaders values, and Subscription.TransformTemplate are
// replaced with user-supplied values at apply time via plain string
// substitution. The (substituted) transform template is then passed verbatim
// to Sparrow and rendered server-side per delivery.
type Recipe struct {
	Version     int     `yaml:"version" json:"version" doc:"Recipe schema version. Always 1."`
	Name        string  `yaml:"name" json:"name" doc:"Recipe name, e.g. slack. Unique among shipped recipes."`
	Description string  `yaml:"description" json:"description" doc:"One-line summary of the destination the recipe targets."`
	Params      []Param `yaml:"params" json:"params,omitempty" doc:"Values the user supplies when applying the recipe. Each {{param \"name\"}} token in webhook.url, webhook.headers, webhook.secret_headers and subscription.transform_template is replaced with the value by plain string substitution before registration."`
	// Consumer suggests the consumer to register the recipe under, when it
	// matters (e.g. sendgrid's alert emails only work under _sparrow).
	Consumer     *ConsumerHint `yaml:"consumer,omitempty" json:"consumer,omitempty" doc:"The consumer the recipe is meant to be registered under, when it matters."`
	Webhook      Webhook       `yaml:"webhook" json:"webhook" doc:"The webhook to register."`
	Subscription Subscription  `yaml:"subscription" json:"subscription" doc:"The transform given to the subscriptions created for the webhook."`
	// Guidance describes what an ideal message for this destination looks
	// like (tone, length, structure, a rendered example). It is advice, not
	// a rule: nothing checks templates against it. AI template drafting
	// gives it to the model, below the user's instructions in precedence.
	Guidance string `yaml:"guidance,omitempty" json:"guidance,omitempty" doc:"What an ideal message for this destination looks like. Advice for template authors and AI drafting, not a rule."`
}

// ConsumerHint names the consumer a recipe is meant for and why.
type ConsumerHint struct {
	Name string `yaml:"name" json:"name" doc:"Consumer to register the recipe under, e.g. _sparrow."`
	Note string `yaml:"note" json:"note" doc:"Why, and what the recipe does under any other consumer."`
}

// Param is a value the user supplies when applying a recipe.
type Param struct {
	Name string `yaml:"name" json:"name" doc:"Param name, as referenced by {{param \"name\"}}."`
	// Prompt is the short label shown for the param.
	Prompt   string `yaml:"prompt" json:"prompt" doc:"Short label shown for the param."`
	Required bool   `yaml:"required" json:"required" doc:"Whether a value must be supplied."`
	Default  string `yaml:"default,omitempty" json:"default,omitempty" doc:"Value used when none is supplied."`
	Secret   bool   `yaml:"secret,omitempty" json:"secret,omitempty" doc:"Whether the value is sensitive (a token or password); clients should mask the input."`
	// Help is a sentence on what the value is and where to find it.
	Help string `yaml:"help,omitempty" json:"help,omitempty" doc:"What the value is and where to find it."`
	// Example is a sample value, shown as the input's placeholder.
	Example string `yaml:"example,omitempty" json:"example,omitempty" doc:"A sample value."`
	// Enum, when set, lists the only accepted values.
	Enum []string `yaml:"enum,omitempty" json:"enum,omitempty" doc:"The only accepted values, when set."`
	// DocsURL links the destination's documentation for this value.
	DocsURL             string `yaml:"docs_url,omitempty" json:"docs_url,omitempty" doc:"Destination documentation for this value."`
	ActivationRequired  bool   `yaml:"activation_required,omitempty" json:"activation_required,omitempty" doc:"A missing value blocks enabling an auto-provisioned recipe."`
	MustOverrideDefault bool   `yaml:"must_override_default,omitempty" json:"must_override_default,omitempty" doc:"The default is a placeholder: it must be replaced with a real value before the recipe is applied."`
}

// Webhook describes the destination endpoint registered for the recipe.
type Webhook struct {
	URL     string            `yaml:"url" json:"url" doc:"Destination URL; may contain {{param}} tokens."`
	Headers map[string]string `yaml:"headers" json:"headers,omitempty" doc:"Static headers sent with every delivery; values may contain {{param}} tokens."`
	// SecretHeaders are HTTP headers whose values are envelope-encrypted at
	// rest and masked in every API response — use for upstream auth tokens
	// and passwords (e.g. Twilio Basic auth, ClickHouse key). Values support
	// {{param "name"}} substitution like URL and Headers.
	SecretHeaders map[string]string `yaml:"secret_headers" json:"secret_headers,omitempty" doc:"Headers whose values are encrypted at rest and masked in API responses (upstream tokens, passwords); values may contain {{param}} tokens."`
	// RequiresTransform marks a destination that only accepts the recipe's
	// transformed payload. Applying the recipe registers the webhook with
	// requires_transform, so Sparrow refuses any of its subscriptions without
	// an enabled transform and never sends it the default envelope.
	RequiresTransform bool `yaml:"requires_transform,omitempty" json:"requires_transform,omitempty" doc:"The destination only accepts the transformed payload; the webhook is registered with requires_transform."`
}

// Subscription holds the per-subscription transform applied to deliveries.
type Subscription struct {
	TransformTemplate string `yaml:"transform_template" json:"transform_template" doc:"Go template rendered per delivery to build the destination-native body; may contain {{param}} tokens."`
}

// paramToken matches the exact substitution token the CLI replaces at apply
// time: {{param "name"}}.
var paramToken = regexp.MustCompile(`\{\{param "([^"]+)"\}\}`)

// Validate checks structural invariants the YAML schema can't express: every
// {{param "x"}} token in the URL, headers, and transform template must
// reference a declared param, so a typo can't silently ship a literal token.
func (r *Recipe) Validate() error {
	declared := make(map[string]bool, len(r.Params))
	for _, p := range r.Params {
		if p.Name == "" {
			return fmt.Errorf("recipe %s: param with empty name", r.Name)
		}
		declared[p.Name] = true
		if len(p.Enum) > 0 && p.Default != "" && !slices.Contains(p.Enum, p.Default) {
			return fmt.Errorf("recipe %s: param %s default %q is not one of %v", r.Name, p.Name, p.Default, p.Enum)
		}
		if p.DocsURL != "" {
			if u, err := url.Parse(p.DocsURL); err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host == "" {
				return fmt.Errorf("recipe %s: param %s docs_url %q is not an http(s) URL", r.Name, p.Name, p.DocsURL)
			}
		}
	}
	if r.Consumer != nil && strings.TrimSpace(r.Consumer.Name) == "" {
		return fmt.Errorf("recipe %s: consumer hint without a name", r.Name)
	}
	if r.Webhook.RequiresTransform && strings.TrimSpace(r.Subscription.TransformTemplate) == "" {
		return fmt.Errorf("recipe %s: webhook.requires_transform needs a subscription.transform_template", r.Name)
	}
	fields := []string{r.Webhook.URL, r.Subscription.TransformTemplate}
	for _, v := range r.Webhook.Headers {
		fields = append(fields, v)
	}
	for _, v := range r.Webhook.SecretHeaders {
		fields = append(fields, v)
	}
	for _, s := range fields {
		for _, m := range paramToken.FindAllStringSubmatch(s, -1) {
			if !declared[m[1]] {
				return fmt.Errorf("recipe %s: {{param %q}} references undeclared param", r.Name, m[1])
			}
		}
	}
	return nil
}
