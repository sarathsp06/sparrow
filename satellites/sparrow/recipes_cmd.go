package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/sarathsp06/sparrow/satellites/recipes"
)

// catalogEntry is one recipe and where it comes from: "built-in", or the
// local file that defines it (local files override built-ins by name).
type catalogEntry struct {
	recipe recipe
	source string
}

// catalog lists every recipe this CLI can apply by name: the built-ins
// embedded in the binary, then ./recipes/*.yaml and $SPARROW_RECIPES_DIR/*.yaml,
// in resolveRecipe's precedence. A local file that does not parse is skipped
// here (sparrow recipes validate reports why).
func catalog() ([]catalogEntry, error) {
	builtin, err := recipes.All()
	if err != nil {
		return nil, err
	}
	byName := make(map[string]catalogEntry, len(builtin))
	for _, r := range builtin {
		byName[r.Name] = catalogEntry{recipe: r, source: "built-in"}
	}
	// Lowest precedence first, so ./recipes wins over $SPARROW_RECIPES_DIR.
	var dirs []string
	if dir := os.Getenv("SPARROW_RECIPES_DIR"); dir != "" {
		dirs = append(dirs, dir)
	}
	dirs = append(dirs, "recipes")
	for _, dir := range dirs {
		files, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
		for _, path := range files {
			name := strings.TrimSuffix(filepath.Base(path), ".yaml")
			r, err := loadRecipe(path)
			if err != nil {
				continue
			}
			r.Name = name
			byName[name] = catalogEntry{recipe: r, source: path}
		}
	}
	out := make([]catalogEntry, 0, len(byName))
	for _, e := range byName {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].recipe.Name < out[j].recipe.Name })
	return out, nil
}

// newRecipesCmd lists recipes and shows or validates one.
func newRecipesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recipes [name]",
		Short: "List recipes, or show one (apply one with 'sparrow use <recipe>')",
		Long: `List the recipes this CLI can apply: the built-in catalog, plus local
./recipes/<name>.yaml and $SPARROW_RECIPES_DIR/<name>.yaml files, which
override a built-in of the same name. Params marked * are required.

'sparrow recipes <name>' is short for 'sparrow recipes show <name>'.`,
		Example: `  sparrow recipes
  sparrow recipes show pagerduty
  sparrow recipes validate --file ./recipes/teams.yaml -e order.created`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				return showRecipe(cmd.OutOrStdout(), outputFmt(cmd), args[0])
			}
			all, err := catalog()
			if err != nil {
				return err
			}
			if done, err := renderStructured(cmd.OutOrStdout(), outputFmt(cmd), catalogRecipes(all)); done {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
			_, _ = fmt.Fprintln(w, "NAME\tPARAMS\tSOURCE\tDESCRIPTION")
			for _, e := range all {
				names := make([]string, len(e.recipe.Params))
				for i, p := range e.recipe.Params {
					names[i] = p.Name
					if p.Required && p.Default == "" {
						names[i] += "*"
					}
				}
				_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.recipe.Name, strings.Join(names, ","), e.source, e.recipe.Description)
			}
			return w.Flush()
		},
	}
	addOutputFlag(cmd)
	cmd.AddCommand(newRecipeShowCmd(), newRecipeValidateCmd())
	return cmd
}

func catalogRecipes(all []catalogEntry) []recipe {
	out := make([]recipe, len(all))
	for i, e := range all {
		out[i] = e.recipe
	}
	return out
}

func newRecipeShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Show a recipe: its params, destination, headers and template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return showRecipe(cmd.OutOrStdout(), outputFmt(cmd), args[0])
		},
	}
	addOutputFlag(cmd)
	return cmd
}

