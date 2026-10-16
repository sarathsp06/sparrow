// Command sparrow-sinks receives signed Sparrow webhook deliveries and
// forwards them to non-HTTP destinations (SMTP email, S3, OTLP logs). It is a
// thin server around the sinks packages: each sinks.yaml section is that
// sink's Config, mounted at /sinks/<name>.
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

	"gopkg.in/yaml.v3"

	"github.com/sarathsp06/sparrow/satellites/sparrow-sinks/sinks/email"
	"github.com/sarathsp06/sparrow/satellites/sparrow-sinks/sinks/otlp"
	"github.com/sarathsp06/sparrow/satellites/sparrow-sinks/sinks/s3"
)

// fileConfig is the sinks.yaml shape. A nil sink section is disabled.
type fileConfig struct {
	Listen string `yaml:"listen"`
	// WebhookSecret and SigningPublicKey are the defaults for every sink that
	// sets neither of its own.
	WebhookSecret    string        `yaml:"webhook_secret"`
	SigningPublicKey string        `yaml:"signing_public_key"`
	Email            *email.Config `yaml:"email"`
	S3               *s3.Config    `yaml:"s3"`
	OTLP             *otlp.Config  `yaml:"otlp"`
}

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	configPath := flag.String("config", "sinks.yaml", "path to sinks config YAML")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Println("sparrow-sinks", version)
		return
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *configPath, logger); err != nil {
		logger.Error("sparrow-sinks exited", "error", err)
		os.Exit(1)
	}
}

func loadConfig(path string) (*fileConfig, error) {
	b, err := os.ReadFile(path) //nolint:gosec // the operator's own -config path
	if err != nil {
		return nil, err
	}
	var c fileConfig
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.Listen == "" {
		c.Listen = ":8788"
	}
	return &c, nil
}

func run(ctx context.Context, configPath string, logger *slog.Logger) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	handler, err := newHandler(ctx, *cfg, logger)
	if err != nil {
		return fmt.Errorf("%s: %w", configPath, err)
	}

	srv := &http.Server{Addr: cfg.Listen, Handler: handler, ReadHeaderTimeout: 10 * time.Second}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	logger.Info("sparrow-sinks listening", "addr", cfg.Listen)

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		logger.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

// newHandler serves every configured sink: POST /sinks/{email,s3,otlp} plus
// GET /healthz.
func newHandler(ctx context.Context, cfg fileConfig, logger *slog.Logger) (http.Handler, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mount := func(name string, h http.Handler, attrs ...any) {
		path := "/sinks/" + name
		mux.Handle("POST "+path, h)
		logger.Info(name+" sink enabled", append([]any{"path", path}, attrs...)...)
	}

	if cfg.Email == nil && cfg.S3 == nil && cfg.OTLP == nil {
		return nil, errors.New("no sinks configured (need email:, s3:, and/or otlp:)")
	}
	if c := cfg.Email; c != nil {
		e := *c
		inheritVerification(&e.WebhookSecret, &e.SigningPublicKey, cfg)
		h, err := email.Handler(e, logger)
		if err != nil {
			return nil, err
		}
		mount("email", h, "smtp", e.SMTP.Host, "to", e.To)
	}
	if c := cfg.S3; c != nil {
		s := *c
		inheritVerification(&s.WebhookSecret, &s.SigningPublicKey, cfg)
		h, err := s3.Handler(ctx, s, logger)
		if err != nil {
			return nil, err
		}
		mount("s3", h, "bucket", s.Bucket)
	}
	if c := cfg.OTLP; c != nil {
		o := *c
		inheritVerification(&o.WebhookSecret, &o.SigningPublicKey, cfg)
		h, err := otlp.Handler(o, logger)
		if err != nil {
			return nil, err
		}
		mount("otlp", h, "endpoint", o.Endpoint)
	}
	return mux, nil
}

// inheritVerification gives a sink the top-level verification settings when
// it sets neither of its own. A sink that sets either keeps only its own, so
// a per-sink secret is never overridden by a top-level public key.
func inheritVerification(secret, publicKey *string, cfg fileConfig) {
	if *secret == "" && *publicKey == "" {
		*secret, *publicKey = cfg.WebhookSecret, cfg.SigningPublicKey
	}
}
