package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sarathsp06/sparrow/satellites/recipes"
)

// The validator and the catalog tests must agree: every shipped recipe is
// valid with no warnings.
func TestValidateRecipe_BuiltinsAreCleanAndValid(t *testing.T) {
	all, err := recipes.All()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range all {
		rep := validateRecipe(r, nil, "", nil)
		if !rep.Valid || len(rep.Errors) > 0 || len(rep.Warnings) > 0 {
			t.Errorf("%s: valid=%v errors=%v warnings=%v", r.Name, rep.Valid, rep.Errors, rep.Warnings)
		}
		if rep.Rendered == "" {
			t.Errorf("%s: no rendered body", r.Name)
		}
	}
}

const draftRecipe = `version: 1
name: acme
description: Post to Acme's events API
guidance: One line per event.
params:
  - name: api_token
    prompt: API token
    help: Settings > API in Acme.
    required: true
    secret: true
  - name: channel
    prompt: Channel
    help: Where events land.
    default: general
webhook:
  requires_transform: true
  url: https://api.acme.test/v1/channels/{{param "channel"}}
  headers: {Content-Type: application/json}
  secret_headers:
    Authorization: 'Bearer {{param "api_token"}}'
subscription:
  transform_template: '{"text": {{printf "%s for %v" .event_name .payload.id | json}}, "token_echo": {{"{{param \"api_token\"}}" | json}}}'
`

func TestValidateRecipeYAML_Draft(t *testing.T) {
	// The template echoes the secret token, so the report must mask it.
	draft := strings.Replace(draftRecipe, `{{"{{param \"api_token\"}}" | json}}`, `"{{param "api_token"}}"`, 1)
	rep := validateRecipeYAML([]byte(draft), map[string]string{"api_token": "s3cret-tok"}, "order.created", map[string]any{"id": "o_1"})
	if !rep.Valid || len(rep.Errors)+len(rep.Warnings) != 0 {
		t.Fatalf("draft: %+v", rep)
	}
	if rep.URL != "https://api.acme.test/v1/channels/general" {
		t.Errorf("url = %q, want the default channel substituted", rep.URL)
	}
	var body map[string]string
	if err := json.Unmarshal([]byte(rep.Rendered), &body); err != nil || body["text"] != "order.created for o_1" {
		t.Errorf("rendered = %s (%v)", rep.Rendered, err)
	}
	if strings.Contains(rep.Rendered, "s3cret-tok") || body["token_echo"] != "****" {
		t.Errorf("secret param leaked into the report: %s", rep.Rendered)
	}
}

func TestValidateRecipeYAML_Problems(t *testing.T) {
	cases := []struct {
		name, find, replace string
		wantErr, wantWarn   string
	}{
		{"typo key is ignored on apply", "requires_transform: true", "requires_tranform: true", "", "requires_tranform"},
		{"undeclared param", `{{param "channel"}}`, `{{param "chanel"}}`, "undeclared param", ""},
		{"non-JSON output with JSON content type", `{"text": {{printf`, `text={{printf`, "not valid JSON", ""},
		{"missing payload key", ".payload.id", ".payload.missing", "does not render", ""},
		{"url not http after substitution", "https://api.acme.test/v1/channels/", "ftp://acme.test/", "not an http(s) URL", ""},
		{"no guidance", "guidance: One line per event.\n", "", "", "guidance is empty"},
		{"bad yaml", "version: 1", "version: [", "parse", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			draft := strings.Replace(draftRecipe, tc.find, tc.replace, 1)
			rep := validateRecipeYAML([]byte(draft), nil, "", map[string]any{"id": "o_1"})
			if tc.wantErr != "" && (rep.Valid || !strings.Contains(strings.Join(rep.Errors, "\n"), tc.wantErr)) {
				t.Errorf("want error containing %q, got valid=%v errors=%v", tc.wantErr, rep.Valid, rep.Errors)
			}
			if tc.wantWarn != "" && !strings.Contains(strings.Join(rep.Warnings, "\n"), tc.wantWarn) {
				t.Errorf("want warning containing %q, got %v", tc.wantWarn, rep.Warnings)
			}
		})
	}
}

