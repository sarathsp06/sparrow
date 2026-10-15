package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sarathsp06/sparrow/satellites/recipes"
)

type recipeNameIn struct {
	Name string `json:"name" jsonschema:"recipe name, e.g. slack; a local ./recipes/<name>.yaml or $SPARROW_RECIPES_DIR/<name>.yaml overrides the built-in"`
}

type validateRecipeIn struct {
	Name      string            `json:"name,omitempty" jsonschema:"recipe to validate by name (built-in or local file); give name or yaml"`
	YAML      string            `json:"yaml,omitempty" jsonschema:"full recipe YAML (schema version 1) to validate, e.g. a draft; give name or yaml"`
	Params    map[string]string `json:"params,omitempty" jsonschema:"param values to substitute; missing ones use the default or a placeholder"`
	EventName string            `json:"event_name,omitempty" jsonschema:"event name for the sample render, default example.event"`
	Payload   map[string]any    `json:"payload,omitempty" jsonschema:"sample event payload for the render, default a small built-in sample"`
}

type applyRecipeIn struct {
	Name           string            `json:"name,omitempty" jsonschema:"recipe to apply by name; give name or yaml"`
	YAML           string            `json:"yaml,omitempty" jsonschema:"full recipe YAML to apply, e.g. one just checked with validate_recipe; give name or yaml"`
	Params         map[string]string `json:"params,omitempty" jsonschema:"param values; required params without a default must be given"`
	Events         []string          `json:"events" jsonschema:"event types the webhook subscribes to (at least one)"`
	LabelFilters   map[string]string `json:"label_filters,omitempty" jsonschema:"label key/values an event must carry to be delivered"`
	DryRun         bool              `json:"dry_run,omitempty" jsonschema:"validate and report what would be created or changed, without changing anything"`
	UpdateExisting bool              `json:"update_existing,omitempty" jsonschema:"when a webhook for the recipe's destination already exists, re-apply the recipe to it (transform, labels, headers; missing subscriptions are created) instead of failing"`
}

// recipeSummary is a catalog row: enough to pick a recipe and fill its params.
type recipeSummary struct {
	Name string `json:"name"`
	// Source is "built-in" or the local file that defines (or overrides) it.
	Source            string                `json:"source"`
	Description       string                `json:"description"`
	RequiresTransform bool                  `json:"requires_transform"`
	Consumer          *recipes.ConsumerHint `json:"consumer,omitempty"`
	Params            []recipes.Param       `json:"params"`
}

func addRecipeTools(srv *mcp.Server, client *apiClient, consumer string, allowWrite bool) {
	ro := &mcp.ToolAnnotations{ReadOnlyHint: true}

	mcp.AddTool(srv, &mcp.Tool{Name: "list_recipes", Annotations: ro,
		Description: "List the recipes (destination adapters such as Slack, PagerDuty, Twilio): the built-in catalog plus local ./recipes/*.yaml and $SPARROW_RECIPES_DIR files, which override built-ins of the same name. Each comes with its params; use get_recipe for the full template."},
		func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
			all, err := catalog()
			if err != nil {
				return nil, nil, err
			}
			out := make([]recipeSummary, 0, len(all))
			for _, e := range all {
				r := e.recipe
				params := r.Params
				if params == nil {
					params = []recipes.Param{}
				}
				out = append(out, recipeSummary{Name: r.Name, Source: e.source, Description: r.Description, RequiresTransform: r.Webhook.RequiresTransform, Consumer: r.Consumer, Params: params})
			}
			return nil, map[string]any{"items": out}, nil
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "get_recipe", Annotations: ro,
		Description: "Get one recipe in full: webhook url and headers with {{param}} tokens, transform template, params and guidance. A good starting point for writing a new recipe."},
		func(_ context.Context, _ *mcp.CallToolRequest, in recipeNameIn) (*mcp.CallToolResult, any, error) {
			r, err := resolveRecipe(in.Name, "")
			return nil, r, err
		})

	mcp.AddTool(srv, &mcp.Tool{Name: "validate_recipe", Annotations: ro,
		Description: "Validate a recipe without applying it: schema and param rules, the built-in catalog's shipping rules (as warnings), and a strict local render of the transform template with params substituted. The render uses payload when given, else the registered sample payload of event_name, else a built-in sample. Returns errors, warnings and the rendered body. Nothing is created."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in validateRecipeIn) (*mcp.CallToolResult, any, error) {
			if (in.Name == "") == (in.YAML == "") {
				return nil, nil, errors.New("give exactly one of name or yaml")
			}
			if in.Payload == nil && in.EventName != "" {
				in.Payload, _ = eventSample(ctx, client, in.EventName)
			}
			if in.YAML != "" {
				return nil, validateRecipeYAML([]byte(in.YAML), in.Params, in.EventName, in.Payload), nil
			}
			if path, _ := localRecipePath(in.Name); path != "" {
				data, err := os.ReadFile(path)
				if err != nil {
					return nil, nil, err
				}
				return nil, validateRecipeYAML(data, in.Params, in.EventName, in.Payload), nil
			}
			r, err := resolveRecipe(in.Name, "")
			if err != nil {
				return nil, nil, err
			}
			return nil, validateRecipe(r, in.Params, in.EventName, in.Payload), nil
		})

	if !allowWrite {
		return
	}

	mcp.AddTool(srv, &mcp.Tool{Name: "apply_recipe",
		Description: "Apply a recipe (like `sparrow use`): register a webhook to the destination, subscribed to events, with the recipe's transform enabled. The recipe is validated first against the first event type's registered sample payload, and nothing is created if validation fails. A consumer has one webhook per URL: if one already exists for the destination this fails and names it, unless update_existing is set. Use dry_run to preview."},
		func(ctx context.Context, _ *mcp.CallToolRequest, in applyRecipeIn) (*mcp.CallToolResult, any, error) {
			if (in.Name == "") == (in.YAML == "") {
				return nil, nil, errors.New("give exactly one of name or yaml")
			}
			if len(in.Events) == 0 {
				return nil, nil, errors.New("events needs at least one event type")
			}
			var r recipe
			var err error
			if in.YAML != "" {
				r, err = parseRecipe([]byte(in.YAML), "inline recipe")
			} else {
				r, err = resolveRecipe(in.Name, "")
			}
			if err != nil {
				return nil, nil, err
			}
			values, err := resolveParams(r, in.Params)
			var missing *missingParamsError
			if errors.As(err, &missing) {
				return nil, nil, fmt.Errorf("%w; set them in params (get_recipe shows what each one is)", err)
			}
			if err != nil {
				return nil, nil, err
			}
			res, err := applyRecipeTo(ctx, client, consumer, r, values, applyRequest{
				Events: in.Events, Labels: in.LabelFilters, DryRun: in.DryRun, Update: in.UpdateExisting,
			})
			var exists *webhookExistsError
			switch {
			case errors.As(err, &exists):
				return nil, nil, fmt.Errorf("%w; set update_existing to re-apply the recipe to it, or delete_webhook it first", err)
			case err != nil && res.WebhookID != "":
				return nil, nil, fmt.Errorf("%w (webhook %s was created; delete_webhook it or retry with update_existing)", err, res.WebhookID)
			case err != nil:
				return nil, nil, err
			}
			return nil, res, nil
		})
}
