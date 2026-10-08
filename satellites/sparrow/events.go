package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newEventsCmd() *cobra.Command {
	var activeOnly bool
	cmd := &cobra.Command{
		Use:   "events [name]",
		Short: "List event types, or show one type's detail",
		Long: `List the event types the consumer can subscribe to. Sparrow's own
sparrow.* events are listed only with --consumer _sparrow. With a name
argument, show that event type's schema and sample payload.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			var name string
			if len(args) == 1 {
				name = args[0]
			}
			return runEvents(cmd.Context(), cmd.OutOrStdout(), client, cfg.Consumer, name, activeOnly, outputFmt(cmd))
		},
	}
	cmd.Flags().BoolVar(&activeOnly, "active", false, "only show active event types")
	addOutputFlag(cmd)
	cmd.AddCommand(newEventsExportCmd(), newEventsImportCmd(), newEventsVersionsCmd())
	return cmd
}

// runEvents lists the event types consumer can subscribe to, or shows one
// type's full detail when named.
func runEvents(ctx context.Context, out io.Writer, client *apiClient, consumer, name string, activeOnly bool, format string) error {
	if name != "" {
		et, err := client.getEventType(ctx, name)
		if err != nil {
			return err
		}
		if done, err := renderStructured(out, format, et); done {
			return err
		}
		return printEventType(out, et)
	}

	types, err := client.listEventTypes(ctx, consumer, activeOnly)
	if err != nil {
		return err
	}
	if done, err := renderStructured(out, format, types); done {
		return err
	}
	if len(types) == 0 {
		_, _ = fmt.Fprintln(out, "no event types registered")
		return nil
	}
	sort.Slice(types, func(i, j int) bool { return types[i].Name < types[j].Name })

	pal := newPalette(out)
	_, _ = fmt.Fprintln(out, pal.bold(fmt.Sprintf("%-32s %-8s %-7s %s", "EVENT", "VERSION", "ACTIVE", "DESCRIPTION")))
	for _, et := range types {
		_, _ = fmt.Fprintf(out, "%-32s %-8s %-7s %s\n", et.Name, fmt.Sprintf("v%d", et.Version), yesNo(et.Active), et.Description)
	}
	return nil
}

// printEventType renders one event type's schema and sample payload.
func printEventType(out io.Writer, et eventTypeItem) error {
	pal := newPalette(out)
	_, _ = fmt.Fprintln(out, pal.bold(et.Name))
	if et.Description != "" {
		_, _ = fmt.Fprintln(out, et.Description)
	}
	_, _ = fmt.Fprintf(out, "active: %s\n", yesNo(et.Active))
	if et.Version > 0 {
		_, _ = fmt.Fprintf(out, "version: v%d (history: sparrow events versions %s)\n", et.Version, et.Name)
	}
	if len(et.JSONSchema) > 0 {
		_, _ = fmt.Fprintf(out, "\n%s\n%s\n", pal.dim("schema:"), indentJSON(et.JSONSchema))
	}
	if len(et.SamplePayload) > 0 {
		_, _ = fmt.Fprintf(out, "\n%s\n%s\n", pal.dim("sample payload:"), indentJSON(et.SamplePayload))
	}
	return nil
}

func indentJSON(v any) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprint(v)
	}
	return strings.TrimRight(string(b), "\n")
}
