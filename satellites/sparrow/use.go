package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/sarathsp06/sparrow/satellites/recipes"
)

// recipe is the shared recipe schema (version 1), defined in the recipes module
// alongside the built-in catalog embedded in this binary.
type recipe = recipes.Recipe

var paramToken = regexp.MustCompile(`\{\{\s*param\s+"([^"]+)"\s*\}\}`)

// substituteParams replaces {{param "name"}} tokens with values. Unknown
// tokens are an error so a typo can't silently ship a literal template token.
func substituteParams(s string, values map[string]string) (string, error) {
	var missing []string
	result := paramToken.ReplaceAllStringFunc(s, func(tok string) string {
		name := paramToken.FindStringSubmatch(tok)[1]
		v, ok := values[name]
		if !ok {
			missing = append(missing, name)
			return tok
		}
		return v
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("missing values for params: %s", strings.Join(missing, ", "))
	}
	return result, nil
}

// resolveRecipe resolves the recipe: explicit --file, ./recipes/<name>.yaml,
// $SPARROW_RECIPES_DIR/<name>.yaml (local files can override built-ins), then
// the catalog embedded in the binary.
func resolveRecipe(name, file string) (recipe, error) {
	if file != "" {
		return loadRecipe(file)
	}
	path, candidates := localRecipePath(name)
	if path != "" {
		return loadRecipe(path)
	}
	builtin, err := recipes.All()
	if err != nil {
		return recipe{}, err
	}
	for _, r := range builtin {
		if r.Name == name {
			return r, nil
		}
	}
	return recipe{}, fmt.Errorf("recipe %q not found (no built-in recipe by that name — run 'sparrow recipes' to list them — and no %s; use --file or set SPARROW_RECIPES_DIR)", name, strings.Join(candidates, ", "))
}

// localRecipePath returns the first local file that overrides a built-in
// recipe of that name ("" when none exists) and every path it looked at.
func localRecipePath(name string) (string, []string) {
	candidates := []string{filepath.Join("recipes", name+".yaml")}
	if dir := os.Getenv("SPARROW_RECIPES_DIR"); dir != "" {
		candidates = append(candidates, filepath.Join(dir, name+".yaml"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, candidates
		}
	}
	return "", candidates
}

func loadRecipe(path string) (recipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return recipe{}, err
	}
	return parseRecipe(data, path)
}

// parseRecipe decodes recipe YAML and checks the invariants every recipe must
// hold to be applied. src names the input in errors (a path, or "inline").
func parseRecipe(data []byte, src string) (recipe, error) {
	var r recipe
	if err := yaml.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("parse %s: %w", src, err)
	}
	if r.Version != 1 {
		return r, fmt.Errorf("%s: unsupported recipe version %d (want 1)", src, r.Version)
	}
	if r.Webhook.URL == "" {
		return r, fmt.Errorf("%s: recipe has no webhook.url", src)
	}
	if err := r.Validate(); err != nil {
		return r, fmt.Errorf("%s: %w", src, err)
	}
	return r, nil
}

const useHelp = `Apply a recipe: register a webhook for the recipe's destination, subscribed
to the given events, with the recipe's transform template enabled.

The recipe is validated first, rendering its template against the event
type's registered sample payload (or a built-in sample when it has none);
nothing is created if that fails. Params not given with --param are prompted
for on a terminal; elsewhere a missing required param is an error.

Each consumer has one webhook per URL, so applying a recipe again to the same
destination is refused unless --update re-applies it: the subscriptions for
--event get the recipe's transform and labels (missing ones are created), and
the webhook gets the recipe's headers. --dry-run shows what would happen.`

const useExample = `  sparrow use slack -e order.created --param webhook_url=https://hooks.slack.com/services/T/B/X
  sparrow use pagerduty -e payment.failed --param routing_key=R0UT1NG --dry-run
  sparrow use slack -e order.created -e order.refunded --update --param webhook_url=...
  sparrow use --file ./recipes/teams.yaml -e order.created -o json`

