// Command sparrow-sources turns external things into Sparrow events:
// cron schedules, Stripe webhooks, and GitHub webhooks. It is a thin server
// around the sources packages: each sources.yaml section is that source's
// Config.
package main

import (
	"cmp"
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/sarathsp06/sparrow/satellites/sparrow-sources/sources"
	"github.com/sarathsp06/sparrow/satellites/sparrow-sources/sources/cron"
	"github.com/sarathsp06/sparrow/satellites/sparrow-sources/sources/github"
	"github.com/sarathsp06/sparrow/satellites/sparrow-sources/sources/stripe"
)

const usage = `sparrow-sources: push external events into Sparrow.

Usage:
  sparrow-sources [--config sources.yaml]
  sparrow-sources --version

Config file (YAML):
  sparrow:                        # SPARROW_URL / SPARROW_API_KEY /
    url: http://localhost:8080    # SPARROW_CONSUMER env vars override
    api_key: ""
    consumer: default
  cron:                           # 5-field cron (min hour dom mon dow;
    - schedule: "*/5 * * * *"     # numbers, ranges, steps, *)
      event: report.tick
      payload: {kind: hourly}
      labels: {source: cron}
  webhook:
    listen: :8787
    providers:                    # more providers added over time; each takes
      stripe:                     # a path, a secret, and a Sparrow event prefix
        path: /webhooks/stripe    # you choose this; register it with Stripe
        signing_secret: whsec_...
        sparrow_event_prefix: stripe   # -> stripe.payment_intent.succeeded
      github:
        path: /webhooks/github    # you choose this; set it as the Payload URL
        secret: ...
        sparrow_event_prefix: github   # -> github.pull_request.opened
`

// fileConfig is the sources.yaml shape. A nil provider is disabled.
type fileConfig struct {
	Sparrow sources.SparrowConfig `yaml:"sparrow"`
	Cron    []cron.Job            `yaml:"cron"`
	Webhook struct {
		Listen    string    `yaml:"listen"`
		Providers providers `yaml:"providers"`
	} `yaml:"webhook"`
}

type providers struct {
	Stripe *stripe.Config `yaml:"stripe"`
	GitHub *github.Config `yaml:"github"`
}

func (p providers) any() bool { return p.Stripe != nil || p.GitHub != nil }

// loadConfig reads the YAML file and applies the SPARROW_URL,
// SPARROW_API_KEY and SPARROW_CONSUMER env overrides.
func loadConfig(path string) (*fileConfig, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg fileConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if v := os.Getenv("SPARROW_URL"); v != "" {
		cfg.Sparrow.URL = v
	}
	if v := os.Getenv("SPARROW_API_KEY"); v != "" {
		cfg.Sparrow.APIKey = v
	}
	if v := os.Getenv("SPARROW_CONSUMER"); v != "" {
		cfg.Sparrow.Consumer = v
	}
	if cfg.Sparrow.URL == "" {
		return nil, fmt.Errorf("sparrow.url is required (or set SPARROW_URL)")
	}
	if cfg.Webhook.Listen == "" {
		cfg.Webhook.Listen = ":8787"
	}
	return &cfg, nil
}

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	fs := flag.NewFlagSet("sparrow-sources", flag.ExitOnError)
	configPath := fs.String("config", "sources.yaml", "path to YAML config file")
	showVersion := fs.Bool("version", false, "print the version and exit")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		fs.PrintDefaults()
	}
	_ = fs.Parse(os.Args[1:])
	if *showVersion {
		fmt.Println("sparrow-sources", version)
		return
	}

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	fail := func(msg string, err error) {
		log.Error(msg, "path", *configPath, "error", err)
		os.Exit(1)
	}

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fail("failed to load config", err)
	}
	if len(cfg.Cron) == 0 && !cfg.Webhook.Providers.any() {
		log.Error("nothing to do: configure cron jobs and/or webhook providers")
		os.Exit(1)
	}

	client := sources.NewClient(cfg.Sparrow)
	emitter, err := cron.New(cfg.Cron)
	if err != nil {
		fail("invalid cron config", err)
	}
	var handler http.Handler
	if cfg.Webhook.Providers.any() {
		if handler, err = newWebhookHandler(cfg.Webhook.Providers, client, log); err != nil {
			fail("invalid webhook config", err)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if len(cfg.Cron) > 0 {
		log.Info("starting cron emitter", "jobs", len(cfg.Cron))
		go emitter.Run(ctx, client, log)
	}

	var srv *http.Server
	if handler != nil {
		srv = &http.Server{
			Addr:         cfg.Webhook.Listen,
			Handler:      handler,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		}
		go func() {
			log.Info("webhook listening", "addr", cfg.Webhook.Listen)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("webhook server failed", "error", err)
				stop()
			}
		}()
	}

	<-ctx.Done()
	log.Info("shutting down")
	if srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
}

// newWebhookHandler mounts each configured provider at its path (defaults
// /webhooks/stripe, /webhooks/github). Unknown paths 404.
func newWebhookHandler(cfg providers, p sources.Pusher, log *slog.Logger) (http.Handler, error) {
	mux := http.NewServeMux()
	paths := map[string]string{} // path -> provider, to refuse duplicates
	mount := func(name, path string, h http.Handler, err error) error {
		if err != nil {
			return fmt.Errorf("webhook.providers.%w", err) // "webhook.providers.stripe: ..."
		}
		if other, dup := paths[path]; dup {
			return fmt.Errorf("webhook.providers: %s and %s both use path %s", other, name, path)
		}
		paths[path] = name
		mux.Handle("POST "+path, h)
		return nil
	}
	if c := cfg.Stripe; c != nil {
		h, err := stripe.Handler(*c, p, log)
		if err := mount("stripe", cmp.Or(c.Path, "/webhooks/stripe"), h, err); err != nil {
			return nil, err
		}
	}
	if c := cfg.GitHub; c != nil {
		h, err := github.Handler(*c, p, log)
		if err := mount("github", cmp.Or(c.Path, "/webhooks/github"), h, err); err != nil {
			return nil, err
		}
	}
	return mux, nil
}
