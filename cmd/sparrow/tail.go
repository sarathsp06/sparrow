package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"
)

const tailPollInterval = 2 * time.Second

// runTail polls the deliveries API and prints rows not seen before.
func runTail(ctx context.Context, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("tail", flag.ContinueOnError)
	urlFlag, apiKeyFlag, nsFlag := configFlags(fs)
	status := fs.String("status", "", "filter by delivery status (pending, success, failed, retrying)")
	once := fs.Bool("once", false, "print the current batch and exit (no polling)")
	limit := fs.Int("limit", 50, "max deliveries fetched per poll")
	target, err := parseWithArg(fs, args, "sparrow tail deliveries [--status s] [--once]")
	if err != nil {
		return err
	}
	if target != "deliveries" {
		return fmt.Errorf("usage: sparrow tail deliveries [--status s] [--once]")
	}
	cfg, err := resolveConfig(*urlFlag, *apiKeyFlag, *nsFlag)
	if err != nil {
		return err
	}
	client := newAPIClient(cfg)

	_, _ = fmt.Fprintf(out, "%-20s  %-36s  %-9s  %4s  %-8s  %s\n", "TIME", "EVENT", "STATUS", "CODE", "ATTEMPTS", "URL")
	seen := map[string]bool{}
	webhookURLs := map[string]string{} // webhook_id -> url cache
	for {
		items, err := client.listDeliveries(ctx, cfg.Namespace, *status, *limit)
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
			printDelivery(ctx, out, client, cfg.Namespace, webhookURLs, d)
		}
		if *once {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(tailPollInterval):
		}
	}
}

func printDelivery(ctx context.Context, out io.Writer, client *apiClient, namespace string, urls map[string]string, d deliveryItem) {
	dest, ok := urls[d.WebhookID]
	if !ok {
		if hooks, err := client.listWebhooks(ctx, namespace); err == nil {
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
	_, _ = fmt.Fprintf(out, "%-20s  %-36s  %-9s  %4s  %-8s  %s\n",
		ts, d.EventID, d.Status, code, fmt.Sprintf("%d/%d", d.AttemptCount, d.MaxAttempts), dest)
}
