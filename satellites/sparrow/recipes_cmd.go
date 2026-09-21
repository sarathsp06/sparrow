package main

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/sarathsp06/sparrow/satellites/recipes"
)

// newRecipesCmd lists the recipes built into the binary.
func newRecipesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recipes",
		Short: "List built-in recipes (apply one with 'sparrow use <recipe>')",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			all, err := recipes.All()
			if err != nil {
				return err
			}
			if done, err := renderStructured(cmd.OutOrStdout(), outputFmt(cmd), all); done {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tPARAMS\tDESCRIPTION")
			for _, r := range all {
				names := make([]string, len(r.Params))
				for i, p := range r.Params {
					names[i] = p.Name
				}
				fmt.Fprintf(w, "%s\t%s\t%s\n", r.Name, strings.Join(names, ","), r.Description)
			}
			return w.Flush()
		},
	}
	addOutputFlag(cmd)
	return cmd
}
