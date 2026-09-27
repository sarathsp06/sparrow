package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func newAccessLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "access-link",
		Short: "Share UI access without pasting the API key",
	}
	cmd.AddCommand(newAccessLinkCreateCmd())
	return cmd
}

func newAccessLinkCreateCmd() *cobra.Command {
	var ttl time.Duration
	var uiURL string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Mint a one-time link that signs a browser into the web UI",
		Long: `Mint a one-time link that signs a browser into the Sparrow web UI.

Opening the link makes the UI exchange it for the API key once and remember
the key in that browser. The link stops working after first use, after --ttl,
or when SPARROW_API_KEY is rotated. Whoever opens it holds the real API key,
so send it only to people who should have admin access.

The link points at --ui-url, which defaults to the server URL (right when the
server serves the UI itself). Pass the UI's address if it is hosted separately.`,
		Example: `  sparrow access-link create --ttl 15m
  sparrow access-link create --ttl 1h --ui-url https://sparrow.example.com`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			if uiURL == "" {
				uiURL = cfg.ServerURL
			}
			return runAccessLinkCreate(cmd.Context(), cmd.OutOrStdout(), client, uiURL, ttl, outputFmt(cmd))
		},
	}
	cmd.Flags().DurationVar(&ttl, "ttl", 15*time.Minute, "how long the link stays valid if unused (max 24h)")
	cmd.Flags().StringVar(&uiURL, "ui-url", "", "base URL of the web UI (default: the server URL)")
	addOutputFlag(cmd)
	return cmd
}

type accessLinkResult struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func runAccessLinkCreate(ctx context.Context, out io.Writer, client *apiClient, uiURL string, ttl time.Duration, format string) error {
	if ttl < time.Second || ttl > 24*time.Hour {
		return fmt.Errorf("--ttl must be between 1s and 24h, got %s", ttl)
	}
	link, err := client.createAccessLink(ctx, int64(ttl/time.Second))
	if err != nil {
		return err
	}
	res := accessLinkResult{URL: strings.TrimRight(uiURL, "/") + link.Path, ExpiresAt: link.ExpiresAt}
	if done, err := renderStructured(out, format, res); done {
		return err
	}
	_, _ = fmt.Fprintln(out, res.URL)
	_, _ = fmt.Fprintf(out, "single use, expires %s (in %s)\n", res.ExpiresAt.Local().Format(time.RFC1123), time.Until(res.ExpiresAt).Round(time.Second))
	return nil
}

type accessLinkOut struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	Path      string    `json:"path"`
}

func (c *apiClient) createAccessLink(ctx context.Context, ttlSeconds int64) (accessLinkOut, error) {
	var out accessLinkOut
	q := url.Values{"ttl_seconds": {fmt.Sprint(ttlSeconds)}}
	err := c.do(ctx, "POST", "/v1/access-links?"+q.Encode(), nil, &out)
	return out, err
}
