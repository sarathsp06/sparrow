package main

import (
	"os"
	"strings"
	"testing"

	"github.com/sarathsp06/sparrow/internal/config"

	"gopkg.in/yaml.v3"
)

// TestSendGridTemplateMatchesRecipe pins the hand-copied sendGridTransformTemplate
// and sendGridMailSendURL to satellites/recipes/sendgrid.yaml (a separate Go
// module, so go:embed can't share it). Fails when someone edits the recipe
// without updating the bootstrap copy, or vice versa.
func TestSendGridTemplateMatchesRecipe(t *testing.T) {
	raw, err := os.ReadFile("../../satellites/recipes/sendgrid.yaml")
	if err != nil {
		t.Fatalf("read sendgrid recipe: %v", err)
	}
	var recipe struct {
		Webhook struct {
			URL string `yaml:"url"`
		} `yaml:"webhook"`
		Subscription struct {
			TransformTemplate string `yaml:"transform_template"`
		} `yaml:"subscription"`
	}
	if err := yaml.Unmarshal(raw, &recipe); err != nil {
		t.Fatalf("parse sendgrid recipe: %v", err)
	}

	if recipe.Webhook.URL != sendGridMailSendURL {
		t.Errorf("webhook URL drifted: recipe=%q bootstrap=%q", recipe.Webhook.URL, sendGridMailSendURL)
	}
	if got, want := strings.TrimSpace(sendGridTransformTemplate), strings.TrimSpace(recipe.Subscription.TransformTemplate); got != want {
		t.Errorf("transform template drifted from satellites/recipes/sendgrid.yaml:\n--- recipe ---\n%s\n--- bootstrap const ---\n%s", want, got)
	}
}

func TestSendGridAlertReady(t *testing.T) {
	ready := sendGridAlertReady(&config.Config{
		SendGridAPIKey: "SG.real",
		AlertFromEmail: "alerts@example.com",
	})
	if ready {
		t.Fatal("placeholder from email must not activate SendGrid alerts")
	}

	ready = sendGridAlertReady(&config.Config{
		SendGridAPIKey: "SG.real",
		AlertFromEmail: "alerts@yourdomain.com",
	})
	if !ready {
		t.Fatal("real key and non-placeholder from email should activate SendGrid alerts")
	}
}

func TestSendGridAlertTemplateUsesConfig(t *testing.T) {
	got := sendGridAlertTemplate(&config.Config{
		AlertFromEmail: "alerts@yourdomain.com",
		AlertFromName:  "Sparrow Alerts",
	})
	if strings.Contains(got, `{{param "from_email"}}`) || strings.Contains(got, `{{param "from_name"}}`) {
		t.Fatalf("template still contains recipe params:\n%s", got)
	}
	if !strings.Contains(got, `"email": "alerts@yourdomain.com"`) || !strings.Contains(got, `"name": "Sparrow Alerts"`) {
		t.Fatalf("template missing configured sender:\n%s", got)
	}
}