func showRecipe(out io.Writer, format, name string) error {
	r, err := resolveRecipe(name, "")
	if err != nil {
		return err
	}
	if done, err := renderStructured(out, format, r); done {
		return err
	}
	source := "built-in"
	if path, _ := localRecipePath(name); path != "" {
		source = path
	}
	_, _ = fmt.Fprintf(out, "%s: %s\nsource: %s\n", r.Name, r.Description, source)
	if r.Consumer != nil {
		_, _ = fmt.Fprintf(out, "meant for consumer %q: %s\n", r.Consumer.Name, r.Consumer.Note)
	}
	_, _ = fmt.Fprintln(out, "\nparams:")
	if len(r.Params) == 0 {
		_, _ = fmt.Fprintln(out, "  (none)")
	}
	for _, p := range r.Params {
		var tags []string
		if p.Required && p.Default == "" {
			tags = append(tags, "required")
		}
		if p.Default != "" {
			def := "default " + p.Default
			if p.MustOverrideDefault {
				def += ", must be changed"
			}
			tags = append(tags, def)
		}
		if p.Secret {
			tags = append(tags, "secret")
		}
		if len(p.Enum) > 0 {
			tags = append(tags, "one of "+strings.Join(p.Enum, "|"))
		}
		line := "  " + p.Name
		if p.Prompt != "" {
			line += ": " + p.Prompt
		}
		if len(tags) > 0 {
			line += " (" + strings.Join(tags, "; ") + ")"
		}
		_, _ = fmt.Fprintln(out, line)
		if p.Help != "" {
			_, _ = fmt.Fprintf(out, "      %s\n", p.Help)
		}
		if p.Example != "" {
			_, _ = fmt.Fprintf(out, "      e.g. %s\n", p.Example)
		}
		if p.DocsURL != "" {
			_, _ = fmt.Fprintf(out, "      docs: %s\n", p.DocsURL)
		}
	}
	_, _ = fmt.Fprintf(out, "\nwebhook: %s\n", r.Webhook.URL)
	for k, v := range r.Webhook.Headers {
		_, _ = fmt.Fprintf(out, "  header %s: %s\n", k, v)
	}
	for k := range r.Webhook.SecretHeaders {
		_, _ = fmt.Fprintf(out, "  secret header %s\n", k)
	}
	if r.Webhook.RequiresTransform {
		_, _ = fmt.Fprintln(out, "  requires a transform: the destination cannot read Sparrow's default envelope")
	}
	if t := strings.TrimSpace(r.Subscription.TransformTemplate); t != "" {
		_, _ = fmt.Fprintf(out, "\ntransform template:\n%s\n", indent(t, "  "))
	}
	if g := strings.TrimSpace(r.Guidance); g != "" {
		_, _ = fmt.Fprintf(out, "\nguidance:\n%s\n", indent(g, "  "))
	}
	_, _ = fmt.Fprintf(out, "\napply it with: sparrow use %s -e <event>%s\n", r.Name, paramFlags(r))
	return nil
}

// paramFlags is the --param flags a 'sparrow use' of r needs at minimum.
func paramFlags(r recipe) string {
	var b strings.Builder
	for _, p := range r.Params {
		if p.Required && (p.Default == "" || p.MustOverrideDefault) {
			b.WriteString(" --param " + p.Name + "=...")
		}
	}
	return b.String()
}

func indent(s, prefix string) string {
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}

func newRecipeValidateCmd() *cobra.Command {
	var file, event, payloadArg string
	params := kvFlag{}
	cmd := &cobra.Command{
		Use:   "validate [name]",
		Short: "Check a recipe without applying it, and show what it renders",
		Long: `Check a recipe without applying it: the schema and param rules, the built-in
catalog's shipping rules (as warnings), and a strict render of the transform
template with params substituted (placeholders for any not given).

The render uses --payload when given, else the registered sample payload of
--event on the server, else a built-in sample. Exits non-zero when the recipe
has errors.`,
		Example: `  sparrow recipes validate slack -e order.created
  sparrow recipes validate --file ./recipes/teams.yaml --param webhook_url=https://example.com/x
  sparrow recipes validate --file draft.yaml --payload '{"id":"ord_1","total":12}' -o json`,
		Args: func(cmd *cobra.Command, args []string) error {
			if (file == "") == (len(args) == 0) {
				return fmt.Errorf("give a recipe name or --file, not both")
			}
			return cobra.MaximumNArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			path := file
			if path == "" {
				path, _ = localRecipePath(args[0])
			}
			var payload map[string]any
			if payloadArg != "" {
				var err error
				if payload, err = parseJSONArg(payloadArg); err != nil {
					return fmt.Errorf("--payload: %w", err)
				}
			}
			source := "the --payload sample"
			if payload == nil {
				source = "the built-in sample payload"
				if event != "" {
					if client, _, err := clientFromCmd(cmd); err == nil {
						payload, source = eventSample(cmd.Context(), client, event)
					}
				}
			}
			var rep recipeReport
			if path != "" {
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				rep = validateRecipeYAML(data, params, event, payload)
			} else {
				r, err := resolveRecipe(args[0], "")
				if err != nil {
					return err
				}
				rep = validateRecipe(r, params, event, payload)
			}
			out := cmd.OutOrStdout()
			if done, err := renderStructured(out, outputFmt(cmd), rep); done {
				if err == nil && !rep.Valid {
					err = fmt.Errorf("recipe has %d error(s)", len(rep.Errors))
				}
				return err
			}
			for _, e := range rep.Errors {
				_, _ = fmt.Fprintf(out, "error: %s\n", e)
			}
			for _, w := range rep.Warnings {
				_, _ = fmt.Fprintf(out, "warning: %s\n", w)
			}
			if rep.URL != "" {
				_, _ = fmt.Fprintf(out, "destination: %s\n", rep.URL)
			}
			if rep.Rendered != "" {
				_, _ = fmt.Fprintf(out, "rendered against %s:\n%s\n", source, strings.TrimRight(rep.Rendered, "\n"))
			}
			if !rep.Valid {
				return fmt.Errorf("recipe has %d error(s)", len(rep.Errors))
			}
			_, _ = fmt.Fprintln(out, "ok")
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&file, "file", "", "recipe file to check (instead of a name)")
	f.Var(params, "param", "param value name=value to substitute (repeatable)")
	f.StringVarP(&event, "event", "e", "", "event type whose registered sample payload the template is rendered against")
	f.StringVar(&payloadArg, "payload", "", "sample payload JSON to render against (overrides --event's sample)")
	addOutputFlag(cmd)
	return cmd
}