func TestValidateRecipe_WholeURLParamCheckedOnceGiven(t *testing.T) {
	all, err := recipes.All()
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range all {
		if r.Name != "slack" {
			continue
		}
		if rep := validateRecipe(r, map[string]string{"webhook_url": "not a url"}, "", nil); rep.Valid {
			t.Fatalf("slack with webhook_url=%q validated: %+v", "not a url", rep)
		}
		if rep := validateRecipe(r, map[string]string{"webhook_url": "https://hooks.slack.test/x"}, "", nil); !rep.Valid {
			t.Fatalf("slack with a real url: %+v", rep.Errors)
		}
		return
	}
	t.Fatal("slack recipe not found")
}

func TestResolveParams(t *testing.T) {
	r, err := parseRecipe([]byte(draftRecipe), "draft")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolveParams(r, nil); err == nil || !strings.Contains(err.Error(), "api_token") {
		t.Errorf("missing required param: err = %v", err)
	}
	v, err := resolveParams(r, map[string]string{"api_token": "t"})
	if err != nil || v["channel"] != "general" {
		t.Errorf("defaults: %v %v", v, err)
	}
}

func TestMCPRecipeTools(t *testing.T) {
	api := newFakeSparrow(t)
	srv, _ := newMCPServer(newAPIClient(config{ServerURL: api.URL}), "acme", true)
	cs := connectMCP(t, srv)

	out := call(t, cs, "list_recipes", map[string]any{})
	if !strings.Contains(out, `"name":"slack"`) || !strings.Contains(out, `"webhook_url"`) {
		t.Fatalf("list_recipes = %.300s", out)
	}
	if out = call(t, cs, "get_recipe", map[string]any{"name": "slack"}); !strings.Contains(out, "transform_template") {
		t.Fatalf("get_recipe = %.300s", out)
	}
	if out = call(t, cs, "validate_recipe", map[string]any{"name": "slack"}); !strings.Contains(out, `"valid":true`) {
		t.Fatalf("validate_recipe slack = %s", out)
	}
	if out = call(t, cs, "validate_recipe", map[string]any{"yaml": strings.Replace(draftRecipe, `"channel"}}`, `"chanel"}}`, 1)}); !strings.Contains(out, `"valid":false`) {
		t.Fatalf("validate_recipe draft = %s", out)
	}

	// Apply a built-in: webhook registered with the transform, subscription patched.
	out = call(t, cs, "apply_recipe", map[string]any{
		"name": "slack", "params": map[string]string{"webhook_url": "https://hooks.slack.test/T/B/X"},
		"events": []string{"order.created"}, "label_filters": map[string]string{"env": "prod"},
	})
	if !strings.Contains(out, `"webhook_id":"wh_1"`) || !strings.Contains(out, `"subscription_id":"sub_1"`) {
		t.Fatalf("apply_recipe = %s", out)
	}
	reg := api.body(http.MethodPost, "/v1/consumers/acme/webhooks")
	if !strings.Contains(reg, `"url":"https://hooks.slack.test/T/B/X"`) || !strings.Contains(reg, `"requires_transform":true`) || !strings.Contains(reg, `"transform_template"`) {
		t.Fatalf("register body = %s", reg)
	}
	if patch := api.body(http.MethodPatch, "/v1/consumers/acme/subscriptions/sub_1"); !strings.Contains(patch, `"env":"prod"`) || !strings.Contains(patch, `"transform_enabled":true`) {
		t.Fatalf("patch body = %s", patch)
	}
}

func TestMCPApplyRecipeRefusesInvalid(t *testing.T) {
	api := newFakeSparrow(t)
	srv, _ := newMCPServer(newAPIClient(config{ServerURL: api.URL}), "acme", true)
	cs := connectMCP(t, srv)

	for name, args := range map[string]map[string]any{
		"render fails":  {"yaml": strings.Replace(draftRecipe, ".payload.id", ".payload.missing", 1), "params": map[string]string{"api_token": "t"}, "events": []string{"x"}},
		"missing param": {"name": "slack", "events": []string{"x"}},
		"no events":     {"name": "slack", "params": map[string]string{"webhook_url": "https://h.test"}},
		"name and yaml": {"name": "slack", "yaml": draftRecipe, "events": []string{"x"}},
	} {
		res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "apply_recipe", Arguments: args})
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !res.IsError {
			t.Errorf("%s: want a tool error", name)
		}
	}
	if api.sawPath(http.MethodPost, "/webhooks") {
		t.Fatal("an invalid recipe must not register a webhook")
	}
}
