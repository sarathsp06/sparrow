package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

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
func runUse(ctx context.Context, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("use", flag.ContinueOnError)
	urlFlag, apiKeyFlag, nsFlag := configFlags(fs)
	file := fs.String("file", "", "explicit recipe file path (overrides recipe search)")
	params := kvFlag{}
	fs.Var(params, "param", "recipe parameter name=value (repeatable; prompted if omitted)")
	var events listFlag
	fs.Var(&events, "event", "event type to subscribe to (repeatable, required)")
	labels := kvFlag{}
	fs.Var(labels, "label", "label filter key=value applied to the subscription (repeatable)")
	recipe, err := parseWithArg(fs, args, "sparrow use <recipe> --event <name> [--param k=v] [--label k=v] [--file path]")
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return fmt.Errorf("at least one --event is required")
	}

	path, err := findRecipe(recipe, *file)
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
			fmt.Fprintf(out, "%s: ", promptText)
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

	cfg, err := resolveConfig(*urlFlag, *apiKeyFlag, *nsFlag)
	if err != nil {
		return err
	}
	client := newAPIClient(cfg)

	hook, err := client.registerWebhook(ctx, cfg.Namespace, webhookRequest{
		URL:         hookURL,
		Events:      events,
		Active:      true,
		Description: fmt.Sprintf("recipe %s: %s", r.Name, r.Description),
		Headers:     headers,
	})
	if err != nil {
		return fmt.Errorf("register webhook: %w", err)
	}
	fmt.Fprintf(out, "webhook %s -> %s\n", hook.WebhookID, hookURL)

	subs, err := client.listSubscriptions(ctx, cfg.Namespace, hook.WebhookID)
	if err != nil {
		return fmt.Errorf("list subscriptions: %w", err)
	}
	for _, sub := range subs {
		patch := subscriptionPatch{LabelFilters: labels}
		if tmpl != "" {
			patch.TransformEnabled = true
			patch.TransformTemplate = tmpl
		}
		if err := client.patchSubscription(ctx, cfg.Namespace, sub.SubscriptionID, patch); err != nil {
			return fmt.Errorf("update subscription %s: %w", sub.SubscriptionID, err)
		}
		fmt.Fprintf(out, "subscription %s (%s): transform %v, label filters %v\n", sub.SubscriptionID, sub.EventName, tmpl != "", map[string]string(labels))
	}
	fmt.Fprintf(out, "recipe %q applied in namespace %q\n", r.Name, cfg.Namespace)
	return nil
}
