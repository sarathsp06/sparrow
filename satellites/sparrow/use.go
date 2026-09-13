package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// recipe is the on-disk recipe schema (version 1).
type recipe struct {
	Version     int           `yaml:"version"`
	Name        string        `yaml:"name"`
	Description string        `yaml:"description"`
	Params      []recipeParam `yaml:"params"`
	Webhook     struct {
		URL     string            `yaml:"url"`
		Headers map[string]string `yaml:"headers"`
	} `yaml:"webhook"`
	Subscription struct {
		TransformTemplate string `yaml:"transform_template"`
	} `yaml:"subscription"`
}

type recipeParam struct {
	Name     string `yaml:"name"`
	Prompt   string `yaml:"prompt"`
	Required bool   `yaml:"required"`
}

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

// findRecipe resolves the recipe file: explicit --file, ./recipes/<name>.yaml,
// then $SPARROW_RECIPES_DIR/<name>.yaml.
func findRecipe(name, file string) (string, error) {
	if file != "" {
		return file, nil
	}
	candidates := []string{filepath.Join("recipes", name+".yaml")}
	if dir := os.Getenv("SPARROW_RECIPES_DIR"); dir != "" {
		candidates = append(candidates, filepath.Join(dir, name+".yaml"))
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("recipe %q not found (tried %s; use --file or set SPARROW_RECIPES_DIR)", name, strings.Join(candidates, ", "))
}

func loadRecipe(path string) (recipe, error) {
	var r recipe
	data, err := os.ReadFile(path)
	if err != nil {
		return r, err
	}
	if err := yaml.Unmarshal(data, &r); err != nil {
		return r, fmt.Errorf("parse %s: %w", path, err)
	}
	if r.Version != 1 {
		return r, fmt.Errorf("%s: unsupported recipe version %d (want 1)", path, r.Version)
	}
	if r.Webhook.URL == "" {
		return r, fmt.Errorf("%s: recipe has no webhook.url", path)
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
			return runUse(cmd.Context(), cmd.OutOrStdout(), client, cfg.Namespace, args[0], file, events, params, labels)
		},
	}
	f := cmd.Flags()
	f.StringVar(&file, "file", "", "explicit recipe file path (overrides recipe search)")
	f.Var(params, "param", "recipe parameter name=value (repeatable; prompted if omitted)")
	f.VarP(&events, "event", "e", "event type to subscribe to (repeatable, required)")
	f.Var(labels, "label", "label filter key=value applied to the subscription (repeatable)")
	return cmd
}

func runUse(ctx context.Context, out io.Writer, client *apiClient, namespace, recipeArg, file string, events listFlag, params, labels kvFlag) error {
	path, err := findRecipe(recipeArg, file)
	if err != nil {
		return err
	}
	r, err := loadRecipe(path)
	if err != nil {
		return err
	}

	// Collect params: flags first, prompt for the rest.
	stdin := bufio.NewScanner(os.Stdin)
	for _, p := range r.Params {
		if _, ok := params[p.Name]; ok {
			continue
		}
		promptText := p.Prompt
		if promptText == "" {
			promptText = p.Name
		}
		if isTerminal(os.Stdin) {
			_, _ = fmt.Fprintf(out, "%s: ", promptText)
			if stdin.Scan() {
				if v := strings.TrimSpace(stdin.Text()); v != "" {
					params[p.Name] = v
					continue
				}
			}
		}
		if p.Required {
			return fmt.Errorf("param %q is required (pass --param %s=value)", p.Name, p.Name)
		}
		params[p.Name] = ""
	}

	// Substitute {{param "x"}} in url, header values, and the template.
	hookURL, err := substituteParams(r.Webhook.URL, params)
	if err != nil {
		return err
	}
	headers := make(map[string]string, len(r.Webhook.Headers))
	for k, v := range r.Webhook.Headers {
		if headers[k], err = substituteParams(v, params); err != nil {
			return err
		}
	}
	tmpl, err := substituteParams(r.Subscription.TransformTemplate, params)
	if err != nil {
		return err
	}

	hook, err := client.registerWebhook(ctx, namespace, webhookRequest{
		URL:         hookURL,
		Events:      events,
		Active:      true,
		Description: fmt.Sprintf("recipe %s: %s", r.Name, r.Description),
		Headers:     headers,
	})
	if err != nil {
		return fmt.Errorf("register webhook: %w", err)
	}
	_, _ = fmt.Fprintf(out, "webhook %s -> %s\n", hook.WebhookID, hookURL)

	subs, err := client.listSubscriptions(ctx, namespace, hook.WebhookID)
	if err != nil {
		return fmt.Errorf("list subscriptions: %w", err)
	}
	for _, sub := range subs {
		patch := subscriptionPatch{LabelFilters: labels}
		if tmpl != "" {
			patch.TransformEnabled = true
			patch.TransformTemplate = tmpl
		}
		if err := client.patchSubscription(ctx, namespace, sub.SubscriptionID, patch); err != nil {
			return fmt.Errorf("update subscription %s: %w", sub.SubscriptionID, err)
		}
		_, _ = fmt.Fprintf(out, "subscription %s (%s): transform %v, label filters %v\n", sub.SubscriptionID, sub.EventName, tmpl != "", map[string]string(labels))
	}
	_, _ = fmt.Fprintf(out, "recipe %q applied in namespace %q\n", r.Name, namespace)
	return nil
}
