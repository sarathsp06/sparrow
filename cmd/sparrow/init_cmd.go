package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// runInit writes ~/.sparrow/config.yaml. Values come from flags, or from
// interactive prompts when attached to a terminal. Idempotent: re-running
// overwrites the file.
func runInit(ctx context.Context, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	urlFlag := fs.String("url", "", "Sparrow server URL (default "+defaultServerURL+")")
	apiKeyFlag := fs.String("api-key", "", "API key sent as X-API-Key (optional)")
	nsFlag := fs.String("namespace", "", "default tenant namespace (default "+defaultNamespace+")")
	if err := fs.Parse(args); err != nil {
		return err
	}

	interactive := isTerminal(os.Stdin)
	prompt := func(label, def, flagVal string) string {
		if flagVal != "" {
			return flagVal
		}
		if !interactive {
			return def
		}
		fmt.Fprintf(out, "%s [%s]: ", label, def)
		sc := bufio.NewScanner(os.Stdin)
		if sc.Scan() {
			if v := strings.TrimSpace(sc.Text()); v != "" {
				return v
			}
		}
		return def
	}

	cfg := config{
		ServerURL: strings.TrimRight(prompt("Server URL", defaultServerURL, *urlFlag), "/"),
		APIKey:    prompt("API key (empty for none)", "", *apiKeyFlag),
		Namespace: prompt("Namespace", defaultNamespace, *nsFlag),
	}

	if err := probeServer(ctx, cfg); err != nil {
		fmt.Fprintf(out, "warning: server not reachable at %s (%v) — config written anyway\n", cfg.ServerURL, err)
	} else {
		fmt.Fprintf(out, "server reachable at %s\n", cfg.ServerURL)
	}

	path, err := saveConfig(cfg)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	fmt.Fprintf(out, "wrote %s (namespace %q)\n", path, cfg.Namespace)
	return nil
}

// probeServer checks reachability via GET /health, falling back to the v1 API.
func probeServer(ctx context.Context, cfg config) error {
	client := newAPIClient(cfg)
	client.hc.Timeout = 5 * time.Second
	err := client.do(ctx, http.MethodGet, "/health", nil, nil)
	if err == nil {
		return nil
	}
	if err2 := client.do(ctx, http.MethodGet, "/v1/event-types?limit=1", nil, nil); err2 == nil {
		return nil
	}
	return err
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}
