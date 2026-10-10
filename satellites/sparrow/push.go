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
			return runPush(cmd.Context(), cmd.OutOrStdout(), client, cfg.Consumer, args[0], data, labels, idempotencyKey)
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
func runPush(ctx context.Context, out io.Writer, client *apiClient, consumer, event, data string, labels kvFlag, idempotencyKey string) error {
	payload, err := parseJSONArg(data)
	if err != nil {
		return err
	}
	body := pushBody{Payload: payload, Labels: labels, IdempotencyKey: idempotencyKey}

	res, created, err := pushWithAutoCreate(ctx, client, consumer, event, body)
	if created {
		_, _ = fmt.Fprintf(out, "event type %q auto-created\n", event)
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

// pushWithAutoCreate pushes one event; when the event type is unknown it is
// registered and the push retried once. created reports whether that happened.
func pushWithAutoCreate(ctx context.Context, client *apiClient, consumer, event string, body pushBody) (res pushResult, created bool, err error) {
	res, err = client.pushEvent(ctx, consumer, event, body)
	var apiErr *apiError
	if !errors.As(err, &apiErr) || (apiErr.Status != http.StatusNotFound && apiErr.Status != http.StatusUnprocessableEntity) {
		return res, false, err
	}
	if cerr := client.createEventType(ctx, event, "created by sparrow push"); cerr != nil {
		return res, false, fmt.Errorf("push failed (%v) and auto-creating event type failed: %w", err, cerr)
	}
	res, err = client.pushEvent(ctx, consumer, event, body)
	return res, true, err
}
