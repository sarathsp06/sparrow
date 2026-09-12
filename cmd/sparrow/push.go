package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
)

// runPush pushes one event occurrence. If the event type is not registered
// yet, it is auto-created and the push retried once.
func runPush(ctx context.Context, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("push", flag.ContinueOnError)
	urlFlag, apiKeyFlag, nsFlag := configFlags(fs)
	var data string
	fs.StringVar(&data, "d", "{}", "event payload: inline JSON object or @file")
	fs.StringVar(&data, "data", "{}", "event payload: inline JSON object or @file")
	labels := kvFlag{}
	fs.Var(labels, "l", "event label key=value (repeatable)")
	fs.Var(labels, "label", "event label key=value (repeatable)")
	idempotencyKey := fs.String("idempotency-key", "", "client-supplied dedup key")
	event, err := parseWithArg(fs, args, "sparrow push <event-name> [-d json|@file] [-l k=v] [--idempotency-key k]")
	if err != nil {
		return err
	}

	payload, err := parseJSONArg(data)
	if err != nil {
		return err
	}
	cfg, err := resolveConfig(*urlFlag, *apiKeyFlag, *nsFlag)
	if err != nil {
		return err
	}
	client := newAPIClient(cfg)
	body := pushBody{Payload: payload, Labels: labels, IdempotencyKey: *idempotencyKey}

	res, err := client.pushEvent(ctx, cfg.Namespace, event, body)
	var apiErr *apiError
	if errors.As(err, &apiErr) && (apiErr.Status == http.StatusNotFound || apiErr.Status == http.StatusUnprocessableEntity) {
		// Unknown event type: register it and retry once.
		if cerr := client.createEventType(ctx, event, "created by sparrow push"); cerr != nil {
			return fmt.Errorf("push failed (%v) and auto-creating event type failed: %w", err, cerr)
		}
		_, _ = fmt.Fprintf(out, "event type %q auto-created\n", event)
		res, err = client.pushEvent(ctx, cfg.Namespace, event, body)
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
