// Package recipes defines the schema for Sparrow adapter recipes:
// declarative YAML files that pair a webhook target with a server-side
// transform template, turning Sparrow deliveries into destination-native
// payloads (Slack, Discord, PagerDuty, ...). The sparrow CLI loads these
// files and registers them via the REST API.
package recipes

// Recipe is one adapter recipe, schema version 1.
//
// Occurrences of {{param "name"}} in Webhook.URL, Webhook.Headers values,
// Webhook.SecretHeaders values, and Subscription.TransformTemplate are
// replaced with user-supplied values at apply time via plain string
// substitution. The (substituted) transform template is then passed verbatim
// to Sparrow and rendered server-side per delivery.
type Recipe struct {
	Version      int          `yaml:"version" json:"version"`
	Name         string       `yaml:"name" json:"name"`
	Description  string       `yaml:"description" json:"description"`
	Params       []Param      `yaml:"params" json:"params,omitempty"`
	Webhook      Webhook      `yaml:"webhook" json:"webhook"`
	Subscription Subscription `yaml:"subscription" json:"subscription"`
}

// Param is a value the user supplies when applying a recipe.
type Param struct {
	Name     string `yaml:"name" json:"name"`
	Prompt   string `yaml:"prompt" json:"prompt"`
	Required bool   `yaml:"required" json:"required"`
}

// Webhook describes the destination endpoint registered for the recipe.
type Webhook struct {
	URL     string            `yaml:"url" json:"url"`
	Headers map[string]string `yaml:"headers" json:"headers,omitempty"`
	// SecretHeaders are HTTP headers whose values are envelope-encrypted at
	// rest and masked in every API response — use for upstream auth tokens
	// and passwords (e.g. Twilio Basic auth, ClickHouse key). Values support
	// {{param "name"}} substitution like URL and Headers.
	SecretHeaders map[string]string `yaml:"secret_headers" json:"secret_headers,omitempty"`
}

// Subscription holds the per-subscription transform applied to deliveries.
type Subscription struct {
	TransformTemplate string `yaml:"transform_template" json:"transform_template"`
}
