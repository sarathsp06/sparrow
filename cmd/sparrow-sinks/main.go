package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("config", "sinks.yaml", "path to sinks config YAML")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, *configPath, logger); err != nil {
		logger.Error("sparrow-sinks exited", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, configPath string, logger *slog.Logger) error {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	if cfg.Email != nil {
		sink, err := newEmailSink(*cfg.Email)
		if err != nil {
			return err
		}
		mux.Handle("POST /sinks/email", sinkHandler("email", cfg.secretFor(cfg.Email.WebhookSecret), logger, sink.deliver))
		logger.Info("email sink enabled", "path", "/sinks/email", "smtp", cfg.Email.SMTP.Host)
	}
	if cfg.S3 != nil {
		sink, err := newS3Sink(ctx, *cfg.S3)
		if err != nil {
			return err
		}
		mux.Handle("POST /sinks/s3", sinkHandler("s3", cfg.secretFor(cfg.S3.WebhookSecret), logger, sink.deliver))
		logger.Info("s3 sink enabled", "path", "/sinks/s3", "bucket", cfg.S3.Bucket)
	}

	srv := &http.Server{Addr: cfg.Listen, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
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
