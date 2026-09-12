package main

import (
	"strings"
	"testing"
)

func TestSubstituteParams(t *testing.T) {
	values := map[string]string{"webhook_url": "https://hooks.example/x", "token": "t0k"}

	got, err := substituteParams(`{{param "webhook_url"}}?auth={{ param "token" }}`, values)
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://hooks.example/x?auth=t0k" {
		t.Fatalf("substitution wrong: %q", got)
	}

	// Non-param template actions pass through untouched.
	tmpl := `{"event":"{{.event_name}}","url":"{{param "webhook_url"}}"}`
	got, err = substituteParams(tmpl, values)
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"event":"{{.event_name}}","url":"https://hooks.example/x"}` {
		t.Fatalf("template actions must survive: %q", got)
	}

	// Unknown param is an error, not a silent literal.
	if _, err := substituteParams(`{{param "nope"}}`, values); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("expected missing-param error, got %v", err)
	}
}

func TestLoadRecipeFixture(t *testing.T) {
	r, err := loadRecipe("testdata/echo.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if r.Name != "echo" || r.Version != 1 {
		t.Fatalf("unexpected recipe: %+v", r)
	}
	if len(r.Params) != 1 || r.Params[0].Name != "webhook_url" || !r.Params[0].Required {
		t.Fatalf("unexpected params: %+v", r.Params)
	}
	if !strings.Contains(r.Subscription.TransformTemplate, "{{json .payload}}") {
		t.Fatalf("template not loaded: %q", r.Subscription.TransformTemplate)
	}
}
