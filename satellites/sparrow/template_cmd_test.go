package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTemplateTestRendersFixtureRecipe renders the fixture recipe's transform
// template through the real server-side engine and checks the golden output.
func TestTemplateTestRendersFixtureRecipe(t *testing.T) {
	r, err := loadRecipe("testdata/echo.yaml")
	if err != nil {
		t.Fatal(err)
	}
	tmplPath := filepath.Join(t.TempDir(), "echo.tmpl")
	if err := os.WriteFile(tmplPath, []byte(r.Subscription.TransformTemplate), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	err = runTemplateTest([]string{
		"--event-name", "order.created",
		"--payload", `{"order_id":"ord_1","amount":42}`,
		tmplPath,
	}, &out)
	if err != nil {
		t.Fatal(err)
	}

	const golden = `{"kind":"echo","event":"order.created","data":{"amount":42,"order_id":"ord_1"}}`
	if got := strings.TrimSpace(out.String()); got != golden {
		t.Fatalf("rendered output:\n  got  %s\n  want %s", got, golden)
	}
}

func TestTemplateTestReportsParseError(t *testing.T) {
	tmplPath := filepath.Join(t.TempDir(), "bad.tmpl")
	if err := os.WriteFile(tmplPath, []byte(`{{.payload`), 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := runTemplateTest([]string{tmplPath}, &out); err == nil {
		t.Fatal("expected parse error")
	}
}
