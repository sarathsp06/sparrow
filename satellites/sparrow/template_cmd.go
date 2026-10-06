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
	var missingKey string
	parent := &cobra.Command{
		Use:   "template",
		Short: "Debug transform templates locally",
	}
	test := &cobra.Command{
		Use:   "test <template-file>",
		Short: "Render a transform template locally with the server's engine",
		Long: `Render a transform template locally, with the same engine the server uses
for deliveries, against a sample payload. Pass "-" as the file to read the
template from stdin (a heredoc works well).`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTemplateTest(cmd.InOrStdin(), cmd.OutOrStdout(), args[0], eventName, payloadArg, missingKey)
		},
	}
	test.Flags().StringVar(&eventName, "event-name", "example.event", "event name exposed as {{.event_name}}")
	test.Flags().StringVar(&payloadArg, "payload", "{}", "sample payload: inline JSON object or @file")
	test.Flags().StringVar(&missingKey, "missing-key", "error", `how a key the payload lacks renders: "error" (the subscription default) or "zero"`)
	parent.AddCommand(test)
	return parent
}

// runTemplateTest renders a transform template locally with the same engine
// the server uses for deliveries - the debugging story for recipe templates.
// Like a subscription, it renders missing keys as errors unless missingKey is
// "zero". A file of "-" reads the template from in (stdin).
func runTemplateTest(in io.Reader, out io.Writer, file, eventName, payloadArg, missingKey string) error {
	if missingKey != "error" && missingKey != "zero" {
		return fmt.Errorf(`--missing-key must be "error" or "zero", got %q`, missingKey)
	}
	var tmpl []byte
	var err error
	if file == "-" {
		tmpl, err = io.ReadAll(in)
	} else {
		tmpl, err = os.ReadFile(file)
	}
	if err != nil {
		return err
	}
	payload, err := parseJSONArg(payloadArg)
	if err != nil {
		return err
	}

	engine := template.NewTemplateEngine()
	rendered, err := engine.TransformPayloadWith(string(tmpl), template.NewWebhookTemplateContext(
		"evt_sample", eventName, time.Now().UTC().Format(time.RFC3339), 1, payload),
		template.ExecOptions{StrictMissingKeys: missingKey == "error"})
	if err != nil {
		return fmt.Errorf("template error: %w", err)
	}
	_, _ = fmt.Fprintln(out, string(rendered))
	return nil
}
