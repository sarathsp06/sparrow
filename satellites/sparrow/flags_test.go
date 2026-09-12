package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKVFlag(t *testing.T) {
	f := kvFlag{}
	if err := f.Set("env=prod"); err != nil {
		t.Fatal(err)
	}
	if err := f.Set("region=us=east"); err != nil { // value may contain '='
		t.Fatal(err)
	}
	if f["env"] != "prod" || f["region"] != "us=east" {
		t.Fatalf("unexpected map: %v", f)
	}
	if err := f.Set("noequals"); err == nil {
		t.Fatal("expected error for value without '='")
	}
	if err := f.Set("=v"); err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestParseJSONArg(t *testing.T) {
	got, err := parseJSONArg(`{"a":1}`)
	if err != nil {
		t.Fatal(err)
	}
	if got["a"] != float64(1) {
		t.Fatalf("unexpected payload: %v", got)
	}

	path := filepath.Join(t.TempDir(), "payload.json")
	if err := os.WriteFile(path, []byte(`{"b":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err = parseJSONArg("@" + path)
	if err != nil {
		t.Fatal(err)
	}
	if got["b"] != "x" {
		t.Fatalf("unexpected file payload: %v", got)
	}

	if _, err := parseJSONArg(`[1,2]`); err == nil {
		t.Fatal("expected error for non-object JSON")
	}
}
