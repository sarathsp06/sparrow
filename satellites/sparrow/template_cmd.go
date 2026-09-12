package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	webhookclient "github.com/sarathsp06/sparrow/internal/webhooks/client"
)

// runTemplateTest renders a transform template locally with the same engine
// the server uses for deliveries — the debugging story for recipe templates.
func runTemplateTest(args []string, out io.Writer) error {
	fs := flag.NewFlagSet("template test", flag.ContinueOnError)
	eventName := fs.String("event-name", "example.event", "event name exposed as {{.event_name}}")
	payloadArg := fs.String("payload", "{}", "sample payload: inline JSON object or @file")
	file, err := parseWithArg(fs, args, "sparrow template test <template-file> [--event-name n] [--payload json|@file]")
	if err != nil {
		return err
	}
	tmpl, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	payload, err := parseJSONArg(*payloadArg)
	if err != nil {
		return err
	}

	engine := webhookclient.NewTemplateEngine()
	rendered, err := engine.TransformPayload(string(tmpl), webhookclient.NewWebhookTemplateContext(
		"evt_sample", *eventName, time.Now().UTC().Format(time.RFC3339), 1, payload))
	if err != nil {
		return fmt.Errorf("template error: %w", err)
	}
	_, _ = fmt.Fprintln(out, string(rendered))
	return nil
}
