// Package sources turns external things (cron ticks, Stripe webhooks,
// GitHub webhooks) into Sparrow events.
package sources

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level sources.yaml shape.
type Config struct {
	Sparrow SparrowConfig `yaml:"sparrow"`
	Cron    []CronJob     `yaml:"cron"`
	Webhook WebhookConfig `yaml:"webhook"`
}

// SparrowConfig is the Sparrow server connection. Env vars SPARROW_URL,
// SPARROW_API_KEY and SPARROW_NAMESPACE override the file values.
type SparrowConfig struct {
	URL       string `yaml:"url"`
	APIKey    string `yaml:"api_key"`
	Namespace string `yaml:"namespace"`
}

// CronJob emits a fixed event on a 5-field cron schedule.
type CronJob struct {
	Schedule string            `yaml:"schedule"`
	Event    string            `yaml:"event"`
	Payload  map[string]any    `yaml:"payload"`
	Labels   map[string]string `yaml:"labels"`

	spec        *cronSpec
	payloadJSON json.RawMessage
}

// WebhookConfig configures the webhook-ingest HTTP listener that receives
// provider webhooks (Stripe, GitHub) and republishes them as Sparrow events.
type WebhookConfig struct {
	Listen    string          `yaml:"listen"`
	Providers ProvidersConfig `yaml:"providers"`
}

// ProvidersConfig holds per-provider webhook settings; nil = disabled.
type ProvidersConfig struct {
	Stripe *StripeConfig `yaml:"stripe"`
	GitHub *GitHubConfig `yaml:"github"`
}

// StripeConfig verifies Stripe-Signature and maps Stripe events.
type StripeConfig struct {
	SigningSecret string `yaml:"signing_secret"`
	EventPrefix   string `yaml:"event_prefix"`
}

// GitHubConfig verifies X-Hub-Signature-256 and maps GitHub events.
type GitHubConfig struct {
	Secret      string `yaml:"secret"`
	EventPrefix string `yaml:"event_prefix"`
}

// LoadConfig reads and validates the YAML config file, applying env
// overrides and pre-parsing cron schedules and payloads.
func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if v := os.Getenv("SPARROW_URL"); v != "" {
		cfg.Sparrow.URL = v
	}
	if v := os.Getenv("SPARROW_API_KEY"); v != "" {
		cfg.Sparrow.APIKey = v
	}
	if v := os.Getenv("SPARROW_NAMESPACE"); v != "" {
		cfg.Sparrow.Namespace = v
	}
	if cfg.Sparrow.Namespace == "" {
		cfg.Sparrow.Namespace = "default"
	}
	if cfg.Sparrow.URL == "" {
		return nil, fmt.Errorf("sparrow.url is required (or set SPARROW_URL)")
	}

	for i := range cfg.Cron {
		job := &cfg.Cron[i]
		if job.Event == "" {
			return nil, fmt.Errorf("cron[%d]: event is required", i)
		}
		spec, err := parseCron(job.Schedule)
		if err != nil {
			return nil, fmt.Errorf("cron[%d] schedule %q: %w", i, job.Schedule, err)
		}
		job.spec = spec
		if job.Payload == nil {
			job.payloadJSON = json.RawMessage(`{}`)
		} else {
			b, err := json.Marshal(job.Payload)
			if err != nil {
				return nil, fmt.Errorf("cron[%d] payload: %w", i, err)
			}
			job.payloadJSON = b
		}
	}

	if s := cfg.Webhook.Providers.Stripe; s != nil {
		if s.SigningSecret == "" {
			return nil, fmt.Errorf("webhook.providers.stripe.signing_secret is required")
		}
		if s.EventPrefix == "" {
			s.EventPrefix = "stripe"
		}
	}
	if g := cfg.Webhook.Providers.GitHub; g != nil {
		if g.Secret == "" {
			return nil, fmt.Errorf("webhook.providers.github.secret is required")
		}
		if g.EventPrefix == "" {
			g.EventPrefix = "github"
		}
	}
	if (cfg.Webhook.Providers.Stripe != nil || cfg.Webhook.Providers.GitHub != nil) && cfg.Webhook.Listen == "" {
		cfg.Webhook.Listen = ":8787"
	}

	return &cfg, nil
}
