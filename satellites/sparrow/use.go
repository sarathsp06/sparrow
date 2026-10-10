package main

import (
	"bufio"
	"context"
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

// runUse applies a recipe: registers a webhook and enables the recipe's
// transform template on the auto-created subscriptions.
func newUseCmd() *cobra.Command {
	var file string
	params := kvFlag{}
	var events listFlag
	labels := kvFlag{}
	cmd := &cobra.Command{
		Use:   "use <recipe>",
		Short: "Apply a recipe: register a webhook and enable its transform",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(events) == 0 {
				return fmt.Errorf("at least one --event is required")
			}
			client, cfg, err := clientFromCmd(cmd)
			if err != nil {
				return err
			}
			return runUse(cmd.Context(), cmd.OutOrStdout(), client, cfg.Consumer, args[0], file, events, params, labels)
		},
	}
	f := cmd.Flags()
	f.StringVar(&file, "file", "", "explicit recipe file path (overrides recipe search)")
	f.Var(params, "param", "recipe parameter name=value (repeatable; prompted if omitted)")
	f.VarP(&events, "event", "e", "event type to subscribe to (repeatable, required)")
	f.Var(labels, "label", "label filter key=value applied to the subscription (repeatable)")
	return cmd
}

func runUse(ctx context.Context, out io.Writer, client *apiClient, consumer, recipeArg, file string, events listFlag, params, labels kvFlag) error {
	r, err := resolveRecipe(recipeArg, file)
	if err != nil {
		return err
	}

	// Prompt for params not given as flags; resolveParams fills the rest.
	if isTerminal(os.Stdin) {
		stdin := bufio.NewScanner(os.Stdin)
		for _, p := range r.Params {
			if _, ok := params[p.Name]; ok {
				continue
			}
			promptText := p.Prompt
			if promptText == "" {
				promptText = p.Name
			}
			if len(p.Enum) > 0 {
				promptText += " (" + strings.Join(p.Enum, "|") + ")"
			}
			if p.Default != "" && !p.MustOverrideDefault {
				promptText += " [" + p.Default + "]"
			}
			if p.Help != "" {
				_, _ = fmt.Fprintf(out, "%s\n", p.Help)
			}
			_, _ = fmt.Fprintf(out, "%s: ", promptText)
			if stdin.Scan() {
				if v := strings.TrimSpace(stdin.Text()); v != "" {
					params[p.Name] = v
				}
			}
		}
	}
	values, err := resolveParams(r, params)
	if err != nil {
		return err
	}

	res, err := applyRecipe(ctx, client, consumer, r, values, events, labels)
	if res.WebhookID != "" {
		_, _ = fmt.Fprintf(out, "webhook %s -> %s\n", res.WebhookID, res.URL)
	}
	for _, sub := range res.Subscriptions {
		_, _ = fmt.Fprintf(out, "subscription %s (%s): transform %v, label filters %v\n", sub.SubscriptionID, sub.EventName, sub.Transform, map[string]string(labels))
	}
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "recipe %q applied in consumer %q\n", r.Name, consumer)
	return nil
}

// resolveParams turns the values a user gave into the full value set for r:
// defaults for the rest, then the required, must-override and enum rules.
// It never prompts, so the CLI (after prompting) and MCP share it.
func resolveParams(r recipe, given map[string]string) (map[string]string, error) {
	values := make(map[string]string, len(r.Params))
	for _, p := range r.Params {
		if v, ok := given[p.Name]; ok {
			values[p.Name] = v
			continue
		}
		switch {
		case p.Default != "" && !p.MustOverrideDefault:
			values[p.Name] = p.Default
		case p.Required:
			return nil, fmt.Errorf("param %q is required (pass --param %s=value)", p.Name, p.Name)
		default:
			values[p.Name] = ""
		}
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

type appliedSubscription struct {
	SubscriptionID string `json:"subscription_id"`
	EventName      string `json:"event_name"`
	Transform      bool   `json:"transform"`
}

type appliedRecipe struct {
	Recipe        string                `json:"recipe"`
	WebhookID     string                `json:"webhook_id"`
	URL           string                `json:"url"`
	Subscriptions []appliedSubscription `json:"subscriptions"`
}

// applyRecipe registers a webhook for r with resolved param values and
// enables the recipe's transform (and label filters) on the subscriptions
// created for events. On a partial failure the result still names the
// webhook that was registered.
func applyRecipe(ctx context.Context, client *apiClient, consumer string, r recipe, values map[string]string, events []string, labels map[string]string) (appliedRecipe, error) {
	res := appliedRecipe{Recipe: r.Name, Subscriptions: []appliedSubscription{}}
	rr, err := substituteRecipe(r, values)
	if err != nil {
		return res, err
	}
	hook, err := client.registerWebhook(ctx, consumer, webhookRequest{
		URL:           rr.URL,
		Events:        events,
		Active:        true,
		Description:   fmt.Sprintf("recipe %s: %s", r.Name, r.Description),
		Headers:       rr.Headers,
		SecretHeaders: rr.SecretHeaders,

		RequiresTransform: r.Webhook.RequiresTransform,
		TransformTemplate: rr.Template,
	})
	if err != nil {
		return res, fmt.Errorf("register webhook: %w", err)
	}
	res.WebhookID, res.URL = hook.WebhookID, rr.URL

	subs, err := client.listSubscriptions(ctx, consumer, hook.WebhookID)
	if err != nil {
		return res, fmt.Errorf("list subscriptions of webhook %s: %w", hook.WebhookID, err)
	}
	for _, sub := range subs {
		patch := subscriptionPatch{LabelFilters: labels}
		if rr.Template != "" {
			patch.TransformEnabled = true
			patch.TransformTemplate = rr.Template
		}
		if err := client.patchSubscription(ctx, consumer, sub.SubscriptionID, patch); err != nil {
			return res, fmt.Errorf("update subscription %s of webhook %s: %w", sub.SubscriptionID, hook.WebhookID, err)
		}
		res.Subscriptions = append(res.Subscriptions, appliedSubscription{SubscriptionID: sub.SubscriptionID, EventName: sub.EventName, Transform: rr.Template != ""})
	}
	return res, nil
}
