package main

import (
	"context"
	"fmt"
	"io"
	"sort"

	"github.com/spf13/cobra"
)

func newWebhooksCmd() *cobra.Command {
	var activeOnly bool
	cmd := &cobra.Command{
		Use:   "webhooks [id]",
		Short: "List webhooks, or show one webhook's detail",
		Long: `List registered webhooks and inspect what each one receives. With an id
argument, show that webhook plus its subscriptions.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			var id string
			if len(args) == 1 {
				id = args[0]
			}
			return runWebhooks(cmd.Context(), cmd.OutOrStdout(), client, cfg.Consumer, id, activeOnly, outputFmt(cmd))
		},
	}
	cmd.Flags().BoolVar(&activeOnly, "active", false, "only show active webhooks")
	addOutputFlag(cmd)
	return cmd
}

// runWebhooks lists webhooks, or shows one webhook plus its subscriptions.
func runWebhooks(ctx context.Context, out io.Writer, client *apiClient, consumer, id string, activeOnly bool, format string) error {
	if id != "" {
		hook, err := client.getWebhook(ctx, consumer, id)
		if err != nil {
			return err
		}
		subs, err := client.listSubscriptions(ctx, consumer, id)
		if err != nil {
			return err
		}
		if done, err := renderStructured(out, format, webhookDetail{Webhook: hook, Subscriptions: subs}); done {
			return err
		}
		return printWebhook(out, hook, subs)
	}

	hooks, err := client.listWebhooks(ctx, consumer, activeOnly)
	if err != nil {
		return err
	}
	if done, err := renderStructured(out, format, hooks); done {
		return err
	}
	if len(hooks) == 0 {
		_, _ = fmt.Fprintln(out, "no webhooks registered")
		return nil
	}
	sort.Slice(hooks, func(i, j int) bool { return hooks[i].WebhookID < hooks[j].WebhookID })

	pal := newPalette(out)
	_, _ = fmt.Fprintln(out, pal.bold(fmt.Sprintf("%-36s %-7s %-10s %s", "WEBHOOK ID", "ACTIVE", "HEALTH", "URL")))
	for _, h := range hooks {
		_, _ = fmt.Fprintf(out, "%-36s %-7s %-10s %s\n", h.WebhookID, yesNo(h.Active), h.Health, h.URL)
	}
	return nil
}

// webhookDetail is the structured (-o) shape for a single webhook.
type webhookDetail struct {
	Webhook       webhookOut         `json:"webhook"`
	Subscriptions []subscriptionItem `json:"subscriptions"`
}

// printWebhook renders one webhook and the events it is subscribed to.
func printWebhook(out io.Writer, h webhookOut, subs []subscriptionItem) error {
	pal := newPalette(out)
	_, _ = fmt.Fprintln(out, pal.bold(h.WebhookID))
	_, _ = fmt.Fprintf(out, "url:     %s\n", h.URL)
	_, _ = fmt.Fprintf(out, "active:  %s\n", yesNo(h.Active))
	if h.Health != "" {
		_, _ = fmt.Fprintf(out, "health:  %s\n", h.Health)
	}
	if h.Description != "" {
		_, _ = fmt.Fprintf(out, "note:    %s\n", h.Description)
	}
	if len(subs) == 0 {
		_, _ = fmt.Fprintln(out, pal.dim("\nno subscriptions"))
		return nil
	}
	sort.Slice(subs, func(i, j int) bool { return subs[i].EventName < subs[j].EventName })
	_, _ = fmt.Fprintf(out, "\n%s\n", pal.dim("subscriptions:"))
	for _, s := range subs {
		transform := ""
		if s.TransformEnabled {
			transform = pal.dim("  (transform)")
		}
		_, _ = fmt.Fprintf(out, "  %s%s\n", s.EventName, transform)
	}
	return nil
}
