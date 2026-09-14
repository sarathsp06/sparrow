package main

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newStatsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show aggregate delivery statistics for the consumer",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			return runStats(cmd.Context(), cmd.OutOrStdout(), client, cfg.Consumer, outputFmt(cmd))
		},
	}
	addOutputFlag(cmd)
	return cmd
}

// runStats prints delivery statistics for the configured consumer.
func runStats(ctx context.Context, out io.Writer, client *apiClient, consumer, format string) error {
	stats, err := client.getConsumerStats(ctx, consumer)
	if err != nil {
		return err
	}
	if done, err := renderStructured(out, format, stats); done {
		return err
	}

	pal := newPalette(out)
	_, _ = fmt.Fprintln(out, pal.bold("consumer: "+consumer))
	_, _ = fmt.Fprintf(out, "webhooks:     %d (%d active)\n", stats.TotalWebhooks, stats.ActiveWebhooks)
	_, _ = fmt.Fprintf(out, "deliveries:   %d\n", stats.TotalDeliveries)
	_, _ = fmt.Fprintf(out, "  succeeded:  %d\n", stats.SuccessfulDeliveries)
	_, _ = fmt.Fprintf(out, "  failed:     %d\n", stats.FailedDeliveries)
	_, _ = fmt.Fprintf(out, "  pending:    %d\n", stats.PendingDeliveries)
	_, _ = fmt.Fprintf(out, "success rate: %.1f%%\n", stats.SuccessRate*100)
	return nil
}
