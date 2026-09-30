package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newEventsExportCmd() *cobra.Command {
	var prefix, outFile string
	var all bool
	cmd := &cobra.Command{
		Use:   "export [name...]",
		Short: "Export event type definitions to one JSON bundle",
		Long: `Export event type definitions as one JSON bundle file, to import into
another environment with "sparrow events import". Pick types by name, by
--prefix, or --all. The bundle carries no version numbers or timestamps, so
the same definitions produce the same file in any environment.`,
		Example: `  sparrow events export order.created order.shipped -o events.json
  sparrow events export --prefix order. > events.json
  sparrow events export --all -o events.json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			return runEventsExport(cmd.Context(), cmd.OutOrStdout(), client, args, prefix, all, outFile)
		},
	}
	cmd.Flags().StringVar(&prefix, "prefix", "", "export every event type whose name starts with this")
	cmd.Flags().BoolVar(&all, "all", false, "export every event type")
	cmd.Flags().StringVarP(&outFile, "output-file", "f", "", "write the bundle to this file instead of stdout")
	return cmd
}

func runEventsExport(ctx context.Context, out io.Writer, client *apiClient, names []string, prefix string, all bool, outFile string) error {
	selectors := 0
	for _, set := range []bool{len(names) > 0, prefix != "", all} {
		if set {
			selectors++
		}
	}
	if selectors != 1 {
		return errors.New("choose exactly one of: event type names, --prefix, or --all")
	}
	sel := map[string]any{}
	switch {
	case len(names) > 0:
		sel["names"] = names
	case prefix != "":
		sel["prefix"] = prefix
	default:
		sel["all"] = true
	}
	var raw json.RawMessage
	if err := client.do(ctx, http.MethodPost, "/v1/event-types:export", sel, &raw); err != nil {
		return err
	}
	// Keep the server's field order; only indent it.
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err != nil {
		return fmt.Errorf("format bundle: %w", err)
	}
	pretty.WriteByte('\n')

	if outFile == "" {
		_, err := out.Write(pretty.Bytes())
		return err
	}
	if err := os.WriteFile(outFile, pretty.Bytes(), 0o644); err != nil {
		return err
	}
	var b struct {
		Items []json.RawMessage `json:"items"`
	}
	_ = json.Unmarshal(raw, &b)
	_, _ = fmt.Fprintf(out, "wrote %d event type(s) to %s\n", len(b.Items), outFile)
	return nil
}

type importFlags struct {
	file                  string
	dryRun                bool
	acceptVersionMismatch bool
	acceptEdited          bool
	allowBreaking         bool
	pauseAffected         bool
}

func newEventsImportCmd() *cobra.Command {
	var f importFlags
	cmd := &cobra.Command{
		Use:   "import -f <bundle.json>",
		Short: "Import a bundle of event type definitions",
		Long: `Import a JSON bundle made by "sparrow events export" (or written by hand).
Each entry replaces the named event type; types not in the file are never
touched. Everything is applied in one transaction, or nothing is.

Nothing is written, and the command exits non-zero, when:
  - the bundle was exported by a different Sparrow version or format
    (accept with --accept-version-mismatch)
  - the file was edited after export (accept with --accept-edited)
  - a schema change is breaking for existing subscriptions
    (apply with --allow-breaking)

Use --dry-run to see the full result first, including which subscriptions'
templates would fail against the new schema.`,
		Example: `  sparrow events import -f events.json --dry-run
  sparrow events import -f events.json
  cat events.json | sparrow events import -f -`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			return runEventsImport(cmd.Context(), cmd.OutOrStdout(), cmd.InOrStdin(), client, f, outputFmt(cmd))
		},
	}
	cmd.Flags().StringVarP(&f.file, "file", "f", "", `bundle file to import ("-" reads stdin)`)
	cmd.Flags().BoolVar(&f.dryRun, "dry-run", false, "show what would happen without writing anything")
	cmd.Flags().BoolVar(&f.acceptVersionMismatch, "accept-version-mismatch", false, "import a bundle exported by a different Sparrow version or format")
	cmd.Flags().BoolVar(&f.acceptEdited, "accept-edited", false, "import a bundle whose items changed after it was exported")
	cmd.Flags().BoolVar(&f.allowBreaking, "allow-breaking", false, "apply schema changes that are breaking for existing subscriptions")
	cmd.Flags().BoolVar(&f.pauseAffected, "pause-affected", false, "pause subscriptions whose template fails against the new schema")
	_ = cmd.MarkFlagRequired("file")
	addOutputFlag(cmd)
	return cmd
}

type importResultOut struct {
	Applied    bool   `json:"applied"`
	DryRun     bool   `json:"dry_run"`
	ImportedAt string `json:"imported_at"`
	Stamp      struct {
		Status     string   `json:"status"`
		ExportedBy string   `json:"exported_by"`
		Server     string   `json:"server"`
		Warnings   []string `json:"warnings"`
	} `json:"stamp"`
	BlockedBy []string `json:"blocked_by"`
	Items     []struct {
		Name            string   `json:"name"`
		Action          string   `json:"action"`
		Version         int      `json:"version"`
		PreviousVersion int      `json:"previous_version"`
		Changes         []string `json:"changes"`
		ActiveChange    string   `json:"active_change"`
		Blocked         bool     `json:"blocked"`
		Compatibility   *struct {
			Result  string   `json:"result"`
			Reasons []string `json:"reasons"`
		} `json:"compatibility"`
		Subscriptions *struct {
			Failures []struct {
				SubscriptionID string `json:"subscription_id"`
				Consumer       string `json:"consumer"`
				Payload        string `json:"payload"`
				Error          string `json:"error"`
			} `json:"failures"`
			Passed           int      `json:"passed"`
			WithoutTransform int      `json:"without_transform"`
			CatchAll         int      `json:"catch_all"`
			Paused           []string `json:"paused"`
		} `json:"subscriptions"`
	} `json:"items"`
}

func runEventsImport(ctx context.Context, out io.Writer, in io.Reader, client *apiClient, f importFlags, format string) error {
	var raw []byte
	var err error
	if f.file == "-" {
		raw, err = io.ReadAll(in)
	} else {
		raw, err = os.ReadFile(f.file)
	}
	if err != nil {
		return err
	}
	body := map[string]any{}
	if err := json.Unmarshal(raw, &body); err != nil {
		return fmt.Errorf("%s is not a JSON bundle: %w", f.file, err)
	}
	if _, ok := body["items"]; !ok {
		return fmt.Errorf("%s has no items", f.file)
	}
	var ack []string
	if f.acceptVersionMismatch {
		ack = append(ack, "version_differs", "format_unsupported")
	}
	if f.acceptEdited {
		ack = append(ack, "items_changed")
	}
	body["dry_run"] = f.dryRun
	body["acknowledge"] = ack
	body["allow_breaking"] = f.allowBreaking
	if f.pauseAffected {
		body["subscription_policy"] = "pause"
	}

	var res importResultOut
	if err := client.do(ctx, http.MethodPost, "/v1/event-types:import", body, &res); err != nil {
		return err
	}
	if done, err := renderStructured(out, format, res); done {
		if err == nil {
			err = importOutcomeError(res)
		}
		return err
	}
	printImportResult(out, res)
	return importOutcomeError(res)
}

// importOutcomeError makes a blocked import exit non-zero. A dry run exits 0
// but still reports what would block the real import.
func importOutcomeError(res importResultOut) error {
	if res.Applied || (res.DryRun && len(res.BlockedBy) == 0) {
		return nil
	}
	if res.DryRun {
		return fmt.Errorf("the import would be blocked by: %s", strings.Join(res.BlockedBy, ", "))
	}
	return fmt.Errorf("nothing was imported; blocked by: %s", strings.Join(res.BlockedBy, ", "))
}

var blockHints = map[string]string{
	"version_differs":    "exported by a different Sparrow version (--accept-version-mismatch)",
	"format_unsupported": "bundle format is newer than this server (--accept-version-mismatch)",
	"items_changed":      "items were edited after export (--accept-edited)",
	"breaking":           "a schema change is breaking for existing subscriptions (--allow-breaking)",
}

func printImportResult(out io.Writer, res importResultOut) {
	pal := newPalette(out)

	switch res.Stamp.Status {
	case "unsigned":
		_, _ = fmt.Fprintln(out, pal.dim("notice: no stamp (hand-written bundle)"))
	case "warning", "valid":
		if res.Stamp.ExportedBy != res.Stamp.Server {
			_, _ = fmt.Fprintf(out, "exported by %s, this server is %s\n", res.Stamp.ExportedBy, res.Stamp.Server)
		}
	}

	_, _ = fmt.Fprintln(out, pal.bold(fmt.Sprintf("%-32s %-12s %-9s %s", "EVENT", "ACTION", "VERSION", "CHANGES")))
	for _, it := range res.Items {
		version := fmt.Sprintf("v%d", it.Version)
		if it.PreviousVersion > 0 && it.PreviousVersion != it.Version {
			version = fmt.Sprintf("v%d→v%d", it.PreviousVersion, it.Version)
		}
		action := it.Action
		if it.Blocked {
			action = "blocked"
		}
		changes := strings.Join(it.Changes, ",")
		if it.ActiveChange != "" {
			changes += " (" + it.ActiveChange + ")"
		}
		_, _ = fmt.Fprintf(out, "%-32s %-12s %-9s %s\n", it.Name, action, version, changes)
		if c := it.Compatibility; c != nil && c.Result == "breaking" {
			for _, r := range c.Reasons {
				_, _ = fmt.Fprintf(out, "    breaking: %s\n", r)
			}
		}
		if s := it.Subscriptions; s != nil {
			for _, fl := range s.Failures {
				_, _ = fmt.Fprintf(out, "    template fails (%s payload): subscription %s in %s: %s\n", fl.Payload, fl.SubscriptionID, fl.Consumer, fl.Error)
			}
			if s.WithoutTransform > 0 {
				_, _ = fmt.Fprintf(out, "    %d subscription(s) without a transform receive the new payload as is\n", s.WithoutTransform)
			}
			if len(s.Paused) > 0 {
				_, _ = fmt.Fprintf(out, "    paused %d subscription(s)\n", len(s.Paused))
			}
		}
	}

	for _, b := range res.BlockedBy {
		_, _ = fmt.Fprintf(out, "blocked: %s\n", blockHints[b])
	}
	switch {
	case res.Applied:
		_, _ = fmt.Fprintf(out, "imported %d event type(s) at %s\n", len(res.Items), res.ImportedAt)
	case res.DryRun:
		_, _ = fmt.Fprintln(out, "dry run: nothing was written")
	}
}

func newEventsVersionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "versions <name>",
		Short: "List every version of an event type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			return runEventsVersions(cmd.Context(), cmd.OutOrStdout(), client, args[0], outputFmt(cmd))
		},
	}
	addOutputFlag(cmd)
	return cmd
}

type eventTypeVersion struct {
	Version         int            `json:"version"`
	Description     string         `json:"description,omitempty"`
	JSONSchema      map[string]any `json:"event_schema,omitempty"`
	SchemaDefinedAt string         `json:"schema_defined_at,omitempty"`
	CreatedAt       string         `json:"created_at"`
}

func runEventsVersions(ctx context.Context, out io.Writer, client *apiClient, name, format string) error {
	var res struct {
		Items []eventTypeVersion `json:"items"`
	}
	if err := client.do(ctx, http.MethodGet, "/v1/event-types/"+url.PathEscape(name)+"/versions", nil, &res); err != nil {
		return err
	}
	if done, err := renderStructured(out, format, res.Items); done {
		return err
	}
	pal := newPalette(out)
	_, _ = fmt.Fprintln(out, pal.bold(fmt.Sprintf("%-8s %-26s %-8s %s", "VERSION", "CREATED", "SCHEMA", "NOTE")))
	for _, v := range res.Items {
		schema := "none"
		if len(v.JSONSchema) > 0 {
			schema = "yes"
		}
		note := ""
		if v.SchemaDefinedAt != "" {
			note = "schema added " + v.SchemaDefinedAt
		}
		_, _ = fmt.Fprintf(out, "v%-7d %-26s %-8s %s\n", v.Version, v.CreatedAt, schema, note)
	}
	return nil
}
