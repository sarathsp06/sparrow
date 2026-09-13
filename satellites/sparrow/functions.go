package main

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

func newFunctionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "functions [name]",
		Short: "List transform template functions, or show one's docs",
		Long: `List template functions available inside a transform_template. With a name
argument, show that function's full documentation.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, _, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			var name string
			if len(args) == 1 {
				name = args[0]
			}
			return runFunctions(cmd.Context(), cmd.OutOrStdout(), client, name, outputFmt(cmd))
		},
	}
	addOutputFlag(cmd)
	return cmd
}

// runFunctions lists transform template functions, or shows one's docs.
func runFunctions(ctx context.Context, out io.Writer, client *apiClient, name, format string) error {
	fns, err := client.listTemplateFunctions(ctx)
	if err != nil {
		return err
	}
	sort.Slice(fns, func(i, j int) bool { return fns[i].Name < fns[j].Name })

	if name != "" {
		for _, f := range fns {
			if f.Name == name {
				if done, err := renderStructured(out, format, f); done {
					return err
				}
				pal := newPalette(out)
				_, _ = fmt.Fprintln(out, pal.bold(f.Name))
				_, _ = fmt.Fprintln(out, f.Description)
				return nil
			}
		}
		return fmt.Errorf("no such template function %q", name)
	}

	if done, err := renderStructured(out, format, fns); done {
		return err
	}
	if len(fns) == 0 {
		_, _ = fmt.Fprintln(out, "no template functions available")
		return nil
	}
	pal := newPalette(out)
	_, _ = fmt.Fprintln(out, pal.bold(fmt.Sprintf("%-16s %s", "FUNCTION", "SUMMARY")))
	for _, f := range fns {
		_, _ = fmt.Fprintf(out, "%-16s %s\n", f.Name, firstLine(f.Description))
	}
	return nil
}

// firstLine returns the first non-empty prose line of s, skipping markdown headings.
func firstLine(s string) string {
	for _, ln := range strings.Split(s, "\n") {
		ln = strings.TrimSpace(ln)
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		return ln
	}
	return ""
}
