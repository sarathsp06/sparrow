package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
)

const tailPollInterval = 2 * time.Second

func newTailCmd() *cobra.Command {
	var status string
	var once bool
	var limit int
	cmd := &cobra.Command{
		Use:   "tail deliveries",
		Short: "Stream delivery rows as they happen",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] != "deliveries" {
				return fmt.Errorf("usage: sparrow tail deliveries [--status s] [--once]")
			}
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			return runTail(cmd.Context(), cmd.OutOrStdout(), client, cfg.Namespace, status, once, limit)
		},
	}
	f := cmd.Flags()
	f.StringVar(&status, "status", "", "filter by delivery status (pending, success, failed, retrying)")
	f.BoolVar(&once, "once", false, "print the current batch and exit (no polling)")
	f.IntVar(&limit, "limit", 50, "max deliveries fetched per poll")
	return cmd
}

// runTail polls the deliveries API and prints rows not seen before.
func runTail(ctx context.Context, out io.Writer, client *apiClient, namespace, status string, once bool, limit int) error {
	pal := newPalette(out)

	header := fmt.Sprintf("%-20s  %-36s  %-9s  %4s  %-8s  %s", "TIME", "EVENT", "STATUS", "CODE", "ATTEMPTS", "URL")
	_, _ = fmt.Fprintln(out, pal.bold(header))
	seen := map[string]bool{}
	webhookURLs := map[string]string{} // webhook_id -> url cache
	for {
		items, err := client.listDeliveries(ctx, namespace, status, limit)
		if err != nil {
			if ctx.Err() != nil {
				return nil // Ctrl-C during request
			}
			return err
		}
		// API returns newest first; print unseen rows oldest first.
		for i := len(items) - 1; i >= 0; i-- {
			d := items[i]
			if seen[d.DeliveryID] {
				continue
			}
			seen[d.DeliveryID] = true
			printDelivery(ctx, out, pal, client, namespace, webhookURLs, d)
		}
		if once {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(tailPollInterval):
		}
	}
}

func printDelivery(ctx context.Context, out io.Writer, pal palette, client *apiClient, namespace string, urls map[string]string, d deliveryItem) {
	dest, ok := urls[d.WebhookID]
	if !ok {
		if hooks, err := client.listWebhooks(ctx, namespace, false); err == nil {
			for _, h := range hooks {
				urls[h.WebhookID] = h.URL
			}
		}
		if dest = urls[d.WebhookID]; dest == "" {
			dest = d.WebhookID
			urls[d.WebhookID] = dest // don't refetch for a deleted webhook
		}
	}
	ts := d.CreatedAt
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		ts = t.Local().Format("2006-01-02 15:04:05")
	}
	code := "-"
	if d.ResponseCode != 0 {
		code = fmt.Sprint(d.ResponseCode)
	}
	statusCell := pal.status(d.Status, fmt.Sprintf("%-9s", d.Status))
	_, _ = fmt.Fprintf(out, "%-20s  %-36s  %s  %4s  %-8s  %s\n",
		ts, d.EventID, statusCell, code, fmt.Sprintf("%d/%d", d.AttemptCount, d.MaxAttempts), dest)
}
