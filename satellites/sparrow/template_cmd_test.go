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
	root := newRootCmd(&out)
	root.SetArgs([]string{
		"template", "test",
		"--event-name", "order.created",
		"--payload", `{"order_id":"ord_1","amount":42}`,
		tmplPath,
	})
	if err := root.Execute(); err != nil {
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
	root := newRootCmd(&out)
	root.SetArgs([]string{"template", "test", tmplPath})
	if err := root.Execute(); err == nil {
		t.Fatal("expected parse error")
	}
}

// TestTemplateTestMissingKey checks the default matches a subscription's
// default (a missing key is an error) and that --missing-key zero relaxes it.
func TestTemplateTestMissingKey(t *testing.T) {
	tmplPath := filepath.Join(t.TempDir(), "total.tmpl")
	if err := os.WriteFile(tmplPath, []byte(`total={{.payload.total}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	root := newRootCmd(&out)
	root.SetArgs([]string{"template", "test", "--payload", `{"order_id":"ord_1"}`, tmplPath})
	if err := root.Execute(); err == nil || !strings.Contains(err.Error(), "total") {
		t.Fatalf("expected a missing-key error naming total, got %v", err)
	}

	out.Reset()
	root = newRootCmd(&out)
	root.SetArgs([]string{"template", "test", "--missing-key", "zero", "--payload", `{"order_id":"ord_1"}`, tmplPath})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != "total=<no value>" {
		t.Fatalf("got %q", got)
	}
}

// TestTemplateTestReadsStdin checks that "-" renders the template from stdin,
// the form the docs use with a heredoc.
func TestTemplateTestReadsStdin(t *testing.T) {
	var out bytes.Buffer
	root := newRootCmd(&out)
	root.SetIn(strings.NewReader(`{{ dict "id" .payload.id "event" .event_name | json }}`))
	root.SetArgs([]string{"template", "test", "-", "--event-name", "order.created", "--payload", `{"id":"ord_1"}`})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(out.String()); got != `{"event":"order.created","id":"ord_1"}` {
		t.Fatalf("got %q", got)
	}
}