// newUseCmd applies a recipe: registers a webhook and enables the recipe's
// transform template on the auto-created subscriptions.
func newUseCmd() *cobra.Command {
	var file string
	var dryRun, update bool
	params := kvFlag{}
	var events listFlag
	labels := kvFlag{}
	cmd := &cobra.Command{
		Use:     "use <recipe>",
		Short:   "Apply a recipe: register a webhook and enable its transform",
		Long:    useHelp,
		Example: useExample,
		Args: func(cmd *cobra.Command, args []string) error {
			if file != "" {
				return cobra.MaximumNArgs(1)(cmd, args)
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(events) == 0 {
				return fmt.Errorf("at least one --event is required")
			}
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			name := ""
			if len(args) > 0 {
				name = args[0]
			}
			req := applyRequest{Events: events, Labels: labels, DryRun: dryRun, Update: update}
			return runUse(cmd.Context(), cmd.OutOrStdout(), outputFmt(cmd), client, cfg.Consumer, name, file, params, req)
		},
	}
	f := cmd.Flags()
	f.StringVar(&file, "file", "", "recipe file path (instead of a built-in or ./recipes/<name>.yaml)")
	f.Var(params, "param", "recipe parameter name=value (repeatable; prompted for on a terminal if omitted)")
	f.VarP(&events, "event", "e", "event type to subscribe to (repeatable, at least one)")
	f.Var(labels, "label", "label filter key=value applied to the subscriptions (repeatable)")
	f.BoolVar(&dryRun, "dry-run", false, "validate and show what would be created or changed, without changing anything")
	f.BoolVar(&update, "update", false, "re-apply to the existing webhook for the same destination instead of refusing")
	addOutputFlag(cmd)
	return cmd
}

func runUse(ctx context.Context, out io.Writer, format string, client *apiClient, consumer, recipeArg, file string, params kvFlag, req applyRequest) error {
	r, err := resolveRecipe(recipeArg, file)
	if err != nil {
		return err
	}
	if isTerminal(os.Stdin) && format == "" {
		promptParams(out, r, params)
	}
	values, err := resolveParams(r, params)
	var missing *missingParamsError
	if errors.As(err, &missing) {
		return fmt.Errorf("%w; pass %s", err, missing.flags())
	}
	if err != nil {
		return err
	}

	res, err := applyRecipeTo(ctx, client, consumer, r, values, req)
	if done, ferr := renderStructured(out, format, res); done {
		if ferr != nil {
			return ferr
		}
		return useError(err, r, recipeArg, file)
	}
	printApplied(out, res, err)
	return useError(err, r, recipeArg, file)
}

// promptParams asks for every param not given as a flag; resolveParams fills
// in the rest.
func promptParams(out io.Writer, r recipe, params kvFlag) {
	stdin := bufio.NewScanner(os.Stdin)
	for _, p := range r.Params {
		if _, ok := params[p.Name]; ok {
			continue
		}
		label := p.Prompt
		if label == "" {
			label = p.Name
		}
		if len(p.Enum) > 0 {
			label += " (" + strings.Join(p.Enum, "|") + ")"
		}
		if p.Default != "" && !p.MustOverrideDefault {
			label += " [" + p.Default + "]"
		}
		if p.Help != "" {
			_, _ = fmt.Fprintf(out, "\n  %s\n", p.Help)
		}
		if p.Example != "" {
			_, _ = fmt.Fprintf(out, "  e.g. %s\n", p.Example)
		}
		_, _ = fmt.Fprintf(out, "%s: ", label)
		if stdin.Scan() {
			if v := strings.TrimSpace(stdin.Text()); v != "" {
				params[p.Name] = v
			}
		}
	}
}

// useError adds the CLI's next step to an apply error.
func useError(err error, r recipe, recipeArg, file string) error {
	var exists *webhookExistsError
	if errors.As(err, &exists) {
		target := recipeArg
		if file != "" {
			target = "--file " + file
		}
		return fmt.Errorf("%w; re-run 'sparrow use %s ... --update' to re-apply the recipe to it, or delete it first", err, target)
	}
	return err
}

// printApplied writes what applying did (or would do) for people.
func printApplied(out io.Writer, res appliedRecipe, err error) {
	for _, w := range res.Warnings {
		_, _ = fmt.Fprintf(out, "warning: %s\n", w)
	}
	verb := map[string]string{"created": "created", "updated": "updated", "would_create": "would create", "would_update": "would update"}
	if res.Action != "" {
		hook := "webhook"
		if res.WebhookID != "" {
			hook += " " + res.WebhookID
		}
		_, _ = fmt.Fprintf(out, "%s %s -> %s\n", verb[res.Action], hook, res.URL)
	}
	for _, sub := range res.Subscriptions {
		line := fmt.Sprintf("  %s subscription", verb[sub.Action])
		if sub.SubscriptionID != "" {
			line += " " + sub.SubscriptionID
		}
		line += " for " + sub.EventName
		if sub.Transform {
			line += ", transform on"
		}
		if len(sub.LabelFilters) > 0 {
			line += ", labels " + kvFlag(sub.LabelFilters).String()
		}
		_, _ = fmt.Fprintln(out, line)
	}
	if res.DryRun && err == nil {
		_, _ = fmt.Fprintf(out, "\nsample delivery, rendered against %s:\n%s\n", res.SampleSource, strings.TrimRight(res.Rendered, "\n"))
		_, _ = fmt.Fprintln(out, "dry run: nothing was changed")
		return
	}
	if err == nil {
		_, _ = fmt.Fprintf(out, "recipe %q applied in consumer %q\n", res.Recipe, res.Consumer)
	}
}

// resolveParams turns the values a user gave into the full value set for r:
// defaults for the rest, then the required, must-override and enum rules.
// It never prompts, so the CLI (after prompting) and MCP share it.
func resolveParams(r recipe, given map[string]string) (map[string]string, error) {
	values := make(map[string]string, len(r.Params))
	var missing []string
	for _, p := range r.Params {
		if v, ok := given[p.Name]; ok {
			values[p.Name] = v
			continue
		}
		switch {
		case p.Default != "" && !p.MustOverrideDefault:
			values[p.Name] = p.Default
		case p.Required:
			missing = append(missing, p.Name)
		default:
			values[p.Name] = ""
		}
	}
	if len(missing) > 0 {
		return nil, &missingParamsError{names: missing}
	}
	for _, p := range r.Params {
		if p.MustOverrideDefault && p.Default != "" && strings.TrimSpace(values[p.Name]) == p.Default {
			return nil, fmt.Errorf("param %q must be changed from its default %q", p.Name, p.Default)
		}
		if v := values[p.Name]; len(p.Enum) > 0 && v != "" && !slices.Contains(p.Enum, v) {
			return nil, fmt.Errorf("param %q must be one of %s, not %q", p.Name, strings.Join(p.Enum, ", "), v)
		}
	}
	return values, nil
}

// missingParamsError names every required param without a value, so a caller
// can ask for all of them at once.
type missingParamsError struct{ names []string }

func (e *missingParamsError) Error() string {
	return "missing required params: " + strings.Join(e.names, ", ")
}

// flags is the --param form of the missing params, for CLI messages.
func (e *missingParamsError) flags() string {
	parts := make([]string, len(e.names))
	for i, n := range e.names {
		parts[i] = "--param " + n + "=..."
	}
	return strings.Join(parts, " ")
}

// renderedRecipe is r with every {{param "x"}} token replaced.
type renderedRecipe struct {
	URL           string
	Headers       map[string]string
	SecretHeaders map[string]string
	Template      string
}

func substituteRecipe(r recipe, values map[string]string) (renderedRecipe, error) {
	var out renderedRecipe
	var err error
	if out.URL, err = substituteParams(r.Webhook.URL, values); err != nil {
		return out, err
	}
	out.Headers = make(map[string]string, len(r.Webhook.Headers))
	for k, v := range r.Webhook.Headers {
		if out.Headers[k], err = substituteParams(v, values); err != nil {
			return out, err
		}
	}
	out.SecretHeaders = make(map[string]string, len(r.Webhook.SecretHeaders))
	for k, v := range r.Webhook.SecretHeaders {
		if out.SecretHeaders[k], err = substituteParams(v, values); err != nil {
			return out, err
		}
	}
	out.Template, err = substituteParams(r.Subscription.TransformTemplate, values)
	return out, err
}
