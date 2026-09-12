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
	for _, d := range sinkDefs {
		if !d.configured(cfg) {
			continue
		}
		deliver, attrs, err := d.build(ctx, cfg)
		if err != nil {
			return err
		}
		path := "/sinks/" + d.name
		mux.Handle("POST "+path, sinkHandler(d.name, cfg.secretFor(d.secret(cfg)), logger, deliver))
		logger.Info(d.name+" sink enabled", append([]any{"path", path}, attrs...)...)
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
