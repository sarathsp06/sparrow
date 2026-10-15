package main

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// existingHookAPI answers like a server where the consumer already has a
// webhook for slackURL, subscribed to order.created.
type existingHookAPI struct {
	*httptest.Server
	mu     sync.Mutex
	writes []string
	bodies map[string]string
}

const slackURL = "https://hooks.slack.test/T/B/X"

func newExistingHookAPI(t *testing.T) *existingHookAPI {
	t.Helper()
	a := &existingHookAPI{bodies: map[string]string{}}
	a.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		a.mu.Lock()
		if r.Method != http.MethodGet {
			a.writes = append(a.writes, r.Method+" "+r.URL.Path)
			a.bodies[r.Method+" "+r.URL.Path] = string(body)
		}
		a.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/webhooks"):
			_, _ = w.Write([]byte(`{"items":[{"webhook_id":"wh_old","url":"` + slackURL + `","description":"recipe slack: old"}]}`))
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/subscriptions"):
			_, _ = w.Write([]byte(`{"items":[{"subscription_id":"sub_old","event_name":"order.created"}]}`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/subscriptions"):
			_, _ = w.Write([]byte(`{"subscription_id":"sub_new"}`))
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/event-types/"):
			_, _ = w.Write([]byte(`{"name":"order.created","sample_payload":{"id":"ord_7"}}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(a.Close)
	return a
}

func (a *existingHookAPI) wrote() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]string(nil), a.writes...)
}

func slackValues(t *testing.T) (recipe, map[string]string) {
	t.Helper()
	r, err := resolveRecipe("slack", "")
	if err != nil {
		t.Fatal(err)
	}
	return r, map[string]string{"webhook_url": slackURL}
}

// Applying a recipe whose destination already has a webhook is refused,
// naming that webhook, and writes nothing.
func TestApplyRecipeRefusesAnExistingDestination(t *testing.T) {
	api := newExistingHookAPI(t)
	r, values := slackValues(t)
	_, err := applyRecipeTo(context.Background(), newAPIClient(config{ServerURL: api.URL}), "acme", r, values, applyRequest{Events: []string{"order.created"}})
	var exists *webhookExistsError
	if !errors.As(err, &exists) || exists.WebhookID != "wh_old" {
		t.Fatalf("err = %v, want webhookExistsError for wh_old", err)
	}
	if w := api.wrote(); len(w) != 0 {
		t.Fatalf("refused apply wrote %v", w)
	}
}

// --update re-applies: the existing subscription is patched, a missing one
// is created with the template, then the webhook gets the recipe's headers.
func TestApplyRecipeUpdateReappliesToTheExistingWebhook(t *testing.T) {
	api := newExistingHookAPI(t)
	r, values := slackValues(t)
	res, err := applyRecipeTo(context.Background(), newAPIClient(config{ServerURL: api.URL}), "acme", r, values,
		applyRequest{Events: []string{"order.created", "order.refunded"}, Labels: map[string]string{"env": "prod"}, Update: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Action != "updated" || res.WebhookID != "wh_old" || len(res.Subscriptions) != 2 ||
		res.Subscriptions[0].Action != "updated" || res.Subscriptions[1].Action != "created" || res.Subscriptions[1].SubscriptionID != "sub_new" {
		t.Fatalf("result = %+v", res)
	}
	want := []string{
		"PATCH /v1/consumers/acme/subscriptions/sub_old",
		"POST /v1/consumers/acme/subscriptions",
		"PATCH /v1/consumers/acme/webhooks/wh_old",
	}
	if got := api.wrote(); strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("writes = %v, want %v", got, want)
	}
	if b := api.bodies["POST /v1/consumers/acme/subscriptions"]; !strings.Contains(b, `"transform_template"`) || !strings.Contains(b, `"env":"prod"`) {
		t.Fatalf("new subscription body = %s", b)
	}
	if !strings.Contains(res.SampleSource, "registered sample payload of order.created") {
		t.Fatalf("sample source = %q, want the event type's sample", res.SampleSource)
	}
}

// A dry run reports the plan and writes nothing, for create and update alike.
func TestApplyRecipeDryRunWritesNothing(t *testing.T) {
	api := newExistingHookAPI(t)
	r, values := slackValues(t)
	client := newAPIClient(config{ServerURL: api.URL})

	res, err := applyRecipeTo(context.Background(), client, "acme", r, values, applyRequest{Events: []string{"order.created", "order.refunded"}, Update: true, DryRun: true})
	if err != nil || res.Action != "would_update" || res.Subscriptions[1].Action != "would_create" {
		t.Fatalf("update dry run = %+v, %v", res, err)
	}
	values["webhook_url"] = "https://hooks.slack.test/other"
	res, err = applyRecipeTo(context.Background(), client, "acme", r, values, applyRequest{Events: []string{"order.created"}, DryRun: true})
	if err != nil || res.Action != "would_create" || res.Rendered == "" {
		t.Fatalf("create dry run = %+v, %v", res, err)
	}
	if w := api.wrote(); len(w) != 0 {
		t.Fatalf("dry runs wrote %v", w)
	}
}

func TestResolveParamsNamesEveryMissingParam(t *testing.T) {
	r, err := resolveRecipe("sendgrid", "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = resolveParams(r, nil)
	var missing *missingParamsError
	if !errors.As(err, &missing) || len(missing.names) < 2 {
		t.Fatalf("err = %v, want every missing required param", err)
	}
	if f := missing.flags(); !strings.Contains(f, "--param api_key=...") {
		t.Fatalf("flags = %q", f)
	}
}

// Redirecting stdin from /dev/null (CI, scripts) must not count as a
// terminal: it is a character device, but nobody can answer a prompt.
func TestDevNullIsNotATerminal(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck
	if isTerminal(f) {
		t.Fatal("isTerminal(/dev/null) = true")
	}
}

// A local ./recipes file joins the catalog and overrides a built-in.
func TestCatalogIncludesLocalRecipes(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("SPARROW_RECIPES_DIR", "")
	if err := os.MkdirAll("recipes", 0o755); err != nil {
		t.Fatal(err)
	}
	draft := "version: 1\nname: teams\ndescription: Teams\nwebhook:\n  url: https://teams.test/x\n"
	if err := os.WriteFile(filepath.Join("recipes", "teams.yaml"), []byte(draft), 0o644); err != nil {
		t.Fatal(err)
	}
	all, err := catalog()
	if err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{}
	for _, e := range all {
		sources[e.recipe.Name] = e.source
	}
	if sources["teams"] != filepath.Join("recipes", "teams.yaml") || sources["slack"] != "built-in" {
		t.Fatalf("sources = %v", sources)
	}
}
