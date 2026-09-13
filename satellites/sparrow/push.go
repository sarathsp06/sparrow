package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	var data string
	labels := kvFlag{}
	var idempotencyKey string
	cmd := &cobra.Command{
		Use:   "push <event-name>",
		Short: "Push an event occurrence",
		Long: `Push one event occurrence. The payload is an inline JSON object or @file.
If the event type is not registered yet, it is auto-created and the push is
retried once.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			return runPush(cmd.Context(), cmd.OutOrStdout(), client, cfg.Namespace, args[0], data, labels, idempotencyKey)
		},
	}
	f := cmd.Flags()
	f.StringVarP(&data, "data", "d", "{}", "event payload: inline JSON object or @file")
	f.VarP(labels, "label", "l", "event label key=value (repeatable)")
	f.StringVar(&idempotencyKey, "idempotency-key", "", "client-supplied dedup key")
	return cmd
}

// runPush pushes one event occurrence. If the event type is not registered
// yet, it is auto-created and the push retried once.
func runPush(ctx context.Context, out io.Writer, client *apiClient, namespace, event, data string, labels kvFlag, idempotencyKey string) error {
	payload, err := parseJSONArg(data)
	if err != nil {
		return err
	}
	body := pushBody{Payload: payload, Labels: labels, IdempotencyKey: idempotencyKey}

	res, err := client.pushEvent(ctx, namespace, event, body)
	var apiErr *apiError
	if errors.As(err, &apiErr) && (apiErr.Status == http.StatusNotFound || apiErr.Status == http.StatusUnprocessableEntity) {
		// Unknown event type: register it and retry once.
		if cerr := client.createEventType(ctx, event, "created by sparrow push"); cerr != nil {
			return fmt.Errorf("push failed (%v) and auto-creating event type failed: %w", err, cerr)
		}
		_, _ = fmt.Fprintf(out, "event type %q auto-created\n", event)
		res, err = client.pushEvent(ctx, namespace, event, body)
	}
	if err != nil {
		return err
	}
	if res.Duplicate {
		_, _ = fmt.Fprintf(out, "%s (duplicate: idempotency key matched an existing event)\n", res.EventID)
	} else {
		_, _ = fmt.Fprintln(out, res.EventID)
	}
	return nil
}
