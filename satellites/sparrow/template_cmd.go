package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/sarathsp06/sparrow/pkg/template"
)

func newTemplateCmd() *cobra.Command {
	var eventName string
	var payloadArg string
	parent := &cobra.Command{
		Use:   "template",
		Short: "Debug transform templates locally",
	}
	test := &cobra.Command{
		Use:   "test <template-file>",
		Short: "Render a transform template locally with the server's engine",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemplateTest(cmd.OutOrStdout(), args[0], eventName, payloadArg)
		},
	}
	test.Flags().StringVar(&eventName, "event-name", "example.event", "event name exposed as {{.event_name}}")
	test.Flags().StringVar(&payloadArg, "payload", "{}", "sample payload: inline JSON object or @file")
	parent.AddCommand(test)
	return parent
}

// runTemplateTest renders a transform template locally with the same engine
// the server uses for deliveries - the debugging story for recipe templates.
func runTemplateTest(out io.Writer, file, eventName, payloadArg string) error {
	tmpl, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	payload, err := parseJSONArg(payloadArg)
	if err != nil {
		return err
	}

	engine := template.NewTemplateEngine()
	rendered, err := engine.TransformPayload(string(tmpl), template.NewWebhookTemplateContext(
		"evt_sample", eventName, time.Now().UTC().Format(time.RFC3339), 1, payload))
	if err != nil {
		return fmt.Errorf("template error: %w", err)
	}
	_, _ = fmt.Fprintln(out, string(rendered))
	return nil
}
