package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Configure server URL, API key, and namespace",
		Long: `Write ~/.sparrow/config.yaml with the server URL, API key, and default
namespace. Values come from --url/--api-key/--namespace, or from interactive
prompts when stdin is a terminal. Re-running overwrites the file.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			urlFlag, _ := cmd.Flags().GetString("url")
			apiKeyFlag, _ := cmd.Flags().GetString("api-key")
			nsFlag, _ := cmd.Flags().GetString("namespace")
			return runInit(cmd.Context(), cmd.OutOrStdout(), urlFlag, apiKeyFlag, nsFlag)
		},
	}
}

// runInit writes ~/.sparrow/config.yaml. Values come from flags, or from
// interactive prompts when attached to a terminal. Idempotent: re-running
// overwrites the file.
func runInit(ctx context.Context, out io.Writer, urlFlag, apiKeyFlag, nsFlag string) error {

	interactive := isTerminal(os.Stdin)
	prompt := func(label, def, flagVal string) string {
		if flagVal != "" {
			return flagVal
		}
		if !interactive {
			return def
		}
		_, _ = fmt.Fprintf(out, "%s [%s]: ", label, def)
		sc := bufio.NewScanner(os.Stdin)
		if sc.Scan() {
			if v := strings.TrimSpace(sc.Text()); v != "" {
				return v
			}
		}
		return def
	}

	cfg := config{
		ServerURL: strings.TrimRight(prompt("Server URL", defaultServerURL, urlFlag), "/"),
		APIKey:    prompt("API key (empty for none)", "", apiKeyFlag),
		Namespace: prompt("Namespace", defaultNamespace, nsFlag),
	}

	if err := probeServer(ctx, cfg); err != nil {
		_, _ = fmt.Fprintf(out, "warning: server not reachable at %s (%v) — config written anyway\n", cfg.ServerURL, err)
	} else {
		_, _ = fmt.Fprintf(out, "server reachable at %s\n", cfg.ServerURL)
	}

	path, err := saveConfig(cfg)
	if err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	_, _ = fmt.Fprintf(out, "wrote %s (namespace %q)\n", path, cfg.Namespace)
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
