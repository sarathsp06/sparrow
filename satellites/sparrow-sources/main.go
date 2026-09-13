// Command sparrow-sources turns external things into Sparrow events:
// cron schedules, Stripe webhooks, GitHub webhooks, and Kafka messages.
package main

import (
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

	"github.com/sarathsp06/sparrow/satellites/sparrow-sources/sources"
)

const usage = `sparrow-sources: push external events into Sparrow.

Usage:
  sparrow-sources [--config sources.yaml]

Config file (YAML):
  sparrow:                        # SPARROW_URL / SPARROW_API_KEY /
    url: http://localhost:8080    # SPARROW_NAMESPACE env vars override
    api_key: ""
    namespace: default
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
  kafka:
    - brokers: [localhost:9092]   # seed brokers (default: localhost:9092)
      topic: events               # topic to consume (default: events)
      group_id: sparrow-sources   # consumer group id (default: sparrow-sources)
      sparrow_event_prefix: kafka # -> kafka.events
`

func main() {
	fs := flag.NewFlagSet("sparrow-sources", flag.ExitOnError)
	configPath := fs.String("config", "sources.yaml", "path to YAML config file")
	fs.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
		fs.PrintDefaults()
	}
	_ = fs.Parse(os.Args[1:])

	log := slog.New(slog.NewTextHandler(os.Stderr, nil))

	cfg, err := sources.LoadConfig(*configPath)
	if err != nil {
		log.Error("failed to load config", "path", *configPath, "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	client := sources.NewClient(cfg.Sparrow)

	if len(cfg.Cron) > 0 {
		log.Info("starting cron emitter", "jobs", len(cfg.Cron))
		go sources.RunCron(ctx, cfg.Cron, client, log)
	}

	if len(cfg.Kafka) > 0 {
		log.Info("starting kafka consumer", "jobs", len(cfg.Kafka))
		sources.RunKafka(ctx, cfg.Kafka, client, log)
	}

	var srv *http.Server
	if cfg.Webhook.Providers.Stripe != nil || cfg.Webhook.Providers.GitHub != nil {
		srv = &http.Server{
			Addr:         cfg.Webhook.Listen,
			Handler:      sources.NewWebhookHandler(cfg.Webhook, client, log),
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

	if len(cfg.Cron) == 0 && srv == nil && len(cfg.Kafka) == 0 {
		log.Error("nothing to do: configure cron jobs, webhook providers, and/or kafka consumers")
		os.Exit(1)
	}

	<-ctx.Done()
	log.Info("shutting down")
	if srv != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
}
